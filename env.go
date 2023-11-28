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
	"os"
	"strings"

	"github.com/bank-vaults/secret-init/provider"
	"github.com/bank-vaults/secret-init/provider/file"
)

func GetEnvironMap() map[string]string {
	environ := make(map[string]string, len(os.Environ()))
	for _, env := range os.Environ() {
		split := strings.SplitN(env, "=", 2)
		name := split[0]
		value := split[1]
		environ[name] = value
	}

	return environ
}

func ExtractPathsFromEnvs(envs map[string]string) []string {
	var secretPaths []string

	for _, path := range envs {
		if p, path := getProviderPath(path); p != nil {
			secretPaths = append(secretPaths, path)
		}
	}

	return secretPaths
}

func CreateSecretEnvsFrom(envs map[string]string, secrets []provider.Secret) ([]string, error) {
	// Reverse the map so we can match
	// the environment variable key to the secret
	// by using the secret path
	reversedEnvs := make(map[string]string)
	for envKey, path := range envs {
		if p, path := getProviderPath(path); p != nil {
			reversedEnvs[path] = envKey
		}
	}

	var secretsEnv []string
	for _, secret := range secrets {
		path := secret.Path
		value := secret.Value
		key, ok := reversedEnvs[path]
		if !ok {
			return nil, fmt.Errorf("failed to find environment variable key for secret path: %s", path)
		}
		secretsEnv = append(secretsEnv, fmt.Sprintf("%s=%s", key, value))
	}

	return secretsEnv, nil
}

// Returns the detected provider name and path with removed prefix
func getProviderPath(path string) (*string, string) {
	if strings.HasPrefix(path, "file:") {
		var fileProviderName = file.ProviderName
		return &fileProviderName, strings.TrimPrefix(path, "file:")
	}

	return nil, path
}
