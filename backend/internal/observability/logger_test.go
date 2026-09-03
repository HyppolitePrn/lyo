package observability_test

import (
	"context"
	"testing"

	"log/slog"

	"github.com/hyppoliteprn/lyo/internal/observability"
)

func TestNewLogger_LevelParsing(t *testing.T) {
	tests := map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"warn":    slog.LevelWarn,
		"error":   slog.LevelError,
		"info":    slog.LevelInfo,
		"":        slog.LevelInfo, // unrecognised values fall back to info
		"verbose": slog.LevelInfo,
	}

	for level, want := range tests {
		t.Run(level, func(t *testing.T) {
			logger := observability.NewLogger(level)
			if logger == nil {
				t.Fatal("expected a logger")
			}
			if !logger.Enabled(context.Background(), want) {
				t.Errorf("level %q: expected %v to be enabled", level, want)
			}
			if want > slog.LevelDebug && logger.Enabled(context.Background(), want-4) {
				t.Errorf("level %q: expected %v to be filtered out", level, want-4)
			}
		})
	}
}
