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

package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractEntrypoint(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		expectedBinaryPath string
		expectedBinaryArgs []string
		err                error
	}{
		{
			name:               "Valid case with one argument",
			args:               []string{"secret-init", "env"},
			expectedBinaryPath: "/usr/bin/env",
			expectedBinaryArgs: []string{},
		},
		{
			name:               "Valid case with more than two arguments",
			args:               []string{"secret-init", "env", "|", "grep", "secrets"},
			expectedBinaryPath: "/usr/bin/env",
			expectedBinaryArgs: []string{"|", "grep", "secrets"},
		},
		{
			name: "Invalid case - no arguments",
			args: []string{"secret-init"},
			err:  fmt.Errorf("no args provided"),
		},
		{
			name: "Invalid case - binary not found",
			args: []string{"secret-init", "nonexistentBinary"},
			err:  fmt.Errorf("binary nonexistentBinary not found"),
		},
	}

	for _, tt := range tests {
		ttp := tt
		t.Run(ttp.name, func(t *testing.T) {
			binaryPath, binaryArgs, err := ExtractEntrypoint(ttp.args)
			if err != nil {
				assert.EqualError(t, ttp.err, err.Error(), "Unexpected error message")
			} else {
				assert.Equal(t, ttp.expectedBinaryPath, binaryPath, "Unexpected binary path")
				assert.Equal(t, ttp.expectedBinaryArgs, binaryArgs, "Unexpected binary args")
			}
		})
	}
}
