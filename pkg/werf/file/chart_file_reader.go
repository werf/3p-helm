package file

import (
	"context"
)

type ChartFileReaderInterface interface {
	LocateChart(ctx context.Context, name string) (string, error)
	ReadChartFile(ctx context.Context, filePath string) ([]byte, error)
	LoadChartDir(ctx context.Context, dir string) ([]*ChartExtenderBufferedFile, error)
}

// FIXME(ilya-lesikov): keep it global, but separate package? Make non-giterminism default implementation
var ChartFileReader ChartFileReaderInterface
