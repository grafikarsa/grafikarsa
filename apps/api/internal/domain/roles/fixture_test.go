package roles_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/roles"
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

func (f *fakeInvalidator) contains(userID uuid.UUID) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, call := range f.calls {
		if call == userID {
			return true
		}
	}
	return false
}

type fixture struct {
	t       *testing.T
	ctx     context.Context
	pool    *pgxpool.Pool
	store   *store.Store
	service *roles.Service
	inv     *fakeInvalidator

	rplID uuid.UUID
	tkjID uuid.UUID

	globalAdminID  uuid.UUID
	scopedAdminID  uuid.UUID
	limitedAdminID uuid.UUID
	plainUserID    uuid.UUID
	targetID       uuid.UUID
	inactiveID     uuid.UUID

	allPermissions        []string
	majorAdminPermissions []string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	pool := testutil.NewSeededDB(t)
	st := store.NewStore(pool)
	inv := &fakeInvalidator{}

	f := &fixture{
		t:       t,
		ctx:     context.Background(),
		pool:    pool,
		store:   st,
		service: roles.NewService(st, inv),
		inv:     inv,
	}

	f.rplID = f.majorID("rpl")
	f.tkjID = f.majorID("tkj")
	f.allPermissions = f.permissionKeys()
	f.majorAdminPermissions = f.rolePermissionKeys("major_admin")

	f.globalAdminID = f.insertUser("Admin Global", "active")
	f.scopedAdminID = f.insertUser("Admin RPL", "active")
	f.limitedAdminID = f.insertUser("Admin Terbatas", "active")
	f.plainUserID = f.insertUser("Pengguna Biasa", "active")
	f.targetID = f.insertUser("Target Role", "active")
	f.inactiveID = f.insertUser("Target Nonaktif", "inactive")

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

func (f *fixture) permissionKeys() []string {
	f.t.Helper()
	rows, err := f.pool.Query(f.ctx, "SELECT key FROM permissions ORDER BY key")
	if err != nil {
		f.t.Fatalf("ambil katalog izin: %v", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			f.t.Fatalf("baca izin: %v", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		f.t.Fatalf("iterasi izin: %v", err)
	}
	return keys
}

func (f *fixture) rolePermissionKeys(slug string) []string {
	f.t.Helper()
	rows, err := f.pool.Query(f.ctx, `
		SELECT rp.permission_key
		FROM role_permissions rp
		JOIN roles r ON r.id = rp.role_id
		WHERE r.slug = $1
		ORDER BY rp.permission_key`,
		slug,
	)
	if err != nil {
		f.t.Fatalf("ambil izin role %s: %v", slug, err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			f.t.Fatalf("baca izin role: %v", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		f.t.Fatalf("iterasi izin role: %v", err)
	}
	return keys
}

func (f *fixture) insertUser(displayName, status string) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	err := f.pool.QueryRow(f.ctx, `
		INSERT INTO users (account_type, display_name, password_hash, password_kind)
		VALUES ('staff', $1, 'x', 'user')
		RETURNING id`,
		displayName,
	).Scan(&id)
	if err != nil {
		f.t.Fatalf("buat user %q: %v", displayName, err)
	}

	if status != "active" {
		if _, err := f.pool.Exec(f.ctx,
			"UPDATE users SET status = $2, deactivated_at = now() WHERE id = $1", id, status,
		); err != nil {
			f.t.Fatalf("ubah status user: %v", err)
		}
	}
	return id
}

func (f *fixture) roleID(slug string) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	if err := f.pool.QueryRow(f.ctx, "SELECT id FROM roles WHERE slug = $1", slug).Scan(&id); err != nil {
		f.t.Fatalf("ambil role %s: %v", slug, err)
	}
	return id
}

func globalGrants(permissions []string) []auth.Grant {
	grants := make([]auth.Grant, 0, len(permissions))
	for _, permission := range permissions {
		grants = append(grants, auth.Grant{Permission: permission, Global: true})
	}
	return grants
}

func scopedGrants(permissions []string, majorID uuid.UUID) []auth.Grant {
	grants := make([]auth.Grant, 0, len(permissions))
	for _, permission := range permissions {
		grants = append(grants, auth.Grant{
			Permission: permission,
			MajorIDs:   []uuid.UUID{majorID},
		})
	}
	return grants
}

func (f *fixture) globalActor() roles.Actor {
	return roles.Actor{
		Principal: &auth.Principal{UserID: f.globalAdminID, Grants: globalGrants(f.allPermissions)},
		IP:        testActorIP,
	}
}

func (f *fixture) scopedActor() roles.Actor {
	return roles.Actor{
		Principal: &auth.Principal{UserID: f.scopedAdminID, Grants: scopedGrants(f.majorAdminPermissions, f.rplID)},
		IP:        testActorIP,
	}
}

// limitedActor can manage roles globally but only holds content.create, so it
// must not be able to add permissions it does not have.
func (f *fixture) limitedActor() roles.Actor {
	return roles.Actor{
		Principal: &auth.Principal{
			UserID: f.limitedAdminID,
			Grants: globalGrants([]string{"content.create", "role.manage"}),
		},
		IP: testActorIP,
	}
}

func (f *fixture) plainActor() roles.Actor {
	return roles.Actor{
		Principal: &auth.Principal{
			UserID: f.plainUserID,
			Grants: globalGrants([]string{"content.create"}),
		},
		IP: testActorIP,
	}
}

func (f *fixture) createRole(a roles.Actor, slug, name string, permissions []string) roles.Role {
	f.t.Helper()
	role, err := f.service.CreateRole(f.ctx, a, slug, name, nil, permissions)
	if err != nil {
		f.t.Fatalf("CreateRole %q: %v", slug, err)
	}
	return role
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

func (f *fixture) auditCount(action string) int {
	f.t.Helper()
	count := 0
	for _, got := range f.auditActions() {
		if got == action {
			count++
		}
	}
	return count
}

func (f *fixture) hasAuditAction(action string) bool {
	f.t.Helper()
	return f.auditCount(action) > 0
}

func (f *fixture) assignmentRevocation(id uuid.UUID) (revokedAt *time.Time, revokedBy *uuid.UUID) {
	f.t.Helper()
	err := f.pool.QueryRow(f.ctx,
		"SELECT revoked_at, revoked_by FROM user_roles WHERE id = $1", id,
	).Scan(&revokedAt, &revokedBy)
	if err != nil {
		f.t.Fatalf("ambil status pencabutan: %v", err)
	}
	return revokedAt, revokedBy
}

func (f *fixture) roleCreatedBy(id uuid.UUID) *uuid.UUID {
	f.t.Helper()
	var createdBy *uuid.UUID
	if err := f.pool.QueryRow(f.ctx, "SELECT created_by FROM roles WHERE id = $1", id).Scan(&createdBy); err != nil {
		f.t.Fatalf("ambil pembuat role: %v", err)
	}
	return createdBy
}

func (f *fixture) roleExists(id uuid.UUID) bool {
	f.t.Helper()
	var exists bool
	if err := f.pool.QueryRow(f.ctx, "SELECT EXISTS (SELECT 1 FROM roles WHERE id = $1)", id).Scan(&exists); err != nil {
		f.t.Fatalf("cek role: %v", err)
	}
	return exists
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

func requireField(t *testing.T, e *apperr.Error, field, code string) {
	t.Helper()
	for _, got := range e.Fields {
		if got.Field == field && got.Code == code {
			return
		}
	}
	t.Fatalf("field error %s/%s tidak ada di %+v", field, code, e.Fields)
}
