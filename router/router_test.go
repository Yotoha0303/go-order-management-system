package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-order-management-system/internal/auth"
	"go-order-management-system/internal/handler"

	"github.com/gin-gonic/gin"
)

type denyRoleChecker struct {
	calls int
}

func (c *denyRoleChecker) HasRole(context.Context, int64, string) (bool, error) {
	c.calls++
	return false, nil
}

func TestBusinessRoutesRequireAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager, err := auth.NewTokenManager("0123456789abcdef0123456789abcdef", "test", time.Hour)
	if err != nil {
		t.Fatalf("new token manager: %v", err)
	}
	router := SetupRouters(nil, Handlers{
		Product:   &handler.ProductHandler{},
		Inventory: &handler.InventoryHandler{},
		StockLog:  &handler.StockLogHandler{},
		Order:     &handler.OrderHandler{},
		Health:    &handler.HealthHandler{},
		User:      &handler.UserHandler{},
	}, tokenManager, nil)

	for _, path := range []string{
		"/api/v1/products",
		"/api/v1/inventory/products/1",
		"/api/v1/stock-logs",
		"/api/v1/operation-logs",
		"/api/v1/orders",
		"/api/v1/users/me",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected %s to require authentication, got %d", path, recorder.Code)
		}
	}
}

func TestAdminRoutesRequireAdminRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager, err := auth.NewTokenManager("0123456789abcdef0123456789abcdef", "test", time.Hour)
	if err != nil {
		t.Fatalf("new token manager: %v", err)
	}
	token, err := tokenManager.GenerateAccessToken(7, "normal-user")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	checker := &denyRoleChecker{}
	r := SetupRouters(nil, Handlers{
		Product:   &handler.ProductHandler{},
		Inventory: &handler.InventoryHandler{},
		StockLog:  &handler.StockLogHandler{},
		Order:     &handler.OrderHandler{},
		Health:    &handler.HealthHandler{},
		User:      &handler.UserHandler{},
	}, tokenManager, checker)

	for _, route := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/products"},
		{method: http.MethodPost, path: "/api/v1/inventory/redis/rebuild"},
		{method: http.MethodGet, path: "/api/v1/inventory/redis/reconcile"},
	} {
		request := httptest.NewRequest(route.method, route.path, nil)
		request.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()
		r.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("expected %s status %d, got %d", route.path, http.StatusForbidden, recorder.Code)
		}
	}
	if checker.calls != 3 {
		t.Fatalf("expected three role checks, got %d", checker.calls)
	}
}

func TestMetricsEndpointIsPublicAndRecordsRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager, err := auth.NewTokenManager("0123456789abcdef0123456789abcdef", "test", time.Hour)
	if err != nil {
		t.Fatalf("new token manager: %v", err)
	}
	r := SetupRouters(nil, Handlers{
		Product:   &handler.ProductHandler{},
		Inventory: &handler.InventoryHandler{},
		StockLog:  &handler.StockLogHandler{},
		Order:     &handler.OrderHandler{},
		Health:    &handler.HealthHandler{},
		User:      &handler.UserHandler{},
	}, tokenManager, nil)

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/orders", nil))

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}
	body := recorder.Body.String()
	want := `app_http_requests_total{method="GET",route="/api/v1/orders",status="401"} 1`
	if !strings.Contains(body, want) {
		t.Fatalf("expected metrics output to contain %q, got:\n%s", want, body)
	}
}
