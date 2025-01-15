package chartextender

import (
	"context"
	"text/template"

	"github.com/werf/3p-helm/pkg/chart"
	"github.com/werf/3p-helm/pkg/cli"
	"github.com/werf/3p-helm/pkg/werf/file"
)

var _ chart.ChartExtender = (*WerfChartStub)(nil)

func NewWerfChartStub(ctx context.Context) *WerfChartStub {
	return &WerfChartStub{}
}

type WerfChartStub struct {
	HelmChart *chart.Chart
	ChartDir  string
}

// ChartCreated method for the chart.Extender interface
func (wc *WerfChartStub) ChartCreated(c *chart.Chart) error {
	wc.HelmChart = c
	return nil
}

// ChartLoaded method for the chart.Extender interface
func (wc *WerfChartStub) ChartLoaded(files []*file.ChartExtenderBufferedFile) error {
	var opts GetHelmChartMetadataOptions
	opts.DefaultName = "stubchartname"
	opts.DefaultVersion = "1.0.0"
	wc.HelmChart.Metadata = AutosetChartMetadata(wc.HelmChart.Metadata, opts)

	wc.HelmChart.Templates = append(wc.HelmChart.Templates, &chart.File{
		Name: "templates/_werf_helpers.tpl",
	})

	return nil
}

// ChartDependenciesLoaded method for the chart.Extender interface
func (wc *WerfChartStub) ChartDependenciesLoaded() error {
	return nil
}

// MakeValues method for the chart.Extender interface
func (wc *WerfChartStub) MakeValues(_ map[string]interface{}) (
	map[string]interface{},
	error,
) {
	return nil, nil
}

// SetupTemplateFuncs method for the chart.Extender interface
func (wc *WerfChartStub) SetupTemplateFuncs(t *template.Template, funcMap template.FuncMap) {
}

// LoadDir method for the chart.Extender interface
func (wc *WerfChartStub) LoadDir(dir string) (bool, []*file.ChartExtenderBufferedFile, error) {
	return false, nil, nil
}

// LocateChart method for the chart.Extender interface
func (wc *WerfChartStub) LocateChart(name string, settings *cli.EnvSettings) (bool, string, error) {
	return false, "", nil
}

// ReadFile method for the chart.Extender interface
func (wc *WerfChartStub) ReadFile(filePath string) (bool, []byte, error) {
	return false, nil, nil
}

func (wc *WerfChartStub) Type() string {
	return "chartstub"
}

func (wc *WerfChartStub) GetChartFileReader() file.ChartFileReader {
	panic("not implemented")
}

func (wc *WerfChartStub) GetDisableDefaultSecretValues() bool {
	panic("not implemented")
}

func (wc *WerfChartStub) GetSecretValueFiles() []string {
	return []string{}
}

func (wc *WerfChartStub) GetServiceValues() map[string]interface{} {
	panic("not implemented")
}

func (wc *WerfChartStub) GetProjectDir() string {
	panic("not implemented")
}

func (wc *WerfChartStub) GetChartDir() string {
	return wc.ChartDir
}

func (wc *WerfChartStub) SetChartDir(dir string) {
	wc.ChartDir = dir
}

func (wc *WerfChartStub) GetHelmEnvSettings() *cli.EnvSettings {
	panic("not implemented")
}

func (wc *WerfChartStub) GetBuildChartDependenciesOpts() chart.BuildChartDependenciesOptions {
	panic("not implemented")
}
