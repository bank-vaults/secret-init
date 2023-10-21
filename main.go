// Copyright © 2018 Banzai Cloud
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
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/bank-vaults/secret-init/logger"
	"github.com/bank-vaults/secret-init/providers"
	"github.com/bank-vaults/secret-init/providers/vault"
	"github.com/spf13/cast"
)

func main() {
	logger := logger.SetupSlog()

	var providerName string
	if len(os.Args) >= 3 && os.Args[1] == "-p" {
		providerName = os.Args[2]

		// Remove the "-p {provider name}" argument from the command-line arguments
		os.Args = append(os.Args[:1], os.Args[3:]...)
	}

	var provider providers.Provider
	switch providerName {
	case "aws":
		// Handle AWS provider
	case "vault":
		// Handle Vault provider
		provider = vault.NewVaultProvider()
	case "gcp":
		// Handle GCP provider

	default:
		logger.Error("Invalid provider specified.", slog.String("provider name", providerName))

		os.Exit(1)
	}

	daemonMode := cast.ToBool(os.Getenv("VAULT_ENV_DAEMON"))
	delayExec := cast.ToDuration(os.Getenv("VAULT_ENV_DELAY"))

	if len(os.Args) == 1 {
		logger.Error("no command is given, vault-env can't determine the entrypoint (command), please specify it explicitly or let the webhook query it (see documentation)")

		os.Exit(1)
	}

	entrypointCmd := os.Args[1:]

	binary, err := exec.LookPath(entrypointCmd[0])
	if err != nil {
		logger.Error("binary not found", slog.String("binary", entrypointCmd[0]))

		os.Exit(1)
	}

	if delayExec > 0 {
		logger.Info(fmt.Sprintf("sleeping for %s...", delayExec))
		time.Sleep(delayExec)
	}

	logger.Info("spawning process", slog.String("entrypoint", fmt.Sprint(entrypointCmd)))

	var envs []string
	envs, err = provider.RetrieveSecrets(os.Environ())
	if err != nil {
		logger.Error("could not retrieve secrets from the provider.")

		os.Exit(1)
	}

	if daemonMode {
		logger.Info("in daemon mode...")
	} else { //nolint:revive
		err = syscall.Exec(binary, entrypointCmd, envs)
		if err != nil {
			logger.Error(fmt.Errorf("failed to exec process: %w", err).Error(), slog.String("entrypoint", fmt.Sprint(entrypointCmd)))

			os.Exit(1)
		}
	}
}
