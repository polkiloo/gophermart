package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestNewBcryptHasher_DefaultCost(t *testing.T) {
	hasher := NewBcryptHasher(0)
	if hasher.cost != bcrypt.DefaultCost {
		t.Fatalf("unexpected cost: %d", hasher.cost)
	}
}

func TestNewBcryptHasher_CustomCost(t *testing.T) {
	cost := bcrypt.DefaultCost + 2
	hasher := NewBcryptHasher(cost)
	if hasher.cost != cost {
		t.Fatalf("unexpected cost: %d", hasher.cost)
	}
}

func TestBcryptHasher_HashAndCompare(t *testing.T) {
	hasher := NewBcryptHasher(bcrypt.DefaultCost)
	hash, err := hasher.Hash("secret")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if err := hasher.Compare(hash, "secret"); err != nil {
		t.Fatalf("compare: %v", err)
	}
	if err := hasher.Compare(hash, "wrong"); err == nil {
		t.Fatal("expected compare error for wrong password")
	}
}

func TestBcryptHasher_HashError(t *testing.T) {
	hasher := &BcryptHasher{cost: bcrypt.MaxCost + 1}
	if _, err := hasher.Hash("password"); err == nil {
		t.Fatal("expected hash error for invalid cost")
	}
}
