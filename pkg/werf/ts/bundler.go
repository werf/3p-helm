package ts

import (
	"context"

	helmchart "github.com/werf/3p-helm/pkg/chart"
)

type Bundler interface {
	BundleChartsRecursive(ctx context.Context, chart *helmchart.Chart, path string) error
}

var DefaultBundler Bundler
