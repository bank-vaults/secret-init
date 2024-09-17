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

package provider

import (
	"context"

	"github.com/bank-vaults/secret-init/pkg/common"
)

type Factory struct {
	ProviderType string
	Validator    func(envValue string) bool
	Create       func(ctx context.Context, cfg *common.Config) (Provider, error)
}

// Provider is an interface for securely loading secrets based on environment variables.
type Provider interface {
	// LoadSecrets loads secrets from the provider based on the given paths
	LoadSecrets(ctx context.Context, paths []string) ([]Secret, error)
}

// Secret holds Provider-specific secret data.
type Secret struct {
	Key   string
	Value string
}
