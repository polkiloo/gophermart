package worker

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/polkiloo/gophermart/internal/adapter/accrual"
	"github.com/polkiloo/gophermart/internal/domain/model"
)

// LoyaltyFacade exposes the subset of application functionality required by the worker.
type LoyaltyFacade interface {
	OrdersForProcessing(ctx context.Context, limit int) ([]model.Order, error)
	CheckAccrual(ctx context.Context, number string) (*model.Accrual, error)
	UpdateOrderStatus(ctx context.Context, orderID int64, status model.OrderStatus, accrual *float64) error
}

// OrderProcessor polls accrual system and updates order statuses concurrently.
type OrderProcessor struct {
	facade       LoyaltyFacade
	pollInterval time.Duration
	batchSize    int
	workers      int
	logger       *slog.Logger

	jobs   chan model.Order
	wg     sync.WaitGroup
	cancel context.CancelFunc
	mu     sync.Mutex
}

// NewOrderProcessor constructs order processor worker pool.
func NewOrderProcessor(facade LoyaltyFacade, pollInterval time.Duration, batchSize, workers int, logger *slog.Logger) *OrderProcessor {
	if workers <= 0 {
		workers = 1
	}
	if batchSize <= 0 {
		batchSize = 1
	}
	return &OrderProcessor{
		facade:       facade,
		pollInterval: pollInterval,
		batchSize:    batchSize,
		workers:      workers,
		logger:       logger,
		jobs:         make(chan model.Order, batchSize*workers),
	}
}

// Start launches background processing.
func (p *OrderProcessor) Start(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()

	runCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(runCtx)
	}

	p.wg.Add(1)
	go p.dispatch(runCtx)
}

// Stop waits for all workers to finish.
func (p *OrderProcessor) Stop() {
	p.mu.Lock()
	if p.cancel != nil {
		p.cancel()
		p.cancel = nil
	}
	p.mu.Unlock()

	p.wg.Wait()
}

func (p *OrderProcessor) dispatch(ctx context.Context) {
	defer p.wg.Done()
	defer close(p.jobs)
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.fetchAndDispatch(ctx)
		}
	}
}

func (p *OrderProcessor) fetchAndDispatch(ctx context.Context) {
	orders, err := p.facade.OrdersForProcessing(ctx, p.batchSize)
	if err != nil {
		p.logger.Error("fetch orders for processing failed", slog.String("error", err.Error()))
		return
	}
	for _, order := range orders {
		select {
		case <-ctx.Done():
			return
		case p.jobs <- order:
		}
	}
}

func (p *OrderProcessor) worker(ctx context.Context) {
	defer p.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case order, ok := <-p.jobs:
			if !ok {
				return
			}
			p.handleOrder(ctx, order)
		}
	}
}

func (p *OrderProcessor) handleOrder(ctx context.Context, order model.Order) {
	result, err := p.facade.CheckAccrual(ctx, order.Number)
	if err != nil {
		switch e := err.(type) {
		case accrual.TooManyRequestsError:
			p.logger.Warn("accrual rate limited", slog.Duration("retry_after", e.RetryAfter))
			time.Sleep(e.RetryAfter)
		default:
			if errors.Is(err, accrual.ErrOrderNotRegistered) {
				time.Sleep(p.pollInterval)
				return
			}
			p.logger.Error("accrual fetch failed", slog.String("order", order.Number), slog.String("error", err.Error()))
		}
		return
	}

	var status model.OrderStatus
	switch result.Status {
	case model.AccrualStatusRegistered, model.AccrualStatusProcessing:
		status = model.OrderStatusProcessing
	case model.AccrualStatusInvalid:
		status = model.OrderStatusInvalid
	case model.AccrualStatusProcessed:
		status = model.OrderStatusProcessed
	default:
		status = model.OrderStatusProcessing
	}

	if err := p.facade.UpdateOrderStatus(ctx, order.ID, status, result.Accrual); err != nil {
		p.logger.Error("update order status failed", slog.String("order", order.Number), slog.String("error", err.Error()))
	}
}
