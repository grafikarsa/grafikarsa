package audit_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/audit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

type fixture struct {
	ctx   context.Context
	pool  *pgxpool.Pool
	store *store.Store
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	pool := testutil.NewDB(t)
	return &fixture{
		ctx:   context.Background(),
		pool:  pool,
		store: store.NewStore(pool),
	}
}

// recordAt records e through the domain API and backdates it to at, so filter
// and pagination tests do not depend on the wall clock.
func (f *fixture) recordAt(t *testing.T, e audit.Entry, at time.Time) {
	t.Helper()
	if err := audit.Record(f.ctx, f.store.Queries, e); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if _, err := f.pool.Exec(f.ctx,
		"UPDATE audit_logs SET occurred_at = $1 WHERE id = (SELECT max(id) FROM audit_logs)",
		at,
	); err != nil {
		t.Fatalf("set occurred_at: %v", err)
	}
}

func (f *fixture) createUser(t *testing.T) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := f.pool.QueryRow(f.ctx, `
		INSERT INTO users (account_type, display_name, password_hash, password_kind, password_expires_at)
		VALUES ('staff', 'Audit Tester', 'x', 'initial', now() + interval '1 day')
		RETURNING id`,
	).Scan(&id)
	if err != nil {
		t.Fatalf("buat user uji: %v", err)
	}
	return id
}

func (f *fixture) createMajor(t *testing.T) uuid.UUID {
	t.Helper()
	return f.createMajorCode(t, "audit-test", "Audit Test", "AT")
}

func (f *fixture) createMajorCode(t *testing.T, code, name, shortName string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := f.pool.QueryRow(f.ctx, `
		INSERT INTO majors (code, name, short_name)
		VALUES ($1, $2, $3)
		RETURNING id`, code, name, shortName,
	).Scan(&id)
	if err != nil {
		t.Fatalf("buat major uji: %v", err)
	}
	return id
}

func (f *fixture) createUserNamed(t *testing.T, displayName string, username *string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := f.pool.QueryRow(f.ctx, `
		INSERT INTO users (account_type, display_name, username, password_hash, password_kind, password_expires_at)
		VALUES ('staff', $1, $2, 'x', 'initial', now() + interval '1 day')
		RETURNING id`, displayName, username,
	).Scan(&id)
	if err != nil {
		t.Fatalf("buat user uji: %v", err)
	}
	return id
}

func (f *fixture) countLogs(t *testing.T) int {
	t.Helper()
	var count int
	if err := f.pool.QueryRow(f.ctx, "SELECT count(*) FROM audit_logs").Scan(&count); err != nil {
		t.Fatalf("hitung audit log: %v", err)
	}
	return count
}

func TestRecordAndList_RoundTrip(t *testing.T) {
	f := newFixture(t)
	actorID := f.createUser(t)
	majorID := f.createMajor(t)

	entry := audit.Entry{
		ActorID:      &actorID,
		ActorIP:      "203.0.113.7",
		Action:       "user.reset_password",
		TargetType:   "user",
		TargetID:     &actorID,
		ScopeMajorID: &majorID,
		Data:         map[string]any{"reason": "manual", "delivered": true},
	}
	if err := audit.Record(f.ctx, f.store.Queries, entry); err != nil {
		t.Fatalf("Record: %v", err)
	}

	page, err := audit.List(f.ctx, f.store.Queries, audit.Filter{ActorID: &actorID})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("List mengembalikan %d item, mau 1", len(page.Items))
	}
	if page.NextCursor != "" {
		t.Errorf("NextCursor = %q, mau kosong", page.NextCursor)
	}

	item := page.Items[0]
	if item.ID <= 0 {
		t.Errorf("ID = %d, mau > 0", item.ID)
	}
	if item.OccurredAt.IsZero() {
		t.Error("OccurredAt kosong")
	}
	if item.ActorID == nil || *item.ActorID != actorID {
		t.Errorf("ActorID = %v, mau %s", item.ActorID, actorID)
	}
	if item.ActorIP == nil || *item.ActorIP != "203.0.113.7" {
		t.Errorf("ActorIP = %v, mau 203.0.113.7", item.ActorIP)
	}
	if item.Action != entry.Action {
		t.Errorf("Action = %q, mau %q", item.Action, entry.Action)
	}
	if item.TargetType == nil || *item.TargetType != entry.TargetType {
		t.Errorf("TargetType = %v, mau %q", item.TargetType, entry.TargetType)
	}
	if item.TargetID == nil || *item.TargetID != actorID {
		t.Errorf("TargetID = %v, mau %s", item.TargetID, actorID)
	}
	if item.ScopeMajorID == nil || *item.ScopeMajorID != majorID {
		t.Errorf("ScopeMajorID = %v, mau %s", item.ScopeMajorID, majorID)
	}
	if item.Data["reason"] != "manual" {
		t.Errorf("Data[reason] = %v, mau manual", item.Data["reason"])
	}
	if item.Data["delivered"] != true {
		t.Errorf("Data[delivered] = %v, mau true", item.Data["delivered"])
	}
}

func TestRecord_InvalidActionReturnsError(t *testing.T) {
	f := newFixture(t)

	err := audit.Record(f.ctx, f.store.Queries, audit.Entry{Action: "User.Reset"})
	if err == nil {
		t.Fatal("Record menerima action tidak valid")
	}
	if count := f.countLogs(t); count != 0 {
		t.Fatalf("audit log tersimpan %d baris, mau 0", count)
	}
}

func TestRecord_RejectsSensitiveDataKeys(t *testing.T) {
	f := newFixture(t)

	cases := map[string]map[string]any{
		"top level":       {"password": "rahasia"},
		"nested":          {"meta": map[string]any{"Token": "x"}},
		"nested array":    {"events": []any{map[string]any{"Code_Hash": "x"}}},
		"recovery code":   {"recovery_code": "abc"},
		"different case":  {"PASSWORD_HASH": "x"},
		"unknown secret":  {"secret": "x"},
		"password change": {"new_password": "x"},
	}

	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			err := audit.Record(f.ctx, f.store.Queries, audit.Entry{
				Action: "user.reset_password",
				Data:   data,
			})
			if err == nil {
				t.Fatalf("Record menerima data sensitif: %v", data)
			}
		})
	}

	if count := f.countLogs(t); count != 0 {
		t.Fatalf("audit log tersimpan %d baris, mau 0", count)
	}
}

func TestList_Filters(t *testing.T) {
	f := newFixture(t)
	actorID := f.createUser(t)
	otherUserID := f.createUser(t)
	base := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	entries := []struct {
		entry audit.Entry
		at    time.Time
	}{
		{
			entry: audit.Entry{ActorID: &actorID, Action: "user.reset_password", TargetType: "user", TargetID: &actorID},
			at:    base,
		},
		{
			entry: audit.Entry{ActorID: &actorID, Action: "user.deactivate", TargetType: "user", TargetID: &otherUserID},
			at:    base.Add(24 * time.Hour),
		},
		{
			entry: audit.Entry{ActorID: &otherUserID, Action: "content.approve", TargetType: "content", TargetID: &actorID},
			at:    base.Add(48 * time.Hour),
		},
		{
			entry: audit.Entry{ActorID: &actorID, Action: "user.reset_password", TargetType: "user", TargetID: &otherUserID},
			at:    base.Add(72 * time.Hour),
		},
	}
	for _, e := range entries {
		f.recordAt(t, e.entry, e.at)
	}

	t.Run("action prefix", func(t *testing.T) {
		page, err := audit.List(f.ctx, f.store.Queries, audit.Filter{ActionPrefix: "user."})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(page.Items) != 3 {
			t.Fatalf("List mengembalikan %d item, mau 3", len(page.Items))
		}
		for _, item := range page.Items {
			if !strings.HasPrefix(item.Action, "user.") {
				t.Errorf("Action = %q, mau berawalan user.", item.Action)
			}
		}
	})

	t.Run("exact action", func(t *testing.T) {
		page, err := audit.List(f.ctx, f.store.Queries, audit.Filter{Action: "content.approve"})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(page.Items) != 1 || page.Items[0].Action != "content.approve" {
			t.Fatalf("List = %+v, mau hanya content.approve", page.Items)
		}
	})

	t.Run("target", func(t *testing.T) {
		page, err := audit.List(f.ctx, f.store.Queries, audit.Filter{
			TargetType: "user",
			TargetID:   &otherUserID,
		})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(page.Items) != 2 {
			t.Fatalf("List mengembalikan %d item, mau 2", len(page.Items))
		}
		for _, item := range page.Items {
			if item.TargetID == nil || *item.TargetID != otherUserID {
				t.Errorf("TargetID = %v, mau %s", item.TargetID, otherUserID)
			}
			if item.TargetType == nil || *item.TargetType != "user" {
				t.Errorf("TargetType = %v, mau user", item.TargetType)
			}
		}
	})

	t.Run("time range", func(t *testing.T) {
		from := base.Add(12 * time.Hour)
		to := base.Add(72 * time.Hour)
		page, err := audit.List(f.ctx, f.store.Queries, audit.Filter{From: &from, To: &to})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(page.Items) != 2 {
			t.Fatalf("List mengembalikan %d item, mau 2", len(page.Items))
		}
		if page.Items[0].Action != "content.approve" || page.Items[1].Action != "user.deactivate" {
			t.Fatalf("urutan = [%s, %s], mau [content.approve, user.deactivate]",
				page.Items[0].Action, page.Items[1].Action)
		}
	})
}

func TestList_Pagination(t *testing.T) {
	f := newFixture(t)
	base := time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC)

	const total = 7
	for i := 0; i < total; i++ {
		f.recordAt(t, audit.Entry{
			Action: "content.step",
			Data:   map[string]any{"step": i},
		}, base.Add(time.Duration(i)*time.Minute))
	}

	rows, err := f.pool.Query(f.ctx,
		"SELECT id FROM audit_logs ORDER BY occurred_at DESC, id DESC")
	if err != nil {
		t.Fatalf("query id: %v", err)
	}
	defer rows.Close()

	var wantIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan id: %v", err)
		}
		wantIDs = append(wantIDs, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("baca id: %v", err)
	}
	if len(wantIDs) != total {
		t.Fatalf("ada %d baris audit, mau %d", len(wantIDs), total)
	}

	var (
		gotIDs     []int64
		pageSizes  []int
		seen       = map[int64]bool{}
		nextCursor string
	)
	for {
		page, err := audit.List(f.ctx, f.store.Queries, audit.Filter{
			Limit:  3,
			Cursor: nextCursor,
		})
		if err != nil {
			t.Fatalf("List halaman: %v", err)
		}
		pageSizes = append(pageSizes, len(page.Items))
		for _, item := range page.Items {
			if seen[item.ID] {
				t.Fatalf("id %d muncul di lebih dari satu halaman", item.ID)
			}
			seen[item.ID] = true
			gotIDs = append(gotIDs, item.ID)
		}
		if page.NextCursor == "" {
			break
		}
		nextCursor = page.NextCursor
	}

	if want := []int{3, 3, 1}; !reflect.DeepEqual(pageSizes, want) {
		t.Fatalf("ukuran halaman = %v, mau %v", pageSizes, want)
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("urutan id = %v, mau %v", gotIDs, wantIDs)
	}
}

func TestList_InvalidCursorReturnsBadRequest(t *testing.T) {
	f := newFixture(t)

	_, err := audit.List(f.ctx, f.store.Queries, audit.Filter{Cursor: "not-a-cursor"})
	if err == nil {
		t.Fatal("List menerima cursor tidak valid")
	}
	e, ok := apperr.As(err)
	if !ok {
		t.Fatalf("error = %v, mau *apperr.Error", err)
	}
	if e.Code != "BAD_REQUEST" {
		t.Fatalf("kode error = %q, mau BAD_REQUEST", e.Code)
	}
}

func TestRecord_TransactionRollbackPersistsNothing(t *testing.T) {
	f := newFixture(t)
	errRollback := errors.New("uji rollback")

	err := f.store.WithTx(f.ctx, func(q *store.Queries) error {
		if err := audit.Record(f.ctx, q, audit.Entry{Action: "user.reset_password"}); err != nil {
			return err
		}
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatalf("WithTx error = %v, mau %v", err, errRollback)
	}

	if count := f.countLogs(t); count != 0 {
		t.Fatalf("audit log tersimpan %d baris, mau 0", count)
	}
}

func TestRecord_InvalidActorIPStoredAsNull(t *testing.T) {
	f := newFixture(t)

	if err := audit.Record(f.ctx, f.store.Queries, audit.Entry{
		Action:  "user.reset_password",
		ActorIP: "bukan-ip",
	}); err != nil {
		t.Fatalf("Record: %v", err)
	}

	page, err := audit.List(f.ctx, f.store.Queries, audit.Filter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("List mengembalikan %d item, mau 1", len(page.Items))
	}
	if page.Items[0].ActorIP != nil {
		t.Fatalf("ActorIP = %v, mau nil", page.Items[0].ActorIP)
	}
}

func TestListForActor_GlobalSeesAllWithActorIdentity(t *testing.T) {
	f := newFixture(t)
	username := "audit.admin"
	actorID := f.createUserNamed(t, "Audit Admin", &username)
	majorID := f.createMajor(t)

	if err := audit.Record(f.ctx, f.store.Queries, audit.Entry{
		ActorID: &actorID,
		ActorIP: "203.0.113.9",
		Action:  "user.create",
		Data:    map[string]any{"account_type": "student"},
	}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := audit.Record(f.ctx, f.store.Queries, audit.Entry{
		ActorID:      &actorID,
		Action:       "user.update",
		ScopeMajorID: &majorID,
	}); err != nil {
		t.Fatalf("Record: %v", err)
	}

	principal := audit.Scope{Granted: true, Global: true}
	page, err := audit.ListForActor(f.ctx, f.store.Queries, principal, audit.Filter{})
	if err != nil {
		t.Fatalf("ListForActor: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("ListForActor mengembalikan %d item, mau 2", len(page.Items))
	}
	for _, item := range page.Items {
		if item.ActorName == nil || *item.ActorName != "Audit Admin" {
			t.Errorf("ActorName = %v, mau Audit Admin", item.ActorName)
		}
		if item.ActorUsername == nil || *item.ActorUsername != username {
			t.Errorf("ActorUsername = %v, mau %s", item.ActorUsername, username)
		}
		if item.Action == "user.create" && item.ActorIP == nil {
			t.Error("aktor global tidak menerima actorIp")
		}
	}
}

func TestListForActor_ScopedSeesOnlyItsMajorsAndHidesNullScopes(t *testing.T) {
	f := newFixture(t)
	actorID := f.createUser(t)
	rplID := f.createMajorCode(t, "rpl", "Rekayasa Perangkat Lunak", "RPL")
	tkjID := f.createMajorCode(t, "tkj", "Teknik Komputer dan Jaringan", "TKJ")

	entries := []audit.Entry{
		{ActorID: &actorID, Action: "user.create", ScopeMajorID: &rplID},
		{ActorID: &actorID, Action: "user.update", ScopeMajorID: &tkjID},
		{ActorID: &actorID, Action: "role.assign"},
	}
	for _, entry := range entries {
		if err := audit.Record(f.ctx, f.store.Queries, entry); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}

	principal := audit.Scope{Granted: true, MajorIDs: []uuid.UUID{rplID}}
	page, err := audit.ListForActor(f.ctx, f.store.Queries, principal, audit.Filter{})
	if err != nil {
		t.Fatalf("ListForActor: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("ListForActor mengembalikan %d item, mau 1", len(page.Items))
	}
	item := page.Items[0]
	if item.Action != "user.create" {
		t.Errorf("Action = %q, mau user.create", item.Action)
	}
	if item.ScopeMajorID == nil || *item.ScopeMajorID != rplID {
		t.Errorf("ScopeMajorID = %v, mau %s", item.ScopeMajorID, rplID)
	}
}

func TestListForActor_ScopedNeverReceivesActorIP(t *testing.T) {
	f := newFixture(t)
	actorID := f.createUser(t)
	rplID := f.createMajor(t)

	if err := audit.Record(f.ctx, f.store.Queries, audit.Entry{
		ActorID:      &actorID,
		ActorIP:      "203.0.113.10",
		Action:       "user.create",
		ScopeMajorID: &rplID,
	}); err != nil {
		t.Fatalf("Record: %v", err)
	}

	principal := audit.Scope{Granted: true, MajorIDs: []uuid.UUID{rplID}}
	page, err := audit.ListForActor(f.ctx, f.store.Queries, principal, audit.Filter{})
	if err != nil {
		t.Fatalf("ListForActor: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("ListForActor mengembalikan %d item, mau 1", len(page.Items))
	}
	if page.Items[0].ActorIP != nil {
		t.Fatalf("ActorIP = %v, mau nil untuk aktor ber-scope", page.Items[0].ActorIP)
	}
}

func TestListForActor_WithoutPermissionReturnsNotFound(t *testing.T) {
	f := newFixture(t)
	if err := audit.Record(f.ctx, f.store.Queries, audit.Entry{Action: "user.create"}); err != nil {
		t.Fatalf("Record: %v", err)
	}

	_, err := audit.ListForActor(f.ctx, f.store.Queries, audit.Scope{}, audit.Filter{})
	if err == nil {
		t.Fatal("ListForActor menerima scope tanpa izin audit.view")
	}
	e, ok := apperr.As(err)
	if !ok {
		t.Fatalf("error = %v, mau *apperr.Error", err)
	}
	if e.Code != "NOT_FOUND" {
		t.Fatalf("kode error = %q, mau NOT_FOUND", e.Code)
	}
}
