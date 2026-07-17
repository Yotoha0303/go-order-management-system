package handler

import (
	"net/http"

	"go-order-management-system/internal/observability"

	"github.com/gin-gonic/gin"
)

type MetricsHandler struct {
	metrics *observability.Metrics
}

func NewMetricsHandler(metrics *observability.Metrics) *MetricsHandler {
	return &MetricsHandler{metrics: metrics}
}

func (h *MetricsHandler) Prometheus(c *gin.Context) {
	if h == nil || h.metrics == nil {
		c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", nil)
		return
	}
	c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(h.metrics.RenderPrometheus()))
}
