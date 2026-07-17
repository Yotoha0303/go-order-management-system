package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func Run() error {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	deps, err := InitDeps(logger)
	if err != nil {
		return err
	}

	server := NewHTTPServer(deps)
	workerCtx, stopWorkers := context.WithCancel(context.Background())
	var workerWG sync.WaitGroup
	startWorker := func(name string, run func(context.Context) error) {
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			if err := run(workerCtx); err != nil {
				logger.Error(name+" stopped", "error", err)
			}
		}()
	}
	startWorker("order timeout worker", deps.OrderTimeoutWorker.Run)
	if deps.InventoryWorker != nil {
		startWorker("inventory reconcile worker", deps.InventoryWorker.Run)
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info("server starting", "addr", server.Addr)
		serverErr <- server.ListenAndServe()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var runErr error
	select {
	case <-quit:
		logger.Info("shutdown signal received")
	case err := <-serverErr:
		runErr = fmt.Errorf("server stopped: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stopWorkers()

	shutdownErr := server.Shutdown(ctx)
	workerDone := make(chan struct{})
	go func() {
		workerWG.Wait()
		close(workerDone)
	}()
	select {
	case <-workerDone:
	case <-ctx.Done():
		logger.Warn("background worker shutdown timed out")
	}
	if runErr != nil {
		return runErr
	}
	return shutdownErr
}
