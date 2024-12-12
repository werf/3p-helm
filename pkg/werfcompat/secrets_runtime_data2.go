package werfcompat

import (
	"context"
	"fmt"
	"os"

	"github.com/samber/lo"
	"github.com/werf/3p-helm/pkg/chart/loader"
	"github.com/werf/3p-helm/pkg/werfcompat/secret"
	"github.com/werf/3p-helm/pkg/werfcompat/secrets_manager"
	"sigs.k8s.io/yaml"
)

type Secrets struct {
	DecryptedSecretValues    map[string]interface{}
	DecryptedSecretFilesData map[string]string
	SecretValuesToMask       []string

	chartDir                   string
	secretsWorkingDir          string
	secretsManager             *secrets_manager.SecretsManager
	customSecretValueFiles     []string
	withoutDefaultSecretValues bool
}

func NewSecrets(chartDir string, secretsManager *secrets_manager.SecretsManager, options SecretsOptions) *Secrets {
	var secretsWorkDir string
	if options.SecretsWorkDir != "" {
		secretsWorkDir = options.SecretsWorkDir
	} else {
		secretsWorkDir = lo.Must(os.Getwd())
	}

	return &Secrets{
		DecryptedSecretFilesData:   make(map[string]string),
		chartDir:                   chartDir,
		secretsWorkingDir:          secretsWorkDir,
		secretsManager:             secretsManager,
		customSecretValueFiles:     options.CustomSecretValueFiles,
		withoutDefaultSecretValues: options.WithoutDefaultSecretValues,
	}
}

type SecretsOptions struct {
	CustomSecretValueFiles     []string
	WithoutDefaultSecretValues bool
	SecretsWorkDir             string
}

func (s *Secrets) DecodeAndLoad(
	ctx context.Context,
	loadedChartFiles []*loader.BufferedFile,
) error {
	secretDirFiles := GetSecretDirFiles(loadedChartFiles)

	var loadedSecretValuesFiles []*loader.BufferedFile

	if !s.withoutDefaultSecretValues {
		if defaultSecretValues := GetDefaultSecretValuesFile(loadedChartFiles); defaultSecretValues != nil {
			loadedSecretValuesFiles = append(loadedSecretValuesFiles, defaultSecretValues)
		}
	}

	for _, customSecretValuesFileName := range s.customSecretValueFiles {
		file := &loader.BufferedFile{Name: customSecretValuesFileName}

		data, err := Reader.ReadChartFile(ctx, customSecretValuesFileName)
		if err != nil {
			return fmt.Errorf("unable to read custom secret values file %q: %w", customSecretValuesFileName, err)
		}
		file.Data = data

		loadedSecretValuesFiles = append(loadedSecretValuesFiles, file)
	}

	var encoder *secret.YamlEncoder
	if len(secretDirFiles)+len(loadedSecretValuesFiles) > 0 {
		if enc, err := s.secretsManager.GetYamlEncoder(ctx, s.secretsWorkingDir); err != nil {
			return fmt.Errorf("error getting Secrets yaml encoder: %w", err)
		} else {
			encoder = enc
		}
	}

	if len(secretDirFiles) > 0 {
		if data, err := LoadChartSecretDirFilesData(s.chartDir, secretDirFiles, encoder); err != nil {
			return fmt.Errorf("error loading secret files data: %w", err)
		} else {
			s.DecryptedSecretFilesData = data
			for _, fileData := range s.DecryptedSecretFilesData {
				s.SecretValuesToMask = append(s.SecretValuesToMask, fileData)
			}
		}
	}

	if len(loadedSecretValuesFiles) > 0 {
		if values, err := LoadChartSecretValueFiles(s.chartDir, loadedSecretValuesFiles, encoder); err != nil {
			return fmt.Errorf("error loading secret value files: %w", err)
		} else {
			s.DecryptedSecretValues = values
			s.SecretValuesToMask = append(s.SecretValuesToMask, extractSecretValuesFromMap(values)...)
		}
	}

	return nil
}

func (s *Secrets) GetEncodedSecretValues(
	ctx context.Context,
) (map[string]interface{}, error) {
	if len(s.DecryptedSecretValues) == 0 {
		return nil, nil
	}

	// FIXME: Secrets encoder should receive interface{} raw data instead of []byte yaml data

	var encoder *secret.YamlEncoder
	if enc, err := s.secretsManager.GetYamlEncoder(ctx, s.secretsWorkingDir); err != nil {
		return nil, fmt.Errorf("error getting Secrets yaml encoder: %w", err)
	} else {
		encoder = enc
	}

	decryptedSecretsData, err := yaml.Marshal(s.DecryptedSecretValues)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal decrypted Secrets yaml: %w", err)
	}

	encryptedSecretsData, err := encoder.EncryptYamlData(decryptedSecretsData)
	if err != nil {
		return nil, fmt.Errorf("unable to encrypt Secrets data: %w", err)
	}

	var encryptedData map[string]interface{}
	if err := yaml.Unmarshal(encryptedSecretsData, &encryptedData); err != nil {
		return nil, fmt.Errorf("unable to unmarshal encrypted Secrets data: %w", err)
	}

	return encryptedData, nil
}
