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

package releaseutil // import "k8s.io/helm/pkg/releaseutil"

import (
	"sort"

	rspb "k8s.io/helm/pkg/hapi/release"
)

type list []*rspb.Release

func (s list) Len() int      { return len(s) }
func (s list) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type ByName struct{ list }

func (s ByName) Less(i, j int) bool { return s.list[i].Name < s.list[j].Name }

type ByDate struct{ list }

func (s ByDate) Less(i, j int) bool {
	ti := s.list[i].Info.LastDeployed.Second()
	tj := s.list[j].Info.LastDeployed.Second()
	return ti < tj
}

type ByRevision struct{ list }

func (s ByRevision) Less(i, j int) bool {
	return s.list[i].Version < s.list[j].Version
}

// Reverse reverses the list of releases sorted by the sort func.
func Reverse(list []*rspb.Release, sortFn func([]*rspb.Release)) {
	sortFn(list)
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
}

// SortByName returns the list of releases sorted
// in lexicographical order.
func SortByName(list []*rspb.Release) {
	sort.Sort(ByName{list})
}

// SortByDate returns the list of releases sorted by a
// release's last deployed time (in seconds).
func SortByDate(list []*rspb.Release) {
	sort.Sort(ByDate{list})
}

// SortByRevision returns the list of releases sorted by a
// release's revision number (release.Version).
func SortByRevision(list []*rspb.Release) {
	sort.Sort(ByRevision{list})
}
