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

package registry // import "helm.sh/helm/v3/pkg/registry"

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"helm.sh/helm/v3/pkg/chart"
	helmtime "helm.sh/helm/v3/pkg/time"
)

func TestGenerateOCIChartAnnotations(t *testing.T) {

	tests := []struct {
		name   string
		chart  *chart.Metadata
		expect map[string]string
	}{
		{
			"Baseline chart",
			&chart.Metadata{
				Name:    "oci",
				Version: "0.0.1",
			},
			map[string]string{
				"org.opencontainers.image.title":   "oci",
				"org.opencontainers.image.version": "0.0.1",
			},
		},
		{
			"Simple chart values",
			&chart.Metadata{
				Name:        "oci",
				Version:     "0.0.1",
				Description: "OCI Helm Chart",
				Home:        "https://helm.sh",
			},
			map[string]string{
				"org.opencontainers.image.title":       "oci",
				"org.opencontainers.image.version":     "0.0.1",
				"org.opencontainers.image.description": "OCI Helm Chart",
				"org.opencontainers.image.url":         "https://helm.sh",
			},
		},
		{
			"Maintainer without email",
			&chart.Metadata{
				Name:        "oci",
				Version:     "0.0.1",
				Description: "OCI Helm Chart",
				Home:        "https://helm.sh",
				Maintainers: []*chart.Maintainer{
					{
						Name: "John Snow",
					},
				},
			},
			map[string]string{
				"org.opencontainers.image.title":       "oci",
				"org.opencontainers.image.version":     "0.0.1",
				"org.opencontainers.image.description": "OCI Helm Chart",
				"org.opencontainers.image.url":         "https://helm.sh",
				"org.opencontainers.image.authors":     "John Snow",
			},
		},
		{
			"Maintainer with email",
			&chart.Metadata{
				Name:        "oci",
				Version:     "0.0.1",
				Description: "OCI Helm Chart",
				Home:        "https://helm.sh",
				Maintainers: []*chart.Maintainer{
					{Name: "John Snow", Email: "john@winterfell.com"},
				},
			},
			map[string]string{
				"org.opencontainers.image.title":       "oci",
				"org.opencontainers.image.version":     "0.0.1",
				"org.opencontainers.image.description": "OCI Helm Chart",
				"org.opencontainers.image.url":         "https://helm.sh",
				"org.opencontainers.image.authors":     "John Snow (john@winterfell.com)",
			},
		},
		{
			"Multiple Maintainers",
			&chart.Metadata{
				Name:        "oci",
				Version:     "0.0.1",
				Description: "OCI Helm Chart",
				Home:        "https://helm.sh",
				Maintainers: []*chart.Maintainer{
					{Name: "John Snow", Email: "john@winterfell.com"},
					{Name: "Jane Snow"},
				},
			},
			map[string]string{
				"org.opencontainers.image.title":       "oci",
				"org.opencontainers.image.version":     "0.0.1",
				"org.opencontainers.image.description": "OCI Helm Chart",
				"org.opencontainers.image.url":         "https://helm.sh",
				"org.opencontainers.image.authors":     "John Snow (john@winterfell.com), Jane Snow",
			},
		},
		{
			"Chart with Sources",
			&chart.Metadata{
				Name:        "oci",
				Version:     "0.0.1",
				Description: "OCI Helm Chart",
				Sources: []string{
					"https://github.com/helm/helm",
				},
			},
			map[string]string{
				"org.opencontainers.image.title":       "oci",
				"org.opencontainers.image.version":     "0.0.1",
				"org.opencontainers.image.description": "OCI Helm Chart",
				"org.opencontainers.image.source":      "https://github.com/helm/helm",
			},
		},
	}

	for _, tt := range tests {

		result := generateChartOCIAnnotations(tt.chart, true)

		if !reflect.DeepEqual(tt.expect, result) {
			t.Errorf("%s: expected map %v, got %v", tt.name, tt.expect, result)
		}

	}
}

func TestGenerateOCIAnnotations(t *testing.T) {

	tests := []struct {
		name   string
		chart  *chart.Metadata
		expect map[string]string
	}{
		{
			"Baseline chart",
			&chart.Metadata{
				Name:    "oci",
				Version: "0.0.1",
			},
			map[string]string{
				"org.opencontainers.image.title":   "oci",
				"org.opencontainers.image.version": "0.0.1",
			},
		},
		{
			"Simple chart values with custom Annotations",
			&chart.Metadata{
				Name:        "oci",
				Version:     "0.0.1",
				Description: "OCI Helm Chart",
				Annotations: map[string]string{
					"extrakey":   "extravlue",
					"anotherkey": "anothervalue",
				},
			},
			map[string]string{
				"org.opencontainers.image.title":       "oci",
				"org.opencontainers.image.version":     "0.0.1",
				"org.opencontainers.image.description": "OCI Helm Chart",
				"extrakey":                             "extravlue",
				"anotherkey":                           "anothervalue",
			},
		},
		{
			"Verify Chart Name and Version cannot be overridden from annotations",
			&chart.Metadata{
				Name:        "oci",
				Version:     "0.0.1",
				Description: "OCI Helm Chart",
				Annotations: map[string]string{
					"org.opencontainers.image.title":   "badchartname",
					"org.opencontainers.image.version": "1.0.0",
					"extrakey":                         "extravlue",
				},
			},
			map[string]string{
				"org.opencontainers.image.title":       "oci",
				"org.opencontainers.image.version":     "0.0.1",
				"org.opencontainers.image.description": "OCI Helm Chart",
				"extrakey":                             "extravlue",
			},
		},
	}

	for _, tt := range tests {

		result := generateOCIAnnotations(tt.chart, true)

		if !reflect.DeepEqual(tt.expect, result) {
			t.Errorf("%s: expected map %v, got %v", tt.name, tt.expect, result)
		}

	}
}

func TestGenerateOCICreatedAnnotations(t *testing.T) {
	chart := &chart.Metadata{
		Name:    "oci",
		Version: "0.0.1",
	}

	result := generateOCIAnnotations(chart, false)

	// Check that created annotation exists
	if _, ok := result[ocispec.AnnotationCreated]; !ok {
		t.Errorf("%s annotation not created", ocispec.AnnotationCreated)
	}

	// Verify value of created artifact in RFC3339 format
	if _, err := helmtime.Parse(time.RFC3339, result[ocispec.AnnotationCreated]); err != nil {
		t.Errorf("%s annotation with value '%s' not in RFC3339 format", ocispec.AnnotationCreated, result[ocispec.AnnotationCreated])
	}

}

func Test_basicAuth(t *testing.T) {
	type args struct {
		username string
		password string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Basic Auth",
			args: args{
				username: "admin",
				password: "passw0rd",
			},
			want: "YWRtaW46cGFzc3cwcmQ=",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := basicAuth(tt.args.username, tt.args.password); got != tt.want {
				t.Errorf("basicAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_authHeader(t *testing.T) {
	tests := []struct {
		name           string
		username       string
		password       string
		expectedHeader http.Header
	}{
		{
			name:     "basic login header with username and password",
			username: "admin",
			password: "passw0rd",
			expectedHeader: func() http.Header {
				header := http.Header{}
				header.Set("Authorization", "Basic YWRtaW46cGFzc3cwcmQ=")
				return header
			}(),
		},
		{
			name:     "bearer login header with no username and password",
			username: "",
			password: "hunter2",
			expectedHeader: func() http.Header {
				header := http.Header{}
				header.Set("Authorization", "Bearer hunter2")
				return header
			}(),
		},
		{
			name:           "no change in header with neither username nor password",
			username:       "",
			password:       "",
			expectedHeader: http.Header{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &http.Header{}
			authHeader(tt.username, tt.password, got)
			if !reflect.DeepEqual(*got, tt.expectedHeader) {
				t.Errorf("authHeader got %#v wanted %#v", *got, tt.expectedHeader)
			}
		})
	}
}
