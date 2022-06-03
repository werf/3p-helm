package phasemanagers

import (
	"fmt"

	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/phasemanagers/phases"
	"helm.sh/helm/v3/pkg/phasemanagers/stages"
	rel "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage"
)

func NewRolloutPhaseManager(rolloutPhase *phases.RolloutPhase, deployedResCalc *DeployedResourcesCalculator, release *rel.Release, storage *storage.Storage, kubeClient kube.Interface) *RolloutPhaseManager {
	return &RolloutPhaseManager{
		Phase:                       rolloutPhase,
		Release:                     release,
		Storage:                     storage,
		deployedResourcesCalculator: deployedResCalc,
		kubeClient:                  kubeClient,
	}
}

type RolloutPhaseManager struct {
	Phase   *phases.RolloutPhase
	Release *rel.Release
	Storage *storage.Storage

	deployedResourcesCalculator *DeployedResourcesCalculator
	previouslyDeployedResources kube.ResourceList
	kubeClient                  kube.Interface
}

func (m *RolloutPhaseManager) AddCalculatedPreviouslyDeployedResources() (*RolloutPhaseManager, error) {
	resources, err := m.deployedResourcesCalculator.Calculate()
	if err != nil {
		return nil, fmt.Errorf("error calculating previously deployed resources: %w", err)
	}

	m.previouslyDeployedResources.Merge(resources)

	return m, nil
}

func (m *RolloutPhaseManager) AddPreviouslyDeployedResources(resources kube.ResourceList) *RolloutPhaseManager {
	m.previouslyDeployedResources.Merge(resources)

	return m
}

func (m *RolloutPhaseManager) DoStage(doFn func(stgIndex int, stage *stages.Stage, prevDeployedStgResources kube.ResourceList) error) error {
	for i, stg := range m.Phase.SortedStages {
		rel.SetRolloutPhaseStageInfo(m.Release, i)
		if err := m.Storage.Update(m.Release); err != nil {
			return fmt.Errorf("error updating release in storage: %w", err)
		}

		if err := doFn(i, stg, m.previouslyDeployedResources.Intersect(stg.DesiredResources)); err != nil {
			return fmt.Errorf("error processing stage: %w", err)
		}
	}

	return nil
}

func (m *RolloutPhaseManager) DeleteOrphanedResources() error {
	orphanedResources := m.previouslyDeployedResources.Difference(m.Phase.AllResources())
	_, errs := m.kubeClient.Delete(orphanedResources, kube.DeleteOptions{Wait: true})
	if len(errs) > 0 {
		return fmt.Errorf("while deleting previously deployed but now orphaned resources got %d error(s): %s", len(errs), joinErrors(errs))
	}

	return nil
}
