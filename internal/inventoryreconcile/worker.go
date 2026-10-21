package inventoryreconcile

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"go-order-management-system/internal/service"
)

type Config struct {
	Interval time.Duration
	Timeout  time.Duration
	AutoFix  bool
}

type Reconciler interface {
	ReconcileRedisInventoryStock(ctx context.Context) (service.InventoryRedisReconcileReport, error)
}

type StockWriter interface {
	SetInventoryStock(ctx context.Context, productID int64, quantity int64)
}

type MetricsRecorder interface {
	RecordInventoryReconcileDiff(status string)
	RecordInventoryReconcileAutoFix(result string)
}

type Worker struct {
	config      Config
	reconciler  Reconciler
	stockWriter StockWriter
	metrics     MetricsRecorder
	logger      *slog.Logger
}

func NewWorker(config Config, reconciler Reconciler, logger *slog.Logger) (*Worker, error) {
	return NewWorkerWithStockWriterAndMetrics(config, reconciler, nil, nil, logger)
}

func NewWorkerWithStockWriterAndMetrics(
	config Config,
	reconciler Reconciler,
	stockWriter StockWriter,
	metrics MetricsRecorder,
	logger *slog.Logger,
) (*Worker, error) {
	if config.Interval <= 0 || config.Timeout <= 0 || config.Timeout > config.Interval {
		return nil, errors.New("create inventory reconcile worker: invalid interval or timeout")
	}
	if reconciler == nil {
		return nil, errors.New("create inventory reconcile worker: reconciler is required")
	}
	if logger == nil {
		return nil, errors.New("create inventory reconcile worker: logger is required")
	}
	if stockWriter == nil {
		if sw, ok := reconciler.(StockWriter); ok {
			stockWriter = sw
		}
	}
	return &Worker{
		config:      config,
		reconciler:  reconciler,
		stockWriter: stockWriter,
		metrics:     metrics,
		logger:      logger,
	}, nil
}

func (w *Worker) Run(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		_ = w.RunOnce(ctx)
		if !waitFor(ctx, w.config.Interval) {
			return nil
		}
	}
}

func (w *Worker) RunOnce(ctx context.Context) error {
	reconcileCtx, cancel := context.WithTimeout(ctx, w.config.Timeout)
	defer cancel()

	report, err := w.reconciler.ReconcileRedisInventoryStock(reconcileCtx)
	if err != nil {
		if ctx.Err() == nil {
			w.logger.Error("inventory Redis reconcile failed", "error", err)
		}
		return err
	}

	if report.DiffCount > 0 {
		for _, item := range report.Items {
			if w.metrics != nil {
				w.metrics.RecordInventoryReconcileDiff(item.Status)
			}
		}

		if w.config.AutoFix {
			if w.stockWriter == nil {
				w.logger.Error("inventory reconcile autofix failed: stock writer unavailable",
					"diff_count", report.DiffCount,
				)
				if w.metrics != nil {
					w.metrics.RecordInventoryReconcileAutoFix("failed")
				}
				return nil
			}

			for _, item := range report.Items {
				w.stockWriter.SetInventoryStock(reconcileCtx, item.ProductID, item.MySQLQuantity)
			}

			w.logger.Info("inventory reconcile autofix applied",
				"checked_count", report.CheckedCount,
				"diff_count", report.DiffCount,
			)
			if w.metrics != nil {
				w.metrics.RecordInventoryReconcileAutoFix("success")
			}
			return nil
		}

		w.logger.Warn("inventory Redis reconcile found differences",
			"checked_count", report.CheckedCount,
			"diff_count", report.DiffCount,
		)
		if w.metrics != nil {
			w.metrics.RecordInventoryReconcileAutoFix("disabled")
		}
		return nil
	}

	w.logger.Info("inventory Redis reconcile finished",
		"checked_count", report.CheckedCount,
		"diff_count", report.DiffCount,
	)
	return nil
}

func waitFor(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
