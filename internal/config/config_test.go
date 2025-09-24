package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadDefaultsAndOverrides(t *testing.T) {
	_, err := load(nil, func(string) (string, bool) { return "", false })
	if err == nil {
		t.Fatalf("expected error due to missing required envs, got nil")
	}

	env := map[string]string{
		"DATABASE_URI":           "postgres://user:pass@localhost/db",
		"ACCRUAL_SYSTEM_ADDRESS": "http://accrual.local",
	}

	cfg, err := load(nil, func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	})
	if err != nil {
		t.Fatalf("load returned unexpected error: %v", err)
	}

	if cfg.RunAddress != defaultRunAddress {
		t.Errorf("expected default run address %q, got %q", defaultRunAddress, cfg.RunAddress)
	}
	if cfg.JWTSecret != defaultJWTSecret {
		t.Errorf("expected default jwt secret %q, got %q", defaultJWTSecret, cfg.JWTSecret)
	}
	if cfg.OrderPollInterval != defaultOrderPollInterval {
		t.Errorf("expected default poll interval %v, got %v", defaultOrderPollInterval, cfg.OrderPollInterval)
	}
	if cfg.WorkerPoolSize != defaultWorkerPoolSize {
		t.Errorf("expected default worker pool %d, got %d", defaultWorkerPoolSize, cfg.WorkerPoolSize)
	}
	if cfg.MaxOrdersBatch != defaultMaxOrdersBatch {
		t.Errorf("expected default batch size %d, got %d", defaultMaxOrdersBatch, cfg.MaxOrdersBatch)
	}
}

func TestLoadWithFlagOverrides(t *testing.T) {
	env := map[string]string{
		"DATABASE_URI":           "postgres://user:pass@localhost/db",
		"ACCRUAL_SYSTEM_ADDRESS": "http://accrual.local",
		"WORKER_POOL_SIZE":       "3",
		"POLL_BATCH_SIZE":        "10",
		"ORDER_POLL_INTERVAL":    "5s",
	}

	args := []string{
		"-a", ":9090",
		"-d", "postgres://override",
		"-r", "http://override",
		"--poll-interval", "7s",
		"--shutdown-timeout", "20s",
		"--worker-pool", "9",
		"--poll-batch", "11",
		"--jwt-secret", "flag-secret",
	}

	cfg, err := load(args, func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	})
	if err != nil {
		t.Fatalf("load returned unexpected error: %v", err)
	}

	if cfg.RunAddress != ":9090" {
		t.Errorf("expected run address :9090, got %q", cfg.RunAddress)
	}
	if cfg.DatabaseURI != "postgres://override" {
		t.Errorf("expected database uri override, got %q", cfg.DatabaseURI)
	}
	if cfg.AccrualSystemAddress != "http://override" {
		t.Errorf("expected accrual override, got %q", cfg.AccrualSystemAddress)
	}
	if cfg.OrderPollInterval != 7*time.Second {
		t.Errorf("expected poll interval 7s, got %v", cfg.OrderPollInterval)
	}
	if cfg.ShutdownTimeout != 20*time.Second {
		t.Errorf("expected shutdown timeout 20s, got %v", cfg.ShutdownTimeout)
	}
	if cfg.WorkerPoolSize != 9 {
		t.Errorf("expected worker pool 9, got %d", cfg.WorkerPoolSize)
	}
	if cfg.MaxOrdersBatch != 11 {
		t.Errorf("expected batch size 11, got %d", cfg.MaxOrdersBatch)
	}
	if cfg.JWTSecret != "flag-secret" {
		t.Errorf("expected jwt secret override, got %q", cfg.JWTSecret)
	}
}

func TestLoadValidationErrors(t *testing.T) {
	env := map[string]string{
		"DATABASE_URI":           "postgres://user:pass@localhost/db",
		"ACCRUAL_SYSTEM_ADDRESS": "http://accrual.local",
	}

	_, err := load([]string{"--poll-interval", "bad"}, func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	})
	if err == nil || !strings.Contains(err.Error(), "invalid poll interval") {
		t.Fatalf("expected poll interval error, got %v", err)
	}

	_, err = load([]string{"--shutdown-timeout", "bad"}, func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	})
	if err == nil || !strings.Contains(err.Error(), "invalid shutdown timeout") {
		t.Fatalf("expected shutdown timeout error, got %v", err)
	}
}

func TestLoadNormalizesNonPositiveValues(t *testing.T) {
	env := map[string]string{
		"DATABASE_URI":           "postgres://user:pass@localhost/db",
		"ACCRUAL_SYSTEM_ADDRESS": "http://accrual.local",
		"WORKER_POOL_SIZE":       "-1",
		"POLL_BATCH_SIZE":        "0",
		"ORDER_POLL_INTERVAL":    "0",
		"SHUTDOWN_TIMEOUT":       "0",
	}

	cfg, err := load(nil, func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	})
	if err != nil {
		t.Fatalf("load returned unexpected error: %v", err)
	}

	if cfg.WorkerPoolSize != defaultWorkerPoolSize {
		t.Errorf("expected default worker pool %d, got %d", defaultWorkerPoolSize, cfg.WorkerPoolSize)
	}
	if cfg.MaxOrdersBatch != defaultMaxOrdersBatch {
		t.Errorf("expected default batch size %d, got %d", defaultMaxOrdersBatch, cfg.MaxOrdersBatch)
	}
	if cfg.OrderPollInterval != defaultOrderPollInterval {
		t.Errorf("expected default poll interval %v, got %v", defaultOrderPollInterval, cfg.OrderPollInterval)
	}
	if cfg.ShutdownTimeout != defaultShutdownTimeout {
		t.Errorf("expected default shutdown timeout %v, got %v", defaultShutdownTimeout, cfg.ShutdownTimeout)
	}
}

func TestLoadReadsSecretFromFile(t *testing.T) {
	dir := t.TempDir()
	secretFile := filepath.Join(dir, "secret")
	if err := os.WriteFile(secretFile, []byte("file-secret"), 0o600); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	env := map[string]string{
		"DATABASE_URI":           "postgres://user:pass@localhost/db",
		"ACCRUAL_SYSTEM_ADDRESS": "http://accrual.local",
		"JWT_SECRET_FILE":        secretFile,
	}

	cfg, err := load(nil, func(key string) (string, bool) {
		v, ok := env[key]
		return v, ok
	})
	if err != nil {
		t.Fatalf("load returned unexpected error: %v", err)
	}

	if cfg.JWTSecret != "file-secret" {
		t.Errorf("expected secret from file, got %q", cfg.JWTSecret)
	}
}
