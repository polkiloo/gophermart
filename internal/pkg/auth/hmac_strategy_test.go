package auth

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewHMACStrategy_DefaultTTL(t *testing.T) {
	strategy := NewHMACStrategy("secret", Options{})
	if strategy == nil {
		t.Fatal("expected strategy instance")
	}
	if string(strategy.secret) != "secret" {
		t.Fatalf("unexpected secret: %q", string(strategy.secret))
	}
	if strategy.ttl != 24*time.Hour {
		t.Fatalf("unexpected ttl: %s", strategy.ttl)
	}
}

func TestNewHMACStrategy_CustomTTL(t *testing.T) {
	ttl := 2 * time.Hour
	strategy := NewHMACStrategy("secret", Options{TTL: ttl})
	if strategy.ttl != ttl {
		t.Fatalf("unexpected ttl: %s", strategy.ttl)
	}
}

func TestHMACStrategy_IssueAndParse(t *testing.T) {
	strategy := NewHMACStrategy("secret", Options{TTL: time.Minute})
	token, err := strategy.IssueToken(42)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	userID, err := strategy.ParseToken(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if userID != 42 {
		t.Fatalf("unexpected user id: %d", userID)
	}
}

func TestHMACStrategy_ParseInvalidBase64(t *testing.T) {
	strategy := NewHMACStrategy("secret", Options{})
	if _, err := strategy.ParseToken("not-base64"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestHMACStrategy_ParseInvalidParts(t *testing.T) {
	strategy := NewHMACStrategy("secret", Options{})
	token := base64.StdEncoding.EncodeToString([]byte("only:two"))
	if _, err := strategy.ParseToken(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestHMACStrategy_ParseInvalidSignature(t *testing.T) {
	strategy := NewHMACStrategy("secret", Options{TTL: time.Minute})
	token, err := strategy.IssueToken(7)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	raw, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("decode token: %v", err)
	}
	parts := strings.Split(string(raw), ":")
	if len(parts) != 3 {
		t.Fatalf("unexpected parts count: %d", len(parts))
	}
	parts[2] = "tampered"
	tamperedToken := base64.StdEncoding.EncodeToString([]byte(strings.Join(parts, ":")))
	if _, err := strategy.ParseToken(tamperedToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestHMACStrategy_ParseInvalidUserID(t *testing.T) {
	strategy := NewHMACStrategy("secret", Options{})
	payload := fmt.Sprintf("abc:%d", time.Now().Add(time.Minute).Unix())
	sig := strategy.sign(payload)
	token := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", payload, sig)))
	if _, err := strategy.ParseToken(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestHMACStrategy_ParseInvalidExpiry(t *testing.T) {
	strategy := NewHMACStrategy("secret", Options{})
	payload := "10:not-a-number"
	sig := strategy.sign(payload)
	token := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", payload, sig)))
	if _, err := strategy.ParseToken(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestHMACStrategy_ParseExpired(t *testing.T) {
	strategy := NewHMACStrategy("secret", Options{})
	payload := fmt.Sprintf("10:%d", time.Now().Add(-time.Minute).Unix())
	sig := strategy.sign(payload)
	token := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", payload, sig)))
	if _, err := strategy.ParseToken(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestHMACStrategy_Name(t *testing.T) {
	strategy := NewHMACStrategy("secret", Options{})
	if strategy.Name() != "hmac" {
		t.Fatalf("unexpected name: %s", strategy.Name())
	}
}
