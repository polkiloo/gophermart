package repository

import (
	"context"

	"github.com/polkiloo/gophermart/internal/domain/model"
)

// OrderRepository describes persistence operations with orders.
type OrderRepository interface {
	Create(ctx context.Context, userID int64, number string) (*model.Order, bool, error)
	GetByNumber(ctx context.Context, number string) (*model.Order, error)
	ListByUser(ctx context.Context, userID int64) ([]model.Order, error)
	SelectBatchForProcessing(ctx context.Context, limit int) ([]model.Order, error)
	UpdateStatus(ctx context.Context, orderID int64, status model.OrderStatus, accrual *float64) error
}
