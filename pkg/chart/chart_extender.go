package chart

import (
	"text/template"

	"github.com/werf/3p-helm/pkg/cli"
	"github.com/werf/3p-helm/pkg/werf/file"
	"github.com/werf/3p-helm/pkg/werf/secrets/runtimedata"
)

type ChartExtender interface {
	ChartCreated(c *Chart) error
	ChartLoaded(files []*file.ChartExtenderBufferedFile) error
	ChartDependenciesLoaded() error
	MakeValues(inputVals map[string]interface{}) (map[string]interface{}, error)
	SetupTemplateFuncs(t *template.Template, funcMap template.FuncMap)

	LoadDir(dir string) (bool, []*file.ChartExtenderBufferedFile, error)
	LocateChart(name string, settings *cli.EnvSettings) (bool, string, error)
	ReadFile(filePath string) (bool, []byte, error)

	GetChartFileReader() file.ChartFileReader
	GetDisableDefaultSecretValues() bool
	GetSecretValueFiles() []string
	GetServiceValues() map[string]interface{}
	GetProjectDir() string
	GetChartDir() string
	SetChartDir(dir string)
	GetBuildChartDependenciesOpts() BuildChartDependenciesOptions
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
