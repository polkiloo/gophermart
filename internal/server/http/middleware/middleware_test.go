package middleware

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	pkgAuth "github.com/polkiloo/gophermart/internal/pkg/auth"
	testhelpers "github.com/polkiloo/gophermart/internal/test"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAuthRequired(t *testing.T) {
	router := gin.New()
	router.Use(AuthRequired(testhelpers.TokenParserStub{}))
	router.GET("/", func(c *gin.Context) {})
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/", nil))
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", resp.Code)
	}

	router = gin.New()
	router.Use(AuthRequired(testhelpers.TokenParserStub{Err: pkgAuth.ErrInvalidToken}))
	router.GET("/", func(c *gin.Context) {})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid token, got %d", resp.Code)
	}

	router = gin.New()
	router.Use(AuthRequired(testhelpers.TokenParserStub{Err: context.DeadlineExceeded}))
	router.GET("/", func(c *gin.Context) {})
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.Code)
	}

	var storedID int64
	router = gin.New()
	router.Use(AuthRequired(testhelpers.TokenParserStub{ID: 42}))
	router.GET("/", func(c *gin.Context) {
		if v, ok := c.Get(UserIDContextKey); ok {
			storedID = v.(int64)
		}
		c.Status(http.StatusOK)
	})
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if storedID != 42 {
		t.Fatalf("expected user id 42, got %d", storedID)
	}
}

func TestSetAuthCookie(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	SetAuthCookie(c, "token")
	if got := recorder.Header().Get("Authorization"); got != "Bearer token" {
		t.Fatalf("expected auth header, got %q", got)
	}
	result := recorder.Result()
	t.Cleanup(func() {
		_ = result.Body.Close()
	})
	cookies := result.Cookies()
	if len(cookies) == 0 || cookies[0].Value != "token" {
		t.Fatalf("expected cookie with token, got %+v", cookies)
	}
}

func TestExtractToken(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	if token := extractToken(c); token != "" {
		t.Fatalf("expected empty token, got %q", token)
	}
	c.Request.Header.Set("Authorization", "Bearer abc")
	if token := extractToken(c); token != "abc" {
		t.Fatalf("expected token from header, got %q", token)
	}
	c.Request.Header.Del("Authorization")
	c.Request.AddCookie(&http.Cookie{Name: authCookieName, Value: "cookie"})
	if token := extractToken(c); token != "cookie" {
		t.Fatalf("expected token from cookie, got %q", token)
	}
}

func TestDecompressRequest(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write([]byte("payload"))
	_ = gz.Close()

	router := gin.New()
	router.Use(DecompressRequest())
	var body string
	router.POST("/", func(c *gin.Context) {
		data, _ := io.ReadAll(c.Request.Body)
		body = string(data)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/", io.NopCloser(bytes.NewReader(buf.Bytes())))
	req.Header.Set("Content-Encoding", "gzip")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if body != "payload" {
		t.Fatalf("expected decompressed payload, got %q", body)
	}

	req = httptest.NewRequest(http.MethodPost, "/", io.NopCloser(bytes.NewReader([]byte("plain"))))
	resp = httptest.NewRecorder()
	body = ""
	router.ServeHTTP(resp, req)
	if body != "plain" {
		t.Fatalf("expected plain body, got %q", body)
	}
}

func TestRequestLogger(t *testing.T) {
	var logged bool
	handler := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.LevelKey && a.Value.Any() == slog.LevelInfo {
			logged = true
		}
		return a
	}})
	logger := slog.New(handler)

	router := gin.New()
	router.Use(RequestLogger(logger))
	router.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/", nil))
	if !logged {
		t.Fatalf("expected request to be logged")
	}
}
