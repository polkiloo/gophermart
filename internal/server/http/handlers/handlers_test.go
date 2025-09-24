package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	domainErrors "github.com/polkiloo/gophermart/internal/domain/errors"
	"github.com/polkiloo/gophermart/internal/domain/model"
	"github.com/polkiloo/gophermart/internal/server/http/dto"
	"github.com/polkiloo/gophermart/internal/server/http/middleware"
	testhelpers "github.com/polkiloo/gophermart/internal/test"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func performRequest(t *testing.T, method, path string, handler gin.HandlerFunc, setup func(*gin.Context), body []byte, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	router.Handle(method, path, func(c *gin.Context) {
		if setup != nil {
			setup(c)
		}
		handler(c)
	})

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestCurrentUserID(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	if got := CurrentUserID(c); got != 0 {
		t.Fatalf("expected 0 when not set, got %d", got)
	}

	c.Set(middleware.UserIDContextKey, int64(42))
	if got := CurrentUserID(c); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
}

func TestAuthHandlerRegister(t *testing.T) {
	body, _ := json.Marshal(dto.AuthRequest{Login: "user", Password: "pass"})
	resp := performRequest(t, http.MethodPost, "/register", NewAuthHandler(testhelpers.AuthFacadeStub{}).Register, nil, body, map[string]string{"Content-Type": "application/json"})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	if resp.Header().Get("Authorization") == "" {
		t.Fatalf("expected auth header to be set")
	}
}

func TestAuthHandlerRegisterScenarioMatchesE2E(t *testing.T) {
	login := testhelpers.RandomASCIIString(7, 14)
	password := testhelpers.RandomASCIIString(16, 32)
	body, _ := json.Marshal(dto.AuthRequest{Login: login, Password: password})
	handler := NewAuthHandler(testhelpers.AuthFacadeStub{RegisterFn: func(ctx context.Context, gotLogin, gotPassword string) (string, error) {
		if gotLogin != login || gotPassword != password {
			t.Fatalf("unexpected credentials passed to facade: %q %q", gotLogin, gotPassword)
		}
		return "session-token", nil
	}})
	resp := performRequest(t, http.MethodPost, "/register", handler.Register, nil, body, map[string]string{"Content-Type": "application/json"})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	authHeader := resp.Header().Get("Authorization")
	if authHeader != "Bearer session-token" {
		t.Fatalf("unexpected authorization header %q", authHeader)
	}
	result := resp.Result()
	t.Cleanup(func() {
		_ = result.Body.Close()
	})
	cookies := result.Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected auth cookie to be set")
	}
	foundCookie := false
	for _, cookie := range cookies {
		if cookie.Name == "gophermart_token" {
			if cookie.Value != "session-token" {
				t.Fatalf("unexpected token stored in cookie: %q", cookie.Value)
			}
			foundCookie = true
			break
		}
	}
	if !foundCookie {
		t.Fatal("expected auth cookie named gophermart_token")
	}
}

func TestAuthHandlerRegisterFailures(t *testing.T) {
	tests := []struct {
		name   string
		facade testhelpers.AuthFacadeStub
		body   []byte
		status int
	}{
		{name: "bad json", body: []byte("not json"), status: http.StatusBadRequest},
		{name: "invalid credentials", body: []byte(`{"login":"","password":""}`), facade: testhelpers.AuthFacadeStub{RegisterFn: func(context.Context, string, string) (string, error) {
			return "", domainErrors.ErrInvalidCredentials
		}}, status: http.StatusBadRequest},
		{name: "already exists", body: []byte(`{"login":"a","password":"b"}`), facade: testhelpers.AuthFacadeStub{RegisterFn: func(context.Context, string, string) (string, error) {
			return "", domainErrors.ErrAlreadyExists
		}}, status: http.StatusConflict},
		{name: "internal", body: []byte(`{"login":"a","password":"b"}`), facade: testhelpers.AuthFacadeStub{RegisterFn: func(context.Context, string, string) (string, error) {
			return "", errors.New("boom")
		}}, status: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := performRequest(t, http.MethodPost, "/register", NewAuthHandler(tt.facade).Register, nil, tt.body, map[string]string{"Content-Type": "application/json"})
			if resp.Code != tt.status {
				t.Fatalf("expected status %d, got %d", tt.status, resp.Code)
			}
		})
	}
}

func TestAuthHandlerLogin(t *testing.T) {
	body, _ := json.Marshal(dto.AuthRequest{Login: "user", Password: "pass"})
	resp := performRequest(t, http.MethodPost, "/login", NewAuthHandler(testhelpers.AuthFacadeStub{}).Login, nil, body, map[string]string{"Content-Type": "application/json"})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
}

func TestAuthHandlerLoginFailures(t *testing.T) {
	tests := []struct {
		name   string
		facade testhelpers.AuthFacadeStub
		body   []byte
		status int
	}{
		{name: "bad json", body: []byte("not json"), status: http.StatusBadRequest},
		{name: "invalid", body: []byte(`{"login":"a","password":"b"}`), facade: testhelpers.AuthFacadeStub{AuthenticateFn: func(context.Context, string, string) (string, error) {
			return "", domainErrors.ErrInvalidCredentials
		}}, status: http.StatusUnauthorized},
		{name: "internal", body: []byte(`{"login":"a","password":"b"}`), facade: testhelpers.AuthFacadeStub{AuthenticateFn: func(context.Context, string, string) (string, error) {
			return "", errors.New("boom")
		}}, status: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := performRequest(t, http.MethodPost, "/login", NewAuthHandler(tt.facade).Login, nil, tt.body, map[string]string{"Content-Type": "application/json"})
			if resp.Code != tt.status {
				t.Fatalf("expected status %d, got %d", tt.status, resp.Code)
			}
		})
	}
}

func TestOrderHandlerUpload(t *testing.T) {
	facade := testhelpers.OrderFacadeStub{UploadFn: func(context.Context, int64, string) (*model.Order, bool, error) {
		return &model.Order{Number: "1"}, true, nil
	}}
	handler := NewOrderHandler(facade)
	body := []byte("79927398713")
	resp := performRequest(t, http.MethodPost, "/upload", handler.Upload, func(c *gin.Context) {
		c.Set(middleware.UserIDContextKey, int64(1))
	}, body, nil)
	if resp.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", resp.Code)
	}
}

func TestOrderHandlerUploadFailures(t *testing.T) {
	tests := []struct {
		name   string
		facade testhelpers.OrderFacadeStub
		body   []byte
		status int
	}{
		{name: "bad body", body: []byte(""), status: http.StatusBadRequest},
		{name: "conflict", body: []byte("79927398713"), facade: testhelpers.OrderFacadeStub{UploadFn: func(context.Context, int64, string) (*model.Order, bool, error) {
			return nil, false, domainErrors.ErrAlreadyExists
		}}, status: http.StatusConflict},
		{name: "invalid", body: []byte("123"), facade: testhelpers.OrderFacadeStub{UploadFn: func(context.Context, int64, string) (*model.Order, bool, error) {
			return nil, false, domainErrors.ErrInvalidOrderNumber
		}}, status: http.StatusUnprocessableEntity},
		{name: "internal", body: []byte("79927398713"), facade: testhelpers.OrderFacadeStub{UploadFn: func(context.Context, int64, string) (*model.Order, bool, error) {
			return nil, false, errors.New("boom")
		}}, status: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := performRequest(t, http.MethodPost, "/upload", NewOrderHandler(tt.facade).Upload, func(c *gin.Context) {
				c.Set(middleware.UserIDContextKey, int64(1))
			}, tt.body, nil)
			if resp.Code != tt.status {
				t.Fatalf("expected status %d, got %d", tt.status, resp.Code)
			}
		})
	}
}

func TestOrderHandlerList(t *testing.T) {
	orders := []model.Order{{Number: "1"}, {Number: "2"}}
	facade := testhelpers.OrderFacadeStub{OrdersFn: func(context.Context, int64) ([]model.Order, error) {
		return orders, nil
	}}
	handler := NewOrderHandler(facade)
	resp := performRequest(t, http.MethodGet, "/orders", handler.List, func(c *gin.Context) {
		c.Set(middleware.UserIDContextKey, int64(1))
	}, nil, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	var decoded []model.Order
	if err := json.Unmarshal(resp.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(decoded) != len(orders) {
		t.Fatalf("expected %d orders, got %d", len(orders), len(decoded))
	}
}

func TestBalanceHandlerSummary(t *testing.T) {
	summary := &model.BalanceSummary{Current: 10, Withdrawn: 5}
	facade := testhelpers.BalanceFacadeStub{BalanceFn: func(context.Context, int64) (*model.BalanceSummary, error) {
		return summary, nil
	}}
	handler := NewBalanceHandler(facade)
	resp := performRequest(t, http.MethodGet, "/balance", handler.Summary, func(c *gin.Context) {
		c.Set(middleware.UserIDContextKey, int64(1))
	}, nil, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	var decoded dto.BalanceResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if decoded.Current != summary.Current {
		t.Fatalf("unexpected summary: %+v", decoded)
	}
}

func TestBalanceHandlerWithdraw(t *testing.T) {
	facade := testhelpers.BalanceFacadeStub{}
	handler := NewBalanceHandler(facade)
	body := []byte(`{"order":"79927398713","sum":10}`)
	resp := performRequest(t, http.MethodPost, "/withdraw", handler.Withdraw, func(c *gin.Context) {
		c.Set(middleware.UserIDContextKey, int64(1))
	}, body, map[string]string{"Content-Type": "application/json"})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
}

func TestBalanceHandlerWithdrawFailures(t *testing.T) {
	tests := []struct {
		name   string
		facade testhelpers.BalanceFacadeStub
		body   []byte
		status int
	}{
		{name: "bad json", body: []byte("oops"), status: http.StatusBadRequest},
		{name: "invalid order", body: []byte(`{"order":"1","sum":10}`), facade: testhelpers.BalanceFacadeStub{WithdrawFn: func(context.Context, int64, string, float64) error {
			return domainErrors.ErrInvalidOrderNumber
		}}, status: http.StatusUnprocessableEntity},
		{name: "invalid amount", body: []byte(`{"order":"79927398713","sum":-1}`), facade: testhelpers.BalanceFacadeStub{WithdrawFn: func(context.Context, int64, string, float64) error {
			return domainErrors.ErrInvalidAmount
		}}, status: http.StatusUnprocessableEntity},
		{name: "insufficient", body: []byte(`{"order":"79927398713","sum":10}`), facade: testhelpers.BalanceFacadeStub{WithdrawFn: func(context.Context, int64, string, float64) error {
			return domainErrors.ErrInsufficientBalance
		}}, status: http.StatusPaymentRequired},
		{name: "internal", body: []byte(`{"order":"79927398713","sum":10}`), facade: testhelpers.BalanceFacadeStub{WithdrawFn: func(context.Context, int64, string, float64) error {
			return errors.New("boom")
		}}, status: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := performRequest(t, http.MethodPost, "/withdraw", NewBalanceHandler(tt.facade).Withdraw, func(c *gin.Context) {
				c.Set(middleware.UserIDContextKey, int64(1))
			}, tt.body, map[string]string{"Content-Type": "application/json"})
			if resp.Code != tt.status {
				t.Fatalf("expected status %d, got %d", tt.status, resp.Code)
			}
		})
	}
}

func TestBalanceHandlerWithdrawals(t *testing.T) {
	withdrawals := []model.Withdrawal{{OrderNumber: "1", Sum: 1, ProcessedAt: time.Unix(0, 0)}}
	facade := testhelpers.BalanceFacadeStub{WithdrawalsFn: func(context.Context, int64) ([]model.Withdrawal, error) {
		return withdrawals, nil
	}}
	handler := NewBalanceHandler(facade)
	resp := performRequest(t, http.MethodGet, "/withdrawals", handler.Withdrawals, func(c *gin.Context) {
		c.Set(middleware.UserIDContextKey, int64(1))
	}, nil, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
}
