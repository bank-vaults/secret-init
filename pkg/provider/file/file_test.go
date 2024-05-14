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

package file

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"github.com/bank-vaults/secret-init/pkg/provider"
)

func TestNewProvider(t *testing.T) {
	tempDir := t.TempDir()
	secretFile := newSecretFile(t, "3xtr3ms3cr3t")
	defer os.Remove(secretFile)

	tests := []struct {
		name     string
		config   *Config
		err      error
		wantType bool
		wantFs   fs.FS
	}{
		{
			name: "Valid config - directory",
			config: &Config{
				MountPath: tempDir,
			},
			wantType: true,
			wantFs:   os.DirFS(tempDir),
		},
		{
			name: "Invalid config - directory does not exist",
			config: &Config{
				MountPath: "test/secrets/invalid",
			},
			err: fmt.Errorf("failed to access path: stat test/secrets/invalid: no such file or directory"),
		},
		{
			name: "Invalid config - file instead of directory",
			config: &Config{
				MountPath: secretFile,
			},
			err: fmt.Errorf("provided path is not a directory"),
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			provider, err := NewProvider(ttp.config)
			if err != nil {
				assert.EqualError(t, err, ttp.err.Error(), "Unexpected error message")
			}
			if ttp.wantType {
				assert.Equal(t, ttp.wantType, provider != nil, "Unexpected provider type")

				if ttp.wantFs != nil {
					assert.Equal(t, ttp.wantFs, provider.(*Provider).fs, "Unexpected file system")
				}
			}
		})
	}
}

func TestLoadSecrets(t *testing.T) {
	tests := []struct {
		name        string
		paths       []string
		err         error
		wantSecrets []provider.Secret
	}{
		{
			name: "Load secrets successfully",
			paths: []string{
				"MYSQL_PASSWORD=file:test/secrets/sqlpass.txt",
				"AWS_SECRET_ACCESS_KEY=file:test/secrets/awsaccess.txt",
				"AWS_ACCESS_KEY_ID=file:test/secrets/awsid.txt",
			},
			wantSecrets: []provider.Secret{
				{Key: "MYSQL_PASSWORD", Value: "3xtr3ms3cr3t"},
				{Key: "AWS_SECRET_ACCESS_KEY", Value: "s3cr3t"},
				{Key: "AWS_ACCESS_KEY_ID", Value: "secretId"},
			},
		},
		{
			name: "Fail to load secrets due to invalid path",
			paths: []string{
				"MYSQL_PASSWORD=file:test/secrets/mistake/sqlpass.txt",
				"AWS_SECRET_ACCESS_KEY=file:test/secrets/mistake/awsaccess.txt",
				"AWS_ACCESS_KEY_ID=file:test/secrets/mistake/awsid.txt",
			},
			err: fmt.Errorf("failed to get secret from file: failed to read file: open test/secrets/mistake/sqlpass.txt: file does not exist"),
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			fs := fstest.MapFS{
				"test/secrets/sqlpass.txt":   {Data: []byte("3xtr3ms3cr3t")},
				"test/secrets/awsaccess.txt": {Data: []byte("s3cr3t")},
				"test/secrets/awsid.txt":     {Data: []byte("secretId")},
			}
			provider := Provider{fs: fs}
			secrets, err := provider.LoadSecrets(context.Background(), ttp.paths)
			if err != nil {
				assert.EqualError(t, ttp.err, err.Error(), "Unexpected error message")
			}
			if ttp.wantSecrets != nil {
				assert.ElementsMatch(t, ttp.wantSecrets, secrets, "Unexpected secrets")
			}
		})
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
