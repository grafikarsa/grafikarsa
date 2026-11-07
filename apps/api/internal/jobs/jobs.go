// Package jobs provides the River job queue foundation shared by the API and
// the worker process.
//
// Job argument types live next to the domain that owns them (for example
// internal/domain/users/jobs.go), implement Kind returning "<domain>.<action>"
// (for example "users.send_welcome_email"), and implement InsertOpts to choose
// the queue and the maximum number of attempts.
//
// Jobs are inserted in the same database transaction that changes the data,
// which gives an outbox-like guarantee: a committed change always has its job
// and a rolled back change never does. Domain services receive an insert-only
// client and call it inside store.WithTx:
//
//	_, err := client.InsertTx(ctx, tx, users.SendWelcomeEmailArgs{UserID: id}, nil)
//
// The API builds an insert-only client with NewInsertClient. The worker process
// builds a working client with NewWorkerClient and registers the same workers
// and periodic jobs through a Registry.
package jobs

import (
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// Queue names operated by the worker process.
const (
	QueueDefault = river.QueueDefault
	QueueMedia   = "media"
	QueueBulk    = "bulk"
)

// JobTimeout is the maximum time a single job may run before River cancels it.
const JobTimeout = 5 * time.Minute

// RescueStuckJobsAfter is how long a job may stay running before another
// worker assumes its process died and rescues it.
const RescueStuckJobsAfter = 10 * time.Minute

// Registry collects the workers and periodic jobs shared by every River
// client. Both the API (insert only) and the worker process use it.
type Registry struct {
	workers  *river.Workers
	periodic []*river.PeriodicJob
}

// NewRegistry returns a registry with no workers or periodic jobs.
func NewRegistry() *Registry {
	return &Registry{workers: river.NewWorkers()}
}

// Workers returns the underlying River worker bundle.
func (r *Registry) Workers() *river.Workers {
	return r.workers
}

// AddPeriodic registers a periodic job. Periodic jobs are scheduled by the
// elected leader of the worker client.
func (r *Registry) AddPeriodic(job *river.PeriodicJob) {
	r.periodic = append(r.periodic, job)
}

// Periodic returns the registered periodic jobs.
func (r *Registry) Periodic() []*river.PeriodicJob {
	return r.periodic
}

// Register adds a worker for the job args type T.
func Register[T river.JobArgs](r *Registry, w river.Worker[T]) {
	river.AddWorker(r.workers, w)
}

// NewInsertClient returns a River client that can only insert jobs. It never
// starts the configured queues, so it is safe to use inside request handlers
// and domain services.
func NewInsertClient(pool *pgxpool.Pool, r *Registry, logger *slog.Logger) (*river.Client[pgx.Tx], error) {
	return river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger:  logger,
		Workers: r.Workers(),
	})
}

// NewWorkerClient returns a River client that processes jobs from the default,
// media, and bulk queues.
func NewWorkerClient(pool *pgxpool.Pool, r *Registry, logger *slog.Logger) (*river.Client[pgx.Tx], error) {
	return river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger:               logger,
		JobTimeout:           JobTimeout,
		RescueStuckJobsAfter: RescueStuckJobsAfter,
		Queues: map[string]river.QueueConfig{
			QueueDefault: {MaxWorkers: 10},
			QueueMedia:   {MaxWorkers: 2},
			QueueBulk:    {MaxWorkers: 1},
		},
		Workers:      r.Workers(),
		PeriodicJobs: r.Periodic(),
	})
}
