package helm_v3

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
	// NewRollbackCmd   = newRollbackCmd
	NewPullCmd = newPullCmd
	NewPushCmd = newPushCmd

	LoadPlugins = loadPlugins
)
