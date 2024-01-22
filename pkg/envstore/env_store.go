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

package envstore

import (
	"fmt"
	"os"
	"strings"

	"github.com/bank-vaults/secret-init/provider"
	"github.com/bank-vaults/secret-init/provider/file"
	"github.com/bank-vaults/secret-init/provider/vault"
)

type EnvStore struct {
	data map[string]string
}

func NewEnvStore() *EnvStore {
	environ := make(map[string]string, len(os.Environ()))
	for _, env := range os.Environ() {
		split := strings.SplitN(env, "=", 2)
		name := split[0]
		value := split[1]
		environ[name] = value
	}

	return &EnvStore{
		data: environ,
	}
}

func (s *EnvStore) GetPathsFor(provider provider.Provider) ([]string, error) {
	var secretPaths []string

	for envKey, path := range s.data {
		p, path := getProviderPath(path)

		// TODO(csatib02): Implement multi-provider support
		if p == provider.GetProviderName() {
			// The injector function expects a map of key:value pairs
			if p == vault.ProviderName {
				path = envKey + "=" + path
			}

			secretPaths = append(secretPaths, path)
		}
	}

	return secretPaths, nil
}

func (s *EnvStore) GetProviderSecrets(provider provider.Provider, secrets []provider.Secret) ([]string, error) {
	switch provider.GetProviderName() {
	case vault.ProviderName:
		// The Vault provider already returns the secrets with the environment variable keys
		var vaultEnv []string
		for _, secret := range secrets {
			vaultEnv = append(vaultEnv, secret.Format())
		}
		return vaultEnv, nil

	default:
		return createSecretEnvsFrom(s.data, secrets)
	}
}

// Returns the detected provider name and path with removed prefix
func getProviderPath(path string) (string, string) {
	if strings.HasPrefix(path, "file:") {
		var fileProviderName = file.ProviderName
		return fileProviderName, strings.TrimPrefix(path, "file:")
	}
	if strings.HasPrefix(path, "vault:") {
		var vaultProviderName = vault.ProviderName
		// Do not remove the prefix since it will be processed during injection
		return vaultProviderName, path
	}

	return "", path
}

func createSecretEnvsFrom(envs map[string]string, secrets []provider.Secret) ([]string, error) {
	// Reverse the map so we can match
	// the environment variable key to the secret
	// by using the secret path
	reversedEnvs := make(map[string]string)
	for envKey, path := range envs {
		p, path := getProviderPath(path)
		if p != "" {
			reversedEnvs[path] = envKey
		}
	}

	var secretsEnv []string
	for _, secret := range secrets {
		path := secret.Path
		key, ok := reversedEnvs[path]
		if !ok {
			return nil, fmt.Errorf("failed to find environment variable key for secret path: %s", path)
		}
		secret.Path = key

		secretsEnv = append(secretsEnv, secret.Format())
	}

	return secretsEnv, nil
}
