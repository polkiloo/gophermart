package app

import (
	"context"
	"errors"
	"testing"

	domainErrors "github.com/polkiloo/gophermart/internal/domain/errors"
	"github.com/polkiloo/gophermart/internal/domain/model"
	testhelpers "github.com/polkiloo/gophermart/internal/test"
	"github.com/polkiloo/gophermart/internal/usecase"
)

func newFacade() (*LoyaltyFacade, *testhelpers.UserRepositoryStub, *testhelpers.OrderRepositoryStub, *testhelpers.BalanceRepositoryStub, *testhelpers.WithdrawalRepositoryStub, *testhelpers.AccrualProviderStub) {
	userRepo := testhelpers.NewUserRepositoryStub()
	strategy := testhelpers.StrategyStub{ParseFn: func(string) (int64, error) { return 99, nil }}
	authUC := usecase.NewAuthUseCase(userRepo, testhelpers.HasherStub{}, strategy)

	orderRepo := &testhelpers.OrderRepositoryStub{}
	orderUC := usecase.NewOrderUseCase(orderRepo)

	balanceRepo := &testhelpers.BalanceRepositoryStub{Summary: &model.BalanceSummary{Current: 10, Withdrawn: 5}}
	withdrawals := &testhelpers.WithdrawalRepositoryStub{Items: []model.Withdrawal{{OrderNumber: "123", Sum: 7}}}
	balanceUC := usecase.NewBalanceUseCase(balanceRepo, withdrawals)

	accrual := &testhelpers.AccrualProviderStub{}

	facade := NewLoyaltyFacade(authUC, orderUC, balanceUC, accrual)
	return facade, userRepo, orderRepo, balanceRepo, withdrawals, accrual
}

func TestLoyaltyFacadeAuth(t *testing.T) {
	facade, users, _, _, _, _ := newFacade()
	token, err := facade.Register(context.Background(), "user", "pass")
	if err != nil {
		t.Fatalf("register returned error: %v", err)
	}
	if token != "token" {
		t.Fatalf("unexpected token %q", token)
	}

	stored, err := users.GetByLogin(context.Background(), "user")
	if err != nil {
		t.Fatalf("user not stored: %v", err)
	}
	if stored.Login != "user" {
		t.Fatalf("unexpected stored login %q", stored.Login)
	}

	token, err = facade.Authenticate(context.Background(), "user", "pass")
	if err != nil {
		t.Fatalf("authenticate returned error: %v", err)
	}
	if token != "token" {
		t.Fatalf("unexpected token %q", token)
	}

	id, err := facade.ParseToken("anything")
	if err != nil {
		t.Fatalf("parse token returned error: %v", err)
	}
	if id != 99 {
		t.Fatalf("expected id 99, got %d", id)
	}
}

func TestLoyaltyFacadeOrders(t *testing.T) {
	facade, _, orders, _, _, _ := newFacade()
	orders.Orders = []model.Order{{Number: "1"}, {Number: "2"}}
	orders.Processing = []model.Order{{Number: "3"}}
	orders.CreateFn = func(context.Context, int64, string) (*model.Order, bool, error) {
		return &model.Order{ID: 10, Number: "1"}, true, nil
	}

	order, created, err := facade.UploadOrder(context.Background(), 7, "79927398713")
	if err != nil || !created || order == nil {
		t.Fatalf("unexpected upload result: order=%v created=%v err=%v", order, created, err)
	}

	listed, err := facade.Orders(context.Background(), 7)
	if err != nil || len(listed) != 2 {
		t.Fatalf("expected two orders, got %v err=%v", listed, err)
	}

	batch, err := facade.OrdersForProcessing(context.Background(), 1)
	if err != nil || len(batch) != 1 {
		t.Fatalf("expected batch of one, got %v err=%v", batch, err)
	}

	accr := 12.0
	if err := facade.UpdateOrderStatus(context.Background(), 1, model.OrderStatusProcessed, &accr); err != nil {
		t.Fatalf("update status error: %v", err)
	}
	if len(orders.UpdateCalls) != 1 {
		t.Fatalf("expected update call, got %d", len(orders.UpdateCalls))
	}
}

func TestLoyaltyFacadeBalance(t *testing.T) {
	facade, _, _, balances, withdrawals, _ := newFacade()

	summary, err := facade.Balance(context.Background(), 1)
	if err != nil {
		t.Fatalf("balance returned error: %v", err)
	}
	if summary.Current != 10 || summary.Withdrawn != 5 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	balances.Summary = nil
	summary, err = facade.Balance(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected nil error for not found, got %v", err)
	}
	if summary.Current != 0 || summary.Withdrawn != 0 {
		t.Fatalf("expected empty summary, got %+v", summary)
	}

	balances.WithdrawErr = domainErrors.ErrInsufficientBalance
	if err := facade.Withdraw(context.Background(), 1, "79927398713", 5); !errors.Is(err, domainErrors.ErrInsufficientBalance) {
		t.Fatalf("expected insufficient balance error, got %v", err)
	}
	balances.WithdrawErr = nil
	if err := facade.Withdraw(context.Background(), 1, "79927398713", 5); err != nil {
		t.Fatalf("expected successful withdraw, got %v", err)
	}

	list, err := facade.Withdrawals(context.Background(), 1)
	if err != nil || len(list) != len(withdrawals.Items) {
		t.Fatalf("unexpected withdrawals result: %v err=%v", list, err)
	}
}

func TestLoyaltyFacadeAccrual(t *testing.T) {
	facade, _, _, _, _, accrual := newFacade()
	accrual.Accrual = &model.Accrual{Order: "42"}
	result, err := facade.CheckAccrual(context.Background(), "42")
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if result.Order != "42" {
		t.Fatalf("unexpected accrual %v", result)
	}
}
