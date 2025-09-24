package repository

import (
	"context"

	"github.com/polkiloo/gophermart/internal/domain/model"
)

// UserRepository describes persistence operations for users.
type UserRepository interface {
	Create(ctx context.Context, login, passwordHash string) (*model.User, error)
	GetByLogin(ctx context.Context, login string) (*model.User, error)
	GetByID(ctx context.Context, id int64) (*model.User, error)
}
