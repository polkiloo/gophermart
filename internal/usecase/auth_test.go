package usecase

import (
	"context"
	"fmt"
	"testing"

	domainErrors "github.com/polkiloo/gophermart/internal/domain/errors"
	"github.com/polkiloo/gophermart/internal/domain/model"
	pkgAuth "github.com/polkiloo/gophermart/internal/pkg/auth"
)

type fakeHasher struct{}

func (fakeHasher) Hash(password string) (string, error) {
	return "hash:" + password, nil
}

func (fakeHasher) Compare(hash string, password string) error {
	if hash != "hash:"+password {
		return fmt.Errorf("mismatch")
	}
	return nil
}

type fakeStrategy struct{}

func (fakeStrategy) IssueToken(userID int64) (string, error) {
	return fmt.Sprintf("token-%d", userID), nil
}

func (fakeStrategy) ParseToken(token string) (int64, error) {
	var id int64
	if _, err := fmt.Sscanf(token, "token-%d", &id); err != nil {
		return 0, pkgAuth.ErrInvalidToken
	}
	return id, nil
}

func (fakeStrategy) Name() string { return "fake" }

type fakeUserRepository struct {
	byLogin map[string]*model.User
	byID    map[int64]*model.User
	nextID  int64
}

func newFakeUserRepository() *fakeUserRepository {
	return &fakeUserRepository{byLogin: make(map[string]*model.User), byID: make(map[int64]*model.User)}
}

func (r *fakeUserRepository) Create(_ context.Context, login, passwordHash string) (*model.User, error) {
	if _, exists := r.byLogin[login]; exists {
		return nil, domainErrors.ErrAlreadyExists
	}
	r.nextID++
	user := &model.User{ID: r.nextID, Login: login, PasswordHash: passwordHash}
	r.byLogin[login] = user
	r.byID[user.ID] = user
	return user, nil
}

func (r *fakeUserRepository) GetByLogin(_ context.Context, login string) (*model.User, error) {
	user, ok := r.byLogin[login]
	if !ok {
		return nil, domainErrors.ErrNotFound
	}
	return user, nil
}

func (r *fakeUserRepository) GetByID(_ context.Context, id int64) (*model.User, error) {
	user, ok := r.byID[id]
	if !ok {
		return nil, domainErrors.ErrNotFound
	}
	return user, nil
}

func TestAuthUseCaseRegisterSuccess(t *testing.T) {
	repo := newFakeUserRepository()
	uc := NewAuthUseCase(repo, fakeHasher{}, fakeStrategy{})

	ctx := context.Background()
	user, token, err := uc.Register(ctx, "alice", "password")
	if err != nil {
		t.Fatalf("register returned error: %v", err)
	}
	if user.ID == 0 {
		t.Fatalf("expected user to have ID assigned")
	}
	if token != "token-1" {
		t.Fatalf("unexpected token %q", token)
	}
	stored, _ := repo.GetByLogin(ctx, "alice")
	if stored.PasswordHash != "hash:password" {
		t.Fatalf("password hash not stored: %v", stored.PasswordHash)
	}
}

func TestAuthUseCaseRegisterDuplicate(t *testing.T) {
	repo := newFakeUserRepository()
	uc := NewAuthUseCase(repo, fakeHasher{}, fakeStrategy{})

	ctx := context.Background()
	if _, _, err := uc.Register(ctx, "bob", "secret"); err != nil {
		t.Fatalf("unexpected error on first register: %v", err)
	}
	if _, _, err := uc.Register(ctx, "bob", "secret"); err != domainErrors.ErrAlreadyExists {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestAuthUseCaseAuthenticate(t *testing.T) {
	repo := newFakeUserRepository()
	uc := NewAuthUseCase(repo, fakeHasher{}, fakeStrategy{})

	ctx := context.Background()
	if _, _, err := uc.Register(ctx, "carol", "123456"); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	if _, _, err := uc.Authenticate(ctx, "carol", "bad"); err != domainErrors.ErrInvalidCredentials {
		t.Fatalf("expected invalid credentials error, got %v", err)
	}

	_, token, err := uc.Authenticate(ctx, "carol", "123456")
	if err != nil {
		t.Fatalf("authenticate returned error: %v", err)
	}
	if token != "token-1" {
		t.Fatalf("unexpected token %q", token)
	}
}

func TestAuthUseCaseParseToken(t *testing.T) {
	uc := NewAuthUseCase(newFakeUserRepository(), fakeHasher{}, fakeStrategy{})

	id, err := uc.ParseToken("token-42")
	if err != nil {
		t.Fatalf("parse token failed: %v", err)
	}
	if id != 42 {
		t.Fatalf("expected id 42, got %d", id)
	}

	if _, err := uc.ParseToken("bad-token"); err != pkgAuth.ErrInvalidToken {
		t.Fatalf("expected invalid token error, got %v", err)
	}
}
