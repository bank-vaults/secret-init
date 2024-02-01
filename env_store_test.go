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

package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bank-vaults/secret-init/provider"
)

func TestEnvStore_GetProviderPaths(t *testing.T) {
	secretFile := newSecretFile(t, "secretId")
	defer os.Remove(secretFile)

	tests := []struct {
		name      string
		wantPaths map[string][]string
	}{
		{
			name: "multi provider",
			wantPaths: map[string][]string{
				"vault": {
					"MYSQL_PASSWORD=vault:secret/data/test/mysql#MYSQL_PASSWORD",
					"AWS_SECRET_ACCESS_KEY=vault:secret/data/test/aws#AWS_SECRET_ACCESS_KEY",
				},
				"file": {
					secretFile,
				},
			},
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			createEnvsForProvider(true, secretFile)
			envStore := NewEnvStore()
			paths := envStore.GetProviderPaths()

			for key, expectedSlice := range ttp.wantPaths {
				actualSlice, ok := paths[key]
				assert.True(t, ok, "Key not found in actual paths")
				assert.ElementsMatch(t, expectedSlice, actualSlice, "Slices for key %s do not match", key)
			}
		})
	}
}

func TestEnvStore_GetProviderSecrets(t *testing.T) {
	secretFile := newSecretFile(t, "secretId")
	defer os.Remove(secretFile)

	tests := []struct {
		name                string
		providerPaths       map[string][]string
		wantProviderSecrets map[string][]provider.Secret
		err                 error
	}{
		{
			name: "Load secrets successfully",
			providerPaths: map[string][]string{
				"file": {
					secretFile,
				},
			},
			wantProviderSecrets: map[string][]provider.Secret{
				"file": {
					{
						Path:  secretFile,
						Value: "secretId",
					},
				},
			},
		},
		{
			name: "Fail to create provider",
			providerPaths: map[string][]string{
				"invalid": {
					secretFile,
				},
			},
			err: fmt.Errorf("failed to create provider invalid: provider invalid is not supported"),
		},
		{
			name: "Fail to load secrets due to invalid path",
			providerPaths: map[string][]string{
				"file": {
					secretFile + "/invalid",
				},
			},
			err: fmt.Errorf("failed to load secrets for provider file: failed to get secret from file: failed to read file: open " + secretFile + "/invalid: not a directory"),
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			createEnvsForProvider(false, secretFile)
			envStore := NewEnvStore()

			providerSecrets, err := envStore.GetProviderSecrets(ttp.providerPaths)
			if err != nil {
				assert.EqualError(t, ttp.err, err.Error(), "Unexpected error message")
			}
			if ttp.wantProviderSecrets != nil {
				assert.Equal(t, ttp.wantProviderSecrets, providerSecrets, "Unexpected secrets")
			}
		})
	}
}

func TestEnvStore_ConvertProviderSecrets(t *testing.T) {
	secretFile := newSecretFile(t, "secretId")
	defer os.Remove(secretFile)

	tests := []struct {
		name            string
		providerSecrets map[string][]provider.Secret
		wantSecretsEnv  []string
		err             error
	}{
		{
			name: "Convert secrets successfully",
			providerSecrets: map[string][]provider.Secret{
				"file": {
					{
						Path:  secretFile,
						Value: "secretId",
					},
				},
			},
			wantSecretsEnv: []string{
				"AWS_SECRET_ACCESS_KEY_ID=secretId",
			},
		},
		{
			name: "Fail to convert secrets due to fail to find env-key",
			providerSecrets: map[string][]provider.Secret{
				"file": {
					{
						Path:  secretFile + "/invalid",
						Value: "secretId",
					},
				},
			},
			err: fmt.Errorf("failed to create secret environment variables: failed to find environment variable key for secret path: " + secretFile + "/invalid"),
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			createEnvsForProvider(false, secretFile)
			envStore := NewEnvStore()

			secretsEnv, err := envStore.ConvertProviderSecrets(ttp.providerSecrets)
			if err != nil {
				assert.EqualError(t, ttp.err, err.Error(), "Unexpected error message")
			}
			if ttp.wantSecretsEnv != nil {
				assert.Equal(t, ttp.wantSecretsEnv, secretsEnv, "Unexpected secrets")
			}
		})
	}
}

func createEnvsForProvider(addVault bool, secretFile string) {
	os.Setenv("AWS_SECRET_ACCESS_KEY_ID", "file:"+secretFile)
	if addVault {
		os.Setenv("MYSQL_PASSWORD", "vault:secret/data/test/mysql#MYSQL_PASSWORD")
		os.Setenv("AWS_SECRET_ACCESS_KEY", "vault:secret/data/test/aws#AWS_SECRET_ACCESS_KEY")
	}
}

func newSecretFile(t *testing.T, content string) string {
	dir := t.TempDir() + "/test/secrets"
	err := os.MkdirAll(dir, 0755)
	assert.Nil(t, err, "Failed to create directory")

	file, err := os.CreateTemp(dir, "secret.txt")
	assert.Nil(t, err, "Failed to create a temporary file")
	defer file.Close()

	_, err = file.WriteString(content)
	assert.Nil(t, err, "Failed to write to the temporary file")

	return file.Name()
}
