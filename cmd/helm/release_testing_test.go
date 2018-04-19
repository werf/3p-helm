/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package main

import (
	"io"
	"testing"

	"github.com/spf13/cobra"

	"k8s.io/helm/pkg/hapi/release"
	"k8s.io/helm/pkg/helm"
)

func TestReleaseTesting(t *testing.T) {
	tests := []releaseCase{
		{
			name:      "basic test",
			args:      []string{"example-release"},
			flags:     []string{},
			responses: map[string]release.TestRunStatus{"PASSED: green lights everywhere": release.TestRun_SUCCESS},
			err:       false,
		},
		{
			name:      "test failure",
			args:      []string{"example-fail"},
			flags:     []string{},
			responses: map[string]release.TestRunStatus{"FAILURE: red lights everywhere": release.TestRun_FAILURE},
			err:       true,
		},
		{
			name:      "test unknown",
			args:      []string{"example-unknown"},
			flags:     []string{},
			responses: map[string]release.TestRunStatus{"UNKNOWN: yellow lights everywhere": release.TestRun_UNKNOWN},
			err:       false,
		},
		{
			name:      "test error",
			args:      []string{"example-error"},
			flags:     []string{},
			responses: map[string]release.TestRunStatus{"ERROR: yellow lights everywhere": release.TestRun_FAILURE},
			err:       true,
		},
		{
			name:      "test running",
			args:      []string{"example-running"},
			flags:     []string{},
			responses: map[string]release.TestRunStatus{"RUNNING: things are happpeningggg": release.TestRun_RUNNING},
			err:       false,
		},
		{
			name:  "multiple tests example",
			args:  []string{"example-suite"},
			flags: []string{},
			responses: map[string]release.TestRunStatus{
				"RUNNING: things are happpeningggg":           release.TestRun_RUNNING,
				"PASSED: party time":                          release.TestRun_SUCCESS,
				"RUNNING: things are happening again":         release.TestRun_RUNNING,
				"FAILURE: good thing u checked :)":            release.TestRun_FAILURE,
				"RUNNING: things are happpeningggg yet again": release.TestRun_RUNNING,
				"PASSED: feel free to party again":            release.TestRun_SUCCESS},
			err: true,
		},
	}

	runReleaseCases(t, tests, func(c *helm.FakeClient, out io.Writer) *cobra.Command {
		return newReleaseTestCmd(c, out)
	})
}
