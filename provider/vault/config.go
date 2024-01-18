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

	"github.com/bank-vaults/secret-init/common"
)

// The special value for SECRET_INIT which marks that the login token needs to be passed through to the application
// which was acquired during the vault client initialization.
const vaultLogin = "vault:login"

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
}

type envType struct {
	login bool
}

var sanitizeEnvmap = map[string]envType{
	common.VaultToken:                {login: true},
	common.VaultAddr:                 {login: true},
	common.VaultAgentAddr:            {login: true},
	common.VaultCACert:               {login: true},
	common.VaultCAPath:               {login: true},
	common.VaultClientCert:           {login: true},
	common.VaultClientKey:            {login: true},
	common.VaultClientTimeout:        {login: true},
	common.VaultSRVLookup:            {login: true},
	common.VaultSkipVerify:           {login: true},
	common.VaultNamespace:            {login: true},
	common.VaultTLSServerName:        {login: true},
	common.VaultWrapTTL:              {login: true},
	common.VaultMFA:                  {login: true},
	common.VaultMaxRetries:           {login: true},
	common.VaultClusterAddr:          {login: false},
	common.VaultRedirectAddr:         {login: false},
	common.VaultCLINoColor:           {login: false},
	common.VaultRateLimit:            {login: false},
	common.VaultRole:                 {login: false},
	common.VaultPath:                 {login: false},
	common.VaultAuthMethod:           {login: false},
	common.VaultTransitKeyID:         {login: false},
	common.VaultTransitPath:          {login: false},
	common.VaultTransitBatchSize:     {login: false},
	common.VaultIgnoreMissingSecrets: {login: false},
	common.VaultPassthrough:          {login: false},
	common.VaultRevokeToken:          {login: false},
	common.VaultFromPath:             {login: false},
	common.SecretInitDaemon:          {login: false},
}

func NewConfig() (*Config, error) {
	var (
		role, authPath, authMethod      string
		hasRole, hasPath, hasAuthMethod bool
	)

	// The login procedure takes the token from a file (if using Vault Agent)
	// or requests one for itself (Kubernetes Auth, or GCP, etc...),
	// so if we got a VAULT_TOKEN for the special value with "vault:login"
	vaultToken := os.Getenv(common.VaultToken)
	isLogin := vaultToken == vaultLogin
	tokenFile, ok := os.LookupEnv(common.VaultTokenFile)
	if ok {
		// load token from vault-agent .vault-token or injected webhook
		if b, err := os.ReadFile(tokenFile); err == nil {
			vaultToken = string(b)
		} else {
			return nil, fmt.Errorf("failed to read token file %s: %w", tokenFile, err)
		}
	} else {
		if isLogin {
			_ = os.Unsetenv(common.VaultToken)
		}
		// will use role/path based authentication
		role, hasRole = os.LookupEnv(common.VaultRole)
		authPath, hasPath = os.LookupEnv(common.VaultPath)
		authMethod, hasAuthMethod = os.LookupEnv(common.VaultAuthMethod)
		if !hasRole || !hasPath || !hasAuthMethod {
			return nil, fmt.Errorf("incomplete authentication configuration %s, %s, and %s",
				"VAULT_ROLE", "VAULT_PATH", "VAULT_AUTH_METHOD")
		}
	}

	passthroughEnvVars := strings.Split(os.Getenv(common.VaultPassthrough), ",")
	if isLogin {
		_ = os.Setenv(common.VaultToken, vaultLogin)
		passthroughEnvVars = append(passthroughEnvVars, common.VaultToken)
	}

	// do not sanitize env vars specified in VAULT_PASSTHROUGH
	for _, envVar := range passthroughEnvVars {
		if trimmed := strings.TrimSpace(envVar); trimmed != "" {
			delete(sanitizeEnvmap, trimmed)
		}
	}

	// injector configuration
	transitKeyID := os.Getenv(common.VaultTransitKeyID)
	transitPath := os.Getenv(common.VaultTransitPath)
	transitBatchSize := cast.ToInt(os.Getenv(common.VaultTransitBatchSize))
	daemonMode := cast.ToBool(os.Getenv(common.SecretInitDaemon))
	// Used both for reading secrets and transit encryption
	ignoreMissingSecrets := cast.ToBool(os.Getenv(common.VaultIgnoreMissingSecrets))

	fromPath := os.Getenv(common.VaultFromPath)
	revokeToken := cast.ToBool(os.Getenv(common.VaultRevokeToken))

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
	}, nil
}
