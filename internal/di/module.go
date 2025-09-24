package di

import (
	"go.uber.org/fx"
)

func Module() fx.Option {
	return fx.Options(
	// config.Module,
	// logger.Module,
	// auth.Module,
	// postgres.Module,
	// accrual.Module,
	// usecase.Module,
	// httpRouter.Module,
	// app.Module,
	)
}
