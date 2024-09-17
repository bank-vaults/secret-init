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
	"github.com/bank-vaults/secret-init/pkg/provider/aws"
	"github.com/bank-vaults/secret-init/pkg/provider/azure"
	"github.com/bank-vaults/secret-init/pkg/provider/bao"
	"github.com/bank-vaults/secret-init/pkg/provider/file"
	"github.com/bank-vaults/secret-init/pkg/provider/gcp"
	"github.com/bank-vaults/secret-init/pkg/provider/vault"
)

var factories = []provider.Factory{
	{
		ProviderType: file.ProviderType,
		Validator:    file.Valid,
		Create:       file.NewProvider,
	},
	{
		ProviderType: vault.ProviderType,
		Validator:    vault.Valid,
		Create:       vault.NewProvider,
	},
	{
		ProviderType: bao.ProviderType,
		Validator:    bao.Valid,
		Create:       bao.NewProvider,
	},
	{
		ProviderType: aws.ProviderType,
		Validator:    aws.Valid,
		Create:       aws.NewProvider,
	},
	{
		ProviderType: gcp.ProviderType,
		Validator:    gcp.Valid,
		Create:       gcp.NewProvider,
	},
	{
		ProviderType: azure.ProviderType,
		Validator:    azure.Valid,
		Create:       azure.NewProvider,
	},
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

// GetSecretReferences returns a map of secret key=value pairs for each provider
func (s *EnvStore) GetSecretReferences() map[string][]string {
	secretReferences := make(map[string][]string)
	for envKey, envPath := range s.data {
		for _, factory := range factories {
			if factory.Validator(envPath) {
				secretReferences[factory.ProviderType] = append(secretReferences[factory.ProviderType], fmt.Sprintf("%s=%s", envKey, envPath))
			}
		}
	}

	return secretReferences
}

// LoadProviderSecrets creates a new provider for each detected provider using a specified config.
// It then asynchronously loads secrets using each provider and it's corresponding paths.
// The secrets from each provider are then placed into a single slice.
func (s *EnvStore) LoadProviderSecrets(ctx context.Context, providerPaths map[string][]string) ([]provider.Secret, error) {
	var providerSecrets []provider.Secret
	// Workaround for openBao
	// Remove once openBao uses BAO_ADDR in their client, instead of VAULT_ADDR
	if _, ok := providerPaths[vault.ProviderType]; ok {
		vaultSecrets, err := s.workaroundForBao(ctx, providerPaths[vault.ProviderType])
		if err != nil {
			return nil, err
		}

		providerSecrets = append(providerSecrets, vaultSecrets...)
		delete(providerPaths, vault.ProviderType)
	}

	// At most, we will have one error per provider
	errCh := make(chan error, len(factories))
	var wg sync.WaitGroup
	var mu sync.Mutex
	for providerName, paths := range providerPaths {
		wg.Add(1)
		go func(providerName string, paths []string, errCh chan<- error) {
			defer wg.Done()

			for _, factory := range factories {
				if factory.ProviderType == providerName {
					provider, err := factory.Create(ctx, s.appConfig)
					if err != nil {
						errCh <- fmt.Errorf("failed to create provider %s: %w", providerName, err)
						return
					}

					secrets, err := provider.LoadSecrets(ctx, paths)
					if err != nil {
						errCh <- fmt.Errorf("failed to load secrets for provider %s: %w", providerName, err)
						return
					}

					mu.Lock()
					providerSecrets = append(providerSecrets, secrets...)
					mu.Unlock()
				}
			}
		}(providerName, paths, errCh)
	}
	wg.Wait()
	close(errCh)

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
func (s *EnvStore) workaroundForBao(ctx context.Context, vaultPaths []string) ([]provider.Secret, error) {
	var providerSecrets []provider.Secret
	for _, factory := range factories {
		if factory.ProviderType == vault.ProviderType {
			provider, err := factory.Create(ctx, s.appConfig)
			if err != nil {
				return nil, fmt.Errorf("failed to create provider %s: %w", factory.ProviderType, err)
			}

			secrets, err := provider.LoadSecrets(ctx, vaultPaths)
			if err != nil {
				return nil, fmt.Errorf("failed to load secrets for provider %s: %w", factory.ProviderType, err)
			}

			providerSecrets = append(providerSecrets, secrets...)
			break
		}
	}

	return providerSecrets, nil
}

// ConvertProviderSecrets converts the loaded secrets to environment variables
func (s *EnvStore) ConvertProviderSecrets(providerSecrets []provider.Secret) []string {
	var secretsEnv []string

	for _, secret := range providerSecrets {
		secretsEnv = append(secretsEnv, fmt.Sprintf("%s=%s", secret.Key, secret.Value))
	}

	return secretsEnv
}
