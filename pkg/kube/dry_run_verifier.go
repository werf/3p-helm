package kube

import (
	"fmt"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
)

type DryRunVerifierGetter struct {
	clientGetter genericclioptions.RESTClientGetter
}

func NewDryRunVerifierGetter(clientGetter genericclioptions.RESTClientGetter) *DryRunVerifierGetter {
	return &DryRunVerifierGetter{clientGetter: clientGetter}
}

func (getter *DryRunVerifierGetter) DryRunVerifier() (*resource.DryRunVerifier, error) {
	restConfig, err := getter.clientGetter.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to get rest config: %w", err)
	}
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create discovery client: %w", err)
	}
	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create dynamic client: %w", err)
	}

	return resource.NewDryRunVerifier(dynamicClient, discoveryClient), nil
}
