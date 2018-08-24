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
	"strings"
	"testing"

	"k8s.io/helm/pkg/hapi"
	"k8s.io/helm/pkg/hapi/release"
)

func TestUninstallRelease(t *testing.T) {
	rs := rsFixture(t)
	rs.Releases.Create(releaseStub())

	req := &hapi.UninstallReleaseRequest{
		Name: "angry-panda",
	}

	res, err := rs.UninstallRelease(req)
	if err != nil {
		t.Fatalf("Failed uninstall: %s", err)
	}

	if res.Release.Name != "angry-panda" {
		t.Errorf("Expected angry-panda, got %q", res.Release.Name)
	}

	if res.Release.Info.Status != release.StatusUninstalled {
		t.Errorf("Expected status code to be UNINSTALLED, got %s", res.Release.Info.Status)
	}

	if res.Release.Hooks[0].LastRun.IsZero() {
		t.Error("Expected LastRun to be greater than zero.")
	}

	if res.Release.Info.Deleted.Second() <= 0 {
		t.Errorf("Expected valid UNIX date, got %d", res.Release.Info.Deleted.Second())
	}

	if res.Release.Info.Description != "Uninstallation complete" {
		t.Errorf("Expected Uninstallation complete, got %q", res.Release.Info.Description)
	}
}

func TestUninstallPurgeRelease(t *testing.T) {
	rs := rsFixture(t)
	rel := releaseStub()
	rs.Releases.Create(rel)
	upgradedRel := upgradeReleaseVersion(rel)
	rs.Releases.Update(rel)
	rs.Releases.Create(upgradedRel)

	req := &hapi.UninstallReleaseRequest{
		Name:  "angry-panda",
		Purge: true,
	}

	res, err := rs.UninstallRelease(req)
	if err != nil {
		t.Fatalf("Failed uninstall: %s", err)
	}

	if res.Release.Name != "angry-panda" {
		t.Errorf("Expected angry-panda, got %q", res.Release.Name)
	}

	if res.Release.Info.Status != release.StatusUninstalled {
		t.Errorf("Expected status code to be UNINSTALLED, got %s", res.Release.Info.Status)
	}

	if res.Release.Info.Deleted.Second() <= 0 {
		t.Errorf("Expected valid UNIX date, got %d", res.Release.Info.Deleted.Second())
	}
	rels, err := rs.GetHistory(&hapi.GetHistoryRequest{Name: "angry-panda"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rels) != 0 {
		t.Errorf("Expected no releases in storage, got %d", len(rels))
	}
}

func TestUninstallPurgeDeleteRelease(t *testing.T) {
	rs := rsFixture(t)
	rs.Releases.Create(releaseStub())

	req := &hapi.UninstallReleaseRequest{
		Name: "angry-panda",
	}

	_, err := rs.UninstallRelease(req)
	if err != nil {
		t.Fatalf("Failed uninstall: %s", err)
	}

	req2 := &hapi.UninstallReleaseRequest{
		Name:  "angry-panda",
		Purge: true,
	}

	_, err2 := rs.UninstallRelease(req2)
	if err2 != nil && err2.Error() != "'angry-panda' has no deployed releases" {
		t.Errorf("Failed uninstall: %s", err2)
	}
}

func TestUninstallReleaseWithKeepPolicy(t *testing.T) {
	rs := rsFixture(t)
	name := "angry-bunny"
	rs.Releases.Create(releaseWithKeepStub(name))

	req := &hapi.UninstallReleaseRequest{
		Name: name,
	}

	res, err := rs.UninstallRelease(req)
	if err != nil {
		t.Fatalf("Failed uninstall: %s", err)
	}

	if res.Release.Name != name {
		t.Errorf("Expected angry-bunny, got %q", res.Release.Name)
	}

	if res.Release.Info.Status != release.StatusUninstalled {
		t.Errorf("Expected status code to be UNINSTALLED, got %s", res.Release.Info.Status)
	}

	if res.Info == "" {
		t.Errorf("Expected response info to not be empty")
	} else {
		if !strings.Contains(res.Info, "[ConfigMap] test-cm-keep") {
			t.Errorf("unexpected output: %s", res.Info)
		}
	}
}

func TestUninstallReleaseNoHooks(t *testing.T) {
	rs := rsFixture(t)
	rs.Releases.Create(releaseStub())

	req := &hapi.UninstallReleaseRequest{
		Name:         "angry-panda",
		DisableHooks: true,
	}

	res, err := rs.UninstallRelease(req)
	if err != nil {
		t.Errorf("Failed uninstall: %s", err)
	}

	// The default value for a protobuf timestamp is nil.
	if !res.Release.Hooks[0].LastRun.IsZero() {
		t.Errorf("Expected LastRun to be zero, got %s.", res.Release.Hooks[0].LastRun)
	}
}
