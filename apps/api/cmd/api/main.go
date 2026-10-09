package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"grafikarsa/api/internal/cache"
	"grafikarsa/api/internal/config"
	"grafikarsa/api/internal/db"
	apiphttp "grafikarsa/api/internal/http"
	"grafikarsa/api/internal/repo"
	"grafikarsa/api/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	ctx := context.Background()

	pool, err := db.NewPgxPool(ctx, cfg)
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	c := cache.New(cfg)
	defer c.Close()

	st := storage.New(cfg)
	app := apiphttp.NewApp(cfg, repo.New(pool), c, st)
	app.Tx = apiphttp.NewTxRunner(pool)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      apiphttp.NewRouter(apiphttp.Deps{Pool: pool, Cache: c, Storage: st}),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	_ = app

	go func() {
		slog.Info("api listen", "addr", ":8080", "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
