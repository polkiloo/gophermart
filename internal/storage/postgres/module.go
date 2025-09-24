package postgres

import (
	"context"
	"log/slog"

	"go.uber.org/fx"

	"github.com/polkiloo/gophermart/internal/config"
	"github.com/polkiloo/gophermart/internal/domain/repository"
)

// Module wires PostgreSQL storage and repository adapters.
var Module = fx.Options(
	fx.Provide(newStorage),
	fx.Provide(
		func(s *Storage) repository.UserRepository { return s.Users() },
		func(s *Storage) repository.OrderRepository { return s.Orders() },
		func(s *Storage) repository.BalanceRepository { return s.Balances() },
		func(s *Storage) repository.WithdrawalRepository { return s.Withdrawals() },
	),
	fx.Invoke(registerLifecycle),
)

type storageParams struct {
	fx.In

	Ctx    context.Context
	Config *config.Config
	Logger *slog.Logger
}

func newStorage(p storageParams) (*Storage, error) {
	return New(p.Ctx, p.Config.DatabaseURI, p.Logger)
}

func registerLifecycle(lc fx.Lifecycle, storage *Storage) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			storage.Close()
			return nil
		},
	})
}
