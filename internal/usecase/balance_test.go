package usecase

import (
	"context"
	"testing"
	"time"

	domainErrors "github.com/polkiloo/gophermart/internal/domain/errors"
	"github.com/polkiloo/gophermart/internal/domain/model"
)

type stubBalanceRepository struct {
	withdrawFn func(context.Context, int64, string, float64) error
	summaryFn  func(context.Context, int64) (*model.BalanceSummary, error)
}

func (s stubBalanceRepository) GetSummary(ctx context.Context, userID int64) (*model.BalanceSummary, error) {
	return s.summaryFn(ctx, userID)
}

func (stubBalanceRepository) AddAccrual(context.Context, int64, float64) error {
	return nil
}

func (s stubBalanceRepository) Withdraw(ctx context.Context, userID int64, order string, sum float64) error {
	return s.withdrawFn(ctx, userID, order, sum)
}

type stubWithdrawalRepository struct {
	listFn func(context.Context, int64) ([]model.Withdrawal, error)
}

func (s stubWithdrawalRepository) ListByUser(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
	return s.listFn(ctx, userID)
}

func TestBalanceUseCaseWithdrawValidation(t *testing.T) {
	uc := NewBalanceUseCase(
		stubBalanceRepository{withdrawFn: func(context.Context, int64, string, float64) error {
			t.Fatal("withdraw should not be called on validation errors")
			return nil
		}, summaryFn: func(context.Context, int64) (*model.BalanceSummary, error) {
			return &model.BalanceSummary{}, nil
		}},
		&stubWithdrawalRepository{listFn: func(context.Context, int64) ([]model.Withdrawal, error) { return nil, nil }},
	)

	if err := uc.Withdraw(context.Background(), 1, "123", 10); err != domainErrors.ErrInvalidOrderNumber {
		t.Fatalf("expected invalid order error, got %v", err)
	}
	if err := uc.Withdraw(context.Background(), 1, "79927398713", -5); err != domainErrors.ErrInvalidAmount {
		t.Fatalf("expected invalid amount error, got %v", err)
	}
}

func TestBalanceUseCaseWithdrawPropagatesError(t *testing.T) {
	uc := NewBalanceUseCase(
		stubBalanceRepository{withdrawFn: func(context.Context, int64, string, float64) error {
			return domainErrors.ErrInsufficientBalance
		}, summaryFn: func(context.Context, int64) (*model.BalanceSummary, error) {
			return &model.BalanceSummary{}, nil
		}},
		&stubWithdrawalRepository{listFn: func(context.Context, int64) ([]model.Withdrawal, error) { return nil, nil }},
	)

	if err := uc.Withdraw(context.Background(), 1, "79927398713", 5); err != domainErrors.ErrInsufficientBalance {
		t.Fatalf("expected insufficient balance error, got %v", err)
	}
}

func TestBalanceUseCaseWithdrawSuccess(t *testing.T) {
	called := false
	uc := NewBalanceUseCase(
		stubBalanceRepository{withdrawFn: func(ctx context.Context, userID int64, order string, sum float64) error {
			called = true
			if userID != 42 || order != "79927398713" || sum != 5 {
				t.Fatalf("unexpected arguments: %d %s %f", userID, order, sum)
			}
			return nil
		}, summaryFn: func(context.Context, int64) (*model.BalanceSummary, error) {
			return &model.BalanceSummary{}, nil
		}},
		&stubWithdrawalRepository{listFn: func(context.Context, int64) ([]model.Withdrawal, error) { return nil, nil }},
	)

	if err := uc.Withdraw(context.Background(), 42, "79927398713", 5); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected withdraw to be invoked")
	}
}

func TestBalanceUseCaseSummaryAndHistory(t *testing.T) {
	summary := &model.BalanceSummary{Current: 10, Withdrawn: 2}
	withdrawals := []model.Withdrawal{{OrderNumber: "1", Sum: 1, ProcessedAt: time.Now()}}
	uc := NewBalanceUseCase(
		stubBalanceRepository{withdrawFn: func(context.Context, int64, string, float64) error { return nil }, summaryFn: func(context.Context, int64) (*model.BalanceSummary, error) {
			return summary, nil
		}},
		&stubWithdrawalRepository{listFn: func(context.Context, int64) ([]model.Withdrawal, error) {
			return withdrawals, nil
		}},
	)

	gotSummary, err := uc.Summary(context.Background(), 1)
	if err != nil {
		t.Fatalf("summary returned error: %v", err)
	}
	if gotSummary.Current != summary.Current || gotSummary.Withdrawn != summary.Withdrawn {
		t.Fatalf("unexpected summary: %+v", gotSummary)
	}

	gotWithdrawals, err := uc.WithdrawalsHistory(context.Background(), 1)
	if err != nil {
		t.Fatalf("withdrawals returned error: %v", err)
	}
	if len(gotWithdrawals) != len(withdrawals) {
		t.Fatalf("expected %d withdrawals, got %d", len(withdrawals), len(gotWithdrawals))
	}
}
