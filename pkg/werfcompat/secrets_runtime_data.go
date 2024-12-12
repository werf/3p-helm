package werfcompat

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"

	"github.com/werf/3p-helm/pkg/chart/loader"
	"github.com/werf/3p-helm/pkg/werfcompat/secret"
	"helm.sh/helm/v3/pkg/werfcompat/secrets_manager"
	"sigs.k8s.io/yaml"
)

// FIXME(ilya-lesikov): remote all of this
type SecretsRuntimeData struct {
	DecryptedSecretValues    map[string]interface{}
	DecryptedSecretFilesData map[string]string
	SecretValuesToMask       []string
}

func NewSecretsRuntimeData() *SecretsRuntimeData {
	return &SecretsRuntimeData{
		DecryptedSecretFilesData: make(map[string]string),
	}
}

type DecodeAndLoadSecretsOptions struct {
	CustomSecretValueFiles     []string
	LoadFromLocalFilesystem    bool
	WithoutDefaultSecretValues bool
}

func (secretsRuntimeData *SecretsRuntimeData) DecodeAndLoadSecrets(
	ctx context.Context,
	loadedChartFiles []*loader.BufferedFile,
	chartDir, secretsWorkingDir string,
	secretsManager *secrets_manager.SecretsManager,
	opts DecodeAndLoadSecretsOptions,
) error {
	secretDirFiles := GetSecretDirFiles(loadedChartFiles)

	var loadedSecretValuesFiles []*loader.BufferedFile

	if !opts.WithoutDefaultSecretValues {
		if defaultSecretValues := GetDefaultSecretValuesFile(loadedChartFiles); defaultSecretValues != nil {
			loadedSecretValuesFiles = append(loadedSecretValuesFiles, defaultSecretValues)
		}
	}

	for _, customSecretValuesFileName := range opts.CustomSecretValueFiles {
		file := &loader.BufferedFile{Name: customSecretValuesFileName}

		if opts.LoadFromLocalFilesystem {
			data, err := ioutil.ReadFile(customSecretValuesFileName)
			if err != nil {
				return fmt.Errorf("unable to read custom secret values file %q from local filesystem: %w", customSecretValuesFileName, err)
			}
			file.Data = data
		} else {
			data, err := Reader.ReadChartFile(ctx, customSecretValuesFileName)
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
			return fmt.Errorf("error getting Secrets yaml encoder: %w", err)
		} else {
			encoder = enc
		}
	}

	if len(secretDirFiles) > 0 {
		if data, err := LoadChartSecretDirFilesData(chartDir, secretDirFiles, encoder); err != nil {
			return fmt.Errorf("error loading secret files data: %w", err)
		} else {
			secretsRuntimeData.DecryptedSecretFilesData = data
			for _, fileData := range secretsRuntimeData.DecryptedSecretFilesData {
				secretsRuntimeData.SecretValuesToMask = append(secretsRuntimeData.SecretValuesToMask, fileData)
			}
		}
	}

	if len(loadedSecretValuesFiles) > 0 {
		if values, err := LoadChartSecretValueFiles(chartDir, loadedSecretValuesFiles, encoder); err != nil {
			return fmt.Errorf("error loading secret value files: %w", err)
		} else {
			secretsRuntimeData.DecryptedSecretValues = values
			secretsRuntimeData.SecretValuesToMask = append(secretsRuntimeData.SecretValuesToMask, extractSecretValuesFromMap(values)...)
		}
	}

	return nil
}

func (secretsRuntimeData *SecretsRuntimeData) GetEncodedSecretValues(
	ctx context.Context,
	secretsManager *secrets_manager.SecretsManager,
	secretsWorkingDir string,
) (map[string]interface{}, error) {
	if len(secretsRuntimeData.DecryptedSecretValues) == 0 {
		return nil, nil
	}

	// FIXME: Secrets encoder should receive interface{} raw data instead of []byte yaml data

	var encoder *secret.YamlEncoder
	if enc, err := secretsManager.GetYamlEncoder(ctx, secretsWorkingDir); err != nil {
		return nil, fmt.Errorf("error getting Secrets yaml encoder: %w", err)
	} else {
		encoder = enc
	}

	decryptedSecretsData, err := yaml.Marshal(secretsRuntimeData.DecryptedSecretValues)
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

func extractSecretValuesFromMap(data map[string]interface{}) []string {
	queue := []interface{}{data}
	maskedValues := []string{}

	for len(queue) > 0 {
		var elemI interface{}
		elemI, queue = queue[0], queue[1:]

		elemType := reflect.TypeOf(elemI)
		if elemType == nil {
			continue
		}

		switch elemType.Kind() {
		case reflect.Slice, reflect.Array:
			elem := reflect.ValueOf(elemI)
			for i := 0; i < elem.Len(); i++ {
				value := elem.Index(i)
				queue = append(queue, value.Interface())
			}
		case reflect.Map:
			elem := reflect.ValueOf(elemI)
			for _, key := range elem.MapKeys() {
				value := elem.MapIndex(key)
				queue = append(queue, value.Interface())
			}
		default:
			elemStr := fmt.Sprintf("%v", elemI)
			if len(elemStr) >= 4 {
				maskedValues = append(maskedValues, elemStr)
			}
			for _, line := range strings.Split(elemStr, "\n") {
				trimmedLine := strings.TrimSpace(line)
				if len(trimmedLine) >= 4 {
					maskedValues = append(maskedValues, trimmedLine)
				}
			}

			dataMap := map[string]interface{}{}
			if err := json.Unmarshal([]byte(elemStr), &dataMap); err == nil {
				for _, v := range dataMap {
					queue = append(queue, v)
				}
			}

			dataArr := []interface{}{}
			if err := json.Unmarshal([]byte(elemStr), &dataArr); err == nil {
				queue = append(queue, dataArr...)
			}
		}
	}

	return maskedValues
}
