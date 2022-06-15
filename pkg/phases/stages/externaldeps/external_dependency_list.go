package externaldeps

import (
	"helm.sh/helm/v3/pkg/kube"
)

type ExternalDependencyList []*ExternalDependency

func (l ExternalDependencyList) AsResourceList() kube.ResourceList {
	var resourceList kube.ResourceList
	for _, extDep := range l {
		resourceList = append(resourceList, extDep.Info)
	}

	return resourceList
}
