// Copyright © 2024 Bank-Vaults Maintainers
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

package bao

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cast"
)

const (
	// The special value for BAO_TOKEN which marks
	// that the login token needs to be passed through to the application
	// which was acquired during the bao client initialization.
	baoLogin = "bao:login"

	TokenEnv                = "BAO_TOKEN"
	TokenFileEnv            = "BAO_TOKEN_FILE"
	AddrEnv                 = "BAO_ADDR"
	AgentAddrEnv            = "BAO_AGENT_ADDR"
	CACertEnv               = "BAO_CACERT"
	CAPathEnv               = "BAO_CAPATH"
	ClientCertEnv           = "BAO_CLIENT_CERT"
	ClientKeyEnv            = "BAO_CLIENT_KEY"
	ClientTimeoutEnv        = "BAO_CLIENT_TIMEOUT"
	SRVLookupEnv            = "BAO_SRV_LOOKUP"
	SkipVerifyEnv           = "BAO_SKIP_VERIFY"
	NamespaceEnv            = "BAO_NAMESPACE"
	TLSServerNameEnv        = "BAO_TLS_SERVER_NAME"
	WrapTTLEnv              = "BAO_WRAP_TTL"
	MFAEnv                  = "BAO_MFA"
	MaxRetriesEnv           = "BAO_MAX_RETRIES"
	ClusterAddrEnv          = "BAO_CLUSTER_ADDR"
	RedirectAddrEnv         = "BAO_REDIRECT_ADDR"
	CLINoColorEnv           = "BAO_CLI_NO_COLOR"
	RateLimitEnv            = "BAO_RATE_LIMIT"
	RoleEnv                 = "BAO_ROLE"
	PathEnv                 = "BAO_PATH"
	AuthMethodEnv           = "BAO_AUTH_METHOD"
	TransitKeyIDEnv         = "BAO_TRANSIT_KEY_ID"
	TransitPathEnv          = "BAO_TRANSIT_PATH"
	TransitBatchSizeEnv     = "BAO_TRANSIT_BATCH_SIZE"
	IgnoreMissingSecretsEnv = "BAO_IGNORE_MISSING_SECRETS"
	PassthroughEnv          = "BAO_PASSTHROUGH"
	LogLevelEnv             = "BAO_LOG_LEVEL"
	RevokeTokenEnv          = "BAO_REVOKE_TOKEN"
	FromPathEnv             = "BAO_FROM_PATH"
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
	LogLevelEnv:             {login: false},
	RevokeTokenEnv:          {login: false},
	FromPathEnv:             {login: false},
}

func LoadConfig() (*Config, error) {
	var (
		role, authPath, authMethod      string
		hasRole, hasPath, hasAuthMethod bool
	)

	// This workaround is necessary because the BAO_ADDR
	// is not yet used directly by the Bao client.
	// This is why env_store.go/workaroundForBao() has been implemented.
	baoAddr := os.Getenv(AddrEnv)
	os.Setenv("VAULT_ADDR", baoAddr)

	// The login procedure takes the token from a file (if using Bao Agent)
	// or requests one for itself (Kubernetes Auth, or GCP, etc...),
	// so if we got a BAO_TOKEN for the special value with "bao:login"
	baoToken := os.Getenv(TokenEnv)
	isLogin := baoToken == baoLogin
	tokenFile, ok := os.LookupEnv(TokenFileEnv)
	if ok {
		// load token from bao-agent .bao-token or injected webhook
		tokenFileContent, err := os.ReadFile(tokenFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read token file %s: %w", tokenFile, err)
		}
		baoToken = string(tokenFileContent)
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
		_ = os.Setenv(TokenEnv, baoLogin)
		passthroughEnvVars = append(passthroughEnvVars, TokenEnv)
	}

	// do not sanitize env vars specified in BAO_PASSTHROUGH
	for _, envVar := range passthroughEnvVars {
		if trimmed := strings.TrimSpace(envVar); trimmed != "" {
			delete(sanitizeEnvmap, trimmed)
		}
	}

	return &Config{
		IsLogin:              isLogin,
		Token:                baoToken,
		TokenFile:            tokenFile,
		Role:                 role,
		AuthPath:             authPath,
		AuthMethod:           authMethod,
		TransitKeyID:         os.Getenv(TransitKeyIDEnv),
		TransitPath:          os.Getenv(TransitPathEnv),
		TransitBatchSize:     cast.ToInt(os.Getenv(TransitBatchSizeEnv)),
		IgnoreMissingSecrets: cast.ToBool(os.Getenv(IgnoreMissingSecretsEnv)), // Used both for reading secrets and transit encryption
		FromPath:             os.Getenv(FromPathEnv),
		RevokeToken:          cast.ToBool(os.Getenv(RevokeTokenEnv)),
	}, nil
}
