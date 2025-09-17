package repository

import (
	"context"

	"github.com/polkiloo/gophermart/internal/domain/model"
)

// WithdrawalRepository provides access to withdrawals history.
type WithdrawalRepository interface {
	ListByUser(ctx context.Context, userID int64) ([]model.Withdrawal, error)
}
