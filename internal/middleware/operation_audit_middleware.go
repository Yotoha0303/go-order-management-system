package middleware

import (
	"context"
	"strings"

	"go-order-management-system/internal/model"

	"github.com/gin-gonic/gin"
)

type OperationAuditRecorder interface {
	CreateOperationLog(ctx context.Context, log *model.OperationLog) error
}

func OperationAudit(recorder OperationAuditRecorder) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if recorder == nil {
			return
		}

		userID, _ := c.Get(UserIDKey)
		id, ok := userID.(int64)
		if !ok || id <= 0 {
			return
		}

		username, _ := c.Get(UsernameKey)
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}

		log := &model.OperationLog{
			UserID:     id,
			Username:   stringValue(username),
			Action:     buildOperationAction(c.Request.Method, route),
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			Route:      route,
			HTTPStatus: c.Writer.Status(),
			RequestID:  GetRequestID(c),
			ClientIP:   c.ClientIP(),
			UserAgent:  truncateString(c.Request.UserAgent(), 255),
		}
		_ = recorder.CreateOperationLog(context.WithoutCancel(c.Request.Context()), log)
	}
}

func buildOperationAction(method, route string) string {
	return strings.TrimSpace(method + " " + route)
}

func stringValue(value interface{}) string {
	text, _ := value.(string)
	return text
}

func truncateString(value string, maxLength int) string {
	if len(value) <= maxLength {
		return value
	}
	return value[:maxLength]
}
