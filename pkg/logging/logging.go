package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New creates a slog.Logger using the provided level string.
func New(level string) *slog.Logger {
	l := slog.LevelInfo
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	}

	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: l})
	return slog.New(handler)
}
