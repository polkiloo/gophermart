package accrual

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/polkiloo/gophermart/internal/domain/model"
)

// ErrOrderNotRegistered indicates accrual system doesn't know order yet.
var ErrOrderNotRegistered = errors.New("order not registered")

// TooManyRequestsError represents rate limiting signal from accrual system.
type TooManyRequestsError struct {
	RetryAfter time.Duration
}

func (e TooManyRequestsError) Error() string {
	return fmt.Sprintf("too many requests, retry after %s", e.RetryAfter)
}

// Client exposes operations to query accrual service.
type Client interface {
	Fetch(ctx context.Context, number string) (*model.Accrual, error)
}

// HTTPClient implements Client via HTTP API.
type HTTPClient struct {
	baseURL    *url.URL
	httpClient *http.Client
	logger     *slog.Logger
}

// response mirrors JSON payload from accrual system.
type response struct {
	Order   string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual,omitempty"`
}

// NewHTTPClient creates HTTP accrual client with default timeout.
func NewHTTPClient(baseURL string, logger *slog.Logger) (*HTTPClient, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse accrual url: %w", err)
	}
	if !parsed.IsAbs() {
		return nil, fmt.Errorf("accrual url must be absolute")
	}
	return &HTTPClient{
		baseURL: parsed,
		logger:  logger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// Fetch queries accrual system for order status.
func (c *HTTPClient) Fetch(ctx context.Context, number string) (*model.Accrual, error) {
	endpoint := *c.baseURL
	endpoint.Path = path.Join(endpoint.Path, "/api/orders/", number)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var data response
		if err := json.Unmarshal(body, &data); err != nil {
			return nil, err
		}
		return &model.Accrual{Order: data.Order, Status: model.AccrualStatus(data.Status), Accrual: data.Accrual}, nil
	case http.StatusNoContent:
		return nil, ErrOrderNotRegistered
	case http.StatusTooManyRequests:
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		return nil, TooManyRequestsError{RetryAfter: retryAfter}
	default:
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error("accrual request failed", slog.Int("status", resp.StatusCode), slog.String("body", string(body)))
		return nil, fmt.Errorf("accrual error: %s", resp.Status)
	}
}

func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 5 * time.Second
	}
	if seconds, err := strconv.Atoi(header); err == nil {
		return time.Duration(seconds) * time.Second
	}
	if t, err := http.ParseTime(header); err == nil {
		return time.Until(t)
	}
	return 5 * time.Second
}
