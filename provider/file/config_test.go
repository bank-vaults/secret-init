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

package file

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		name          string
		env           map[string]string
		wantMountPath string
	}{
		{
			name:          "Default mount path",
			env:           map[string]string{},
			wantMountPath: "/",
		},
		{
			name: "Custom mount path",
			env: map[string]string{
				MountPathEnv: "/test/secrets",
			},
			wantMountPath: "/test/secrets",
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			for envKey, envVal := range ttp.env {
				os.Setenv(envKey, envVal)
			}

			config := LoadConfig()

			assert.Equal(t, ttp.wantMountPath, config.MountPath, "Unexpected mount path")

			// unset envs for the next test
			for envKey := range ttp.env {
				os.Unsetenv(envKey)
			}
		})
	}
}
