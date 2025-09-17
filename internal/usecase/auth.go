package usecase

import (
	"context"
	"errors"
	"strings"

	domainErrors "github.com/polkiloo/gophermart/internal/domain/errors"
	"github.com/polkiloo/gophermart/internal/domain/model"
	"github.com/polkiloo/gophermart/internal/domain/repository"
	pkgAuth "github.com/polkiloo/gophermart/internal/pkg/auth"
)

// AuthUseCase handles user lifecycle and token management.
type AuthUseCase struct {
	users  repository.UserRepository
	hasher pkgAuth.PasswordHasher
	tokens pkgAuth.Strategy
}

// NewAuthUseCase constructs AuthUseCase.
func NewAuthUseCase(users repository.UserRepository, hasher pkgAuth.PasswordHasher, strategy pkgAuth.Strategy) *AuthUseCase {
	return &AuthUseCase{users: users, hasher: hasher, tokens: strategy}
}

// Register creates a new user with login/password and returns auth token.
func (u *AuthUseCase) Register(ctx context.Context, login, password string) (*model.User, string, error) {
	login = strings.TrimSpace(login)
	if login == "" || password == "" {
		return nil, "", domainErrors.ErrInvalidCredentials
	}

	hash, err := u.hasher.Hash(password)
	if err != nil {
		return nil, "", err
	}

	usr, err := u.users.Create(ctx, login, hash)
	if err != nil {
		if errors.Is(err, domainErrors.ErrAlreadyExists) {
			return nil, "", domainErrors.ErrAlreadyExists
		}
		return nil, "", err
	}

	token, err := u.tokens.IssueToken(usr.ID)
	if err != nil {
		return nil, "", err
	}

	return usr, token, nil
}

// Authenticate validates credentials and returns auth token.
func (u *AuthUseCase) Authenticate(ctx context.Context, login, password string) (*model.User, string, error) {
	login = strings.TrimSpace(login)
	if login == "" || password == "" {
		return nil, "", domainErrors.ErrInvalidCredentials
	}

	usr, err := u.users.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, domainErrors.ErrNotFound) {
			return nil, "", domainErrors.ErrInvalidCredentials
		}
		return nil, "", err
	}

	if err := u.hasher.Compare(usr.PasswordHash, password); err != nil {
		return nil, "", domainErrors.ErrInvalidCredentials
	}

	token, err := u.tokens.IssueToken(usr.ID)
	if err != nil {
		return nil, "", err
	}

	return usr, token, nil
}

// ParseToken extracts user ID from provided token.
func (u *AuthUseCase) ParseToken(token string) (int64, error) {
	if token == "" {
		return 0, pkgAuth.ErrInvalidToken
	}
	return u.tokens.ParseToken(token)
}

// GetByID fetches user by identifier.
func (u *AuthUseCase) GetByID(ctx context.Context, id int64) (*model.User, error) {
	return u.users.GetByID(ctx, id)
}
