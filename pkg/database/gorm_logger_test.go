package database

import (
	"bytes"
	"context"
	"errors"
	"go-order-management-system/config"
	"log/slog"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func TestGORMLoggerTraceSlowQuery(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	gormLogger := newGORMLogger(config.MySQLConfig{
		SlowThreshold: time.Millisecond,
		LogLevel:      "warn",
	}, logger)

	gormLogger.Trace(context.Background(), time.Now().Add(-10*time.Millisecond), func() (string, int64) {
		return "SELECT * FROM orders", 3
	}, nil)

	output := buf.String()
	if !strings.Contains(output, "gorm slow query") {
		t.Fatalf("slow query log missing: %s", output)
	}
	if !strings.Contains(output, "SELECT * FROM orders") {
		t.Fatalf("sql missing: %s", output)
	}
}

func TestGORMLoggerTraceError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	gormLogger := newGORMLogger(config.MySQLConfig{LogLevel: "error"}, logger)

	gormLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "UPDATE orders SET status = 2", 0
	}, errors.New("deadlock found"))

	output := buf.String()
	if !strings.Contains(output, "gorm query error") {
		t.Fatalf("query error log missing: %s", output)
	}
	if !strings.Contains(output, "deadlock found") {
		t.Fatalf("error detail missing: %s", output)
	}
}

func TestGORMLoggerIgnoresRecordNotFoundAtErrorLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	gormLogger := newGORMLogger(config.MySQLConfig{LogLevel: "error"}, logger)

	gormLogger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "SELECT * FROM orders WHERE id = 404", 0
	}, gorm.ErrRecordNotFound)

	if output := buf.String(); output != "" {
		t.Fatalf("record not found should not be logged as error: %s", output)
	}
}

func TestParseGORMLogLevelDefaultsToWarn(t *testing.T) {
	if got := parseGORMLogLevel(""); got != gormlogger.Warn {
		t.Fatalf("empty log level=%v", got)
	}
	if got := parseGORMLogLevel("unknown"); got != gormlogger.Warn {
		t.Fatalf("unknown log level=%v", got)
	}
}
