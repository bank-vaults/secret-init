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
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"github.com/bank-vaults/secret-init/provider"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		wantErr  bool
		wantType bool
	}{
		{
			name: "Valid config",
			config: &Config{
				MountPath: "test/secrets",
			},
			wantErr:  false,
			wantType: true,
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			provider, err := NewProvider(ttp.config)
			assert.Equal(t, ttp.wantErr, err != nil, "Unexpected error status")
			assert.Equal(t, ttp.wantType, provider != nil, "Unexpected provider type")
		})
	}
}

func TestLoadSecrets(t *testing.T) {
	tests := []struct {
		name     string
		paths    []string
		wantErr  bool
		wantData []provider.Secret
	}{
		{
			name: "Load secrets successfully",
			paths: []string{
				"test/secrets/sqlpass.txt",
				"test/secrets/awsaccess.txt",
				"test/secrets/awsid.txt",
			},
			wantErr: false,
			wantData: []provider.Secret{
				{Path: "test/secrets/sqlpass.txt", Value: "3xtr3ms3cr3t"},
				{Path: "test/secrets/awsaccess.txt", Value: "s3cr3t"},
				{Path: "test/secrets/awsid.txt", Value: "secretId"},
			},
		},
		{
			name: "Fail to load secrets due to invalid path",
			paths: []string{
				"test/secrets/mistake/sqlpass.txt",
				"test/secrets/mistake/awsaccess.txt",
				"test/secrets/mistake/awsid.txt",
			},
			wantErr:  true,
			wantData: nil,
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
			assert.Equal(t, ttp.wantErr, err != nil, "Unexpected error status")
			assert.Equal(t, ttp.wantData, secrets, "Unexpected secrets")
		})
	}
}
