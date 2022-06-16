package stages

import (
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/phasemanagers/extdeps"
)

type Stage struct {
	Weight               int
	ExternalDependencies extdeps.ExternalDependencyList
	DesiredResources     kube.ResourceList
	Result               *kube.Result
}
