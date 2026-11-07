package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
)

// PingArgs is a no-op job used by tests and as a worker smoke test.
type PingArgs struct{}

// Kind implements river.JobArgs.
func (PingArgs) Kind() string { return "system.ping" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (PingArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueDefault, MaxAttempts: 3}
}

// PingWorker logs every ping job at debug level.
type PingWorker struct {
	river.WorkerDefaults[PingArgs]

	Logger *slog.Logger
}

// Work implements river.Worker.
func (w *PingWorker) Work(ctx context.Context, job *river.Job[PingArgs]) error {
	logger := w.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.DebugContext(ctx, "ping job executed", "job_id", job.ID)
	return nil
}
