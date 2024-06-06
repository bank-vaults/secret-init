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
	"github.com/bank-vaults/secret-init/pkg/provider/bitwarden"
	"github.com/bank-vaults/secret-init/pkg/provider/file"
	"github.com/bank-vaults/secret-init/pkg/provider/gcp"
	"github.com/bank-vaults/secret-init/pkg/provider/vault"
)

var supportedProviders = []string{
	file.ProviderName,
	vault.ProviderName,
	bao.ProviderName,
	aws.ProviderName,
	gcp.ProviderName,
	azure.ProviderName,
	bitwarden.ProviderName,
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
		providerName, envSecretReference := getProviderPath(envPath)
		envSecretReference = envKey + "=" + envSecretReference
		switch providerName {
		case file.ProviderName:
			secretReferences[file.ProviderName] = append(secretReferences[file.ProviderName], envSecretReference)

		case vault.ProviderName:
			secretReferences[vault.ProviderName] = append(secretReferences[vault.ProviderName], envSecretReference)

		case bao.ProviderName:
			secretReferences[bao.ProviderName] = append(secretReferences[bao.ProviderName], envSecretReference)

		case aws.ProviderName:
			secretReferences[aws.ProviderName] = append(secretReferences[aws.ProviderName], envSecretReference)

		case gcp.ProviderName:
			secretReferences[gcp.ProviderName] = append(secretReferences[gcp.ProviderName], envSecretReference)

		case azure.ProviderName:
			secretReferences[azure.ProviderName] = append(secretReferences[azure.ProviderName], envSecretReference)

		case bitwarden.ProviderName:
			secretReferences[bitwarden.ProviderName] = append(secretReferences[bitwarden.ProviderName], envSecretReference)
		}
	}

	return secretReferences
}

// LoadProviderSecrets creates a new provider for each detected provider using a specified config.
// It then asynchronously loads secrets using each provider and it's corresponding paths.
// The secrets from each provider are then placed into a single slice.
func (s *EnvStore) LoadProviderSecrets(ctx context.Context, providerPaths map[string][]string) ([]provider.Secret, error) {
	// At most, we will have one error per provider
	errCh := make(chan error, len(supportedProviders))
	var providerSecrets []provider.Secret

	// Workaround for openBao
	// Remove once openBao uses BAO_ADDR in their client, instead of VAULT_ADDR
	vaultPaths, ok := providerPaths[vault.ProviderName]
	if ok {
		var err error
		providerSecrets, err = s.workaroundForBao(ctx, vaultPaths)
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

			provider, err := newProvider(ctx, providerName, s.appConfig)
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
func (s *EnvStore) workaroundForBao(ctx context.Context, vaultPaths []string) ([]provider.Secret, error) {
	var secrets []provider.Secret

	provider, err := newProvider(ctx, vault.ProviderName, s.appConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider %s: %w", vault.ProviderName, err)
	}

	secrets, err = provider.LoadSecrets(ctx, vaultPaths)
	if err != nil {
		return nil, fmt.Errorf("failed to load secrets for provider %s: %w", vault.ProviderName, err)
	}

	return secrets, nil
}

// ConvertProviderSecrets converts the loaded secrets to environment variables
func (s *EnvStore) ConvertProviderSecrets(providerSecrets []provider.Secret) ([]string, error) {
	var secretsEnv []string

	for _, secret := range providerSecrets {
		secretsEnv = append(secretsEnv, fmt.Sprintf("%s=%s", secret.Key, secret.Value))
	}

	return secretsEnv, nil
}

// Returns the detected provider name and path with removed prefix
func getProviderPath(path string) (string, string) {
	if strings.HasPrefix(path, "file:") {
		return file.ProviderName, path
	}

	// If the path contains some string formatted as "vault:{STR}#{STR}"
	// it is most probably a vault path
	if vault.ProviderEnvRegex.MatchString(path) {
		return vault.ProviderName, path
	}

	// If the path contains some string formatted as "bao:{STR}#{STR}"
	// it is most probably a vault path
	if bao.ProviderEnvRegex.MatchString(path) {
		return bao.ProviderName, path
	}

	// Example AWS prefixes:
	// arn:aws:secretsmanager:us-west-2:123456789012:secret:my-secret
	// arn:aws:ssm:us-west-2:123456789012:parameter/my-parameter
	if strings.HasPrefix(path, "arn:aws:secretsmanager:") || strings.HasPrefix(path, "arn:aws:ssm:") {
		return aws.ProviderName, path
	}

	// Example GCP prefixes:
	// gcp:secretmanager:projects/{PROJECT_ID}/secrets/{SECRET_NAME}
	// gcp:secretmanager:projects/{PROJECT_ID}/secrets/{SECRET_NAME}/versions/{VERSION|latest}
	if strings.HasPrefix(path, "gcp:secretmanager:") {
		return gcp.ProviderName, path
	}

	// Example Azure Key Vault secret examples:
	// azure:keyvault:{SECRET_NAME}
	// azure:keyvault:{SECRET_NAME}/{VERSION}
	if strings.HasPrefix(path, "azure:keyvault:") {
		return azure.ProviderName, path
	}

	// Example Bitwarden secret examples:
	// bw:{SECRET_ID}
	// To retrieve all secrets in an organization:
	// bw:{ORGANIZATION_ID}
	// NOTE: (only works if BITWARDEN_ORGANIZATION_ID is also set to the same value)
	if strings.HasPrefix(path, "bitwarden:") {
		return bitwarden.ProviderName, path
	}

	return "", path
}

func newProvider(ctx context.Context, providerName string, appConfig *common.Config) (provider.Provider, error) {
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

	case aws.ProviderName:
		config, err := aws.LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create aws config: %w", err)
		}

		provider := aws.NewProvider(config)
		return provider, nil

	case gcp.ProviderName:
		provider, err := gcp.NewProvider(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create gcp provider: %w", err)
		}
		return provider, nil

	case azure.ProviderName:
		config, err := azure.LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create azure config: %w", err)
		}
		provider, err := azure.NewProvider(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create azure provider: %w", err)
		}
		return provider, nil

	case bitwarden.ProviderName:
		config, err := bitwarden.LoadConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create bitwarden config: %w", err)
		}

		provider, err := bitwarden.NewProvider(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create bitwarden provider: %w", err)
		}
		return provider, nil

	default:
		return nil, fmt.Errorf("provider %s is not supported", providerName)
	}
}
