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

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/bank-vaults/secret-init/pkg/common"
	"github.com/bank-vaults/secret-init/pkg/provider"
	"github.com/bank-vaults/secret-init/pkg/provider/bao"
	"github.com/bank-vaults/secret-init/pkg/provider/file"
	"github.com/bank-vaults/secret-init/pkg/provider/vault"
)

var supportedProviders = []string{
	file.ProviderName,
	vault.ProviderName,
	bao.ProviderName,
}

// EnvStore is a helper for managing interactions between environment variables and providers,
// including tasks like extracting and converting provider-specific paths and secrets.
type EnvStore struct {
	data      map[string]string
	appConfig *common.Config
}

func NewEnvStore(appConfig *common.Config) *EnvStore {
	environ := make(map[string]string, len(os.Environ()))
	for _, env := range os.Environ() {
		split := strings.SplitN(env, "=", 2)
		name := split[0]
		value := split[1]
		environ[name] = value
	}

	return &EnvStore{
		data:      environ,
		appConfig: appConfig,
	}
}

// GetProviderPaths returns a map of secret paths for each provider
func (s *EnvStore) GetProviderPaths() map[string][]string {
	providerPaths := make(map[string][]string)

	for envKey, path := range s.data {
		providerName, path := getProviderPath(path)
		switch providerName {
		case file.ProviderName:
			providerPaths[file.ProviderName] = append(providerPaths[file.ProviderName], path)

		case vault.ProviderName:
			// The injector function expects a map of key:value pairs
			path = envKey + "=" + path
			providerPaths[vault.ProviderName] = append(providerPaths[vault.ProviderName], path)

		case bao.ProviderName:
			// The injector function expects a map of key:value pairs
			path = envKey + "=" + path
			providerPaths[bao.ProviderName] = append(providerPaths[bao.ProviderName], path)
		}
	}

	return providerPaths
}

// LoadProviderSecrets creates a new provider for each detected provider using a specified config.
// It then asynchronously loads secrets using each provider and it's corresponding paths.
// The secrets from each provider are then placed into a map with the provider name as the key.
func (s *EnvStore) LoadProviderSecrets(providerPaths map[string][]string) (map[string][]provider.Secret, error) {
	// At most, we will have one error per provider
	errCh := make(chan error, len(supportedProviders))
	providerSecrets := make(map[string][]provider.Secret)

	// Workaround for openBao
	// Remove once openBao uses BAO_ADDR in their client, instead of VAULT_ADDR
	vaultPaths, ok := providerPaths[vault.ProviderName]
	if ok {
		var err error
		providerSecrets[vault.ProviderName], err = s.workaroundForBao(vaultPaths)
		if err != nil {
			return nil, fmt.Errorf("failed to workaround for bao: %w", err)
		}

		// Remove the vault paths since they have been processed
		delete(providerPaths, vault.ProviderName)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for providerName, paths := range providerPaths {
		wg.Add(1)

		go func(providerName string, paths []string, errCh chan<- error) {
			defer wg.Done()

			provider, err := newProvider(providerName, s.appConfig)
			if err != nil {
				errCh <- fmt.Errorf("failed to create provider %s: %w", providerName, err)
				return
			}

			secrets, err := provider.LoadSecrets(context.Background(), paths)
			if err != nil {
				errCh <- fmt.Errorf("failed to load secrets for provider %s: %w", providerName, err)
				return
			}

			mu.Lock()
			providerSecrets[providerName] = secrets
			mu.Unlock()
		}(providerName, paths, errCh)
	}

	// Wait for all providers to finish
	wg.Wait()
	close(errCh)

	// Check for errors
	var errs error
	for e := range errCh {
		if e != nil {
			errs = errors.Join(errs, e)
		}
	}
	if errs != nil {
		return nil, errs
	}

	return providerSecrets, nil
}

// Workaround for openBao, essentially loading secretes from Vault first.
func (s *EnvStore) workaroundForBao(vaultPaths []string) ([]provider.Secret, error) {
	var secrets []provider.Secret

	provider, err := newProvider(vault.ProviderName, s.appConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", vault.ProviderName, err)
	}

	secrets, err = provider.LoadSecrets(context.Background(), vaultPaths)
	if err != nil {
		return nil, fmt.Errorf("failed to load secrets for provider %s: %w", vault.ProviderName, err)
	}

	return secrets, nil
}

// ConvertProviderSecrets converts the loaded secrets to environment variables
func (s *EnvStore) ConvertProviderSecrets(providerSecrets map[string][]provider.Secret) ([]string, error) {
	var secretsEnv []string

	for providerName, secrets := range providerSecrets {
		switch providerName {
		case vault.ProviderName, bao.ProviderName:
			// The Vault and Bao providers already returns the secrets with the environment variable keys
			for _, secret := range secrets {
				secretsEnv = append(secretsEnv, fmt.Sprintf("%s=%s", secret.Path, secret.Value))
			}

		default:
			secrets, err := createSecretEnvsFrom(s.data, secrets)
			if err != nil {
				return nil, fmt.Errorf("failed to create secret environment variables: %w", err)
			}

			secretsEnv = append(secretsEnv, secrets...)
		}
	}

	return secretsEnv, nil
}

// Returns the detected provider name and path with removed prefix
func getProviderPath(path string) (string, string) {
	if strings.HasPrefix(path, "file:") {
		var fileProviderName = file.ProviderName
		return fileProviderName, strings.TrimPrefix(path, "file:")
	}

	// If the path contains some string formatted as "vault:{STR}#{STR}"
	// it is most probably a vault path
	if vault.ProviderEnvRegex.MatchString(path) {
		// Do not remove the prefix since it will be processed during injection
		return vault.ProviderName, path
	}

	// If the path contains some string formatted as "bao:{STR}#{STR}"
	// it is most probably a vault path
	if bao.ProviderEnvRegex.MatchString(path) {
		// Do not remove the prefix since it will be processed during injection
		return bao.ProviderName, path
	}

	return "", path
}

func newProvider(providerName string, appConfig *common.Config) (provider.Provider, error) {
	switch providerName {
	case file.ProviderName:
		config := file.LoadConfig()
		provider, err := file.NewProvider(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create file provider: %w", err)
		}
		return provider, nil

	case vault.ProviderName:
		config, err := vault.LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create vault config: %w", err)
		}

		provider, err := vault.NewProvider(config, appConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create vault provider: %w", err)
		}
		return provider, nil

	case bao.ProviderName:
		config, err := bao.LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create bao config: %w", err)
		}

		provider, err := bao.NewProvider(config, appConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create bao provider: %w", err)
		}
		return provider, nil

	default:
		return nil, fmt.Errorf("provider %s is not supported", providerName)
	}
}

func createSecretEnvsFrom(envs map[string]string, secrets []provider.Secret) ([]string, error) {
	// Reverse the map so we can match
	// the environment variable key to the secret
	// by using the secret path
	reversedEnvs := make(map[string]string)
	for envKey, path := range envs {
		providerName, path := getProviderPath(path)
		if providerName != "" {
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

		secretsEnv = append(secretsEnv, fmt.Sprintf("%s=%s", key, secret.Value))
	}

	return secretsEnv, nil
}
