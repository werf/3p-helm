package werfcompat

import (
	"context"
	"os"
	"path/filepath"

	"github.com/samber/lo"
	helm_v3 "github.com/werf/3p-helm/cmd/helm"
	"github.com/werf/3p-helm/pkg/action"
	"github.com/werf/3p-helm/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/registry"

	"github.com/werf/lockgate"
)

var ChartDependenciesCacheDir string = filepath.Join(lo.Must(os.UserHomeDir()), ".werf", "local_cache", "helm_chart_dependencies", "1")

var Reader ChartFileReader = NewBasicChartFileReader()

var RegistryClient *registry.Client

var BuildChartDependenciesOpts BuildChartDependenciesOptions

var EnvSettings *cli.EnvSettings = helm_v3.Settings

var ChartPathOptions action.ChartPathOptions

var AcquireHostLock AcquireHostLocker = func(
	ctx context.Context,
	j string,
	opts lockgate.AcquireOptions,
) (bool, lockgate.LockHandle, error) {
	return true, lockgate.LockHandle{}, nil
}

var ReleaseHostLock ReleaseHostLocker = func(lock lockgate.LockHandle) error {
	return nil
}

type ChartFileReader interface {
	LocateChart(ctx context.Context, name string, settings *cli.EnvSettings) (string, error)
	ReadChartFile(ctx context.Context, filePath string) ([]byte, error)
	LoadChartDir(ctx context.Context, dir string) ([]*loader.BufferedFile, error)
}

func NewBasicChartFileReader() *BasicChartFileReader {
	return &BasicChartFileReader{}
}

type BasicChartFileReader struct{}

func (r *BasicChartFileReader) LocateChart(ctx context.Context, name string, settings *cli.EnvSettings) (string, error) {
	return ChartPathOptions.LocateChart(name, settings)
}

func (r *BasicChartFileReader) ReadChartFile(ctx context.Context, filePath string) ([]byte, error) {
	// FIXME(ilya-lesikov):
	panic("implement me")
}

func (r *BasicChartFileReader) LoadChartDir(ctx context.Context, dir string) ([]*loader.BufferedFile, error) {
	return loader.GetFilesFromLocalFilesystem(dir)
}

type AcquireHostLocker func(ctx context.Context, lockName string, opts lockgate.AcquireOptions) (bool, lockgate.LockHandle, error)

type ReleaseHostLocker func(lock lockgate.LockHandle) error

type BuildChartDependenciesOptions struct {
	Keyring                           string
	SkipUpdate                        bool
	Verify                            downloader.VerificationStrategy
	IgnoreInvalidAnnotationsAndLabels bool
}
