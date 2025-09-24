package router

import "go.uber.org/fx"

// Module registers HTTP router construction for fx runtime.
var Module = fx.Provide(Setup)
