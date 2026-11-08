package users

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/audit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/jobs"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const importCommitTimeout = 30 * time.Minute

// ImportCommitArgs is the River job that executes a validated import.
type ImportCommitArgs struct {
	ImportJobID uuid.UUID `json:"importJobId"`
	ActorID     uuid.UUID `json:"actorId"`
	ActorIP     string    `json:"actorIp,omitempty"`
}

// Kind implements river.JobArgs.
func (ImportCommitArgs) Kind() string { return "users.import_commit" }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (ImportCommitArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueBulk, MaxAttempts: 3}
}

// ImportCommitWorker processes users.import_commit jobs.
type ImportCommitWorker struct {
	river.WorkerDefaults[ImportCommitArgs]

	Importer *Importer
	Logger   *slog.Logger
}

// Timeout overrides the client-wide timeout: creating accounts hashes one
// password per row and may take longer than a regular job.
func (w *ImportCommitWorker) Timeout(*river.Job[ImportCommitArgs]) time.Duration {
	return importCommitTimeout
}

// Work implements river.Worker.
func (w *ImportCommitWorker) Work(ctx context.Context, job *river.Job[ImportCommitArgs]) error {
	if w.Importer == nil {
		return fmt.Errorf("users: import commit worker is not configured")
	}
	return w.Importer.RunCommitJob(ctx, job.Args)
}

type importRowOutcome struct {
	created bool
	reason  string
}

// RunCommitJob executes a committing import in batches of 100 rows. Each batch
// is one transaction. Running it again is safe: rows already created count as
// created instead of skipped.
func (s *Importer) RunCommitJob(ctx context.Context, args ImportCommitArgs) error {
	record, err := s.users.st.GetImportJob(ctx, args.ImportJobID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("users: load import job %s: %w", args.ImportJobID, err)
	}
	if record.Status != importStatusCommitting {
		return nil
	}

	summary := importSummary{ByClass: map[string]int{}}
	if len(record.Summary) > 0 {
		if err := json.Unmarshal(record.Summary, &summary); err != nil {
			s.failImportJob(args.ImportJobID, &summary, "Data impor tidak dapat dibaca.")
			return fmt.Errorf("users: decode import summary %s: %w", args.ImportJobID, err)
		}
	}
	if summary.ByClass == nil {
		summary.ByClass = map[string]int{}
	}

	created := 0
	skipped := make([]ImportSkippedRow, 0)

	for start := 0; start < len(summary.Rows); start += importBatchSize {
		end := min(start+importBatchSize, len(summary.Rows))
		batch := summary.Rows[start:end]

		batchCreated := 0
		batchSkipped := make([]ImportSkippedRow, 0)
		err := s.users.st.WithTx(ctx, func(q *store.Queries) error {
			for _, row := range batch {
				outcome, err := s.commitImportRow(ctx, q, row)
				if err != nil {
					return err
				}
				if outcome.created {
					batchCreated++
					continue
				}
				batchSkipped = append(batchSkipped, ImportSkippedRow{
					Row:    row.Row,
					Reason: outcome.reason,
				})
			}
			return nil
		})
		if err != nil {
			summary.Created = &created
			summary.Skipped = skipped
			s.failImportJob(args.ImportJobID, &summary, "Impor gagal diproses. Silakan coba lagi.")
			return fmt.Errorf("users: commit import batch for %s: %w", args.ImportJobID, err)
		}

		created += batchCreated
		skipped = append(skipped, batchSkipped...)
	}

	summary.Created = &created
	summary.Skipped = skipped
	summary.Failure = ""
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		s.failImportJob(args.ImportJobID, &summary, "Hasil impor tidak dapat disimpan.")
		return fmt.Errorf("users: encode import summary %s: %w", args.ImportJobID, err)
	}

	err = s.users.st.WithTx(ctx, func(q *store.Queries) error {
		affected, err := q.CompleteImportJob(ctx, store.CompleteImportJobParams{
			Summary: summaryJSON,
			ID:      args.ImportJobID,
		})
		if err != nil {
			return fmt.Errorf("users: complete import job: %w", err)
		}
		if affected == 0 {
			return nil
		}

		actorID := args.ActorID
		if err := audit.Record(ctx, q, audit.Entry{
			ActorID:    &actorID,
			ActorIP:    args.ActorIP,
			Action:     "user.import_commit",
			TargetType: targetTypeImport,
			TargetID:   &args.ImportJobID,
			Data: map[string]any{
				"kind":    record.Kind,
				"created": created,
				"skipped": len(skipped),
			},
		}); err != nil {
			return fmt.Errorf("users: record audit user.import_commit: %w", err)
		}
		return nil
	})
	if err != nil {
		summary.Created = &created
		summary.Skipped = skipped
		s.failImportJob(args.ImportJobID, &summary, "Hasil impor tidak dapat disimpan.")
		return err
	}
	return nil
}

// commitImportRow creates one account inside the batch transaction or decides
// that an existing account already satisfies the row.
func (s *Importer) commitImportRow(
	ctx context.Context,
	q *store.Queries,
	row storedImportRow,
) (importRowOutcome, error) {
	existing, err := q.GetImportExistingUser(ctx, store.GetImportExistingUserParams{
		Nisn:        row.NISN,
		StaffNumber: row.StaffNumber,
	})
	switch {
	case err == nil:
		return s.classifyExistingImportRow(ctx, q, row, existing)
	case !errors.Is(err, pgx.ErrNoRows):
		return importRowOutcome{}, fmt.Errorf("users: find existing import account: %w", err)
	}

	if row.AccountType == accountTypeStudent && row.ClassID == nil {
		return importRowOutcome{reason: "Data kelas tidak lengkap."}, nil
	}

	_, _, err = s.users.createUserTx(ctx, q, CreateInput{
		AccountType:     row.AccountType,
		DisplayName:     row.DisplayName,
		NISN:            row.NISN,
		StaffNumber:     row.StaffNumber,
		ClassID:         row.ClassID,
		EntryYear:       row.EntryYear,
		TeacherMajorIDs: row.TeacherMajorIDs,
	}, "import")
	if err != nil {
		return importRowOutcome{}, err
	}
	return importRowOutcome{created: true}, nil
}

func (s *Importer) classifyExistingImportRow(
	ctx context.Context,
	q *store.Queries,
	row storedImportRow,
	existing store.GetImportExistingUserRow,
) (importRowOutcome, error) {
	switch row.AccountType {
	case accountTypeStudent:
		if existing.AccountType == accountTypeStudent && row.ClassID != nil {
			enrolled, err := q.ImportUserHasEnrollment(ctx, store.ImportUserHasEnrollmentParams{
				UserID:  existing.ID,
				ClassID: *row.ClassID,
			})
			if err != nil {
				return importRowOutcome{}, fmt.Errorf("users: check import enrollment: %w", err)
			}
			if enrolled {
				return importRowOutcome{created: true}, nil
			}
		}
		return importRowOutcome{reason: "NISN sudah terdaftar sebelum impor dijalankan."}, nil

	case accountTypeTeacher, accountTypeStaff:
		if existing.AccountType == row.AccountType &&
			existing.DisplayName == row.DisplayName {
			return importRowOutcome{created: true}, nil
		}
		return importRowOutcome{reason: "NIP sudah terdaftar sebelum impor dijalankan."}, nil

	default:
		return importRowOutcome{reason: "Jenis akun tidak dikenal."}, nil
	}
}

// failImportJob records a short failure reason without leaking error details.
// It uses a detached context so a cancelled job context cannot prevent it.
func (s *Importer) failImportJob(jobID uuid.UUID, summary *importSummary, message string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(context.Background()), 10*time.Second)
	defer cancel()

	if summary.ByClass == nil {
		summary.ByClass = map[string]int{}
	}
	summary.Failure = message
	raw, err := json.Marshal(summary)
	if err != nil {
		return
	}
	_, _ = s.users.st.FailImportJob(ctx, store.FailImportJobParams{
		Summary: raw,
		ID:      jobID,
	})
}
