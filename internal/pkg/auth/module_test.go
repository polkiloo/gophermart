package auth

import (
	"testing"
	"time"

	"github.com/polkiloo/gophermart/internal/config"
	"golang.org/x/crypto/bcrypt"
)

func TestNewPasswordHasher(t *testing.T) {
	hasher := newPasswordHasher()
	bcryptHasher, ok := hasher.(*BcryptHasher)
	if !ok {
		t.Fatalf("expected *BcryptHasher, got %T", hasher)
	}
	if bcryptHasher.cost != bcrypt.DefaultCost {
		t.Fatalf("unexpected cost: %d", bcryptHasher.cost)
	}
}

func TestNewTokenStrategy(t *testing.T) {
	strategy := newTokenStrategy(strategyParams{Config: &config.Config{JWTSecret: "top-secret"}})
	hmacStrategy, ok := strategy.(*HMACStrategy)
	if !ok {
		t.Fatalf("expected *HMACStrategy, got %T", strategy)
	}
	if string(hmacStrategy.secret) != "top-secret" {
		t.Fatalf("unexpected secret: %q", string(hmacStrategy.secret))
	}
	if hmacStrategy.ttl != 24*time.Hour {
		t.Fatalf("unexpected ttl: %s", hmacStrategy.ttl)
	}
}
