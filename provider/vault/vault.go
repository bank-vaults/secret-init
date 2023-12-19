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
	"strings"

	"github.com/bank-vaults/internal/injector"
	"github.com/bank-vaults/vault-sdk/vault"

	"github.com/bank-vaults/secret-init/provider"
)

const ProviderName = "vault"

type Provider struct {
	isLogin            bool
	client             *vault.Client
	injectorConfig     injector.Config
	secretRenewer      injector.SecretRenewer
	passthroughEnvVars []string
	paths              string
	revokeToken        bool
	logger             *slog.Logger
}

type sanitizedEnviron struct {
	secrets []provider.Secret
	login   bool
}

// Appends variable an entry (name=value) into the environ list.
// VAULT_* variables are not populated into this list if this is not a login scenario.
func (e *sanitizedEnviron) append(path string, key string, value string) {
	envType, ok := sanitizeEnvmap[key]
	if !ok || (e.login && envType.login) {
		secret := provider.Secret{
			Path:  path,
			Value: value,
		}

		e.secrets = append(e.secrets, secret)
	}
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
		config.Logger.Info("Daemon mode enabled. Will renew secrets in the background.")
	}

	return &Provider{
		isLogin:            config.Islogin,
		client:             client,
		injectorConfig:     injectorConfig,
		secretRenewer:      secretRenewer,
		passthroughEnvVars: config.PassthroughEnvVars,
		paths:              config.Paths,
		revokeToken:        config.RevokeToken,
		logger:             config.Logger,
	}, nil
}

func (p *Provider) LoadSecrets(_ context.Context, paths []string) ([]provider.Secret, error) {
	secretInjector := injector.NewSecretInjector(p.injectorConfig, p.client, p.secretRenewer, p.logger)

	// do not sanitize env vars specified in SECRET_INIT_PASSTHROUGH
	for _, envVar := range p.passthroughEnvVars {
		if trimmed := strings.TrimSpace(envVar); trimmed != "" {
			delete(sanitizeEnvmap, trimmed)
		}
	}

	sanitized := sanitizedEnviron{login: p.isLogin}

	// inject secrets from VAULT_FROM_PATH
	paths = append(paths, strings.Split(p.paths, ",")...)

	for _, path := range paths {
		// Create a closure to capture the current path
		injectClosure := func(path string) func(key, value string) {
			return func(key, value string) {
				sanitized.append(path, key, value)
			}
		}(path)

		err := secretInjector.InjectSecretsFromVaultPath(path, injectClosure)
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

	return sanitized.secrets, nil
}
