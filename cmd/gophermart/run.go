package main

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/fx"
)

func run(ctx context.Context, app *fx.App) {
	if err := app.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start application: %v\n", err)
		os.Exit(1)
	}

	select {
	case <-ctx.Done():
	case <-app.Done():
	}

	if err := app.Stop(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "failed to stop application: %v\n", err)
		os.Exit(1)
	}
}
