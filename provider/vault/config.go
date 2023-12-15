package vault

import (
	"errors"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cast"
)

const (
	EnvPrefix  = "VAULT_"
	vaultLogin = "vault:login"
)

type Config struct {
	Islogin              bool
	Token                string   `json:"token"`
	TokenFile            string   `json:"tokenFile"`
	Role                 string   `json:"role"`
	Path                 string   `json:"path"`
	AuthMethod           string   `json:"authMethod"`
	PassthroughEnvVars   []string `json:"passthroughEnvVars"`
	TransitKeyID         string   `json:"transitKeyID"`
	TransitPath          string   `json:"transitPath"`
	TransitBatchSize     int      `json:"transitBatchSize"`
	DaemonMode           bool     `json:"daemonMode"`
	IgnoreMissingSecrets bool     `json:"ignoreMissingSecrets"`
	Paths                string   `json:"paths"`
	RevokeToken          string   `json:"revokeToken"`
}

func NewConfig(logger *slog.Logger) (*Config, error) {
	var (
		role, path, authMethod          string
		hasRole, hasPath, hasAuthMethod bool
	)

	vaultToken := os.Getenv(EnvPrefix + "TOKEN")
	isLogin := vaultToken == vaultLogin
	tokenFile, ok := os.LookupEnv(EnvPrefix + "TOKEN_FILE")
	if !ok {
		if b, err := os.ReadFile(tokenFile); err == nil {
			vaultToken = string(b)
		} else {
			logger.Error("could not read vault token file", slog.String("file", tokenFile))

			return nil, err
		}
	} else {
		if isLogin {
			_ = os.Unsetenv(EnvPrefix + "TOKEN")
		}

		role, hasRole = os.LookupEnv(EnvPrefix + "ROLE")
		path, hasPath = os.LookupEnv(EnvPrefix + "PATH")
		authMethod, hasAuthMethod = os.LookupEnv(EnvPrefix + "AUTH_METHOD")
		if !hasRole || !hasPath || !hasAuthMethod {
			logger.Error("Incomplete authentication configuration. Make sure VAULT_ROLE, VAULT_PATH, and VAULT_AUTH_METHOD are set.")

			return nil, errors.New("incomplete authentication configuration")
		}
	}

	passthroughEnvVars := strings.Split(os.Getenv("SECRET_INIT_PASSTHROUGH"), ",")

	if isLogin {
		_ = os.Setenv(EnvPrefix+"TOKEN", vaultLogin)
		passthroughEnvVars = append(passthroughEnvVars, EnvPrefix+"TOKEN")
	}

	transitKeyID := os.Getenv(EnvPrefix + "TRANSIT_KEY_ID")
	transitPath := os.Getenv(EnvPrefix + "TRANSIT_PATH")
	transitBatchSize := cast.ToInt(os.Getenv(EnvPrefix + "TRANSIT_BATCH_SIZE"))
	daemonMode := cast.ToBool(os.Getenv(EnvPrefix + "DAEMON_MODE"))
	ignoreMissingSecrets := cast.ToBool(os.Getenv(EnvPrefix + "IGNORE_MISSING_SECRETS"))

	paths := os.Getenv("SECRET_INIT_FROM_PATH")
	revokeToken := os.Getenv(EnvPrefix + "REVOKE_TOKEN")

	return &Config{
		Islogin:              isLogin,
		Token:                vaultToken,
		TokenFile:            tokenFile,
		Role:                 role,
		Path:                 path,
		AuthMethod:           authMethod,
		PassthroughEnvVars:   passthroughEnvVars,
		TransitKeyID:         transitKeyID,
		TransitPath:          transitPath,
		TransitBatchSize:     transitBatchSize,
		DaemonMode:           daemonMode,
		IgnoreMissingSecrets: ignoreMissingSecrets,
		Paths:                paths,
		RevokeToken:          revokeToken,
	}, nil
}
