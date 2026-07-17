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
}

type Reconciler interface {
	ReconcileRedisInventoryStock(ctx context.Context) (service.InventoryRedisReconcileReport, error)
}

type Worker struct {
	config     Config
	reconciler Reconciler
	logger     *slog.Logger
}

func NewWorker(config Config, reconciler Reconciler, logger *slog.Logger) (*Worker, error) {
	if config.Interval <= 0 || config.Timeout <= 0 || config.Timeout > config.Interval {
		return nil, errors.New("create inventory reconcile worker: invalid interval or timeout")
	}
	if reconciler == nil {
		return nil, errors.New("create inventory reconcile worker: reconciler is required")
	}
	if logger == nil {
		return nil, errors.New("create inventory reconcile worker: logger is required")
	}
	return &Worker{
		config:     config,
		reconciler: reconciler,
		logger:     logger,
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
		w.logger.Warn("inventory Redis reconcile found differences",
			"checked_count", report.CheckedCount,
			"diff_count", report.DiffCount,
		)
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
