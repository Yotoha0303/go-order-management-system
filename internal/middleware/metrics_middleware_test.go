package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-order-management-system/internal/observability"

	"github.com/gin-gonic/gin"
)

func TestHTTPMetricsMiddlewareRecordsRouteStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	metrics := observability.NewHTTPMetrics()
	router := gin.New()
	router.Use(HTTPMetrics(metrics))
	router.GET("/products/:id", func(c *gin.Context) {
		c.Status(http.StatusAccepted)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/products/1", nil))

	output := metrics.RenderPrometheus()
	want := `app_http_requests_total{method="GET",route="/products/:id",status="202"} 1`
	if !strings.Contains(output, want) {
		t.Fatalf("expected metrics output to contain %q, got:\n%s", want, output)
	}
}
