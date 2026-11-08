//go:build !vips

package main

import (
	"log/slog"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/jobs"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/storage"
	dbstore "github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

// setupMediaProcessing is the build without libvips (local development, CI
// without -tags vips). No media worker is registered, so media.process jobs
// stay queued and retry until a worker built with `-tags vips` handles them.
func setupMediaProcessing(_ *jobs.Registry, _ *dbstore.Store, _ *storage.Client, log *slog.Logger) (func(), error) {
	log.Warn("media processing disabled: worker built without -tags vips (libvips)")
	return func() {}, nil
}
