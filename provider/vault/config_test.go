package vault

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	// mock logger, sigs, and tokenfile
	logger := slog.Default()
	sigs := make(chan os.Signal, 1)
	tokenFile := newTokenFile(t)
	defer os.Remove(tokenFile)

	tests := []struct {
		name       string
		env        map[string]string
		wantConfig *Config
		wantErr    bool
	}{
		{
			name: "Valid login configuration with Token",
			env: map[string]string{
				"VAULT_TOKEN":      vaultLogin,
				"VAULT_TOKEN_FILE": tokenFile,
			},
			wantConfig: &Config{
				Islogin:   true,
				Token:     "root",
				TokenFile: tokenFile,
				Logger:    logger,
				Sigs:      sigs,
			},
			wantErr: false,
		},
		{
			name: "Valid login configuration with Role and Path",
			env: map[string]string{
				"VAULT_TOKEN":       vaultLogin,
				"VAULT_ROLE":        "test-app-role",
				"VAULT_PATH":        "auth/approle/test/login",
				"VAULT_AUTH_METHOD": "test-approle",
			},
			wantConfig: &Config{
				Islogin:    true,
				Token:      vaultLogin,
				Role:       "test-app-role",
				AuthPath:   "auth/approle/test/login",
				AuthMethod: "test-approle",
				Logger:     logger,
				Sigs:       sigs,
			},
			wantErr: false,
		},
		{
			name: "Invalid login configuration missing token file",
			env: map[string]string{
				"VAULT_TOKEN_FILE": tokenFile + "/invalid",
			},
			wantConfig: nil,
			wantErr:    true,
		},
		{
			name: "Invalid login configuration missing role/path credentials",
			env: map[string]string{
				"VAULT_PATH":        "auth/approle/test/login",
				"VAULT_AUTH_METHOD": "test-approle",
			},
			wantConfig: nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			for envKey, envVal := range ttp.env {
				os.Setenv(envKey, envVal)
			}

			config, err := NewConfig(logger, sigs)

			assert.Equal(t, ttp.wantErr, err != nil, "Unexpected error status")
			assert.Equal(t, ttp.wantConfig, config, "Unexpected config")

			// unset envs for the next test
			for envKey := range ttp.env {
				os.Unsetenv(envKey)
			}
		})
	}
}

func newTokenFile(t *testing.T) string {
	tokenFilePath := filepath.Join(t.TempDir(), "vault-token")
	tokenFile, err := os.Create(tokenFilePath)
	if err != nil {
		t.Fatalf("Failed to create a temporary token file: %v", err)
	}

	_, err = tokenFile.Write([]byte("root"))
	if err != nil {
		t.Fatalf("Failed to write to a temporary token file: %v", err)
	}
	return tokenFile.Name()
}
