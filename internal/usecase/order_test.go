package usecase

import (
	"context"
	"testing"

	domainErrors "github.com/polkiloo/gophermart/internal/domain/errors"
	"github.com/polkiloo/gophermart/internal/domain/model"
	testhelpers "github.com/polkiloo/gophermart/internal/test"
)

func TestOrderUseCaseRegisterRejectsInvalidNumber(t *testing.T) {
	uc := NewOrderUseCase(&testhelpers.OrderRepositoryStub{CreateFn: func(context.Context, int64, string) (*model.Order, bool, error) {
		t.Fatal("create should not be called for invalid number")
		return nil, false, nil
	}})

	if _, _, err := uc.Register(context.Background(), 1, "123"); err != domainErrors.ErrInvalidOrderNumber {
		t.Fatalf("expected invalid order number error, got %v", err)
	}
}

func TestOrderUseCaseRegisterSuccess(t *testing.T) {
	repo := &testhelpers.OrderRepositoryStub{CreateFn: func(ctx context.Context, userID int64, number string) (*model.Order, bool, error) {
		if userID != 7 || number != "79927398713" {
			t.Fatalf("unexpected arguments: %d %s", userID, number)
		}
		return &model.Order{ID: 1, UserID: userID, Number: number}, true, nil
	}}

	uc := NewOrderUseCase(repo)

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
	repo := &testhelpers.OrderRepositoryStub{CreateFn: func(context.Context, int64, string) (*model.Order, bool, error) {
		return nil, false, domainErrors.ErrAlreadyExists
	}}
	uc := NewOrderUseCase(repo)

	if _, _, err := uc.Register(context.Background(), 1, "79927398713"); err != domainErrors.ErrAlreadyExists {
		t.Fatalf("expected repository error to be returned, got %v", err)
	}
}

func TestOrderUseCaseListAndProcessing(t *testing.T) {
	repo := &testhelpers.OrderRepositoryStub{
		Orders:     []model.Order{{Number: "1"}},
		Processing: []model.Order{{Number: "2"}},
	}
	uc := NewOrderUseCase(repo)

	orders, err := uc.ListByUser(context.Background(), 1)
	if err != nil || len(orders) != 1 {
		t.Fatalf("unexpected list result: %v %v", orders, err)
	}

	processing, err := uc.SelectBatchForProcessing(context.Background(), 1)
	if err != nil || len(processing) != 1 {
		t.Fatalf("unexpected processing result: %v %v", processing, err)
	}
}

func TestOrderUseCaseUpdateStatus(t *testing.T) {
	repo := &testhelpers.OrderRepositoryStub{}
	uc := NewOrderUseCase(repo)
	if err := uc.UpdateStatus(context.Background(), 1, model.OrderStatusProcessed, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.UpdateCalls) != 1 {
		t.Fatalf("expected update call to be recorded")
	}
}
