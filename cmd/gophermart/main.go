package main

import (
	"context"
	"os/signal"
	"syscall"

	"go.uber.org/fx"

	"github.com/polkiloo/gophermart/internal/di"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app := fx.New(
		fx.Provide(func() context.Context { return ctx }),
		di.Module(),
	)

	run(ctx, app)
}
