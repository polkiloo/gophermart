package postgres

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v3"

	domainErrors "github.com/polkiloo/gophermart/internal/domain/errors"
	"github.com/polkiloo/gophermart/internal/domain/model"
)

func newMockStorage(t *testing.T) (*Storage, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	storage := &Storage{pool: mock, logger: logger}
	return storage, mock
}

func TestWithinTransactionCommit(t *testing.T) {
	storage, mock := newMockStorage(t)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectCommit()

	if err := storage.WithinTransaction(context.Background(), func(tx pgx.Tx) error { return nil }); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestWithinTransactionRollback(t *testing.T) {
	storage, mock := newMockStorage(t)
	defer mock.Close()

	mock.ExpectBegin()
	mock.ExpectRollback()

	err := storage.WithinTransaction(context.Background(), func(tx pgx.Tx) error { return context.Canceled })
	if err != context.Canceled {
		t.Fatalf("expected canceled error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestUserRepositoryCreate(t *testing.T) {
	storage, mock := newMockStorage(t)
	defer mock.Close()
	repo := &userRepository{storage: storage}

	createdAt := time.Now()
	mock.ExpectQuery("INSERT INTO users").WithArgs("user", "hash").WillReturnRows(
		pgxmock.NewRows([]string{"id", "created_at"}).AddRow(int64(1), createdAt),
	)

	user, err := repo.Create(context.Background(), "user", "hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Login != "user" || user.PasswordHash != "hash" {
		t.Fatalf("unexpected user: %+v", user)
	}

	mock.ExpectQuery("INSERT INTO users").WithArgs("user", "hash").WillReturnError(&pgconn.PgError{Code: "23505"})
	if _, err := repo.Create(context.Background(), "user", "hash"); !errors.Is(err, domainErrors.ErrAlreadyExists) {
		t.Fatalf("expected already exists error, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestOrderRepositoryListByUser(t *testing.T) {
	storage, mock := newMockStorage(t)
	defer mock.Close()
	repo := &orderRepository{storage: storage}

	mock.ExpectQuery("SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE user_id=").WithArgs(int64(1)).WillReturnRows(
		pgxmock.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at", "updated_at"}).
			AddRow(int64(1), int64(1), "1", model.OrderStatusProcessed, nil, time.Now(), time.Now()),
	)

	orders, err := repo.ListByUser(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orders) != 1 || orders[0].Number != "1" {
		t.Fatalf("unexpected orders: %+v", orders)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestBalanceRepositoryGetSummary(t *testing.T) {
	storage, mock := newMockStorage(t)
	defer mock.Close()
	repo := &balanceRepository{storage: storage}

	mock.ExpectQuery("SELECT current, withdrawn FROM balances WHERE user_id=").WithArgs(int64(1)).WillReturnRows(
		pgxmock.NewRows([]string{"current", "withdrawn"}).AddRow(1.0, 2.0),
	)
	summary, err := repo.GetSummary(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Current != 1 || summary.Withdrawn != 2 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	mock.ExpectQuery("SELECT current, withdrawn FROM balances WHERE user_id=").WithArgs(int64(2)).WillReturnError(pgx.ErrNoRows)
	summary, err = repo.GetSummary(context.Background(), 2)
	if err != nil {
		t.Fatalf("expected nil error for missing user, got %v", err)
	}
	if summary.Current != 0 || summary.Withdrawn != 0 {
		t.Fatalf("expected zero summary, got %+v", summary)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestWithdrawalRepositoryListByUser(t *testing.T) {
	storage, mock := newMockStorage(t)
	defer mock.Close()
	repo := &withdrawalRepository{storage: storage}

	processedAt := time.Now()
	mock.ExpectQuery("SELECT id, user_id, order_number, sum, processed_at FROM withdrawals").WithArgs(int64(1)).WillReturnRows(
		pgxmock.NewRows([]string{"id", "user_id", "order_number", "sum", "processed_at"}).AddRow(int64(1), int64(1), "1", 5.0, processedAt),
	)
	withdrawals, err := repo.ListByUser(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(withdrawals) != 1 || withdrawals[0].OrderNumber != "1" {
		t.Fatalf("unexpected withdrawals: %+v", withdrawals)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
