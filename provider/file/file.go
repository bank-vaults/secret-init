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

	"github.com/bank-vaults/secret-init/provider"
)

type Provider struct {
	SecretsFilePath string
	SecretData      []byte
}

func NewFileProvider(secretsFilePath string) (provider.Provider, error) {
	data, err := os.ReadFile(secretsFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read from file: %w", err)
	}

	return &Provider{SecretsFilePath: secretsFilePath, SecretData: data}, nil
}

func (provider *Provider) LoadSecrets(_ context.Context, envs map[string]string) ([]string, error) {
	// envs that has a "file:" prefix needs to be loaded
	var secrets []string
	for key, value := range envs {
		if strings.HasPrefix(value, "file:") {
			secret, err := provider.getSecretFromFile(key)
			if err != nil {
				return nil, fmt.Errorf("failed to load secret: %w", err)
			}
			secrets = append(secrets, fmt.Sprintf("%s=%s", key, secret))
		}
	}

	return secrets, nil
}

func (provider *Provider) getSecretFromFile(key string) (string, error) {
	lines := strings.Split(string(provider.SecretData), "\n")
	for _, line := range lines {
		split := strings.SplitN(line, "=", 2)
		if split[0] == key {
			return split[1], nil
		}
	}

	return "", fmt.Errorf("key: '%s' not found in file", key)
}
