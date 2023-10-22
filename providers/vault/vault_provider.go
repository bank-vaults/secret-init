// Copyright © 2018 Banzai Cloud
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
	"log/slog"

	"github.com/bank-vaults/secret-init/logger"
	"github.com/bank-vaults/secret-init/providers"
)

type VaultProvider struct {
	name   string
	logger *slog.Logger
}

func NewVaultProvider() providers.Provider {
	logger := logger.SetupSlog()
	// clientOptions := []vault.ClientOption{vault.ClientLogger(clientLogger{logger})}
	logger = logger.With(slog.String("provider", "hashicorp-vault"))
	return &VaultProvider{name: "I'm a Vault-provider", logger: logger}
}

func (vp VaultProvider) RetrieveSecrets(envVars []string) ([]string, error) {
	empty := make([]string, 1)
	empty = append(empty, vp.name)
	return empty, nil
}
