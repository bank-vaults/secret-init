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
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"syscall"
	"time"

	slogmulti "github.com/samber/slog-multi"
	slogsyslog "github.com/samber/slog-syslog"
	"github.com/spf13/cast"

	"github.com/bank-vaults/secret-init/common"
	"github.com/bank-vaults/secret-init/provider"
	"github.com/bank-vaults/secret-init/provider/file"
	"github.com/bank-vaults/secret-init/provider/vault"
)

func NewProvider(providerName string, logger *slog.Logger, sigs chan os.Signal) (provider.Provider, error) {
	switch providerName {
	case file.ProviderName:
		config := file.NewConfig(logger)
		provider, err := file.NewProvider(config)
		if err != nil {
			return nil, fmt.Errorf("failed to create file provider: %w", err)
		}

		return provider, nil
	case vault.ProviderName:
		config, err := vault.NewConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create vault config: %w", err)
		}

		provider, err := vault.NewProvider(config, logger, sigs)
		if err != nil {
			return nil, fmt.Errorf("failed to create vault provider: %w", err)
		}

		return provider, nil

	default:
		return nil, errors.New("invalid provider specified")
	}
}

func main() {
	var logger *slog.Logger
	{
		var level slog.Level

		err := level.UnmarshalText([]byte(os.Getenv(common.SecretInitLogLevel)))
		if err != nil { // Silently fall back to info level
			level = slog.LevelInfo
		}

		levelFilter := func(levels ...slog.Level) func(ctx context.Context, r slog.Record) bool {
			return func(ctx context.Context, r slog.Record) bool {
				return slices.Contains(levels, r.Level)
			}
		}

		router := slogmulti.Router()

		if cast.ToBool(os.Getenv(common.SecretInitJSONLog)) {
			// Send logs with level higher than warning to stderr
			router = router.Add(
				slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}),
				levelFilter(slog.LevelWarn, slog.LevelError),
			)

			// Send info and debug logs to stdout
			router = router.Add(
				slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
				levelFilter(slog.LevelDebug, slog.LevelInfo),
			)
		} else {
			// Send logs with level higher than warning to stderr
			router = router.Add(
				slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}),
				levelFilter(slog.LevelWarn, slog.LevelError),
			)

			// Send info and debug logs to stdout
			router = router.Add(
				slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
				levelFilter(slog.LevelDebug, slog.LevelInfo),
			)
		}

		if logServerAddr := os.Getenv(common.SecretInitLogServer); logServerAddr != "" {
			writer, err := net.Dial("udp", logServerAddr)

			// We silently ignore syslog connection errors for the lack of a better solution
			if err == nil {
				router = router.Add(slogsyslog.Option{Level: slog.LevelInfo, Writer: writer}.NewSyslogHandler())
			}
		}

		// TODO: add level filter handler
		logger = slog.New(router.Handler())
		logger = logger.With(slog.String("app", "vault-secret-init"))

		slog.SetDefault(logger)
	}

	daemonMode := cast.ToBool(os.Getenv(common.SecretInitDaemon))
	delayExec := cast.ToDuration(os.Getenv(common.SecretInitDelay))
	sigs := make(chan os.Signal, 1)

	provider, err := NewProvider(os.Getenv(common.Provider), logger, sigs)
	if err != nil {
		logger.Error(fmt.Errorf("failed to create provider: %w", err).Error())

		os.Exit(1)
	}

	if len(os.Args) == 1 {
		logger.Error("no command is given, secret-init can't determine the entrypoint (command), please specify it explicitly or let the webhook query it (see documentation)")

		os.Exit(1)
	}

	entrypointCmd := os.Args[1:]

	binary, err := exec.LookPath(entrypointCmd[0])
	if err != nil {
		logger.Error("binary not found", slog.String("binary", entrypointCmd[0]))

		os.Exit(1)
	}

	environ := GetEnvironMap()

	//TODO(csatib02): Implement multi-provider support
	paths := ExtractPathsFromEnvs(environ, provider.GetProviderName())

	ctx := context.Background()
	secrets, err := provider.LoadSecrets(ctx, paths)
	if err != nil {
		logger.Error(fmt.Errorf("failed to load secrets from provider: %w", err).Error())

		os.Exit(1)
	}

	var secretsEnv []string
	if provider.GetProviderName() == vault.ProviderName {
		// The Vault provider already returns the secrets with the environment variable key
		secretsEnv = CreateSecretsEnvForVaultProvider(secrets)
	} else {
		secretsEnv, err = CreateSecretEnvsFrom(environ, secrets)
		if err != nil {
			logger.Error(fmt.Errorf("failed to create environment variables from loaded secrets: %w", err).Error())

			os.Exit(1)
		}
	}

	if delayExec > 0 {
		logger.Info(fmt.Sprintf("sleeping for %s...", delayExec))
		time.Sleep(delayExec)
	}

	logger.Info("spawning process", slog.String("entrypoint", fmt.Sprint(entrypointCmd)))

	if daemonMode {
		logger.Info("in daemon mode...")
		cmd := exec.Command(binary, entrypointCmd[1:]...)
		cmd.Env = append(os.Environ(), secretsEnv...)
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
	}
	err = syscall.Exec(binary, entrypointCmd, secretsEnv)
	if err != nil {
		logger.Error(fmt.Errorf("failed to exec process: %w", err).Error(), slog.String("entrypoint", fmt.Sprint(entrypointCmd)))

		os.Exit(1)
	}
}
