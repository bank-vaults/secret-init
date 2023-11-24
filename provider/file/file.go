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

	"github.com/bank-vaults/secret-init/provider"
)

const ProviderName = "file"

type Provider struct {
	fs fs.FS
}

func NewProvider(fs fs.FS) (provider.Provider, error) {
	if fs == nil {
		return nil, fmt.Errorf("file system is nil")
	}

	isEmpty, err := isFileSystemEmpty(fs)
	if err != nil {
		return nil, fmt.Errorf("failed to check if file system is empty: %w", err)
	}
	if isEmpty {
		return nil, fmt.Errorf("file system is empty")
	}

	return &Provider{fs: fs}, nil
}

func (provider *Provider) LoadSecrets(_ context.Context, paths []string) ([]string, error) {
	var secrets []string

	for i, path := range paths {
		secret, err := provider.getSecretFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret from file: %w", err)
		}
		// Add the secret path with a "|" separator character
		// to the secrets slice along with the secret
		// so later we can match it to the environment key
		secrets = append(secrets, paths[i]+"|"+secret)
	}

	return secrets, nil
}

func isFileSystemEmpty(fsys fs.FS) (bool, error) {
	dir, err := fs.ReadDir(fsys, ".")
	fmt.Println(dir, err)
	if err != nil {
		return false, err
	}

	for _, entry := range dir {
		if entry.IsDir() || entry.Type().IsRegular() {
			return false, nil
		}
	}

	return true, nil
}

func (provider *Provider) getSecretFromFile(path string) (string, error) {
	content, err := provider.readFile(path)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func (provider *Provider) readFile(path string) ([]byte, error) {
	content, err := fs.ReadFile(provider.fs, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return content, nil
}
