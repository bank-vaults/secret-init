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
	"os"
	"strings"

	"github.com/spf13/cast"
)

const (
	// The special value for SECRET_INIT which marks
	// that the login token needs to be passed through to the application
	// which was acquired during the vault client initialization.
	vaultLogin = "vault:login"

	TokenEnv                = "VAULT_TOKEN"
	TokenFileEnv            = "VAULT_TOKEN_FILE"
	AddrEnv                 = "VAULT_ADDR"
	AgentAddrEnv            = "VAULT_AGENT_ADDR"
	CACertEnv               = "VAULT_CACERT"
	CAPathEnv               = "VAULT_CAPATH"
	ClientCertEnv           = "VAULT_CLIENT_CERT"
	ClientKeyEnv            = "VAULT_CLIENT_KEY"
	ClientTimeoutEnv        = "VAULT_CLIENT_TIMEOUT"
	SRVLookupEnv            = "VAULT_SRV_LOOKUP"
	SkipVerifyEnv           = "VAULT_SKIP_VERIFY"
	NamespaceEnv            = "VAULT_NAMESPACE"
	TLSServerNameEnv        = "VAULT_TLS_SERVER_NAME"
	WrapTTLEnv              = "VAULT_WRAP_TTL"
	MFAEnv                  = "VAULT_MFA"
	MaxRetriesEnv           = "VAULT_MAX_RETRIES"
	ClusterAddrEnv          = "VAULT_CLUSTER_ADDR"
	RedirectAddrEnv         = "VAULT_REDIRECT_ADDR"
	CLINoColorEnv           = "VAULT_CLI_NO_COLOR"
	RateLimitEnv            = "VAULT_RATE_LIMIT"
	RoleEnv                 = "VAULT_ROLE"
	PathEnv                 = "VAULT_PATH"
	AuthMethodEnv           = "VAULT_AUTH_METHOD"
	TransitKeyIDEnv         = "VAULT_TRANSIT_KEY_ID"
	TransitPathEnv          = "VAULT_TRANSIT_PATH"
	TransitBatchSizeEnv     = "VAULT_TRANSIT_BATCH_SIZE"
	IgnoreMissingSecretsEnv = "VAULT_IGNORE_MISSING_SECRETS"
	PassthroughEnv          = "VAULT_PASSTHROUGH"
	RevokeTokenEnv          = "VAULT_REVOKE_TOKEN"
	FromPathEnv             = "VAULT_FROM_PATH"
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

func LoadConfig() (*Config, error) {
	var (
		role, authPath, authMethod      string
		hasRole, hasPath, hasAuthMethod bool
	)

	// The login procedure takes the token from a file (if using Vault Agent)
	// or requests one for itself (Kubernetes Auth, or GCP, etc...),
	// so if we got a VAULT_TOKEN for the special value with "vault:login"
	vaultToken := os.Getenv(TokenEnv)
	isLogin := vaultToken == vaultLogin
	tokenFile, ok := os.LookupEnv(TokenFileEnv)
	if ok {
		// load token from vault-agent .vault-token or injected webhook
		tokenFileContent, err := os.ReadFile(tokenFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read token file %s: %w", tokenFile, err)
		}
		vaultToken = string(tokenFileContent)
	} else {
		if isLogin {
			_ = os.Unsetenv(TokenEnv)
		}

		// will use role/path based authentication
		role, hasRole = os.LookupEnv(RoleEnv)
		if !hasRole {
			return nil, fmt.Errorf("incomplete authentication configuration: %s missing", RoleEnv)
		}
		authPath, hasPath = os.LookupEnv(PathEnv)
		if !hasPath {
			return nil, fmt.Errorf("incomplete authentication configuration: %s missing", PathEnv)
		}
		authMethod, hasAuthMethod = os.LookupEnv(AuthMethodEnv)
		if !hasAuthMethod {
			return nil, fmt.Errorf("incomplete authentication configuration: %s missing", AuthMethodEnv)
		}
	}

	passthroughEnvVars := strings.Split(os.Getenv(PassthroughEnv), ",")
	if isLogin {
		_ = os.Setenv(TokenEnv, vaultLogin)
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
		TransitKeyID:     os.Getenv(TransitKeyIDEnv),
		TransitPath:      os.Getenv(TransitPathEnv),
		TransitBatchSize: cast.ToInt(os.Getenv(TransitBatchSizeEnv)),
		// Used both for reading secrets and transit encryption
		IgnoreMissingSecrets: cast.ToBool(os.Getenv(IgnoreMissingSecretsEnv)),
		FromPath:             os.Getenv(FromPathEnv),
		RevokeToken:          cast.ToBool(os.Getenv(RevokeTokenEnv)),
	}, nil
}
