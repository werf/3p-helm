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

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/lint"
	"k8s.io/helm/pkg/lint/support"
	"k8s.io/helm/pkg/strvals"
)

var longLintHelp = `
This command takes a path to a chart and runs a series of tests to verify that
the chart is well-formed.

If the linter encounters things that will cause the chart to fail installation,
it will emit [ERROR] messages. If it encounters issues that break with convention
or recommendation, it will emit [WARNING] messages.
`

type lintOptions struct {
	strict bool
	paths  []string

	valuesOptions
}

func newLintCmd(out io.Writer) *cobra.Command {
	o := &lintOptions{paths: []string{"."}}

	cmd := &cobra.Command{
		Use:   "lint PATH",
		Short: "examines a chart for possible issues",
		Long:  longLintHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				o.paths = args
			}
			return o.run(out)
		},
	}

	fs := cmd.Flags()
	fs.BoolVar(&o.strict, "strict", false, "fail on lint warnings")
	o.valuesOptions.addFlags(fs)

	return cmd
}

var errLintNoChart = errors.New("no chart found for linting (missing Chart.yaml)")

func (o *lintOptions) run(out io.Writer) error {
	var lowestTolerance int
	if o.strict {
		lowestTolerance = support.WarningSev
	} else {
		lowestTolerance = support.ErrorSev
	}

	// Get the raw values
	rvals, err := o.vals()
	if err != nil {
		return err
	}

	var total int
	var failures int
	for _, path := range o.paths {
		if linter, err := lintChart(path, rvals, getNamespace(), o.strict); err != nil {
			fmt.Println("==> Skipping", path)
			fmt.Println(err)
			if err == errLintNoChart {
				failures = failures + 1
			}
		} else {
			fmt.Println("==> Linting", path)

			if len(linter.Messages) == 0 {
				fmt.Println("Lint OK")
			}

			for _, msg := range linter.Messages {
				fmt.Println(msg)
			}

			total = total + 1
			if linter.HighestSeverity >= lowestTolerance {
				failures = failures + 1
			}
		}
		fmt.Println("")
	}

	msg := fmt.Sprintf("%d chart(s) linted", total)
	if failures > 0 {
		return errors.Errorf("%s, %d chart(s) failed", msg, failures)
	}

	fmt.Fprintf(out, "%s, no failures\n", msg)

	return nil
}

func lintChart(path string, vals []byte, namespace string, strict bool) (support.Linter, error) {
	var chartPath string
	linter := support.Linter{}

	if strings.HasSuffix(path, ".tgz") {
		tempDir, err := ioutil.TempDir("", "helm-lint")
		if err != nil {
			return linter, err
		}
		defer os.RemoveAll(tempDir)

		file, err := os.Open(path)
		if err != nil {
			return linter, err
		}
		defer file.Close()

		if err = chartutil.Expand(tempDir, file); err != nil {
			return linter, err
		}

		lastHyphenIndex := strings.LastIndex(filepath.Base(path), "-")
		if lastHyphenIndex <= 0 {
			return linter, errors.Errorf("unable to parse chart archive %q, missing '-'", filepath.Base(path))
		}
		base := filepath.Base(path)[:lastHyphenIndex]
		chartPath = filepath.Join(tempDir, base)
	} else {
		chartPath = path
	}

	// Guard: Error out of this is not a chart.
	if _, err := os.Stat(filepath.Join(chartPath, "Chart.yaml")); err != nil {
		return linter, errLintNoChart
	}

	return lint.All(chartPath, vals, namespace, strict), nil
}

func (o *lintOptions) vals() ([]byte, error) {
	base := map[string]interface{}{}

	// User specified a values files via -f/--values
	for _, filePath := range o.valueFiles {
		currentMap := map[string]interface{}{}
		bytes, err := ioutil.ReadFile(filePath)
		if err != nil {
			return []byte{}, err
		}

		if err := yaml.Unmarshal(bytes, &currentMap); err != nil {
			return []byte{}, errors.Wrapf(err, "failed to parse %s", filePath)
		}
		// Merge with the previous map
		base = mergeValues(base, currentMap)
	}

	// User specified a value via --set
	for _, value := range o.values {
		if err := strvals.ParseInto(value, base); err != nil {
			return []byte{}, errors.Wrap(err, "failed parsing --set data")
		}
	}

	// User specified a value via --set-string
	for _, value := range o.stringValues {
		if err := strvals.ParseIntoString(value, base); err != nil {
			return []byte{}, errors.Wrap(err, "failed parsing --set-string data")
		}
	}

	return yaml.Marshal(base)
}
