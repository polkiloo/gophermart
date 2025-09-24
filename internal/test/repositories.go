package test

import (
	"context"

	domainErrors "github.com/polkiloo/gophermart/internal/domain/errors"
	"github.com/polkiloo/gophermart/internal/domain/model"
)

// UserRepositoryStub stores users in-memory for tests.
type UserRepositoryStub struct {
	Users map[string]*model.User
	ByID  map[int64]*model.User
	Next  int64
	Err   error
}

// NewUserRepositoryStub constructs stub repository with initialized maps.
func NewUserRepositoryStub() *UserRepositoryStub {
	return &UserRepositoryStub{
		Users: make(map[string]*model.User),
		ByID:  make(map[int64]*model.User),
		Next:  1,
	}
}

// Create registers user unless already exists or stub has explicit error.
func (s *UserRepositoryStub) Create(ctx context.Context, login, passwordHash string) (*model.User, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	if s.Users == nil {
		s.Users = make(map[string]*model.User)
	}
	if s.ByID == nil {
		s.ByID = make(map[int64]*model.User)
	}
	if _, exists := s.Users[login]; exists {
		return nil, domainErrors.ErrAlreadyExists
	}
	if s.Next == 0 {
		s.Next = 1
	}
	user := &model.User{ID: s.Next, Login: login, PasswordHash: passwordHash}
	s.Next++
	s.Users[login] = user
	s.ByID[user.ID] = user
	return user, nil
}

// GetByLogin fetches user by login or returns not found.
func (s *UserRepositoryStub) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	if user, ok := s.Users[login]; ok {
		return user, nil
	}
	return nil, domainErrors.ErrNotFound
}

// GetByID fetches user by identifier or returns not found.
func (s *UserRepositoryStub) GetByID(ctx context.Context, id int64) (*model.User, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	if user, ok := s.ByID[id]; ok {
		return user, nil
	}
	return nil, domainErrors.ErrNotFound
}

// OrderRepositoryStub allows tests to customize behaviour.
type OrderRepositoryStub struct {
	CreateFn                   func(context.Context, int64, string) (*model.Order, bool, error)
	GetByNumberFn              func(context.Context, string) (*model.Order, error)
	ListByUserFn               func(context.Context, int64) ([]model.Order, error)
	SelectBatchForProcessingFn func(context.Context, int) ([]model.Order, error)
	UpdateStatusFn             func(context.Context, int64, model.OrderStatus, *float64) error

	Created []struct {
		UserID int64
		Number string
	}
	Orders      []model.Order
	Processing  []model.Order
	UpdateCalls []OrderUpdateCall
}

// Create tracks invocations and returns configured responses.
func (s *OrderRepositoryStub) Create(ctx context.Context, userID int64, number string) (*model.Order, bool, error) {
	s.Created = append(s.Created, struct {
		UserID int64
		Number string
	}{userID, number})
	if s.CreateFn != nil {
		return s.CreateFn(ctx, userID, number)
	}
	order := &model.Order{ID: 1, UserID: userID, Number: number}
	return order, true, nil
}

// GetByNumber returns matched order either via override or stored slice.
func (s *OrderRepositoryStub) GetByNumber(ctx context.Context, number string) (*model.Order, error) {
	if s.GetByNumberFn != nil {
		return s.GetByNumberFn(ctx, number)
	}
	for _, o := range s.Orders {
		if o.Number == number {
			order := o
			return &order, nil
		}
	}
	return nil, domainErrors.ErrNotFound
}

// ListByUser returns orders from configured slice.
func (s *OrderRepositoryStub) ListByUser(ctx context.Context, userID int64) ([]model.Order, error) {
	if s.ListByUserFn != nil {
		return s.ListByUserFn(ctx, userID)
	}
	return s.Orders, nil
}

// SelectBatchForProcessing returns queued orders for processing.
func (s *OrderRepositoryStub) SelectBatchForProcessing(ctx context.Context, limit int) ([]model.Order, error) {
	if s.SelectBatchForProcessingFn != nil {
		return s.SelectBatchForProcessingFn(ctx, limit)
	}
	return s.Processing, nil
}

// UpdateStatus records update invocations.
func (s *OrderRepositoryStub) UpdateStatus(ctx context.Context, orderID int64, status model.OrderStatus, accrual *float64) error {
	if s.UpdateStatusFn != nil {
		return s.UpdateStatusFn(ctx, orderID, status, accrual)
	}
	s.UpdateCalls = append(s.UpdateCalls, OrderUpdateCall{OrderID: orderID, Status: status, Accrual: accrual})
	return nil
}

// BalanceRepositoryStub lets tests control balance data.
type BalanceRepositoryStub struct {
	GetSummaryFn func(context.Context, int64) (*model.BalanceSummary, error)
	AddAccrualFn func(context.Context, int64, float64) error
	WithdrawFn   func(context.Context, int64, string, float64) error
	Summary      *model.BalanceSummary
	WithdrawErr  error
}

// GetSummary returns configured summary or default error.
func (s *BalanceRepositoryStub) GetSummary(ctx context.Context, userID int64) (*model.BalanceSummary, error) {
	if s.GetSummaryFn != nil {
		return s.GetSummaryFn(ctx, userID)
	}
	if s.Summary == nil {
		return nil, domainErrors.ErrNotFound
	}
	return s.Summary, nil
}

// AddAccrual applies override when provided.
func (s *BalanceRepositoryStub) AddAccrual(ctx context.Context, userID int64, sum float64) error {
	if s.AddAccrualFn != nil {
		return s.AddAccrualFn(ctx, userID, sum)
	}
	return nil
}

// Withdraw returns configured error or executes override.
func (s *BalanceRepositoryStub) Withdraw(ctx context.Context, userID int64, order string, sum float64) error {
	if s.WithdrawFn != nil {
		return s.WithdrawFn(ctx, userID, order, sum)
	}
	return s.WithdrawErr
}

// WithdrawalRepositoryStub stores withdrawals history for tests.
type WithdrawalRepositoryStub struct {
	ListFn func(context.Context, int64) ([]model.Withdrawal, error)
	Items  []model.Withdrawal
}

// ListByUser returns configured withdrawals.
func (s *WithdrawalRepositoryStub) ListByUser(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
	if s.ListFn != nil {
		return s.ListFn(ctx, userID)
	}
	return s.Items, nil
}
