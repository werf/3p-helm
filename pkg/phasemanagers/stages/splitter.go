package stages

import (
	"fmt"

	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/resource"
)

type Splitter interface {
	Split(resources kube.ResourceList) (SortedStageList, error)
}

type SingleStageSplitter struct{}

func (s SingleStageSplitter) Split(resources kube.ResourceList) (SortedStageList, error) {
	stage := &Stage{}

	if err := resources.Visit(func(res *resource.Info, err error) error {
		if err != nil {
			return err
		}

		unstructuredObj := unstructured.Unstructured{}
		unstructuredObj.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(res.Object)
		if err != nil {
			return fmt.Errorf("error converting object to unstructured type: %w", err)
		}

		stage.DesiredResources.Append(res)

		return nil
	}); err != nil {
		return nil, fmt.Errorf("error visiting resources list: %w", err)
	}

	return SortedStageList{stage}, nil
}
