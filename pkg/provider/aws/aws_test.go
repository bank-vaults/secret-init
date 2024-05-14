// Copyright © 2024 Bank-Vaults Maintainers
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

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/stretchr/testify/assert"
)

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		wantType bool
	}{
		{
			name: "Valid config",
			config: &Config{
				session: createSession(),
			},
			wantType: true,
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			provider := NewProvider(ttp.config)
			if ttp.wantType {
				assert.Equal(t, ttp.wantType, provider != nil, "Unexpected provider type")
			}
		})
	}

}

func createSession() *session.Session {
	return session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigDisable,
	}))
}
