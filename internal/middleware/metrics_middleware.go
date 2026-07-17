package middleware

import (
	"time"

	"go-order-management-system/internal/observability"

	"github.com/gin-gonic/gin"
)

func HTTPMetrics(metrics *observability.HTTPMetrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if metrics == nil {
			return
		}
		metrics.Record(c.Request.Method, c.FullPath(), c.Writer.Status(), time.Since(start))
	}
}
