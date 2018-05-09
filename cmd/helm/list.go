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
	"fmt"
	"io"
	"strings"

	"github.com/gosuri/uitable"
	"github.com/spf13/cobra"

	"k8s.io/helm/pkg/hapi"
	"k8s.io/helm/pkg/hapi/release"
	"k8s.io/helm/pkg/helm"
)

var listHelp = `
This command lists all of the releases.

By default, it lists only releases that are deployed or failed. Flags like
'--deleted' and '--all' will alter this behavior. Such flags can be combined:
'--deleted --failed'.

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

type listCmd struct {
	filter        string
	short         bool
	limit         int
	offset        string
	byDate        bool
	sortDesc      bool
	out           io.Writer
	all           bool
	deleted       bool
	deleting      bool
	deployed      bool
	failed        bool
	superseded    bool
	pending       bool
	client        helm.Interface
	colWidth      uint
	allNamespaces bool
}

func newListCmd(client helm.Interface, out io.Writer) *cobra.Command {
	list := &listCmd{
		out:    out,
		client: client,
	}

	cmd := &cobra.Command{
		Use:     "list [flags] [FILTER]",
		Short:   "list releases",
		Long:    listHelp,
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				list.filter = strings.Join(args, " ")
			}
			list.client = ensureHelmClient(list.client, list.allNamespaces)
			return list.run()
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&list.short, "short", "q", false, "output short (quiet) listing format")
	f.BoolVarP(&list.byDate, "date", "d", false, "sort by release date")
	f.BoolVarP(&list.sortDesc, "reverse", "r", false, "reverse the sort order")
	f.IntVarP(&list.limit, "max", "m", 256, "maximum number of releases to fetch")
	f.StringVarP(&list.offset, "offset", "o", "", "next release name in the list, used to offset from start value")
	f.BoolVarP(&list.all, "all", "a", false, "show all releases, not just the ones marked deployed")
	f.BoolVar(&list.deleted, "deleted", false, "show deleted releases")
	f.BoolVar(&list.deleting, "deleting", false, "show releases that are currently being deleted")
	f.BoolVar(&list.deployed, "deployed", false, "show deployed releases. If no other is specified, this will be automatically enabled")
	f.BoolVar(&list.failed, "failed", false, "show failed releases")
	f.BoolVar(&list.pending, "pending", false, "show pending releases")
	f.UintVar(&list.colWidth, "col-width", 60, "specifies the max column width of output")
	f.BoolVar(&list.allNamespaces, "all-namespaces", false, "list releases across all namespaces")

	return cmd
}

func (l *listCmd) run() error {
	sortBy := hapi.SortByName
	if l.byDate {
		sortBy = hapi.SortByLastReleased
	}

	sortOrder := hapi.SortAsc
	if l.sortDesc {
		sortOrder = hapi.SortDesc
	}

	stats := l.statusCodes()

	res, err := l.client.ListReleases(
		helm.ReleaseListLimit(l.limit),
		helm.ReleaseListOffset(l.offset),
		helm.ReleaseListFilter(l.filter),
		helm.ReleaseListSort(sortBy),
		helm.ReleaseListOrder(sortOrder),
		helm.ReleaseListStatuses(stats),
	)

	if err != nil {
		return err
	}

	if len(res) == 0 {
		return nil
	}

	rels := filterList(res)

	if l.short {
		for _, r := range rels {
			fmt.Fprintln(l.out, r.Name)
		}
		return nil
	}
	fmt.Fprintln(l.out, formatList(rels, l.colWidth))
	return nil
}

// filterList returns a list scrubbed of old releases.
func filterList(rels []*release.Release) []*release.Release {
	idx := map[string]int{}

	for _, r := range rels {
		name, version := r.Name, r.Version
		if max, ok := idx[name]; ok {
			// check if we have a greater version already
			if max > version {
				continue
			}
		}
		idx[name] = version
	}

	uniq := make([]*release.Release, 0, len(idx))
	for _, r := range rels {
		if idx[r.Name] == r.Version {
			uniq = append(uniq, r)
		}
	}
	return uniq
}

// statusCodes gets the list of status codes that are to be included in the results.
func (l *listCmd) statusCodes() []release.ReleaseStatus {
	if l.all {
		return []release.ReleaseStatus{
			release.StatusUnknown,
			release.StatusDeployed,
			release.StatusDeleted,
			release.StatusDeleting,
			release.StatusFailed,
			release.StatusPendingInstall,
			release.StatusPendingUpgrade,
			release.StatusPendingRollback,
		}
	}
	status := []release.ReleaseStatus{}
	if l.deployed {
		status = append(status, release.StatusDeployed)
	}
	if l.deleted {
		status = append(status, release.StatusDeleted)
	}
	if l.deleting {
		status = append(status, release.StatusDeleting)
	}
	if l.failed {
		status = append(status, release.StatusFailed)
	}
	if l.superseded {
		status = append(status, release.StatusSuperseded)
	}
	if l.pending {
		status = append(status, release.StatusPendingInstall, release.StatusPendingUpgrade, release.StatusPendingRollback)
	}

	// Default case.
	if len(status) == 0 {
		status = append(status, release.StatusDeployed, release.StatusFailed)
	}
	return status
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
