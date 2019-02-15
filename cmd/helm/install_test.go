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

package main

import (
	"reflect"
	"regexp"
	"testing"
)

func TestInstall(t *testing.T) {
	tests := []cmdTestCase{
		// Install, base case
		{
			name:   "basic install",
			cmd:    "install aeneas testdata/testcharts/empty",
			golden: "output/install.txt",
		},

		// Install, values from cli
		{
			name:   "install with values",
			cmd:    "install virgil testdata/testcharts/alpine --set test.Name=bar",
			golden: "output/install-with-values.txt",
		},
		// Install, values from cli via multiple --set
		{
			name:   "install with multiple values",
			cmd:    "install virgil testdata/testcharts/alpine --set test.Color=yellow --set test.Name=banana",
			golden: "output/install-with-multiple-values.txt",
		},
		// Install, values from yaml
		{
			name:   "install with values file",
			cmd:    "install virgil testdata/testcharts/alpine -f testdata/testcharts/alpine/extra_values.yaml",
			golden: "output/install-with-values-file.txt",
		},
		// Install, no hooks
		{
			name:   "install without hooks",
			cmd:    "install aeneas testdata/testcharts/alpine --no-hooks --set test.Name=hello",
			golden: "output/install-no-hooks.txt",
		},
		// Install, values from multiple yaml
		{
			name:   "install with values",
			cmd:    "install virgil testdata/testcharts/alpine -f testdata/testcharts/alpine/extra_values.yaml -f testdata/testcharts/alpine/more_values.yaml",
			golden: "output/install-with-multiple-values-files.txt",
		},
		// Install, no charts
		{
			name:      "install with no chart specified",
			cmd:       "install",
			golden:    "output/install-no-args.txt",
			wantError: true,
		},
		// Install, re-use name
		{
			name:   "install and replace release",
			cmd:    "install aeneas testdata/testcharts/empty --replace",
			golden: "output/install-and-replace.txt",
		},
		// Install, with timeout
		{
			name:   "install with a timeout",
			cmd:    "install foobar testdata/testcharts/empty --timeout 120",
			golden: "output/install-with-timeout.txt",
		},
		// Install, with wait
		{
			name:   "install with a wait",
			cmd:    "install apollo testdata/testcharts/empty --wait",
			golden: "output/install-with-wait.txt",
		},
		// Install, using the name-template
		{
			name:   "install with name-template",
			cmd:    "install testdata/testcharts/empty --name-template '{{upper \"foobar\"}}'",
			golden: "output/install-name-template.txt",
		},
		// Install, perform chart verification along the way.
		{
			name:      "install with verification, missing provenance",
			cmd:       "install bogus testdata/testcharts/compressedchart-0.1.0.tgz --verify --keyring testdata/helm-test-key.pub",
			wantError: true,
		},
		{
			name:      "install with verification, directory instead of file",
			cmd:       "install bogus testdata/testcharts/signtest --verify --keyring testdata/helm-test-key.pub",
			wantError: true,
		},
		{
			name: "install with verification, valid",
			cmd:  "install signtest testdata/testcharts/signtest-0.1.0.tgz --verify --keyring testdata/helm-test-key.pub",
		},
		// Install, chart with missing dependencies in /charts
		{
			name:      "install chart with missing dependencies",
			cmd:       "install nodeps testdata/testcharts/chart-missing-deps",
			wantError: true,
		},
		// Install, chart with bad dependencies in Chart.yaml in /charts
		{
			name:      "install chart with bad  dependencies in Chart.yaml",
			cmd:       "install badreq testdata/testcharts/chart-bad-requirements",
			wantError: true,
		},
		// Install, chart with library chart dependency
		{
			name: "install chart with library chart dependency",
			cmd:  "install withlibchartp testdata/testcharts/chart-with-lib-dep",
		},
		// Install, library chart
		{
			name:      "install library chart",
			cmd:       "install libchart testdata/testcharts/lib-chart",
			wantError: true,
			golden:    "output/template-lib-chart.txt",
		},
		// Install, chart with bad type
		{
			name:      "install chart with bad type",
			cmd:       "install badtype testdata/testcharts/chart-bad-type",
			wantError: true,
			golden:    "output/install-chart-bad-type.txt",
		},
	}

	runTestActionCmd(t, tests)
}

type nameTemplateTestCase struct {
	tpl              string
	expected         string
	expectedErrorStr string
}

func TestNameTemplate(t *testing.T) {
	testCases := []nameTemplateTestCase{
		// Just a straight up nop please
		{
			tpl:              "foobar",
			expected:         "foobar",
			expectedErrorStr: "",
		},
		// Random numbers at the end for fun & profit
		{
			tpl:              "foobar-{{randNumeric 6}}",
			expected:         "foobar-[0-9]{6}$",
			expectedErrorStr: "",
		},
		// Random numbers in the middle for fun & profit
		{
			tpl:              "foobar-{{randNumeric 4}}-baz",
			expected:         "foobar-[0-9]{4}-baz$",
			expectedErrorStr: "",
		},
		// No such function
		{
			tpl:              "foobar-{{randInt}}",
			expected:         "",
			expectedErrorStr: "function \"randInt\" not defined",
		},
		// Invalid template
		{
			tpl:              "foobar-{{",
			expected:         "",
			expectedErrorStr: "unexpected unclosed action",
		},
	}

	for _, tc := range testCases {

		n, err := templateName(tc.tpl)
		if err != nil {
			if tc.expectedErrorStr == "" {
				t.Errorf("Was not expecting error, but got: %v", err)
				continue
			}
			re, compErr := regexp.Compile(tc.expectedErrorStr)
			if compErr != nil {
				t.Errorf("Expected error string failed to compile: %v", compErr)
				continue
			}
			if !re.MatchString(err.Error()) {
				t.Errorf("Error didn't match for %s expected %s but got %v", tc.tpl, tc.expectedErrorStr, err)
				continue
			}
		}
		if err == nil && tc.expectedErrorStr != "" {
			t.Errorf("Was expecting error %s but didn't get an error back", tc.expectedErrorStr)
		}

		if tc.expected != "" {
			re, err := regexp.Compile(tc.expected)
			if err != nil {
				t.Errorf("Expected string failed to compile: %v", err)
				continue
			}
			if !re.MatchString(n) {
				t.Errorf("Returned name didn't match for %s expected %s but got %s", tc.tpl, tc.expected, n)
			}
		}
	}
}

func TestMergeValues(t *testing.T) {
	nestedMap := map[string]interface{}{
		"foo": "bar",
		"baz": map[string]string{
			"cool": "stuff",
		},
	}
	anotherNestedMap := map[string]interface{}{
		"foo": "bar",
		"baz": map[string]string{
			"cool":    "things",
			"awesome": "stuff",
		},
	}
	flatMap := map[string]interface{}{
		"foo": "bar",
		"baz": "stuff",
	}
	anotherFlatMap := map[string]interface{}{
		"testing": "fun",
	}

	testMap := mergeValues(flatMap, nestedMap)
	equal := reflect.DeepEqual(testMap, nestedMap)
	if !equal {
		t.Errorf("Expected a nested map to overwrite a flat value. Expected: %v, got %v", nestedMap, testMap)
	}

	testMap = mergeValues(nestedMap, flatMap)
	equal = reflect.DeepEqual(testMap, flatMap)
	if !equal {
		t.Errorf("Expected a flat value to overwrite a map. Expected: %v, got %v", flatMap, testMap)
	}

	testMap = mergeValues(nestedMap, anotherNestedMap)
	equal = reflect.DeepEqual(testMap, anotherNestedMap)
	if !equal {
		t.Errorf("Expected a nested map to overwrite another nested map. Expected: %v, got %v", anotherNestedMap, testMap)
	}

	testMap = mergeValues(anotherFlatMap, anotherNestedMap)
	expectedMap := map[string]interface{}{
		"testing": "fun",
		"foo":     "bar",
		"baz": map[string]string{
			"cool":    "things",
			"awesome": "stuff",
		},
	}
	equal = reflect.DeepEqual(testMap, expectedMap)
	if !equal {
		t.Errorf("Expected a map with different keys to merge properly with another map. Expected: %v, got %v", expectedMap, testMap)
	}
}
