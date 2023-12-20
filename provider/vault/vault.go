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
	"strings"

	"github.com/bank-vaults/internal/injector"
	"github.com/bank-vaults/vault-sdk/vault"

	"github.com/bank-vaults/secret-init/provider"
)

const ProviderName = "vault"

type Provider struct {
	isLogin        bool
	client         *vault.Client
	injectorConfig injector.Config
	secretRenewer  injector.SecretRenewer
	fromPath       string
	revokeToken    bool
	logger         *slog.Logger
}

type sanitized struct {
	secrets []provider.Secret
	login   bool
}

// VAULT_* variables are not populated into this list if this is not a login scenario.
func (s *sanitized) append(key string, value string) {
	envType, ok := sanitizeEnvmap[key]
	if !ok || (s.login && envType.login) {
		// Path here is actually the secrets key
		secret := provider.Secret{
			Path:  key,
			Value: value,
		}

		s.secrets = append(s.secrets, secret)
	}
}

func NewProvider(config *Config) (provider.Provider, error) {
	clientOptions := []vault.ClientOption{vault.ClientLogger(clientLogger{config.Logger})}
	if config.TokenFile != "" {
		clientOptions = append(clientOptions, vault.ClientToken(config.Token))
	} else {
		// use role/path based authentication
		clientOptions = append(clientOptions,
			vault.ClientRole(config.Role),
			vault.ClientAuthPath(config.AuthPath),
			vault.ClientAuthMethod(config.AuthMethod),
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
		config.Logger.Info("Daemon mode enabled. Will renew secrets in the background.")
	}

	return &Provider{
		isLogin:        config.Islogin,
		client:         client,
		injectorConfig: injectorConfig,
		secretRenewer:  secretRenewer,
		fromPath:       config.FromPath,
		revokeToken:    config.RevokeToken,
		logger:         config.Logger,
	}, nil
}

func (p *Provider) LoadSecrets(_ context.Context, paths []string) ([]provider.Secret, error) {
	sanitized := sanitized{login: p.isLogin}
	vaultEnviron := parsePathsToMap(paths)

	secretInjector := injector.NewSecretInjector(p.injectorConfig, p.client, p.secretRenewer, p.logger)
	inject := func(key, value string) {
		sanitized.append(key, value)
	}

	err := secretInjector.InjectSecretsFromVault(vaultEnviron, inject)
	if err != nil {
		p.logger.Error(fmt.Errorf("failed to inject secrets from vault: %w", err).Error())

		return nil, err
	}

	if p.fromPath != "" {
		err = secretInjector.InjectSecretsFromVaultPath(p.fromPath, inject)
		if err != nil {
			p.logger.Error(fmt.Errorf("failed to inject secrets from vault path: %w", err).Error())

			return nil, err
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

	return sanitized.secrets, nil
}

func parsePathsToMap(paths []string) map[string]string {
	vaultEnviron := make(map[string]string)

	for _, path := range paths {
		split := strings.SplitN(path, "=", 2)
		key := split[0]
		value := split[1]
		vaultEnviron[key] = value
	}

	return vaultEnviron
}

func (p *Provider) GetProviderName() string {
	return ProviderName
}
