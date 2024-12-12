/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package loader

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/werf/3p-helm/internal/sympath"
	"github.com/werf/3p-helm/pkg/chart"
	"github.com/werf/3p-helm/pkg/cli"
	"github.com/werf/3p-helm/pkg/downloader"
	"github.com/werf/3p-helm/pkg/getter"
	"github.com/werf/3p-helm/pkg/ignore"
	"github.com/werf/3p-helm/pkg/provenance"
	"github.com/werf/3p-helm/pkg/registry"
	"github.com/werf/3p-helm/pkg/werfcompat"
	"sigs.k8s.io/yaml"

	"github.com/werf/lockgate"
	"github.com/werf/logboek"
	"github.com/werf/logboek/pkg/types"
)

var utf8bom = []byte{0xEF, 0xBB, 0xBF}

// DirLoader loads a chart from a directory
type DirLoader string

// Load loads the chart
func (l DirLoader) Load(opts LoadOptions) (*chart.Chart, error) {
	return LoadDir(string(l), opts)
}

// LoadDir loads from a directory.
//
// This loads charts only from directories.
func LoadDir(dir string, opts LoadOptions) (*chart.Chart, error) {
	chartFiles, err := werfcompat.Reader.LoadChartDir(context.TODO(), dir)
	if err != nil {
		return nil, fmt.Errorf("load chart directory: %w", err)
	}

	files, err := LoadChartDependencies(context.TODO(), dir, chartFiles, werfcompat.EnvSettings, werfcompat.RegistryClient, werfcompat.BuildChartDependenciesOpts)
	if err != nil {
		return nil, fmt.Errorf("load chart dependencies: %w", err)
	}

	return LoadFiles(files, opts)
}

func GetFilesFromLocalFilesystem(dir string) ([]*BufferedFile, error) {
	topdir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	rules := ignore.Empty()
	ifile := filepath.Join(topdir, ignore.HelmIgnore)
	if _, err := os.Stat(ifile); err == nil {
		r, err := ignore.ParseFile(ifile)
		if err != nil {
			return nil, err
		}
		rules = r
	}
	rules.AddDefaults()

	files := []*BufferedFile{}
	topdir += string(filepath.Separator)

	walk := func(name string, fi os.FileInfo, err error) error {
		n := strings.TrimPrefix(name, topdir)
		if n == "" {
			// No need to process top level. Avoid bug with helmignore .* matching
			// empty names. See issue 1779.
			return nil
		}

		// Normalize to / since it will also work on Windows
		n = filepath.ToSlash(n)

		if err != nil {
			return err
		}
		if fi.IsDir() {
			// Directory-based ignore rules should involve skipping the entire
			// contents of that directory.
			if rules.Ignore(n, fi) {
				return filepath.SkipDir
			}
			return nil
		}

		// If a .helmignore file matches, skip this file.
		if rules.Ignore(n, fi) {
			return nil
		}

		// Irregular files include devices, sockets, and other uses of files that
		// are not regular files. In Go they have a file mode type bit set.
		// See https://golang.org/pkg/os/#FileMode for examples.
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("cannot load irregular file %s as it has file mode type bits set", name)
		}

		data, err := os.ReadFile(name)
		if err != nil {
			return errors.Wrapf(err, "error reading %s", n)
		}

		data = bytes.TrimPrefix(data, utf8bom)

		files = append(files, &BufferedFile{Name: n, Data: data})
		return nil
	}
	if err = sympath.Walk(topdir, walk); err != nil {
		return nil, err
	}

	return files, nil
}

func LoadChartDependencies(
	ctx context.Context,
	chartDir string,
	loadedChartFiles []*BufferedFile,
	helmEnvSettings *cli.EnvSettings,
	registryClient *registry.Client,
	buildChartDependenciesOpts werfcompat.BuildChartDependenciesOptions,
) ([]*BufferedFile, error) {
	res := loadedChartFiles

	var chartMetadata *chart.Metadata
	var chartMetadataLock *chart.Lock

	for _, f := range loadedChartFiles {
		switch f.Name {
		case "Chart.yaml":
			chartMetadata = new(chart.Metadata)
			if err := yaml.Unmarshal(f.Data, chartMetadata); err != nil {
				return nil, errors.Wrap(err, "cannot load Chart.yaml")
			}
			if chartMetadata.APIVersion == "" {
				chartMetadata.APIVersion = chart.APIVersionV1
			}

		case "Chart.lock":
			chartMetadataLock = new(chart.Lock)
			if err := yaml.Unmarshal(f.Data, chartMetadataLock); err != nil {
				return nil, errors.Wrap(err, "cannot load Chart.lock")
			}
		}
	}

	for _, f := range loadedChartFiles {
		switch f.Name {
		case "requirements.yaml":
			if chartMetadata == nil {
				chartMetadata = new(chart.Metadata)
			}
			if err := yaml.Unmarshal(f.Data, chartMetadata); err != nil {
				return nil, errors.Wrap(err, "cannot load requirements.yaml")
			}

		case "requirements.lock":
			if chartMetadataLock == nil {
				chartMetadataLock = new(chart.Lock)
			}
			if err := yaml.Unmarshal(f.Data, chartMetadataLock); err != nil {
				return nil, errors.Wrap(err, "cannot load requirements.lock")
			}
		}
	}

	if chartMetadata == nil {
		return res, nil
	}

	if chartMetadataLock == nil {
		if len(chartMetadata.Dependencies) > 0 {
			logboek.Context(ctx).Error().LogLn("Cannot build chart dependencies and preload charts without lock file (.helm/Chart.lock or .helm/requirements.lock)")
			logboek.Context(ctx).Error().LogLn("It is recommended to add Chart.lock file to your project repository or remove chart dependencies.")
			logboek.Context(ctx).Error().LogLn()
			logboek.Context(ctx).Error().LogLn("To generate a lock file run 'werf helm dependency update .helm' and commit resulting .helm/Chart.lock or .helm/requirements.lock (it is not required to commit whole .helm/charts directory, better add it to the .gitignore).")
			logboek.Context(ctx).Error().LogLn()
		}

		return res, nil
	}

	conf := NewChartDependenciesConfiguration(chartMetadata, chartMetadataLock)

	// Append virtually loaded files from custom dependency repositories in the local filesystem,
	// pretending these files are located in the charts/ dir as designed in the Helm.
	for _, chartDep := range chartMetadataLock.Dependencies {
		if !strings.HasPrefix(chartDep.Repository, "file://") {
			continue
		}

		relativeLocalChartPath := strings.TrimPrefix(chartDep.Repository, "file://")
		localChartPath := filepath.Join(chartDir, relativeLocalChartPath)

		chartFiles, err := werfcompat.Reader.LoadChartDir(ctx, localChartPath)
		if err != nil {
			return nil, fmt.Errorf("unable to load custom subchart dir %q: %w", localChartPath, err)
		}

		for _, f := range chartFiles {
			f.Name = filepath.Join("charts", chartDep.Name, f.Name)
		}

		res = append(res, chartFiles...)
	}

	haveExternalDependencies, metadataFile, metadataLockFile, err := conf.GetExternalDependenciesFiles(loadedChartFiles)
	if err != nil {
		return nil, fmt.Errorf("unable to get external dependencies chart configuration files: %w", err)
	}
	if !haveExternalDependencies {
		return res, nil
	}

	depsDir, err := GetPreparedChartDependenciesDir(ctx, metadataFile, metadataLockFile, helmEnvSettings, registryClient, buildChartDependenciesOpts)
	if err != nil {
		return nil, fmt.Errorf("error preparing chart dependencies: %w", err)
	}
	localFiles, err := GetFilesFromLocalFilesystem(depsDir)
	if err != nil {
		return nil, err
	}

	for _, f := range localFiles {
		if strings.HasPrefix(f.Name, "charts/") {
			f1 := new(BufferedFile)
			*f1 = BufferedFile(*f)
			res = append(res, f1)
			logboek.Context(ctx).Debug().LogF("-- LoadChartDependencies: loading subchart %q from the dependencies dir %q\n", f.Name, depsDir)
		}
	}

	return res, nil
}

func NewChartDependenciesConfiguration(chartMetadata *chart.Metadata, chartMetadataLock *chart.Lock) *ChartDependenciesConfiguration {
	return &ChartDependenciesConfiguration{ChartMetadata: chartMetadata, ChartMetadataLock: chartMetadataLock}
}

type ChartDependenciesConfiguration struct {
	ChartMetadata     *chart.Metadata
	ChartMetadataLock *chart.Lock
}

func (conf *ChartDependenciesConfiguration) GetExternalDependenciesFiles(loadedChartFiles []*BufferedFile) (bool, *BufferedFile, *BufferedFile, error) {
	metadataBytes, err := yaml.Marshal(conf.ChartMetadata)
	if err != nil {
		return false, nil, nil, fmt.Errorf("unable to marshal original chart metadata into yaml: %w", err)
	}
	metadata := new(chart.Metadata)
	if err := yaml.Unmarshal(metadataBytes, metadata); err != nil {
		return false, nil, nil, fmt.Errorf("unable to unmarshal original chart metadata yaml: %w", err)
	}

	metadataLockBytes, err := yaml.Marshal(conf.ChartMetadataLock)
	if err != nil {
		return false, nil, nil, fmt.Errorf("unable to marshal original chart metadata lock into yaml: %w", err)
	}
	metadataLock := new(chart.Lock)
	if err := yaml.Unmarshal(metadataLockBytes, metadataLock); err != nil {
		return false, nil, nil, fmt.Errorf("unable to unmarshal original chart metadata lock yaml: %w", err)
	}

	metadata.APIVersion = "v2"

	var externalDependenciesNames []string
	isExternalDependency := func(depName string) bool {
		for _, externalDepName := range externalDependenciesNames {
			if depName == externalDepName {
				return true
			}
		}
		return false
	}

FindExternalDependencies:
	for _, depLock := range metadataLock.Dependencies {
		if depLock.Repository == "" || strings.HasPrefix(depLock.Repository, "file://") {
			continue
		}

		for _, loadedFile := range loadedChartFiles {
			if strings.HasPrefix(loadedFile.Name, "charts/") {
				if filepath.Base(loadedFile.Name) == MakeDependencyArchiveName(depLock.Name, depLock.Version) {
					continue FindExternalDependencies
				}
			}
		}

		externalDependenciesNames = append(externalDependenciesNames, depLock.Name)
	}

	var filteredLockDependencies []*chart.Dependency
	for _, depLock := range metadataLock.Dependencies {
		if isExternalDependency(depLock.Name) {
			filteredLockDependencies = append(filteredLockDependencies, depLock)
		}
	}
	metadataLock.Dependencies = filteredLockDependencies

	var filteredDependencies []*chart.Dependency
	for _, dep := range metadata.Dependencies {
		if isExternalDependency(dep.Name) {
			filteredDependencies = append(filteredDependencies, dep)
		}
	}
	metadata.Dependencies = filteredDependencies

	if len(metadata.Dependencies) == 0 {
		return false, nil, nil, nil
	}

	// Set resolved repository from the lock file
	for _, dep := range metadata.Dependencies {
		for _, depLock := range metadataLock.Dependencies {
			if dep.Name == depLock.Name {
				dep.Repository = depLock.Repository
				break
			}
		}
	}

	if newDigest, err := HashReq(metadata.Dependencies, metadataLock.Dependencies); err != nil {
		return false, nil, nil, fmt.Errorf("unable to calculate external dependencies Chart.yaml digest: %w", err)
	} else {
		metadataLock.Digest = newDigest
	}

	metadataFile := &BufferedFile{Name: "Chart.yaml"}
	if data, err := yaml.Marshal(metadata); err != nil {
		return false, nil, nil, fmt.Errorf("unable to marshal chart metadata file with external dependencies: %w", err)
	} else {
		metadataFile.Data = data
	}

	metadataLockFile := &BufferedFile{Name: "Chart.lock"}
	if data, err := yaml.Marshal(metadataLock); err != nil {
		return false, nil, nil, fmt.Errorf("unable to marshal chart metadata lock file with external dependencies: %w", err)
	} else {
		metadataLockFile.Data = data
	}

	return true, metadataFile, metadataLockFile, nil
}

func MakeDependencyArchiveName(depName, depVersion string) string {
	return fmt.Sprintf("%s-%s.tgz", depName, depVersion)
}

func HashReq(req, lock []*chart.Dependency) (string, error) {
	data, err := json.Marshal([2][]*chart.Dependency{req, lock})
	if err != nil {
		return "", err
	}
	s, err := provenance.Digest(bytes.NewBuffer(data))
	return "sha256:" + s, err
}

func GetPreparedChartDependenciesDir(ctx context.Context, metadataFile, metadataLockFile *BufferedFile, helmEnvSettings *cli.EnvSettings, registryClient *registry.Client, buildChartDependenciesOpts werfcompat.BuildChartDependenciesOptions) (string, error) {
	return prepareDependenciesDir(ctx, metadataFile.Data, metadataLockFile.Data, func(tmpDepsDir string) error {
		if err := BuildChartDependenciesInDir(ctx, tmpDepsDir, helmEnvSettings, registryClient, buildChartDependenciesOpts); err != nil {
			return fmt.Errorf("error building chart dependencies: %w", err)
		}
		return nil
	}, logboek.Context(ctx).Default())
}

func prepareDependenciesDir(ctx context.Context, metadataBytes, metadataLockBytes []byte, prepareFunc func(tmpDepsDir string) error, logger types.ManagerInterface) (string, error) {
	depsDir := filepath.Join(werfcompat.ChartDependenciesCacheDir, Sha256Hash(string(metadataLockBytes)))

	_, err := os.Stat(depsDir)
	switch {
	case os.IsNotExist(err):
		if err := logger.LogProcess("Preparing chart dependencies").DoError(func() error {
			logger.LogF("Using chart dependencies directory: %s\n", depsDir)
			_, lock, err := werfcompat.AcquireHostLock(ctx, depsDir, lockgate.AcquireOptions{})
			if err != nil {
				return fmt.Errorf("error acquiring lock for %q: %w", depsDir, err)
			}
			defer werfcompat.ReleaseHostLock(lock)

			switch _, err := os.Stat(depsDir); {
			case os.IsNotExist(err):
			case err != nil:
				return fmt.Errorf("error accessing %s: %w", depsDir, err)
			default:
				// at the time we have acquired a lock the target directory was created
				return nil
			}

			tmpDepsDir := fmt.Sprintf("%s.tmp.%s", depsDir, uuid.NewString())

			if err := createChartDependenciesDir(tmpDepsDir, metadataBytes, metadataLockBytes); err != nil {
				return err
			}

			if err := prepareFunc(tmpDepsDir); err != nil {
				return err
			}

			if err := os.Rename(tmpDepsDir, depsDir); err != nil {
				return fmt.Errorf("error renaming %q to %q: %w", tmpDepsDir, depsDir, err)
			}
			return nil
		}); err != nil {
			return "", err
		}
	case err != nil:
		return "", fmt.Errorf("error accessing %q: %w", depsDir, err)
	default:
		logger.LogF("Using cached chart dependencies directory: %s\n", depsDir)
	}

	return depsDir, nil
}

func Sha256Hash(args ...string) string {
	sum := sha256.Sum256([]byte(prepareHashArgs(args...)))
	return fmt.Sprintf("%x", sum)
}

func prepareHashArgs(args ...string) string {
	return strings.Join(args, ":::")
}

func createChartDependenciesDir(destDir string, metadataBytes, metadataLockBytes []byte) error {
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating dir %q: %w", destDir, err)
	}

	files := []*BufferedFile{
		{Name: "Chart.yaml", Data: metadataBytes},
		{Name: "Chart.lock", Data: metadataLockBytes},
	}

	for _, file := range files {
		if file == nil {
			continue
		}

		path := filepath.Join(destDir, file.Name)
		if err := ioutil.WriteFile(path, file.Data, 0o644); err != nil {
			return fmt.Errorf("error writing %q: %w", path, err)
		}
	}

	return nil
}

func BuildChartDependenciesInDir(ctx context.Context, targetDir string, helmEnvSettings *cli.EnvSettings, registryClient *registry.Client, opts werfcompat.BuildChartDependenciesOptions) error {
	logboek.Context(ctx).Debug().LogF("-- BuildChartDependenciesInDir\n")

	man := &downloader.Manager{
		Out:               logboek.Context(ctx).OutStream(),
		ChartPath:         targetDir,
		Keyring:           werfcompat.BuildChartDependenciesOpts.Keyring,
		SkipUpdate:        werfcompat.BuildChartDependenciesOpts.SkipUpdate,
		Verify:            werfcompat.BuildChartDependenciesOpts.Verify,
		AllowMissingRepos: true,

		Getters:          getter.All(helmEnvSettings),
		RegistryClient:   registryClient,
		RepositoryConfig: helmEnvSettings.RepositoryConfig,
		RepositoryCache:  helmEnvSettings.RepositoryCache,
		Debug:            helmEnvSettings.Debug,
	}

	err := man.Build()

	if e, ok := err.(downloader.ErrRepoNotFound); ok {
		return fmt.Errorf("%w. Please add the missing repos via 'helm repo add'", e)
	}

	return err
}
