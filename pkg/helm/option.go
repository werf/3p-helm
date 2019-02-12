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

package helm

import (
	"k8s.io/client-go/discovery"

	"k8s.io/helm/pkg/hapi"
	"k8s.io/helm/pkg/storage/driver"
	"k8s.io/helm/pkg/tiller/environment"
)

// Option allows specifying various settings configurable by
// the helm client user for overriding the defaults.
type Option func(*options)

// options specify optional settings used by the helm client.
type options struct {
	// if set dry-run helm client calls
	dryRun bool
	// if set, re-use an existing name
	reuseName bool
	// if set, performs pod restart during upgrade/rollback
	recreate bool
	// if set, force resource update through uninstall/recreate if needed
	force bool
	// if set, skip running hooks
	disableHooks bool
	// release list options are applied directly to the list releases request
	listReq hapi.ListReleasesRequest
	// release install options are applied directly to the install release request
	instReq hapi.InstallReleaseRequest
	// release update options are applied directly to the update release request
	updateReq hapi.UpdateReleaseRequest
	// release uninstall options are applied directly to the uninstall release request
	uninstallReq hapi.UninstallReleaseRequest
	// release get status options are applied directly to the get release status request
	statusReq hapi.GetReleaseStatusRequest
	// release get content options are applied directly to the get release content request
	contentReq hapi.GetReleaseContentRequest
	// release rollback options are applied directly to the rollback release request
	rollbackReq hapi.RollbackReleaseRequest
	// before intercepts client calls before sending
	before func(interface{}) error
	// release history options are applied directly to the get release history request
	histReq hapi.GetHistoryRequest
	// resetValues instructs Helm to reset values to their defaults.
	resetValues bool
	// reuseValues instructs Helm to reuse the values from the last release.
	reuseValues bool
	// release test options are applied directly to the test release history request
	testReq hapi.TestReleaseRequest

	driver     driver.Driver
	kubeClient environment.KubeClient
	discovery  discovery.DiscoveryInterface
}

func (opts *options) runBefore(msg interface{}) error {
	if opts.before != nil {
		return opts.before(msg)
	}
	return nil
}

// BeforeCall returns an option that allows intercepting a helm client rpc
// before being sent OTA to tiller. The intercepting function should return
// an error to indicate that the call should not proceed or nil otherwise.
func BeforeCall(fn func(interface{}) error) Option {
	return func(opts *options) {
		opts.before = fn
	}
}

// InstallOption allows specifying various settings
// configurable by the helm client user for overriding
// the defaults used when running the `helm install` command.
type InstallOption func(*options)

// ValueOverrides specifies a list of values to include when installing.
func ValueOverrides(raw map[string]interface{}) InstallOption {
	return func(opts *options) {
		opts.instReq.Values = raw
	}
}

// ReleaseName specifies the name of the release when installing.
func ReleaseName(name string) InstallOption {
	return func(opts *options) {
		opts.instReq.Name = name
	}
}

// InstallTimeout specifies the number of seconds before kubernetes calls timeout
func InstallTimeout(timeout int64) InstallOption {
	return func(opts *options) {
		opts.instReq.Timeout = timeout
	}
}

// UpgradeTimeout specifies the number of seconds before kubernetes calls timeout
func UpgradeTimeout(timeout int64) UpdateOption {
	return func(opts *options) {
		opts.updateReq.Timeout = timeout
	}
}

// UninstallTimeout specifies the number of seconds before kubernetes calls timeout
func UninstallTimeout(timeout int64) UninstallOption {
	return func(opts *options) {
		opts.uninstallReq.Timeout = timeout
	}
}

// ReleaseTestTimeout specifies the number of seconds before kubernetes calls timeout
func ReleaseTestTimeout(timeout int64) ReleaseTestOption {
	return func(opts *options) {
		opts.testReq.Timeout = timeout
	}
}

// ReleaseTestCleanup is a boolean value representing whether to cleanup test pods
func ReleaseTestCleanup(cleanup bool) ReleaseTestOption {
	return func(opts *options) {
		opts.testReq.Cleanup = cleanup
	}
}

// RollbackTimeout specifies the number of seconds before kubernetes calls timeout
func RollbackTimeout(timeout int64) RollbackOption {
	return func(opts *options) {
		opts.rollbackReq.Timeout = timeout
	}
}

// InstallWait specifies whether or not to wait for all resources to be ready
func InstallWait(wait bool) InstallOption {
	return func(opts *options) {
		opts.instReq.Wait = wait
	}
}

// UpgradeWait specifies whether or not to wait for all resources to be ready
func UpgradeWait(wait bool) UpdateOption {
	return func(opts *options) {
		opts.updateReq.Wait = wait
	}
}

// RollbackWait specifies whether or not to wait for all resources to be ready
func RollbackWait(wait bool) RollbackOption {
	return func(opts *options) {
		opts.rollbackReq.Wait = wait
	}
}

// UpdateValueOverrides specifies a list of values to include when upgrading
func UpdateValueOverrides(raw map[string]interface{}) UpdateOption {
	return func(opts *options) {
		opts.updateReq.Values = raw
	}
}

// UninstallDisableHooks will disable hooks for a deletion operation.
func UninstallDisableHooks(disable bool) UninstallOption {
	return func(opts *options) {
		opts.disableHooks = disable
	}
}

// UninstallDryRun will (if true) execute a deletion as a dry run.
func UninstallDryRun(dry bool) UninstallOption {
	return func(opts *options) {
		opts.dryRun = dry
	}
}

// UninstallPurge removes the release from the store and make its name free for later use.
func UninstallPurge(purge bool) UninstallOption {
	return func(opts *options) {
		opts.uninstallReq.Purge = purge
	}
}

// InstallDryRun will (if true) execute an installation as a dry run.
func InstallDryRun(dry bool) InstallOption {
	return func(opts *options) {
		opts.dryRun = dry
	}
}

// InstallDisableHooks disables hooks during installation.
func InstallDisableHooks(disable bool) InstallOption {
	return func(opts *options) {
		opts.disableHooks = disable
	}
}

// InstallReuseName will (if true) instruct Helm to re-use an existing name.
func InstallReuseName(reuse bool) InstallOption {
	return func(opts *options) {
		opts.reuseName = reuse
	}
}

// RollbackDisableHooks will disable hooks for a rollback operation
func RollbackDisableHooks(disable bool) RollbackOption {
	return func(opts *options) {
		opts.disableHooks = disable
	}
}

// RollbackDryRun will (if true) execute a rollback as a dry run.
func RollbackDryRun(dry bool) RollbackOption {
	return func(opts *options) {
		opts.dryRun = dry
	}
}

// RollbackRecreate will (if true) recreate pods after rollback.
func RollbackRecreate(recreate bool) RollbackOption {
	return func(opts *options) {
		opts.recreate = recreate
	}
}

// RollbackForce will (if true) force resource update through uninstall/recreate if needed
func RollbackForce(force bool) RollbackOption {
	return func(opts *options) {
		opts.force = force
	}
}

// RollbackVersion sets the version of the release to deploy.
func RollbackVersion(ver int) RollbackOption {
	return func(opts *options) {
		opts.rollbackReq.Version = ver
	}
}

// UpgradeDisableHooks will disable hooks for an upgrade operation.
func UpgradeDisableHooks(disable bool) UpdateOption {
	return func(opts *options) {
		opts.disableHooks = disable
	}
}

// UpgradeDryRun will (if true) execute an upgrade as a dry run.
func UpgradeDryRun(dry bool) UpdateOption {
	return func(opts *options) {
		opts.dryRun = dry
	}
}

// ResetValues will (if true) trigger resetting the values to their original state.
func ResetValues(reset bool) UpdateOption {
	return func(opts *options) {
		opts.resetValues = reset
	}
}

// ReuseValues will cause Helm to reuse the values from the last release.
// This is ignored if ResetValues is true.
func ReuseValues(reuse bool) UpdateOption {
	return func(opts *options) {
		opts.reuseValues = reuse
	}
}

// UpgradeRecreate will (if true) recreate pods after upgrade.
func UpgradeRecreate(recreate bool) UpdateOption {
	return func(opts *options) {
		opts.recreate = recreate
	}
}

// UpgradeForce will (if true) force resource update through uninstall/recreate if needed
func UpgradeForce(force bool) UpdateOption {
	return func(opts *options) {
		opts.force = force
	}
}

// MaxHistory limits the maximum number of revisions saved per release
func MaxHistory(maxHistory int) UpdateOption {
	return func(opts *options) {
		opts.updateReq.MaxHistory = maxHistory
	}
}

// UninstallOption allows setting optional attributes when
// performing a UninstallRelease tiller rpc.
type UninstallOption func(*options)

// UpdateOption allows specifying various settings
// configurable by the helm client user for overriding
// the defaults used when running the `helm upgrade` command.
type UpdateOption func(*options)

// RollbackOption allows specififying various settings configurable
// by the helm client user for overriding the defaults used when
// running the `helm rollback` command.
type RollbackOption func(*options)

// ReleaseTestOption allows configuring optional request data for
// issuing a TestRelease rpc.
type ReleaseTestOption func(*options)

// Driver set the driver option
func Driver(d driver.Driver) Option {
	return func(opts *options) {
		opts.driver = d
	}
}

// KubeClient sets the cluster environment
func KubeClient(kc environment.KubeClient) Option {
	return func(opts *options) {
		opts.kubeClient = kc
	}
}

// Discovery sets the discovery interface
func Discovery(dc discovery.DiscoveryInterface) Option {
	return func(opts *options) {
		opts.discovery = dc
	}
}
