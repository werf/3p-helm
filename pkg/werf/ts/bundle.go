package tsruntime

import (
	"context"

	helmchart "github.com/werf/3p-helm/pkg/chart"
)

type RuntimeInterface interface {
	BundleChartsRecursive(ctx context.Context, chart *helmchart.Chart, path string) error
}

var TSRuntime RuntimeInterface
