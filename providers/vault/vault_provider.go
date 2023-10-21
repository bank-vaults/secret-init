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
