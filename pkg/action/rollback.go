/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package action

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/phases"
	"helm.sh/helm/v3/pkg/phases/phasemanagers"
	"helm.sh/helm/v3/pkg/phases/stages"
	"helm.sh/helm/v3/pkg/release"
	helmtime "helm.sh/helm/v3/pkg/time"
)

// Rollback is the action for rolling back to a given release.
//
// It provides the implementation of 'helm rollback'.
type Rollback struct {
	cfg *Configuration

	Version       int
	Timeout       time.Duration
	Wait          bool
	WaitForJobs   bool
	DisableHooks  bool
	DryRun        bool
	Recreate      bool // will (if true) recreate pods after a rollback.
	Force         bool // will (if true) force resource upgrade through uninstall/recreate if needed
	CleanupOnFail bool
	MaxHistory    int // MaxHistory limits the maximum number of revisions saved per release

	StagesSplitter phases.Splitter

	StagesExternalDepsGenerator phases.ExternalDepsGenerator
}

// NewRollback creates a new Rollback object with the given configuration.
func NewRollback(cfg *Configuration, stagesSplitter phases.Splitter, stagesExternalDepsGenerator phases.ExternalDepsGenerator) *Rollback {
	if stagesSplitter == nil {
		stagesSplitter = &phases.SingleStageSplitter{}
	}

	if stagesExternalDepsGenerator == nil {
		stagesExternalDepsGenerator = &phases.NoExternalDepsGenerator{}
	}

	return &Rollback{
		cfg:            cfg,
		StagesSplitter: stagesSplitter,

		StagesExternalDepsGenerator: stagesExternalDepsGenerator,
	}
}

// Run executes 'helm rollback' against the given release.
func (r *Rollback) Run(name string) error {
	if err := r.cfg.KubeClient.IsReachable(); err != nil {
		return err
	}

	r.cfg.Releases.MaxHistory = r.MaxHistory

	r.cfg.Log("preparing rollback of %s", name)
	currentRelease, targetRelease, err := r.prepareRollback(name)
	if err != nil {
		return err
	}

	if !r.DryRun {
		r.cfg.Log("creating rolled back release for %s", name)
		if err := r.cfg.Releases.Create(targetRelease); err != nil {
			return err
		}
	}

	r.cfg.Log("performing rollback of %s", name)
	if _, err := r.performRollback(currentRelease, targetRelease); err != nil {
		return err
	}

	if !r.DryRun {
		r.cfg.Log("updating status for rolled back release for %s", name)
		if err := r.cfg.Releases.Update(targetRelease); err != nil {
			return err
		}
	}
	return nil
}

// prepareRollback finds the previous release and prepares a new release object with
// the previous release's configuration
func (r *Rollback) prepareRollback(name string) (*release.Release, *release.Release, error) {
	if err := chartutil.ValidateReleaseName(name); err != nil {
		return nil, nil, errors.Errorf("prepareRollback: Release name is invalid: %s", name)
	}

	if r.Version < 0 {
		return nil, nil, errInvalidRevision
	}

	currentRelease, err := r.cfg.Releases.Last(name)
	if err != nil {
		return nil, nil, err
	}

	previousVersion := r.Version
	if r.Version == 0 {
		previousVersion = currentRelease.Version - 1
	}

	r.cfg.Log("rolling back %s (current: v%d, target: v%d)", name, currentRelease.Version, previousVersion)

	previousRelease, err := r.cfg.Releases.Get(name, previousVersion)
	if err != nil {
		return nil, nil, err
	}

	// Store a new release object with previous release's configuration
	targetRelease := release.SetInitPhaseStageInfo(&release.Release{
		Name:      name,
		Namespace: currentRelease.Namespace,
		Chart:     previousRelease.Chart,
		Config:    previousRelease.Config,
		Info: &release.Info{
			FirstDeployed: currentRelease.Info.FirstDeployed,
			LastDeployed:  helmtime.Now(),
			Status:        release.StatusPendingRollback,
			Notes:         previousRelease.Info.Notes,
			// Because we lose the reference to previous version elsewhere, we set the
			// message here, and only override it later if we experience failure.
			Description: fmt.Sprintf("Rollback to %d", previousVersion),
		},
		Version:  currentRelease.Version + 1,
		Manifest: previousRelease.Manifest,
		Hooks:    previousRelease.Hooks,
	})

	return currentRelease, targetRelease, nil
}

func (r *Rollback) performRollback(currentRelease, targetRelease *release.Release) (*release.Release, error) {
	if r.DryRun {
		r.cfg.Log("dry run for %s", targetRelease.Name)
		return targetRelease, nil
	}

	// pre-rollback hooks
	if !r.DisableHooks {
		if err := r.cfg.execHook(targetRelease, release.HookPreRollback, r.Timeout); err != nil {
			return targetRelease, err
		}
	} else {
		r.cfg.Log("rollback hooks disabled for %s", targetRelease.Name)
	}

	history, err := r.cfg.Releases.HistoryUntilRevision(targetRelease.Name, targetRelease.Version)
	if err != nil {
		recordFailedStatus(r.cfg, currentRelease, targetRelease, err)
		return targetRelease, err
	}

	rolloutPhase, err := phases.NewRolloutPhase(targetRelease, r.StagesSplitter, r.cfg.KubeClient).
		ParseStagesFromString(targetRelease.Manifest)
	if err != nil {
		recordFailedStatus(r.cfg, currentRelease, targetRelease, err)
		return targetRelease, err
	}

	if err := rolloutPhase.GenerateStagesExternalDeps(r.StagesExternalDepsGenerator); err != nil {
		recordFailedStatus(r.cfg, currentRelease, targetRelease, err)
		return targetRelease, err
	}

	deployedResourcesCalculator := phases.NewDeployedResourcesCalculator(history, r.StagesSplitter, r.cfg.KubeClient)

	rolloutPhaseManager, err := phasemanagers.NewRolloutPhaseManager(rolloutPhase, deployedResourcesCalculator, targetRelease, r.cfg.Releases, r.cfg.KubeClient).
		AddCalculatedPreviouslyDeployedResources()
	if err != nil {
		recordFailedStatus(r.cfg, currentRelease, targetRelease, err)
		return targetRelease, err
	}

	if err := rolloutPhaseManager.DoStage(
		func(stgIndex int, stage *stages.Stage) error {
			if len(stage.ExternalDependencies) == 0 || !r.Wait {
				return nil
			}

			if r.WaitForJobs {
				return r.cfg.KubeClient.WaitWithJobs(stage.ExternalDependencies.AsResourceList(), r.Timeout)
			} else {
				return r.cfg.KubeClient.Wait(stage.ExternalDependencies.AsResourceList(), r.Timeout)
			}
		},
		func(stgIndex int, stage *stages.Stage, prevDeployedStgResources kube.ResourceList) error {
			if len(prevDeployedStgResources) == 0 {
				stage.Result, err = r.cfg.KubeClient.Create(stage.DesiredResources)
				if err != nil {
					return err
				}
			} else {
				stage.Result, err = r.cfg.KubeClient.Update(prevDeployedStgResources, stage.DesiredResources, kube.UpdateOptions{
					Force:                        r.Force,
					SkipDeleteIfInvalidOwnership: true,
					ReleaseName:                  targetRelease.Name,
					ReleaseNamespace:             targetRelease.Namespace,
				})
				if err != nil {
					return err
				}
			}

			if r.Recreate {
				// NOTE: Because this is not critical for a release to succeed, we just
				// log if an error occurs and continue onward. If we ever introduce log
				// levels, we should make these error level logs so users are notified
				// that they'll need to go do the cleanup on their own
				if err := recreate(r.cfg, stage.Result.Updated); err != nil {
					r.cfg.Log(err.Error())
				}
			}

			return nil
		},
		func(stgIndex int, stage *stages.Stage) error {
			if !r.Wait {
				return nil
			}

			if r.WaitForJobs {
				return r.cfg.KubeClient.WaitWithJobs(stage.DesiredResources, r.Timeout)
			} else {
				return r.cfg.KubeClient.Wait(stage.DesiredResources, r.Timeout)
			}
		},
	); err != nil {
		recordFailedStatus(r.cfg, currentRelease, targetRelease, err)

		if r.CleanupOnFail {
			createdResourcesToDelete := kube.ResourceList{}
			var applyErr *phasemanagers.ApplyError
			if errors.As(err, &applyErr) {
				createdResourcesToDelete = rolloutPhaseManager.Phase.SortedStages[len(rolloutPhaseManager.Phase.SortedStages)-1].Result.Created
			}

			if len(createdResourcesToDelete) > 0 {
				r.cfg.Log("Cleanup on fail set, cleaning up %d resources", len(createdResourcesToDelete))
				_, errs := r.cfg.KubeClient.Delete(createdResourcesToDelete, kube.DeleteOptions{
					Wait:                   r.Wait,
					WaitTimeout:            r.Timeout,
					SkipIfInvalidOwnership: true,
					ReleaseName:            targetRelease.Name,
					ReleaseNamespace:       targetRelease.Namespace,
				})
				if errs != nil {
					var errorList []string
					for _, e := range errs {
						errorList = append(errorList, e.Error())
					}
					return targetRelease, errors.Wrapf(fmt.Errorf("unable to cleanup resources: %s", strings.Join(errorList, ", ")), "an error occurred while cleaning up resources. original rollback error: %s", err)
				}
				r.cfg.Log("Resource cleanup complete")
			}
		}

		return targetRelease, err
	}

	if err := rolloutPhaseManager.DeleteOrphanedResources(); err != nil {
		r.cfg.Log("failure removing resources no longer present in the release: %w", err)
	}

	// post-rollback hooks
	if !r.DisableHooks {
		if err := r.cfg.execHook(targetRelease, release.HookPostRollback, r.Timeout); err != nil {
			return targetRelease, err
		}
	}

	deployed, err := r.cfg.Releases.DeployedAll(currentRelease.Name)
	if err != nil && !strings.Contains(err.Error(), "has no deployed releases") {
		return nil, err
	}
	// Supersede all previous deployments, see issue #2941.
	for _, rel := range deployed {
		if rel.Version == targetRelease.Version {
			continue
		}

		r.cfg.Log("superseding previous deployment %d", rel.Version)
		rel.Info.Status = release.StatusSuperseded
		r.cfg.recordRelease(rel)
	}

	targetRelease.Info.Status = release.StatusDeployed

	return targetRelease, nil
}

func recordFailedStatus(cfg *Configuration, currentRelease, targetRelease *release.Release, err error) {
	msg := fmt.Sprintf("Rollback %q failed: %s", targetRelease.Name, err)

	cfg.Log("warning: %s", msg)
	targetRelease.Info.Description = msg

	currentRelease.Info.Status = release.StatusSuperseded
	targetRelease.Info.Status = release.StatusFailed

	cfg.recordRelease(currentRelease)
	cfg.recordRelease(targetRelease)
}
