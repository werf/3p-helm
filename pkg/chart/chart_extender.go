package chart

import (
	"github.com/werf/3p-helm/pkg/werf/file"
	"github.com/werf/3p-helm/pkg/werf/secrets/runtimedata"
)

type ChartExtender interface {
	GetBuildChartDependenciesOpts() BuildChartDependenciesOptions
	GetChartDir() string
	GetChartFileReader() file.ChartFileReader
	GetDisableDefaultValues() bool
	GetProjectDir() string
	GetSecretValueFiles() []string
	SetChartDir(dir string)
	Type() string
}

type BuildChartDependenciesOptions struct {
	LoadOptions *LoadOptions
}

type LoadOptions struct {
	ChartExtender                 ChartExtender
	SecretsRuntimeDataFactoryFunc func() runtimedata.RuntimeData
}
