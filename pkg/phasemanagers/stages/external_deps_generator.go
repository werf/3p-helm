package stages

type ExternalDepsGenerator interface {
	Generate(stages SortedStageList) error
}

type NoExternalDepsGenerator struct{}

func (g *NoExternalDepsGenerator) Generate(_ SortedStageList) error {
	return nil
}
