package di

import (
	"github.com/polkiloo/gophermart/internal/adapter/accrual"
	"github.com/polkiloo/gophermart/internal/app"
	"github.com/polkiloo/gophermart/internal/config"
	"github.com/polkiloo/gophermart/internal/logger"
	"github.com/polkiloo/gophermart/internal/pkg/auth"
	"github.com/polkiloo/gophermart/internal/server/http/router"
	"github.com/polkiloo/gophermart/internal/storage/postgres"
	"github.com/polkiloo/gophermart/internal/usecase"
	"go.uber.org/fx"
)

func Module(opts ...fx.Option) fx.Option {
	modules := []fx.Option{
		config.Module,
		logger.Module,
		auth.Module,
		postgres.Module,
		accrual.Module,
		usecase.Module,
		fx.Provide(func(client accrual.Client) app.AccrualProvider { return client }),
		router.Module,
		app.Module,
	}
	modules = append(modules, opts...)
	return fx.Options(modules...)
}
