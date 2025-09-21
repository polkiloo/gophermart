package logger

import "go.uber.org/fx"

// Module wires slog logger for dependency injection.
var Module = fx.Provide(New)
