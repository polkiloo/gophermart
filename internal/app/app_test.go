package app

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/polkiloo/gophermart/internal/config"
	"github.com/polkiloo/gophermart/internal/domain/model"
	testhelpers "github.com/polkiloo/gophermart/internal/test"
	"github.com/polkiloo/gophermart/internal/worker"
)

func newTestOrderProcessor() *worker.OrderProcessor {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	return worker.NewOrderProcessor(&testhelpers.WorkerFacadeStub{}, 10*time.Millisecond, 1, 1, logger)
}

func TestNewHTTPServer(t *testing.T) {
	cfg := &config.Config{RunAddress: ":9999"}
	router := gin.New()
	server := newHTTPServer(serverParams{Config: cfg, Router: router})
	if server.Addr != ":9999" {
		t.Fatalf("expected address :9999, got %q", server.Addr)
	}
	if server.Handler != router {
		t.Fatalf("expected handler to be router")
	}
}

func TestNewOrderProcessorUsesConfig(t *testing.T) {
	proc := newOrderProcessor(workerParams{
		Facade: &LoyaltyFacade{},
		Config: &config.Config{OrderPollInterval: 15 * time.Second, MaxOrdersBatch: 3, WorkerPoolSize: 4},
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	})
	if proc == nil {
		t.Fatal("expected order processor instance")
	}
}

func TestRegisterLifecycleStartStop(t *testing.T) {
	recorder := &testhelpers.LifecycleRecorder{}
	shutdowner := &testhelpers.ShutdownerStub{Called: make(chan struct{}, 1)}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	server := &http.Server{Addr: "127.0.0.1:0", Handler: http.NewServeMux()}
	worker := newTestOrderProcessor()
	cfg := &config.Config{ShutdownTimeout: 100 * time.Millisecond}

	registerLifecycle(lifecycleParams{
		Lifecycle:  recorder,
		Shutdowner: shutdowner,
		Logger:     logger,
		Server:     server,
		Worker:     worker,
		Config:     cfg,
	})

	if len(recorder.Hooks) != 1 {
		t.Fatalf("expected one hook registered, got %d", len(recorder.Hooks))
	}

	hook := recorder.Hooks[0]
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := hook.OnStart(ctx); err != nil {
		t.Fatalf("on start failed: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = hook.OnStop(context.Background())
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected on stop to finish")
	}
}

func TestRegisterLifecycleShutdownOnServerError(t *testing.T) {
	recorder := &testhelpers.LifecycleRecorder{}
	shutdowner := &testhelpers.ShutdownerStub{Called: make(chan struct{}, 1)}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	server := &http.Server{Addr: "bad addr"}
	worker := newTestOrderProcessor()

	registerLifecycle(lifecycleParams{
		Lifecycle:  recorder,
		Shutdowner: shutdowner,
		Logger:     logger,
		Server:     server,
		Worker:     worker,
		Config:     &config.Config{ShutdownTimeout: time.Second},
	})

	hook := recorder.Hooks[0]
	if err := hook.OnStart(context.Background()); err != nil {
		t.Fatalf("on start returned error: %v", err)
	}

	select {
	case <-shutdowner.Called:
	case <-time.After(time.Second):
		t.Fatal("expected shutdown to be triggered on server error")
	}

	_ = hook.OnStop(context.Background())
}

func TestWorkerFacadeStubRecording(t *testing.T) {
	facade := &testhelpers.WorkerFacadeStub{}
	if err := facade.UpdateOrderStatus(context.Background(), 1, model.OrderStatusProcessed, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(facade.Updates) != 1 {
		t.Fatalf("expected update to be recorded")
	}
}

func TestLifecycleRecorderAppend(t *testing.T) {
	recorder := &testhelpers.LifecycleRecorder{}
	hook := fx.Hook{}
	recorder.Append(hook)
	if len(recorder.Hooks) != 1 {
		t.Fatalf("expected hook to be appended")
	}
}

func TestShutdownerStub(t *testing.T) {
	shutdowner := &testhelpers.ShutdownerStub{Called: make(chan struct{}, 1)}
	if err := shutdowner.Shutdown(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	select {
	case <-shutdowner.Called:
	default:
		t.Fatal("expected shutdown notification")
	}
}
