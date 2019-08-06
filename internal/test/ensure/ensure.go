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

package ensure

import (
	"io/ioutil"
	"os"
	"testing"

	"helm.sh/helm/pkg/helmpath"
	"helm.sh/helm/pkg/helmpath/xdg"
)

// HelmHome sets up a Helm Home in a temp dir.
func HelmHome(t *testing.T) {
	t.Helper()
	cachePath := TempDir(t)
	configPath := TempDir(t)
	dataPath := TempDir(t)
	os.Setenv(xdg.CacheHomeEnvVar, cachePath)
	os.Setenv(xdg.ConfigHomeEnvVar, configPath)
	os.Setenv(xdg.DataHomeEnvVar, dataPath)
	HomeDirs(t)
}

// HomeDirs creates a home directory like ensureHome, but without remote references.
func HomeDirs(t *testing.T) {
	t.Helper()
	for _, p := range []string{
		helmpath.CachePath(),
		helmpath.ConfigPath(),
		helmpath.DataPath(),
		helmpath.RepositoryCache(),
		helmpath.Plugins(),
		helmpath.PluginCache(),
		helmpath.Starters(),
	} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
	}
}

// CleanHomeDirs removes the directories created by HomeDirs.
func CleanHomeDirs(t *testing.T) {
	t.Helper()
	for _, p := range []string{
		helmpath.CachePath(),
		helmpath.ConfigPath(),
		helmpath.DataPath(),
		helmpath.RepositoryCache(),
		helmpath.Plugins(),
		helmpath.PluginCache(),
		helmpath.Starters(),
	} {
		if err := os.RemoveAll(p); err != nil {
			t.Log(err)
		}
	}
}

// TempDir ensures a scratch test directory for unit testing purposes.
func TempDir(t *testing.T) string {
	t.Helper()
	d, err := ioutil.TempDir("", "helm")
	if err != nil {
		t.Fatal(err)
	}
	return d
}
