package tsbundle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/samber/lo"

	helmchart "github.com/werf/3p-helm/pkg/chart"
)

const (
	// ChartTSSourceDir is the directory containing TypeScript sources in a Helm chart.
	ChartTSSourceDir = "ts/"
	// ChartTSBundleFile is the path to the bundle in a Helm chart.
	ChartTSBundleFile = ChartTSSourceDir + "dist/bundle.js"
	// ChartTSEntryPointTS is the TypeScript entry point path.
	ChartTSEntryPointTS = "src/index.ts"
	// ChartTSEntryPointJS is the JavaScript entry point path.
	ChartTSEntryPointJS = "src/index.js"
)

// ChartTSEntryPoints defines supported TypeScript/JavaScript entry points (in priority order).
var ChartTSEntryPoints = [...]string{ChartTSEntryPointTS, ChartTSEntryPointJS}

var (
	BundleEnabled = false
	denoBinary    = ""
)

func GetDenoBinary() string {
	return denoBinary
}

func SetDenoBinary(path string) {
	denoBinary = path
}

func runDenoBundle(ctx context.Context, chartPath, entryPoint string) ([]uint8, error) {
	denoBin := GetDenoBinary()
	cmd := exec.CommandContext(ctx, denoBin, "bundle", entryPoint)
	cmd.Dir = filepath.Join(chartPath, ChartTSSourceDir)

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			_, _ = os.Stderr.Write(exitErr.Stderr)
		}

		return nil, fmt.Errorf("get deno build output: %w", err)
	}

	return output, nil
}

func BundleTSChartsRecursive(ctx context.Context, chart *helmchart.Chart, path string, rebuild bool) error {
	entrypoint, bundle := GetEntrypointAndBundle(chart.RuntimeFiles)
	if entrypoint == "" {
		return nil
	}

	if bundle == nil || rebuild {
		bundleRes, err := runDenoBundle(ctx, path, entrypoint)
		if err != nil {
			return fmt.Errorf("build TypeScript bundle: %w", err)
		}

		if rebuild && bundle != nil {
			chart.RemoveRuntimeFile(ChartTSBundleFile)
		}

		chart.AddRuntimeFile(ChartTSBundleFile, bundleRes)
	}

	deps := chart.Dependencies()
	if len(deps) == 0 {
		return nil
	}

	for _, dep := range deps {
		depPath := filepath.Join(path, "charts", dep.Name())

		if _, err := os.Stat(depPath); err != nil {
			// Subchart loaded from .tgz or missing on disk — skip,
			// deno bundle needs a real directory to work with.
			continue
		}

		if err := BundleTSChartsRecursive(ctx, dep, depPath, rebuild); err != nil {
			return fmt.Errorf("process dependency %q: %w", dep.Name(), err)
		}
	}

	return nil
}

func GetEntrypointAndBundle(files []*helmchart.File) (string, *helmchart.File) {
	entrypoint := findEntrypointInFiles(files)
	if entrypoint == "" {
		return "", nil
	}

	bundleFile, foundBundle := lo.Find(files, func(f *helmchart.File) bool {
		return f.Name == ChartTSBundleFile
	})

	if !foundBundle {
		return entrypoint, nil
	}

	return entrypoint, bundleFile
}

func findEntrypointInFiles(files []*helmchart.File) string {
	sourceFiles := make(map[string][]byte)

	for _, f := range files {
		if strings.HasPrefix(f.Name, ChartTSSourceDir+"src/") {
			sourceFiles[strings.TrimPrefix(f.Name, ChartTSSourceDir)] = f.Data
		}
	}

	if len(sourceFiles) == 0 {
		return ""
	}

	for _, ep := range ChartTSEntryPoints {
		if _, ok := sourceFiles[ep]; ok {
			return ep
		}
	}

	return ""
}
