package auth

import (
	"github.com/polkiloo/gophermart/internal/config"
	"go.uber.org/fx"
)

// Module provides authentication primitives via fx.
var Module = fx.Options(
	fx.Provide(newPasswordHasher),
	fx.Provide(newTokenStrategy),
)

func newPasswordHasher() PasswordHasher {
	return NewBcryptHasher(0)
}

type strategyParams struct {
	fx.In

	Config *config.Config
}

func newTokenStrategy(p strategyParams) Strategy {
	return NewHMACStrategy(p.Config.JWTSecret, Options{})
}
