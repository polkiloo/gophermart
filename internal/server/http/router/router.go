package router

import (
	"log/slog"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"

	"github.com/polkiloo/gophermart/internal/server/http/handlers"
	"github.com/polkiloo/gophermart/internal/server/http/middleware"
)

// Setup configures gin router with handlers and middleware.
func Setup(facade handlers.LoyaltyFacade, logger *slog.Logger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	engine.Use(gin.Recovery())
	engine.Use(middleware.RequestLogger(logger))
	engine.Use(middleware.DecompressRequest())
	engine.Use(gzip.Gzip(gzip.DefaultCompression))

	authHandler := handlers.NewAuthHandler(facade)
	orderHandler := handlers.NewOrderHandler(facade)
	balanceHandler := handlers.NewBalanceHandler(facade)

	api := engine.Group("/api")
	user := api.Group("/user")
	user.POST("/register", authHandler.Register)
	user.POST("/login", authHandler.Login)

	userAuth := user.Group("")
	userAuth.Use(middleware.AuthRequired(facade))
	userAuth.POST("/orders", orderHandler.Upload)
	userAuth.GET("/orders", orderHandler.List)
	userAuth.GET("/balance", balanceHandler.Summary)
	userAuth.POST("/balance/withdraw", balanceHandler.Withdraw)
	userAuth.GET("/withdrawals", balanceHandler.Withdrawals)

	return engine
}
