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

package pusher

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"helm.sh/helm/v3/internal/experimental/registry"
	"helm.sh/helm/v3/pkg/chart/loader"
)

// OCIPusher is the default OCI backend handler
type OCIPusher struct {
	opts options
}

// Push performs a Push from repo.Pusher.
func (pusher *OCIPusher) Push(chartRef, href string, options ...Option) error {
	for _, opt := range options {
		opt(&pusher.opts)
	}
	return pusher.push(chartRef, href)
}

func (pusher *OCIPusher) push(chartRef, href string) error {
	meta, err := loader.Load(chartRef)
	if err != nil {
		return err
	}

	client := pusher.opts.registryClient

	chartBytes, err := ioutil.ReadFile(chartRef)
	if err != nil {
		return err
	}

	var pushOpts []registry.PushOption
	var provBytes []byte
	if pusher.opts.withProv {
		provRef := fmt.Sprintf("%s.prov", chartRef)
		provBytes, err = ioutil.ReadFile(provRef)
		if err != nil {
			return err
		}
		pushOpts = append(pushOpts, registry.PushOptProvData(provBytes))
	}

	ref := fmt.Sprintf("%s:%s",
		path.Join(strings.TrimPrefix(href, "oci://"), meta.Metadata.Name),
		meta.Metadata.Version)

	_, err = client.Push(chartBytes, ref, pushOpts...)
	if err != nil {
		return err
	}

	return nil
}

// NewOCIPusher constructs a valid OCI client as a Pusher
func NewOCIPusher(ops ...Option) (Pusher, error) {
	registryClient, err := registry.NewClient()
	if err != nil {
		return nil, err
	}

	client := OCIPusher{
		opts: options{
			registryClient: registryClient,
		},
	}

	for _, opt := range ops {
		opt(&client.opts)
	}

	return &client, nil
}
