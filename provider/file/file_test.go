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
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"

	"github.com/bank-vaults/secret-init/provider"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name     string
		fs       fs.FS
		wantErr  bool
		wantType bool
	}{
		{
			name: "Valid file system",
			fs: fstest.MapFS{
				"test/secrets/sqlpass.txt":   &fstest.MapFile{Data: []byte("3xtr3ms3cr3t")},
				"test/secrets/awsaccess.txt": &fstest.MapFile{Data: []byte("s3cr3t")},
				"test/secrets/awsid.txt":     &fstest.MapFile{Data: []byte("secretId")},
			},
			wantErr:  false,
			wantType: true,
		},
		{
			name:     "Nil file system",
			fs:       nil,
			wantErr:  true,
			wantType: false,
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			prov, err := NewProvider(ttp.fs)
			if (err != nil) != ttp.wantErr {
				t.Fatalf("NewProvider() error = %v, wantErr %v", err, ttp.wantErr)
				return
			}
			// Use type assertion to check if the provider is of the correct type
			_, ok := prov.(*Provider)
			if ok != ttp.wantType {
				t.Fatalf("NewProvider() = %v, wantType %v", ok, ttp.wantType)
			}
		})
	}
}

func TestLoadSecrets(t *testing.T) {
	tests := []struct {
		name     string
		fs       fs.FS
		paths    []string
		wantErr  bool
		wantData []provider.Secret
	}{
		{
			name: "Load secrets successfully",
			fs: fstest.MapFS{
				"test/secrets/sqlpass.txt":   &fstest.MapFile{Data: []byte("3xtr3ms3cr3t")},
				"test/secrets/awsaccess.txt": &fstest.MapFile{Data: []byte("s3cr3t")},
				"test/secrets/awsid.txt":     &fstest.MapFile{Data: []byte("secretId")},
			},
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
			fs: fstest.MapFS{
				"test/secrets/sqlpass.txt":   &fstest.MapFile{Data: []byte("3xtr3ms3cr3t")},
				"test/secrets/awsaccess.txt": &fstest.MapFile{Data: []byte("s3cr3t")},
				"test/secrets/awsid.txt":     &fstest.MapFile{Data: []byte("secretId")},
			},
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
			provider, err := NewProvider(ttp.fs)
			if assert.NoError(t, err, "Unexpected error") {
				secrets, err := provider.LoadSecrets(context.Background(), ttp.paths)
				assert.Equal(t, ttp.wantErr, err != nil, "Unexpected error status")
				assert.ElementsMatch(t, ttp.wantData, secrets, "Unexpected secrets loaded")
			}
		})
	}
}
