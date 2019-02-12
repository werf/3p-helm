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

package tiller

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions/resource"
	"k8s.io/client-go/kubernetes/fake"

	"k8s.io/helm/pkg/chart"
	"k8s.io/helm/pkg/engine"
	"k8s.io/helm/pkg/hapi"
	"k8s.io/helm/pkg/hapi/release"
	"k8s.io/helm/pkg/hooks"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/tiller/environment"
)

var verbose = flag.Bool("test.log", false, "enable tiller logging")

const notesText = "my notes here"

var manifestWithHook = `kind: ConfigMap
metadata:
  name: test-cm
  annotations:
    "helm.sh/hook": post-install,pre-delete
data:
  name: value`

var manifestWithTestHook = `kind: Pod
metadata:
  name: finding-nemo,
  annotations:
    "helm.sh/hook": test-success
spec:
  containers:
  - name: nemo-test
    image: fake-image
    cmd: fake-command
`

var manifestWithKeep = `kind: ConfigMap
metadata:
  name: test-cm-keep
  annotations:
    "helm.sh/resource-policy": keep
data:
  name: value
`

var manifestWithUpgradeHooks = `kind: ConfigMap
metadata:
  name: test-cm
  annotations:
    "helm.sh/hook": post-upgrade,pre-upgrade
data:
  name: value`

var manifestWithRollbackHooks = `kind: ConfigMap
metadata:
  name: test-cm
  annotations:
    "helm.sh/hook": post-rollback,pre-rollback
data:
  name: value
`

func rsFixture(t *testing.T) *ReleaseServer {
	t.Helper()

	dc := fake.NewSimpleClientset().Discovery()
	kc := &environment.PrintingKubeClient{Out: ioutil.Discard}
	rs := NewReleaseServer(dc, kc)
	rs.Log = func(format string, v ...interface{}) {
		t.Helper()
		if *verbose {
			t.Logf(format, v...)
		}
	}
	return rs
}

type chartOptions struct {
	*chart.Chart
}

type chartOption func(*chartOptions)

func buildChart(opts ...chartOption) *chart.Chart {
	c := &chartOptions{
		Chart: &chart.Chart{
			// TODO: This should be more complete.
			Metadata: &chart.Metadata{
				Name: "hello",
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

func withKube(version string) chartOption {
	return func(opts *chartOptions) {
		opts.Metadata.KubeVersion = version
	}
}

func withDependency(dependencyOpts ...chartOption) chartOption {
	return func(opts *chartOptions) {
		opts.AddDependency(buildChart(dependencyOpts...))
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

type installOptions struct {
	*hapi.InstallReleaseRequest
}

type installOption func(*installOptions)

func withName(name string) installOption {
	return func(opts *installOptions) {
		opts.Name = name
	}
}

func withDryRun() installOption {
	return func(opts *installOptions) {
		opts.DryRun = true
	}
}

func withDisabledHooks() installOption {
	return func(opts *installOptions) {
		opts.DisableHooks = true
	}
}

func withReuseName() installOption {
	return func(opts *installOptions) {
		opts.ReuseName = true
	}
}

func withChart(chartOpts ...chartOption) installOption {
	return func(opts *installOptions) {
		opts.Chart = buildChart(chartOpts...)
	}
}

func installRequest(opts ...installOption) *hapi.InstallReleaseRequest {
	reqOpts := &installOptions{
		&hapi.InstallReleaseRequest{
			Namespace: "spaced",
			Chart:     buildChart(),
		},
	}

	for _, opt := range opts {
		opt(reqOpts)
	}

	return reqOpts.InstallReleaseRequest
}

// chartStub creates a fully stubbed out chart.
func chartStub() *chart.Chart {
	return buildChart(withSampleTemplates())
}

// releaseStub creates a release stub, complete with the chartStub as its chart.
func releaseStub() *release.Release {
	return namedReleaseStub("angry-panda", release.StatusDeployed)
}

func namedReleaseStub(name string, status release.Status) *release.Release {
	return &release.Release{
		Name: name,
		Info: &release.Info{
			FirstDeployed: time.Now(),
			LastDeployed:  time.Now(),
			Status:        status,
			Description:   "Named Release Stub",
		},
		Chart:   chartStub(),
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
					release.HookReleaseTestSuccess,
				},
			},
		},
	}
}

func upgradeReleaseVersion(rel *release.Release) *release.Release {
	rel.Info.Status = release.StatusSuperseded
	return &release.Release{
		Name: rel.Name,
		Info: &release.Info{
			FirstDeployed: rel.Info.FirstDeployed,
			LastDeployed:  time.Now(),
			Status:        release.StatusDeployed,
		},
		Chart:   rel.Chart,
		Config:  rel.Config,
		Version: rel.Version + 1,
	}
}

func TestValidName(t *testing.T) {
	for name, valid := range map[string]error{
		"nina pinta santa-maria": errInvalidName,
		"nina-pinta-santa-maria": nil,
		"-nina":                  errInvalidName,
		"pinta-":                 errInvalidName,
		"santa-maria":            nil,
		"niña":                   errInvalidName,
		"...":                    errInvalidName,
		"pinta...":               errInvalidName,
		"santa...maria":          nil,
		"":                       errMissingRelease,
		" ":                      errInvalidName,
		".nina.":                 errInvalidName,
		"nina.pinta":             nil,
		"abcdefghi-abcdefghi-abcdefghi-abcdefghi-abcdefghi-abcd": errInvalidName,
	} {
		if valid != validateReleaseName(name) {
			t.Errorf("Expected %q to be %s", name, valid)
		}
	}
}

func TestUniqName(t *testing.T) {
	rs := rsFixture(t)

	rel1 := releaseStub()
	rel2 := releaseStub()
	rel2.Name = "happy-panda"
	rel2.Info.Status = release.StatusUninstalled

	rs.Releases.Create(rel1)
	rs.Releases.Create(rel2)

	tests := []struct {
		name   string
		expect string
		reuse  bool
		err    bool
	}{
		{"", "", false, true}, // Blank name is illegal
		{"first", "first", false, false},
		{"angry-panda", "", false, true},
		{"happy-panda", "", false, true},
		{"happy-panda", "happy-panda", true, false},
		{"hungry-hungry-hungry-hungry-hungry-hungry-hungry-hungry-hippos", "", true, true}, // Exceeds max name length
	}

	for _, tt := range tests {
		u, err := rs.uniqName(tt.name, tt.reuse)
		if err != nil {
			if tt.err {
				continue
			}
			t.Fatal(err)
		}
		if tt.err {
			t.Errorf("Expected an error for %q", tt.name)
		}
		if match, err := regexp.MatchString(tt.expect, u); err != nil {
			t.Fatal(err)
		} else if !match {
			t.Errorf("Expected %q to match %q", u, tt.expect)
		}
	}
}

func releaseWithKeepStub(rlsName string) *release.Release {
	ch := &chart.Chart{
		Metadata: &chart.Metadata{
			Name: "bunnychart",
		},
		Templates: []*chart.File{
			{Name: "templates/configmap", Data: []byte(manifestWithKeep)},
		},
	}

	return &release.Release{
		Name: rlsName,
		Info: &release.Info{
			FirstDeployed: time.Now(),
			LastDeployed:  time.Now(),
			Status:        release.StatusDeployed,
		},
		Chart:    ch,
		Config:   map[string]interface{}{"name": "value"},
		Version:  1,
		Manifest: manifestWithKeep,
	}
}

func newUpdateFailingKubeClient() *updateFailingKubeClient {
	return &updateFailingKubeClient{
		PrintingKubeClient: environment.PrintingKubeClient{Out: os.Stdout},
	}

}

type updateFailingKubeClient struct {
	environment.PrintingKubeClient
}

func (u *updateFailingKubeClient) Update(namespace string, originalReader, modifiedReader io.Reader, force, recreate bool, timeout int64, shouldWait bool) error {
	return errors.New("Failed update in kube client")
}

func newHookFailingKubeClient() *hookFailingKubeClient {
	return &hookFailingKubeClient{
		PrintingKubeClient: environment.PrintingKubeClient{Out: ioutil.Discard},
	}
}

type hookFailingKubeClient struct {
	environment.PrintingKubeClient
}

func (h *hookFailingKubeClient) WatchUntilReady(ns string, r io.Reader, timeout int64, shouldWait bool) error {
	return errors.New("Failed watch")
}

type mockHooksManifest struct {
	Metadata struct {
		Name        string
		Annotations map[string]string
	}
}
type mockHooksKubeClient struct {
	Resources map[string]*mockHooksManifest
}

var errResourceExists = errors.New("resource already exists")

func (kc *mockHooksKubeClient) makeManifest(r io.Reader) (*mockHooksManifest, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	manifest := &mockHooksManifest{}
	if err = yaml.Unmarshal(b, manifest); err != nil {
		return nil, err
	}

	return manifest, nil
}
func (kc *mockHooksKubeClient) Create(ns string, r io.Reader, timeout int64, shouldWait bool) error {
	manifest, err := kc.makeManifest(r)
	if err != nil {
		return err
	}

	if _, hasKey := kc.Resources[manifest.Metadata.Name]; hasKey {
		return errResourceExists
	}

	kc.Resources[manifest.Metadata.Name] = manifest

	return nil
}
func (kc *mockHooksKubeClient) Get(ns string, r io.Reader) (string, error) {
	return "", nil
}
func (kc *mockHooksKubeClient) Delete(ns string, r io.Reader) error {
	manifest, err := kc.makeManifest(r)
	if err != nil {
		return err
	}

	delete(kc.Resources, manifest.Metadata.Name)

	return nil
}
func (kc *mockHooksKubeClient) WatchUntilReady(ns string, r io.Reader, timeout int64, shouldWait bool) error {
	paramManifest, err := kc.makeManifest(r)
	if err != nil {
		return err
	}

	manifest, hasManifest := kc.Resources[paramManifest.Metadata.Name]
	if !hasManifest {
		return errors.Errorf("mockHooksKubeClient.WatchUntilReady: no such resource %s found", paramManifest.Metadata.Name)
	}

	if manifest.Metadata.Annotations["mockHooksKubeClient/Emulate"] == "hook-failed" {
		return errors.Errorf("mockHooksKubeClient.WatchUntilReady: hook-failed")
	}

	return nil
}
func (kc *mockHooksKubeClient) Update(_ string, _, _ io.Reader, _, _ bool, _ int64, _ bool) error {
	return nil
}
func (kc *mockHooksKubeClient) Build(_ string, _ io.Reader) (kube.Result, error) {
	return []*resource.Info{}, nil
}
func (kc *mockHooksKubeClient) BuildUnstructured(_ string, _ io.Reader) (kube.Result, error) {
	return []*resource.Info{}, nil
}
func (kc *mockHooksKubeClient) WaitAndGetCompletedPodPhase(_ string, _ io.Reader, _ time.Duration) (v1.PodPhase, error) {
	return v1.PodUnknown, nil
}

func deletePolicyStub(kubeClient *mockHooksKubeClient) *ReleaseServer {
	return &ReleaseServer{
		engine:     engine.New(),
		discovery:  fake.NewSimpleClientset().Discovery(),
		KubeClient: kubeClient,
		Log:        func(_ string, _ ...interface{}) {},
	}
}

func deletePolicyHookStub(hookName string, extraAnnotations map[string]string, DeletePolicies []release.HookDeletePolicy) *release.Hook {
	extraAnnotationsStr := ""
	for k, v := range extraAnnotations {
		extraAnnotationsStr += fmt.Sprintf("    \"%s\": \"%s\"\n", k, v)
	}

	return &release.Hook{
		Name: hookName,
		Kind: "Job",
		Path: hookName,
		Manifest: fmt.Sprintf(`kind: Job
metadata:
  name: %s
  annotations:
    "helm.sh/hook": pre-install,pre-upgrade
%sdata:
name: value`, hookName, extraAnnotationsStr),
		Events: []release.HookEvent{
			release.HookPreInstall,
			release.HookPreUpgrade,
		},
		DeletePolicies: DeletePolicies,
	}
}

func execHookShouldSucceed(rs *ReleaseServer, hook *release.Hook, releaseName, namespace, hookType string) error {
	err := rs.execHook([]*release.Hook{hook}, releaseName, namespace, hookType, 600)
	return errors.Wrapf(err, "expected hook %s to be successful", hook.Name)
}

func execHookShouldFail(rs *ReleaseServer, hook *release.Hook, releaseName, namespace, hookType string) error {
	if err := rs.execHook([]*release.Hook{hook}, releaseName, namespace, hookType, 600); err == nil {
		return errors.Errorf("expected hook %s to be failed", hook.Name)
	}
	return nil
}

func execHookShouldFailWithError(rs *ReleaseServer, hook *release.Hook, releaseName, namespace, hookType string, expectedError error) error {
	err := rs.execHook([]*release.Hook{hook}, releaseName, namespace, hookType, 600)
	if cause := errors.Cause(err); cause != expectedError {
		return errors.Errorf("expected hook %s to fail with error \n%v \ngot \n%v", hook.Name, expectedError, cause)
	}
	return nil
}

type deletePolicyContext struct {
	ReleaseServer *ReleaseServer
	ReleaseName   string
	Namespace     string
	HookName      string
	KubeClient    *mockHooksKubeClient
}

func newDeletePolicyContext() *deletePolicyContext {
	kubeClient := &mockHooksKubeClient{
		Resources: make(map[string]*mockHooksManifest),
	}

	return &deletePolicyContext{
		KubeClient:    kubeClient,
		ReleaseServer: deletePolicyStub(kubeClient),
		ReleaseName:   "flying-carp",
		Namespace:     "river",
		HookName:      "migration-job",
	}
}

func TestSuccessfulHookWithoutDeletePolicy(t *testing.T) {
	ctx := newDeletePolicyContext()
	hook := deletePolicyHookStub(ctx.HookName, nil, nil)

	if err := execHookShouldSucceed(ctx.ReleaseServer, hook, ctx.ReleaseName, ctx.Namespace, hooks.PreInstall); err != nil {
		t.Error(err)
	}
	if _, hasResource := ctx.KubeClient.Resources[hook.Name]; !hasResource {
		t.Errorf("expected resource %s to be created by kube client", hook.Name)
	}
}

func TestFailedHookWithoutDeletePolicy(t *testing.T) {
	ctx := newDeletePolicyContext()
	hook := deletePolicyHookStub(ctx.HookName,
		map[string]string{"mockHooksKubeClient/Emulate": "hook-failed"},
		nil,
	)

	if err := execHookShouldFail(ctx.ReleaseServer, hook, ctx.ReleaseName, ctx.Namespace, hooks.PreInstall); err != nil {
		t.Error(err)
	}
	if _, hasResource := ctx.KubeClient.Resources[hook.Name]; !hasResource {
		t.Errorf("expected resource %s to be created by kube client", hook.Name)
	}
}

func TestSuccessfulHookWithSucceededDeletePolicy(t *testing.T) {
	ctx := newDeletePolicyContext()
	hook := deletePolicyHookStub(ctx.HookName,
		map[string]string{"helm.sh/hook-delete-policy": "hook-succeeded"},
		[]release.HookDeletePolicy{release.HookSucceeded},
	)

	if err := execHookShouldSucceed(ctx.ReleaseServer, hook, ctx.ReleaseName, ctx.Namespace, hooks.PreInstall); err != nil {
		t.Error(err)
	}
	if _, hasResource := ctx.KubeClient.Resources[hook.Name]; hasResource {
		t.Errorf("expected resource %s to be unexisting after hook succeeded", hook.Name)
	}
}

func TestSuccessfulHookWithFailedDeletePolicy(t *testing.T) {
	ctx := newDeletePolicyContext()
	hook := deletePolicyHookStub(ctx.HookName,
		map[string]string{"helm.sh/hook-delete-policy": "hook-failed"},
		[]release.HookDeletePolicy{release.HookFailed},
	)

	if err := execHookShouldSucceed(ctx.ReleaseServer, hook, ctx.ReleaseName, ctx.Namespace, hooks.PreInstall); err != nil {
		t.Error(err)
	}
	if _, hasResource := ctx.KubeClient.Resources[hook.Name]; !hasResource {
		t.Errorf("expected resource %s to be existing after hook succeeded", hook.Name)
	}
}

func TestFailedHookWithSucceededDeletePolicy(t *testing.T) {
	ctx := newDeletePolicyContext()

	hook := deletePolicyHookStub(ctx.HookName,
		map[string]string{
			"mockHooksKubeClient/Emulate": "hook-failed",
			"helm.sh/hook-delete-policy":  "hook-succeeded",
		},
		[]release.HookDeletePolicy{release.HookSucceeded},
	)

	if err := execHookShouldFail(ctx.ReleaseServer, hook, ctx.ReleaseName, ctx.Namespace, hooks.PreInstall); err != nil {
		t.Error(err)
	}
	if _, hasResource := ctx.KubeClient.Resources[hook.Name]; !hasResource {
		t.Errorf("expected resource %s to be existing after hook failed", hook.Name)
	}
}

func TestFailedHookWithFailedDeletePolicy(t *testing.T) {
	ctx := newDeletePolicyContext()

	hook := deletePolicyHookStub(ctx.HookName,
		map[string]string{
			"mockHooksKubeClient/Emulate": "hook-failed",
			"helm.sh/hook-delete-policy":  "hook-failed",
		},
		[]release.HookDeletePolicy{release.HookFailed},
	)

	if err := execHookShouldFail(ctx.ReleaseServer, hook, ctx.ReleaseName, ctx.Namespace, hooks.PreInstall); err != nil {
		t.Error(err)
	}
	if _, hasResource := ctx.KubeClient.Resources[hook.Name]; hasResource {
		t.Errorf("expected resource %s to be unexisting after hook failed", hook.Name)
	}
}

func TestSuccessfulHookWithSuccededOrFailedDeletePolicy(t *testing.T) {
	ctx := newDeletePolicyContext()

	hook := deletePolicyHookStub(ctx.HookName,
		map[string]string{
			"helm.sh/hook-delete-policy": "hook-succeeded,hook-failed",
		},
		[]release.HookDeletePolicy{release.HookSucceeded, release.HookFailed},
	)

	if err := execHookShouldSucceed(ctx.ReleaseServer, hook, ctx.ReleaseName, ctx.Namespace, hooks.PreInstall); err != nil {
		t.Error(err)
	}
	if _, hasResource := ctx.KubeClient.Resources[hook.Name]; hasResource {
		t.Errorf("expected resource %s to be unexisting after hook succeeded", hook.Name)
	}
}

func TestFailedHookWithSuccededOrFailedDeletePolicy(t *testing.T) {
	ctx := newDeletePolicyContext()

	hook := deletePolicyHookStub(ctx.HookName,
		map[string]string{
			"mockHooksKubeClient/Emulate": "hook-failed",
			"helm.sh/hook-delete-policy":  "hook-succeeded,hook-failed",
		},
		[]release.HookDeletePolicy{release.HookSucceeded, release.HookFailed},
	)

	if err := execHookShouldFail(ctx.ReleaseServer, hook, ctx.ReleaseName, ctx.Namespace, hooks.PreInstall); err != nil {
		t.Error(err)
	}
	if _, hasResource := ctx.KubeClient.Resources[hook.Name]; hasResource {
		t.Errorf("expected resource %s to be unexisting after hook failed", hook.Name)
	}
}

func TestHookAlreadyExists(t *testing.T) {
	ctx := newDeletePolicyContext()

	hook := deletePolicyHookStub(ctx.HookName, nil, nil)

	if err := execHookShouldSucceed(ctx.ReleaseServer, hook, ctx.ReleaseName, ctx.Namespace, hooks.PreInstall); err != nil {
		t.Error(err)
	}
	if _, hasResource := ctx.KubeClient.Resources[hook.Name]; !hasResource {
		t.Errorf("expected resource %s to be existing after hook succeeded", hook.Name)
	}
	if err := execHookShouldFailWithError(ctx.ReleaseServer, hook, ctx.ReleaseName, ctx.Namespace, hooks.PreUpgrade, errResourceExists); err != nil {
		t.Error(err)
	}
	if _, hasResource := ctx.KubeClient.Resources[hook.Name]; !hasResource {
		t.Errorf("expected resource %s to be existing after already exists error", hook.Name)
	}
}

func TestHookDeletingWithBeforeHookCreationDeletePolicy(t *testing.T) {
	ctx := newDeletePolicyContext()

	hook := deletePolicyHookStub(ctx.HookName,
		map[string]string{"helm.sh/hook-delete-policy": "before-hook-creation"},
		[]release.HookDeletePolicy{release.HookBeforeHookCreation},
	)

	if err := execHookShouldSucceed(ctx.ReleaseServer, hook, ctx.ReleaseName, ctx.Namespace, hooks.PreInstall); err != nil {
		t.Error(err)
	}
	if _, hasResource := ctx.KubeClient.Resources[hook.Name]; !hasResource {
		t.Errorf("expected resource %s to be existing after hook succeeded", hook.Name)
	}
	if err := execHookShouldSucceed(ctx.ReleaseServer, hook, ctx.ReleaseName, ctx.Namespace, hooks.PreUpgrade); err != nil {
		t.Error(err)
	}
	if _, hasResource := ctx.KubeClient.Resources[hook.Name]; !hasResource {
		t.Errorf("expected resource %s to be existing after hook succeeded", hook.Name)
	}
}

func TestSuccessfulHookWithMixedDeletePolicies(t *testing.T) {
	ctx := newDeletePolicyContext()

	hook := deletePolicyHookStub(ctx.HookName,
		map[string]string{
			"helm.sh/hook-delete-policy": "hook-succeeded,before-hook-creation",
		},
		[]release.HookDeletePolicy{release.HookSucceeded, release.HookBeforeHookCreation},
	)

	if err := execHookShouldSucceed(ctx.ReleaseServer, hook, ctx.ReleaseName, ctx.Namespace, hooks.PreInstall); err != nil {
		t.Error(err)
	}
	if _, hasResource := ctx.KubeClient.Resources[hook.Name]; hasResource {
		t.Errorf("expected resource %s to be unexisting after hook succeeded", hook.Name)
	}
	if err := execHookShouldSucceed(ctx.ReleaseServer, hook, ctx.ReleaseName, ctx.Namespace, hooks.PreUpgrade); err != nil {
		t.Error(err)
	}
	if _, hasResource := ctx.KubeClient.Resources[hook.Name]; hasResource {
		t.Errorf("expected resource %s to be unexisting after hook succeeded", hook.Name)
	}
}

func TestFailedHookWithMixedDeletePolicies(t *testing.T) {
	ctx := newDeletePolicyContext()

	hook := deletePolicyHookStub(ctx.HookName,
		map[string]string{
			"mockHooksKubeClient/Emulate": "hook-failed",
			"helm.sh/hook-delete-policy":  "hook-succeeded,before-hook-creation",
		},
		[]release.HookDeletePolicy{release.HookSucceeded, release.HookBeforeHookCreation},
	)

	if err := execHookShouldFail(ctx.ReleaseServer, hook, ctx.ReleaseName, ctx.Namespace, hooks.PreInstall); err != nil {
		t.Error(err)
	}
	if _, hasResource := ctx.KubeClient.Resources[hook.Name]; !hasResource {
		t.Errorf("expected resource %s to be existing after hook failed", hook.Name)
	}
	if err := execHookShouldFail(ctx.ReleaseServer, hook, ctx.ReleaseName, ctx.Namespace, hooks.PreUpgrade); err != nil {
		t.Error(err)
	}
	if _, hasResource := ctx.KubeClient.Resources[hook.Name]; !hasResource {
		t.Errorf("expected resource %s to be existing after hook failed", hook.Name)
	}
}

func TestFailedThenSuccessfulHookWithMixedDeletePolicies(t *testing.T) {
	ctx := newDeletePolicyContext()

	hook := deletePolicyHookStub(ctx.HookName,
		map[string]string{
			"mockHooksKubeClient/Emulate": "hook-failed",
			"helm.sh/hook-delete-policy":  "hook-succeeded,before-hook-creation",
		},
		[]release.HookDeletePolicy{release.HookSucceeded, release.HookBeforeHookCreation},
	)

	if err := execHookShouldFail(ctx.ReleaseServer, hook, ctx.ReleaseName, ctx.Namespace, hooks.PreInstall); err != nil {
		t.Error(err)
	}
	if _, hasResource := ctx.KubeClient.Resources[hook.Name]; !hasResource {
		t.Errorf("expected resource %s to be existing after hook failed", hook.Name)
	}

	hook = deletePolicyHookStub(ctx.HookName,
		map[string]string{
			"helm.sh/hook-delete-policy": "hook-succeeded,before-hook-creation",
		},
		[]release.HookDeletePolicy{release.HookSucceeded, release.HookBeforeHookCreation},
	)

	if err := execHookShouldSucceed(ctx.ReleaseServer, hook, ctx.ReleaseName, ctx.Namespace, hooks.PreUpgrade); err != nil {
		t.Error(err)
	}
	if _, hasResource := ctx.KubeClient.Resources[hook.Name]; hasResource {
		t.Errorf("expected resource %s to be unexisting after hook succeeded", hook.Name)
	}
}
