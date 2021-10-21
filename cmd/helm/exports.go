package helm_v3

import (
	"io"

	"helm.sh/helm/v3/internal/experimental/registry"
)

var (
	Settings = settings

	LoadReleasesInMemory = loadReleasesInMemory
	Debug                = debug
	NewRootCmd           = newRootCmd

	NewDependencyCmd = newDependencyCmd
	NewGetCmd        = newGetCmd
	NewHistoryCmd    = newHistoryCmd
	NewLintCmd       = newLintCmd
	NewListCmd       = newListCmd
	NewRepoCmd       = newRepoCmd
	NewRollbackCmd   = newRollbackCmd
	NewCreateCmd     = newCreateCmd
	NewEnvCmd        = newEnvCmd
	NewPackageCmd    = newPackageCmd
	NewPluginCmd     = newPluginCmd
	NewSearchCmd     = newSearchCmd
	NewStatusCmd     = newStatusCmd
	NewTestCmd       = newReleaseTestCmd
	NewVerifyCmd     = newVerifyCmd
	NewVersionCmd    = newVersionCmd
	NewShowCmd       = newShowCmd
	NewRegistryCmd   = newRegistryCmd

	// NOTE: following commands has additional options param and defined in corresponding command files
	// NewTemplateCmd   = newTemplateCmd
	// NewInstallCmd    = newInstallCmd
	// NewUpgradeCmd    = newUpgradeCmd
	// NewUninstallCmd  = newUninstallCmd
	NewPullCmd = newPullCmd
	NewPushCmd = newPushCmd

	LoadPlugins = loadPlugins
)

func NewRegistryClient(debug, insecure bool, out io.Writer) (*registry.Client, error) {
	return registry.NewClient(
		registry.ClientOptDebug(debug),
		registry.ClientOptInsecure(insecure),
		registry.ClientOptWriter(out),
	)
}

type RegistryClientHandle struct {
	RegistryClient *registry.Client
}

func NewRegistryClientHandle(registryClient *registry.Client) *RegistryClientHandle {
	return &RegistryClientHandle{RegistryClient: registryClient}
}
