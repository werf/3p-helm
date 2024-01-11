package chartutil

import (
	"log"

	"helm.sh/helm/v3/pkg/chart"
)

func CoalesceChartValues(c *chart.Chart, v map[string]interface{}, merge bool) {
	coalesceValues(log.Printf, c, v, "", merge)
}

func CoalesceChartDeps(chrt *chart.Chart, dest map[string]interface{}, merge bool) (map[string]interface{}, error) {
	return coalesceDeps(log.Printf, chrt, dest, "", merge)
}
