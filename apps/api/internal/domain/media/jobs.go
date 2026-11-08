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

// ProcessWorker is the placeholder media processing worker. Real processing
// arrives in a later task; this worker succeeds immediately so jobs do not
// accumulate in retries.
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
