package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTraceContextUsesIncomingTraceIDAndCreatesServerSpan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const incomingTraceID = "4bf92f3577b34da6a3ce929d0e0e4736"
	const incomingSpanID = "00f067aa0ba902b7"
	incomingTraceparent := formatTraceparent(incomingTraceID, incomingSpanID)

	router := gin.New()
	router.Use(TraceContext())
	router.GET("/ok", func(c *gin.Context) {
		if got := GetTraceID(c); got != incomingTraceID {
			t.Fatalf("trace id=%q", got)
		}
		if got := TraceIDFromContext(c.Request.Context()); got != incomingTraceID {
			t.Fatalf("context trace id=%q", got)
		}
		if got := GetSpanID(c); got == "" || got == incomingSpanID {
			t.Fatalf("server span id=%q", got)
		}
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ok", nil)
	request.Header.Set(TraceparentHeader, incomingTraceparent)
	router.ServeHTTP(recorder, request)

	responseTraceparent := recorder.Header().Get(TraceparentHeader)
	traceID, spanID := parseTraceparent(responseTraceparent)
	if traceID != incomingTraceID {
		t.Fatalf("response trace id=%q header=%q", traceID, responseTraceparent)
	}
	if spanID == "" || spanID == incomingSpanID {
		t.Fatalf("response span id=%q header=%q", spanID, responseTraceparent)
	}
}

func TestTraceContextRegeneratesInvalidTraceparent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(TraceContext())
	router.GET("/ok", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ok", nil)
	request.Header.Set(TraceparentHeader, "invalid")
	router.ServeHTTP(recorder, request)

	traceID, spanID := parseTraceparent(recorder.Header().Get(TraceparentHeader))
	if traceID == "" || spanID == "" {
		t.Fatalf("expected generated traceparent, got %q", recorder.Header().Get(TraceparentHeader))
	}
}

func TestAccessLogIncludesTraceContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	router := gin.New()
	router.Use(RequestID(), TraceContext(), AccessLog(logger))
	router.GET("/ok", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ok", nil)
	request.Header.Set(TraceparentHeader, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	router.ServeHTTP(recorder, request)

	output := buf.String()
	if !strings.Contains(output, `trace_id=4bf92f3577b34da6a3ce929d0e0e4736`) {
		t.Fatalf("trace id missing from access log: %s", output)
	}
	if !strings.Contains(output, "span_id=") {
		t.Fatalf("span id missing from access log: %s", output)
	}
}
