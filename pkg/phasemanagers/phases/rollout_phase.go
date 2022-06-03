package phases

import (
	"bytes"
	"fmt"

	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/phasemanagers/stages"
	rel "helm.sh/helm/v3/pkg/release"
)

func NewRolloutPhase(release *rel.Release, stagesSplitter stages.Splitter) *RolloutPhase {
	return &RolloutPhase{
		Release:        release,
		stagesSplitter: stagesSplitter,
	}
}

type RolloutPhase struct {
	SortedStages stages.SortedStageList
	Release      *rel.Release

	stagesSplitter stages.Splitter
}

func (m *RolloutPhase) ParseStagesFromString(manifests string, kubeClient kube.Interface) (*RolloutPhase, error) {
	resources, err := kubeClient.Build(bytes.NewBufferString(manifests), false)
	if err != nil {
		return nil, fmt.Errorf("error building kubernetes objects from manifests: %w", err)
	}

	return m.ParseStages(resources)
}

func (m *RolloutPhase) ParseStages(resources kube.ResourceList) (*RolloutPhase, error) {
	var err error
	m.SortedStages, err = m.stagesSplitter.Split(resources)
	if err != nil {
		return nil, fmt.Errorf("error splitting rollout stage resources list: %w", err)
	}

	return m, nil
}

func (m *RolloutPhase) DeployedResources() kube.ResourceList {
	lastDeployedStageIndex := m.LastDeployedStageIndex()
	if lastDeployedStageIndex == nil {
		return nil
	}

	return m.SortedStages.MergedDesiredResourcesInStagesRange(0, *lastDeployedStageIndex)
}

func (m *RolloutPhase) AllResources() kube.ResourceList {
	return m.SortedStages.MergedDesiredResources()
}

func (m *RolloutPhase) LastDeployedStageIndex() *int {
	if !m.IsPhaseStarted() {
		return nil
	}

	lastStage := len(m.SortedStages) - 1

	if m.IsPhaseCompleted() {
		return &lastStage
	}

	// Phase started but not completed.
	if m.Release.Info.LastStage == nil {
		return &lastStage
	} else {
		return m.Release.Info.LastStage
	}
}

func (m *RolloutPhase) IsPhaseStarted() bool {
	if m.Release.Info.LastPhase == nil {
		return true
	}

	switch *m.Release.Info.LastPhase {
	case rel.PhaseRollout, rel.PhaseUninstall, rel.PhaseHooksPost:
		return true
	default:
		return false
	}
}

func (m *RolloutPhase) IsPhaseCompleted() bool {
	if m.Release.Info.LastPhase == nil {
		return true
	}

	switch *m.Release.Info.LastPhase {
	case rel.PhaseRollout:
		if m.Release.Info.LastStage == nil {
			return true
		} else {
			return *m.Release.Info.LastStage == len(m.SortedStages)-1
		}
	case rel.PhaseHooksPost:
		return true
	default:
		return false
	}
}
