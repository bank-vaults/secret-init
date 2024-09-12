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
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bank-vaults/secret-init/pkg/common"
	"github.com/bank-vaults/secret-init/pkg/provider"
)

func TestEnvStore_GetSecretReferences(t *testing.T) {
	tests := []struct {
		name      string
		envs      map[string]string
		wantPaths map[string][]string
	}{
		{
			name: "file provider",
			envs: map[string]string{
				"AWS_SECRET_ACCESS_KEY_ID": "file:secret/data/test/aws",
			},
			wantPaths: map[string][]string{
				"file": {
					"AWS_SECRET_ACCESS_KEY_ID=file:secret/data/test/aws",
				},
			},
		},
		{
			name: "vault provider",
			envs: map[string]string{
				"ACCOUNT_PASSWORD_1":              "vault:secret/data/account#password#1",
				"ACCOUNT_PASSWORD":                "vault:secret/data/account#password",
				"ROOT_CERT":                       ">>vault:pki/root/generate/internal#certificate",
				"ROOT_CERT_CACHED":                ">>vault:pki/root/generate/internal#certificate",
				"INLINE_SECRET":                   "scheme://${vault:secret/data/account#username}:${vault:secret/data/account#password}@127.0.0.1:8080",
				"INLINE_SECRET_EMBEDDED_TEMPLATE": "scheme://${vault:secret/data/account#username}:${vault:secret/data/account#${.password | urlquery}}@127.0.0.1:8080",
				"INLINE_DYNAMIC_SECRET":           "${>>vault:pki/root/generate/internal#certificate}__${>>vault:pki/root/generate/internal#certificate}",
			},
			wantPaths: map[string][]string{
				"vault": {
					"ACCOUNT_PASSWORD_1=vault:secret/data/account#password#1",
					"ACCOUNT_PASSWORD=vault:secret/data/account#password",
					"ROOT_CERT=>>vault:pki/root/generate/internal#certificate",
					"ROOT_CERT_CACHED=>>vault:pki/root/generate/internal#certificate",
					"INLINE_SECRET=scheme://${vault:secret/data/account#username}:${vault:secret/data/account#password}@127.0.0.1:8080",
					"INLINE_SECRET_EMBEDDED_TEMPLATE=scheme://${vault:secret/data/account#username}:${vault:secret/data/account#${.password | urlquery}}@127.0.0.1:8080",
					"INLINE_DYNAMIC_SECRET=${>>vault:pki/root/generate/internal#certificate}__${>>vault:pki/root/generate/internal#certificate}",
				},
			},
		},
		{
			name: "bao provider",
			envs: map[string]string{
				"ACCOUNT_PASSWORD_1":              "bao:secret/data/account#password#1",
				"ACCOUNT_PASSWORD":                "bao:secret/data/account#password",
				"ROOT_CERT":                       ">>bao:pki/root/generate/internal#certificate",
				"ROOT_CERT_CACHED":                ">>bao:pki/root/generate/internal#certificate",
				"INLINE_SECRET":                   "scheme://${bao:secret/data/account#username}:${bao:secret/data/account#password}@127.0.0.1:8080",
				"INLINE_SECRET_EMBEDDED_TEMPLATE": "scheme://${bao:secret/data/account#username}:${bao:secret/data/account#${.password | urlquery}}@127.0.0.1:8080",
				"INLINE_DYNAMIC_SECRET":           "${>>bao:pki/root/generate/internal#certificate}__${>>bao:pki/root/generate/internal#certificate}",
			},
			wantPaths: map[string][]string{
				"bao": {
					"ACCOUNT_PASSWORD_1=bao:secret/data/account#password#1",
					"ACCOUNT_PASSWORD=bao:secret/data/account#password",
					"ROOT_CERT=>>bao:pki/root/generate/internal#certificate",
					"ROOT_CERT_CACHED=>>bao:pki/root/generate/internal#certificate",
					"INLINE_SECRET=scheme://${bao:secret/data/account#username}:${bao:secret/data/account#password}@127.0.0.1:8080",
					"INLINE_SECRET_EMBEDDED_TEMPLATE=scheme://${bao:secret/data/account#username}:${bao:secret/data/account#${.password | urlquery}}@127.0.0.1:8080",
					"INLINE_DYNAMIC_SECRET=${>>bao:pki/root/generate/internal#certificate}__${>>bao:pki/root/generate/internal#certificate}",
				},
			},
		},
		{
			name: "aws provider",
			envs: map[string]string{
				"AWS_SECRET1": "arn:aws:secretsmanager:us-west-2:123456789012:secret:my-secret",
				"AWS_SECRET2": "arn:aws:ssm:us-west-2:123456789012:parameter/my-parameter",
			},
			wantPaths: map[string][]string{
				"aws": {
					"AWS_SECRET1=arn:aws:secretsmanager:us-west-2:123456789012:secret:my-secret",
					"AWS_SECRET2=arn:aws:ssm:us-west-2:123456789012:parameter/my-parameter",
				},
			},
		},
		{
			name: "multi provider",
			envs: map[string]string{
				"AWS_SECRET_ACCESS_KEY_ID": "file:secret/data/test/aws",
				"MYSQL_PASSWORD":           "vault:secret/data/test/mysql#MYSQL_PASSWORD",
				"AWS_SECRET_ACCESS_KEY":    "vault:secret/data/test/aws#AWS_SECRET_ACCESS_KEY",
				"RABBITMQ_USERNAME":        "bao:secret/data/test/rabbitmq#RABBITMQ_USERNAME",
				"RABBITMQ_PASSWORD":        "bao:secret/data/test/rabbitmq#RABBITMQ_PASSWORD",
				"AWS_SECRET1":              "arn:aws:secretsmanager:us-west-2:123456789012:secret:my-secret",
				"AWS_SECRET2":              "arn:aws:ssm:us-west-2:123456789012:parameter/my-parameter",
			},
			wantPaths: map[string][]string{
				"file": {
					"AWS_SECRET_ACCESS_KEY_ID=file:secret/data/test/aws",
				},
				"vault": {
					"MYSQL_PASSWORD=vault:secret/data/test/mysql#MYSQL_PASSWORD",
					"AWS_SECRET_ACCESS_KEY=vault:secret/data/test/aws#AWS_SECRET_ACCESS_KEY",
				},
				"bao": {
					"RABBITMQ_USERNAME=bao:secret/data/test/rabbitmq#RABBITMQ_USERNAME",
					"RABBITMQ_PASSWORD=bao:secret/data/test/rabbitmq#RABBITMQ_PASSWORD",
				},
				"aws": {
					"AWS_SECRET1=arn:aws:secretsmanager:us-west-2:123456789012:secret:my-secret",
					"AWS_SECRET2=arn:aws:ssm:us-west-2:123456789012:parameter/my-parameter",
				},
			},
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			// prepare envs
			for envKey, envVal := range ttp.envs {
				os.Setenv(envKey, envVal)
			}
			t.Cleanup(func() {
				os.Clearenv()
			})

			paths := NewEnvStore(&common.Config{}).GetSecretReferences()

			for key, expectedSlice := range ttp.wantPaths {
				actualSlice, ok := paths[key]
				assert.True(t, ok, "Key not found in actual paths")
				assert.ElementsMatch(t, expectedSlice, actualSlice, "Slices for key %s do not match", key)
			}
		})
	}
}

func TestEnvStore_LoadProviderSecrets(t *testing.T) {
	secretFile := newSecretFile(t, "secretId")
	defer os.Remove(secretFile)

	tests := []struct {
		name                string
		providerPaths       map[string][]string
		wantProviderSecrets []provider.Secret
		addvault            bool
		err                 error
	}{
		{
			name: "Load secrets successfully",
			providerPaths: map[string][]string{
				"file": {
					"AWS_SECRET_ACCESS_KEY_ID=file:" + secretFile,
				},
			},
			wantProviderSecrets: []provider.Secret{
				{
					Key:   "AWS_SECRET_ACCESS_KEY_ID",
					Value: "secretId",
				},
			},
			addvault: false,
		},
		{
			name: "Fail to create provider",
			providerPaths: map[string][]string{
				"invalid": {
					"AWS_SECRET_ACCESS_KEY_ID=file:" + secretFile,
				},
			},
			addvault: false,
			err:      fmt.Errorf("failed to create provider invalid: provider invalid is not supported"),
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			createEnvsForProvider(ttp.addvault, secretFile)

			providerSecrets, err := NewEnvStore(&common.Config{}).LoadProviderSecrets(context.Background(), ttp.providerPaths)
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
		providerSecrets []provider.Secret
		wantSecretsEnv  []string
		addvault        bool
		err             error
	}{
		{
			name: "Convert secrets successfully",
			providerSecrets: []provider.Secret{
				{
					Key:   "AWS_SECRET_ACCESS_KEY_ID",
					Value: "secretId",
				},
			},
			wantSecretsEnv: []string{
				"AWS_SECRET_ACCESS_KEY_ID=secretId",
			},
			addvault: false,
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			createEnvsForProvider(ttp.addvault, secretFile)

			secretsEnv := NewEnvStore(&common.Config{}).ConvertProviderSecrets(ttp.providerSecrets)
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
	err := os.MkdirAll(dir, 0o755)
	assert.Nil(t, err, "Failed to create directory")

	file, err := os.CreateTemp(dir, "secret.txt")
	assert.Nil(t, err, "Failed to create a temporary file")
	defer file.Close()

	_, err = file.WriteString(content)
	assert.Nil(t, err, "Failed to write to the temporary file")

	return file.Name()
}
