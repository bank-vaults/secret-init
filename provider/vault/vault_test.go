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
	}

}
