package logging

import (
	"context"
	"log/slog"
	"testing"
)

func TestNewLoggerLevels(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	cases := []struct {
		name  string
		level string
		check slog.Level
		want  bool
	}{
		{name: "debug enables debug", level: "debug", check: slog.LevelDebug, want: true},
		{name: "info disables debug", level: "info", check: slog.LevelDebug, want: false},
		{name: "warn enables warn", level: "warn", check: slog.LevelWarn, want: true},
		{name: "error disables warn", level: "error", check: slog.LevelWarn, want: false},
		{name: "unknown defaults to info", level: "trace", check: slog.LevelInfo, want: true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			logger := New(tc.level)
			if logger == nil {
				t.Fatalf("expected logger instance")
			}
			got := logger.Enabled(ctx, tc.check)
			if got != tc.want {
				t.Fatalf("Enabled(%v) = %t, want %t", tc.check, got, tc.want)
			}
		})
	}
}
