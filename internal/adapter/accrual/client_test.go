package accrual

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/polkiloo/gophermart/internal/domain/model"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func TestNewHTTPClientValidatesURL(t *testing.T) {
	if _, err := NewHTTPClient("://bad-url", testLogger()); err == nil {
		t.Fatal("expected error for invalid url")
	}
	if _, err := NewHTTPClient("/relative", testLogger()); err == nil {
		t.Fatal("expected error for relative url")
	}
}

func TestHTTPClientAccrualUtilityPositiveScenarios(t *testing.T) {
	baseURL, stop := startAccrualUtility(t)
	if stop == nil {
		t.Skip("accrual utility is not available on this platform")
	}
	defer stop()

	client, err := NewHTTPClient(baseURL, testLogger())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	cases := []struct {
		name string
		run  func(t *testing.T, client *HTTPClient, baseURL string)
	}{
		{
			name: "single order processed",
			run: func(t *testing.T, client *HTTPClient, baseURL string) {
				order := generateOrderNumber()
				registerOrder(t, baseURL, order)
				result := waitForProcessed(t, client, order)
				if result.Status != model.AccrualStatusProcessed {
					t.Fatalf("expected processed status, got %s", result.Status)
				}
				if result.Accrual != nil && *result.Accrual < 0 {
					t.Fatalf("expected non-negative accrual, got %v", *result.Accrual)
				}
			},
		},
		{
			name: "idempotent fetch",
			run: func(t *testing.T, client *HTTPClient, baseURL string) {
				order := generateOrderNumber()
				registerOrder(t, baseURL, order)
				first := waitForProcessed(t, client, order)
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				second, err := client.Fetch(ctx, order)
				if err != nil {
					t.Fatalf("unexpected error on repeated fetch: %v", err)
				}
				if second.Status != first.Status {
					t.Fatalf("expected status %s, got %s", first.Status, second.Status)
				}
				if (first.Accrual == nil) != (second.Accrual == nil) {
					t.Fatalf("accrual presence mismatch: %v vs %v", first.Accrual, second.Accrual)
				}
				if first.Accrual != nil && *first.Accrual != *second.Accrual {
					t.Fatalf("expected accrual %v, got %v", *first.Accrual, *second.Accrual)
				}
			},
		},
		{
			name: "multiple orders sequential",
			run: func(t *testing.T, client *HTTPClient, baseURL string) {
				orders := []string{generateOrderNumber(), generateOrderNumber(), generateOrderNumber()}
				for _, order := range orders {
					registerOrder(t, baseURL, order)
				}
				for _, order := range orders {
					result := waitForProcessed(t, client, order)
					if result.Order != order {
						t.Fatalf("expected order %s, got %s", order, result.Order)
					}
					if result.Status != model.AccrualStatusProcessed {
						t.Fatalf("expected processed status for %s, got %s", order, result.Status)
					}
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.run(t, client, baseURL)
		})
	}
}

func TestFetchHandlesSpecialStatuses(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		header     http.Header
		wantErr    error
	}{
		{name: "not registered", statusCode: http.StatusNoContent, wantErr: ErrOrderNotRegistered},
		{name: "too many requests", statusCode: http.StatusTooManyRequests, header: http.Header{"Retry-After": []string{"5"}}, wantErr: TooManyRequestsError{RetryAfter: 5 * time.Second}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for key, values := range tt.header {
					for _, v := range values {
						w.Header().Add(key, v)
					}
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer srv.Close()

			client, err := NewHTTPClient(srv.URL, testLogger())
			if err != nil {
				t.Fatalf("failed to create client: %v", err)
			}

			_, err = client.Fetch(context.Background(), "1")
			if tt.statusCode == http.StatusTooManyRequests {
				var tm TooManyRequestsError
				if !errors.As(err, &tm) {
					t.Fatalf("expected TooManyRequestsError, got %v", err)
				}
				if tm.RetryAfter != 5*time.Second {
					t.Fatalf("expected retry after 5s, got %v", tm.RetryAfter)
				}
			} else if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestFetchLogsErrorResponses(t *testing.T) {
	called := make(chan struct{}, 1)
	handler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.LevelKey && a.Value.Any() == slog.LevelError {
			select {
			case called <- struct{}{}:
			default:
			}
		}
		return a
	}})
	logger := slog.New(handler)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client, err := NewHTTPClient(srv.URL, logger)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if _, err := client.Fetch(context.Background(), "123"); err == nil {
		t.Fatal("expected error from server")
	}

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("expected error log to be written")
	}
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Now()
	httpTime := now.Add(2 * time.Second).UTC().Format(http.TimeFormat)

	cases := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{name: "empty", header: "", want: 5 * time.Second},
		{name: "seconds", header: "7", want: 7 * time.Second},
		{name: "http date", header: httpTime, want: 2 * time.Second},
		{name: "fallback", header: "bad", want: 5 * time.Second},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseRetryAfter(tc.header)
			if tc.header == httpTime {
				if got <= time.Second || got > 3*time.Second {
					t.Fatalf("unexpected retry duration %v", got)
				}
			} else if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func waitForProcessed(t *testing.T, client *HTTPClient, order string) *model.Accrual {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		result, err := client.Fetch(ctx, order)
		if err != nil {
			var tm TooManyRequestsError
			switch {
			case errors.As(err, &tm):
				time.Sleep(tm.RetryAfter)
				continue
			case errors.Is(err, ErrOrderNotRegistered):
				time.Sleep(50 * time.Millisecond)
				continue
			default:
				t.Fatalf("unexpected error fetching accrual: %v", err)
			}
		}
		if result.Order != order {
			t.Fatalf("unexpected order number %s", result.Order)
		}
		if result.Status == model.AccrualStatusProcessed {
			return result
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func generateOrderNumber() string {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	digits := make([]int, 15)
	for i := range digits {
		digits[i] = rnd.Intn(10)
	}
	sum := 0
	alt := true
	for i := len(digits) - 1; i >= 0; i-- {
		digit := digits[i]
		if alt {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		alt = !alt
	}
	check := (10 - (sum % 10)) % 10
	var buf bytes.Buffer
	for _, d := range digits {
		buf.WriteByte(byte('0' + d))
	}
	buf.WriteByte(byte('0' + check))
	return buf.String()
}

func registerOrder(t *testing.T, baseURL, order string) {
	payload, _ := json.Marshal(map[string]string{"order": order})
	resp, err := http.Post(baseURL+"/api/orders", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to register order: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusConflict {
		t.Fatalf("unexpected status registering order: %d", resp.StatusCode)
	}
}

func startAccrualUtility(t *testing.T) (string, func()) {
	path, ok := accrualBinaryPath()
	if !ok {
		t.Skip("accrual utility not available for current platform")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate port: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	cmd := exec.Command(path, "-a", addr)
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start accrual utility: %v", err)
	}

	readyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		req, _ := http.NewRequestWithContext(readyCtx, http.MethodGet, "http://"+addr+"/api/orders/1", nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			resp.Body.Close()
			break
		}
		if readyCtx.Err() != nil {
			_ = cmd.Process.Kill()
			t.Fatalf("accrual utility did not start: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}

	stop := func() {
		if err := cmd.Process.Signal(os.Interrupt); err != nil {
			_ = cmd.Process.Kill()
		}
		done := make(chan struct{})
		go func() {
			_ = cmd.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_ = cmd.Process.Kill()
		}
	}

	return "http://" + addr, stop
}

func accrualBinaryPath() (string, bool) {
	var name string
	switch runtime.GOOS {
	case "linux":
		if runtime.GOARCH == "amd64" {
			name = "accrual_linux_amd64"
		}
	case "darwin":
		if runtime.GOARCH == "amd64" {
			name = "accrual_darwin_amd64"
		} else if runtime.GOARCH == "arm64" {
			name = "accrual_darwin_arm64"
		}
	case "windows":
		if runtime.GOARCH == "amd64" {
			name = "accrual_windows_amd64"
		}
	}
	if name == "" {
		return "", false
	}
	path := filepath.Join("cmd", "accrual", name)
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	return path, true
}
