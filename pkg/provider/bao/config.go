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

	tokenEnv                = "BAO_TOKEN"
	tokenFileEnv            = "BAO_TOKEN_FILE"
	addrEnv                 = "BAO_ADDR"
	agentAddrEnv            = "BAO_AGENT_ADDR"
	caCertEnv               = "BAO_CACERT"
	caPathEnv               = "BAO_CAPATH"
	clientCertEnv           = "BAO_CLIENT_CERT"
	clientKeyEnv            = "BAO_CLIENT_KEY"
	clientTimeoutEnv        = "BAO_CLIENT_TIMEOUT"
	srvLookupEnv            = "BAO_SRV_LOOKUP"
	skipVerifyEnv           = "BAO_SKIP_VERIFY"
	namespaceEnv            = "BAO_NAMESPACE"
	tlsServerNameEnv        = "BAO_TLS_SERVER_NAME"
	wrapTTLEnv              = "BAO_WRAP_TTL"
	mfaEnv                  = "BAO_MFA"
	maxRetriesEnv           = "BAO_MAX_RETRIES"
	clusterAddrEnv          = "BAO_CLUSTER_ADDR"
	redirectAddrEnv         = "BAO_REDIRECT_ADDR"
	cliNoColorEnv           = "BAO_CLI_NO_COLOR"
	rateLimitEnv            = "BAO_RATE_LIMIT"
	roleEnv                 = "BAO_ROLE"
	pathEnv                 = "BAO_PATH"
	authMethodEnv           = "BAO_AUTH_METHOD"
	transitKeyIDEnv         = "BAO_TRANSIT_KEY_ID"
	transitPathEnv          = "BAO_TRANSIT_PATH"
	transitBatchSizeEnv     = "BAO_TRANSIT_BATCH_SIZE"
	ignoreMissingSecretsEnv = "BAO_IGNORE_MISSING_SECRETS"
	passthroughEnv          = "BAO_PASSTHROUGH"
	logLevelEnv             = "BAO_LOG_LEVEL"
	revokeTokenEnv          = "BAO_REVOKE_TOKEN"
	fromPathEnv             = "BAO_FROM_PATH"
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
	fromPathEnv:             {login: false},
}

func LoadConfig() (*Config, error) {
	var (
		role, authPath, authMethod      string
		hasRole, hasPath, hasAuthMethod bool
	)

	// This workaround is necessary because the BAO_ADDR
	// is not yet used directly by the Bao client.
	// This is why env_store.go/workaroundForBao() has been implemented.
	baoAddr := os.Getenv(addrEnv)
	os.Setenv("VAULT_ADDR", baoAddr)

	// The login procedure takes the token from a file (if using Bao Agent)
	// or requests one for itself (Kubernetes Auth, or GCP, etc...),
	// so if we got a BAO_TOKEN for the special value with "bao:login"
	baoToken := os.Getenv(tokenEnv)
	isLogin := baoToken == baoLogin
	tokenFile, ok := os.LookupEnv(tokenFileEnv)
	if ok {
		// load token from bao-agent .bao-token or injected webhook
		tokenFileContent, err := os.ReadFile(tokenFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read token file %s: %w", tokenFile, err)
		}
		baoToken = string(tokenFileContent)
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
		_ = os.Setenv(tokenEnv, baoLogin)
		passthroughEnvVars = append(passthroughEnvVars, tokenEnv)
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
		TransitKeyID:         os.Getenv(transitKeyIDEnv),
		TransitPath:          os.Getenv(transitPathEnv),
		TransitBatchSize:     cast.ToInt(os.Getenv(transitBatchSizeEnv)),
		IgnoreMissingSecrets: cast.ToBool(os.Getenv(ignoreMissingSecretsEnv)), // Used both for reading secrets and transit encryption
		FromPath:             os.Getenv(fromPathEnv),
		RevokeToken:          cast.ToBool(os.Getenv(revokeTokenEnv)),
	}, nil
}
