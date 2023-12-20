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
	Islogin              bool   `json:"islogin"`
	Token                string `json:"token"`
	TokenFile            string `json:"tokenFile"`
	Role                 string `json:"role"`
	AuthPath             string `json:"authPath"`
	AuthMethod           string `json:"authMethod"`
	TransitKeyID         string `json:"transitKeyID"`
	TransitPath          string `json:"transitPath"`
	TransitBatchSize     int    `json:"transitBatchSize"`
	DaemonMode           bool   `json:"daemonMode"`
	IgnoreMissingSecrets bool   `json:"ignoreMissingSecrets"`
	FromPath             string `json:"fromPath"`
	RevokeToken          bool   `json:"revokeToken"`
	Logger               *slog.Logger
	Sigs                 chan os.Signal
}

type envType struct {
	login bool
}

var sanitizeEnvmap = map[string]envType{
	"VAULT_TOKEN":                  {login: true},
	"VAULT_ADDR":                   {login: true},
	"VAULT_AGENT_ADDR":             {login: true},
	"VAULT_CACERT":                 {login: true},
	"VAULT_CAPATH":                 {login: true},
	"VAULT_CLIENT_CERT":            {login: true},
	"VAULT_CLIENT_KEY":             {login: true},
	"VAULT_CLIENT_TIMEOUT":         {login: true},
	"VAULT_SRV_LOOKUP":             {login: true},
	"VAULT_SKIP_VERIFY":            {login: true},
	"VAULT_NAMESPACE":              {login: true},
	"VAULT_TLS_SERVER_NAME":        {login: true},
	"VAULT_WRAP_TTL":               {login: true},
	"VAULT_MFA":                    {login: true},
	"VAULT_MAX_RETRIES":            {login: true},
	"VAULT_CLUSTER_ADDR":           {login: false},
	"VAULT_REDIRECT_ADDR":          {login: false},
	"VAULT_CLI_NO_COLOR":           {login: false},
	"VAULT_RATE_LIMIT":             {login: false},
	"VAULT_ROLE":                   {login: false},
	"VAULT_PATH":                   {login: false},
	"VAULT_AUTH_METHOD":            {login: false},
	"VAULT_TRANSIT_KEY_ID":         {login: false},
	"VAULT_TRANSIT_PATH":           {login: false},
	"VAULT_TRANSIT_BATCH_SIZE":     {login: false},
	"VAULT_IGNORE_MISSING_SECRETS": {login: false},
	"VAULT_PASSTHROUGH":            {login: false},
	"VAULT_REVOKE_TOKEN":           {login: false},
	"VAULT_FROM_PATH":              {login: false},
	"SECRET_INIT_DAEMON":           {login: false},
}

func NewConfig(logger *slog.Logger, sigs chan os.Signal) (*Config, error) {
	var (
		role, authPath, authMethod      string
		hasRole, hasPath, hasAuthMethod bool
	)

	// The login procedure takes the token from a file (if using Vault Agent)
	// or requests one for itself (Kubernetes Auth, or GCP, etc...),
	// so if we got a VAULT_TOKEN for the special value with "vault:login"
	vaultToken := os.Getenv(EnvPrefix + "TOKEN")
	isLogin := vaultToken == vaultLogin
	tokenFile, ok := os.LookupEnv(EnvPrefix + "TOKEN_FILE")
	if ok {
		// load token from vault-agent .vault-token or injected webhook
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
		// will use role/path based authentication
		role, hasRole = os.LookupEnv(EnvPrefix + "ROLE")
		authPath, hasPath = os.LookupEnv(EnvPrefix + "PATH")
		authMethod, hasAuthMethod = os.LookupEnv(EnvPrefix + "AUTH_METHOD")
		if !hasRole || !hasPath || !hasAuthMethod {
			logger.Error("Incomplete authentication configuration. Make sure VAULT_ROLE, VAULT_PATH, and VAULT_AUTH_METHOD are set.")

			return nil, errors.New("incomplete authentication configuration")
		}
	}

	passthroughEnvVars := strings.Split(os.Getenv(EnvPrefix+"PASSTHROUGH"), ",")
	if isLogin {
		_ = os.Setenv(EnvPrefix+"TOKEN", vaultLogin)
		passthroughEnvVars = append(passthroughEnvVars, EnvPrefix+"TOKEN")
	}

	// do not sanitize env vars specified in VAULT_PASSTHROUGH
	for _, envVar := range passthroughEnvVars {
		if trimmed := strings.TrimSpace(envVar); trimmed != "" {
			delete(sanitizeEnvmap, trimmed)
		}
	}

	// injector configuration
	transitKeyID := os.Getenv(EnvPrefix + "TRANSIT_KEY_ID")
	transitPath := os.Getenv(EnvPrefix + "TRANSIT_PATH")
	transitBatchSize := cast.ToInt(os.Getenv(EnvPrefix + "TRANSIT_BATCH_SIZE"))
	daemonMode := cast.ToBool(os.Getenv("SECRET_INIT_DAEMON_MODE"))
	// Used both for reading secrets and transit encryption
	ignoreMissingSecrets := cast.ToBool(os.Getenv(EnvPrefix + "IGNORE_MISSING_SECRETS"))

	fromPath := os.Getenv(EnvPrefix + "FROM_PATH")
	revokeToken := cast.ToBool(os.Getenv(EnvPrefix + "REVOKE_TOKEN"))

	return &Config{
		Islogin:              isLogin,
		Token:                vaultToken,
		TokenFile:            tokenFile,
		Role:                 role,
		AuthPath:             authPath,
		AuthMethod:           authMethod,
		TransitKeyID:         transitKeyID,
		TransitPath:          transitPath,
		TransitBatchSize:     transitBatchSize,
		DaemonMode:           daemonMode,
		IgnoreMissingSecrets: ignoreMissingSecrets,
		FromPath:             fromPath,
		RevokeToken:          revokeToken,
		Logger:               logger,
		Sigs:                 sigs,
	}, nil
}
