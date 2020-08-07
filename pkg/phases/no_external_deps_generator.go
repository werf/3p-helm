package phases

import (
	"helm.sh/helm/v3/pkg/phases/stages"
)

type NoExternalDepsGenerator struct{}

func (g *NoExternalDepsGenerator) Generate(_ stages.SortedStageList) error {
	return nil
}
