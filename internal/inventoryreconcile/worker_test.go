package inventoryreconcile

import (
	"bytes"
	"context"
	"errors"
	"go-order-management-system/internal/service"
	"log/slog"
	"strings"
	"testing"
	"time"
)

type stubReconciler struct {
	report service.InventoryRedisReconcileReport
	err    error
	calls  int
}

func (s *stubReconciler) ReconcileRedisInventoryStock(context.Context) (service.InventoryRedisReconcileReport, error) {
	s.calls++
	return s.report, s.err
}

func TestWorkerRunOnceLogsDifferences(t *testing.T) {
	var buf bytes.Buffer
	reconciler := &stubReconciler{report: service.InventoryRedisReconcileReport{
		CheckedCount: 3,
		DiffCount:    1,
	}}
	worker, err := NewWorker(Config{Interval: time.Minute, Timeout: time.Second}, reconciler, slog.New(slog.NewTextHandler(&buf, nil)))
	if err != nil {
		t.Fatalf("new worker: %v", err)
	}

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once: %v", err)
	}
	if reconciler.calls != 1 {
		t.Fatalf("calls=%d", reconciler.calls)
	}
	if output := buf.String(); !strings.Contains(output, "inventory Redis reconcile found differences") {
		t.Fatalf("difference log missing: %s", output)
	}
}

func TestWorkerRunOnceReturnsAndLogsError(t *testing.T) {
	var buf bytes.Buffer
	reconciler := &stubReconciler{err: errors.New("redis unavailable")}
	worker, err := NewWorker(Config{Interval: time.Minute, Timeout: time.Second}, reconciler, slog.New(slog.NewTextHandler(&buf, nil)))
	if err != nil {
		t.Fatalf("new worker: %v", err)
	}

	if err := worker.RunOnce(context.Background()); !errors.Is(err, reconciler.err) {
		t.Fatalf("run once error=%v", err)
	}
	if output := buf.String(); !strings.Contains(output, "inventory Redis reconcile failed") {
		t.Fatalf("error log missing: %s", output)
	}
}

func TestNewWorkerRejectsInvalidConfig(t *testing.T) {
	_, err := NewWorker(Config{Interval: time.Second, Timeout: 2 * time.Second}, &stubReconciler{}, slog.Default())
	if err == nil {
		t.Fatal("expected invalid config error")
	}
}
