package stages

import (
	"helm.sh/helm/v3/pkg/kube"
)

type Stage struct {
	Weight           int
	DesiredResources kube.ResourceList
	Result           *kube.Result
}
