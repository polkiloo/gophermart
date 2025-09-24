package accrual

import (
	"io"
	"log/slog"
	"testing"

	"github.com/polkiloo/gophermart/internal/config"
)

func TestNewClientUsesConfig(t *testing.T) {
	cfg := &config.Config{AccrualSystemAddress: "http://example.com"}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	client, err := newClient(clientParams{Config: cfg, Logger: logger})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("expected client instance")
	}
}
