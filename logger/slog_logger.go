package logger

import (
	"context"
	"log/slog"
	"net"
	"os"
	"slices"

	slogmulti "github.com/samber/slog-multi"
	slogsyslog "github.com/samber/slog-syslog"
	"github.com/spf13/cast"
)

var logger *slog.Logger

func SetupSlog() *slog.Logger {
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
	logger = logger.With(slog.String("app", "vault-env"))

	slog.SetDefault(logger)
	return logger
}
