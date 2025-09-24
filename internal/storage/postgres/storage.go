package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	domainErrors "github.com/polkiloo/gophermart/internal/domain/errors"
	"github.com/polkiloo/gophermart/internal/domain/model"
	"github.com/polkiloo/gophermart/internal/domain/repository"
)

// Storage acts as repository facade backed by PostgreSQL.
type Storage struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

type userRepository struct {
	storage *Storage
}

type orderRepository struct {
	storage *Storage
}

type balanceRepository struct {
	storage *Storage
}

type withdrawalRepository struct {
	storage *Storage
}

// New creates storage with schema initialization.
func New(ctx context.Context, dsn string, logger *slog.Logger) (*Storage, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	storage := &Storage{pool: pool, logger: logger}
	if err := storage.initSchema(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return storage, nil
}

// Close releases database resources.
func (s *Storage) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// Factory methods for domain repositories.
func (s *Storage) Users() repository.UserRepository {
	return &userRepository{storage: s}
}

func (s *Storage) Orders() repository.OrderRepository {
	return &orderRepository{storage: s}
}

func (s *Storage) Balances() repository.BalanceRepository {
	return &balanceRepository{storage: s}
}

func (s *Storage) Withdrawals() repository.WithdrawalRepository {
	return &withdrawalRepository{storage: s}
}

func (s *Storage) initSchema(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
            id SERIAL PRIMARY KEY,
            login TEXT UNIQUE NOT NULL,
            password_hash TEXT NOT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )`,
		`CREATE TABLE IF NOT EXISTS orders (
            id SERIAL PRIMARY KEY,
            user_id BIGINT NOT NULL REFERENCES users(id),
            number TEXT UNIQUE NOT NULL,
            status TEXT NOT NULL,
            accrual DOUBLE PRECISION,
            uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )`,
		`CREATE TABLE IF NOT EXISTS balances (
            user_id BIGINT PRIMARY KEY REFERENCES users(id),
            current DOUBLE PRECISION NOT NULL DEFAULT 0,
            withdrawn DOUBLE PRECISION NOT NULL DEFAULT 0
        )`,
		`CREATE TABLE IF NOT EXISTS withdrawals (
            id SERIAL PRIMARY KEY,
            user_id BIGINT NOT NULL REFERENCES users(id),
            order_number TEXT NOT NULL,
            sum DOUBLE PRECISION NOT NULL,
            processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )`,
		`CREATE INDEX IF NOT EXISTS idx_orders_user ON orders(user_id, uploaded_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_withdrawals_user ON withdrawals(user_id, processed_at DESC)`,
	}

	for _, stmt := range statements {
		if _, err := s.pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}

	return nil
}

// --- UserRepository implementation ---

func (r *userRepository) Create(ctx context.Context, login, passwordHash string) (*model.User, error) {
	const query = `INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id, created_at`
	var u model.User
	err := r.storage.pool.QueryRow(ctx, query, login, passwordHash).Scan(&u.ID, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domainErrors.ErrAlreadyExists
		}
		return nil, err
	}
	u.Login = login
	u.PasswordHash = passwordHash
	return &u, nil
}

func (r *userRepository) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	const query = `SELECT id, login, password_hash, created_at FROM users WHERE login=$1`
	var u model.User
	err := r.storage.pool.QueryRow(ctx, query, login).Scan(&u.ID, &u.Login, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErrors.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	const query = `SELECT id, login, password_hash, created_at FROM users WHERE id=$1`
	var u model.User
	err := r.storage.pool.QueryRow(ctx, query, id).Scan(&u.ID, &u.Login, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErrors.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// --- OrderRepository implementation ---

func (r *orderRepository) Create(ctx context.Context, userID int64, number string) (*model.Order, bool, error) {
	const query = `INSERT INTO orders (user_id, number, status) VALUES ($1, $2, $3)
                   ON CONFLICT (number) DO NOTHING
                   RETURNING id, status, uploaded_at, updated_at`
	var order model.Order
	err := r.storage.pool.QueryRow(ctx, query, userID, number, model.OrderStatusNew).Scan(&order.ID, &order.Status, &order.UploadedAt, &order.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			existing, err := r.GetByNumber(ctx, number)
			if err != nil {
				return nil, false, err
			}
			if existing.UserID != userID {
				return existing, false, domainErrors.ErrAlreadyExists
			}
			return existing, false, nil
		}
		return nil, false, err
	}
	order.UserID = userID
	order.Number = number
	return &order, true, nil
}

func (r *orderRepository) GetByNumber(ctx context.Context, number string) (*model.Order, error) {
	const query = `SELECT id, user_id, number, status, accrual, uploaded_at, updated_at FROM orders WHERE number=$1`
	var order model.Order
	err := r.storage.pool.QueryRow(ctx, query, number).Scan(&order.ID, &order.UserID, &order.Number, &order.Status, &order.Accrual, &order.UploadedAt, &order.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainErrors.ErrNotFound
		}
		return nil, err
	}
	return &order, nil
}

func (r *orderRepository) ListByUser(ctx context.Context, userID int64) ([]model.Order, error) {
	const query = `SELECT id, user_id, number, status, accrual, uploaded_at, updated_at
                   FROM orders WHERE user_id=$1 ORDER BY uploaded_at DESC`
	rows, err := r.storage.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.Order
	for rows.Next() {
		var o model.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.Number, &o.Status, &o.Accrual, &o.UploadedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *orderRepository) SelectBatchForProcessing(ctx context.Context, limit int) ([]model.Order, error) {
	const selectQuery = `SELECT id, user_id, number, status, accrual, uploaded_at, updated_at
                         FROM orders
                         WHERE status IN ('NEW', 'PROCESSING')
                         ORDER BY uploaded_at
                         LIMIT $1
                         FOR UPDATE SKIP LOCKED`

	var orders []model.Order
	err := r.storage.WithinTransaction(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, selectQuery, limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var o model.Order
			if err := rows.Scan(&o.ID, &o.UserID, &o.Number, &o.Status, &o.Accrual, &o.UploadedAt, &o.UpdatedAt); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `UPDATE orders SET status='PROCESSING', updated_at=NOW() WHERE id=$1`, o.ID); err != nil {
				return err
			}
			o.Status = model.OrderStatusProcessing
			orders = append(orders, o)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *orderRepository) UpdateStatus(ctx context.Context, orderID int64, status model.OrderStatus, accrual *float64) error {
	return r.storage.WithinTransaction(ctx, func(tx pgx.Tx) error {
		const updateQuery = `UPDATE orders SET status=$1, accrual=$2, updated_at=NOW() WHERE id=$3`
		if _, err := tx.Exec(ctx, updateQuery, status, accrual, orderID); err != nil {
			return err
		}

		if status == model.OrderStatusProcessed && accrual != nil && *accrual > 0 {
			const selectUser = `SELECT user_id FROM orders WHERE id=$1`
			var userID int64
			if err := tx.QueryRow(ctx, selectUser, orderID).Scan(&userID); err != nil {
				return err
			}
			if err := r.storage.addAccrualTx(ctx, tx, userID, *accrual); err != nil {
				return err
			}
		}

		return nil
	})
}

// --- BalanceRepository implementation ---

func (s *Storage) addAccrualTx(ctx context.Context, tx pgx.Tx, userID int64, sum float64) error {
	const updateBalance = `INSERT INTO balances (user_id, current, withdrawn)
                           VALUES ($1, $2, 0)
                           ON CONFLICT (user_id) DO UPDATE SET current = balances.current + EXCLUDED.current`
	if _, err := tx.Exec(ctx, updateBalance, userID, sum); err != nil {
		return err
	}
	return nil
}

func (r *balanceRepository) AddAccrual(ctx context.Context, userID int64, sum float64) error {
	return r.storage.WithinTransaction(ctx, func(tx pgx.Tx) error {
		return r.storage.addAccrualTx(ctx, tx, userID, sum)
	})
}

func (r *balanceRepository) GetSummary(ctx context.Context, userID int64) (*model.BalanceSummary, error) {
	const query = `SELECT current, withdrawn FROM balances WHERE user_id=$1`
	var summary model.BalanceSummary
	err := r.storage.pool.QueryRow(ctx, query, userID).Scan(&summary.Current, &summary.Withdrawn)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return &model.BalanceSummary{}, nil
		}
		return nil, err
	}
	return &summary, nil
}

func (r *balanceRepository) Withdraw(ctx context.Context, userID int64, order string, sum float64) error {
	return r.storage.WithinTransaction(ctx, func(tx pgx.Tx) error {
		const balanceQuery = `SELECT current FROM balances WHERE user_id=$1 FOR UPDATE`
		var current float64
		err := tx.QueryRow(ctx, balanceQuery, userID).Scan(&current)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				current = 0
			} else {
				return err
			}
		}
		if current < sum {
			return domainErrors.ErrInsufficientBalance
		}

		const updateBalance = `INSERT INTO balances (user_id, current, withdrawn)
                               VALUES ($1, $2, $3)
                               ON CONFLICT (user_id) DO UPDATE
                               SET current = balances.current - $2,
                                   withdrawn = balances.withdrawn + $3`
		if _, err := tx.Exec(ctx, updateBalance, userID, sum, sum); err != nil {
			return err
		}

		const insertWithdrawal = `INSERT INTO withdrawals (user_id, order_number, sum) VALUES ($1, $2, $3)`
		if _, err := tx.Exec(ctx, insertWithdrawal, userID, order, sum); err != nil {
			return err
		}
		return nil
	})
}

// --- WithdrawalRepository implementation ---

func (r *withdrawalRepository) ListByUser(ctx context.Context, userID int64) ([]model.Withdrawal, error) {
	const query = `SELECT id, user_id, order_number, sum, processed_at
                   FROM withdrawals WHERE user_id=$1 ORDER BY processed_at DESC`
	rows, err := r.storage.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.Withdrawal
	for rows.Next() {
		var w model.Withdrawal
		if err := rows.Scan(&w.ID, &w.UserID, &w.OrderNumber, &w.Sum, &w.ProcessedAt); err != nil {
			return nil, err
		}
		result = append(result, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

// WithinTransaction executes function inside transaction boundary.
func (s *Storage) WithinTransaction(ctx context.Context, fn func(pgx.Tx) error) (err error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	err = fn(tx)
	return err
}

// HealthCheck verifies database connectivity.
func (s *Storage) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return s.pool.Ping(ctx)
}

// Pool exposes raw connection pool for advanced use.
func (s *Storage) Pool() *pgxpool.Pool {
	return s.pool
}

// Logger returns storage logger.
func (s *Storage) Logger() *slog.Logger {
	return s.logger
}
