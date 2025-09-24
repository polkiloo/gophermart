package usecase

import (
	"context"

	domainErrors "github.com/polkiloo/gophermart/internal/domain/errors"
	"github.com/polkiloo/gophermart/internal/domain/model"
	"github.com/polkiloo/gophermart/internal/domain/repository"
)

// BalanceUseCase manages operations with loyalty balance.
type BalanceUseCase struct {
	balances    repository.BalanceRepository
	withdrawals repository.WithdrawalRepository
}

// NewBalanceUseCase constructs BalanceUseCase.
func NewBalanceUseCase(b repository.BalanceRepository, w repository.WithdrawalRepository) *BalanceUseCase {
	return &BalanceUseCase{balances: b, withdrawals: w}
}

// Summary returns aggregated balance info for user.
func (u *BalanceUseCase) Summary(ctx context.Context, userID int64) (*model.BalanceSummary, error) {
	return u.balances.GetSummary(ctx, userID)
}

// Withdraw performs withdrawal transaction for user.
func (u *BalanceUseCase) Withdraw(ctx context.Context, userID int64, order string, sum float64) error {
	if !ValidateOrderNumber(order) {
		return domainErrors.ErrInvalidOrderNumber
	}
	if sum <= 0 {
		return domainErrors.ErrInvalidAmount
	}
	if err := u.balances.Withdraw(ctx, userID, order, sum); err != nil {
		return err
	}
	return nil
}

// WithdrawalsHistory returns withdrawals sorted by time.
func (u *BalanceUseCase) WithdrawalsHistory(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
	return u.withdrawals.ListByUser(ctx, userID)
}
