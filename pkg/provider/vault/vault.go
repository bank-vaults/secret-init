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
	"os/signal"
	"regexp"
	"strings"

	injector "github.com/bank-vaults/vault-sdk/injector/vault"
	"github.com/bank-vaults/vault-sdk/vault"

	"github.com/bank-vaults/secret-init/pkg/common"
	"github.com/bank-vaults/secret-init/pkg/provider"
)

const (
	ProviderType      = "vault"
	referenceSelector = `(vault:)(.*)#(.*)`
)

type Provider struct {
	isLogin        bool
	client         *vault.Client
	injectorConfig injector.Config
	secretRenewer  injector.SecretRenewer
	fromPath       string
	revokeToken    bool
}

type sanitized struct {
	secrets []provider.Secret
	login   bool
}

// VAULT_* variables are not populated into this list if this is not a login scenario.
func (s *sanitized) append(key string, value string) {
	envType, ok := sanitizeEnvmap[key]
	// If the key being appended is not present in sanitizeEnvmap, it signifies that
	// it is not a VAULT_* variable.
	// Additionally, in a login scenario, we include VAULT_* variables in the secrets list.
	if !ok || (s.login && envType.login) {
		// Example can be found at the LoadSecrets() function below
		secret := provider.Secret{
			Key:   key,
			Value: value,
		}

		s.secrets = append(s.secrets, secret)
	}
}

func NewProvider(_ context.Context, appConfig *common.Config) (provider.Provider, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create vault config: %w", err)
	}

	clientOptions := []vault.ClientOption{vault.ClientLogger(clientLogger{slog.Default()})}
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
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	injectorConfig := injector.Config{
		TransitKeyID:         config.TransitKeyID,
		TransitPath:          config.TransitPath,
		TransitBatchSize:     config.TransitBatchSize,
		IgnoreMissingSecrets: config.IgnoreMissingSecrets,
		DaemonMode:           appConfig.Daemon,
	}

	var secretRenewer injector.SecretRenewer

	if appConfig.Daemon {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs)

		secretRenewer = daemonSecretRenewer{client: client, sigs: sigs}
		slog.Info("Daemon mode enabled. Will renew secrets in the background.")
	}

	return &Provider{
		isLogin:        config.IsLogin,
		client:         client,
		injectorConfig: injectorConfig,
		secretRenewer:  secretRenewer,
		fromPath:       config.FromPath,
		revokeToken:    config.RevokeToken,
	}, nil
}

// GetName returns the name of the provider
func (p *Provider) GetName() string {
	return providerName
}

// LoadSecret's path formatting: <key>=<path>
// This formatting is necessary because the injector expects a map of key=value pairs.
// It also returns a map of key:value pairs, where the key is the environment variable name
// and the value is the secret value
// E.g. paths: MYSQL_PASSWORD=secret/data/mysql/password
// returns: []provider.Secret{provider.Secret{Path: "MYSQL_PASSWORD", Value: "password"}}
func (p *Provider) LoadSecrets(ctx context.Context, paths []string) ([]provider.Secret, error) {
	sanitized := sanitized{login: p.isLogin}
	vaultEnviron := parsePathsToMap(paths)

	secretInjector := injector.NewSecretInjector(p.injectorConfig, p.client, p.secretRenewer, slog.Default())
	inject := func(key, value string) {
		sanitized.append(key, value)
	}

	err := secretInjector.InjectSecretsFromVault(vaultEnviron, inject)
	if err != nil {
		return nil, fmt.Errorf("failed to inject secrets from vault: %w", err)
	}

	if p.fromPath != "" {
		err = secretInjector.InjectSecretsFromVaultPath(p.fromPath, inject)
		if err != nil {
			return nil, fmt.Errorf("failed to inject secrets from vault path: %w", err)
		}
	}

	if p.revokeToken {
		// ref: https://www.vaultproject.io/api/auth/token/index.html#revoke-a-token-self-
		err := p.client.RawClient().Auth().Token().RevokeSelfWithContext(ctx, p.client.RawClient().Token())
		if err != nil {
			// Do not exit on error, token revoking can be denied by policy
			slog.Warn("failed to revoke token")
		}

		p.client.Close()
	}

	return sanitized.secrets, nil
}

// If the path contains some string formatted as "vault:{STR}#{STR}"
// it is most probably a vault path
func Valid(envValue string) bool {
	return regexp.MustCompile(referenceSelector).MatchString(envValue)
}

func parsePathsToMap(paths []string) map[string]string {
	vaultEnviron := make(map[string]string)

	for _, path := range paths {
		split := strings.SplitN(path, "=", 2)
		originalKey, value := split[0], split[1]
		vaultEnviron[originalKey] = value
	}

	return vaultEnviron
}
