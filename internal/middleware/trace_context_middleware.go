package middleware

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	TraceparentHeader = "traceparent"
	TraceKeyID        = "trace_id"
	SpanKeyID         = "span_id"
)

type traceIDContextKey struct{}
type spanIDContextKey struct{}

func TraceContext() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID, _ := parseTraceparent(c.Request.Header.Get(TraceparentHeader))
		if traceID == "" {
			traceID = randomHex(16)
		}
		spanID := randomHex(8)

		c.Set(TraceKeyID, traceID)
		c.Set(SpanKeyID, spanID)

		ctx := context.WithValue(c.Request.Context(), traceIDContextKey{}, traceID)
		ctx = context.WithValue(ctx, spanIDContextKey{}, spanID)
		c.Request = c.Request.WithContext(ctx)

		c.Header(TraceparentHeader, formatTraceparent(traceID, spanID))
		c.Next()
	}
}

func parseTraceparent(header string) (string, string) {
	parts := strings.Split(strings.TrimSpace(header), "-")
	if len(parts) != 4 {
		return "", ""
	}
	if len(parts[0]) != 2 || len(parts[1]) != 32 || len(parts[2]) != 16 || len(parts[3]) != 2 {
		return "", ""
	}
	if !isLowerHex(parts[0]) || !isLowerHex(parts[1]) || !isLowerHex(parts[2]) || !isLowerHex(parts[3]) {
		return "", ""
	}
	if isAllZero(parts[1]) || isAllZero(parts[2]) {
		return "", ""
	}
	return parts[1], parts[2]
}

func formatTraceparent(traceID, spanID string) string {
	return "00-" + traceID + "-" + spanID + "-01"
}

func randomHex(bytesLen int) string {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		sum := sha256.Sum256([]byte(strconv.FormatInt(time.Now().UnixNano(), 10)))
		return hex.EncodeToString(sum[:])[:bytesLen*2]
	}
	return hex.EncodeToString(buf)
}

func isLowerHex(value string) bool {
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}

func isAllZero(value string) bool {
	for _, ch := range value {
		if ch != '0' {
			return false
		}
	}
	return true
}

func GetTraceID(c *gin.Context) string {
	value, exists := c.Get(TraceKeyID)
	if !exists {
		return ""
	}
	traceID, _ := value.(string)
	return traceID
}

func GetSpanID(c *gin.Context) string {
	value, exists := c.Get(SpanKeyID)
	if !exists {
		return ""
	}
	spanID, _ := value.(string)
	return spanID
}

func TraceIDFromContext(ctx context.Context) string {
	traceID, _ := ctx.Value(traceIDContextKey{}).(string)
	return traceID
}

func SpanIDFromContext(ctx context.Context) string {
	spanID, _ := ctx.Value(spanIDContextKey{}).(string)
	return spanID
}

func ensureTraceContext(r *http.Request) (string, string) {
	traceID, _ := parseTraceparent(r.Header.Get(TraceparentHeader))
	if traceID == "" {
		traceID = randomHex(16)
	}
	spanID := randomHex(8)
	r.Header.Set(TraceparentHeader, formatTraceparent(traceID, spanID))
	return traceID, spanID
}
