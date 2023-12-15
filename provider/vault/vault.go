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
	"context"

	"github.com/bank-vaults/secret-init/provider"
)

const ProviderName = "vault"

type Provider struct {
}

func NewProvider(_ *Config) (provider.Provider, error) {

	return &Provider{}, nil
}

func (p *Provider) LoadSecrets(_ context.Context, _ []string) ([]provider.Secret, error) {

	return nil, nil
}
