package chartextender

import (
	"github.com/werf/3p-helm/pkg/chart"
	"github.com/werf/3p-helm/pkg/werf/file"
)

var _ chart.ChartExtender = (*WerfChartStub)(nil)

func NewWerfChartStub() *WerfChartStub {
	return &WerfChartStub{}
}

type WerfChartStub struct {
	chartDir string
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
	return wc.chartDir
}

func (wc *WerfChartStub) SetChartDir(dir string) {
	wc.chartDir = dir
}

func (wc *WerfChartStub) GetBuildChartDependenciesOpts() chart.BuildChartDependenciesOptions {
	panic("not implemented")
}
