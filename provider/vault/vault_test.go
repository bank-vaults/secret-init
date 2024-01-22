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
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var originalLogger *slog.Logger

func TestMain(m *testing.M) {
	setupTestLogger()
	code := m.Run()
	restoreLogger()
	os.Exit(code)
}

func setupTestLogger() {
	originalLogger = slog.Default()

	// Redirect logs to avoid polluting the test output
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	testLogger := slog.New(handler)
	slog.SetDefault(testLogger)
}

func restoreLogger() {
	slog.SetDefault(originalLogger)
}

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		daemonMode bool
		err        error
		wantType   bool
	}{
		{
			name: "Valid Provider with Token",
			config: &Config{
				IsLogin:              true,
				TokenFile:            "root",
				Token:                "root",
				TransitKeyID:         "test-key",
				TransitPath:          "transit",
				TransitBatchSize:     10,
				IgnoreMissingSecrets: true,
				FromPath:             "secret/data/test",
				RevokeToken:          true,
			},
			daemonMode: true,
			wantType:   true,
		},
		{
			name:   "Fail to create vault client due to timeout",
			config: &Config{},
			err:    fmt.Errorf("timeout [10s] during waiting for Vault token"),
		},
	}

	for _, tt := range tests {
		ttp := tt

		t.Run(ttp.name, func(t *testing.T) {
			provider, err := NewProvider(ttp.config, ttp.daemonMode)
			if err != nil {
				assert.EqualError(t, err, ttp.err.Error(), "Unexpected error message")
			}
			if ttp.wantType {
				assert.Equal(t, ttp.wantType, provider != nil, "Unexpected provider type")
			}
		})
	}
}
