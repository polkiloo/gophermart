package logger

import (
	"log/slog"
	"os"
)

// New creates a preconfigured slog.Logger.
func New() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(handler)
}
