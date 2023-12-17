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

package vault

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/bank-vaults/internal/injector"
	"github.com/bank-vaults/vault-sdk/vault"

	"github.com/bank-vaults/secret-init/provider"
)

const ProviderName = "vault"

type Provider struct {
	client         *vault.Client
	injectorConfig injector.Config
	secretRenewer  injector.SecretRenewer
	paths          string
	revokeToken    bool
	logger         *slog.Logger
}

func NewProvider(config *Config) (provider.Provider, error) {
	clientOptions := []vault.ClientOption{vault.ClientLogger(clientLogger{config.Logger})}
	if config.TokenFile != "" {
		clientOptions = append(clientOptions, vault.ClientToken(config.TokenFile))
	} else {
		// use role/path based authentication
		clientOptions = append(clientOptions,
			vault.ClientRole(os.Getenv("VAULT_ROLE")),
			vault.ClientAuthPath(os.Getenv("VAULT_PATH")),
			vault.ClientAuthMethod(os.Getenv("VAULT_AUTH_METHOD")),
		)
	}

	client, err := vault.NewClientWithOptions(clientOptions...)
	if err != nil {
		config.Logger.Error(fmt.Errorf("failed to create vault client: %w", err).Error())

		return nil, err
	}

	injectorConfig := injector.Config{
		TransitKeyID:         config.TransitKeyID,
		TransitPath:          config.TransitPath,
		TransitBatchSize:     config.TransitBatchSize,
		DaemonMode:           config.DaemonMode,
		IgnoreMissingSecrets: config.IgnoreMissingSecrets,
	}

	var secretRenewer injector.SecretRenewer

	if config.DaemonMode {
		secretRenewer = daemonSecretRenewer{client: client, sigs: config.Sigs, logger: config.Logger}
	}

	return &Provider{
		client:         client,
		injectorConfig: injectorConfig,
		secretRenewer:  secretRenewer,
		paths:          config.Paths,
		revokeToken:    config.RevokeToken,
		logger:         config.Logger,
	}, nil
}

func (p *Provider) LoadSecrets(_ context.Context, paths []string) ([]provider.Secret, error) {
	secretInjector := injector.NewSecretInjector(p.injectorConfig, p.client, p.secretRenewer, p.logger)

	var secrets []provider.Secret
	inject := func(key, value string) {
		secret := provider.Secret{
			Path:  key,
			Value: value,
		}

		secrets = append(secrets, secret)
	}

	for _, path := range paths {
		err := secretInjector.InjectSecretsFromVaultPath(path, inject)
		if err != nil {
			return nil, fmt.Errorf("failed to inject secrets: %w", err)
		}
	}

	if p.revokeToken {
		// ref: https://www.vaultproject.io/api/auth/token/index.html#revoke-a-token-self-
		err := p.client.RawClient().Auth().Token().RevokeSelf(p.client.RawClient().Token())
		if err != nil {
			// Do not exit on error, token revoking can be denied by policy
			p.logger.Warn("failed to revoke token")
		}

		p.client.Close()
	}

	return secrets, nil
}
