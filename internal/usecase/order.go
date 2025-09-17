package usecase

import (
	"context"

	domainErrors "github.com/polkiloo/gophermart/internal/domain/errors"
	"github.com/polkiloo/gophermart/internal/domain/model"
	"github.com/polkiloo/gophermart/internal/domain/repository"
)

// OrderUseCase encapsulates order lifecycle logic.
type OrderUseCase struct {
	orders repository.OrderRepository
}

// NewOrderUseCase constructs OrderUseCase.
func NewOrderUseCase(orders repository.OrderRepository) *OrderUseCase {
	return &OrderUseCase{orders: orders}
}

// Register registers new order for processing. Returns whether order was newly created.
func (u *OrderUseCase) Register(ctx context.Context, userID int64, number string) (*model.Order, bool, error) {
	if !ValidateOrderNumber(number) {
		return nil, false, domainErrors.ErrInvalidOrderNumber
	}

	order, created, err := u.orders.Create(ctx, userID, number)
	if err != nil {
		return nil, false, err
	}

	return order, created, nil
}

// ListByUser returns orders sorted by upload time.
func (u *OrderUseCase) ListByUser(ctx context.Context, userID int64) ([]model.Order, error) {
	return u.orders.ListByUser(ctx, userID)
}

// SelectBatchForProcessing returns pending orders to process.
func (u *OrderUseCase) SelectBatchForProcessing(ctx context.Context, limit int) ([]model.Order, error) {
	return u.orders.SelectBatchForProcessing(ctx, limit)
}

// UpdateStatus persists status/accrual for order.
func (u *OrderUseCase) UpdateStatus(ctx context.Context, orderID int64, status model.OrderStatus, accrual *float64) error {
	return u.orders.UpdateStatus(ctx, orderID, status, accrual)
}
