package runtimedata

import (
	"context"

	"github.com/werf/3p-helm/pkg/werf/file"
	"github.com/werf/common-go/pkg/secrets_manager"
)

type RuntimeData interface {
	DecodeAndLoadSecrets(ctx context.Context, loadedChartFiles []*file.ChartExtenderBufferedFile, noSecretsWorkingDir bool, secretsManager *secrets_manager.SecretsManager, opts DecodeAndLoadSecretsOptions) error
	GetEncodedSecretValues(ctx context.Context, secretsManager *secrets_manager.SecretsManager, noSecretsWorkingDir bool) (map[string]interface{}, error)
	GetDecryptedSecretValues() map[string]interface{}
	GetDecryptedSecretFilesData() map[string]string
	GetSecretValuesToMask() []string
}

type DecodeAndLoadSecretsOptions struct {
	ChartFileReader            file.ChartFileReader
	CustomSecretValueFiles     []string
	LoadFromLocalFilesystem    bool
	WithoutDefaultSecretValues bool
}
