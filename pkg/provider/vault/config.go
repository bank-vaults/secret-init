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
	// The special value for VAULT_TOKEN which marks
	// that the login token needs to be passed through to the application
	// which was acquired during the vault client initialization.
	vaultLogin = "vault:login"

	tokenEnv                = "VAULT_TOKEN"
	tokenFileEnv            = "VAULT_TOKEN_FILE"
	addrEnv                 = "VAULT_ADDR"
	agentAddrEnv            = "VAULT_AGENT_ADDR"
	caCertEnv               = "VAULT_CACERT"
	caPathEnv               = "VAULT_CAPATH"
	clientCertEnv           = "VAULT_CLIENT_CERT"
	clientKeyEnv            = "VAULT_CLIENT_KEY"
	clientTimeoutEnv        = "VAULT_CLIENT_TIMEOUT"
	srvLookupEnv            = "VAULT_SRV_LOOKUP"
	skipVerifyEnv           = "VAULT_SKIP_VERIFY"
	namespaceEnv            = "VAULT_NAMESPACE"
	tlsServerNameEnv        = "VAULT_TLS_SERVER_NAME"
	wrapTTLEnv              = "VAULT_WRAP_TTL"
	mfaEnv                  = "VAULT_MFA"
	maxRetriesEnv           = "VAULT_MAX_RETRIES"
	clusterAddrEnv          = "VAULT_CLUSTER_ADDR"
	redirectAddrEnv         = "VAULT_REDIRECT_ADDR"
	cliNoColorEnv           = "VAULT_CLI_NO_COLOR"
	rateLimitEnv            = "VAULT_RATE_LIMIT"
	roleEnv                 = "VAULT_ROLE"
	pathEnv                 = "VAULT_PATH"
	authMethodEnv           = "VAULT_AUTH_METHOD"
	transitKeyIDEnv         = "VAULT_TRANSIT_KEY_ID"
	transitPathEnv          = "VAULT_TRANSIT_PATH"
	transitBatchSizeEnv     = "VAULT_TRANSIT_BATCH_SIZE"
	ignoreMissingSecretsEnv = "VAULT_IGNORE_MISSING_SECRETS"
	passthroughEnv          = "VAULT_PASSTHROUGH"
	logLevelEnv             = "VAULT_LOG_LEVEL"
	revokeTokenEnv          = "VAULT_REVOKE_TOKEN"
	FromPathEnv             = "VAULT_FROM_PATH"
)

type Config struct {
	IsLogin              bool   `json:"is_login"`
	Addr                 string `json:"addr"`
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
	tokenEnv:                {login: true},
	addrEnv:                 {login: true},
	agentAddrEnv:            {login: true},
	caCertEnv:               {login: true},
	caPathEnv:               {login: true},
	clientCertEnv:           {login: true},
	clientKeyEnv:            {login: true},
	clientTimeoutEnv:        {login: true},
	srvLookupEnv:            {login: true},
	skipVerifyEnv:           {login: true},
	namespaceEnv:            {login: true},
	tlsServerNameEnv:        {login: true},
	wrapTTLEnv:              {login: true},
	mfaEnv:                  {login: true},
	maxRetriesEnv:           {login: true},
	clusterAddrEnv:          {login: false},
	redirectAddrEnv:         {login: false},
	cliNoColorEnv:           {login: false},
	rateLimitEnv:            {login: false},
	roleEnv:                 {login: false},
	pathEnv:                 {login: false},
	authMethodEnv:           {login: false},
	transitKeyIDEnv:         {login: false},
	transitPathEnv:          {login: false},
	transitBatchSizeEnv:     {login: false},
	ignoreMissingSecretsEnv: {login: false},
	passthroughEnv:          {login: false},
	logLevelEnv:             {login: false},
	revokeTokenEnv:          {login: false},
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
	vaultToken := os.Getenv(tokenEnv)
	isLogin := vaultToken == vaultLogin
	tokenFile, ok := os.LookupEnv(tokenFileEnv)
	if ok {
		// load token from vault-agent .vault-token or injected webhook
		tokenFileContent, err := os.ReadFile(tokenFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read token file %s: %w", tokenFile, err)
		}
		vaultToken = string(tokenFileContent)
	} else {
		if isLogin {
			_ = os.Unsetenv(tokenEnv)
		}

		// will use role/path based authentication
		role, hasRole = os.LookupEnv(roleEnv)
		if !hasRole {
			return nil, fmt.Errorf("incomplete authentication configuration: %s missing", roleEnv)
		}
		authPath, hasPath = os.LookupEnv(pathEnv)
		if !hasPath {
			return nil, fmt.Errorf("incomplete authentication configuration: %s missing", pathEnv)
		}
		authMethod, hasAuthMethod = os.LookupEnv(authMethodEnv)
		if !hasAuthMethod {
			return nil, fmt.Errorf("incomplete authentication configuration: %s missing", authMethodEnv)
		}
	}

	passthroughEnvVars := strings.Split(os.Getenv(passthroughEnv), ",")
	if isLogin {
		_ = os.Setenv(tokenEnv, vaultLogin)
		passthroughEnvVars = append(passthroughEnvVars, tokenEnv)
	}

	// do not sanitize env vars specified in VAULT_PASSTHROUGH
	for _, envVar := range passthroughEnvVars {
		if trimmed := strings.TrimSpace(envVar); trimmed != "" {
			delete(sanitizeEnvmap, trimmed)
		}
	}

	return &Config{
		IsLogin:              isLogin,
		Addr:                 os.Getenv(addrEnv),
		Token:                vaultToken,
		TokenFile:            tokenFile,
		Role:                 role,
		AuthPath:             authPath,
		AuthMethod:           authMethod,
		TransitKeyID:         os.Getenv(transitKeyIDEnv),
		TransitPath:          os.Getenv(transitPathEnv),
		TransitBatchSize:     cast.ToInt(os.Getenv(transitBatchSizeEnv)),
		IgnoreMissingSecrets: cast.ToBool(os.Getenv(ignoreMissingSecretsEnv)), // Used both for reading secrets and transit encryption
		FromPath:             os.Getenv(FromPathEnv),
		RevokeToken:          cast.ToBool(os.Getenv(revokeTokenEnv)),
	}, nil
}
