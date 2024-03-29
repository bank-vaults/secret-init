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
	"os/exec"
)

// ExtractEntrypoint extracts entrypoint data in the form of binary path and its arguments from the
// os.Args. Note that the path to the binary will be returned as the first element.
func ExtractEntrypoint(args []string) (string, []string, error) {
	if len(args) <= 1 {
		return "", nil, fmt.Errorf("no args provided")
	}

	binaryPath, err := exec.LookPath(args[1])
	if err != nil {
		return "", nil, fmt.Errorf("binary %s not found", args[1])
	}

	var binaryArgs []string
	if len(args) >= 2 {
		binaryArgs = args[2:] // returns the arguments for the binary
	}

	return binaryPath, binaryArgs, nil
}
