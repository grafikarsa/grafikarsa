package content

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/jobs"
)

// ExpireInvitesKind is the River job kind of the invitation expiry sweep.
const ExpireInvitesKind = "content.expire_invites"

// expireInvitesInterval is how often the worker expires stale invitations.
const expireInvitesInterval = time.Hour

// ExpireInvitesArgs is the payload of the periodic expiry job. It carries no
// data because the sweep covers every content.
type ExpireInvitesArgs struct{}

// Kind implements river.JobArgs.
func (ExpireInvitesArgs) Kind() string { return ExpireInvitesKind }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (ExpireInvitesArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault, MaxAttempts: 3}
}

// ExpireInvitesPeriodicJob returns the hourly periodic job that marks pending
// invitations as expired.
func ExpireInvitesPeriodicJob() *river.PeriodicJob {
	return river.NewPeriodicJob(
		river.PeriodicInterval(expireInvitesInterval),
		func() (river.JobArgs, *river.InsertOpts) {
			return ExpireInvitesArgs{}, nil
		},
		&river.PeriodicJobOpts{ID: ExpireInvitesKind, RunOnStart: true},
	)
}

// ExpireInvitesWorker runs the invitation expiry sweep in the worker process.
type ExpireInvitesWorker struct {
	river.WorkerDefaults[ExpireInvitesArgs]

	Service *Service
	Logger  *slog.Logger
}

// Work implements river.Worker.
func (w *ExpireInvitesWorker) Work(ctx context.Context, _ *river.Job[ExpireInvitesArgs]) error {
	if w.Service == nil {
		return fmt.Errorf("content: expire invites worker is not configured")
	}

	expired, err := w.Service.ExpireInvitations(ctx)
	if err != nil {
		return err
	}
	if w.Logger != nil {
		w.Logger.InfoContext(ctx, "expired collaboration invitations", "count", expired)
	}
	return nil
}

// ExpireInvitations marks every pending invitation whose deadline has passed
// as expired and returns how many rows changed.
func (s *Service) ExpireInvitations(ctx context.Context) (int64, error) {
	now := s.now().UTC()
	expired, err := s.st.ExpirePendingInvitations(ctx, &now)
	if err != nil {
		return 0, fmt.Errorf("content: expire invitations: %w", err)
	}
	return expired, nil
}
