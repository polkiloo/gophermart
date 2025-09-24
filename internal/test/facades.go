package test

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/polkiloo/gophermart/internal/domain/model"
)

// OrderFacadeStub provides controllable behaviour for order endpoints.
type OrderFacadeStub struct {
	UploadFn func(context.Context, int64, string) (*model.Order, bool, error)
	OrdersFn func(context.Context, int64) ([]model.Order, error)
}

// UploadOrder delegates to provided function or returns default order.
func (s OrderFacadeStub) UploadOrder(ctx context.Context, userID int64, number string) (*model.Order, bool, error) {
	if s.UploadFn != nil {
		return s.UploadFn(ctx, userID, number)
	}
	return &model.Order{Number: number, UserID: userID}, true, nil
}

// Orders returns predefined orders for given user.
func (s OrderFacadeStub) Orders(ctx context.Context, userID int64) ([]model.Order, error) {
	if s.OrdersFn != nil {
		return s.OrdersFn(ctx, userID)
	}
	return []model.Order{{Number: "1"}}, nil
}

// BalanceFacadeStub simulates balance operations.
type BalanceFacadeStub struct {
	BalanceFn     func(context.Context, int64) (*model.BalanceSummary, error)
	WithdrawFn    func(context.Context, int64, string, float64) error
	WithdrawalsFn func(context.Context, int64) ([]model.Withdrawal, error)
}

// Balance returns stored summary or default data.
func (s BalanceFacadeStub) Balance(ctx context.Context, userID int64) (*model.BalanceSummary, error) {
	if s.BalanceFn != nil {
		return s.BalanceFn(ctx, userID)
	}
	return &model.BalanceSummary{Current: 10, Withdrawn: 5}, nil
}

// Withdraw executes configured withdrawal handler.
func (s BalanceFacadeStub) Withdraw(ctx context.Context, userID int64, order string, sum float64) error {
	if s.WithdrawFn != nil {
		return s.WithdrawFn(ctx, userID, order, sum)
	}
	return nil
}

// Withdrawals returns preconfigured history.
func (s BalanceFacadeStub) Withdrawals(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
	if s.WithdrawalsFn != nil {
		return s.WithdrawalsFn(ctx, userID)
	}
	return []model.Withdrawal{{OrderNumber: "1", Sum: 1, ProcessedAt: time.Unix(0, 0)}}, nil
}

// OrderUpdateCall stores information about UpdateOrderStatus invocations.
type OrderUpdateCall struct {
	OrderID int64
	Status  model.OrderStatus
	Accrual *float64
}

// WorkerFacadeStub mimics worker interactions with loyalty facade.
type WorkerFacadeStub struct {
	Orders          [][]model.Order
	OrdersFn        func(context.Context, int) ([]model.Order, error)
	CheckFn         func(context.Context, string) (*model.Accrual, error)
	UpdateFn        func(context.Context, int64, model.OrderStatus, *float64) error
	Updates         []OrderUpdateCall
	mu              sync.Mutex
	ordersCallCount int32
}

// Lock exposes internal mutex for external synchronization.
func (s *WorkerFacadeStub) Lock() { s.mu.Lock() }

// Unlock releases previously acquired lock.
func (s *WorkerFacadeStub) Unlock() { s.mu.Unlock() }

// OrdersForProcessing returns batches from configured queue.
func (s *WorkerFacadeStub) OrdersForProcessing(ctx context.Context, limit int) ([]model.Order, error) {
	if s.OrdersFn != nil {
		return s.OrdersFn(ctx, limit)
	}
	call := atomic.AddInt32(&s.ordersCallCount, 1)
	if int(call) <= len(s.Orders) {
		return s.Orders[call-1], nil
	}
	time.Sleep(10 * time.Millisecond)
	return nil, nil
}

// CheckAccrual returns configured accrual data.
func (s *WorkerFacadeStub) CheckAccrual(ctx context.Context, number string) (*model.Accrual, error) {
	if s.CheckFn != nil {
		return s.CheckFn(ctx, number)
	}
	accrual := 5.0
	return &model.Accrual{Status: model.AccrualStatusProcessed, Accrual: &accrual}, nil
}

// UpdateOrderStatus records update requests.
func (s *WorkerFacadeStub) UpdateOrderStatus(ctx context.Context, orderID int64, status model.OrderStatus, accrual *float64) error {
	if s.UpdateFn != nil {
		return s.UpdateFn(ctx, orderID, status, accrual)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Updates = append(s.Updates, OrderUpdateCall{OrderID: orderID, Status: status, Accrual: accrual})
	return nil
}

// AccrualProviderStub fetches accrual information for tests.
type AccrualProviderStub struct {
	FetchFn func(context.Context, string) (*model.Accrual, error)
	Accrual *model.Accrual
	Err     error
}

// Fetch returns configured response or default processed status.
func (s AccrualProviderStub) Fetch(ctx context.Context, number string) (*model.Accrual, error) {
	if s.FetchFn != nil {
		return s.FetchFn(ctx, number)
	}
	if s.Err != nil {
		return nil, s.Err
	}
	if s.Accrual != nil {
		return s.Accrual, nil
	}
	return &model.Accrual{Order: number, Status: model.AccrualStatusProcessed}, nil
}
