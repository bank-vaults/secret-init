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
	defer func() {
		if err := os.Remove(tokenFile); err != nil {
			t.Fatalf("Failed to remove secret file: %v", err)
		}
	}()

	tests := []struct {
		name       string
		env        map[string]string
		wantConfig *Config
		err        error
	}{
		{
			name: "Valid login configuration with Token",
			env: map[string]string{
				addrEnv:                 "http://127.0.0.1:8200",
				tokenEnv:                vaultLogin,
				tokenFileEnv:            tokenFile,
				passthroughEnv:          agentAddrEnv + ", " + cliNoColorEnv,
				transitKeyIDEnv:         "test-key",
				transitPathEnv:          "transit",
				transitBatchSizeEnv:     "10",
				ignoreMissingSecretsEnv: "true",
				revokeTokenEnv:          "true",
				FromPathEnv:             "secret/data/test",
			},
			wantConfig: &Config{
				IsLogin:              true,
				Addr:                 "http://127.0.0.1:8200",
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
				tokenEnv:      vaultLogin,
				roleEnv:       "test-app-role",
				pathEnv:       "auth/approle/test/login",
				authMethodEnv: "test-approle",
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
				tokenFileEnv: tokenFile + "/invalid",
			},
			err: fmt.Errorf("failed to read token file %s/invalid: open %s/invalid: not a directory", tokenFile, tokenFile),
		},
		{
			name: "Invalid login configuration using role/path - missing role",
			env: map[string]string{
				pathEnv:       "auth/approle/test/login",
				authMethodEnv: "k8s",
			},
			err: fmt.Errorf("incomplete authentication configuration: VAULT_ROLE missing"),
		},
		{
			name: "Invalid login configuration using role/path - missing path",
			env: map[string]string{
				roleEnv:       "test-app-role",
				authMethodEnv: "k8s",
			},
			err: fmt.Errorf("incomplete authentication configuration: VAULT_PATH missing"),
		},
		{
			name: "Invalid login configuration using role/path - missing auth method",
			env: map[string]string{
				roleEnv: "test-app-role",
				pathEnv: "auth/approle/test/login",
			},
			err: fmt.Errorf("incomplete authentication configuration: VAULT_AUTH_METHOD missing"),
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			for envKey, envVal := range ttp.env {
				if err := os.Setenv(envKey, envVal); err != nil {
					t.Fatalf("Failed to set environment variable %s: %v", envKey, err)
				}
			}
			t.Cleanup(func() {
				os.Clearenv()
			})

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
	defer func() {
		if err := tokenFile.Close(); err != nil {
			t.Fatalf("Failed to close token file: %v", err)
		}
	}()

	_, err = tokenFile.Write([]byte("root"))
	assert.Nil(t, err, "Failed to write to a temporary token file")

	return tokenFile.Name()
}
