//go:build vips

package process

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/riverqueue/river"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/media"
)

const (
	failureReasonLimit  = 300
	failureWriteTimeout = 10 * time.Second
)

// Worker processes media.Process jobs. It retries with River backoff until the
// final attempt, then records the failure on the asset.
type Worker struct {
	river.WorkerDefaults[media.ProcessArgs]

	processor *Processor
}

// NewWorker builds the River worker for media processing jobs.
func NewWorker(processor *Processor) *Worker {
	return &Worker{processor: processor}
}

// Work implements river.Worker.
func (w *Worker) Work(ctx context.Context, job *river.Job[media.ProcessArgs]) error {
	err := w.processor.Process(ctx, job.Args.AssetID)
	if err == nil {
		return nil
	}

	if job.Attempt < job.MaxAttempts {
		w.processor.logger.WarnContext(ctx, "media processing attempt failed; River will retry",
			"asset_id", job.Args.AssetID, "attempt", job.Attempt, "max_attempts", job.MaxAttempts, "error", err)
		return err
	}

	reason := failureReason(err)
	failCtx := ctx
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		// A River job timeout has already expired; the failure must still be
		// recorded. Shutdown cancellations keep the original context so River
		// can retry the job later.
		detached, cancel := context.WithTimeout(context.WithoutCancel(ctx), failureWriteTimeout)
		defer cancel()
		failCtx = detached
	}
	if failErr := w.processor.Fail(failCtx, job.Args.AssetID, reason); failErr != nil {
		w.processor.logger.ErrorContext(ctx, "media processing failed and the asset status could not be updated",
			"asset_id", job.Args.AssetID, "error", failErr)
		return failErr
	}

	w.processor.logger.ErrorContext(ctx, "media processing failed permanently",
		"asset_id", job.Args.AssetID, "attempt", job.Attempt, "reason", reason)
	return nil
}

// failureReason keeps the stored reason short, single-line, and free of stack
// traces.
func failureReason(err error) string {
	if err == nil {
		return ""
	}

	text := strings.Join(strings.Fields(err.Error()), " ")
	if text == "" {
		text = "unknown media processing error"
	}
	if len(text) > failureReasonLimit {
		cut := failureReasonLimit
		for cut > 0 && !utf8.RuneStart(text[cut]) {
			cut--
		}
		text = text[:cut]
	}
	return text
}
