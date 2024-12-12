package chart

import (
	"text/template"

	"github.com/werf/3p-helm/pkg/chart/loader"
	"github.com/werf/3p-helm/pkg/cli"
)

type ChartExtender interface {
	ChartCreated(c *Chart) error
	ChartLoaded(files []*loader.BufferedFile) error
	ChartDependenciesLoaded() error
	MakeValues(inputVals map[string]interface{}) (map[string]interface{}, error)
	SetupTemplateFuncs(t *template.Template, funcMap template.FuncMap)

	LoadDir(dir string) (bool, []*loader.BufferedFile, error)
	LocateChart(name string, settings *cli.EnvSettings) (bool, string, error)
	ReadFile(filePath string) (bool, []byte, error)
}
