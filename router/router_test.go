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
	"go-order-management-system/internal/model"
	"go-order-management-system/internal/request"

	"github.com/gin-gonic/gin"
)

type denyRoleChecker struct {
	calls int
}

func (c *denyRoleChecker) HasRole(context.Context, int64, string) (bool, error) {
	c.calls++
	return false, nil
}

type allowRoleChecker struct {
	calls int
}

func (c *allowRoleChecker) HasRole(context.Context, int64, string) (bool, error) {
	c.calls++
	return true, nil
}

type spyAuditRecorder struct {
	logs []*model.OperationLog
}

func (s *spyAuditRecorder) CreateOperationLog(_ context.Context, log *model.OperationLog) error {
	s.logs = append(s.logs, log)
	return nil
}

type mockOrderServiceForRouter struct {
	listAdminCalls int
	lastReq        request.ListAdminOrdersRequest
}

func (m *mockOrderServiceForRouter) CreateOrder(context.Context, int64, request.CreateOrderRequest) (*model.Order, error) {
	panic("unexpected CreateOrder call")
}

func (m *mockOrderServiceForRouter) ListOrders(context.Context, int64, int, int) ([]*model.Order, int64, error) {
	panic("unexpected ListOrders call")
}

func (m *mockOrderServiceForRouter) ListAdminOrders(_ context.Context, req request.ListAdminOrdersRequest) ([]*model.Order, int64, error) {
	m.listAdminCalls++
	m.lastReq = req
	return []*model.Order{
		{ID: 101, OrderNo: "ORD-101", UserID: 7, TotalAmountFen: 500, Status: model.OrderStatusPending},
	}, 1, nil
}

func (m *mockOrderServiceForRouter) GetOrderByID(context.Context, int64, int64) (*model.Order, []*model.OrderItem, error) {
	panic("unexpected GetOrderByID call")
}

func (m *mockOrderServiceForRouter) PayOrder(context.Context, int64, int64) error {
	panic("unexpected PayOrder call")
}

func (m *mockOrderServiceForRouter) FinishOrder(context.Context, int64, int64) error {
	panic("unexpected FinishOrder call")
}

func (m *mockOrderServiceForRouter) CancelOrder(context.Context, int64, int64) error {
	panic("unexpected CancelOrder call")
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
		{method: http.MethodGet, path: "/api/v1/admin/orders"},
	} {
		request := httptest.NewRequest(route.method, route.path, nil)
		request.Header.Set("Authorization", "Bearer "+token)
		recorder := httptest.NewRecorder()
		r.ServeHTTP(recorder, request)

		if recorder.Code != http.StatusForbidden {
			t.Fatalf("expected %s status %d, got %d", route.path, http.StatusForbidden, recorder.Code)
		}
	}
	if checker.calls != 4 {
		t.Fatalf("expected four role checks, got %d", checker.calls)
	}
}

func TestAdminOrdersRoute_AllowedForAdminAndAudited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokenManager, err := auth.NewTokenManager("0123456789abcdef0123456789abcdef", "test", time.Hour)
	if err != nil {
		t.Fatalf("new token manager: %v", err)
	}
	token, err := tokenManager.GenerateAccessToken(7, "admin-user")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	checker := &allowRoleChecker{}
	audit := &spyAuditRecorder{}
	orderSvc := &mockOrderServiceForRouter{}
	orderH := handler.NewOrderHandler(orderSvc)

	r := SetupRouters(nil, Handlers{
		Product:      &handler.ProductHandler{},
		Inventory:    &handler.InventoryHandler{},
		StockLog:     &handler.StockLogHandler{},
		Order:        orderH,
		Health:       &handler.HealthHandler{},
		User:         &handler.UserHandler{},
		OperationLog: &handler.OperationLogHandler{},
		Audit:        audit,
	}, tokenManager, checker)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/orders?page=1&page_size=10&status=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}
	if checker.calls != 1 {
		t.Fatalf("expected 1 role check, got %d", checker.calls)
	}
	if orderSvc.listAdminCalls != 1 {
		t.Fatalf("expected 1 ListAdminOrders call, got %d", orderSvc.listAdminCalls)
	}
	if orderSvc.lastReq.Status == nil || *orderSvc.lastReq.Status != 1 {
		t.Fatalf("expected status filter 1, got %v", orderSvc.lastReq.Status)
	}

	// 验证 OperationAudit 审计组件正确捕获本次管理员调用
	if len(audit.logs) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(audit.logs))
	}
	auditLog := audit.logs[0]
	if auditLog.UserID != 7 || auditLog.Username != "admin-user" || auditLog.Method != http.MethodGet {
		t.Fatalf("unexpected audit log: %+v", auditLog)
	}
	if auditLog.Path != "/api/v1/admin/orders" || auditLog.HTTPStatus != http.StatusOK {
		t.Fatalf("unexpected audit log path or status: %+v", auditLog)
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
