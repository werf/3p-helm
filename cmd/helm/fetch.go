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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"k8s.io/helm/cmd/helm/require"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/repo"
)

const fetchDesc = `
Retrieve a package from a package repository, and download it locally.

This is useful for fetching packages to inspect, modify, or repackage. It can
also be used to perform cryptographic verification of a chart without installing
the chart.

There are options for unpacking the chart after download. This will create a
directory for the chart and uncompress into that directory.

If the --verify flag is specified, the requested chart MUST have a provenance
file, and MUST pass the verification process. Failure in any part of this will
result in an error, and the chart will not be saved locally.
`

type fetchOptions struct {
	destdir     string // --destination
	devel       bool   // --devel
	untar       bool   // --untar
	untardir    string // --untardir
	verifyLater bool   // --prov

	chartRef string

	chartPathOptions
}

func newFetchCmd(out io.Writer) *cobra.Command {
	o := &fetchOptions{}

	cmd := &cobra.Command{
		Use:   "fetch [chart URL | repo/chartname] [...]",
		Short: "download a chart from a repository and (optionally) unpack it in local directory",
		Long:  fetchDesc,
		Args:  require.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if o.version == "" && o.devel {
				debug("setting version to >0.0.0-0")
				o.version = ">0.0.0-0"
			}

			for i := 0; i < len(args); i++ {
				o.chartRef = args[i]
				if err := o.run(out); err != nil {
					return err
				}
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVar(&o.devel, "devel", false, "use development versions, too. Equivalent to version '>0.0.0-0'. If --version is set, this is ignored.")
	f.BoolVar(&o.untar, "untar", false, "if set to true, will untar the chart after downloading it")
	f.BoolVar(&o.verifyLater, "prov", false, "fetch the provenance file, but don't perform verification")
	f.StringVar(&o.untardir, "untardir", ".", "if untar is specified, this flag specifies the name of the directory into which the chart is expanded")
	f.StringVarP(&o.destdir, "destination", "d", ".", "location to write the chart. If this and tardir are specified, tardir is appended to this")

	o.chartPathOptions.addFlags(f)

	return cmd
}

func (o *fetchOptions) run(out io.Writer) error {
	c := downloader.ChartDownloader{
		HelmHome: settings.Home,
		Out:      out,
		Keyring:  o.keyring,
		Verify:   downloader.VerifyNever,
		Getters:  getter.All(settings),
		Username: o.username,
		Password: o.password,
	}

	if o.verify {
		c.Verify = downloader.VerifyAlways
	} else if o.verifyLater {
		c.Verify = downloader.VerifyLater
	}

	// If untar is set, we fetch to a tempdir, then untar and copy after
	// verification.
	dest := o.destdir
	if o.untar {
		var err error
		dest, err = ioutil.TempDir("", "helm-")
		if err != nil {
			return errors.Wrap(err, "failed to untar")
		}
		defer os.RemoveAll(dest)
	}

	if o.repoURL != "" {
		chartURL, err := repo.FindChartInAuthRepoURL(o.repoURL, o.username, o.password, o.chartRef, o.version, o.certFile, o.keyFile, o.caFile, getter.All(settings))
		if err != nil {
			return err
		}
		o.chartRef = chartURL
	}

	saved, v, err := c.DownloadTo(o.chartRef, o.version, dest)
	if err != nil {
		return err
	}

	if o.verify {
		fmt.Fprintf(out, "Verification: %v\n", v)
	}

	// After verification, untar the chart into the requested directory.
	if o.untar {
		ud := o.untardir
		if !filepath.IsAbs(ud) {
			ud = filepath.Join(o.destdir, ud)
		}
		if fi, err := os.Stat(ud); err != nil {
			if err := os.MkdirAll(ud, 0755); err != nil {
				return errors.Wrap(err, "failed to untar (mkdir)")
			}

		} else if !fi.IsDir() {
			return errors.Errorf("failed to untar: %s is not a directory", ud)
		}

		return chartutil.ExpandFile(ud, saved)
	}
	return nil
}
