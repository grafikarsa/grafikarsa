package jobs_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/jobs"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

// jobStateCompleted mirrors river_job.state for a job that ran successfully.
const jobStateCompleted = "completed"

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newRegistry() *jobs.Registry {
	registry := jobs.NewRegistry()
	jobs.Register(registry, &jobs.PingWorker{Logger: testLogger()})
	return registry
}

func countJobs(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()

	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM river_job").Scan(&count); err != nil {
		t.Fatalf("hitung river_job: %v", err)
	}
	return count
}

func waitForJobState(t *testing.T, ctx context.Context, pool *pgxpool.Pool, jobID int64, want string) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for {
		var state string
		if err := pool.QueryRow(ctx, "SELECT state FROM river_job WHERE id = $1", jobID).Scan(&state); err != nil {
			t.Fatalf("baca state job %d: %v", jobID, err)
		}
		if state == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("state job %d = %s setelah 10 detik, mau %s", jobID, state, want)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestInsertClient_RolledBackTxLeavesNoJob(t *testing.T) {
	ctx := context.Background()
	pool := testutil.NewDB(t)

	client, err := jobs.NewInsertClient(pool, newRegistry(), testLogger())
	if err != nil {
		t.Fatalf("buat insert client: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("mulai transaksi: %v", err)
	}
	// Inside the transaction the job row exists; rolling back must remove it.
	if _, err := client.InsertTx(ctx, tx, jobs.PingArgs{}, nil); err != nil {
		t.Fatalf("insert job: %v", err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback transaksi: %v", err)
	}

	if got := countJobs(t, pool); got != 0 {
		t.Fatalf("jumlah river_job setelah rollback = %d, mau 0", got)
	}
}

func TestInsertClient_CommittedTxInsertsJob(t *testing.T) {
	ctx := context.Background()
	pool := testutil.NewDB(t)

	client, err := jobs.NewInsertClient(pool, newRegistry(), testLogger())
	if err != nil {
		t.Fatalf("buat insert client: %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("mulai transaksi: %v", err)
	}
	result, err := client.InsertTx(ctx, tx, jobs.PingArgs{}, nil)
	if err != nil {
		t.Fatalf("insert job: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit transaksi: %v", err)
	}

	if got := countJobs(t, pool); got != 1 {
		t.Fatalf("jumlah river_job setelah commit = %d, mau 1", got)
	}

	var kind, queue string
	if err := pool.QueryRow(ctx, "SELECT kind, queue FROM river_job WHERE id = $1", result.Job.ID).Scan(&kind, &queue); err != nil {
		t.Fatalf("baca job %d: %v", result.Job.ID, err)
	}
	if kind != "system.ping" {
		t.Fatalf("kind job = %q, mau %q", kind, "system.ping")
	}
	if queue != jobs.QueueDefault {
		t.Fatalf("queue job = %q, mau %q", queue, jobs.QueueDefault)
	}
}

func TestWorkerClient_CompletesPingJob(t *testing.T) {
	ctx := context.Background()
	pool := testutil.NewDB(t)
	registry := newRegistry()

	insertClient, err := jobs.NewInsertClient(pool, registry, testLogger())
	if err != nil {
		t.Fatalf("buat insert client: %v", err)
	}

	result, err := insertClient.Insert(ctx, jobs.PingArgs{}, nil)
	if err != nil {
		t.Fatalf("insert job: %v", err)
	}

	workerClient, err := jobs.NewWorkerClient(pool, registry, testLogger())
	if err != nil {
		t.Fatalf("buat worker client: %v", err)
	}
	if err := workerClient.Start(ctx); err != nil {
		t.Fatalf("jalankan worker client: %v", err)
	}
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := workerClient.Stop(stopCtx); err != nil {
			t.Errorf("hentikan worker client: %v", err)
		}
	})

	waitForJobState(t, ctx, pool, result.Job.ID, jobStateCompleted)
}
