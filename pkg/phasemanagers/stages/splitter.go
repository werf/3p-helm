package stages

import (
	"fmt"

	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/cli-runtime/pkg/resource"
)

type Splitter interface {
	Split(resources kube.ResourceList) (SortedStageList, error)
}

type SingleStageSplitter struct{}

func (s *SingleStageSplitter) Split(resources kube.ResourceList) (SortedStageList, error) {
	stage := &Stage{}

	if err := resources.Visit(func(res *resource.Info, err error) error {
		if err != nil {
			return err
		}

		stage.DesiredResources.Append(res)

		return nil
	}); err != nil {
		return nil, fmt.Errorf("error visiting resources list: %w", err)
	}

	return SortedStageList{stage}, nil
}
