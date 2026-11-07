// Package logger builds the application's structured logger.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a slog logger that writes to stdout. Production and staging use
// JSON records, development uses human-readable text records.
func New(environment, level string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}

	var handler slog.Handler
	switch strings.ToLower(environment) {
	case "production", "staging":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
