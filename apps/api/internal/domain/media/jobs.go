package media

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/riverqueue/river"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/jobs"
)

// ProcessArgs asks the worker to process one uploaded asset (strip metadata,
// build variants, and later publish it).
type ProcessArgs struct {
	AssetID uuid.UUID `json:"assetId"`
}

// Kind implements river.JobArgs.
func (ProcessArgs) Kind() string { return "media.process" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (ProcessArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueMedia, MaxAttempts: 5}
}

// ProcessWorker is a no-op worker used by the API's insert-only River client.
// The API must stay CGO_ENABLED=0 and must not import the libvips processing
// package, so the real worker lives in internal/domain/media/process and is
// registered by cmd/worker.
type ProcessWorker struct {
	river.WorkerDefaults[ProcessArgs]

	Logger *slog.Logger
}

// Work implements river.Worker.
func (w *ProcessWorker) Work(ctx context.Context, job *river.Job[ProcessArgs]) error {
	logger := w.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.InfoContext(ctx, "media processing not implemented yet",
		"job_id", job.ID, "asset_id", job.Args.AssetID)
	return nil
}
