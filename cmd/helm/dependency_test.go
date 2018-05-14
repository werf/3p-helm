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
	"testing"
)

func TestDependencyListCmd(t *testing.T) {
	tests := []cmdTestCase{{
		name:      "No such chart",
		cmd:       "dependency list /no/such/chart",
		golden:    "output/dependency-list-no-chart.txt",
		wantError: true,
	}, {
		name:   "No requirements.yaml",
		cmd:    "dependency list testdata/testcharts/alpine",
		golden: "output/dependency-list-no-requirements.txt",
	}, {
		name:   "Requirements in chart dir",
		cmd:    "dependency list testdata/testcharts/reqtest",
		golden: "output/dependency-list.txt",
	}, {
		name:   "Requirements in chart archive",
		cmd:    "dependency list testdata/testcharts/reqtest-0.1.0.tgz",
		golden: "output/dependency-list-archive.txt",
	}}
	runTestCmd(t, tests)
}
