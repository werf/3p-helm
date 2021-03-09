package chartutil

import (
	"log"

	"helm.sh/helm/v3/pkg/chart"
)

func CoalesceChartValues(c *chart.Chart, v map[string]interface{}) {
	coalesceValues(log.Printf, c, v, "")
}

func CoalesceChartDeps(chrt *chart.Chart, dest map[string]interface{}) (map[string]interface{}, error) {
	return coalesceDeps(log.Printf, chrt, dest, "")
}
