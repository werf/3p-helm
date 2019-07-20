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

package action

import (
	"bytes"
	"sort"
	"time"

	"github.com/pkg/errors"

	"helm.sh/helm/pkg/release"
)

// execHook executes all of the hooks for the given hook event.
func (cfg *Configuration) execHook(hs []*release.Hook, hook release.HookEvent, timeout time.Duration) error {
	executingHooks := []*release.Hook{}

	for _, h := range hs {
		for _, e := range h.Events {
			if e == hook {
				executingHooks = append(executingHooks, h)
			}
		}
	}

	sort.Sort(hookByWeight(executingHooks))

	for _, h := range executingHooks {
		if err := deleteHookByPolicy(cfg, h, release.HookBeforeHookCreation); err != nil {
			return err
		}

		b := bytes.NewBufferString(h.Manifest)
		if err := cfg.KubeClient.Create(b); err != nil {
			return errors.Wrapf(err, "warning: Hook %s %s failed", hook, h.Path)
		}
		b.Reset()
		b.WriteString(h.Manifest)

		if err := cfg.KubeClient.WatchUntilReady(b, timeout); err != nil {
			// If a hook is failed, checkout the annotation of the hook to determine whether the hook should be deleted
			// under failed condition. If so, then clear the corresponding resource object in the hook
			if err := deleteHookByPolicy(cfg, h, release.HookFailed); err != nil {
				return err
			}
			return err
		}
	}

	// If all hooks are succeeded, checkout the annotation of each hook to determine whether the hook should be deleted
	// under succeeded condition. If so, then clear the corresponding resource object in each hook
	for _, h := range executingHooks {
		if err := deleteHookByPolicy(cfg, h, release.HookSucceeded); err != nil {
			return err
		}
		h.LastRun = time.Now()
	}

	return nil
}

// hookByWeight is a sorter for hooks
type hookByWeight []*release.Hook

func (x hookByWeight) Len() int      { return len(x) }
func (x hookByWeight) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x hookByWeight) Less(i, j int) bool {
	if x[i].Weight == x[j].Weight {
		return x[i].Name < x[j].Name
	}
	return x[i].Weight < x[j].Weight
}

// deleteHookByPolicy deletes a hook if the hook policy instructs it to
func deleteHookByPolicy(cfg *Configuration, h *release.Hook, policy release.HookDeletePolicy) error {
	if hookHasDeletePolicy(h, policy) {
		b := bytes.NewBufferString(h.Manifest)
		return cfg.KubeClient.Delete(b)
	}
	return nil
}

// hookHasDeletePolicy determines whether the defined hook deletion policy matches the hook deletion polices
// supported by helm. If so, mark the hook as one should be deleted.
func hookHasDeletePolicy(h *release.Hook, policy release.HookDeletePolicy) bool {
	for _, v := range h.DeletePolicies {
		if policy == v {
			return true
		}
	}
	return false
}
