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

package helm // import "k8s.io/helm/pkg/helm"

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pkg/errors"

	cpb "k8s.io/helm/pkg/chart"
	"k8s.io/helm/pkg/chart/loader"
	"k8s.io/helm/pkg/hapi"
	rls "k8s.io/helm/pkg/hapi/release"
)

// Path to example charts relative to pkg/helm.
const chartsDir = "../../docs/examples/"

// Sentinel error to indicate to the Helm client to not send the request to Tiller.
var errSkip = errors.New("test: skip")

// Verify each ReleaseListOption is applied to a ListReleasesRequest correctly.
func TestListReleases_VerifyOptions(t *testing.T) {
	// Options testdata
	var limit = 2
	var offset = "offset"
	var filter = "filter"
	var sortBy = hapi.SortByLastReleased
	var sortOrd = hapi.SortAsc
	var codes = []rls.ReleaseStatus{
		rls.StatusFailed,
		rls.StatusUninstalled,
		rls.StatusDeployed,
		rls.StatusSuperseded,
	}

	// Expected ListReleasesRequest message
	exp := &hapi.ListReleasesRequest{
		Limit:       int64(limit),
		Offset:      offset,
		Filter:      filter,
		SortBy:      sortBy,
		SortOrder:   sortOrd,
		StatusCodes: codes,
	}

	// Options used in ListReleases
	ops := []ReleaseListOption{
		ReleaseListSort(sortBy),
		ReleaseListOrder(sortOrd),
		ReleaseListLimit(limit),
		ReleaseListOffset(offset),
		ReleaseListFilter(filter),
		ReleaseListStatuses(codes),
	}

	// BeforeCall option to intercept Helm client ListReleasesRequest
	b4c := BeforeCall(func(msg interface{}) error {
		switch act := msg.(type) {
		case *hapi.ListReleasesRequest:
			t.Logf("ListReleasesRequest: %#+v\n", act)
			assert(t, exp, act)
		default:
			t.Fatalf("expected message of type ListReleasesRequest, got %T\n", act)
		}
		return errSkip
	})

	client := NewClient(b4c)

	if _, err := client.ListReleases(ops...); err != errSkip {
		t.Fatalf("did not expect error but got (%v)\n``", err)
	}

	// ensure options for call are not saved to client
	assert(t, "", client.opts.listReq.Filter)
}

// Verify each InstallOption is applied to an InstallReleaseRequest correctly.
func TestInstallRelease_VerifyOptions(t *testing.T) {
	// Options testdata
	var disableHooks = true
	var releaseName = "test"
	var reuseName = true
	var dryRun = true
	var chartName = "alpine"
	var chartPath = filepath.Join(chartsDir, chartName)
	var overrides = []byte("key1=value1,key2=value2")

	// Expected InstallReleaseRequest message
	exp := &hapi.InstallReleaseRequest{
		Chart:        loadChart(t, chartName),
		Values:       overrides,
		DryRun:       dryRun,
		Name:         releaseName,
		DisableHooks: disableHooks,
		ReuseName:    reuseName,
	}

	// Options used in InstallRelease
	ops := []InstallOption{
		ValueOverrides(overrides),
		InstallDryRun(dryRun),
		ReleaseName(releaseName),
		InstallReuseName(reuseName),
		InstallDisableHooks(disableHooks),
	}

	// BeforeCall option to intercept Helm client InstallReleaseRequest
	b4c := BeforeCall(func(msg interface{}) error {
		switch act := msg.(type) {
		case *hapi.InstallReleaseRequest:
			t.Logf("InstallReleaseRequest: %#+v\n", act)
			assert(t, exp, act)
		default:
			t.Fatalf("expected message of type InstallReleaseRequest, got %T\n", act)
		}
		return errSkip
	})

	client := NewClient(b4c)
	if _, err := client.InstallRelease(chartPath, "", ops...); err != errSkip {
		t.Fatalf("did not expect error but got (%v)\n``", err)
	}

	// ensure options for call are not saved to client
	assert(t, "", client.opts.instReq.Name)
}

// Verify each UninstallOptions is applied to an UninstallReleaseRequest correctly.
func TestUninstallRelease_VerifyOptions(t *testing.T) {
	// Options testdata
	var releaseName = "test"
	var disableHooks = true
	var purgeFlag = true

	// Expected UninstallReleaseRequest message
	exp := &hapi.UninstallReleaseRequest{
		Name:         releaseName,
		Purge:        purgeFlag,
		DisableHooks: disableHooks,
	}

	// Options used in UninstallRelease
	ops := []UninstallOption{
		UninstallPurge(purgeFlag),
		UninstallDisableHooks(disableHooks),
	}

	// BeforeCall option to intercept Helm client UninstallReleaseRequest
	b4c := BeforeCall(func(msg interface{}) error {
		switch act := msg.(type) {
		case *hapi.UninstallReleaseRequest:
			t.Logf("UninstallReleaseRequest: %#+v\n", act)
			assert(t, exp, act)
		default:
			t.Fatalf("expected message of type UninstallReleaseRequest, got %T\n", act)
		}
		return errSkip
	})

	client := NewClient(b4c)
	if _, err := client.UninstallRelease(releaseName, ops...); err != errSkip {
		t.Fatalf("did not expect error but got (%v)\n``", err)
	}

	// ensure options for call are not saved to client
	assert(t, "", client.opts.uninstallReq.Name)
}

// Verify each UpdateOption is applied to an UpdateReleaseRequest correctly.
func TestUpdateRelease_VerifyOptions(t *testing.T) {
	// Options testdata
	var chartName = "alpine"
	var chartPath = filepath.Join(chartsDir, chartName)
	var releaseName = "test"
	var disableHooks = true
	var overrides = []byte("key1=value1,key2=value2")
	var dryRun = false

	// Expected UpdateReleaseRequest message
	exp := &hapi.UpdateReleaseRequest{
		Name:         releaseName,
		Chart:        loadChart(t, chartName),
		Values:       overrides,
		DryRun:       dryRun,
		DisableHooks: disableHooks,
	}

	// Options used in UpdateRelease
	ops := []UpdateOption{
		UpgradeDryRun(dryRun),
		UpdateValueOverrides(overrides),
		UpgradeDisableHooks(disableHooks),
	}

	// BeforeCall option to intercept Helm client UpdateReleaseRequest
	b4c := BeforeCall(func(msg interface{}) error {
		switch act := msg.(type) {
		case *hapi.UpdateReleaseRequest:
			t.Logf("UpdateReleaseRequest: %#+v\n", act)
			assert(t, exp, act)
		default:
			t.Fatalf("expected message of type UpdateReleaseRequest, got %T\n", act)
		}
		return errSkip
	})

	client := NewClient(b4c)
	if _, err := client.UpdateRelease(releaseName, chartPath, ops...); err != errSkip {
		t.Fatalf("did not expect error but got (%v)\n``", err)
	}

	// ensure options for call are not saved to client
	assert(t, "", client.opts.updateReq.Name)
}

// Verify each RollbackOption is applied to a RollbackReleaseRequest correctly.
func TestRollbackRelease_VerifyOptions(t *testing.T) {
	// Options testdata
	var disableHooks = true
	var releaseName = "test"
	var revision = 2
	var dryRun = true

	// Expected RollbackReleaseRequest message
	exp := &hapi.RollbackReleaseRequest{
		Name:         releaseName,
		DryRun:       dryRun,
		Version:      revision,
		DisableHooks: disableHooks,
	}

	// Options used in RollbackRelease
	ops := []RollbackOption{
		RollbackDryRun(dryRun),
		RollbackVersion(revision),
		RollbackDisableHooks(disableHooks),
	}

	// BeforeCall option to intercept Helm client RollbackReleaseRequest
	b4c := BeforeCall(func(msg interface{}) error {
		switch act := msg.(type) {
		case *hapi.RollbackReleaseRequest:
			t.Logf("RollbackReleaseRequest: %#+v\n", act)
			assert(t, exp, act)
		default:
			t.Fatalf("expected message of type RollbackReleaseRequest, got %T\n", act)
		}
		return errSkip
	})

	client := NewClient(b4c)
	if _, err := client.RollbackRelease(releaseName, ops...); err != errSkip {
		t.Fatalf("did not expect error but got (%v)\n``", err)
	}

	// ensure options for call are not saved to client
	assert(t, "", client.opts.rollbackReq.Name)
}

// Verify each StatusOption is applied to a GetReleaseStatusRequest correctly.
func TestReleaseStatus_VerifyOptions(t *testing.T) {
	// Options testdata
	var releaseName = "test"
	var revision = 2

	// Expected GetReleaseStatusRequest message
	exp := &hapi.GetReleaseStatusRequest{
		Name:    releaseName,
		Version: revision,
	}

	// BeforeCall option to intercept Helm client GetReleaseStatusRequest
	b4c := BeforeCall(func(msg interface{}) error {
		switch act := msg.(type) {
		case *hapi.GetReleaseStatusRequest:
			t.Logf("GetReleaseStatusRequest: %#+v\n", act)
			assert(t, exp, act)
		default:
			t.Fatalf("expected message of type GetReleaseStatusRequest, got %T\n", act)
		}
		return errSkip
	})

	client := NewClient(b4c)
	if _, err := client.ReleaseStatus(releaseName, revision); err != errSkip {
		t.Fatalf("did not expect error but got (%v)\n``", err)
	}

	// ensure options for call are not saved to client
	assert(t, "", client.opts.statusReq.Name)
}

// Verify each ContentOption is applied to a GetReleaseContentRequest correctly.
func TestReleaseContent_VerifyOptions(t *testing.T) {
	t.Skip("refactoring out")
	// Options testdata
	var releaseName = "test"
	var revision = 2

	// Expected GetReleaseContentRequest message
	exp := &hapi.GetReleaseContentRequest{
		Name:    releaseName,
		Version: revision,
	}

	// BeforeCall option to intercept Helm client GetReleaseContentRequest
	b4c := BeforeCall(func(msg interface{}) error {
		switch act := msg.(type) {
		case *hapi.GetReleaseContentRequest:
			t.Logf("GetReleaseContentRequest: %#+v\n", act)
			assert(t, exp, act)
		default:
			t.Fatalf("expected message of type GetReleaseContentRequest, got %T\n", act)
		}
		return errSkip
	})

	client := NewClient(b4c)
	if _, err := client.ReleaseContent(releaseName, revision); err != errSkip {
		t.Fatalf("did not expect error but got (%v)\n``", err)
	}

	// ensure options for call are not saved to client
	assert(t, "", client.opts.contentReq.Name)
}

func assert(t *testing.T, expect, actual interface{}) {
	if !reflect.DeepEqual(expect, actual) {
		t.Fatalf("expected %#+v, actual %#+v\n", expect, actual)
	}
}

func loadChart(t *testing.T, name string) *cpb.Chart {
	c, err := loader.Load(filepath.Join(chartsDir, name))
	if err != nil {
		t.Fatalf("failed to load test chart (%q): %s\n", name, err)
	}
	return c
}
