package chartextender

import (
	"context"

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

func (wc *WerfChartStub) SetHelmChart(c *chart.Chart) {
	wc.HelmChart = c
}

func (wc *WerfChartStub) Type() string {
	return "chartstub"
}

func (wc *WerfChartStub) GetChartFileReader() file.ChartFileReader {
	panic("not implemented")
}

func (wc *WerfChartStub) GetDisableDefaultValues() bool {
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
