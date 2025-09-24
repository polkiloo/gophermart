package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/polkiloo/gophermart/internal/app"
	"github.com/polkiloo/gophermart/internal/server/http/handlers"
	"go.uber.org/fx"
)

// Module wires router dependencies for fx graphs.
var Module = fx.Options(
	fx.Provide(
		provideFacade,
		newEngine,
	),
)

func provideFacade(facade *app.LoyaltyFacade) handlers.LoyaltyFacade {
	return facade
}

type engineParams struct {
	fx.In

	Facade handlers.LoyaltyFacade
	Logger *slog.Logger
}

func newEngine(p engineParams) *gin.Engine {
	return Setup(p.Facade, p.Logger)
}
