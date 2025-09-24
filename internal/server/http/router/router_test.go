package router

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/polkiloo/gophermart/internal/domain/model"
	"github.com/polkiloo/gophermart/internal/server/http/handlers"
	testhelpers "github.com/polkiloo/gophermart/internal/test"
)

func TestSetupRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	facade := testhelpers.LoyaltyFacadeStub{
		AuthFacadeStub: testhelpers.AuthFacadeStub{},
		OrderFacadeStub: testhelpers.OrderFacadeStub{
			OrdersFn: func(context.Context, int64) ([]model.Order, error) {
				accrual := 5.0
				return []model.Order{{Number: "1", Status: model.OrderStatusProcessed, Accrual: &accrual, UploadedAt: time.Unix(0, 0)}}, nil
			},
		},
		BalanceFacadeStub: testhelpers.BalanceFacadeStub{},
	}
	engine := Setup(facade, logger)

	body, _ := json.Marshal(map[string]string{"login": "user", "password": "pass"})
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	engine.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200 for register, got %d", resp.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/user/orders", nil)
	req.Header.Set("Authorization", "Bearer token")
	resp = httptest.NewRecorder()
	engine.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200 for orders, got %d", resp.Code)
	}
}

var _ handlers.LoyaltyFacade = (*testhelpers.LoyaltyFacadeStub)(nil)
