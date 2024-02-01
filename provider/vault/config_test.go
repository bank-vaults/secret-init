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
				TokenEnv:                vaultLogin,
				TokenFileEnv:            tokenFile,
				PassthroughEnv:          AgentAddrEnv + ", " + CLINoColorEnv,
				TransitKeyIDEnv:         "test-key",
				TransitPathEnv:          "transit",
				TransitBatchSizeEnv:     "10",
				IgnoreMissingSecretsEnv: "true",
				RevokeTokenEnv:          "true",
				FromPathEnv:             "secret/data/test",
			},
			wantConfig: &Config{
				IsLogin:              true,
				Token:                "root",
				TokenFile:            tokenFile,
				TransitKeyID:         "test-key",
				TransitPath:          "transit",
				TransitBatchSize:     10,
				IgnoreMissingSecrets: true,
				FromPath:             "secret/data/test",
				RevokeToken:          true,
			},
		},
		{
			name: "Valid login configuration with Role and Path",
			env: map[string]string{
				TokenEnv:      vaultLogin,
				RoleEnv:       "test-app-role",
				PathEnv:       "auth/approle/test/login",
				AuthMethodEnv: "test-approle",
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
			name: "Invalid login configuration using tokenfile - missing token file",
			env: map[string]string{
				TokenFileEnv: tokenFile + "/invalid",
			},
			err: fmt.Errorf("failed to read token file " + tokenFile + "/invalid: open " + tokenFile + "/invalid: not a directory"),
		},
		{
			name: "Invalid login configuration using role/path - missing role",
			env: map[string]string{
				PathEnv:       "auth/approle/test/login",
				AuthMethodEnv: "k8s",
			},
			err: fmt.Errorf("incomplete authentication configuration: VAULT_ROLE missing"),
		},
		{
			name: "Invalid login configuration using role/path - missing path",
			env: map[string]string{
				RoleEnv:       "test-app-role",
				AuthMethodEnv: "k8s",
			},
			err: fmt.Errorf("incomplete authentication configuration: VAULT_PATH missing"),
		},
		{
			name: "Invalid login configuration using role/path - missing auth method",
			env: map[string]string{
				RoleEnv: "test-app-role",
				PathEnv: "auth/approle/test/login",
			},
			err: fmt.Errorf("incomplete authentication configuration: VAULT_AUTH_METHOD missing"),
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			for envKey, envVal := range ttp.env {
				os.Setenv(envKey, envVal)
			}
			defer os.Clearenv()

			config, err := LoadConfig()
			if err != nil {
				assert.EqualError(t, ttp.err, err.Error(), "Unexpected error message")
			}

			if ttp.wantConfig != nil {
				assert.Equal(t, ttp.wantConfig, config, "Unexpected config")
			}
		})
	}
}

func newTokenFile(t *testing.T) string {
	tokenFile, err := os.CreateTemp("", "vault-token")
	assert.Nil(t, err, "Failed to create a temporary token file")
	defer tokenFile.Close()

	_, err = tokenFile.Write([]byte("root"))
	assert.Nil(t, err, "Failed to write to a temporary token file")

	return tokenFile.Name()
}
