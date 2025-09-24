package di

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/polkiloo/gophermart/internal/adapter/accrual"
	"github.com/polkiloo/gophermart/internal/app"
	"github.com/polkiloo/gophermart/internal/config"
	"github.com/polkiloo/gophermart/internal/domain/model"
	"github.com/polkiloo/gophermart/internal/domain/repository"
	"github.com/polkiloo/gophermart/internal/storage/postgres"
	"github.com/polkiloo/gophermart/internal/test"
	"go.uber.org/fx"
)

func TestModuleComposesGraphWithReplacements(t *testing.T) {
	cfg := &config.Config{
		RunAddress:           ":0",
		DatabaseURI:          "postgres://stub",
		AccrualSystemAddress: "http://localhost",
		JWTSecret:            "secret",
		OrderPollInterval:    time.Millisecond,
		WorkerPoolSize:       1,
		ShutdownTimeout:      time.Millisecond,
		MaxOrdersBatch:       1,
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	userRepo := test.NewUserRepositoryStub()
	orderRepo := &test.OrderRepositoryStub{}
	balanceRepo := &test.BalanceRepositoryStub{Summary: &model.BalanceSummary{}}
	withdrawalRepo := &test.WithdrawalRepositoryStub{}
	accrualStub := &test.AccrualProviderStub{Accrual: &model.Accrual{Order: "1"}}

	var facade *app.LoyaltyFacade
	fxApp := fx.New(
		fx.NopLogger,
		fx.Supply(context.Background()),
		Module(
			fx.Replace(cfg),
			fx.Replace(logger),
			fx.Replace(&postgres.Storage{}),
			fx.Replace(repository.UserRepository(userRepo)),
			fx.Replace(repository.OrderRepository(orderRepo)),
			fx.Replace(repository.BalanceRepository(balanceRepo)),
			fx.Replace(repository.WithdrawalRepository(withdrawalRepo)),
			fx.Replace(accrual.Client(accrualStub)),
		),
		fx.Populate(&facade),
	)

	if err := fxApp.Err(); err != nil {
		t.Fatalf("fx app returned error: %v", err)
	}
	t.Cleanup(func() { _ = fxApp.Stop(context.Background()) })
	if facade == nil {
		t.Fatal("expected loyalty facade instance")
	}
}
