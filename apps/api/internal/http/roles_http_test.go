package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/roles"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/users"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/password"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

type rolesEnv struct {
	*flowEnv
}

// newRolesEnv builds the real router with auth, user, and role services on a
// seeded database, served over TLS so __Host- cookies are accepted.
func newRolesEnv(t *testing.T) *rolesEnv {
	t.Helper()

	pool := testutil.NewSeededDB(t)
	rdb := testutil.NewValkey(t)
	hasher := password.NewHasher(cheapPasswordParams, 0)
	logger := testLogger()

	st := store.NewStore(pool)
	authService, err := auth.NewService(
		st,
		rdb,
		hasher,
		ratelimit.New(rdb),
		auth.Config{RecoverySecret: testRecoverySecret},
		logger,
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	handler := NewRouter(RouterDeps{
		Logger:         logger,
		Auth:           authService,
		Users:          users.NewService(st, authService, hasher, newUsersTestBox(t)),
		Roles:          roles.NewService(st, authService),
		Limiter:        ratelimit.New(rdb),
		AllowedOrigins: []string{"http://localhost:3020"},
	})

	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	return &rolesEnv{flowEnv: &flowEnv{t: t, pool: pool, server: server, hasher: hasher}}
}

func (e *rolesEnv) majorID(code string) uuid.UUID {
	e.t.Helper()

	var id uuid.UUID
	if err := e.pool.QueryRow(context.Background(),
		`SELECT id FROM majors WHERE code = $1`, code,
	).Scan(&id); err != nil {
		e.t.Fatalf("load major %s: %v", code, err)
	}
	return id
}

func (e *rolesEnv) assignGlobalRole(userID uuid.UUID, roleSlug string) {
	e.t.Helper()

	tag, err := e.pool.Exec(context.Background(), `
		INSERT INTO user_roles (user_id, role_id)
		SELECT $1, id FROM roles WHERE slug = $2`,
		userID, roleSlug,
	)
	if err != nil {
		e.t.Fatalf("assign global role %s: %v", roleSlug, err)
	}
	if tag.RowsAffected() != 1 {
		e.t.Fatalf("assign global role %s: %d rows affected, want 1", roleSlug, tag.RowsAffected())
	}
}

func (e *rolesEnv) assignMajorRole(userID uuid.UUID, roleSlug, majorCode string) {
	e.t.Helper()

	tag, err := e.pool.Exec(context.Background(), `
		INSERT INTO user_roles (user_id, role_id, major_id)
		SELECT $1, r.id, m.id
		FROM roles r
		CROSS JOIN majors m
		WHERE r.slug = $2 AND m.code = $3`,
		userID, roleSlug, majorCode,
	)
	if err != nil {
		e.t.Fatalf("assign %s role for %s: %v", roleSlug, majorCode, err)
	}
	if tag.RowsAffected() != 1 {
		e.t.Fatalf("assign %s role for %s: %d rows affected, want 1", roleSlug, majorCode, tag.RowsAffected())
	}
}

// createAdmin inserts an activated staff account with the global admin role.
func (e *rolesEnv) createAdmin(username, displayName string) uuid.UUID {
	e.t.Helper()

	now := time.Now().UTC()
	id := e.createUser(flowUserSpec{
		accountType: "staff",
		username:    &username,
		displayName: displayName,
		password:    usersTestPassword,
		activatedAt: &now,
	})
	e.assignGlobalRole(id, "admin")
	return id
}

// createMajorAdmin inserts an activated teacher account with a scoped
// major_admin role.
func (e *rolesEnv) createMajorAdmin(username, displayName, majorCode string) uuid.UUID {
	e.t.Helper()

	now := time.Now().UTC()
	id := e.createUser(flowUserSpec{
		accountType: "teacher",
		username:    &username,
		displayName: displayName,
		password:    usersTestPassword,
		activatedAt: &now,
	})
	e.assignMajorRole(id, "major_admin", majorCode)
	return id
}

// createTeacher inserts an activated teacher account without extra roles.
func (e *rolesEnv) createTeacher(username, displayName string) uuid.UUID {
	e.t.Helper()

	now := time.Now().UTC()
	return e.createUser(flowUserSpec{
		accountType: "teacher",
		username:    &username,
		displayName: displayName,
		password:    usersTestPassword,
		activatedAt: &now,
	})
}

// loginAs signs in and returns the client plus the rotated CSRF token.
func (e *rolesEnv) loginAs(identifier string) (*http.Client, string) {
	e.t.Helper()

	c := e.client()
	csrf := e.fetchCSRF(c)
	res, body := e.login(c, identifier, usersTestPassword, csrf)
	if res.StatusCode != http.StatusOK {
		e.t.Fatalf("login %s status = %d, body %s", identifier, res.StatusCode, body)
	}
	return c, e.csrfFromJar(c)
}

func decodeRolesBody[T any](t *testing.T, body []byte) T {
	t.Helper()

	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode response: %v (body %s)", err, body)
	}
	return out
}

// roleGrant returns the effective grant of one permission, or nil.
func roleGrant(session apigen.Session, permission string) *apigen.RoleGrant {
	for i := range session.Grants {
		if session.Grants[i].Permission == permission {
			return &session.Grants[i]
		}
	}
	return nil
}

// requireFieldProblem asserts a validation problem that carries one field
// error with the given code.
func requireFieldProblem(t *testing.T, res *http.Response, body []byte, status int, problemCode, field, fieldCode string) {
	t.Helper()

	requireProblem(t, res, body, status, problemCode)

	problemBody := decodeRolesBody[apigen.Problem](t, body)
	if problemBody.Errors == nil {
		t.Fatalf("problem errors = nil, want a field error for %s", field)
	}
	for _, fieldError := range *problemBody.Errors {
		if fieldError.Field == field && fieldError.Code == fieldCode {
			return
		}
	}
	t.Fatalf("field error %s/%s not found in %+v", field, fieldCode, *problemBody.Errors)
}

func TestRolesHTTP_AdminListsSystemRolesAndStudentGetsNotFound(t *testing.T) {
	env := newRolesEnv(t)

	env.createAdmin("admin.role", "Admin Role")
	adminClient, _ := env.loginAs("admin.role")

	res, body := env.do(adminClient, http.MethodGet, "/api/v1/admin/roles", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /admin/roles status = %d, body %s", res.StatusCode, body)
	}

	roleList := decodeRolesBody[apigen.AdminRoleList](t, body)
	systemRoles := make(map[string]apigen.AdminRole)
	for _, role := range roleList.Items {
		if role.IsSystem {
			systemRoles[role.Slug] = role
		}
	}
	for _, slug := range []string{"member", "teacher", "moderator", "major_admin", "admin"} {
		if _, ok := systemRoles[slug]; !ok {
			t.Fatalf("system role %q is missing from %+v", slug, roleList.Items)
		}
	}
	if len(systemRoles["admin"].Permissions) == 0 {
		t.Fatal("the admin system role has no permissions")
	}

	res, body = env.do(adminClient, http.MethodGet, "/api/v1/admin/permissions", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /admin/permissions status = %d, body %s", res.StatusCode, body)
	}
	catalog := decodeRolesBody[apigen.AdminPermissionList](t, body)
	permissions := make(map[string]apigen.AdminPermission, len(catalog.Items))
	for _, permission := range catalog.Items {
		permissions[permission.Key] = permission
	}
	if permission, ok := permissions["content.moderate"]; !ok || !permission.Scopable {
		t.Fatalf("content.moderate = %+v, want a scopable catalog entry", permission)
	}
	if permission, ok := permissions["role.manage"]; !ok || permission.Scopable {
		t.Fatalf("role.manage = %+v, want a global-only catalog entry", permission)
	}

	// A plain student must not see the role admin area at all.
	env.createActivatedStudent("1000000091", "siswa.role", usersTestPassword)
	studentClient, _ := env.loginAs("siswa.role")

	res, body = env.do(studentClient, http.MethodGet, "/api/v1/admin/roles", nil, "", nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")

	res, body = env.do(studentClient, http.MethodGet, "/api/v1/admin/permissions", nil, "", nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")
}

func TestRolesHTTP_AdminAssignsGlobalModeratorAndRevokes(t *testing.T) {
	env := newRolesEnv(t)

	env.createAdmin("admin.assign", "Admin Pemberi")
	teacherID := env.createTeacher("guru.target", "Guru Target")

	// The teacher signs in before the assignment so the cached principal must
	// be invalidated by the service.
	teacherClient, _ := env.loginAs("guru.target")
	res, body := env.do(teacherClient, http.MethodGet, "/api/v1/auth/session", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("teacher session status = %d, body %s", res.StatusCode, body)
	}
	if grant := roleGrant(decodeSession(t, body), "content.moderate"); grant != nil {
		t.Fatalf("teacher already holds content.moderate: %+v", grant)
	}

	adminClient, adminCSRF := env.loginAs("admin.assign")

	res, body = env.do(adminClient, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/users/%s/roles", teacherID),
		map[string]any{"roleSlug": "moderator"}, adminCSRF, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("assign moderator status = %d, body %s", res.StatusCode, body)
	}
	assignment := decodeRolesBody[apigen.AdminRoleAssignment](t, body)
	if assignment.RoleSlug != "moderator" {
		t.Fatalf("roleSlug = %q, want moderator", assignment.RoleSlug)
	}
	if assignment.MajorId != nil {
		t.Fatalf("majorId = %s, want a global assignment", assignment.MajorId)
	}
	if assignment.AssignedAt == nil || assignment.AssignedAt.IsZero() {
		t.Fatalf("assignedAt = %v, want a timestamp", assignment.AssignedAt)
	}

	res, body = env.do(adminClient, http.MethodGet,
		fmt.Sprintf("/api/v1/admin/users/%s/roles", teacherID), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET user roles status = %d, body %s", res.StatusCode, body)
	}
	assignments := decodeRolesBody[apigen.AdminRoleAssignmentList](t, body)
	if len(assignments.Items) != 1 || assignments.Items[0].Id != assignment.Id {
		t.Fatalf("assignments = %+v, want the moderator assignment %s", assignments.Items, assignment.Id)
	}

	// The invalidated cache now resolves the new grant.
	res, body = env.do(teacherClient, http.MethodGet, "/api/v1/auth/session", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("teacher session after assign status = %d, body %s", res.StatusCode, body)
	}
	grant := roleGrant(decodeSession(t, body), "content.moderate")
	if grant == nil {
		t.Fatal("teacher session is missing content.moderate after the assignment")
	}
	if !grant.Global {
		t.Fatalf("content.moderate grant = %+v, want a global grant", grant)
	}

	res, body = env.do(adminClient, http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/user-roles/%s", assignment.Id), nil, adminCSRF, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke status = %d, body %s", res.StatusCode, body)
	}

	res, body = env.do(teacherClient, http.MethodGet, "/api/v1/auth/session", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("teacher session after revoke status = %d, body %s", res.StatusCode, body)
	}
	if grant := roleGrant(decodeSession(t, body), "content.moderate"); grant != nil {
		t.Fatalf("teacher still holds content.moderate after the revoke: %+v", grant)
	}
}

func TestRolesHTTP_MajorAdminAssignsOnlyWithinOwnScope(t *testing.T) {
	env := newRolesEnv(t)

	env.createMajorAdmin("admin.rpl", "Kaprog RPL", "rpl")
	teacherID := env.createTeacher("guru.scope", "Guru Scope")
	rplID := env.majorID("rpl")
	tkjID := env.majorID("tkj")

	majorClient, majorCSRF := env.loginAs("admin.rpl")
	path := fmt.Sprintf("/api/v1/admin/users/%s/roles", teacherID)

	res, body := env.do(majorClient, http.MethodPost, path,
		map[string]any{"roleSlug": "moderator", "majorId": rplID.String()}, majorCSRF, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("assign moderator in RPL status = %d, body %s", res.StatusCode, body)
	}
	assignment := decodeRolesBody[apigen.AdminRoleAssignment](t, body)
	if assignment.MajorId == nil || *assignment.MajorId != rplID {
		t.Fatalf("majorId = %v, want %s", assignment.MajorId, rplID)
	}

	// Another major is outside the actor's scope.
	res, body = env.do(majorClient, http.MethodPost, path,
		map[string]any{"roleSlug": "moderator", "majorId": tkjID.String()}, majorCSRF, nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")

	// A global role assignment is outside the actor's scope.
	res, body = env.do(majorClient, http.MethodPost, path,
		map[string]any{"roleSlug": "admin"}, majorCSRF, nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")

	// Even inside the scope the actor cannot hand out permissions it does not
	// hold at that scope.
	res, body = env.do(majorClient, http.MethodPost, path,
		map[string]any{"roleSlug": "admin", "majorId": rplID.String()}, majorCSRF, nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")
}

func TestRolesHTTP_AssigningImplicitRoleReturnsValidationError(t *testing.T) {
	env := newRolesEnv(t)

	env.createAdmin("admin.implicit", "Admin Implisit")
	teacherID := env.createTeacher("guru.implicit", "Guru Implisit")
	adminClient, adminCSRF := env.loginAs("admin.implicit")

	res, body := env.do(adminClient, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/users/%s/roles", teacherID),
		map[string]any{"roleSlug": "member"}, adminCSRF, nil)
	requireFieldProblem(t, res, body, http.StatusUnprocessableEntity, "VALIDATION_FAILED", "role", "implicit")
}

func TestRolesHTTP_AdminCreatesCustomRoleAndCannotEditSystemRolePermissions(t *testing.T) {
	env := newRolesEnv(t)

	env.createAdmin("admin.custom", "Admin Kustom")
	adminClient, adminCSRF := env.loginAs("admin.custom")

	res, body := env.do(adminClient, http.MethodPost, "/api/v1/admin/roles", map[string]any{
		"slug":        "kurator_rpl",
		"name":        "Kurator RPL",
		"permissions": []string{"content.moderate"},
	}, adminCSRF, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /admin/roles status = %d, body %s", res.StatusCode, body)
	}
	created := decodeRolesBody[apigen.AdminRole](t, body)
	if created.Slug != "kurator_rpl" {
		t.Fatalf("slug = %q, want kurator_rpl", created.Slug)
	}
	if created.IsSystem {
		t.Fatal("a newly created role must not be a system role")
	}
	if len(created.Permissions) != 1 || created.Permissions[0] != "content.moderate" {
		t.Fatalf("permissions = %v, want [content.moderate]", created.Permissions)
	}

	res, body = env.do(adminClient, http.MethodGet, "/api/v1/admin/roles", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /admin/roles status = %d, body %s", res.StatusCode, body)
	}
	roleList := decodeRolesBody[apigen.AdminRoleList](t, body)
	var adminRoleID uuid.UUID
	for _, role := range roleList.Items {
		if role.Slug == "admin" && role.IsSystem {
			adminRoleID = role.Id
		}
	}
	if adminRoleID == uuid.Nil {
		t.Fatal("the admin system role is missing from the list")
	}

	res, body = env.do(adminClient, http.MethodPut,
		fmt.Sprintf("/api/v1/admin/roles/%s/permissions", adminRoleID),
		map[string]any{"permissions": []string{"content.moderate"}}, adminCSRF, nil)
	requireProblem(t, res, body, http.StatusConflict, "CONFLICT")
}
