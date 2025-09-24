package repository

import (
	"context"

	"github.com/polkiloo/gophermart/internal/domain/model"
)

// BalanceRepository manages user loyalty balance operations.
type BalanceRepository interface {
	GetSummary(ctx context.Context, userID int64) (*model.BalanceSummary, error)
	AddAccrual(ctx context.Context, userID int64, sum float64) error
	Withdraw(ctx context.Context, userID int64, order string, sum float64) error
}
