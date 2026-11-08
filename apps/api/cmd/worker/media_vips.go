//go:build vips

package main

import (
	"log/slog"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/media/process"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/jobs"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/storage"
	dbstore "github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

// setupMediaProcessing starts libvips and registers the real media worker.
// Only binaries built with `-tags vips` (the worker image) include it.
func setupMediaProcessing(registry *jobs.Registry, st *dbstore.Store, s3 *storage.Client, log *slog.Logger) (func(), error) {
	if err := process.Startup(); err != nil {
		return nil, err
	}
	jobs.Register(registry, process.NewWorker(process.NewProcessor(st, s3, log)))
	log.Info("media processing enabled (libvips)")
	return process.Shutdown, nil
}
