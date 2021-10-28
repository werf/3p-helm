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
	"os"
	"strings"

	"github.com/pkg/errors"

	"helm.sh/helm/v3/internal/experimental/registry"
)

// OCIPusher is the default OCI backend handler
type OCIPusherNonStrict struct {
	opts options
}

// Push performs a Push from repo.Pusher.
func (pusher *OCIPusherNonStrict) Push(chartRef, href string, options ...Option) error {
	for _, opt := range options {
		opt(&pusher.opts)
	}
	return pusher.push(chartRef, href)
}

func (pusher *OCIPusherNonStrict) push(chartRef, href string) error {
	stat, err := os.Stat(chartRef)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.Errorf("%s: no such file", chartRef)
		}
		return err
	}
	if stat.IsDir() {
		return errors.New("cannot push directory, must provide chart archive (.tgz)")
	}

	client := pusher.opts.registryClient

	chartBytes, err := ioutil.ReadFile(chartRef)
	if err != nil {
		return err
	}

	var pushOpts []registry.PushOption
	provRef := fmt.Sprintf("%s.prov", chartRef)
	if _, err := os.Stat(provRef); err == nil {
		provBytes, err := ioutil.ReadFile(provRef)
		if err != nil {
			return err
		}
		pushOpts = append(pushOpts, registry.PushOptProvData(provBytes))
	}

	pushOpts = append(pushOpts, registry.PushOptStrictMode(false))

	ref := strings.TrimPrefix(href, fmt.Sprintf("%s://", registry.OCIScheme))

	_, err = client.Push(chartBytes, ref, pushOpts...)
	return err
}

// NewOCIPusher constructs a valid OCI client as a Pusher
func NewOCIPusherNonStrict(ops ...Option) (Pusher, error) {
	registryClient, err := registry.NewClient()
	if err != nil {
		return nil, err
	}

	client := OCIPusherNonStrict{
		opts: options{
			registryClient: registryClient,
		},
	}

	for _, opt := range ops {
		opt(&client.opts)
	}

	return &client, nil
}
