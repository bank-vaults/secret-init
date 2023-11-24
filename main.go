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
	"strings"
	"syscall"
	"time"

	slogmulti "github.com/samber/slog-multi"
	slogsyslog "github.com/samber/slog-syslog"
	"github.com/spf13/cast"

	"github.com/bank-vaults/secret-init/provider"
	"github.com/bank-vaults/secret-init/provider/file"
)

func NewProvider(providerName string) (provider.Provider, error) {
	switch providerName {
	case file.ProviderName:
		provider, err := file.NewProvider(os.DirFS("/secrets"))
		if err != nil {
			return nil, err
		}

		return provider, nil

	default:
		return nil, errors.New("invalid provider specified")
	}
}

func CreateMapOfEnvs() map[string]string {
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
		if strings.HasPrefix(path, "file:") {
			path = strings.TrimPrefix(path, "file://")
			secretPaths = append(secretPaths, path)
		}
	}

	return secretPaths
}

func CreateEnvsFromLoadedSecrets(envs map[string]string, secrets []string) ([]string, error) {
	// Reverse the map so we can match
	// the environment variable key to the secret
	// by using the secret path
	reversedEnvs := make(map[string]string)
	for envKey, path := range envs {
		if strings.HasPrefix(path, "file:") {
			path = strings.TrimPrefix(path, "file://")
			reversedEnvs[path] = envKey
		}
	}

	var secretsEnv []string
	for _, secret := range secrets {
		split := strings.SplitN(secret, "|", 2)
		secretPath := split[0]

		secretValue := split[1]
		secretKey, ok := reversedEnvs[secretPath]
		if !ok {
			return nil, fmt.Errorf("failed to find environment variable key for secret path: %s", secretPath)
		}
		secretsEnv = append(secretsEnv, fmt.Sprintf("%s=%s", secretKey, secretValue))
	}

	return secretsEnv, nil
}

func main() {
	var logger *slog.Logger
	{
		var level slog.Level

		err := level.UnmarshalText([]byte(os.Getenv("VAULT_LOG_LEVEL")))
		if err != nil { // Silently fall back to info level
			level = slog.LevelInfo
		}

		levelFilter := func(levels ...slog.Level) func(ctx context.Context, r slog.Record) bool {
			return func(ctx context.Context, r slog.Record) bool {
				return slices.Contains(levels, r.Level)
			}
		}

		router := slogmulti.Router()

		if cast.ToBool(os.Getenv("VAULT_JSON_LOG")) {
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

		if logServerAddr := os.Getenv("VAULT_ENV_LOG_SERVER"); logServerAddr != "" {
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

	provider, err := NewProvider(os.Getenv("PROVIDER"))
	if err != nil {
		logger.Error(fmt.Errorf("failed to create provider: %w", err).Error())

		os.Exit(1)
	}

	if len(os.Args) == 1 {
		logger.Error("no command is given, vault-env can't determine the entrypoint (command), please specify it explicitly or let the webhook query it (see documentation)")

		os.Exit(1)
	}

	daemonMode := cast.ToBool(os.Getenv("VAULT_ENV_DAEMON"))
	delayExec := cast.ToDuration(os.Getenv("VAULT_ENV_DELAY"))

	entrypointCmd := os.Args[1:]

	binary, err := exec.LookPath(entrypointCmd[0])
	if err != nil {
		logger.Error("binary not found", slog.String("binary", entrypointCmd[0]))

		os.Exit(1)
	}

	environ := CreateMapOfEnvs()
	paths := ExtractPathsFromEnvs(environ)

	ctx := context.Background()
	secrets, err := provider.LoadSecrets(ctx, paths)
	if err != nil {
		logger.Error(fmt.Errorf("failed to load secrets from provider: %w", err).Error())

		os.Exit(1)
	}

	secretsEnv, err := CreateEnvsFromLoadedSecrets(environ, secrets)
	if err != nil {
		logger.Error(fmt.Errorf("failed to create environment variables from loaded secrets: %w", err).Error())

		os.Exit(1)
	}

	sigs := make(chan os.Signal, 1)

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
