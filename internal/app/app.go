package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/polkiloo/gophermart/internal/config"
	"github.com/polkiloo/gophermart/internal/worker"
)

// Module wires application services, runtime components, and lifecycle hooks.
var Module = fx.Options(
	fx.Provide(
		NewLoyaltyFacade,
		newHTTPServer,
		newOrderProcessor,
	),
	fx.Invoke(registerLifecycle),
)

type serverParams struct {
	fx.In

	Config *config.Config
	Router *gin.Engine
}

func newHTTPServer(p serverParams) *http.Server {
	return &http.Server{
		Addr:    p.Config.RunAddress,
		Handler: p.Router,
	}
}

type workerParams struct {
	fx.In

	Facade *LoyaltyFacade
	Config *config.Config
	Logger *slog.Logger
}

func newOrderProcessor(p workerParams) *worker.OrderProcessor {
	return worker.NewOrderProcessor(
		p.Facade,
		p.Config.OrderPollInterval,
		p.Config.MaxOrdersBatch,
		p.Config.WorkerPoolSize,
		p.Logger,
	)
}

type lifecycleParams struct {
	fx.In

	Lifecycle  fx.Lifecycle
	Shutdowner fx.Shutdowner
	Logger     *slog.Logger
	Server     *http.Server
	Worker     *worker.OrderProcessor
	Config     *config.Config
}

func registerLifecycle(p lifecycleParams) {
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			p.Logger.Info("starting gophermart", slog.String("addr", p.Server.Addr))
			p.Worker.Start(ctx)
			go func() {
				if err := p.Server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					p.Logger.Error("http server terminated", slog.String("error", err.Error()))
					_ = p.Shutdowner.Shutdown()
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			p.Worker.Stop()

			shutdownCtx := ctx
			cancel := func() {}
			if _, ok := ctx.Deadline(); !ok {
				shutdownCtx, cancel = context.WithTimeout(ctx, p.Config.ShutdownTimeout)
			}
			defer cancel()

			if err := p.Server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			p.Logger.Info("gophermart stopped")
			return nil
		},
	})
}
