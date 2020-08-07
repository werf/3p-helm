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

package helm_v3

import (
	"fmt"
	"path/filepath"
	"testing"
)

var chartPath = "./../../pkg/chartutil/testdata/subpop/charts/subchart1"

func TestTemplateCmd(t *testing.T) {
	tests := []cmdTestCase{
		{
			name:   "check name",
			cmd:    fmt.Sprintf("template '%s'", chartPath),
			golden: "output/template.txt",
		},
		{
			name:   "check set name",
			cmd:    fmt.Sprintf("template '%s' --set service.name=apache", chartPath),
			golden: "output/template-set.txt",
		},
		{
			name:   "check values files",
			cmd:    fmt.Sprintf("template '%s' --values '%s'", chartPath, filepath.Join(chartPath, "/charts/subchartA/values.yaml")),
			golden: "output/template-values-files.txt",
		},
		{
			name:   "check name template",
			cmd:    fmt.Sprintf(`template '%s' --name-template='foobar-{{ b64enc "abc" }}-baz'`, chartPath),
			golden: "output/template-name-template.txt",
		},
		{
			name:      "check no args",
			cmd:       "template",
			wantError: true,
			golden:    "output/template-no-args.txt",
		},
		{
			name:      "check library chart",
			cmd:       fmt.Sprintf("template '%s'", "testdata/testcharts/lib-chart"),
			wantError: true,
			golden:    "output/template-lib-chart.txt",
		},
		{
			name:      "check chart bad type",
			cmd:       fmt.Sprintf("template '%s'", "testdata/testcharts/chart-bad-type"),
			wantError: true,
			golden:    "output/install-chart-bad-type.txt",
		},
		{
			name:   "check chart with dependency which is an app chart acting as a library chart",
			cmd:    fmt.Sprintf("template '%s'", "testdata/testcharts/chart-with-template-lib-dep"),
			golden: "output/template-chart-with-template-lib-dep.txt",
		},
		{
			name:   "check chart with dependency which is an app chart archive acting as a library chart",
			cmd:    fmt.Sprintf("template '%s'", "testdata/testcharts/chart-with-template-lib-archive-dep"),
			golden: "output/template-chart-with-template-lib-archive-dep.txt",
		},
		{
			name:   "check kube api versions",
			cmd:    fmt.Sprintf("template --api-versions helm.k8s.io/test '%s'", chartPath),
			golden: "output/template-with-api-version.txt",
		},
		{
			name:   "template with CRDs",
			cmd:    fmt.Sprintf("template '%s' --include-crds", chartPath),
			golden: "output/template-with-crds.txt",
		},
		{
			name:   "template with show-only one",
			cmd:    fmt.Sprintf("template '%s' --show-only templates/service.yaml", chartPath),
			golden: "output/template-show-only-one.txt",
		},
		{
			name:   "template with show-only multiple",
			cmd:    fmt.Sprintf("template '%s' --show-only templates/service.yaml --show-only charts/subcharta/templates/service.yaml", chartPath),
			golden: "output/template-show-only-multiple.txt",
		},
		{
			name:   "template with show-only glob",
			cmd:    fmt.Sprintf("template '%s' --show-only templates/subdir/role*", chartPath),
			golden: "output/template-show-only-glob.txt",
			// Repeat to ensure manifest ordering regressions are caught
			repeat: 10,
		},
		{
			name:   "sorted output of manifests (order of filenames, then order of objects within each YAML file)",
			cmd:    fmt.Sprintf("template '%s'", "testdata/testcharts/object-order"),
			golden: "output/object-order.txt",
			// Helm previously used random file order. Repeat the test so we
			// don't accidentally get the expected result.
			repeat: 10,
		},
		{
			name:      "chart with template with invalid yaml",
			cmd:       fmt.Sprintf("template '%s'", "testdata/testcharts/chart-with-template-with-invalid-yaml"),
			wantError: true,
			golden:    "output/template-with-invalid-yaml.txt",
		},
		{
			name:      "chart with template with invalid yaml (--debug)",
			cmd:       fmt.Sprintf("template '%s' --debug", "testdata/testcharts/chart-with-template-with-invalid-yaml"),
			wantError: true,
			golden:    "output/template-with-invalid-yaml-debug.txt",
		},
	}
	runTestCmd(t, tests)
}
