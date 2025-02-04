package chart

import (
	"github.com/werf/3p-helm/pkg/werf/file"
	"github.com/werf/3p-helm/pkg/werf/secrets/runtimedata"
)

type ChartExtender interface {
	GetBuildChartDependenciesOpts() BuildChartDependenciesOptions
	GetChartDir() string
	GetChartFileReader() file.ChartFileReader
	GetDisableDefaultSecretValues() bool
	GetDisableDefaultValues() bool
	GetProjectDir() string
	GetSecretValueFiles() []string
	GetServiceValues() map[string]interface{}
	SetChartDir(dir string)
	SetHelmChart(c *Chart)
	Type() string
}

type BuildChartDependenciesOptions struct {
	LoadOptions *LoadOptions
}

type LoadOptions struct {
	ChartExtender                 ChartExtender
	SubchartExtenderFactoryFunc   func() ChartExtender
	SecretsRuntimeDataFactoryFunc func() runtimedata.RuntimeData
}
