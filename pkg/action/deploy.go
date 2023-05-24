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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/output"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"helm.sh/helm/v3/pkg/werf/plan"
	release2 "helm.sh/helm/v3/pkg/werf/release"
	resource22 "helm.sh/helm/v3/pkg/werf/resource"
	"helm.sh/helm/v3/pkg/werf/resourcebuilder"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// FIXME(ilya-lesikov): if last succeeded release was cleaned up because of release limit, werf
// will see current release as first install. We might want to not delete last succeeded or last
// uninstalled release ever.
// FIXME(ilya-lesikov): rollback should rollback to the last succesful release, not last release
// FIXME(ilya-lesikov): discovery should be without version
// FIXME(ilya-lesikov): add adoption validation, whether we can adopt or not

type Deploy struct {
	ChartPathOptions

	cfg              *Configuration
	settings         *cli.EnvSettings
	chartPath        string
	valueOptions     *values.Options
	releaseName      string
	releaseNamespace string
	outStream        io.Writer
	errStream        io.Writer
	outFormat        output.Format

	trackTimeout      time.Duration
	keepHistoryLimit  int
	rollbackOnFailure bool
	deployReportPath  string

	// lock to control raceconditions when the process receives a TERM, INT
	lock sync.Mutex
}

type DeployOptions struct {
	OutStream         io.Writer
	ErrStream         io.Writer
	ValueOptions      *values.Options
	ReleaseNamespace  string
	RollbackOnFailure bool
	TrackTimeout      time.Duration
	KeepHistoryLimit  int
	DeployReportPath  string
}

func NewDeploy(releaseName string, chartPath string, cfg *Configuration, settings *cli.EnvSettings, options DeployOptions) *Deploy {
	cfg.Client.SetDeletionTimeout(int(options.TrackTimeout))

	chartPathOptions := ChartPathOptions{
		registryClient: cfg.RegistryClient,
	}

	var releaseNamespace string
	if options.ReleaseNamespace != "" {
		releaseNamespace = options.ReleaseNamespace
	} else {
		releaseNamespace = settings.Namespace()
	}

	var outStream io.Writer
	if options.OutStream != nil {
		outStream = options.OutStream
	} else {
		outStream = os.Stdout
	}

	var errStream io.Writer
	if options.ErrStream != nil {
		errStream = options.ErrStream
	} else {
		errStream = os.Stderr
	}

	return &Deploy{
		ChartPathOptions: chartPathOptions,

		cfg:              cfg,
		settings:         settings,
		chartPath:        chartPath,
		valueOptions:     options.ValueOptions,
		releaseName:      releaseName,
		releaseNamespace: releaseNamespace,
		outStream:        outStream,
		errStream:        errStream,
		outFormat:        output.Table,

		trackTimeout:      options.TrackTimeout,
		rollbackOnFailure: options.RollbackOnFailure,
		keepHistoryLimit:  options.KeepHistoryLimit,
		deployReportPath:  options.DeployReportPath,
	}
}

func (d *Deploy) Run(ctx context.Context) error {
	d.cfg.Releases.MaxHistory = d.keepHistoryLimit

	if err := chartutil.ValidateReleaseName(d.releaseName); err != nil {
		return fmt.Errorf("release name is invalid: %s", d.releaseNamespace)
	}

	// TODO(ilya-lesikov): we skip if chart is not found, is this bug?
	if isLocated, path, err := loader.GlobalLoadOptions.ChartExtender.LocateChart(d.chartPath, d.settings); err != nil {
		return fmt.Errorf("error locating chart: %w", err)
	} else if isLocated {
		d.chartPath = path
	} else {
		if path, err := d.ChartPathOptions.LocateChart(d.chartPath, d.settings); err != nil {
			return fmt.Errorf("error locating chart: %w", err)
		} else {
			d.chartPath = path
		}
	}

	getters := getter.All(d.settings)
	valuesMap, err := d.valueOptions.MergeValues(getters, loader.GlobalLoadOptions.ChartExtender)
	if err != nil {
		return fmt.Errorf("error merging values: %w", err)
	}

	chart, err := loader.Load(d.chartPath)
	if err != nil {
		return fmt.Errorf("error loading chart: %w", err)
	}
	if chart == nil {
		return errMissingChart
	}
	if chart.Metadata.Type != "" && chart.Metadata.Type != "application" {
		return fmt.Errorf("chart type %q can't be deployed", chart.Metadata.Type)
	}

	if chart.Metadata.Deprecated {
		fmt.Fprintf(d.errStream, "This chart is deprecated.\n")
	}

	if chart.Metadata.Dependencies != nil {
		if err := CheckDependencies(chart, chart.Metadata.Dependencies); err != nil {
			return fmt.Errorf("An error occurred while checking for chart dependencies. You may need to run `werf helm dependency build` to fetch missing dependencies: %w", err)
		}
	}

	// FIXME(ilya-lesikov):
	// Create context and prepare the handle of SIGTERM
	// ctx, cancel := context.WithCancel(context.Background())

	// FIXME(ilya-lesikov):
	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	// signalCh := make(chan os.Signal, 2)
	// signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	// go func() {
	// 	<-signalCh
	// 	fmt.Fprintf(d.errStream, "Release %q has been cancelled.\n", d.releaseName)
	// 	cancel()
	// }()

	if err := chartutil.ProcessDependencies(chart, &valuesMap); err != nil {
		return fmt.Errorf("error processing dependencies: %w", err)
	}

	if err := d.cfg.DeferredKubeClient.Init(); err != nil {
		return fmt.Errorf("error initializing deferred kube client: %w", err)
	}

	// TODO(ilya-lesikov): for local mode here should be stub. Later might allow overriding this stub by user.
	// {
	// 	d.cfg.Capabilities = chartutil.DefaultCapabilities.Copy()
	// 	if d.KubeVersion != nil {
	// 		d.cfg.Capabilities.KubeVersion = *i.KubeVersion
	// 	}
	// }

	history, err := d.cfg.Releases.History(d.releaseName)
	if err != nil && err != driver.ErrReleaseNotFound {
		return fmt.Errorf("error getting history for release %q: %w", d.releaseName, err)
	}
	historyAnalyzer := release2.NewHistoryAnalyzer(history)
	deployType := historyAnalyzer.DeployTypeForNewRelease()
	revision := historyAnalyzer.RevisionForNewRelease()
	prevRelease := historyAnalyzer.LastRelease()

	var (
		isInstall bool
		isUpgrade bool
	)
	switch deployType {
	case plan.DeployTypeInitial, plan.DeployTypeInstall:
		isInstall = true
	case plan.DeployTypeUpgrade, plan.DeployTypeRollback:
		isUpgrade = true
	}

	releaseOptions := chartutil.ReleaseOptions{
		Name:      d.releaseName,
		Namespace: d.releaseNamespace,
		Revision:  revision,
		IsInstall: isInstall,
		IsUpgrade: isUpgrade,
	}

	// TODO(ilya-lesikov): capabilities should be manually updated with CRDs
	capabilities, err := d.cfg.getCapabilities()
	if err != nil {
		return fmt.Errorf("error getting capabilities: %w", err)
	}

	values, err := chartutil.ToRenderValues(chart, valuesMap, releaseOptions, capabilities)
	if err != nil {
		return fmt.Errorf("error building values: %w", err)
	}

	// TODO(ilya-lesikov): pass dryrun for local version
	legacyHooks, manifestsBuf, notes, err := d.cfg.renderResources(chart, values, "", "", true, false, false, nil, false, false)
	if err != nil {
		return fmt.Errorf("error rendering resources: %w", err)
	}
	manifests := manifestsBuf.String()

	releaseNamespace := resource22.NewUnmanagedResource(
		&unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Namespace",
				"metadata": map[string]interface{}{
					"name": d.releaseNamespace,
				},
			},
		},
		d.cfg.Client.DiscoveryRESTMapper(),
		d.cfg.Client.DiscoveryClient(),
	)
	if err := releaseNamespace.Validate(); err != nil {
		return fmt.Errorf("error validating release namespace: %w", err)
	}

	deployResourceBuilder := resourcebuilder.NewDeployResourceBuilder(releaseNamespace, deployType, d.cfg.Client).
		WithLegacyPreloadedCRDs(chart.CRDObjects()...).
		WithLegacyHelmHooks(legacyHooks...).
		WithReleaseManifests(manifests)

	if prevRelease != nil {
		deployResourceBuilder = deployResourceBuilder.
			WithPrevReleaseManifests(prevRelease.Manifest)
	}

	resources, err := deployResourceBuilder.Build(ctx)
	if err != nil {
		return fmt.Errorf("error building resources: %w", err)
	}

	if os.Getenv("WERF_EXPERIMENTAL_DEPLOY_ENGINE_DEBUG") == "1" {
		for _, res := range resources.HelmResources.UpToDate {
			b, _ := json.MarshalIndent(res.Local.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmResourceUpToDateLocal):\n%s\n", b)
			b, _ = json.MarshalIndent(res.Live.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmResourceUpToDateLive):\n%s\n", b)
		}
		for _, res := range resources.HelmResources.Outdated {
			b, _ := json.MarshalIndent(res.Local.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmResourceOutdatedLocal):\n%s\n", b)
			b, _ = json.MarshalIndent(res.Live.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmResourceOutdatedLive):\n%s\n", b)
			b, _ = json.MarshalIndent(res.Desired.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmResourceOutdatedDesired):\n%s\n", b)
		}
		for _, res := range resources.HelmResources.Unsupported {
			b, _ := json.MarshalIndent(res.Local.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmResourceUnsupportedLocal):\n%s\n", b)
		}
		for _, res := range resources.HelmResources.OutdatedImmutable {
			b, _ := json.MarshalIndent(res.Local.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmResourceOutdatedImmutableLocal):\n%s\n", b)
			b, _ = json.MarshalIndent(res.Live.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmResourceOutdatedImmutableLive):\n%s\n", b)
		}
		for _, res := range resources.HelmResources.NonExisting {
			b, _ := json.MarshalIndent(res.Local.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmResourceNonExistingLocal):\n%s\n", b)
			b, _ = json.MarshalIndent(res.Desired.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmResourceNonExistingDesired):\n%s\n", b)
		}
		for _, res := range resources.HelmHooks.Matched.UpToDate {
			b, _ := json.MarshalIndent(res.Local.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmHooksMatchedUpToDateLocal):\n%s\n", b)
			b, _ = json.MarshalIndent(res.Live.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmHooksMatchedUpToDateLive):\n%s\n", b)
		}
		for _, res := range resources.HelmHooks.Matched.Outdated {
			b, _ := json.MarshalIndent(res.Local.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmHooksMatchedOutdatedLocal):\n%s\n", b)
			b, _ = json.MarshalIndent(res.Live.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmHooksMatchedOutdatedLive):\n%s\n", b)
			b, _ = json.MarshalIndent(res.Desired.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmHooksMatchedOutdatedDesired):\n%s\n", b)
		}
		for _, res := range resources.HelmHooks.Matched.OutdatedImmutable {
			b, _ := json.MarshalIndent(res.Local.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmHooksMatchedOutdatedImmutableLocal):\n%s\n", b)
			b, _ = json.MarshalIndent(res.Live.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmHooksMatchedOutdatedImmutableLive):\n%s\n", b)
		}
		for _, res := range resources.HelmHooks.Matched.NonExisting {
			b, _ := json.MarshalIndent(res.Local.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmHooksMatchedNonExistingLocal):\n%s\n", b)
			b, _ = json.MarshalIndent(res.Desired.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmHooksMatchedNonExistingDesired):\n%s\n", b)
		}
		for _, res := range resources.HelmHooks.Matched.Unsupported {
			b, _ := json.MarshalIndent(res.Local.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmHooksMatchedUnsupportedLocal):\n%s\n", b)
		}
		for _, res := range resources.HelmHooks.Unmatched {
			b, _ := json.MarshalIndent(res.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(helmHooksUnmatched):\n%s\n", b)
		}
		for _, res := range resources.PreloadedCRDs.Outdated {
			b, _ := json.MarshalIndent(res.Local.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(preloadedCrdsOutdatedLocal):\n%s\n", b)
			b, _ = json.MarshalIndent(res.Live.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(preloadedCrdsOutdatedLive):\n%s\n", b)
			b, _ = json.MarshalIndent(res.Desired.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(preloadedCrdsOutdatedDesired):\n%s\n", b)
		}
		for _, res := range resources.PreloadedCRDs.OutdatedImmutable {
			b, _ := json.MarshalIndent(res.Local.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(preloadedCrdsOutdatedImmutableLocal):\n%s\n", b)
			b, _ = json.MarshalIndent(res.Live.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(preloadedCrdsOutdatedImmutableLive):\n%s\n", b)
		}
		for _, res := range resources.PreloadedCRDs.UpToDate {
			b, _ := json.MarshalIndent(res.Local.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(preloadedCrdsUpToDateLocal):\n%s\n", b)
			b, _ = json.MarshalIndent(res.Live.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(preloadedCrdsUpToDateLive):\n%s\n", b)
		}
		for _, res := range resources.PreloadedCRDs.NonExisting {
			b, _ := json.MarshalIndent(res.Local.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(preloadedCrdsNonExistingLocal):\n%s\n", b)
			b, _ = json.MarshalIndent(res.Desired.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(preloadedCrdsNonExistingDesired):\n%s\n", b)
		}
		for _, res := range resources.PrevReleaseHelmResources.Existing {
			b, _ := json.MarshalIndent(res.Local.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(prevReleaseHelmResourcesExistingLocal):\n%s\n", b)
			b, _ = json.MarshalIndent(res.Live.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(prevReleaseHelmResourcesExistingLive):\n%s\n", b)
		}
		for _, res := range resources.PrevReleaseHelmResources.NonExisting {
			b, _ := json.MarshalIndent(res.Unstructured().UnstructuredContent(), "", "\t")
			fmt.Printf("DEBUG(prevReleaseHelmResourcesNonExistingLocal):\n%s\n", b)
		}
	}

	// FIXME(ilya-lesikov): additional validation here
	// FIXME(ilya-lesikov): move it somewhere?
	var helmHooks []*resource22.HelmHook
	for _, hook := range resources.HelmHooks.Matched.UpToDate {
		helmHooks = append(helmHooks, hook.Local)
	}
	for _, hook := range resources.HelmHooks.Matched.Outdated {
		helmHooks = append(helmHooks, hook.Local)
	}
	for _, hook := range resources.HelmHooks.Matched.OutdatedImmutable {
		helmHooks = append(helmHooks, hook.Local)
	}
	for _, hook := range resources.HelmHooks.Matched.NonExisting {
		helmHooks = append(helmHooks, hook.Local)
	}
	for _, hook := range resources.HelmHooks.Matched.Unsupported {
		helmHooks = append(helmHooks, hook.Local)
	}

	// FIXME(ilya-lesikov): move it somewhere?
	var helmResources []*resource22.HelmResource
	for _, res := range resources.HelmResources.UpToDate {
		helmResources = append(helmResources, res.Local)
	}
	for _, res := range resources.HelmResources.Outdated {
		helmResources = append(helmResources, res.Local)
	}
	for _, res := range resources.HelmResources.OutdatedImmutable {
		helmResources = append(helmResources, res.Local)
	}
	for _, res := range resources.HelmResources.NonExisting {
		helmResources = append(helmResources, res.Local)
	}
	for _, res := range resources.HelmResources.Unsupported {
		helmResources = append(helmResources, res.Local)
	}

	deployReleaseBuilder := release2.NewDeployReleaseBuilder(deployType, d.releaseName, releaseNamespace.Name(), chart, valuesMap, revision).
		WithHelmHooks(helmHooks).
		WithHelmResources(helmResources).
		WithNotes(notes).
		WithPrevRelease(prevRelease)

	rel, err := deployReleaseBuilder.BuildPendingRelease()
	if err != nil {
		return fmt.Errorf("error building release: %w", err)
	}

	succeededRel := deployReleaseBuilder.PromotePendingReleaseToSucceeded(rel)

	var supersededRel *release.Release
	if prevRelease != nil {
		supersededRel = deployReleaseBuilder.PromotePreviousReleaseToSuperseded(prevRelease)
	}

	deployPlan, referencesToCleanupOnFailure, err := plan.
		NewDeployPlanBuilder(deployType, resources.ReleaseNamespace, rel, succeededRel).
		WithPreloadedCRDs(resources.PreloadedCRDs).
		WithMatchedHelmHooks(resources.HelmHooks.Matched).
		WithHelmResources(resources.HelmResources).
		WithPreviousReleaseHelmResources(resources.PrevReleaseHelmResources).
		WithSupersededPreviousRelease(supersededRel).
		WithPreviousReleaseDeployed(historyAnalyzer.LastReleaseDeployed()).
		Build(ctx)
	if err != nil {
		return fmt.Errorf("error building deploy plan: %w", err)
	}

	if os.Getenv("WERF_EXPERIMENTAL_DEPLOY_ENGINE_DEBUG") == "1" {
		for _, phase := range deployPlan.Phases {
			fmt.Printf("DEBUG(phaseType): %s\n", phase.Type)
			for _, operation := range phase.Operations {
				fmt.Printf("DEBUG(opType): %s\n", operation.Type())
				switch op := operation.(type) {
				case *plan.OperationCreate:
					for _, target := range op.Targets {
						b, _ := json.MarshalIndent(target.Unstructured().UnstructuredContent(), "", "\t")
						fmt.Printf("DEBUG(opTarget):\n%s\n", b)
					}
				case *plan.OperationUpdate:
					for _, target := range op.Targets {
						b, _ := json.MarshalIndent(target.Unstructured().UnstructuredContent(), "", "\t")
						fmt.Printf("DEBUG(opTarget):\n%s\n", b)
					}
				case *plan.OperationRecreate:
					for _, target := range op.Targets {
						b, _ := json.MarshalIndent(target.Unstructured().UnstructuredContent(), "", "\t")
						fmt.Printf("DEBUG(opTarget):\n%s\n", b)
					}
				case *plan.OperationDelete:
					fmt.Printf("DEBUG(opTargets): %s\n", op.Targets)
				case *plan.OperationCreateReleases:
					for _, r := range op.Releases {
						b, _ := json.MarshalIndent(r, "", "\t")
						fmt.Printf("DEBUG(opTarget):\n%s\n", b)
					}
				case *plan.OperationUpdateReleases:
					for _, r := range op.Releases {
						b, _ := json.MarshalIndent(r, "", "\t")
						fmt.Printf("DEBUG(opTarget):\n%s\n", b)
					}
				}
			}
		}
	}

	if deployPlan.Empty() {
		fmt.Fprintf(d.outStream, "\nRelease %q in namespace %q canceled: no changes to be made.\n", d.releaseName, d.releaseNamespace)
		return nil
	}

	// TODO(ilya-lesikov): add more info from executor report
	if d.deployReportPath != "" {
		defer func() {
			deployReportData, err := release.NewDeployReport().FromRelease(rel).ToJSONData()
			if err != nil {
				d.cfg.Log("warning: error creating deploy report data: %s", err)
				return
			}

			if err := os.WriteFile(d.deployReportPath, deployReportData, 0o644); err != nil {
				d.cfg.Log("warning: error writing deploy report file: %s", err)
				return
			}
		}()
	}

	deployReport, executeErr := plan.NewDeployPlanExecutor(deployPlan, releaseNamespace, d.cfg.Client, d.cfg.Tracker, d.cfg.Releases).WithTrackTimeout(d.trackTimeout).Execute(ctx)
	if executeErr != nil {
		defer func() {
			fmt.Fprintf(d.errStream, "\nRelease %q in namespace %q failed.\n", d.releaseName, d.releaseNamespace)
		}()

		rel = deployReleaseBuilder.PromotePendingReleaseToFailed(rel)

		finalizeFailedDeployPlan := plan.
			NewFinalizeFailedDeployPlanBuilder(rel).
			WithReferencesToCleanup(referencesToCleanupOnFailure).
			Build()

		deployReport, err = plan.NewDeployPlanExecutor(finalizeFailedDeployPlan, releaseNamespace, d.cfg.Client, d.cfg.Tracker, d.cfg.Releases).WithTrackTimeout(d.trackTimeout).WithReport(deployReport).Execute(ctx)
		if err != nil {
			return multierror.Append(executeErr, fmt.Errorf("error finalizing failed deploy plan: %w", err))
		}

		return executeErr
	}

	rel = succeededRel

	defer func() {
		fmt.Fprintf(d.outStream, "\nRelease %q in namespace %q succeeded.\n", d.releaseName, d.releaseNamespace)
	}()

	deployReportPrinter := plan.NewDeployReportPrinter(d.outStream, deployReport)
	deployReportPrinter.PrintSummary()

	// FIXME(ilya-lesikov): better error handling (interrupts, etc)

	// FIXME(ilya-lesikov): don't forget errs.FormatTemplatingError if any errors occurs

	return nil
}
