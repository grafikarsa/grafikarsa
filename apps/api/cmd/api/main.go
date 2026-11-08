// Command api runs the Grafikarsa HTTP API.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/academic"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/audit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/profiles"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/roles"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/users"
	apirouter "github.com/grafikarsa/grafikarsa/apps/api/internal/http"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/health"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/jobs"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/cache"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/config"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/db"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/logger"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/password"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/secretbox"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/storage"
	dbstore "github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const startupTimeout = 10 * time.Second

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}

	log := logger.New(cfg.Env, cfg.LogLevel)

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), startupTimeout)
	defer cancelStartup()

	pool, err := db.Open(startupCtx, cfg.DatabaseURL)
	if err != nil {
		log.Error("postgres connection failed", "error", err)
		os.Exit(1)
	}

	rdb, err := cache.Open(startupCtx, cfg.ValkeyURL)
	if err != nil {
		log.Error("valkey connection failed", "error", err)
		pool.Close()
		os.Exit(1)
	}

	store, err := storage.New(startupCtx,
		cfg.S3Endpoint,
		cfg.S3Region,
		cfg.S3AccessKey,
		cfg.S3SecretKey,
		cfg.S3BucketPrivate,
		cfg.S3BucketPublic,
	)
	if err != nil {
		log.Error("storage client failed", "error", err)
		pool.Close()
		_ = rdb.Close()
		os.Exit(1)
	}

	if !cfg.IsProduction() {
		if err := store.EnsureBuckets(startupCtx); err != nil {
			log.Error("ensure buckets failed", "error", err)
			pool.Close()
			_ = rdb.Close()
			os.Exit(1)
		}
	}
	cancelStartup()

	checkers := []health.Checker{
		health.CheckerFunc{N: "postgres", F: pool.Ping},
		health.CheckerFunc{N: "valkey", F: func(ctx context.Context) error {
			return rdb.Ping(ctx).Err()
		}},
		health.CheckerFunc{N: "storage", F: store.Ping},
	}

	limiter := ratelimit.New(rdb)
	dataStore := dbstore.NewStore(pool)
	hasher := password.NewHasher(password.DefaultParams, 0)
	box, err := secretbox.New([]byte(cfg.SecretKey))
	if err != nil {
		log.Error("secretbox failed", "error", err)
		pool.Close()
		_ = rdb.Close()
		os.Exit(1)
	}
	authService, err := auth.NewService(
		dataStore,
		rdb,
		hasher,
		limiter,
		auth.Config{RecoverySecret: []byte(cfg.SecretKey)},
		log,
	)
	if err != nil {
		log.Error("auth service failed", "error", err)
		pool.Close()
		_ = rdb.Close()
		os.Exit(1)
	}

	academicService := academic.NewService(dataStore)
	usersService := users.NewService(dataStore, authService, hasher, box)
	profilesService := profiles.NewService(dataStore, authService)
	rolesService := roles.NewService(dataStore, authService)
	auditService := audit.NewService(dataStore)

	// The API only inserts jobs. Workers live in cmd/worker; the insert client
	// still needs the shared registry so job argument types stay consistent.
	jobRegistry := jobs.NewRegistry()
	// Register the job args type so the insert client accepts it. The API
	// never starts this worker; execution happens in cmd/worker.
	jobs.Register(jobRegistry, &users.ImportCommitWorker{})
	insertClient, err := jobs.NewInsertClient(pool, jobRegistry, log)
	if err != nil {
		log.Error("river insert client failed", "error", err)
		pool.Close()
		_ = rdb.Close()
		os.Exit(1)
	}
	importer := users.NewImporter(usersService, insertClient)

	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: apirouter.NewRouter(apirouter.RouterDeps{
			Logger:         log,
			Checkers:       checkers,
			Auth:           authService,
			Academic:       academicService,
			Users:          usersService,
			Profiles:       profilesService,
			Imports:        importer,
			Roles:          rolesService,
			Audit:          auditService,
			Limiter:        limiter,
			AllowedOrigins: cfg.AllowedOrigins,
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	log.Info("server started", "addr", cfg.HTTPAddr)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	exitCode := 0
	select {
	case err := <-serveErr:
		log.Error("server failed", "error", err)
		exitCode = 1
	case <-signalCtx.Done():
		log.Info("shutdown signal received")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown failed", "error", err)
		exitCode = 1
	}

	pool.Close()
	if err := rdb.Close(); err != nil {
		log.Warn("valkey close failed", "error", err)
	}

	log.Info("server stopped")

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
