package logger

import (
	"context"
	"log/slog"
	"testing"

	"go.uber.org/fx"
)

func TestModuleProvidesLogger(t *testing.T) {
	var resolved *slog.Logger
	app := fx.New(
		Module,
		fx.Populate(&resolved),
	)
	t.Cleanup(func() { _ = app.Stop(context.Background()) })
	if err := app.Err(); err != nil {
		t.Fatalf("fx app failed: %v", err)
	}
	if resolved == nil {
		t.Fatal("expected logger to be populated")
	}
}
