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

package envstore

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bank-vaults/secret-init/provider"
	"github.com/bank-vaults/secret-init/provider/file"
	"github.com/bank-vaults/secret-init/provider/vault"
)

func TestNewEnvStore(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantEnv map[string]string
	}{
		{
			name: "Non-empty environment",
			env: map[string]string{
				"MYSQL_PASSWORD":        "vault:secret/data/test/mysql#MYSQL_PASSWORD",
				"AWS_SECRET_ACCESS_KEY": "vault:secret/data/test/aws#AWS_SECRET_ACCESS_KEY",
			},
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			for envKey, envVal := range ttp.env {
				os.Setenv(envKey, envVal)
			}

			envStore := NewEnvStore()

			assert.Contains(t, envStore.data, "MYSQL_PASSWORD", "MYSQL_PASSWORD not found in envStore")
			assert.Contains(t, envStore.data, "AWS_SECRET_ACCESS_KEY", "AWS_SECRET_ACCESS_KEY not found in envStore")
		})
	}
}

func TestEnvStore_GetPathsFor(t *testing.T) {
	tests := []struct {
		name      string
		provider  provider.Provider
		wantPaths []string
		err       error
	}{
		{
			name:     "Vault provider",
			provider: &vault.Provider{},
			wantPaths: []string{
				"MYSQL_PASSWORD=vault:secret/data/test/mysql#MYSQL_PASSWORD",
				"AWS_SECRET_ACCESS_KEY=vault:secret/data/test/aws#AWS_SECRET_ACCESS_KEY",
			},
		},
		{
			name:     "File provider",
			provider: &file.Provider{},
			wantPaths: []string{
				"test/secrets/mysql.txt",
				"test/secrets/aws.txt",
			},
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			createEnvsForProvider(ttp.provider)

			envStore := NewEnvStore()

			paths, err := envStore.GetPathsFor(ttp.provider)
			if err != nil {
				assert.EqualError(t, err, ttp.err.Error(), "Unexpected error message")
			}

			if ttp.wantPaths != nil {
				assert.Contains(t, paths, ttp.wantPaths[0], "Unexpected path")
				assert.Contains(t, paths, ttp.wantPaths[1], "Unexpected path")
			}
		})
	}
}

func TestEnvStore_GetProviderSecrets(t *testing.T) {
	tests := []struct {
		name        string
		provider    provider.Provider
		secrets     []provider.Secret
		wantSecrets []string
		err         error
	}{
		{
			name:     "Vault provider",
			provider: &vault.Provider{},
			secrets: []provider.Secret{
				{
					Path:  "MYSQL_PASSWORD",
					Value: "3xtr3ms3cr3t",
				},
				{
					Path:  "AWS_SECRET_ACCESS_KEY",
					Value: "s3cr3t",
				},
			},
			wantSecrets: []string{
				"MYSQL_PASSWORD=3xtr3ms3cr3t",
				"AWS_SECRET_ACCESS_KEY=s3cr3t",
			},
		},
		{
			name:     "File provider",
			provider: &file.Provider{},
			secrets: []provider.Secret{
				{
					Path:  "test/secrets/mysql.txt",
					Value: "3xtr3ms3cr3t",
				},
				{
					Path:  "test/secrets/aws.txt",
					Value: "s3cr3t",
				},
			},
			wantSecrets: []string{
				"MYSQL_PASSWORD=3xtr3ms3cr3t",
				"AWS_SECRET_ACCESS_KEY=s3cr3t",
			},
		},
		{
			name:     "File provider - missing environment variable",
			provider: &file.Provider{},
			secrets: []provider.Secret{
				{
					Path:  "test/secrets/mysql.txt",
					Value: "3xtr3ms3cr3t",
				},
				{
					Path:  "test/secrets/aws/invalid",
					Value: "s3cr3t",
				},
			},
			err: fmt.Errorf("failed to find environment variable key for secret path: test/secrets/aws/invalid"),
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			createEnvsForProvider(ttp.provider)

			envStore := NewEnvStore()

			secretsEnv, err := envStore.GetProviderSecrets(ttp.provider, ttp.secrets)
			if err != nil {
				assert.EqualError(t, err, ttp.err.Error(), "Unexpected error message")
			}

			if ttp.wantSecrets != nil {
				assert.Equal(t, ttp.wantSecrets, secretsEnv, "Unexpected secrets")
			}
		})
	}
}

func createEnvsForProvider(provider provider.Provider) {
	switch provider.GetProviderName() {
	case vault.ProviderName:
		os.Setenv("MYSQL_PASSWORD", "vault:secret/data/test/mysql#MYSQL_PASSWORD")
		os.Setenv("AWS_SECRET_ACCESS_KEY", "vault:secret/data/test/aws#AWS_SECRET_ACCESS_KEY")
	case file.ProviderName:
		os.Setenv("MYSQL_PASSWORD", "file:test/secrets/mysql.txt")
		os.Setenv("AWS_SECRET_ACCESS_KEY", "file:test/secrets/aws.txt")
	}
}
