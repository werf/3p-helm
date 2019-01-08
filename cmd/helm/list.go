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
	"strings"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"k8s.io/helm/cmd/helm/require"
	"k8s.io/helm/pkg/action"
	"k8s.io/helm/pkg/hapi/release"
)

var listHelp = `
This command lists all of the releases.

By default, it lists only releases that are deployed or failed. Flags like
'--uninstalled' and '--all' will alter this behavior. Such flags can be combined:
'--uninstalled --failed'.

By default, items are sorted alphabetically. Use the '-d' flag to sort by
release date.

If an argument is provided, it will be treated as a filter. Filters are
regular expressions (Perl compatible) that are applied to the list of releases.
Only items that match the filter will be returned.

	$ helm list 'ara[a-z]+'
	NAME            	UPDATED                 	CHART
	maudlin-arachnid	Mon May  9 16:07:08 2016	alpine-0.1.0

If no results are found, 'helm list' will exit 0, but with no output (or in
the case of no '-q' flag, only headers).

By default, up to 256 items may be returned. To limit this, use the '--max' flag.
Setting '--max' to 0 will not return all results. Rather, it will return the
server's default, which may be much higher than 256. Pairing the '--max'
flag with the '--offset' flag allows you to page through results.
`

type listOptions struct {
	// flags
	all           bool // --all
	allNamespaces bool // --all-namespaces
	byDate        bool // --date
	colWidth      uint // --col-width
	uninstalled   bool // --uninstalled
	uninstalling  bool // --uninstalling
	deployed      bool // --deployed
	failed        bool // --failed
	limit         int  // --max
	offset        int  // --offset
	pending       bool // --pending
	short         bool // --short
	sortDesc      bool // --reverse
	superseded    bool // --superseded

	filter string
}

func newListCmd(actionConfig *action.Configuration, out io.Writer) *cobra.Command {
	o := &listOptions{}

	cmd := &cobra.Command{
		Use:     "list [FILTER]",
		Short:   "list releases",
		Long:    listHelp,
		Aliases: []string{"ls"},
		Args:    require.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				o.filter = strings.Join(args, " ")
			}

			if o.allNamespaces {
				actionConfig = newActionConfig(true)
			}

			lister := action.NewList(actionConfig)
			lister.All = o.limit == -1
			lister.AllNamespaces = o.allNamespaces
			lister.Limit = o.limit
			lister.Offset = o.offset
			lister.Filter = o.filter

			// Set StateMask
			lister.StateMask = o.setStateMask()

			// Set sorter
			lister.Sort = action.ByNameAsc
			if o.sortDesc {
				lister.Sort = action.ByNameDesc
			}
			if o.byDate {
				lister.Sort = action.ByDate
			}

			results, err := lister.Run()

			if o.short {
				for _, res := range results {
					fmt.Fprintln(out, res.Name)
				}
				return err
			}

			fmt.Fprintln(out, formatList(results, 90))
			return err
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&o.short, "short", "q", false, "output short (quiet) listing format")
	f.BoolVarP(&o.byDate, "date", "d", false, "sort by release date")
	f.BoolVarP(&o.sortDesc, "reverse", "r", false, "reverse the sort order")
	f.IntVarP(&o.limit, "max", "m", 256, "maximum number of releases to fetch")
	f.IntVarP(&o.offset, "offset", "o", 0, "next release name in the list, used to offset from start value")
	f.BoolVarP(&o.all, "all", "a", false, "show all releases, not just the ones marked deployed")
	f.BoolVar(&o.uninstalled, "uninstalled", false, "show uninstalled releases")
	f.BoolVar(&o.superseded, "superseded", false, "show superseded releases")
	f.BoolVar(&o.uninstalling, "uninstalling", false, "show releases that are currently being uninstalled")
	f.BoolVar(&o.deployed, "deployed", false, "show deployed releases. If no other is specified, this will be automatically enabled")
	f.BoolVar(&o.failed, "failed", false, "show failed releases")
	f.BoolVar(&o.pending, "pending", false, "show pending releases")
	f.UintVar(&o.colWidth, "col-width", 60, "specifies the max column width of output")
	f.BoolVar(&o.allNamespaces, "all-namespaces", false, "list releases across all namespaces")

	return cmd
}

// setStateMask calculates the state mask based on parameters.
func (o *listOptions) setStateMask() action.ListStates {
	if o.all {
		return action.ListAll
	}

	state := action.ListStates(0)
	if o.deployed {
		state |= action.ListDeployed
	}
	if o.uninstalled {
		state |= action.ListUninstalled
	}
	if o.uninstalling {
		state |= action.ListUninstalling
	}
	if o.pending {
		state |= action.ListPendingInstall | action.ListPendingRollback | action.ListPendingUpgrade
	}
	if o.failed {
		state |= action.ListFailed
	}

	// Apply a default
	if state == 0 {
		return action.ListDeployed | action.ListFailed
	}

	return state
}

func formatList(rels []*release.Release, colWidth uint) string {
	table := uitable.New()

	table.MaxColWidth = colWidth
	table.AddRow("NAME", "REVISION", "UPDATED", "STATUS", "CHART", "NAMESPACE")
	for _, r := range rels {
		md := r.Chart.Metadata
		c := fmt.Sprintf("%s-%s", md.Name, md.Version)
		t := "-"
		if tspb := r.Info.LastDeployed; !tspb.IsZero() {
			t = tspb.String()
		}
		s := r.Info.Status.String()
		v := r.Version
		n := r.Namespace
		table.AddRow(r.Name, v, t, s, c, n)
	}
	return table.String()
}
