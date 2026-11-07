// Command worker runs the Grafikarsa River job worker.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/jobs"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/config"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/db"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/logger"
)

const (
	startupTimeout    = 10 * time.Second
	hardStopTimeout   = 5 * time.Second
	defaultHealthAddr = ":8092"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Env, cfg.LogLevel)

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), startupTimeout)
	pool, err := db.Open(startupCtx, cfg.DatabaseURL)
	cancelStartup()
	if err != nil {
		log.Error("postgres connection failed", "error", err)
		os.Exit(1)
	}

	registry := jobs.NewRegistry()
	jobs.Register(registry, &jobs.PingWorker{Logger: log})

	client, err := jobs.NewWorkerClient(pool, registry, log)
	if err != nil {
		log.Error("river client failed", "error", err)
		pool.Close()
		os.Exit(1)
	}

	healthAddr := os.Getenv("GK_WORKER_HEALTH_ADDR")
	if healthAddr == "" {
		healthAddr = defaultHealthAddr
	}

	listener, err := net.Listen("tcp", healthAddr)
	if err != nil {
		log.Error("worker health listener failed", "error", err, "addr", healthAddr)
		pool.Close()
		os.Exit(1)
	}

	healthSrv := &http.Server{
		Handler:           healthHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	healthErr := make(chan error, 1)
	go func() {
		if err := healthSrv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			healthErr <- err
		}
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := client.Start(context.Background()); err != nil {
		log.Error("worker start failed", "error", err)
		_ = healthSrv.Close()
		pool.Close()
		os.Exit(1)
	}
	log.Info("worker started", "health_addr", healthAddr)

	exitCode := 0
	select {
	case err := <-healthErr:
		log.Error("worker health server failed", "error", err)
		exitCode = 1
	case <-signalCtx.Done():
		log.Info("shutdown signal received")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancelShutdown()

	if err := client.Stop(shutdownCtx); err != nil {
		log.Error("worker graceful stop failed", "error", err)
		hardCtx, cancelHard := context.WithTimeout(context.Background(), hardStopTimeout)
		if err := client.StopAndCancel(hardCtx); err != nil {
			log.Error("worker hard stop failed", "error", err)
		}
		cancelHard()
		exitCode = 1
	}

	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		log.Warn("worker health server shutdown failed", "error", err)
	}

	pool.Close()
	log.Info("worker stopped")

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

// healthHandler serves the worker liveness endpoint used by Compose and
// uptime checks.
func healthHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	return mux
}
