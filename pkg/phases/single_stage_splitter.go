package phases

import (
	"fmt"

	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/phases/stages"
	"k8s.io/cli-runtime/pkg/resource"
)

type SingleStageSplitter struct{}

func (s *SingleStageSplitter) Split(resources kube.ResourceList) (stages.SortedStageList, error) {
	stage := &stages.Stage{}

	if err := resources.Visit(func(res *resource.Info, err error) error {
		if err != nil {
			return err
		}

		stage.DesiredResources.Append(res)

		return nil
	}); err != nil {
		return nil, fmt.Errorf("error visiting resources list: %w", err)
	}

	return stages.SortedStageList{stage}, nil
}
