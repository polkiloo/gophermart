package app

import (
	"context"

	domainErrors "github.com/polkiloo/gophermart/internal/domain/errors"
	"github.com/polkiloo/gophermart/internal/domain/model"
	"github.com/polkiloo/gophermart/internal/usecase"
)

type AccrualProvider interface {
	Fetch(ctx context.Context, number string) (*model.Accrual, error)
}

type LoyaltyFacade struct {
	auth     *usecase.AuthUseCase
	orders   *usecase.OrderUseCase
	balance  *usecase.BalanceUseCase
	accruals AccrualProvider
}

func NewLoyaltyFacade(auth *usecase.AuthUseCase, orders *usecase.OrderUseCase, balance *usecase.BalanceUseCase, accruals AccrualProvider) *LoyaltyFacade {
	return &LoyaltyFacade{auth: auth, orders: orders, balance: balance, accruals: accruals}
}

func (f *LoyaltyFacade) Register(ctx context.Context, login, password string) (string, error) {
	_, token, err := f.auth.Register(ctx, login, password)
	return token, err
}

func (f *LoyaltyFacade) Authenticate(ctx context.Context, login, password string) (string, error) {
	_, token, err := f.auth.Authenticate(ctx, login, password)
	return token, err
}

func (f *LoyaltyFacade) ParseToken(token string) (int64, error) {
	return f.auth.ParseToken(token)
}

func (f *LoyaltyFacade) UploadOrder(ctx context.Context, userID int64, number string) (*model.Order, bool, error) {
	return f.orders.Register(ctx, userID, number)
}

func (f *LoyaltyFacade) Orders(ctx context.Context, userID int64) ([]model.Order, error) {
	return f.orders.ListByUser(ctx, userID)
}

func (f *LoyaltyFacade) OrdersForProcessing(ctx context.Context, limit int) ([]model.Order, error) {
	return f.orders.SelectBatchForProcessing(ctx, limit)
}

func (f *LoyaltyFacade) UpdateOrderStatus(ctx context.Context, orderID int64, status model.OrderStatus, accrual *float64) error {
	return f.orders.UpdateStatus(ctx, orderID, status, accrual)
}

func (f *LoyaltyFacade) Balance(ctx context.Context, userID int64) (*model.BalanceSummary, error) {
	summary, err := f.balance.Summary(ctx, userID)
	if err != nil {
		if err == domainErrors.ErrNotFound {
			return &model.BalanceSummary{}, nil
		}
		return nil, err
	}
	return summary, nil
}

func (f *LoyaltyFacade) Withdraw(ctx context.Context, userID int64, order string, sum float64) error {
	return f.balance.Withdraw(ctx, userID, order, sum)
}

func (f *LoyaltyFacade) Withdrawals(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
	return f.balance.WithdrawalsHistory(ctx, userID)
}

func (f *LoyaltyFacade) CheckAccrual(ctx context.Context, number string) (*model.Accrual, error) {
	return f.accruals.Fetch(ctx, number)
}
