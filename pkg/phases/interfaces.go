package phases

import (
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/phases/stages"
)

type Splitter interface {
	Split(resources kube.ResourceList) (stages.SortedStageList, error)
}

type ExternalDepsGenerator interface {
	Generate(stages stages.SortedStageList) error
}
