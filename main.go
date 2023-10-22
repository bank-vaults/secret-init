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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cast"

	"github.com/bank-vaults/secret-init/logger"
	"github.com/bank-vaults/secret-init/providers"
	"github.com/bank-vaults/secret-init/providers/vault"
)

var providersMap = map[string]providers.Provider{
	// "aws": aws.NewAWSProvider(),
	// "gcp": gcp.NewGCPProvider(),
	"vault": vault.NewVaultProvider(),
}

func main() {
	logger := logger.SetupSlog()

	var providerName string
	if len(os.Args) >= 3 && os.Args[1] == "-p" {
		providerName = os.Args[2]

		// Remove the "-p {provider name}" argument from the command-line arguments
		os.Args = append(os.Args[:1], os.Args[3:]...)
	}

	provider, found := providersMap[providerName]
	if !found {
		logger.Error("invalid provider specified.", slog.String("provider name", providerName))

		os.Exit(1)
	}

	if len(os.Args) == 1 {
		logger.Error("no command is given, vault-env can't determine the entrypoint (command), please specify it explicitly or let the webhook query it (see documentation)")

		os.Exit(1)
	}

	daemonMode := cast.ToBool(os.Getenv("VAULT_ENV_DAEMON"))
	delayExec := cast.ToDuration(os.Getenv("VAULT_ENV_DELAY"))
	sigs := make(chan os.Signal, 1)

	entrypointCmd := os.Args[1:]

	binary, err := exec.LookPath(entrypointCmd[0])
	if err != nil {
		logger.Error("binary not found", slog.String("binary", entrypointCmd[0]))

		os.Exit(1)
	}

	var envs []string
	envs, err = provider.RetrieveSecrets(os.Environ())
	if err != nil {
		logger.Error("could not retrieve secrets from the provider.", err)

		os.Exit(1)
	}

	if delayExec > 0 {
		logger.Info(fmt.Sprintf("sleeping for %s...", delayExec))
		time.Sleep(delayExec)
	}

	logger.Info("spawning process", slog.String("entrypoint", fmt.Sprint(entrypointCmd)))

	if daemonMode {
		logger.Info("in daemon mode...")
		cmd := exec.Command(binary, entrypointCmd[1:]...)
		cmd.Env = append(os.Environ(), envs...)
		cmd.Stdin = os.Stdin
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout

		signal.Notify(sigs)

		err = cmd.Start()
		if err != nil {
			logger.Error(fmt.Errorf("failed to start process: %w", err).Error(), slog.String("entrypoint", fmt.Sprint(entrypointCmd)))

			os.Exit(1)
		}

		go func() {
			for sig := range sigs {
				// We don't want to signal a non-running process.
				if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
					break
				}

				err := cmd.Process.Signal(sig)
				if err != nil {
					logger.Warn(fmt.Errorf("failed to signal process: %w", err).Error(), slog.String("signal", sig.String()))
				} else {
					logger.Info("received signal", slog.String("signal", sig.String()))
				}
			}
		}()

		err = cmd.Wait()

		close(sigs)

		if err != nil {
			exitCode := -1
			// try to get the original exit code if possible
			var exitError *exec.ExitError
			if errors.As(err, &exitError) {
				exitCode = exitError.ExitCode()
			}

			logger.Error(fmt.Errorf("failed to exec process: %w", err).Error(), slog.String("entrypoint", fmt.Sprint(entrypointCmd)))

			os.Exit(exitCode)
		}

		os.Exit(cmd.ProcessState.ExitCode())
	} else { //nolint:revive
		err = syscall.Exec(binary, entrypointCmd, envs)
		if err != nil {
			logger.Error(fmt.Errorf("failed to exec process: %w", err).Error(), slog.String("entrypoint", fmt.Sprint(entrypointCmd)))

			os.Exit(1)
		}
	}
}
