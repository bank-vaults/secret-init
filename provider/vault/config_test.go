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
	"os"
	"path/filepath"
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
		wantErr    bool
	}{
		{
			name: "Valid login configuration with Token",
			env: map[string]string{
				"VAULT_TOKEN":                  vaultLogin,
				"VAULT_TOKEN_FILE":             tokenFile,
				"VAULT_PASSTHROUGH":            "VAULT_AGENT_ADDR, VAULT_CLI_NO_COLOR",
				"VAULT_TRANSIT_KEY_ID":         "test-key",
				"VAULT_TRANSIT_PATH":           "transit",
				"VAULT_TRANSIT_BATCH_SIZE":     "10",
				"SECRET_INIT_DAEMON":           "true",
				"VAULT_IGNORE_MISSING_SECRETS": "true",
				"VAULT_REVOKE_TOKEN":           "true",
				"VAULT_FROM_PATH":              "secret/data/test",
			},
			wantConfig: &Config{
				Islogin:              true,
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
			wantErr: false,
		},
		{
			name: "Valid login configuration with Role and Path",
			env: map[string]string{
				"VAULT_TOKEN":       vaultLogin,
				"VAULT_ROLE":        "test-app-role",
				"VAULT_PATH":        "auth/approle/test/login",
				"VAULT_AUTH_METHOD": "test-approle",
			},
			wantConfig: &Config{
				Islogin:    true,
				Token:      vaultLogin,
				Role:       "test-app-role",
				AuthPath:   "auth/approle/test/login",
				AuthMethod: "test-approle",
			},
			wantErr: false,
		},
		{
			name: "Invalid login configuration missing token file",
			env: map[string]string{
				"VAULT_TOKEN_FILE": tokenFile + "/invalid",
			},
			wantConfig: nil,
			wantErr:    true,
		},
		{
			name: "Invalid login configuration missing role - path credentials",
			env: map[string]string{
				"VAULT_PATH":        "auth/approle/test/login",
				"VAULT_AUTH_METHOD": "test-approle",
			},
			wantConfig: nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			for envKey, envVal := range ttp.env {
				os.Setenv(envKey, envVal)
			}

			config, err := NewConfig()

			assert.Equal(t, ttp.wantErr, err != nil, "Unexpected error status")
			assert.Equal(t, ttp.wantConfig, config, "Unexpected config")

			// unset envs for the next test
			for envKey := range ttp.env {
				os.Unsetenv(envKey)
			}
		})
	}
}

func newTokenFile(t *testing.T) string {
	tokenFilePath := filepath.Join(t.TempDir(), "vault-token")
	tokenFile, err := os.Create(tokenFilePath)
	if err != nil {
		t.Fatalf("Failed to create a temporary token file: %v", err)
	}

	_, err = tokenFile.Write([]byte("root"))
	if err != nil {
		t.Fatalf("Failed to write to a temporary token file: %v", err)
	}

	return tokenFile.Name()
}
