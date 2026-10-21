package inventoryreconcile

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"go-order-management-system/internal/service"
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

type fakeStockWriter struct {
	setCalls []productStockPair
}

type productStockPair struct {
	productID int64
	quantity  int64
}

func (f *fakeStockWriter) SetInventoryStock(_ context.Context, productID int64, quantity int64) {
	f.setCalls = append(f.setCalls, productStockPair{productID: productID, quantity: quantity})
}

type fakeMetricsRecorder struct {
	diffStatuses []string
	autoFixResults []string
}

func (f *fakeMetricsRecorder) RecordInventoryReconcileDiff(status string) {
	f.diffStatuses = append(f.diffStatuses, status)
}

func (f *fakeMetricsRecorder) RecordInventoryReconcileAutoFix(result string) {
	f.autoFixResults = append(f.autoFixResults, result)
}

func TestWorkerRunOnceLogsDifferences(t *testing.T) {
	var buf bytes.Buffer
	reconciler := &stubReconciler{report: service.InventoryRedisReconcileReport{
		CheckedCount: 3,
		DiffCount:    1,
		Items: []service.InventoryRedisReconcileItem{
			{ProductID: 1, MySQLQuantity: 10, Status: "missing"},
		},
	}}
	metrics := &fakeMetricsRecorder{}
	worker, err := NewWorkerWithStockWriterAndMetrics(
		Config{Interval: time.Minute, Timeout: time.Second, AutoFix: false},
		reconciler,
		nil,
		metrics,
		slog.New(slog.NewTextHandler(&buf, nil)),
	)
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
	if len(metrics.diffStatuses) != 1 || metrics.diffStatuses[0] != "missing" {
		t.Errorf("expected diff metric 'missing', got: %v", metrics.diffStatuses)
	}
	if len(metrics.autoFixResults) != 1 || metrics.autoFixResults[0] != "disabled" {
		t.Errorf("expected autofix metric 'disabled', got: %v", metrics.autoFixResults)
	}
}

func TestWorkerRunOnce_AutoFixEnabled_RestoresRedisAndRecordsMetrics(t *testing.T) {
	var buf bytes.Buffer
	reconciler := &stubReconciler{report: service.InventoryRedisReconcileReport{
		CheckedCount: 5,
		DiffCount:    2,
		Items: []service.InventoryRedisReconcileItem{
			{ProductID: 101, MySQLQuantity: 20, Status: "missing"},
			{ProductID: 102, MySQLQuantity: 15, Status: "mismatch"},
		},
	}}
	writer := &fakeStockWriter{}
	metrics := &fakeMetricsRecorder{}

	worker, err := NewWorkerWithStockWriterAndMetrics(
		Config{Interval: time.Minute, Timeout: time.Second, AutoFix: true},
		reconciler,
		writer,
		metrics,
		slog.New(slog.NewTextHandler(&buf, nil)),
	)
	if err != nil {
		t.Fatalf("new worker: %v", err)
	}

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once: %v", err)
	}

	// 验证 AutoFix 是否针对差异项依次调用了 SetInventoryStock
	if len(writer.setCalls) != 2 {
		t.Fatalf("expected 2 SetInventoryStock calls, got: %d", len(writer.setCalls))
	}
	if writer.setCalls[0].productID != 101 || writer.setCalls[0].quantity != 20 {
		t.Errorf("unexpected call 0: %+v", writer.setCalls[0])
	}
	if writer.setCalls[1].productID != 102 || writer.setCalls[1].quantity != 15 {
		t.Errorf("unexpected call 1: %+v", writer.setCalls[1])
	}

	// 验证指标上报
	if len(metrics.diffStatuses) != 2 {
		t.Errorf("expected 2 diff metric entries, got: %v", metrics.diffStatuses)
	}
	if len(metrics.autoFixResults) != 1 || metrics.autoFixResults[0] != "success" {
		t.Errorf("expected autofix metric 'success', got: %v", metrics.autoFixResults)
	}

	// 验证日志
	output := buf.String()
	if !strings.Contains(output, "inventory reconcile autofix applied") {
		t.Errorf("expected 'autofix applied' log, got: %s", output)
	}
}

func TestWorkerRunOnce_AutoFixEnabled_NoStockWriter_LogsErrorAndRecordsFailedMetric(t *testing.T) {
	var buf bytes.Buffer
	reconciler := &stubReconciler{report: service.InventoryRedisReconcileReport{
		CheckedCount: 3,
		DiffCount:    1,
		Items: []service.InventoryRedisReconcileItem{
			{ProductID: 201, MySQLQuantity: 10, Status: "mismatch"},
		},
	}}
	metrics := &fakeMetricsRecorder{}

	worker, err := NewWorkerWithStockWriterAndMetrics(
		Config{Interval: time.Minute, Timeout: time.Second, AutoFix: true},
		reconciler,
		nil, // 无 stockWriter
		metrics,
		slog.New(slog.NewTextHandler(&buf, nil)),
	)
	if err != nil {
		t.Fatalf("new worker: %v", err)
	}

	if err := worker.RunOnce(context.Background()); err != nil {
		t.Fatalf("run once: %v", err)
	}

	if len(metrics.autoFixResults) != 1 || metrics.autoFixResults[0] != "failed" {
		t.Errorf("expected autofix metric 'failed', got: %v", metrics.autoFixResults)
	}
	output := buf.String()
	if !strings.Contains(output, "inventory reconcile autofix failed: stock writer unavailable") {
		t.Errorf("expected error log, got: %s", output)
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
