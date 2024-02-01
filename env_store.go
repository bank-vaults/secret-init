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

	"github.com/bank-vaults/secret-init/provider"
	"github.com/bank-vaults/secret-init/provider/file"
	"github.com/bank-vaults/secret-init/provider/vault"
)

var supportedProviders = []string{
	file.ProviderName,
	vault.ProviderName,
}

// EnvStore is a helper for managing interactions between environment variables and providers,
// including tasks like extracting and converting provider-specific paths and secrets.
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

// GetProviderPaths returns a map of secret paths for each provider
func (s *EnvStore) GetProviderPaths() map[string][]string {
	providerPaths := make(map[string][]string)

	for envKey, path := range s.data {
		providerName, path := getProviderPath(path)
		switch providerName {
		case file.ProviderName:
			_, ok := providerPaths[file.ProviderName]
			if !ok {
				providerPaths[file.ProviderName] = []string{}
			}

			providerPaths[file.ProviderName] = append(providerPaths[file.ProviderName], path)

		case vault.ProviderName:
			_, ok := providerPaths[vault.ProviderName]
			if !ok {
				providerPaths[vault.ProviderName] = []string{}
			}

			// The injector function expects a map of key:value pairs
			path = envKey + "=" + path
			providerPaths[vault.ProviderName] = append(providerPaths[vault.ProviderName], path)
		}
	}

	return providerPaths
}

// GetProviderSecrets creates a new provider for each detected provider using a specified config.
// It then asynchronously loads secrets using each provider and it's corresponding paths.
// The secrets from each provider are then placed into a map with the provider name as the key.
func (s *EnvStore) GetProviderSecrets(providerPaths map[string][]string) (map[string][]provider.Secret, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	// At most, we will have one error per provider
	errCh := make(chan error, len(supportedProviders))
	providerSecrets := make(map[string][]provider.Secret)

	for providerName, paths := range providerPaths {
		providerSecrets[providerName] = []provider.Secret{}
		wg.Add(1)

		go func(providerName string, paths []string, errCh chan<- error) {
			defer wg.Done()

			provider, err := newProvider(providerName)
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

// ConvertProviderSecrets converts the loaded secrets to environment variables
// In case of the Vault provider, the secrets are already in the correct format
func (s *EnvStore) ConvertProviderSecrets(providerSecrets map[string][]provider.Secret) ([]string, error) {
	var secretsEnv []string

	for providerName, secrets := range providerSecrets {
		switch providerName {
		case vault.ProviderName:
			// The Vault provider already returns the secrets with the environment variable keys
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
	if strings.HasPrefix(path, "vault:") {
		var vaultProviderName = vault.ProviderName
		// Do not remove the prefix since it will be processed during injection
		return vaultProviderName, path
	}

	return "", path
}

func newProvider(providerName string) (provider.Provider, error) {
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

		provider, err := vault.NewProvider(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create vault provider: %w", err)
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
