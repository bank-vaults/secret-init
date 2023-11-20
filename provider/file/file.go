// Copyright © 2023 Bank-Vaults Maintainers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package file

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/bank-vaults/secret-init/provider"
)

const ProviderName = "file"

type Provider struct {
	secretsFilePath string
}

func NewFileProvider(secretsFilePath string) (provider.Provider, error) {

	return &Provider{secretsFilePath: secretsFilePath}, nil
}

func (provider *Provider) LoadSecrets(_ context.Context, envs map[string]string) ([]string, error) {
	// extract secrets from the file to a map
	secretsMap, err := provider.getSecretsFromFile()
	if err != nil {

		return nil, fmt.Errorf("failed to load secrets: %w", err)
	}

	var secrets []string
	for key, value := range envs {
		if strings.HasPrefix(value, "file:") {
			// Check if the requested secret is in the loaded secret map
			value = strings.TrimPrefix(value, "file:")
			secret, ok := secretsMap[value]
			if !ok {

				return nil, fmt.Errorf("secret %s not found", key)
			}
			secrets = append(secrets, fmt.Sprintf("%s=%s", key, secret))
		}
	}

	return secrets, nil
}

func (provider *Provider) getSecretsFromFile() (map[string]string, error) {
	data, err := os.ReadFile(provider.secretsFilePath)
	if err != nil {

		return nil, fmt.Errorf("failed to read secrets file: %w", err)
	}

	secretsMap := make(map[string]string)
	err = yaml.Unmarshal(data, &secretsMap)
	if err != nil {

		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return secretsMap, nil
}
