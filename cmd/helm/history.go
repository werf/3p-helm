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

	"github.com/spf13/cobra"

	"helm.sh/helm/cmd/helm/require"
	"helm.sh/helm/pkg/action"
)

var historyHelp = `
History prints historical revisions for a given release.

A default maximum of 256 revisions will be returned. Setting '--max'
configures the maximum length of the revision list returned.

The historical release set is printed as a formatted table, e.g:

    $ helm history angry-bird --max=4
    REVISION    UPDATED                     STATUS          CHART             APP VERSION     DESCRIPTION
    1           Mon Oct 3 10:15:13 2016     superseded      alpine-0.1.0      1.0             Initial install
    2           Mon Oct 3 10:15:13 2016     superseded      alpine-0.1.0      1.0             Upgraded successfully
    3           Mon Oct 3 10:15:13 2016     superseded      alpine-0.1.0      1.0             Rolled back to 2
    4           Mon Oct 3 10:15:13 2016     deployed        alpine-0.1.0      1.0             Upgraded successfully
`

func newHistoryCmd(cfg *action.Configuration, out io.Writer) *cobra.Command {
	client := action.NewHistory(cfg)

	cmd := &cobra.Command{
		Use:     "history RELEASE_NAME",
		Long:    historyHelp,
		Short:   "fetch release history",
		Aliases: []string{"hist"},
		Args:    require.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			history, err := client.Run(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(out, history)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&client.OutputFormat, "output", "o", action.Table.String(), "prints the output in the specified format (json|table|yaml)")
	f.IntVar(&client.Max, "max", 256, "maximum number of revision to include in history")

	return cmd
}
