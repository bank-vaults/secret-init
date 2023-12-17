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
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cast"
)

const (
	EnvPrefix = "VAULT_"
	// The special value for SECRET_INIT which marks that the login token needs to be passed through to the application
	// which was acquired during the vault client initialization.
	vaultLogin = "vault:login"
)

type Config struct {
	Islogin              bool     `json:"islogin"`
	Token                string   `json:"token"`
	TokenFile            string   `json:"tokenFile"`
	Role                 string   `json:"role"`
	Path                 string   `json:"path"`
	AuthMethod           string   `json:"authMethod"`
	PassthroughEnvVars   []string `json:"passthroughEnvVars"`
	TransitKeyID         string   `json:"transitKeyID"`
	TransitPath          string   `json:"transitPath"`
	TransitBatchSize     int      `json:"transitBatchSize"`
	DaemonMode           bool     `json:"daemonMode"`
	IgnoreMissingSecrets bool     `json:"ignoreMissingSecrets"`
	Paths                string   `json:"paths"`
	RevokeToken          bool     `json:"revokeToken"`
	Logger               *slog.Logger
	Sigs                 chan os.Signal
}

func NewConfig(logger *slog.Logger, sigs chan os.Signal) (*Config, error) {
	var (
		role, path, authMethod          string
		hasRole, hasPath, hasAuthMethod bool
	)

	vaultToken := os.Getenv(EnvPrefix + "TOKEN")
	isLogin := vaultToken == vaultLogin
	tokenFile, ok := os.LookupEnv(EnvPrefix + "TOKEN_FILE")
	if !ok {
		if b, err := os.ReadFile(tokenFile); err == nil {
			vaultToken = string(b)
		} else {
			logger.Error("could not read vault token file", slog.String("file", tokenFile))

			return nil, err
		}
	} else {
		if isLogin {
			_ = os.Unsetenv(EnvPrefix + "TOKEN")
		}

		role, hasRole = os.LookupEnv(EnvPrefix + "ROLE")
		path, hasPath = os.LookupEnv(EnvPrefix + "PATH")
		authMethod, hasAuthMethod = os.LookupEnv(EnvPrefix + "AUTH_METHOD")
		if !hasRole || !hasPath || !hasAuthMethod {
			logger.Error("Incomplete authentication configuration. Make sure VAULT_ROLE, VAULT_PATH, and VAULT_AUTH_METHOD are set.")

			return nil, errors.New("incomplete authentication configuration")
		}
	}

	// TODO: make this generic, since it's not specific to vault
	passthroughEnvVars := strings.Split(os.Getenv("SECRET_INIT_PASSTHROUGH"), ",")
	if isLogin {
		_ = os.Setenv(EnvPrefix+"TOKEN", vaultLogin)
		passthroughEnvVars = append(passthroughEnvVars, EnvPrefix+"TOKEN")
	}

	transitKeyID := os.Getenv(EnvPrefix + "TRANSIT_KEY_ID")
	transitPath := os.Getenv(EnvPrefix + "TRANSIT_PATH")
	transitBatchSize := cast.ToInt(os.Getenv(EnvPrefix + "TRANSIT_BATCH_SIZE"))
	daemonMode := cast.ToBool(os.Getenv(EnvPrefix + "DAEMON_MODE"))
	if daemonMode {
		logger.Info("Daemon mode enabled. Will renew secrets in the background.")
	}

	// Used both for reading secrets and transit encryption
	ignoreMissingSecrets := cast.ToBool(os.Getenv(EnvPrefix + "IGNORE_MISSING_SECRETS"))

	paths := os.Getenv("VAULT_FROM_PATH")
	revokeToken := cast.ToBool(os.Getenv(EnvPrefix + "REVOKE_TOKEN"))

	return &Config{
		Islogin:              isLogin,
		Token:                vaultToken,
		TokenFile:            tokenFile,
		Role:                 role,
		Path:                 path,
		AuthMethod:           authMethod,
		PassthroughEnvVars:   passthroughEnvVars,
		TransitKeyID:         transitKeyID,
		TransitPath:          transitPath,
		TransitBatchSize:     transitBatchSize,
		DaemonMode:           daemonMode,
		IgnoreMissingSecrets: ignoreMissingSecrets,
		Paths:                paths,
		RevokeToken:          revokeToken,
		Logger:               logger,
		Sigs:                 sigs,
	}, nil
}
