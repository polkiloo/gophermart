package logger

import (
	"context"
	"log/slog"
	"testing"
)

func TestNewProvidesJSONLogger(t *testing.T) {
	l := New()
	if l == nil {
		t.Fatal("expected logger, got nil")
	}

	if !l.Enabled(context.Background(), slog.LevelInfo) {
		t.Errorf("expected info level to be enabled")
	}
	if l.Enabled(context.Background(), slog.LevelDebug) {
		t.Errorf("did not expect debug level to be enabled")
	}

	if _, ok := l.Handler().(*slog.JSONHandler); !ok {
		t.Fatalf("expected JSON handler, got %T", l.Handler())
	}
}
