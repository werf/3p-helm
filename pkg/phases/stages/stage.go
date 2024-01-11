package stages

import (
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/phases/stages/externaldeps"
)

type Stage struct {
	Weight               int
	ExternalDependencies externaldeps.ExternalDependencyList
	DesiredResources     kube.ResourceList
	Result               *kube.Result
}
