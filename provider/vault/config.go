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
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cast"
)

// The special value for SECRET_INIT which marks that the login token needs to be passed through to the application
// which was acquired during the vault client initialization.
const (
	defaultEnvPrefix = "VAULT_"

	TokenEnv                = "TOKEN"
	TokenFileEnv            = "TOKEN_FILE"
	AddrEnv                 = "ADDR"
	AgentAddrEnv            = "AGENT_ADDR"
	CACertEnv               = "CACERT"
	CAPathEnv               = "CAPATH"
	ClientCertEnv           = "CLIENT_CERT"
	ClientKeyEnv            = "CLIENT_KEY"
	ClientTimeoutEnv        = "CLIENT_TIMEOUT"
	SRVLookupEnv            = "SRV_LOOKUP"
	SkipVerifyEnv           = "SKIP_VERIFY"
	NamespaceEnv            = "NAMESPACE"
	TLSServerNameEnv        = "TLS_SERVER_NAME"
	WrapTTLEnv              = "WRAP_TTL"
	MFAEnv                  = "MFA"
	MaxRetriesEnv           = "MAX_RETRIES"
	ClusterAddrEnv          = "CLUSTER_ADDR"
	RedirectAddrEnv         = "REDIRECT_ADDR"
	CLINoColorEnv           = "CLI_NO_COLOR"
	RateLimitEnv            = "RATE_LIMIT"
	RoleEnv                 = "ROLE"
	PathEnv                 = "PATH"
	AuthMethodEnv           = "AUTH_METHOD"
	TransitKeyIDEnv         = "TRANSIT_KEY_ID"
	TransitPathEnv          = "TRANSIT_PATH"
	TransitBatchSizeEnv     = "TRANSIT_BATCH_SIZE"
	IgnoreMissingSecretsEnv = "IGNORE_MISSING_SECRETS"
	PassthroughEnv          = "PASSTHROUGH"
	RevokeTokenEnv          = "REVOKE_TOKEN"
	FromPathEnv             = "FROM_PATH"

	vaultLogin = "vault:login"
)

type Config struct {
	IsLogin              bool   `json:"is_login"`
	Token                string `json:"token"`
	TokenFile            string `json:"token_file"`
	Role                 string `json:"role"`
	AuthPath             string `json:"auth_path"`
	AuthMethod           string `json:"auth_method"`
	TransitKeyID         string `json:"transit_key_id"`
	TransitPath          string `json:"transit_path"`
	TransitBatchSize     int    `json:"transit_batch_size"`
	IgnoreMissingSecrets bool   `json:"ignore_missing_secrets"`
	FromPath             string `json:"from_path"`
	RevokeToken          bool   `json:"revoke_token"`
}

type envType struct {
	login bool
}

var sanitizeEnvmap = map[string]envType{
	TokenEnv:                {login: true},
	AddrEnv:                 {login: true},
	AgentAddrEnv:            {login: true},
	CACertEnv:               {login: true},
	CAPathEnv:               {login: true},
	ClientCertEnv:           {login: true},
	ClientKeyEnv:            {login: true},
	ClientTimeoutEnv:        {login: true},
	SRVLookupEnv:            {login: true},
	SkipVerifyEnv:           {login: true},
	NamespaceEnv:            {login: true},
	TLSServerNameEnv:        {login: true},
	WrapTTLEnv:              {login: true},
	MFAEnv:                  {login: true},
	MaxRetriesEnv:           {login: true},
	ClusterAddrEnv:          {login: false},
	RedirectAddrEnv:         {login: false},
	CLINoColorEnv:           {login: false},
	RateLimitEnv:            {login: false},
	RoleEnv:                 {login: false},
	PathEnv:                 {login: false},
	AuthMethodEnv:           {login: false},
	TransitKeyIDEnv:         {login: false},
	TransitPathEnv:          {login: false},
	TransitBatchSizeEnv:     {login: false},
	IgnoreMissingSecretsEnv: {login: false},
	PassthroughEnv:          {login: false},
	RevokeTokenEnv:          {login: false},
	FromPathEnv:             {login: false},
}

func NewConfig() (*Config, error) {
	var (
		role, authPath, authMethod      string
		hasRole, hasPath, hasAuthMethod bool
	)

	// The login procedure takes the token from a file (if using Vault Agent)
	// or requests one for itself (Kubernetes Auth, or GCP, etc...),
	// so if we got a VAULT_TOKEN for the special value with "vault:login"
	vaultToken := os.Getenv(defaultEnvPrefix + TokenEnv)
	isLogin := vaultToken == vaultLogin
	tokenFile, ok := os.LookupEnv(defaultEnvPrefix + TokenFileEnv)
	if ok {
		// load token from vault-agent .vault-token or injected webhook
		tokenFileContent, err := os.ReadFile(tokenFile)
		if err != nil {
			slog.Error(fmt.Errorf("failed to read token file %s: %w", tokenFile, err).Error())

			return nil, fmt.Errorf("failed to read token file %s: %w", tokenFile, err)
		}
		vaultToken = string(tokenFileContent)
	} else {
		if isLogin {
			_ = os.Unsetenv(defaultEnvPrefix + TokenEnv)
		}
		// will use role/path based authentication
		role, hasRole = os.LookupEnv(defaultEnvPrefix + RoleEnv)
		authPath, hasPath = os.LookupEnv(defaultEnvPrefix + PathEnv)
		authMethod, hasAuthMethod = os.LookupEnv(defaultEnvPrefix + AuthMethodEnv)
		var missingConfig []string

		if !hasRole {
			missingConfig = append(missingConfig, defaultEnvPrefix+RoleEnv)
		}
		if !hasPath {
			missingConfig = append(missingConfig, defaultEnvPrefix+PathEnv)
		}
		if !hasAuthMethod {
			missingConfig = append(missingConfig, defaultEnvPrefix+AuthMethodEnv)
		}

		if len(missingConfig) > 0 {
			return nil, fmt.Errorf("incomplete authentication configuration: %s missing", strings.Join(missingConfig, ", "))
		}
	}

	passthroughEnvVars := strings.Split(os.Getenv(defaultEnvPrefix+PassthroughEnv), ",")
	if isLogin {
		_ = os.Setenv(defaultEnvPrefix+TokenEnv, vaultLogin)
		passthroughEnvVars = append(passthroughEnvVars, TokenEnv)
	}

	// do not sanitize env vars specified in VAULT_PASSTHROUGH
	for _, envVar := range passthroughEnvVars {
		if trimmed := strings.TrimSpace(envVar); trimmed != "" {
			delete(sanitizeEnvmap, trimmed)
		}
	}

	return &Config{
		IsLogin:    isLogin,
		Token:      vaultToken,
		TokenFile:  tokenFile,
		Role:       role,
		AuthPath:   authPath,
		AuthMethod: authMethod,
		// injector configuration
		TransitKeyID:     os.Getenv(defaultEnvPrefix + TransitKeyIDEnv),
		TransitPath:      os.Getenv(defaultEnvPrefix + TransitPathEnv),
		TransitBatchSize: cast.ToInt(os.Getenv(defaultEnvPrefix + TransitBatchSizeEnv)),
		// Used both for reading secrets and transit encryption
		IgnoreMissingSecrets: cast.ToBool(os.Getenv(defaultEnvPrefix + IgnoreMissingSecretsEnv)),
		FromPath:             os.Getenv(defaultEnvPrefix + FromPathEnv),
		RevokeToken:          cast.ToBool(os.Getenv(defaultEnvPrefix + RevokeTokenEnv)),
	}, nil
}
