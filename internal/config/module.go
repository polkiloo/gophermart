package config

import "go.uber.org/fx"

// Module exposes configuration loader for fx graphs.
var Module = fx.Provide(Load)
