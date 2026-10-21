package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSMiddleware_PreflightOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORSMiddleware())

	handlerCalled := false
	r.POST("/api/v1/test", func(c *gin.Context) {
		handlerCalled = true
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/test", nil)
	req.Header.Set("Origin", "https://go-order-management-system.vercel.app")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type, X-Request-ID")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 for OPTIONS preflight, got %d", w.Code)
	}

	if handlerCalled {
		t.Fatal("downstream handler should not be called for preflight OPTIONS request")
	}

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://go-order-management-system.vercel.app" {
		t.Fatalf("expected Access-Control-Allow-Origin to match request origin, got %q", got)
	}

	allowMethods := w.Header().Get("Access-Control-Allow-Methods")
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"} {
		if !strings.Contains(allowMethods, method) {
			t.Fatalf("expected allowed methods to contain %s, got %q", method, allowMethods)
		}
	}

	allowHeaders := w.Header().Get("Access-Control-Allow-Headers")
	for _, header := range []string{"Authorization", "Content-Type", "X-Request-ID", "X-Idempotency-Key"} {
		if !strings.Contains(allowHeaders, header) {
			t.Fatalf("expected allowed headers to contain %s, got %q", header, allowHeaders)
		}
	}

	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected Access-Control-Allow-Credentials true, got %q", got)
	}
}

func TestCORSMiddleware_ActualRequestWithOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORSMiddleware())

	r.GET("/api/v1/test", func(c *gin.Context) {
		c.String(http.StatusOK, "hello")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "hello" {
		t.Fatalf("expected body 'hello', got %q", w.Body.String())
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("expected origin http://localhost:5173, got %q", got)
	}
}

func TestCORSMiddleware_RequestWithoutOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORSMiddleware())

	r.GET("/api/v1/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected default *, got %q", got)
	}
}
