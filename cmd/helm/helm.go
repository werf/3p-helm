/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main // import "k8s.io/helm/cmd/helm"

import (
	"fmt"
	"log"
	"os"
	"sync"

	// Import to initialize client auth plugins.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/kubernetes/pkg/kubectl/genericclioptions"

	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/storage/driver"
)

var (
	settings   environment.EnvSettings
	config     genericclioptions.RESTClientGetter
	configOnce sync.Once
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func logf(format string, v ...interface{}) {
	if settings.Debug {
		format = fmt.Sprintf("[debug] %s\n", format)
		log.Output(2, fmt.Sprintf(format, v...))
	}
}

func main() {
	cmd := newRootCmd(nil, os.Stdout, os.Args[1:])
	if err := cmd.Execute(); err != nil {
		logf("%+v", err)
		os.Exit(1)
	}
}

// ensureHelmClient returns a new helm client impl. if h is not nil.
func ensureHelmClient(h helm.Interface, allNamespaces bool) helm.Interface {
	if h != nil {
		return h
	}
	return newClient(allNamespaces)
}

func newClient(allNamespaces bool) helm.Interface {
	kc := kube.New(kubeConfig())
	kc.Log = logf

	clientset, err := kc.KubernetesClientSet()
	if err != nil {
		// TODO return error
		log.Fatal(err)
	}
	var namespace string
	if !allNamespaces {
		namespace = getNamespace()
	}
	// TODO add other backends
	d := driver.NewSecrets(clientset.CoreV1().Secrets(namespace))
	d.Log = logf

	return helm.NewClient(
		helm.KubeClient(kc),
		helm.Driver(d),
		helm.Discovery(clientset.Discovery()),
	)
}

func kubeConfig() genericclioptions.RESTClientGetter {
	configOnce.Do(func() {
		config = kube.GetConfig(settings.KubeConfig, settings.KubeContext, settings.Namespace)
	})
	return config
}

func getNamespace() string {
	if ns, _, err := kubeConfig().ToRawKubeConfigLoader().Namespace(); err == nil {
		return ns
	}
	return "default"
}
