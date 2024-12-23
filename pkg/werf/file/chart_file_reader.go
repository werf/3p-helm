package file

import (
	"context"
)

type ChartFileReader interface {
	LocateChart(ctx context.Context, name string) (string, error)
	ReadChartFile(ctx context.Context, filePath string) ([]byte, error)
	LoadChartDir(ctx context.Context, dir string) ([]*ChartExtenderBufferedFile, error)
}
