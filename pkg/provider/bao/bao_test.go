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

package bao

import (
	"fmt"
	"io"
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

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		err      error
		wantType bool
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
			wantType: true,
		},
		{
			name: "Valid Provider with bao:login as Token and daemon mode",
			config: &Config{
				IsLogin:              true,
				Token:                baoLogin,
				TokenFile:            "root",
				IgnoreMissingSecrets: true,
				FromPath:             "secret/data/test",
			},
			wantType: true,
		},
		{
			name:   "Fail to create bao client due to timeout",
			config: &Config{},
			err:    fmt.Errorf("failed to create bao client: timeout [10s] during waiting for Bao token"),
		},
	}

	for _, tt := range tests {
		ttp := tt

		t.Run(ttp.name, func(t *testing.T) {
			provider, err := NewProvider(ttp.config)
			if err != nil {
				assert.EqualError(t, ttp.err, err.Error(), "Unexpected error message")
			}
			if ttp.wantType {
				assert.Equal(t, ttp.wantType, provider != nil, "Unexpected provider type")
			}
		})
	}
}

func setupTestLogger() {
	originalLogger = slog.Default()

	// Discard logs to avoid polluting the test output
	testLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
	slog.SetDefault(testLogger)
}

func restoreLogger() {
	slog.SetDefault(originalLogger)
}
