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
	"time"

	slogmulti "github.com/samber/slog-multi"
	slogsyslog "github.com/samber/slog-syslog"

	"github.com/bank-vaults/secret-init/common"
)

func main() {
	// Load application config
	config, err := common.LoadConfig()
	if err != nil {
		slog.Error(fmt.Errorf("failed to load config: %w", err).Error())
		os.Exit(1)
	}

	initLogger(config)

	// Get entrypoint data from arguments
	binaryPath, binaryArgs, err := ExtractEntrypoint(os.Args)
	if err != nil {
		slog.Error(fmt.Errorf("failed to extract entrypoint: %w", err).Error())
		os.Exit(1)
	}

	envStore := NewEnvStore()

	providerPaths := envStore.GetProviderPaths()

	providerSecrets, err := envStore.GetProviderSecrets(providerPaths)
	if err != nil {
		slog.Error(fmt.Errorf("failed to extract secrets: %w", err).Error())
		os.Exit(1)
	}

	secretsEnv, err := envStore.ConvertProviderSecrets(providerSecrets)
	if err != nil {
		slog.Error(fmt.Errorf("failed to convert secrets to envs: %w", err).Error())
		os.Exit(1)
	}

	// Delay if needed
	// NOTE(ramizpolic): any specific reason why this is here?
	if config.Delay > 0 {
		slog.Info(fmt.Sprintf("sleeping for %s...", config.Delay))
		time.Sleep(config.Delay)
	}

	slog.Info("spawning process for provided entrypoint command")

	cmd := exec.Command(binaryPath, binaryArgs...)
	cmd.Env = append(os.Environ(), secretsEnv...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs)

	err = cmd.Start()
	if err != nil {
		slog.Error(fmt.Errorf("failed to start process: %w", err).Error())
		os.Exit(1)
	}

	if config.Daemon {
		// in daemon mode, pass signals to the actual process
		slog.Info("running in daemon mode")

		go func() {
			for sig := range sigs {
				slog.Info("received signal", slog.String("signal", sig.String()))

				// We don't want to signal a non-running process.
				if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
					break
				}

				err := cmd.Process.Signal(sig)
				if err != nil {
					slog.Warn(
						fmt.Errorf("failed to signal process: %w", err).Error(),
						slog.String("signal", sig.String()),
					)
				}
			}
		}()
	}

	err = cmd.Wait()

	close(sigs)

	if err != nil {
		slog.Error(fmt.Errorf("failed to exec process: %w", err).Error())

		// Exit with the original exit code if possible
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}

		os.Exit(-1)
	}

	os.Exit(cmd.ProcessState.ExitCode())
}

func initLogger(config *common.Config) {
	var level slog.Level

	err := level.UnmarshalText([]byte(config.LogLevel))
	if err != nil { // Silently fall back to info level
		level = slog.LevelInfo
	}

	levelFilter := func(levels ...slog.Level) func(ctx context.Context, r slog.Record) bool {
		return func(ctx context.Context, r slog.Record) bool {
			return slices.Contains(levels, r.Level)
		}
	}

	router := slogmulti.Router()

	if config.JSONLog {
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

	if config.LogServer != "" {
		writer, err := net.Dial("udp", config.LogServer)

		// We silently ignore syslog connection errors for the lack of a better solution
		if err == nil {
			router = router.Add(slogsyslog.Option{Level: slog.LevelInfo, Writer: writer}.NewSyslogHandler())
		}
	}

	// TODO: add level filter handler
	logger := slog.New(router.Handler())
	logger = logger.With(slog.String("app", "vault-secret-init"))

	// Set the default logger to the configured logger,
	// enabling direct usage of the slog package for logging.
	slog.SetDefault(logger)
}
