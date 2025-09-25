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
	"github.com/jackc/pgx/v5/pgxpool"
	pgxmockv3 "github.com/pashagolub/pgxmock/v3"
	"go.uber.org/fx/fxtest"

	"github.com/polkiloo/gophermart/internal/config"
	domainErrors "github.com/polkiloo/gophermart/internal/domain/errors"
	"github.com/polkiloo/gophermart/internal/domain/model"
)

func newMockStorage(t *testing.T) (*Storage, pgxmockv3.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmockv3.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	storage := &Storage{pool: mock, logger: logger}
	return storage, mock
}

func expectSchema(mock pgxmockv3.PgxPoolIface) {
	tableStatements := []string{
		"CREATE TABLE IF NOT EXISTS users",
		"CREATE TABLE IF NOT EXISTS orders",
		"CREATE TABLE IF NOT EXISTS balances",
		"CREATE TABLE IF NOT EXISTS withdrawals",
	}
	for _, stmt := range tableStatements {
		mock.ExpectExec(stmt).WillReturnResult(pgxmockv3.NewResult("CREATE", 0))
	}
	mock.ExpectExec("CREATE INDEX IF NOT EXISTS idx_orders_user ON orders").WillReturnResult(pgxmockv3.NewResult("CREATE", 0))
	mock.ExpectExec("CREATE INDEX IF NOT EXISTS idx_withdrawals_user ON withdrawals").WillReturnResult(pgxmockv3.NewResult("CREATE", 0))
}

type errorRows struct {
	err error
}

func (r *errorRows) Close()                                       {}
func (r *errorRows) Err() error                                   { return r.err }
func (r *errorRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *errorRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *errorRows) Next() bool                                   { return false }
func (r *errorRows) Scan(dest ...any) error                       { return nil }
func (r *errorRows) Values() ([]any, error)                       { return nil, nil }
func (r *errorRows) RawValues() [][]byte                          { return nil }
func (r *errorRows) Conn() *pgx.Conn                              { return nil }

type rowsErrorPool struct {
	rows pgx.Rows
}

func (p *rowsErrorPool) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (p *rowsErrorPool) Query(context.Context, string, ...any) (pgx.Rows, error) { return p.rows, nil }
func (p *rowsErrorPool) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }
func (p *rowsErrorPool) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return nil, errors.New("not implemented")
}
func (p *rowsErrorPool) Ping(context.Context) error { return nil }
func (p *rowsErrorPool) Close()                     {}

type rowsErrorTx struct {
	rows pgx.Rows
}

func (tx *rowsErrorTx) Begin(context.Context) (pgx.Tx, error) {
	return nil, errors.New("not implemented")
}
func (tx *rowsErrorTx) Commit(context.Context) error   { return nil }
func (tx *rowsErrorTx) Rollback(context.Context) error { return nil }
func (tx *rowsErrorTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, errors.New("not implemented")
}
func (tx *rowsErrorTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (tx *rowsErrorTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (tx *rowsErrorTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, errors.New("not implemented")
}
func (tx *rowsErrorTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (tx *rowsErrorTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return tx.rows, nil }
func (tx *rowsErrorTx) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }
func (tx *rowsErrorTx) Conn() *pgx.Conn                                         { return nil }

type rowsErrorTxPool struct {
	tx pgx.Tx
}

func (p *rowsErrorTxPool) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (p *rowsErrorTxPool) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}
func (p *rowsErrorTxPool) QueryRow(context.Context, string, ...any) pgx.Row       { return nil }
func (p *rowsErrorTxPool) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) { return p.tx, nil }
func (p *rowsErrorTxPool) Ping(context.Context) error                             { return nil }
func (p *rowsErrorTxPool) Close()                                                 {}

func TestNew(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	t.Run("parse error", func(t *testing.T) {
		if _, err := New(context.Background(), ":://bad", logger); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("pool creation error", func(t *testing.T) {
		t.Cleanup(func() {
			newPgxPool = func(ctx context.Context, cfg *pgxpool.Config) (pgxPool, error) {
				return pgxpool.NewWithConfig(ctx, cfg)
			}
		})
		newPgxPool = func(context.Context, *pgxpool.Config) (pgxPool, error) {
			return nil, errors.New("boom")
		}
		if _, err := New(context.Background(), "postgres://user:pass@localhost/db", logger); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("init schema success", func(t *testing.T) {
		_, mock := newMockStorage(t)
		defer mock.Close()

		t.Cleanup(func() {
			newPgxPool = func(ctx context.Context, cfg *pgxpool.Config) (pgxPool, error) {
				return pgxpool.NewWithConfig(ctx, cfg)
			}
		})
		newPgxPool = func(context.Context, *pgxpool.Config) (pgxPool, error) { return mock, nil }
		expectSchema(mock)

		st, err := New(context.Background(), "postgres://user:pass@localhost/db", logger)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations not met: %v", err)
		}
		st.Close()
	})

	t.Run("init schema failure closes pool", func(t *testing.T) {
		_, mock := newMockStorage(t)
		defer mock.Close()

		t.Cleanup(func() {
			newPgxPool = func(ctx context.Context, cfg *pgxpool.Config) (pgxPool, error) {
				return pgxpool.NewWithConfig(ctx, cfg)
			}
		})
		newPgxPool = func(context.Context, *pgxpool.Config) (pgxPool, error) { return mock, nil }

		mock.ExpectExec("CREATE TABLE IF NOT EXISTS users").WillReturnError(errors.New("fail"))
		mock.ExpectClose()

		if _, err := New(context.Background(), "postgres://user:pass@localhost/db", logger); err == nil {
			t.Fatal("expected error")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("expectations not met: %v", err)
		}
	})
}

func TestStorageClose(t *testing.T) {
	storage := &Storage{}
	storage.Close()

	storage, mock := newMockStorage(t)
	mock.ExpectClose()
	storage.Close()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
	mock.Close()
}

func TestRepositoryFactories(t *testing.T) {
	storage, mock := newMockStorage(t)
	defer mock.Close()

	if _, ok := storage.Users().(*userRepository); !ok {
		t.Fatalf("unexpected user repo type")
	}
	if _, ok := storage.Orders().(*orderRepository); !ok {
		t.Fatalf("unexpected order repo type")
	}
	if _, ok := storage.Balances().(*balanceRepository); !ok {
		t.Fatalf("unexpected balance repo type")
	}
	if _, ok := storage.Withdrawals().(*withdrawalRepository); !ok {
		t.Fatalf("unexpected withdrawal repo type")
	}
}

func TestInitSchema(t *testing.T) {
	storage, mock := newMockStorage(t)
	defer mock.Close()

	expectSchema(mock)

	if err := storage.initSchema(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS users").WillReturnError(errors.New("boom"))
	if err := storage.initSchema(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestWithinTransaction(t *testing.T) {
	storage, mock := newMockStorage(t)
	defer mock.Close()

	t.Run("commit", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectCommit()
		if err := storage.WithinTransaction(context.Background(), func(pgx.Tx) error { return nil }); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rollback", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectRollback()
		if err := storage.WithinTransaction(context.Background(), func(pgx.Tx) error { return context.Canceled }); err != context.Canceled {
			t.Fatalf("expected canceled, got %v", err)
		}
	})

	t.Run("commit error", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectCommit().WillReturnError(errors.New("commit fail"))
		if err := storage.WithinTransaction(context.Background(), func(pgx.Tx) error { return nil }); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("begin error", func(t *testing.T) {
		mock.ExpectBegin().WillReturnError(errors.New("begin"))
		if err := storage.WithinTransaction(context.Background(), func(pgx.Tx) error { return nil }); err == nil {
			t.Fatal("expected begin error")
		}
	})

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestUserRepository(t *testing.T) {
	storage, mock := newMockStorage(t)
	defer mock.Close()
	repo := &userRepository{storage: storage}

	createdAt := time.Now()
	mock.ExpectQuery("INSERT INTO users").WithArgs("user", "hash").WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "created_at"}).AddRow(int64(1), createdAt),
	)
	user, err := repo.Create(context.Background(), "user", "hash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 1 || user.Login != "user" {
		t.Fatalf("unexpected user: %+v", user)
	}

	mock.ExpectQuery("INSERT INTO users").WithArgs("user", "hash").WillReturnError(&pgconn.PgError{Code: "23505"})
	if _, err := repo.Create(context.Background(), "user", "hash"); !errors.Is(err, domainErrors.ErrAlreadyExists) {
		t.Fatalf("expected already exists error, got %v", err)
	}

	mock.ExpectQuery("INSERT INTO users").WithArgs("user", "hash").WillReturnError(errors.New("other"))
	if _, err := repo.Create(context.Background(), "user", "hash"); err == nil {
		t.Fatal("expected error")
	}

	mock.ExpectQuery("SELECT id, login, password_hash, created_at FROM users WHERE login=").WithArgs("user").WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "login", "password_hash", "created_at"}).AddRow(int64(1), "user", "hash", createdAt))
	if _, err := repo.GetByLogin(context.Background(), "user"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mock.ExpectQuery("SELECT id, login, password_hash, created_at FROM users WHERE login=").WithArgs("missing").WillReturnError(pgx.ErrNoRows)
	if _, err := repo.GetByLogin(context.Background(), "missing"); !errors.Is(err, domainErrors.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}

	mock.ExpectQuery("SELECT id, login, password_hash, created_at FROM users WHERE login=").WithArgs("err").WillReturnError(errors.New("fail"))
	if _, err := repo.GetByLogin(context.Background(), "err"); err == nil {
		t.Fatal("expected error")
	}

	mock.ExpectQuery("SELECT id, login, password_hash, created_at FROM users WHERE id=").WithArgs(int64(1)).WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "login", "password_hash", "created_at"}).AddRow(int64(1), "user", "hash", createdAt))
	if _, err := repo.GetByID(context.Background(), 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mock.ExpectQuery("SELECT id, login, password_hash, created_at FROM users WHERE id=").WithArgs(int64(2)).WillReturnError(pgx.ErrNoRows)
	if _, err := repo.GetByID(context.Background(), 2); !errors.Is(err, domainErrors.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}

	mock.ExpectQuery("SELECT id, login, password_hash, created_at FROM users WHERE id=").WithArgs(int64(3)).WillReturnError(errors.New("boom"))
	if _, err := repo.GetByID(context.Background(), 3); err == nil {
		t.Fatal("expected error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestOrderRepositoryCreate(t *testing.T) {
	storage, mock := newMockStorage(t)
	defer mock.Close()
	repo := &orderRepository{storage: storage}

	uploadedAt := time.Now()
	mock.ExpectQuery("INSERT INTO orders").WithArgs(int64(1), "order", model.OrderStatusNew).WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "status", "uploaded_at", "updated_at"}).AddRow(int64(10), model.OrderStatusNew, uploadedAt, uploadedAt))
	order, created, err := repo.Create(context.Background(), 1, "order")
	if err != nil || !created || order.ID != 10 {
		t.Fatalf("unexpected result: order=%+v created=%v err=%v", order, created, err)
	}

	existingUploaded := time.Now()
	mock.ExpectQuery("INSERT INTO orders").WithArgs(int64(1), "order", model.OrderStatusNew).WillReturnError(pgx.ErrNoRows)
	mock.ExpectQuery("SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE number=").WithArgs("order").WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at", "updated_at"}).AddRow(int64(10), int64(1), "order", model.OrderStatusProcessing, nil, existingUploaded, existingUploaded))
	order, created, err = repo.Create(context.Background(), 1, "order")
	if err != nil || created || order.Status != model.OrderStatusProcessing {
		t.Fatalf("unexpected result: order=%+v created=%v err=%v", order, created, err)
	}

	mock.ExpectQuery("INSERT INTO orders").WithArgs(int64(1), "order", model.OrderStatusNew).WillReturnError(pgx.ErrNoRows)
	mock.ExpectQuery("SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE number=").WithArgs("order").WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at", "updated_at"}).AddRow(int64(11), int64(2), "order", model.OrderStatusProcessing, nil, existingUploaded, existingUploaded))
	_, _, err = repo.Create(context.Background(), 1, "order")
	if !errors.Is(err, domainErrors.ErrAlreadyExists) {
		t.Fatalf("expected already exists error, got %v", err)
	}

	mock.ExpectQuery("INSERT INTO orders").WithArgs(int64(1), "order", model.OrderStatusNew).WillReturnError(pgx.ErrNoRows)
	mock.ExpectQuery("SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE number=").WithArgs("order").WillReturnError(errors.New("lookup"))
	if _, _, err := repo.Create(context.Background(), 1, "order"); err == nil {
		t.Fatal("expected error")
	}

	mock.ExpectQuery("INSERT INTO orders").WithArgs(int64(1), "order", model.OrderStatusNew).WillReturnError(errors.New("insert"))
	if _, _, err := repo.Create(context.Background(), 1, "order"); err == nil {
		t.Fatal("expected error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestOrderRepositoryGetAndList(t *testing.T) {
	storage, mock := newMockStorage(t)
	defer mock.Close()
	repo := &orderRepository{storage: storage}

	now := time.Now()
	mock.ExpectQuery("SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE number=").WithArgs("num").WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at", "updated_at"}).AddRow(int64(1), int64(2), "num", model.OrderStatusProcessed, nil, now, now))
	order, err := repo.GetByNumber(context.Background(), "num")
	if err != nil || order.Number != "num" || order.Status != model.OrderStatusProcessed {
		t.Fatalf("unexpected order: %+v err=%v", order, err)
	}

	mock.ExpectQuery("SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE number=").WithArgs("missing").WillReturnError(pgx.ErrNoRows)
	if _, err := repo.GetByNumber(context.Background(), "missing"); !errors.Is(err, domainErrors.ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}

	mock.ExpectQuery("SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE number=").WithArgs("err").WillReturnError(errors.New("fail"))
	if _, err := repo.GetByNumber(context.Background(), "err"); err == nil {
		t.Fatal("expected error")
	}

	mock.ExpectQuery("SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE user_id=").WithArgs(int64(1)).WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at", "updated_at"}).
			AddRow(int64(1), int64(1), "1", model.OrderStatusNew, nil, now, now).
			AddRow(int64(2), int64(1), "2", model.OrderStatusProcessing, nil, now, now),
	)
	orders, err := repo.ListByUser(context.Background(), 1)
	if err != nil || len(orders) != 2 {
		t.Fatalf("unexpected result: %v err=%v", orders, err)
	}

	mock.ExpectQuery("SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE user_id=").WithArgs(int64(2)).WillReturnError(errors.New("query"))
	if _, err := repo.ListByUser(context.Background(), 2); err == nil {
		t.Fatal("expected error")
	}

	mock.ExpectQuery("SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE user_id=").WithArgs(int64(3)).WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at", "updated_at"}).AddRow("bad", int64(1), "2", model.OrderStatusProcessing, nil, now, now),
	)
	if _, err := repo.ListByUser(context.Background(), 3); err == nil {
		t.Fatal("expected scan error")
	}

	mock.ExpectQuery("SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE user_id=").WithArgs(int64(4)).WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at", "updated_at"}).
			AddRow(int64(1), int64(1), "1", model.OrderStatusNew, nil, now, now).
			AddRow(int64(2), int64(1), "2", model.OrderStatusProcessing, nil, now, now).
			RowError(1, errors.New("row err")),
	)
	if _, err := repo.ListByUser(context.Background(), 4); err == nil || err.Error() != "row err" {
		t.Fatalf("expected row err, got %v", err)
	}

	mock.ExpectQuery("SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE user_id=").WithArgs(int64(5)).WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at", "updated_at"}),
	)
	orders, err = repo.ListByUser(context.Background(), 5)
	if err != nil || len(orders) != 0 {
		t.Fatalf("expected empty result, got %v err=%v", orders, err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestOrderRepositoryListByUserRowsError(t *testing.T) {
	storage := &Storage{pool: &rowsErrorPool{rows: &errorRows{err: errors.New("rows err")}}}
	repo := &orderRepository{storage: storage}

	if _, err := repo.ListByUser(context.Background(), 1); err == nil || err.Error() != "rows err" {
		t.Fatalf("expected rows err, got %v", err)
	}
}

func TestSelectBatchForProcessing(t *testing.T) {
	storage, mock := newMockStorage(t)
	defer mock.Close()
	repo := &orderRepository{storage: storage}

	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE status IN").WithArgs(5).WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at", "updated_at"}).
			AddRow(int64(1), int64(1), "1", model.OrderStatusNew, nil, now, now).
			AddRow(int64(2), int64(2), "2", model.OrderStatusProcessing, nil, now, now),
	)
	mock.ExpectExec("UPDATE orders SET status='PROCESSING'").WithArgs(pgxmockv3.AnyArg()).WillReturnResult(pgxmockv3.NewResult("UPDATE", 2))
	mock.ExpectCommit()

	orders, err := repo.SelectBatchForProcessing(context.Background(), 5)
	if err != nil || len(orders) != 2 || orders[0].Status != model.OrderStatusProcessing {
		t.Fatalf("unexpected result: %v err=%v", orders, err)
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE status IN").WithArgs(1).WillReturnRows(pgxmockv3.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at", "updated_at"}))
	mock.ExpectCommit()
	orders, err = repo.SelectBatchForProcessing(context.Background(), 1)
	if err != nil || len(orders) != 0 {
		t.Fatalf("expected empty slice: %v err=%v", orders, err)
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE status IN").WithArgs(1).WillReturnError(errors.New("query"))
	mock.ExpectRollback()
	if _, err := repo.SelectBatchForProcessing(context.Background(), 1); err == nil {
		t.Fatal("expected error")
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE status IN").WithArgs(1).WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at", "updated_at"}).AddRow("bad", int64(1), "1", model.OrderStatusNew, nil, now, now),
	)
	mock.ExpectRollback()
	if _, err := repo.SelectBatchForProcessing(context.Background(), 1); err == nil {
		t.Fatal("expected scan error")
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE status IN").WithArgs(1).WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at", "updated_at"}).
			AddRow(int64(1), int64(1), "1", model.OrderStatusNew, nil, now, now).
			RowError(0, errors.New("row")),
	)
	mock.ExpectRollback()
	if _, err := repo.SelectBatchForProcessing(context.Background(), 1); err == nil {
		t.Fatal("expected rows error")
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE status IN").WithArgs(1).WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "user_id", "number", "status", "accrual", "uploaded_at", "updated_at"}).AddRow(int64(1), int64(1), "1", model.OrderStatusNew, nil, now, now),
	)
	mock.ExpectExec("UPDATE orders SET status='PROCESSING'").WithArgs(pgxmockv3.AnyArg()).WillReturnError(errors.New("update"))
	mock.ExpectRollback()
	if _, err := repo.SelectBatchForProcessing(context.Background(), 1); err == nil {
		t.Fatal("expected update error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestSelectBatchForProcessingRowsError(t *testing.T) {
	rows := &errorRows{err: errors.New("rows err")}
	tx := &rowsErrorTx{rows: rows}
	storage := &Storage{pool: &rowsErrorTxPool{tx: tx}}
	repo := &orderRepository{storage: storage}

	if _, err := repo.SelectBatchForProcessing(context.Background(), 1); err == nil || err.Error() != "rows err" {
		t.Fatalf("expected rows err, got %v", err)
	}
}

func TestWithdrawalRepositoryListByUserRowsError(t *testing.T) {
	storage := &Storage{pool: &rowsErrorPool{rows: &errorRows{err: errors.New("rows err")}}}
	repo := &withdrawalRepository{storage: storage}

	if _, err := repo.ListByUser(context.Background(), 1); err == nil || err.Error() != "rows err" {
		t.Fatalf("expected rows err, got %v", err)
	}
}

func TestOrderRepositoryUpdateStatus(t *testing.T) {
	storage, mock := newMockStorage(t)
	defer mock.Close()
	repo := &orderRepository{storage: storage}

	accrual := 5.0
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE orders SET status=").WithArgs(model.OrderStatusProcessed, &accrual, int64(1)).WillReturnResult(pgxmockv3.NewResult("UPDATE", 1))
	mock.ExpectQuery("SELECT user_id FROM orders WHERE id=").WithArgs(int64(1)).WillReturnRows(pgxmockv3.NewRows([]string{"user_id"}).AddRow(int64(7)))
	mock.ExpectExec("INSERT INTO balances").WithArgs(int64(7), accrual).WillReturnResult(pgxmockv3.NewResult("INSERT", 1))
	mock.ExpectCommit()
	if err := repo.UpdateStatus(context.Background(), 1, model.OrderStatusProcessed, &accrual); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE orders SET status=").WithArgs(model.OrderStatusNew, (*float64)(nil), int64(2)).WillReturnResult(pgxmockv3.NewResult("UPDATE", 1))
	mock.ExpectCommit()
	if err := repo.UpdateStatus(context.Background(), 2, model.OrderStatusNew, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	zero := 0.0
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE orders SET status=").WithArgs(model.OrderStatusProcessed, &zero, int64(3)).WillReturnResult(pgxmockv3.NewResult("UPDATE", 1))
	mock.ExpectCommit()
	if err := repo.UpdateStatus(context.Background(), 3, model.OrderStatusProcessed, &zero); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE orders SET status=").WithArgs(model.OrderStatusProcessed, &accrual, int64(4)).WillReturnError(errors.New("update"))
	mock.ExpectRollback()
	if err := repo.UpdateStatus(context.Background(), 4, model.OrderStatusProcessed, &accrual); err == nil {
		t.Fatal("expected update error")
	}

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE orders SET status=").WithArgs(model.OrderStatusProcessed, &accrual, int64(5)).WillReturnResult(pgxmockv3.NewResult("UPDATE", 1))
	mock.ExpectQuery("SELECT user_id FROM orders WHERE id=").WithArgs(int64(5)).WillReturnError(errors.New("select"))
	mock.ExpectRollback()
	if err := repo.UpdateStatus(context.Background(), 5, model.OrderStatusProcessed, &accrual); err == nil {
		t.Fatal("expected select error")
	}

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE orders SET status=").WithArgs(model.OrderStatusProcessed, &accrual, int64(6)).WillReturnResult(pgxmockv3.NewResult("UPDATE", 1))
	mock.ExpectQuery("SELECT user_id FROM orders WHERE id=").WithArgs(int64(6)).WillReturnRows(pgxmockv3.NewRows([]string{"user_id"}).AddRow(int64(8)))
	mock.ExpectExec("INSERT INTO balances").WithArgs(int64(8), accrual).WillReturnError(errors.New("accrual"))
	mock.ExpectRollback()
	if err := repo.UpdateStatus(context.Background(), 6, model.OrderStatusProcessed, &accrual); err == nil {
		t.Fatal("expected accrual error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestBalanceRepository(t *testing.T) {
	storage, mock := newMockStorage(t)
	defer mock.Close()
	repo := &balanceRepository{storage: storage}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO balances").WithArgs(int64(1), 10.0).WillReturnResult(pgxmockv3.NewResult("INSERT", 1))
	mock.ExpectCommit()
	if err := repo.AddAccrual(context.Background(), 1, 10); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO balances").WithArgs(int64(1), 10.0).WillReturnError(errors.New("insert"))
	mock.ExpectRollback()
	if err := repo.AddAccrual(context.Background(), 1, 10); err == nil {
		t.Fatal("expected error")
	}

	mock.ExpectQuery("SELECT current, withdrawn FROM balances WHERE user_id=").WithArgs(int64(1)).WillReturnRows(
		pgxmockv3.NewRows([]string{"current", "withdrawn"}).AddRow(20.0, 5.0),
	)
	summary, err := repo.GetSummary(context.Background(), 1)
	if err != nil || summary.Current != 20 || summary.Withdrawn != 5 {
		t.Fatalf("unexpected summary: %+v err=%v", summary, err)
	}

	mock.ExpectQuery("SELECT current, withdrawn FROM balances WHERE user_id=").WithArgs(int64(2)).WillReturnError(pgx.ErrNoRows)
	summary, err = repo.GetSummary(context.Background(), 2)
	if err != nil || summary.Current != 0 {
		t.Fatalf("expected zero summary, got %+v err=%v", summary, err)
	}

	mock.ExpectQuery("SELECT current, withdrawn FROM balances WHERE user_id=").WithArgs(int64(3)).WillReturnError(errors.New("query"))
	if _, err := repo.GetSummary(context.Background(), 3); err == nil {
		t.Fatal("expected error")
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT current FROM balances WHERE user_id=").WithArgs(int64(1)).WillReturnRows(pgxmockv3.NewRows([]string{"current"}).AddRow(50.0))
	mock.ExpectExec("INSERT INTO balances").WithArgs(int64(1), 30.0, 30.0).WillReturnResult(pgxmockv3.NewResult("UPDATE", 1))
	mock.ExpectExec("INSERT INTO withdrawals").WithArgs(int64(1), "ord", 30.0).WillReturnResult(pgxmockv3.NewResult("INSERT", 1))
	mock.ExpectCommit()
	if err := repo.Withdraw(context.Background(), 1, "ord", 30); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT current FROM balances WHERE user_id=").WithArgs(int64(1)).WillReturnError(errors.New("select"))
	mock.ExpectRollback()
	if err := repo.Withdraw(context.Background(), 1, "ord", 10); err == nil {
		t.Fatal("expected select error")
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT current FROM balances WHERE user_id=").WithArgs(int64(1)).WillReturnError(pgx.ErrNoRows)
	mock.ExpectRollback()
	if err := repo.Withdraw(context.Background(), 1, "ord", 10); !errors.Is(err, domainErrors.ErrInsufficientBalance) {
		t.Fatalf("expected insufficient balance, got %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT current FROM balances WHERE user_id=").WithArgs(int64(1)).WillReturnRows(pgxmockv3.NewRows([]string{"current"}).AddRow(5.0))
	mock.ExpectRollback()
	if err := repo.Withdraw(context.Background(), 1, "ord", 10); !errors.Is(err, domainErrors.ErrInsufficientBalance) {
		t.Fatalf("expected insufficient balance, got %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT current FROM balances WHERE user_id=").WithArgs(int64(1)).WillReturnRows(pgxmockv3.NewRows([]string{"current"}).AddRow(15.0))
	mock.ExpectExec("INSERT INTO balances").WithArgs(int64(1), 10.0, 10.0).WillReturnError(errors.New("update"))
	mock.ExpectRollback()
	if err := repo.Withdraw(context.Background(), 1, "ord", 10); err == nil {
		t.Fatal("expected update error")
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT current FROM balances WHERE user_id=").WithArgs(int64(1)).WillReturnRows(pgxmockv3.NewRows([]string{"current"}).AddRow(15.0))
	mock.ExpectExec("INSERT INTO balances").WithArgs(int64(1), 10.0, 10.0).WillReturnResult(pgxmockv3.NewResult("UPDATE", 1))
	mock.ExpectExec("INSERT INTO withdrawals").WithArgs(int64(1), "ord", 10.0).WillReturnError(errors.New("insert"))
	mock.ExpectRollback()
	if err := repo.Withdraw(context.Background(), 1, "ord", 10); err == nil {
		t.Fatal("expected withdrawal insert error")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestWithdrawalRepository(t *testing.T) {
	storage, mock := newMockStorage(t)
	defer mock.Close()
	repo := &withdrawalRepository{storage: storage}

	processedAt := time.Now()
	mock.ExpectQuery("SELECT id, user_id, order_number, sum, processed_at FROM withdrawals WHERE user_id=").WithArgs(int64(1)).WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "user_id", "order_number", "sum", "processed_at"}).AddRow(int64(1), int64(1), "1", 10.0, processedAt),
	)
	list, err := repo.ListByUser(context.Background(), 1)
	if err != nil || len(list) != 1 {
		t.Fatalf("unexpected result: %v err=%v", list, err)
	}

	mock.ExpectQuery("SELECT id, user_id, order_number, sum, processed_at FROM withdrawals WHERE user_id=").WithArgs(int64(2)).WillReturnError(errors.New("query"))
	if _, err := repo.ListByUser(context.Background(), 2); err == nil {
		t.Fatal("expected error")
	}

	mock.ExpectQuery("SELECT id, user_id, order_number, sum, processed_at FROM withdrawals WHERE user_id=").WithArgs(int64(3)).WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "user_id", "order_number", "sum", "processed_at"}).AddRow("bad", int64(1), "1", 10.0, processedAt),
	)
	if _, err := repo.ListByUser(context.Background(), 3); err == nil {
		t.Fatal("expected scan error")
	}

	mock.ExpectQuery("SELECT id, user_id, order_number, sum, processed_at FROM withdrawals WHERE user_id=").WithArgs(int64(4)).WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "user_id", "order_number", "sum", "processed_at"}).
			AddRow(int64(1), int64(1), "1", 10.0, processedAt).
			AddRow(int64(2), int64(1), "2", 3.0, processedAt).
			RowError(1, errors.New("row")),
	)
	if _, err := repo.ListByUser(context.Background(), 4); err == nil || err.Error() != "row" {
		t.Fatalf("expected row error, got %v", err)
	}

	mock.ExpectQuery("SELECT id, user_id, order_number, sum, processed_at FROM withdrawals WHERE user_id=").WithArgs(int64(5)).WillReturnRows(
		pgxmockv3.NewRows([]string{"id", "user_id", "order_number", "sum", "processed_at"}),
	)
	list, err = repo.ListByUser(context.Background(), 5)
	if err != nil || len(list) != 0 {
		t.Fatalf("expected empty list, got %v err=%v", list, err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestHealthCheck(t *testing.T) {
	storage, mock := newMockStorage(t)
	defer mock.Close()

	mock.ExpectPing().WillReturnError(errors.New("ping"))
	if err := storage.HealthCheck(context.Background()); err == nil {
		t.Fatal("expected error")
	}

	mock.ExpectPing()
	if err := storage.HealthCheck(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}

func TestNewStorageProvider(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	cfg := &config.Config{DatabaseURI: "postgres://user:pass@localhost/db"}
	ctx := context.Background()

	mock, err := pgxmockv3.NewPool()
	if err != nil {
		t.Fatalf("failed to create mock pool: %v", err)
	}
	defer mock.Close()

	t.Cleanup(func() {
		newPgxPool = func(ctx context.Context, cfg *pgxpool.Config) (pgxPool, error) {
			return pgxpool.NewWithConfig(ctx, cfg)
		}
	})
	newPgxPool = func(context.Context, *pgxpool.Config) (pgxPool, error) { return mock, nil }
	expectSchema(mock)

	storage, err := newStorage(storageParams{Ctx: ctx, Config: cfg, Logger: logger})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
	storage.Close()
}

func TestRegisterLifecycle(t *testing.T) {
	storage, mock := newMockStorage(t)
	defer mock.Close()

	lc := fxtest.NewLifecycle(t)
	registerLifecycle(lc, storage)

	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("start failed: %v", err)
	}

	mock.ExpectClose()
	if err := lc.Stop(context.Background()); err != nil {
		t.Fatalf("stop failed: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations not met: %v", err)
	}
}
