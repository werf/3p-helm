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

package httpgetter

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/tlsutil"
	"k8s.io/helm/pkg/urlutil"
)

//HTTPGetter is the efault HTTP(/S) backend handler
type HTTPGetter struct {
	client *http.Client
}

//Get performs a Get from repo.Getter and returns the body.
func (g *HTTPGetter) Get(href string) (*bytes.Buffer, error) {
	buf := bytes.NewBuffer(nil)

	resp, err := g.client.Get(href)
	if err != nil {
		return buf, err
	}
	if resp.StatusCode != 200 {
		return buf, fmt.Errorf("Failed to fetch %s : %s", href, resp.Status)
	}

	_, err = io.Copy(buf, resp.Body)
	resp.Body.Close()
	return buf, err
}

//New constructs a valid http/https client as Getter
func New(URL, CertFile, KeyFile, CAFile string) (getter.Getter, error) {
	var client HTTPGetter
	if CertFile != "" && KeyFile != "" && CAFile != "" {
		tlsConf, err := tlsutil.NewClientTLS(CertFile, KeyFile, CAFile)
		if err != nil {
			return nil, fmt.Errorf("can't create TLS config for client: %s", err.Error())
		}
		tlsConf.BuildNameToCertificate()

		sni, err := urlutil.ExtractHostname(URL)
		if err != nil {
			return nil, err
		}
		tlsConf.ServerName = sni

		client.client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConf,
			},
		}
	} else {
		client.client = http.DefaultClient
	}
	return &client, nil
}
