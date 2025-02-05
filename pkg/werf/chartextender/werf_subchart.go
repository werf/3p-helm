package chartextender

import (
	"github.com/werf/3p-helm/pkg/chart"
	"github.com/werf/3p-helm/pkg/werf/file"
)

var _ chart.ChartExtender = (*WerfSubchart)(nil)

type WerfSubchartOptions struct{}

func NewWerfSubchart(opts WerfSubchartOptions) *WerfSubchart {
	return &WerfSubchart{}
}

type WerfSubchart struct{}

func (wc *WerfSubchart) Type() string {
	return "subchart"
}

func (wc *WerfSubchart) GetChartFileReader() file.ChartFileReader {
	panic("not implemented")
}

func (wc *WerfSubchart) GetDisableDefaultValues() bool {
	panic("not implemented")
}

func (wc *WerfSubchart) GetSecretValueFiles() []string {
	panic("not implemented")
}

func (wc *WerfSubchart) GetProjectDir() string {
	panic("not implemented")
}

func (wc *WerfSubchart) GetChartDir() string {
	panic("not implemented")
}

func (wc *WerfSubchart) SetChartDir(dir string) {
	panic("not implemented")
}

func (wc *WerfSubchart) GetBuildChartDependenciesOpts() chart.BuildChartDependenciesOptions {
	panic("not implemented")
}
