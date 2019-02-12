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
	"reflect"
	"testing"

	"k8s.io/helm/pkg/hapi"
	rpb "k8s.io/helm/pkg/hapi/release"
)

func TestGetHistory_WithRevisions(t *testing.T) {
	mk := func(name string, vers int, status rpb.Status) *rpb.Release {
		return &rpb.Release{
			Name:    name,
			Version: vers,
			Info:    &rpb.Info{Status: status},
		}
	}

	// GetReleaseHistoryTests
	tests := []struct {
		desc string
		req  *hapi.GetHistoryRequest
		res  []*rpb.Release
	}{
		{
			desc: "get release with history and default limit (max=256)",
			req:  &hapi.GetHistoryRequest{Name: "angry-bird", Max: 256},
			res: []*rpb.Release{
				mk("angry-bird", 4, rpb.StatusDeployed),
				mk("angry-bird", 3, rpb.StatusSuperseded),
				mk("angry-bird", 2, rpb.StatusSuperseded),
				mk("angry-bird", 1, rpb.StatusSuperseded),
			},
		},
		{
			desc: "get release with history using result limit (max=2)",
			req:  &hapi.GetHistoryRequest{Name: "angry-bird", Max: 2},
			res: []*rpb.Release{
				mk("angry-bird", 4, rpb.StatusDeployed),
				mk("angry-bird", 3, rpb.StatusSuperseded),
			},
		},
	}

	// test release history for release 'angry-bird'
	hist := []*rpb.Release{
		mk("angry-bird", 4, rpb.StatusDeployed),
		mk("angry-bird", 3, rpb.StatusSuperseded),
		mk("angry-bird", 2, rpb.StatusSuperseded),
		mk("angry-bird", 1, rpb.StatusSuperseded),
	}

	srv := rsFixture(t)
	for _, rls := range hist {
		if err := srv.Releases.Create(rls); err != nil {
			t.Fatalf("Failed to create release: %s", err)
		}
	}

	// run tests
	for _, tt := range tests {
		res, err := srv.GetHistory(tt.req)
		if err != nil {
			t.Fatalf("%s:\nFailed to get History of %q: %s", tt.desc, tt.req.Name, err)
		}
		if !reflect.DeepEqual(res, tt.res) {
			t.Fatalf("%s:\nExpected:\n\t%+v\nActual\n\t%+v", tt.desc, tt.res, res)
		}
	}
}

func TestGetHistory_WithNoRevisions(t *testing.T) {
	tests := []struct {
		desc string
		req  *hapi.GetHistoryRequest
	}{
		{
			desc: "get release with no history",
			req:  &hapi.GetHistoryRequest{Name: "sad-panda", Max: 256},
		},
	}

	// create release 'sad-panda' with no revision history
	rls := namedReleaseStub("sad-panda", rpb.StatusDeployed)
	srv := rsFixture(t)
	srv.Releases.Create(rls)

	for _, tt := range tests {
		res, err := srv.GetHistory(tt.req)
		if err != nil {
			t.Fatalf("%s:\nFailed to get History of %q: %s", tt.desc, tt.req.Name, err)
		}
		if len(res) > 1 {
			t.Fatalf("%s:\nExpected zero items, got %d", tt.desc, len(res))
		}
	}
}
