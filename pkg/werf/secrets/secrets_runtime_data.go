package secrets

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"sigs.k8s.io/yaml"

	"github.com/werf/3p-helm/pkg/werf/file"
	"github.com/werf/3p-helm/pkg/werf/secrets/runtimedata"
	"github.com/werf/common-go/pkg/secret"
	"github.com/werf/common-go/pkg/secrets_manager"
	"github.com/werf/common-go/pkg/secretvalues"
)

var _ runtimedata.RuntimeData = (*SecretsRuntimeData)(nil)

var CoalesceTablesFunc func(dst, src map[string]interface{}) map[string]interface{}

type SecretsRuntimeData struct {
	decryptedSecretValues    map[string]interface{}
	decryptedSecretFilesData map[string]string
	secretValuesToMask       []string
}

func NewSecretsRuntimeData() *SecretsRuntimeData {
	return &SecretsRuntimeData{
		decryptedSecretFilesData: make(map[string]string),
	}
}

func (secretsRuntimeData *SecretsRuntimeData) DecodeAndLoadSecrets(
	ctx context.Context,
	loadedChartFiles []*file.ChartExtenderBufferedFile,
	chartDir, secretsWorkingDir string,
	secretsManager *secrets_manager.SecretsManager,
	opts runtimedata.DecodeAndLoadSecretsOptions,
) error {
	secretDirFiles := GetSecretDirFiles(loadedChartFiles)

	var loadedSecretValuesFiles []*file.ChartExtenderBufferedFile

	if !opts.WithoutDefaultSecretValues {
		if defaultSecretValues := GetDefaultSecretValuesFile(loadedChartFiles); defaultSecretValues != nil {
			loadedSecretValuesFiles = append(loadedSecretValuesFiles, defaultSecretValues)
		}
	}

	for _, customSecretValuesFileName := range opts.CustomSecretValueFiles {
		file := &file.ChartExtenderBufferedFile{Name: customSecretValuesFileName}

		if opts.LoadFromLocalFilesystem {
			data, err := ioutil.ReadFile(customSecretValuesFileName)
			if err != nil {
				return fmt.Errorf("unable to read custom secret values file %q from local filesystem: %w", customSecretValuesFileName, err)
			}
			file.Data = data
		} else {
			data, err := opts.ChartFileReader.ReadChartFile(ctx, customSecretValuesFileName)
			if err != nil {
				return fmt.Errorf("unable to read custom secret values file %q: %w", customSecretValuesFileName, err)
			}
			file.Data = data
		}

		loadedSecretValuesFiles = append(loadedSecretValuesFiles, file)
	}

	var encoder *secret.YamlEncoder
	if len(secretDirFiles)+len(loadedSecretValuesFiles) > 0 {
		if enc, err := secretsManager.GetYamlEncoder(ctx, secretsWorkingDir); err != nil {
			return fmt.Errorf("error getting secrets yaml encoder: %w", err)
		} else {
			encoder = enc
		}
	}

	if len(secretDirFiles) > 0 {
		if data, err := LoadChartSecretDirFilesData(chartDir, secretDirFiles, encoder); err != nil {
			return fmt.Errorf("error loading secret files data: %w", err)
		} else {
			secretsRuntimeData.decryptedSecretFilesData = data
			for _, fileData := range secretsRuntimeData.decryptedSecretFilesData {
				secretsRuntimeData.secretValuesToMask = append(secretsRuntimeData.secretValuesToMask, fileData)
			}
		}
	}

	if len(loadedSecretValuesFiles) > 0 {
		if values, err := LoadChartSecretValueFiles(chartDir, loadedSecretValuesFiles, encoder); err != nil {
			return fmt.Errorf("error loading secret value files: %w", err)
		} else {
			secretsRuntimeData.decryptedSecretValues = values
			secretsRuntimeData.secretValuesToMask = append(secretsRuntimeData.secretValuesToMask, secretvalues.ExtractSecretValuesFromMap(values)...)
		}
	}

	return nil
}

func (secretsRuntimeData *SecretsRuntimeData) GetEncodedSecretValues(
	ctx context.Context,
	secretsManager *secrets_manager.SecretsManager,
	secretsWorkingDir string,
) (map[string]interface{}, error) {
	if len(secretsRuntimeData.decryptedSecretValues) == 0 {
		return nil, nil
	}

	// FIXME: secrets encoder should receive interface{} raw data instead of []byte yaml data

	var encoder *secret.YamlEncoder
	if enc, err := secretsManager.GetYamlEncoder(ctx, secretsWorkingDir); err != nil {
		return nil, fmt.Errorf("error getting secrets yaml encoder: %w", err)
	} else {
		encoder = enc
	}

	decryptedSecretsData, err := yaml.Marshal(secretsRuntimeData.decryptedSecretValues)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal decrypted secrets yaml: %w", err)
	}

	encryptedSecretsData, err := encoder.EncryptYamlData(decryptedSecretsData)
	if err != nil {
		return nil, fmt.Errorf("unable to encrypt secrets data: %w", err)
	}

	var encryptedData map[string]interface{}
	if err := yaml.Unmarshal(encryptedSecretsData, &encryptedData); err != nil {
		return nil, fmt.Errorf("unable to unmarshal encrypted secrets data: %w", err)
	}

	return encryptedData, nil
}

func (secretsRuntimeData *SecretsRuntimeData) GetDecryptedSecretValues() map[string]interface{} {
	return secretsRuntimeData.decryptedSecretValues
}

func (secretsRuntimeData *SecretsRuntimeData) GetDecryptedSecretFilesData() map[string]string {
	return secretsRuntimeData.decryptedSecretFilesData
}

func (secretsRuntimeData *SecretsRuntimeData) GetSecretValuesToMask() []string {
	return secretsRuntimeData.secretValuesToMask
}

func LoadChartSecretValueFiles(
	chartDir string,
	secretDirFiles []*file.ChartExtenderBufferedFile,
	encoder *secret.YamlEncoder,
) (map[string]interface{}, error) {
	var res map[string]interface{}

	for _, file := range secretDirFiles {
		decodedData, err := encoder.DecryptYamlData(file.Data)
		if err != nil {
			return nil, fmt.Errorf("cannot decode file %q secret data: %w", filepath.Join(chartDir, file.Name), err)
		}

		rawValues := map[string]interface{}{}
		if err := yaml.Unmarshal(decodedData, &rawValues); err != nil {
			return nil, fmt.Errorf("cannot unmarshal secret values file %s: %w", filepath.Join(chartDir, file.Name), err)
		}

		res = CoalesceTablesFunc(rawValues, res)
	}

	return res, nil
}
