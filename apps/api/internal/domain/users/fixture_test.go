package users_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/users"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/password"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/securetoken"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

const testActorIP = "203.0.113.7"

type fakeInvalidator struct {
	mu    sync.Mutex
	calls []uuid.UUID
}

func (f *fakeInvalidator) InvalidateUser(_ context.Context, userID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, userID)
	return nil
}

func (f *fakeInvalidator) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func (f *fakeInvalidator) last() uuid.UUID {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) == 0 {
		return uuid.Nil
	}
	return f.calls[len(f.calls)-1]
}

type fixture struct {
	t           *testing.T
	ctx         context.Context
	pool        *pgxpool.Pool
	store       *store.Store
	service     *users.Service
	hasher      *password.Hasher
	invalidator *fakeInvalidator

	yearID uuid.UUID
	rplID  uuid.UUID
	tkjID  uuid.UUID

	rplClassID uuid.UUID
	tkjClassID uuid.UUID

	globalAdminID uuid.UUID
	scopedAdminID uuid.UUID

	nisnSeq     int
	usernameSeq int
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	pool := testutil.NewSeededDB(t)
	st := store.NewStore(pool)
	hasher := password.NewHasher(password.Params{
		Memory:      1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	}, 1)
	invalidator := &fakeInvalidator{}

	f := &fixture{
		t:           t,
		ctx:         context.Background(),
		pool:        pool,
		store:       st,
		service:     users.NewService(st, invalidator, hasher),
		hasher:      hasher,
		invalidator: invalidator,
	}

	f.rplID = f.majorID("rpl")
	f.tkjID = f.majorID("tkj")
	f.yearID = f.insertYear(2026)
	f.rplClassID = f.insertClass(f.rplID, 10, "A")
	f.tkjClassID = f.insertClass(f.tkjID, 10, "A")
	f.globalAdminID = f.insertActor("Admin Global")
	f.scopedAdminID = f.insertActor("Admin RPL")

	return f
}

func (f *fixture) majorID(code string) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	if err := f.pool.QueryRow(f.ctx, "SELECT id FROM majors WHERE code = $1", code).Scan(&id); err != nil {
		f.t.Fatalf("ambil jurusan %s: %v", code, err)
	}
	return id
}

func (f *fixture) insertYear(startYear int16) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	err := f.pool.QueryRow(f.ctx, `
		INSERT INTO academic_years (start_year, starts_on, ends_on)
		VALUES ($1, $2, $3)
		RETURNING id`,
		startYear,
		fmt.Sprintf("%d-07-01", startYear),
		fmt.Sprintf("%d-06-30", startYear+1),
	).Scan(&id)
	if err != nil {
		f.t.Fatalf("buat tahun ajaran: %v", err)
	}
	return id
}

func (f *fixture) insertClass(majorID uuid.UUID, grade int16, section string) uuid.UUID {
	f.t.Helper()

	var shortName string
	if err := f.pool.QueryRow(f.ctx, "SELECT short_name FROM majors WHERE id = $1", majorID).Scan(&shortName); err != nil {
		f.t.Fatalf("ambil nama singkat jurusan: %v", err)
	}

	name := fmt.Sprintf("%s %s %s", romanGrade(grade), shortName, section)
	slug := fmt.Sprintf("%s-%s-%s", lowerRomanGrade(grade), lower(shortName), lower(section))

	var id uuid.UUID
	err := f.pool.QueryRow(f.ctx, `
		INSERT INTO classes (academic_year_id, major_id, grade, section, name, slug)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		f.yearID, majorID, grade, section, name, slug,
	).Scan(&id)
	if err != nil {
		f.t.Fatalf("buat kelas: %v", err)
	}

	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO groups (kind, class_id, status) VALUES ('class', $1, 'active')`, id); err != nil {
		f.t.Fatalf("buat grup kelas: %v", err)
	}
	return id
}

func (f *fixture) insertActor(displayName string) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	err := f.pool.QueryRow(f.ctx, `
		INSERT INTO users (account_type, display_name, password_hash, password_kind)
		VALUES ('staff', $1, 'x', 'user')
		RETURNING id`,
		displayName,
	).Scan(&id)
	if err != nil {
		f.t.Fatalf("buat user aktor: %v", err)
	}
	return id
}

func (f *fixture) nextNISN() string {
	f.nisnSeq++
	return fmt.Sprintf("10%08d", f.nisnSeq)
}

func (f *fixture) nextUsername() string {
	f.usernameSeq++
	return fmt.Sprintf("pengguna%d", f.usernameSeq)
}

func globalUserGrants() []auth.Grant {
	return []auth.Grant{
		{Permission: "user.block_content", Global: true},
		{Permission: "user.create", Global: true},
		{Permission: "user.deactivate", Global: true},
		{Permission: "user.reset_password", Global: true},
		{Permission: "user.update", Global: true},
		{Permission: "user.view", Global: true},
	}
}

func scopedUserGrants(majorID uuid.UUID) []auth.Grant {
	return []auth.Grant{
		{Permission: "user.block_content", MajorIDs: []uuid.UUID{majorID}},
		{Permission: "user.create", MajorIDs: []uuid.UUID{majorID}},
		{Permission: "user.deactivate", MajorIDs: []uuid.UUID{majorID}},
		{Permission: "user.reset_password", MajorIDs: []uuid.UUID{majorID}},
		{Permission: "user.update", MajorIDs: []uuid.UUID{majorID}},
		{Permission: "user.view", MajorIDs: []uuid.UUID{majorID}},
	}
}

func (f *fixture) globalActor() users.Actor {
	return users.Actor{
		Principal: &auth.Principal{UserID: f.globalAdminID, Grants: globalUserGrants()},
		IP:        testActorIP,
	}
}

func (f *fixture) scopedActor() users.Actor {
	return users.Actor{
		Principal: &auth.Principal{UserID: f.scopedAdminID, Grants: scopedUserGrants(f.rplID)},
		IP:        testActorIP,
	}
}

func (f *fixture) createStudent(actor users.Actor, classID uuid.UUID, displayName string) users.Created {
	f.t.Helper()
	nisn := f.nextNISN()
	created, err := f.service.Create(f.ctx, actor, users.CreateInput{
		AccountType: "student",
		DisplayName: displayName,
		NISN:        &nisn,
		ClassID:     &classID,
	})
	if err != nil {
		f.t.Fatalf("Create siswa %q: %v", displayName, err)
	}
	return created
}

func (f *fixture) createTeacher(actor users.Actor, displayName string, majorIDs []uuid.UUID) users.Created {
	f.t.Helper()
	created, err := f.service.Create(f.ctx, actor, users.CreateInput{
		AccountType:     "teacher",
		DisplayName:     displayName,
		TeacherMajorIDs: majorIDs,
	})
	if err != nil {
		f.t.Fatalf("Create guru %q: %v", displayName, err)
	}
	return created
}

func (f *fixture) activate(userID uuid.UUID) {
	f.t.Helper()
	_, err := f.pool.Exec(f.ctx, `
		UPDATE users
		SET activated_at = now(), username = $2
		WHERE id = $1`,
		userID, f.nextUsername(),
	)
	if err != nil {
		f.t.Fatalf("aktivasi user: %v", err)
	}
}

func (f *fixture) insertSession(userID uuid.UUID) uuid.UUID {
	f.t.Helper()
	_, tokenHash, err := securetoken.SessionToken()
	if err != nil {
		f.t.Fatalf("buat token sesi: %v", err)
	}

	var id uuid.UUID

	err = f.pool.QueryRow(f.ctx, `
		INSERT INTO sessions (user_id, token_hash, idle_expires_at, expires_at)
		VALUES ($1, $2, now() + interval '1 day', now() + interval '7 days')
		RETURNING id`,
		userID, tokenHash,
	).Scan(&id)
	if err != nil {
		f.t.Fatalf("buat sesi: %v", err)
	}
	return id
}

func (f *fixture) passwordHash(userID uuid.UUID) string {
	f.t.Helper()
	var hash string
	if err := f.pool.QueryRow(f.ctx, "SELECT password_hash FROM users WHERE id = $1", userID).Scan(&hash); err != nil {
		f.t.Fatalf("ambil hash sandi: %v", err)
	}
	return hash
}

func (f *fixture) passwordInfo(userID uuid.UUID) (kind string, expiresAt time.Time) {
	f.t.Helper()
	if err := f.pool.QueryRow(f.ctx,
		"SELECT password_kind, password_expires_at FROM users WHERE id = $1", userID,
	).Scan(&kind, &expiresAt); err != nil {
		f.t.Fatalf("ambil info sandi: %v", err)
	}
	return kind, expiresAt
}

func (f *fixture) userStatus(userID uuid.UUID) (status string, deactivatedAt *time.Time) {
	f.t.Helper()
	if err := f.pool.QueryRow(f.ctx,
		"SELECT status, deactivated_at FROM users WHERE id = $1", userID,
	).Scan(&status, &deactivatedAt); err != nil {
		f.t.Fatalf("ambil status user: %v", err)
	}
	return status, deactivatedAt
}

func (f *fixture) revokedSessionCount(userID uuid.UUID, reason string) int {
	f.t.Helper()
	var count int
	if err := f.pool.QueryRow(f.ctx, `
		SELECT count(*) FROM sessions
		WHERE user_id = $1 AND revoked_at IS NOT NULL AND revoked_reason = $2`,
		userID, reason,
	).Scan(&count); err != nil {
		f.t.Fatalf("hitung sesi dicabut: %v", err)
	}
	return count
}

func (f *fixture) auditActions() []string {
	f.t.Helper()
	rows, err := f.pool.Query(f.ctx, "SELECT action FROM audit_logs ORDER BY id")
	if err != nil {
		f.t.Fatalf("ambil audit log: %v", err)
	}
	defer rows.Close()

	var actions []string
	for rows.Next() {
		var action string
		if err := rows.Scan(&action); err != nil {
			f.t.Fatalf("baca audit log: %v", err)
		}
		actions = append(actions, action)
	}
	if err := rows.Err(); err != nil {
		f.t.Fatalf("iterasi audit log: %v", err)
	}
	return actions
}

func (f *fixture) hasAuditAction(action string) bool {
	f.t.Helper()
	for _, got := range f.auditActions() {
		if got == action {
			return true
		}
	}
	return false
}

func requireAppError(t *testing.T, err error, code string) *apperr.Error {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, mau kode %s", code)
	}
	e, ok := apperr.As(err)
	if !ok {
		t.Fatalf("error = %v, mau *apperr.Error dengan kode %s", err, code)
	}
	if e.Code != code {
		t.Fatalf("kode error = %s (%v), mau %s", e.Code, err, code)
	}
	return e
}

func romanGrade(grade int16) string {
	switch grade {
	case 10:
		return "X"
	case 11:
		return "XI"
	case 12:
		return "XII"
	default:
		return fmt.Sprintf("%d", grade)
	}
}

func lowerRomanGrade(grade int16) string {
	return lower(romanGrade(grade))
}

func lower(value string) string {
	out := []byte(value)
	for i, b := range out {
		if b >= 'A' && b <= 'Z' {
			out[i] = b + ('a' - 'A')
		}
	}
	return string(out)
}
