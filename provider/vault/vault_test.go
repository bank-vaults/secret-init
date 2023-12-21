package vault

import (
	"bytes"
	"log/slog"
	"os"
	"testing"

	"github.com/bank-vaults/internal/injector"
	"github.com/stretchr/testify/assert"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name               string
		config             *Config
		wantInjectorConfig injector.Config
		wantErr            bool
		wantType           bool
	}{
		{
			name: "Valid Provider with Token",
			config: &Config{
				Islogin:              true,
				TokenFile:            "root",
				Token:                "root",
				TransitKeyID:         "test-key",
				TransitPath:          "transit",
				TransitBatchSize:     10,
				DaemonMode:           true,
				IgnoreMissingSecrets: true,
				FromPath:             "secret/data/test",
				RevokeToken:          true,
			},
			wantErr:  false,
			wantType: true,
		},
		{
			name:     "Fail to create vault client",
			config:   &Config{},
			wantErr:  true,
			wantType: false,
		},
	}

	for _, tt := range tests {
		ttp := tt

		// Redirect logs to avoid polluting the test output
		var buf bytes.Buffer
		handler := slog.NewTextHandler(&buf, nil)
		logger := slog.New(handler)

		t.Run(ttp.name, func(t *testing.T) {
			provider, err := NewProvider(ttp.config, logger, make(chan os.Signal))

			assert.Equal(t, ttp.wantErr, err != nil, "Unexpected error status")
			assert.Equal(t, ttp.wantType, provider != nil, "Unexpected provider type")
		})

		buf.Truncate(0)
	}

}
