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
	"errors"
	"io"

	"github.com/spf13/cobra"

	"k8s.io/helm/pkg/helm"
)

var getHelp = `
This command shows the details of a named release.

It can be used to get extended information about the release, including:

  - The values used to generate the release
  - The chart used to generate the release
  - The generated manifest file

By default, this prints a human readable collection of information about the
chart, the supplied values, and the generated manifest file.
`

var errReleaseRequired = errors.New("release name is required")

type getCmd struct {
	release string
	version int

	client helm.Interface
}

func newGetCmd(client helm.Interface, out io.Writer) *cobra.Command {
	get := &getCmd{client: client}

	cmd := &cobra.Command{
		Use:   "get [flags] RELEASE_NAME",
		Short: "download a named release",
		Long:  getHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errReleaseRequired
			}
			get.release = args[0]
			get.client = ensureHelmClient(get.client, false)
			return get.run(out)
		},
	}

	cmd.Flags().IntVar(&get.version, "revision", 0, "get the named release with revision")

	cmd.AddCommand(newGetValuesCmd(client, out))
	cmd.AddCommand(newGetManifestCmd(client, out))
	cmd.AddCommand(newGetHooksCmd(client, out))

	return cmd
}

// getCmd is the command that implements 'helm get'
func (g *getCmd) run(out io.Writer) error {
	res, err := g.client.ReleaseContent(g.release, g.version)
	if err != nil {
		return err
	}
	return printRelease(out, res)
}
