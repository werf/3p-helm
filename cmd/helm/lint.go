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

package helm_v3

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/werf/3p-helm-legacy/pkg/action"
	"github.com/werf/3p-helm-legacy/pkg/chartutil"
	"github.com/werf/3p-helm-legacy/pkg/cli/values"
	"github.com/werf/3p-helm-legacy/pkg/getter"
	"github.com/werf/3p-helm-legacy/pkg/lint/support"
)

var longLintHelp = `
This command takes a path to a chart and runs a series of tests to verify that
the chart is well-formed.

If the linter encounters things that will cause the chart to fail installation,
it will emit [ERROR] messages. If it encounters issues that break with convention
or recommendation, it will emit [WARNING] messages.
`

func newLintCmd(out io.Writer) *cobra.Command {
	client := action.NewLint()
	valueOpts := &values.Options{}
	var kubeVersion string

	cmd := &cobra.Command{
		Use:   "lint PATH",
		Short: "examine a chart for possible issues",
		Long:  longLintHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			paths := []string{"."}
			if len(args) > 0 {
				paths = args
			}

			if kubeVersion != "" {
				parsedKubeVersion, err := chartutil.ParseKubeVersion(kubeVersion)
				if err != nil {
					return fmt.Errorf("invalid kube version '%s': %s", kubeVersion, err)
				}
				client.KubeVersion = parsedKubeVersion
			}

			if client.WithSubcharts {
				for _, p := range paths {
					filepath.Walk(filepath.Join(p, "charts"), func(path string, info os.FileInfo, err error) error {
						if info != nil {
							if info.Name() == "Chart.yaml" {
								paths = append(paths, filepath.Dir(path))
							} else if strings.HasSuffix(path, ".tgz") || strings.HasSuffix(path, ".tar.gz") {
								paths = append(paths, path)
							}
						}
						return nil
					})
				}
			}

			client.Namespace = settings.Namespace()
			vals, err := valueOpts.MergeValues(getter.All(settings), nil)
			if err != nil {
				return err
			}

			var message strings.Builder
			failed := 0
			errorsOrWarnings := 0

			for _, path := range paths {
				result := client.Run([]string{path}, vals)

				// If there is no errors/warnings and quiet flag is set
				// go to the next chart
				hasWarningsOrErrors := action.HasWarningsOrErrors(result)
				if hasWarningsOrErrors {
					errorsOrWarnings++
				}
				if client.Quiet && !hasWarningsOrErrors {
					continue
				}

				fmt.Fprintf(&message, "==> Linting %s\n", path)

				// All the Errors that are generated by a chart
				// that failed a lint will be included in the
				// results.Messages so we only need to print
				// the Errors if there are no Messages.
				if len(result.Messages) == 0 {
					for _, err := range result.Errors {
						fmt.Fprintf(&message, "Error %s\n", err)
					}
				}

				for _, msg := range result.Messages {
					if !client.Quiet || msg.Severity > support.InfoSev {
						fmt.Fprintf(&message, "%s\n", msg)
					}
				}

				if len(result.Errors) != 0 {
					failed++
				}

				// Adding extra new line here to break up the
				// results, stops this from being a big wall of
				// text and makes it easier to follow.
				fmt.Fprint(&message, "\n")
			}

			fmt.Fprint(out, message.String())

			summary := fmt.Sprintf("%d chart(s) linted, %d chart(s) failed", len(paths), failed)
			if failed > 0 {
				return errors.New(summary)
			}
			if !client.Quiet || errorsOrWarnings > 0 {
				fmt.Fprintln(out, summary)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.BoolVar(&client.Strict, "strict", false, "fail on lint warnings")
	f.BoolVar(&client.WithSubcharts, "with-subcharts", false, "lint dependent charts")
	f.BoolVar(&client.Quiet, "quiet", false, "print only warnings and errors")
	f.StringVar(&kubeVersion, "kube-version", "", "Kubernetes version used for capabilities and deprecation checks")
	addValueOptionsFlags(f, valueOpts)

	return cmd
}
