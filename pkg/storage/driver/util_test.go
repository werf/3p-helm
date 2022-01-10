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

package driver

import (
	"reflect"
	"testing"
)

func TestIsSystemLabel(t *testing.T) {
	tests := map[string]bool{
		"name":  true,
		"owner": true,
		"test":  false,
		"NaMe":  false,
	}
	for label, result := range tests {
		if output := isSystemLabel(label); output != result {
			t.Errorf("Output %t not equal to expected %t", output, result)
		}
	}
}

func TestFilterSystemLabels(t *testing.T) {
	var tests = [][2]map[string]string{
		{nil, nil},
		{map[string]string{}, nil},
		{map[string]string{
			"name":       "name",
			"owner":      "owner",
			"status":     "status",
			"version":    "version",
			"createdAt":  "createdAt",
			"modifiedAt": "modifiedAt",
		}, nil},
		{map[string]string{
			"StaTus": "status",
			"name":   "name",
			"owner":  "owner",
			"key":    "value",
		}, map[string]string{
			"StaTus": "status",
			"key":    "value",
		}},
		{map[string]string{
			"key1": "value1",
			"key2": "value2",
		}, map[string]string{
			"key1": "value1",
			"key2": "value2",
		}},
	}
	for _, test := range tests {
		if output := filterSystemLabels(test[0]); !reflect.DeepEqual(test[1], output) {
			t.Errorf("Expected {%v}, got {%v}", test[1], output)
		}
	}
}
