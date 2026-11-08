package content

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/jobs"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	purgeBatchSize = 100
	purgeInterval  = 24 * time.Hour
)

// PurgeArgs asks the worker to permanently remove trashed contents whose
// retention window has expired.
type PurgeArgs struct{}

// Kind implements river.JobArgs.
func (PurgeArgs) Kind() string { return "content.purge" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (PurgeArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueBulk, MaxAttempts: 3}
}

// PurgePeriodicJob returns the daily purge schedule registered by the worker.
func PurgePeriodicJob() *river.PeriodicJob {
	return river.NewPeriodicJob(
		river.PeriodicInterval(purgeInterval),
		func() (river.JobArgs, *river.InsertOpts) {
			return PurgeArgs{}, nil
		},
		&river.PeriodicJobOpts{ID: "content.purge", RunOnStart: false},
	)
}

// PurgeWorker runs the purge job.
type PurgeWorker struct {
	river.WorkerDefaults[PurgeArgs]

	Store  *store.Store
	Logger *slog.Logger
}

// NewPurgeWorker builds the River worker for content.purge jobs.
func NewPurgeWorker(st *store.Store, logger *slog.Logger) *PurgeWorker {
	return &PurgeWorker{Store: st, Logger: logger}
}

// Work implements river.Worker.
func (w *PurgeWorker) Work(ctx context.Context, job *river.Job[PurgeArgs]) error {
	logger := w.Logger
	if logger == nil {
		logger = slog.Default()
	}

	deleted, err := Purge(ctx, w.Store)
	if err != nil {
		return err
	}
	logger.InfoContext(ctx, "content purge completed", "job_id", job.ID, "deleted", deleted)
	return nil
}

// Purge deletes expired trashed contents in batches so a large backlog does not
// hold one long transaction. CASCADE removes drafts, versions, authors, and
// every other child row.
func Purge(ctx context.Context, st *store.Store) (int64, error) {
	if st == nil {
		return 0, fmt.Errorf("content: purge store is nil")
	}

	var total int64
	for {
		deleted, err := st.DeleteExpiredDeletedContents(ctx, purgeBatchSize)
		if err != nil {
			return total, fmt.Errorf("content: purge deleted contents: %w", err)
		}
		total += deleted
		if deleted < purgeBatchSize {
			return total, nil
		}
	}
}
