package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-order-management-system/internal/model"

	"github.com/gin-gonic/gin"
)

type fakeOperationAuditRecorder struct {
	log   *model.OperationLog
	calls int
}

func (r *fakeOperationAuditRecorder) CreateOperationLog(_ context.Context, log *model.OperationLog) error {
	r.calls++
	copied := *log
	r.log = &copied
	return nil
}

func TestOperationAuditRecordsAdminOperation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := &fakeOperationAuditRecorder{}
	router := gin.New()
	router.Use(RequestID())
	router.Use(func(c *gin.Context) {
		c.Set(UserIDKey, int64(42))
		c.Set(UsernameKey, "admin-user")
		c.Next()
	})
	router.Use(OperationAudit(recorder))
	router.POST("/api/v1/products/:id/on-sale", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/products/9/on-sale", nil)
	request.Header.Set(RequestIDHeader, "11111111-1111-1111-1111-111111111111")
	request.Header.Set("User-Agent", "audit-test")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if recorder.calls != 1 {
		t.Fatalf("expected one operation audit record, got %d", recorder.calls)
	}
	if recorder.log == nil {
		t.Fatal("expected operation audit log")
	}
	if recorder.log.UserID != 42 || recorder.log.Username != "admin-user" {
		t.Fatalf("unexpected actor: %+v", recorder.log)
	}
	if recorder.log.Action != "POST /api/v1/products/:id/on-sale" {
		t.Fatalf("unexpected action: %q", recorder.log.Action)
	}
	if recorder.log.Path != "/api/v1/products/9/on-sale" || recorder.log.Route != "/api/v1/products/:id/on-sale" {
		t.Fatalf("unexpected path/route: %+v", recorder.log)
	}
	if recorder.log.HTTPStatus != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.log.HTTPStatus)
	}
	if recorder.log.RequestID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected request id: %q", recorder.log.RequestID)
	}
}

func TestOperationAuditSkipsMissingUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := &fakeOperationAuditRecorder{}
	router := gin.New()
	router.Use(OperationAudit(recorder))
	router.GET("/admin", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin", nil))

	if recorder.calls != 0 {
		t.Fatalf("expected no operation audit record, got %d", recorder.calls)
	}
}
