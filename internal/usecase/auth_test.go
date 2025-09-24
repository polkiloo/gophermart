package usecase

import (
	"context"
	"fmt"
	"testing"

	domainErrors "github.com/polkiloo/gophermart/internal/domain/errors"
	pkgAuth "github.com/polkiloo/gophermart/internal/pkg/auth"
	testhelpers "github.com/polkiloo/gophermart/internal/test"
)

func newStrategyStub() testhelpers.StrategyStub {
	return testhelpers.StrategyStub{
		IssueFn: func(userID int64) (string, error) {
			return fmt.Sprintf("token-%d", userID), nil
		},
		ParseFn: func(token string) (int64, error) {
			var id int64
			if _, err := fmt.Sscanf(token, "token-%d", &id); err != nil {
				return 0, pkgAuth.ErrInvalidToken
			}
			return id, nil
		},
	}
}
func TestAuthUseCaseRegisterSuccess(t *testing.T) {
	repo := testhelpers.NewUserRepositoryStub()
	uc := NewAuthUseCase(repo, testhelpers.HasherStub{}, newStrategyStub())

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
	stored, err := repo.GetByLogin(ctx, "alice")
	if err != nil {
		t.Fatalf("expected user in repository: %v", err)
	}
	if stored.PasswordHash != "hash:password" {
		t.Fatalf("password hash not stored: %v", stored.PasswordHash)
	}
}

func TestAuthUseCaseRegisterDuplicate(t *testing.T) {
	repo := testhelpers.NewUserRepositoryStub()
	uc := NewAuthUseCase(repo, testhelpers.HasherStub{}, newStrategyStub())

	ctx := context.Background()
	if _, _, err := uc.Register(ctx, "bob", "secret"); err != nil {
		t.Fatalf("unexpected error on first register: %v", err)
	}
	if _, _, err := uc.Register(ctx, "bob", "secret"); err != domainErrors.ErrAlreadyExists {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestAuthUseCaseAuthenticate(t *testing.T) {
	repo := testhelpers.NewUserRepositoryStub()
	uc := NewAuthUseCase(repo, testhelpers.HasherStub{}, newStrategyStub())

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
	uc := NewAuthUseCase(testhelpers.NewUserRepositoryStub(), testhelpers.HasherStub{}, newStrategyStub())

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

	if _, err := uc.ParseToken(""); err != pkgAuth.ErrInvalidToken {
		t.Fatalf("expected invalid token error, got %v", err)
	}
}

func TestAuthUseCaseRegisterValidation(t *testing.T) {
	uc := NewAuthUseCase(testhelpers.NewUserRepositoryStub(), testhelpers.HasherStub{}, newStrategyStub())
	if _, _, err := uc.Register(context.Background(), "", "password"); err != domainErrors.ErrInvalidCredentials {
		t.Fatalf("expected invalid credentials error, got %v", err)
	}
	if _, _, err := uc.Register(context.Background(), "user", ""); err != domainErrors.ErrInvalidCredentials {
		t.Fatalf("expected invalid credentials error, got %v", err)
	}
}

func TestAuthUseCaseRegisterHasherError(t *testing.T) {
	repo := testhelpers.NewUserRepositoryStub()
	uc := NewAuthUseCase(repo, testhelpers.HasherStub{HashFn: func(string) (string, error) {
		return "", fmt.Errorf("hash error")
	}}, newStrategyStub())
	if _, _, err := uc.Register(context.Background(), "user", "pass"); err == nil {
		t.Fatal("expected hashing error")
	}
}

func TestAuthUseCaseRegisterRepositoryError(t *testing.T) {
	repo := testhelpers.NewUserRepositoryStub()
	repo.Err = fmt.Errorf("db down")
	uc := NewAuthUseCase(repo, testhelpers.HasherStub{}, newStrategyStub())
	if _, _, err := uc.Register(context.Background(), "user", "pass"); err == nil {
		t.Fatal("expected repository error")
	}
}

func TestAuthUseCaseRegisterIssueTokenError(t *testing.T) {
	repo := testhelpers.NewUserRepositoryStub()
	strategy := testhelpers.StrategyStub{IssueFn: func(int64) (string, error) {
		return "", fmt.Errorf("cannot issue token")
	}}
	uc := NewAuthUseCase(repo, testhelpers.HasherStub{}, strategy)
	if _, _, err := uc.Register(context.Background(), "user", "pass"); err == nil {
		t.Fatal("expected token issuing error")
	}
}

func TestAuthUseCaseAuthenticateNotFound(t *testing.T) {
	repo := testhelpers.NewUserRepositoryStub()
	uc := NewAuthUseCase(repo, testhelpers.HasherStub{}, newStrategyStub())
	if _, _, err := uc.Authenticate(context.Background(), "absent", "pass"); err != domainErrors.ErrInvalidCredentials {
		t.Fatalf("expected invalid credentials error, got %v", err)
	}
}

func TestAuthUseCaseAuthenticateHasherMismatch(t *testing.T) {
	repo := testhelpers.NewUserRepositoryStub()
	uc := NewAuthUseCase(repo, testhelpers.HasherStub{CompareFn: func(hash, password string) error {
		return fmt.Errorf("mismatch")
	}}, newStrategyStub())
	if _, _, err := uc.Register(context.Background(), "user", "pass"); err != nil {
		t.Fatalf("register returned error: %v", err)
	}
	if _, _, err := uc.Authenticate(context.Background(), "user", "pass"); err != domainErrors.ErrInvalidCredentials {
		t.Fatalf("expected invalid credentials, got %v", err)
	}
}

func TestAuthUseCaseAuthenticateIssueTokenError(t *testing.T) {
	repo := testhelpers.NewUserRepositoryStub()
	calls := 0
	strategy := testhelpers.StrategyStub{
		IssueFn: func(int64) (string, error) {
			calls++
			if calls > 1 {
				return "", fmt.Errorf("issue error")
			}
			return "token", nil
		},
	}
	uc := NewAuthUseCase(repo, testhelpers.HasherStub{}, strategy)
	if _, _, err := uc.Register(context.Background(), "user", "pass"); err != nil {
		t.Fatalf("register returned error: %v", err)
	}
	if _, _, err := uc.Authenticate(context.Background(), "user", "pass"); err == nil {
		t.Fatal("expected issue error on authenticate")
	}
}

func TestAuthUseCaseAuthenticateRepositoryError(t *testing.T) {
	repo := testhelpers.NewUserRepositoryStub()
	uc := NewAuthUseCase(repo, testhelpers.HasherStub{}, newStrategyStub())
	if _, _, err := uc.Register(context.Background(), "user", "pass"); err != nil {
		t.Fatalf("register returned error: %v", err)
	}
	repo.Err = fmt.Errorf("storage unavailable")
	if _, _, err := uc.Authenticate(context.Background(), "user", "pass"); err == nil || err.Error() != "storage unavailable" {
		t.Fatalf("expected repository error, got %v", err)
	}
}

func TestAuthUseCaseAuthenticateValidation(t *testing.T) {
	uc := NewAuthUseCase(testhelpers.NewUserRepositoryStub(), testhelpers.HasherStub{}, newStrategyStub())
	if _, _, err := uc.Authenticate(context.Background(), "", "pass"); err != domainErrors.ErrInvalidCredentials {
		t.Fatalf("expected invalid credentials error, got %v", err)
	}
	if _, _, err := uc.Authenticate(context.Background(), "user", ""); err != domainErrors.ErrInvalidCredentials {
		t.Fatalf("expected invalid credentials error, got %v", err)
	}
}

func TestAuthUseCaseParseTokenStrategyError(t *testing.T) {
	uc := NewAuthUseCase(testhelpers.NewUserRepositoryStub(), testhelpers.HasherStub{}, testhelpers.StrategyStub{
		ParseFn: func(string) (int64, error) { return 0, fmt.Errorf("parse error") },
	})
	if _, err := uc.ParseToken("token"); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestAuthUseCaseGetByID(t *testing.T) {
	repo := testhelpers.NewUserRepositoryStub()
	uc := NewAuthUseCase(repo, testhelpers.HasherStub{}, newStrategyStub())
	user, _, err := uc.Register(context.Background(), "dave", "pwd")
	if err != nil {
		t.Fatalf("register returned error: %v", err)
	}
	fetched, err := uc.GetByID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("get by id returned error: %v", err)
	}
	if fetched.Login != user.Login {
		t.Fatalf("expected login %q, got %q", user.Login, fetched.Login)
	}
}

func TestAuthUseCaseGetByIDErrorPropagation(t *testing.T) {
	repo := testhelpers.NewUserRepositoryStub()
	repo.Err = fmt.Errorf("read error")
	uc := NewAuthUseCase(repo, testhelpers.HasherStub{}, newStrategyStub())
	if _, err := uc.GetByID(context.Background(), 1); err == nil {
		t.Fatal("expected repository error")
	}
}

func TestAuthUseCaseTrimsLogin(t *testing.T) {
	repo := testhelpers.NewUserRepositoryStub()
	uc := NewAuthUseCase(repo, testhelpers.HasherStub{}, newStrategyStub())
	if _, _, err := uc.Register(context.Background(), "  user  ", "pass"); err != nil {
		t.Fatalf("register returned error: %v", err)
	}
	if _, _, err := uc.Authenticate(context.Background(), "  user  ", "pass"); err != nil {
		t.Fatalf("authenticate returned error: %v", err)
	}
}

func TestUserRepositoryStubDuplicate(t *testing.T) {
	repo := testhelpers.NewUserRepositoryStub()
	if _, err := repo.Create(context.Background(), "user", "hash"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := repo.Create(context.Background(), "user", "hash"); err != domainErrors.ErrAlreadyExists {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}
