package chart

type ChartType string

const (
	ChartTypeChart     ChartType = "chart"
	ChartTypeBundle    ChartType = "bundle"
	ChartTypeSubchart  ChartType = "subchart"
	ChartTypeChartStub ChartType = "chartstub"
)

var CurrentChartType ChartType = ChartTypeChart

type LoadOptions struct{}
