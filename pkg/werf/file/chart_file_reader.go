package file

import (
	"context"

	"github.com/werf/3p-helm/pkg/chart"
	"github.com/werf/3p-helm/pkg/cli"
)

type ChartFileReader interface {
	LocateChart(ctx context.Context, name string, settings *cli.EnvSettings) (string, error)
	ReadChartFile(ctx context.Context, filePath string) ([]byte, error)
	LoadChartDir(ctx context.Context, dir string) ([]*chart.ChartExtenderBufferedFile, error)
}
