package phases

import (
	"github.com/werf/3p-helm-for-werf-helm/pkg/phases/stages"
)

type NoExternalDepsGenerator struct{}

func (g *NoExternalDepsGenerator) Generate(_ stages.SortedStageList) error {
	return nil
}
