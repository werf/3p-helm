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
	"testing"
)

var (
	values                       = make(map[string]interface{})
	namespace                    = "testNamespace"
	strict                       = false
	archivedChartPath            = "../../cmd/helm/testdata/testcharts/compressedchart-0.1.0.tgz"
	archivedChartPathWithHyphens = "../../cmd/helm/testdata/testcharts/compressedchart-with-hyphens-0.1.0.tgz"
	invalidArchivedChartPath     = "../../cmd/helm/testdata/testcharts/invalidcompressedchart0.1.0.tgz"
	chartDirPath                 = "../../cmd/helm/testdata/testcharts/decompressedchart/"
	chartMissingManifest         = "../../cmd/helm/testdata/testcharts/chart-missing-manifest"
	chartSchema                  = "../../cmd/helm/testdata/testcharts/chart-with-schema"
	chartSchemaNegative          = "../../cmd/helm/testdata/testcharts/chart-with-schema-negative"
)

func TestLintChart(t *testing.T) {
	if _, err := lintChart(chartDirPath, values, namespace, strict); err != nil {
		t.Error(err)
	}
	if _, err := lintChart(archivedChartPath, values, namespace, strict); err != nil {
		t.Error(err)
	}
	if _, err := lintChart(archivedChartPathWithHyphens, values, namespace, strict); err != nil {
		t.Error(err)
	}
	if _, err := lintChart(invalidArchivedChartPath, values, namespace, strict); err == nil {
		t.Error("Expected a chart parsing error")
	}
	if _, err := lintChart(chartMissingManifest, values, namespace, strict); err == nil {
		t.Error("Expected a chart parsing error")
	}
	if _, err := lintChart(chartSchema, values, namespace, strict); err != nil {
		t.Error(err)
	}
	if _, err := lintChart(chartSchemaNegative, values, namespace, strict); err != nil {
		t.Error(err)
	}
}
