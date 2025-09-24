package worker

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/polkiloo/gophermart/internal/adapter/accrual"
	"github.com/polkiloo/gophermart/internal/domain/model"
	testhelpers "github.com/polkiloo/gophermart/internal/test"
)

func TestNewOrderProcessorDefaults(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	proc := NewOrderProcessor(&testhelpers.WorkerFacadeStub{}, time.Second, 0, 0, logger)
	if proc.batchSize != 1 {
		t.Fatalf("expected batch size default to 1, got %d", proc.batchSize)
	}
	if proc.workers != 1 {
		t.Fatalf("expected workers default to 1, got %d", proc.workers)
	}
}

func TestOrderProcessorProcessesOrders(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	facade := &testhelpers.WorkerFacadeStub{Orders: [][]model.Order{{{ID: 1, Number: "1"}}}}
	proc := NewOrderProcessor(facade, 10*time.Millisecond, 1, 1, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	proc.Start(ctx)

	deadline := time.After(500 * time.Millisecond)
	for {
		facade.Lock()
		processed := len(facade.Updates) > 0
		facade.Unlock()
		if processed {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for order processing")
		case <-time.After(10 * time.Millisecond):
		}
	}

	proc.Stop()
	facade.Lock()
	defer facade.Unlock()
	if len(facade.Updates) == 0 {
		t.Fatalf("expected order status update")
	}
	if facade.Updates[0].Status != model.OrderStatusProcessed {
		t.Fatalf("expected processed status, got %v", facade.Updates[0].Status)
	}
}

func TestOrderProcessorHandlesRateLimiting(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	attempts := int32(0)
	facade := &testhelpers.WorkerFacadeStub{
		Orders: [][]model.Order{{{ID: 1, Number: "1"}}, {{ID: 1, Number: "1"}}},
		CheckFn: func(ctx context.Context, number string) (*model.Accrual, error) {
			if atomic.AddInt32(&attempts, 1) == 1 {
				return nil, accrual.TooManyRequestsError{RetryAfter: 10 * time.Millisecond}
			}
			accrualValue := 1.0
			return &model.Accrual{Status: model.AccrualStatusProcessed, Accrual: &accrualValue}, nil
		},
	}

	proc := NewOrderProcessor(facade, 5*time.Millisecond, 1, 1, logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	proc.Start(ctx)

	deadline := time.After(time.Second)
	for {
		facade.Lock()
		if len(facade.Updates) > 0 {
			facade.Unlock()
			break
		}
		facade.Unlock()
		select {
		case <-deadline:
			t.Fatal("timeout waiting for retry")
		case <-time.After(10 * time.Millisecond):
		}
	}
	proc.Stop()
}
