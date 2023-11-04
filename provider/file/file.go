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

	"github.com/bank-vaults/secret-init/provider"
)

type Provider struct {
	SecretsFilePath string
}

func NewFileProvider(secretsFilePath string) provider.Provider {

	return &Provider{SecretsFilePath: secretsFilePath}
}

func (provider *Provider) LoadSecrets(_ context.Context, paths []string) ([]string, error) {

	return make([]string, 2), nil
}
