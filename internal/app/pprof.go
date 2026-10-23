package app

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"time"
)

// NewPprofServer builds the optional profiling listener, or returns nil when
// profiling is disabled.
//
// Returning nil rather than an inert server keeps the "is it on?" question
// answerable from the value itself: a caller cannot accidentally start a
// listener that was never configured.
//
// The mux is built from the explicit pprof.Handler registrations rather than
// importing net/http/pprof for its side effect on http.DefaultServeMux. That
// side effect is exactly what to avoid here - it would also publish these
// endpoints on any other server in the process that falls back to the default
// mux, including the business one.
func NewPprofServer(deps *Deps, logger *slog.Logger) *http.Server {
	cfg := deps.Config.Pprof
	if !cfg.Enabled {
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	logger.Info("pprof listener starting",
		"addr", cfg.Addr,
		"note", "loopback only; reach a remote instance through an SSH tunnel")

	return &http.Server{
		Addr:    cfg.Addr,
		Handler: mux,
		// Deliberately no ReadTimeout: /debug/pprof/profile?seconds=30 is a
		// single request that is idle for its whole duration by design, so a
		// read timeout would abort the very capture the endpoint exists for.
		//
		// WriteTimeout is set instead and is what actually bounds a runaway
		// capture. It must stay above the longest sampling window an operator
		// is expected to request.
		WriteTimeout:      2 * time.Minute,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

// RunPprofServer starts the profiling listener and returns a stop function.
//
// The listener is best-effort by design: failing to bind it is logged but must
// not take the service down. Profiling is a diagnostic aid, and refusing to
// serve business traffic because a debug port is occupied would invert the
// priority.
func RunPprofServer(server *http.Server, logger *slog.Logger) func() {
	if server == nil {
		return func() {}
	}

	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	go func() {
		if err := <-errCh; err != nil {
			logger.Error("pprof listener stopped", "addr", server.Addr, "error", err)
		}
	}()

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Warn("pprof shutdown error", "error", err)
		}
	}
}
