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

package kube

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/testapi"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/validation"
	"k8s.io/kubernetes/pkg/client/restclient/fake"
	cmdtesting "k8s.io/kubernetes/pkg/kubectl/cmd/testing"
	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/runtime"
)

func objBody(codec runtime.Codec, obj runtime.Object) io.ReadCloser {
	return ioutil.NopCloser(bytes.NewReader([]byte(runtime.EncodeOrDie(codec, obj))))
}

func newPod(name string) api.Pod {
	return api.Pod{
		ObjectMeta: api.ObjectMeta{Name: name},
		Spec: api.PodSpec{
			Containers: []api.Container{{
				Name:  "app:v4",
				Image: "abc/app:v4",
				Ports: []api.ContainerPort{{Name: "http", ContainerPort: 80}},
			}},
		},
	}
}

func newPodList(names ...string) api.PodList {
	var list api.PodList
	for _, name := range names {
		list.Items = append(list.Items, newPod(name))
	}
	return list
}

func notFoundBody() *unversioned.Status {
	return &unversioned.Status{
		Code:    http.StatusNotFound,
		Status:  unversioned.StatusFailure,
		Reason:  unversioned.StatusReasonNotFound,
		Message: " \"\" not found",
		Details: &unversioned.StatusDetails{},
	}
}

func newResponse(code int, obj runtime.Object) (*http.Response, error) {
	header := http.Header{}
	header.Set("Content-Type", runtime.ContentTypeJSON)
	body := ioutil.NopCloser(bytes.NewReader([]byte(runtime.EncodeOrDie(testapi.Default.Codec(), obj))))
	return &http.Response{StatusCode: code, Header: header, Body: body}, nil
}

func TestUpdate(t *testing.T) {
	listA := newPodList("starfish", "otter", "squid")
	listB := newPodList("starfish", "otter", "dolphin")
	listB.Items[0].Spec.Containers[0].Ports = []api.ContainerPort{{Name: "https", ContainerPort: 443}}

	actions := make(map[string]string)

	f, tf, codec, ns := cmdtesting.NewAPIFactory()
	tf.Client = &fake.RESTClient{
		NegotiatedSerializer: ns,
		Client: fake.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
			p, m := req.URL.Path, req.Method
			actions[p] = m
			switch {
			case p == "/namespaces/test/pods/starfish" && m == "GET":
				return newResponse(200, &listA.Items[0])
			case p == "/namespaces/test/pods/otter" && m == "GET":
				return newResponse(200, &listA.Items[1])
			case p == "/namespaces/test/pods/dolphin" && m == "GET":
				return newResponse(404, notFoundBody())
			case p == "/namespaces/test/pods/starfish" && m == "PATCH":
				data, err := ioutil.ReadAll(req.Body)
				if err != nil {
					t.Fatalf("could not dump request: %s", err)
				}
				req.Body.Close()
				expected := `{"spec":{"containers":[{"name":"app:v4","ports":[{"containerPort":443,"name":"https","protocol":"TCP"},{"$patch":"delete","containerPort":80}]}]}}`
				if string(data) != expected {
					t.Errorf("expected patch %s, got %s", expected, string(data))
				}
				return newResponse(200, &listB.Items[0])
			case p == "/namespaces/test/pods" && m == "POST":
				return newResponse(200, &listB.Items[1])
			case p == "/namespaces/test/pods/squid" && m == "DELETE":
				return newResponse(200, &listB.Items[1])
			default:
				t.Fatalf("unexpected request: %#v\n%#v", req.URL, req)
				return nil, nil
			}
		}),
	}

	c := &Client{Factory: f}
	if err := c.Update("test", objBody(codec, &listA), objBody(codec, &listB)); err != nil {
		t.Fatal(err)
	}

	expectedActions := map[string]string{
		"/namespaces/test/pods/dolphin":  "GET",
		"/namespaces/test/pods/otter":    "GET",
		"/namespaces/test/pods/starfish": "PATCH",
		"/namespaces/test/pods":          "POST",
		"/namespaces/test/pods/squid":    "DELETE",
	}

	for k, v := range expectedActions {
		if m, ok := actions[k]; !ok || m != v {
			t.Errorf("expected a %s request to %s", k, v)
		}
	}
}

func TestPerform(t *testing.T) {
	tests := []struct {
		name        string
		namespace   string
		reader      io.Reader
		count       int
		swaggerFile string
		err         bool
		errMessage  string
	}{
		{
			name:      "Valid input",
			namespace: "test",
			reader:    strings.NewReader(guestbookManifest),
			count:     6,
		}, {
			name:       "Empty manifests",
			namespace:  "test",
			reader:     strings.NewReader(""),
			err:        true,
			errMessage: "no objects visited",
		}, {
			name:        "Invalid schema",
			namespace:   "test",
			reader:      strings.NewReader(testInvalidServiceManifest),
			swaggerFile: "../../vendor/k8s.io/kubernetes/api/swagger-spec/" + testapi.Default.GroupVersion().Version + ".json",
			err:         true,
			errMessage:  `error validating "": error validating data: expected type int, for field spec.ports[0].port, got string`,
		},
	}

	for _, tt := range tests {
		results := []*resource.Info{}

		fn := func(info *resource.Info) error {
			results = append(results, info)

			if info.Namespace != tt.namespace {
				t.Errorf("%q. expected namespace to be '%s', got %s", tt.name, tt.namespace, info.Namespace)
			}
			return nil
		}

		f, tf, _, _ := cmdtesting.NewAPIFactory()
		c := &Client{Factory: f}
		if tt.swaggerFile != "" {
			data, err := ioutil.ReadFile(tt.swaggerFile)
			if err != nil {
				t.Fatalf("could not read swagger spec: %s", err)
			}
			validator, err := validation.NewSwaggerSchemaFromBytes(data, nil)
			if err != nil {
				t.Fatalf("could not load swagger spec: %s", err)
			}
			tf.Validator = validator
		}

		err := perform(c, tt.namespace, tt.reader, fn)
		if (err != nil) != tt.err {
			t.Errorf("%q. expected error: %v, got %v", tt.name, tt.err, err)
		}
		if err != nil && err.Error() != tt.errMessage {
			t.Errorf("%q. expected error message: %v, got %v", tt.name, tt.errMessage, err)
		}

		if len(results) != tt.count {
			t.Errorf("%q. expected %d result objects, got %d", tt.name, tt.count, len(results))
		}
	}
}

func TestReal(t *testing.T) {
	t.Skip("This is a live test, comment this line to run")
	c := New(nil)
	if err := c.Create("test", strings.NewReader(guestbookManifest)); err != nil {
		t.Fatal(err)
	}

	testSvcEndpointManifest := testServiceManifest + "\n---\n" + testEndpointManifest
	c = New(nil)
	if err := c.Create("test-delete", strings.NewReader(testSvcEndpointManifest)); err != nil {
		t.Fatal(err)
	}

	if err := c.Delete("test-delete", strings.NewReader(testEndpointManifest)); err != nil {
		t.Fatal(err)
	}

	// ensures that delete does not fail if a resource is not found
	if err := c.Delete("test-delete", strings.NewReader(testSvcEndpointManifest)); err != nil {
		t.Fatal(err)
	}
}

const testServiceManifest = `
kind: Service
apiVersion: v1
metadata:
  name: my-service
spec:
  selector:
    app: myapp
  ports:
    - port: 80
      protocol: TCP
      targetPort: 9376
`

const testInvalidServiceManifest = `
kind: Service
apiVersion: v1
spec:
  ports:
    - port: "80"
`

const testEndpointManifest = `
kind: Endpoints
apiVersion: v1
metadata:
  name: my-service
subsets:
  - addresses:
      - ip: "1.2.3.4"
    ports:
      - port: 9376
`

const guestbookManifest = `
apiVersion: v1
kind: Service
metadata:
  name: redis-master
  labels:
    app: redis
    tier: backend
    role: master
spec:
  ports:
  - port: 6379
    targetPort: 6379
  selector:
    app: redis
    tier: backend
    role: master
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: redis-master
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: redis
        role: master
        tier: backend
    spec:
      containers:
      - name: master
        image: gcr.io/google_containers/redis:e2e  # or just image: redis
        resources:
          requests:
            cpu: 100m
            memory: 100Mi
        ports:
        - containerPort: 6379
---
apiVersion: v1
kind: Service
metadata:
  name: redis-slave
  labels:
    app: redis
    tier: backend
    role: slave
spec:
  ports:
    # the port that this service should serve on
  - port: 6379
  selector:
    app: redis
    tier: backend
    role: slave
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: redis-slave
spec:
  replicas: 2
  template:
    metadata:
      labels:
        app: redis
        role: slave
        tier: backend
    spec:
      containers:
      - name: slave
        image: gcr.io/google_samples/gb-redisslave:v1
        resources:
          requests:
            cpu: 100m
            memory: 100Mi
        env:
        - name: GET_HOSTS_FROM
          value: dns
        ports:
        - containerPort: 6379
---
apiVersion: v1
kind: Service
metadata:
  name: frontend
  labels:
    app: guestbook
    tier: frontend
spec:
  ports:
  - port: 80
  selector:
    app: guestbook
    tier: frontend
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: frontend
spec:
  replicas: 3
  template:
    metadata:
      labels:
        app: guestbook
        tier: frontend
    spec:
      containers:
      - name: php-redis
        image: gcr.io/google-samples/gb-frontend:v4
        resources:
          requests:
            cpu: 100m
            memory: 100Mi
        env:
        - name: GET_HOSTS_FROM
          value: dns
        ports:
        - containerPort: 80
`
