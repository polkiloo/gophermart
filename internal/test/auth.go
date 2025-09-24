package test

import (
	"context"
	"errors"

	pkgAuth "github.com/polkiloo/gophermart/internal/pkg/auth"
)

// HasherStub provides deterministic hashing for tests.
type HasherStub struct {
	HashFn    func(string) (string, error)
	CompareFn func(string, string) error
}

// Hash returns a predictable hash for the supplied password.
func (h HasherStub) Hash(password string) (string, error) {
	if h.HashFn != nil {
		return h.HashFn(password)
	}
	return "hash:" + password, nil
}

// Compare validates password against stored hash.
func (h HasherStub) Compare(hash string, password string) error {
	if h.CompareFn != nil {
		return h.CompareFn(hash, password)
	}
	if hash != "hash:"+password {
		return errors.New("mismatch")
	}
	return nil
}

// StrategyStub issues and parses tokens via function overrides.
type StrategyStub struct {
	IssueFn func(int64) (string, error)
	ParseFn func(string) (int64, error)
	NameVal string
}

// IssueToken returns deterministic tokens for tests.
func (s StrategyStub) IssueToken(userID int64) (string, error) {
	if s.IssueFn != nil {
		return s.IssueFn(userID)
	}
	return "token", nil
}

// ParseToken parses previously issued token strings.
func (s StrategyStub) ParseToken(token string) (int64, error) {
	if s.ParseFn != nil {
		return s.ParseFn(token)
	}
	return 1, nil
}

// Name returns the strategy identifier used in tests.
func (s StrategyStub) Name() string {
	if s.NameVal != "" {
		return s.NameVal
	}
	return "stub"
}

// TokenParserStub implements middleware token parsing contract.
type TokenParserStub struct {
	ID      int64
	Err     error
	ParseFn func(string) (int64, error)
}

// ParseToken either delegates to override or returns predefined result.
func (s TokenParserStub) ParseToken(token string) (int64, error) {
	if s.ParseFn != nil {
		return s.ParseFn(token)
	}
	if s.Err != nil {
		return 0, s.Err
	}
	return s.ID, nil
}

// AuthFacadeStub simulates authentication facade interactions.
type AuthFacadeStub struct {
	RegisterFn     func(context.Context, string, string) (string, error)
	AuthenticateFn func(context.Context, string, string) (string, error)
	ParseFn        func(string) (int64, error)
}

// Register returns token for successful registration scenarios.
func (s AuthFacadeStub) Register(ctx context.Context, login, password string) (string, error) {
	if s.RegisterFn != nil {
		return s.RegisterFn(ctx, login, password)
	}
	return "token", nil
}

// Authenticate returns token for successful authentication scenarios.
func (s AuthFacadeStub) Authenticate(ctx context.Context, login, password string) (string, error) {
	if s.AuthenticateFn != nil {
		return s.AuthenticateFn(ctx, login, password)
	}
	return "token", nil
}

// ParseToken returns stored identifier for authenticated user.
func (s AuthFacadeStub) ParseToken(token string) (int64, error) {
	if s.ParseFn != nil {
		return s.ParseFn(token)
	}
	return 1, nil
}

// LoyaltyFacadeStub aggregates facade dependencies for HTTP layer tests.
type LoyaltyFacadeStub struct {
	AuthFacadeStub
	OrderFacadeStub
	BalanceFacadeStub
}

var _ pkgAuth.PasswordHasher = HasherStub{}
var _ pkgAuth.Strategy = StrategyStub{}
