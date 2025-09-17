package config

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

// Config holds application level configuration loaded from environment and flags.
type Config struct {
	RunAddress           string
	DatabaseURI          string
	AccrualSystemAddress string
	JWTSecret            string
	OrderPollInterval    time.Duration
	WorkerPoolSize       int
	ShutdownTimeout      time.Duration
	MaxOrdersBatch       int
}

const (
	defaultRunAddress        = ":8080"
	defaultJWTSecret         = "change-me-in-production"
	defaultOrderPollInterval = 3 * time.Second
	defaultWorkerPoolSize    = 4
	defaultShutdownTimeout   = 10 * time.Second
	defaultMaxOrdersBatch    = 32
)

// Load parses configuration from flags and environment variables.
func Load() (*Config, error) {
	return load(os.Args[1:], os.LookupEnv)
}

type envLookup func(string) (string, bool)

func load(args []string, lookup envLookup) (*Config, error) {
	cfg := &Config{
		RunAddress:           getString(lookup, "RUN_ADDRESS", defaultRunAddress),
		DatabaseURI:          getString(lookup, "DATABASE_URI", ""),
		AccrualSystemAddress: getString(lookup, "ACCRUAL_SYSTEM_ADDRESS", ""),
		JWTSecret:            getString(lookup, "JWT_SECRET", defaultJWTSecret),
		OrderPollInterval:    getDuration(lookup, "ORDER_POLL_INTERVAL", defaultOrderPollInterval),
		WorkerPoolSize:       getInt(lookup, "WORKER_POOL_SIZE", defaultWorkerPoolSize),
		ShutdownTimeout:      getDuration(lookup, "SHUTDOWN_TIMEOUT", defaultShutdownTimeout),
		MaxOrdersBatch:       getInt(lookup, "POLL_BATCH_SIZE", defaultMaxOrdersBatch),
	}

	fs := flag.NewFlagSet("gophermart", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		pollIntervalStr    = cfg.OrderPollInterval.String()
		shutdownTimeoutStr = cfg.ShutdownTimeout.String()
	)

	fs.StringVar(&cfg.RunAddress, "a", cfg.RunAddress, "HTTP server listen address")
	fs.StringVar(&cfg.DatabaseURI, "d", cfg.DatabaseURI, "PostgreSQL DSN")
	fs.StringVar(&cfg.AccrualSystemAddress, "r", cfg.AccrualSystemAddress, "Accrual system base URL")
	fs.StringVar(&cfg.JWTSecret, "jwt-secret", cfg.JWTSecret, "Secret for signing auth tokens")
	fs.IntVar(&cfg.WorkerPoolSize, "worker-pool", cfg.WorkerPoolSize, "Number of concurrent order workers")
	fs.StringVar(&pollIntervalStr, "poll-interval", pollIntervalStr, "Interval between accrual polls")
	fs.StringVar(&shutdownTimeoutStr, "shutdown-timeout", shutdownTimeoutStr, "Graceful shutdown timeout")
	fs.IntVar(&cfg.MaxOrdersBatch, "poll-batch", cfg.MaxOrdersBatch, "Maximum orders per polling batch")

	if err := fs.Parse(args); err != nil {
		return nil, fmt.Errorf("parse flags: %w", err)
	}

	var err error

	if cfg.OrderPollInterval, err = time.ParseDuration(pollIntervalStr); err != nil {
		return nil, fmt.Errorf("invalid poll interval: %w", err)
	}

	if cfg.ShutdownTimeout, err = time.ParseDuration(shutdownTimeoutStr); err != nil {
		return nil, fmt.Errorf("invalid shutdown timeout: %w", err)
	}

	if secretFile, ok := lookup("JWT_SECRET_FILE"); ok && secretFile != "" {
		content, err := os.ReadFile(secretFile)
		if err != nil {
			return nil, fmt.Errorf("read jwt secret file: %w", err)
		}
		cfg.JWTSecret = string(content)
	}

	if cfg.WorkerPoolSize <= 0 {
		cfg.WorkerPoolSize = defaultWorkerPoolSize
	}

	if cfg.MaxOrdersBatch <= 0 {
		cfg.MaxOrdersBatch = defaultMaxOrdersBatch
	}

	if cfg.OrderPollInterval <= 0 {
		cfg.OrderPollInterval = defaultOrderPollInterval
	}

	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = defaultShutdownTimeout
	}

	if cfg.DatabaseURI == "" {
		return nil, fmt.Errorf("database URI must be provided")
	}

	if cfg.AccrualSystemAddress == "" {
		return nil, fmt.Errorf("accrual system address must be provided")
	}

	return cfg, nil
}

func getString(lookup envLookup, key, def string) string {
	if v, ok := lookup(key); ok && v != "" {
		return v
	}
	return def
}

func getInt(lookup envLookup, key string, def int) int {
	if v, ok := lookup(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getDuration(lookup envLookup, key string, def time.Duration) time.Duration {
	if v, ok := lookup(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
