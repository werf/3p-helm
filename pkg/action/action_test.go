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
package action

import (
	"flag"
	"io"
	"testing"

	fakeclientset "k8s.io/client-go/kubernetes/fake"

	"github.com/werf/3p-helm/for-werf-helm/pkg/chart"
	"github.com/werf/3p-helm/for-werf-helm/pkg/chartutil"
	kubefake "github.com/werf/3p-helm/for-werf-helm/pkg/kube/fake"
	"github.com/werf/3p-helm/for-werf-helm/pkg/registry"
	"github.com/werf/3p-helm/for-werf-helm/pkg/release"
	"github.com/werf/3p-helm/for-werf-helm/pkg/storage"
	"github.com/werf/3p-helm/for-werf-helm/pkg/storage/driver"
	"github.com/werf/3p-helm/for-werf-helm/pkg/time"
)

var verbose = flag.Bool("test.log", false, "enable test logging")

func actionConfigFixture(t *testing.T) *Configuration {
	t.Helper()

	registryClient, err := registry.NewClient()
	if err != nil {
		t.Fatal(err)
	}

	return &Configuration{
		Releases:       storage.Init(driver.NewMemory()),
		KubeClient:     &kubefake.FailingKubeClient{PrintingKubeClient: kubefake.PrintingKubeClient{Out: io.Discard}},
		Capabilities:   chartutil.DefaultCapabilities,
		RegistryClient: registryClient,
		Log: func(format string, v ...interface{}) {
			t.Helper()
			if *verbose {
				t.Logf(format, v...)
			}
		},
	}
}

var manifestWithHook = `kind: ConfigMap
metadata:
  name: test-cm
  annotations:
    "helm.sh/hook": post-install,pre-delete,post-upgrade
data:
  name: value`

var manifestWithTestHook = `kind: Pod
  metadata:
	name: finding-nemo,
	annotations:
	  "helm.sh/hook": test
  spec:
	containers:
	- name: nemo-test
	  image: fake-image
	  cmd: fake-command
  `

var rbacManifests = `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: schedule-agents
rules:
- apiGroups: [""]
  resources: ["pods", "pods/exec", "pods/log"]
  verbs: ["*"]

---

apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: schedule-agents
  namespace: {{ default .Release.Namespace}}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: schedule-agents
subjects:
- kind: ServiceAccount
  name: schedule-agents
  namespace: {{ .Release.Namespace }}
`

type chartOptions struct {
	*chart.Chart
}

type chartOption func(*chartOptions)

func buildChart(opts ...chartOption) *chart.Chart {
	c := &chartOptions{
		Chart: &chart.Chart{
			// TODO: This should be more complete.
			Metadata: &chart.Metadata{
				APIVersion: "v1",
				Name:       "hello",
				Version:    "0.1.0",
			},
			// This adds a basic template and hooks.
			Templates: []*chart.File{
				{Name: "templates/hello", Data: []byte("hello: world")},
				{Name: "templates/hooks", Data: []byte(manifestWithHook)},
			},
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c.Chart
}

func withName(name string) chartOption {
	return func(opts *chartOptions) {
		opts.Metadata.Name = name
	}
}

func withSampleValues() chartOption {
	values := map[string]interface{}{
		"someKey": "someValue",
		"nestedKey": map[string]interface{}{
			"simpleKey": "simpleValue",
			"anotherNestedKey": map[string]interface{}{
				"yetAnotherNestedKey": map[string]interface{}{
					"youReadyForAnotherNestedKey": "No",
				},
			},
		},
	}
	return func(opts *chartOptions) {
		opts.Values = values
	}
}

func withValues(values map[string]interface{}) chartOption {
	return func(opts *chartOptions) {
		opts.Values = values
	}
}

func withNotes(notes string) chartOption {
	return func(opts *chartOptions) {
		opts.Templates = append(opts.Templates, &chart.File{
			Name: "templates/NOTES.txt",
			Data: []byte(notes),
		})
	}
}

func withDependency(dependencyOpts ...chartOption) chartOption {
	return func(opts *chartOptions) {
		opts.AddDependency(buildChart(dependencyOpts...))
	}
}

func withMetadataDependency(dependency chart.Dependency) chartOption {
	return func(opts *chartOptions) {
		opts.Metadata.Dependencies = append(opts.Metadata.Dependencies, &dependency)
	}
}

func withSampleTemplates() chartOption {
	return func(opts *chartOptions) {
		sampleTemplates := []*chart.File{
			// This adds basic templates and partials.
			{Name: "templates/goodbye", Data: []byte("goodbye: world")},
			{Name: "templates/empty", Data: []byte("")},
			{Name: "templates/with-partials", Data: []byte(`hello: {{ template "_planet" . }}`)},
			{Name: "templates/partials/_planet", Data: []byte(`{{define "_planet"}}Earth{{end}}`)},
		}
		opts.Templates = append(opts.Templates, sampleTemplates...)
	}
}

func withSampleIncludingIncorrectTemplates() chartOption {
	return func(opts *chartOptions) {
		sampleTemplates := []*chart.File{
			// This adds basic templates and partials.
			{Name: "templates/goodbye", Data: []byte("goodbye: world")},
			{Name: "templates/empty", Data: []byte("")},
			{Name: "templates/incorrect", Data: []byte("{{ .Values.bad.doh }}")},
			{Name: "templates/with-partials", Data: []byte(`hello: {{ template "_planet" . }}`)},
			{Name: "templates/partials/_planet", Data: []byte(`{{define "_planet"}}Earth{{end}}`)},
		}
		opts.Templates = append(opts.Templates, sampleTemplates...)
	}
}

func withMultipleManifestTemplate() chartOption {
	return func(opts *chartOptions) {
		sampleTemplates := []*chart.File{
			{Name: "templates/rbac", Data: []byte(rbacManifests)},
		}
		opts.Templates = append(opts.Templates, sampleTemplates...)
	}
}

func withKube(version string) chartOption {
	return func(opts *chartOptions) {
		opts.Metadata.KubeVersion = version
	}
}

// releaseStub creates a release stub, complete with the chartStub as its chart.
func releaseStub() *release.Release {
	return namedReleaseStub("angry-panda", release.StatusDeployed)
}

func namedReleaseStub(name string, status release.Status) *release.Release {
	now := time.Now()
	return &release.Release{
		Name: name,
		Info: &release.Info{
			FirstDeployed: now,
			LastDeployed:  now,
			Status:        status,
			Description:   "Named Release Stub",
		},
		Chart:   buildChart(withSampleTemplates()),
		Config:  map[string]interface{}{"name": "value"},
		Version: 1,
		Hooks: []*release.Hook{
			{
				Name:     "test-cm",
				Kind:     "ConfigMap",
				Path:     "test-cm",
				Manifest: manifestWithHook,
				Events: []release.HookEvent{
					release.HookPostInstall,
					release.HookPreDelete,
				},
			},
			{
				Name:     "finding-nemo",
				Kind:     "Pod",
				Path:     "finding-nemo",
				Manifest: manifestWithTestHook,
				Events: []release.HookEvent{
					release.HookTest,
				},
			},
		},
	}
}

func TestGetVersionSet(t *testing.T) {
	client := fakeclientset.NewSimpleClientset()

	vs, err := GetVersionSet(client.Discovery())
	if err != nil {
		t.Error(err)
	}

	if !vs.Has("v1") {
		t.Errorf("Expected supported versions to at least include v1.")
	}
	if vs.Has("nosuchversion/v1") {
		t.Error("Non-existent version is reported found.")
	}
}
