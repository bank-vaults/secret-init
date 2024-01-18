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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bank-vaults/secret-init/common"
)

func TestConfig(t *testing.T) {
	tokenFile := newTokenFile(t)
	defer os.Remove(tokenFile)

	tests := []struct {
		name       string
		env        map[string]string
		wantConfig *Config
		err        error
	}{
		{
			name: "Valid login configuration with Token",
			env: map[string]string{
				common.VaultToken:                vaultLogin,
				common.VaultTokenFile:            tokenFile,
				common.VaultPassthrough:          common.VaultAgentAddr + ", " + common.VaultCLINoColor,
				common.VaultTransitKeyID:         "test-key",
				common.VaultTransitPath:          "transit",
				common.VaultTransitBatchSize:     "10",
				common.SecretInitDaemon:          "true",
				common.VaultIgnoreMissingSecrets: "true",
				common.VaultRevokeToken:          "true",
				common.VaultFromPath:             "secret/data/test",
			},
			wantConfig: &Config{
				IsLogin:              true,
				Token:                "root",
				TokenFile:            tokenFile,
				TransitKeyID:         "test-key",
				TransitPath:          "transit",
				TransitBatchSize:     10,
				DaemonMode:           true,
				IgnoreMissingSecrets: true,
				FromPath:             "secret/data/test",
				RevokeToken:          true,
			},
		},
		{
			name: "Valid login configuration with Role and Path",
			env: map[string]string{
				common.VaultToken:      vaultLogin,
				common.VaultRole:       "test-app-role",
				common.VaultPath:       "auth/approle/test/login",
				common.VaultAuthMethod: "test-approle",
			},
			wantConfig: &Config{
				IsLogin:    true,
				Token:      vaultLogin,
				Role:       "test-app-role",
				AuthPath:   "auth/approle/test/login",
				AuthMethod: "test-approle",
			},
		},
		{
			name: "Invalid login configuration missing token file",
			env: map[string]string{
				common.VaultTokenFile: tokenFile + "/invalid",
			},
			err: fmt.Errorf("failed to read token file " + tokenFile + "/invalid: open " + tokenFile + "/invalid: not a directory"),
		},
		{
			name: "Invalid login configuration using role/path auth, missing role",
			env: map[string]string{
				common.VaultPath:       "auth/approle/test/login",
				common.VaultAuthMethod: "k8s",
			},
			err: fmt.Errorf("incomplete authentication configuration: VAULT_ROLE missing"),
		},
		{
			name: "Invalid login configuration using role/path auth, missing path and auth method",
			env: map[string]string{
				common.VaultRole: "test-app-role",
			},
			err: fmt.Errorf("incomplete authentication configuration: VAULT_PATH, VAULT_AUTH_METHOD missing"),
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			for envKey, envVal := range ttp.env {
				os.Setenv(envKey, envVal)
			}

			config, err := NewConfig()
			if err != nil {
				assert.EqualError(t, err, ttp.err.Error(), "Unexpected error message")
			}
			if ttp.wantConfig != nil {
				assert.Equal(t, ttp.wantConfig, config, "Unexpected config")
			}

			// unset envs for the next test
			for envKey := range ttp.env {
				os.Unsetenv(envKey)
			}
		})
	}
}

func newTokenFile(t *testing.T) string {
	tokenFile, err := os.CreateTemp("", "vault-token")
	if err != nil {
		t.Fatalf("Failed to create a temporary token file: %v", err)
	}
	defer tokenFile.Close()

	_, err = tokenFile.Write([]byte("root"))
	if err != nil {
		t.Fatalf("Failed to write to a temporary token file: %v", err)
	}

	return tokenFile.Name()
}
