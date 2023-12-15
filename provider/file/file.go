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
	"strings"

	"github.com/bank-vaults/secret-init/provider"
)

const ProviderName = "file"

type Provider struct {
	fs fs.FS
}

func NewProvider(config *Config) (provider.Provider, error) {
	fs := os.DirFS(config.MountPath)

	return &Provider{fs: fs}, nil
}

func (p *Provider) LoadSecrets(_ context.Context, paths []string) ([]provider.Secret, error) {
	var secrets []provider.Secret

	for _, path := range paths {
		secret, err := p.getSecretFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret from file: %w", err)
		}

		secrets = append(secrets, provider.Secret{
			Path:  path,
			Value: secret,
		})
	}

	return secrets, nil
}

func (p *Provider) getSecretFromFile(path string) (string, error) {
	path = strings.TrimLeft(path, "/")
	fmt.Println("file path:", path, "fs:", p.fs)
	content, err := fs.ReadFile(p.fs, path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}
