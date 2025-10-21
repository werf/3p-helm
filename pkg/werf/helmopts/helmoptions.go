package helmopts

type HelmOptions struct {
	ChartLoadOpts ChartLoadOptions
}

type ChartLoadOptions struct {
	ChartAppVersion            string
	ChartType                  ChartType
	DefaultChartAPIVersion     string
	DefaultChartName           string
	DefaultChartVersion        string
	DefaultSecretValuesDisable bool
	DefaultValuesDisable       bool
	DepDownloader              DepDownloader
	ExtraValues                map[string]interface{}
	NoDecryptSecrets           bool
	NoSecrets                  bool
	SecretValuesFiles          []string
	SecretsWorkingDir          string
}

type ChartType string

const (
	ChartTypeChart     ChartType = ""
	ChartTypeBundle    ChartType = "bundle"
	ChartTypeSubchart  ChartType = "subchart"
	ChartTypeChartStub ChartType = "chartstub"
)

type DepDownloader interface {
	Build(opts HelmOptions) error
	Update(opts HelmOptions) error
	UpdateRepositories() error
	SetChartPath(path string)
}
