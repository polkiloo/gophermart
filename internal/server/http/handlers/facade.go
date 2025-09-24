package handlers

import (
	"context"

	"github.com/polkiloo/gophermart/internal/domain/model"
)

// AuthFacade describes authentication capabilities required by handlers.
type AuthFacade interface {
	Register(ctx context.Context, login, password string) (string, error)
	Authenticate(ctx context.Context, login, password string) (string, error)
	ParseToken(token string) (int64, error)
}

// OrderFacade encapsulates order operations exposed via HTTP.
type OrderFacade interface {
	UploadOrder(ctx context.Context, userID int64, number string) (*model.Order, bool, error)
	Orders(ctx context.Context, userID int64) ([]model.Order, error)
}

// BalanceFacade provides balance related operations.
type BalanceFacade interface {
	Balance(ctx context.Context, userID int64) (*model.BalanceSummary, error)
	Withdraw(ctx context.Context, userID int64, order string, sum float64) error
	Withdrawals(ctx context.Context, userID int64) ([]model.Withdrawal, error)
}

// LoyaltyFacade aggregates the full set of operations used across handlers.
type LoyaltyFacade interface {
	AuthFacade
	OrderFacade
	BalanceFacade
}
