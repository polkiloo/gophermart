package usecase

import (
	"context"
	"testing"

	domainErrors "github.com/polkiloo/gophermart/internal/domain/errors"
	"github.com/polkiloo/gophermart/internal/domain/model"
)

type stubOrderRepository struct {
	createFn func(context.Context, int64, string) (*model.Order, bool, error)
}

func (s stubOrderRepository) Create(ctx context.Context, userID int64, number string) (*model.Order, bool, error) {
	return s.createFn(ctx, userID, number)
}

func (stubOrderRepository) GetByNumber(context.Context, string) (*model.Order, error) {
	panic("not implemented")
}

func (stubOrderRepository) ListByUser(context.Context, int64) ([]model.Order, error) {
	panic("not implemented")
}

func (stubOrderRepository) SelectBatchForProcessing(context.Context, int) ([]model.Order, error) {
	panic("not implemented")
}

func (stubOrderRepository) UpdateStatus(context.Context, int64, model.OrderStatus, *float64) error {
	panic("not implemented")
}

func TestOrderUseCaseRegisterRejectsInvalidNumber(t *testing.T) {
	uc := NewOrderUseCase(stubOrderRepository{createFn: func(context.Context, int64, string) (*model.Order, bool, error) {
		t.Fatal("create should not be called for invalid number")
		return nil, false, nil
	}})

	if _, _, err := uc.Register(context.Background(), 1, "123"); err != domainErrors.ErrInvalidOrderNumber {
		t.Fatalf("expected invalid order number error, got %v", err)
	}
}

func TestOrderUseCaseRegisterSuccess(t *testing.T) {
	uc := NewOrderUseCase(stubOrderRepository{createFn: func(ctx context.Context, userID int64, number string) (*model.Order, bool, error) {
		if userID != 7 || number != "79927398713" {
			t.Fatalf("unexpected arguments: %d %s", userID, number)
		}
		return &model.Order{ID: 1, UserID: userID, Number: number}, true, nil
	}})

	order, created, err := uc.Register(context.Background(), 7, "79927398713")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Fatalf("expected order to be newly created")
	}
	if order.Number != "79927398713" {
		t.Fatalf("unexpected order number %s", order.Number)
	}
}

func TestOrderUseCaseRegisterPropagatesError(t *testing.T) {
	uc := NewOrderUseCase(stubOrderRepository{createFn: func(context.Context, int64, string) (*model.Order, bool, error) {
		return nil, false, domainErrors.ErrAlreadyExists
	}})

	if _, _, err := uc.Register(context.Background(), 1, "79927398713"); err != domainErrors.ErrAlreadyExists {
		t.Fatalf("expected repository error to be returned, got %v", err)
	}
}
