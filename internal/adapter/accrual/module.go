package accrual

import (
	"log/slog"

	"go.uber.org/fx"

	"github.com/polkiloo/gophermart/internal/config"
)

// Module exposes accrual client implementation to fx graph.
var Module = fx.Provide(newClient)

type clientParams struct {
	fx.In

	Config *config.Config
	Logger *slog.Logger
}

func newClient(p clientParams) (Client, error) {
	return NewHTTPClient(p.Config.AccrualSystemAddress, p.Logger)
}
