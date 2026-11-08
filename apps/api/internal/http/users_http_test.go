package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/users"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/password"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/secretbox"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

const usersTestPassword = "sandi-uji-123"

// usersTestVaultSecret keys the initial-password vault in tests.
var usersTestVaultSecret = []byte("test-secret-key-for-initial-password-vault")

// initialPasswordPattern is the format produced by securetoken.ReadablePassword.
var initialPasswordPattern = regexp.MustCompile(`^[2-9a-hj-km-np-z]{4}-[2-9a-hj-km-np-z]{4}-[2-9a-hj-km-np-z]{4}$`)

type usersEnv struct {
	*flowEnv

	yearID     uuid.UUID
	rplClassID uuid.UUID
	tkjClassID uuid.UUID
}

// newUsersEnv builds the real router with auth and user administration
// services on a seeded database, served over TLS so __Host- cookies are
// accepted.
func newUsersEnv(t *testing.T) *usersEnv {
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
		Limiter:        ratelimit.New(rdb),
		AllowedOrigins: []string{"http://localhost:3020"},
	})

	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	env := &usersEnv{flowEnv: &flowEnv{t: t, pool: pool, server: server, hasher: hasher}}
	env.seedAcademics()
	return env
}

// seedAcademics inserts one academic year and an RPL class and a TKJ class.
func (e *usersEnv) seedAcademics() {
	e.t.Helper()

	err := e.pool.QueryRow(context.Background(), `
		INSERT INTO academic_years (start_year, starts_on, ends_on)
		VALUES (2026, '2026-07-01', '2027-06-30')
		RETURNING id`).Scan(&e.yearID)
	if err != nil {
		e.t.Fatalf("create academic year: %v", err)
	}

	e.rplClassID = e.insertClass(e.majorID("rpl"), 10, "A", "X RPL A", "x-rpl-a")
	e.tkjClassID = e.insertClass(e.majorID("tkj"), 10, "A", "X TKJ A", "x-tkj-a")
}

func (e *usersEnv) insertClass(majorID uuid.UUID, grade int, section, name, slug string) uuid.UUID {
	e.t.Helper()

	var id uuid.UUID
	err := e.pool.QueryRow(context.Background(), `
		INSERT INTO classes (academic_year_id, major_id, grade, section, name, slug)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		e.yearID, majorID, grade, section, name, slug,
	).Scan(&id)
	if err != nil {
		e.t.Fatalf("create class %s: %v", name, err)
	}

	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO groups (kind, class_id, status) VALUES ('class', $1, 'active')`, id); err != nil {
		e.t.Fatalf("create class group %s: %v", name, err)
	}
	return id
}

func (e *usersEnv) majorID(code string) uuid.UUID {
	e.t.Helper()

	var id uuid.UUID
	if err := e.pool.QueryRow(context.Background(),
		`SELECT id FROM majors WHERE code = $1`, code,
	).Scan(&id); err != nil {
		e.t.Fatalf("load major %s: %v", code, err)
	}
	return id
}

func (e *usersEnv) assignGlobalRole(userID uuid.UUID, roleSlug string) {
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

func (e *usersEnv) assignMajorRole(userID uuid.UUID, roleSlug, majorCode string) {
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

// createAdmin inserts an activated staff account with the requested global role.
func (e *usersEnv) createAdmin(username, displayName, roleSlug string) uuid.UUID {
	e.t.Helper()

	now := time.Now().UTC()
	id := e.createUser(flowUserSpec{
		accountType: "staff",
		username:    &username,
		displayName: displayName,
		password:    usersTestPassword,
		activatedAt: &now,
	})
	e.assignGlobalRole(id, roleSlug)
	return id
}

// createMajorAdmin inserts an activated teacher account with a scoped role.
func (e *usersEnv) createMajorAdmin(username, displayName, majorCode string) uuid.UUID {
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

// createStudentInClass inserts an activated student with an enrollment.
func (e *usersEnv) createStudentInClass(nisn, username, displayName string, classID uuid.UUID) uuid.UUID {
	e.t.Helper()

	now := time.Now().UTC()
	id := e.createUser(flowUserSpec{
		nisn:        &nisn,
		username:    &username,
		displayName: displayName,
		password:    usersTestPassword,
		activatedAt: &now,
	})

	var yearID uuid.UUID
	if err := e.pool.QueryRow(context.Background(),
		`SELECT academic_year_id FROM classes WHERE id = $1`, classID,
	).Scan(&yearID); err != nil {
		e.t.Fatalf("load class academic year: %v", err)
	}
	if _, err := e.pool.Exec(context.Background(), `
		INSERT INTO enrollments (user_id, class_id, academic_year_id, source)
		VALUES ($1, $2, $3, 'admin')`,
		id, classID, yearID,
	); err != nil {
		e.t.Fatalf("create enrollment: %v", err)
	}
	return id
}

// loginAs signs in and returns the client plus the rotated CSRF token.
func (e *usersEnv) loginAs(identifier, plain string) (*http.Client, string) {
	e.t.Helper()

	c := e.client()
	csrf := e.fetchCSRF(c)
	res, body := e.login(c, identifier, plain, csrf)
	if res.StatusCode != http.StatusOK {
		e.t.Fatalf("login %s status = %d, body %s", identifier, res.StatusCode, body)
	}
	return c, e.csrfFromJar(c)
}

// newUsersTestBox builds the secretbox used by the user administration service
// in tests.
func newUsersTestBox(t *testing.T) *secretbox.Box {
	t.Helper()

	box, err := secretbox.New(usersTestVaultSecret)
	if err != nil {
		t.Fatalf("secretbox.New: %v", err)
	}
	return box
}

func decodeUsersBody[T any](t *testing.T, body []byte) T {
	t.Helper()

	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode response: %v (body %s)", err, body)
	}
	return out
}

func pageHasUser(page apigen.AdminUserPage, id uuid.UUID) bool {
	for _, item := range page.Items {
		if item.Id == id {
			return true
		}
	}
	return false
}

func hasGrant(session apigen.Session, permission string) bool {
	for _, grant := range session.Grants {
		if grant.Permission == permission {
			return true
		}
	}
	return false
}

func requireInitialPassword(t *testing.T, password string) {
	t.Helper()

	if !initialPasswordPattern.MatchString(password) {
		t.Fatalf("initial password = %q, want the readable three-group format", password)
	}
}

func TestUsersHTTP_AdminCreatesStudentWithInitialPassword(t *testing.T) {
	env := newUsersEnv(t)

	env.createAdmin("admin.pengguna", "Admin Pengguna", "admin")
	adminClient, adminCSRF := env.loginAs("admin.pengguna", usersTestPassword)

	res, body := env.do(adminClient, http.MethodPost, "/api/v1/admin/users", map[string]any{
		"accountType": "student",
		"displayName": "Siswa Baru RPL",
		"nisn":        "1000000001",
		"classId":     env.rplClassID.String(),
	}, adminCSRF, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /admin/users status = %d, body %s", res.StatusCode, body)
	}
	if got := res.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}

	created := decodeUsersBody[apigen.AdminUserCreated](t, body)
	requireInitialPassword(t, created.InitialPassword)
	if created.User.AccountType != apigen.AccountTypeStudent {
		t.Errorf("accountType = %q, want student", created.User.AccountType)
	}
	if created.User.Status != apigen.AccountStatusActive {
		t.Errorf("status = %q, want active", created.User.Status)
	}
	if created.User.Activated {
		t.Error("activated = true, want false for a new account")
	}
	if created.User.PasswordKind != apigen.AdminUserDetailPasswordKindInitial {
		t.Errorf("passwordKind = %q, want initial", created.User.PasswordKind)
	}
	if created.User.CurrentClass == nil || created.User.CurrentClass.Id != env.rplClassID {
		t.Fatalf("currentClass = %+v, want the RPL class %s", created.User.CurrentClass, env.rplClassID)
	}
	if len(created.User.Enrollments) != 1 || created.User.Enrollments[0].ClassId != env.rplClassID {
		t.Fatalf("enrollments = %+v, want one RPL enrollment", created.User.Enrollments)
	}

	// The new student can sign in with the initial password and must activate.
	student := env.client()
	studentCSRF := env.fetchCSRF(student)
	res, body = env.login(student, "1000000001", created.InitialPassword, studentCSRF)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login with initial password status = %d, body %s", res.StatusCode, body)
	}
	session := decodeSession(t, body)
	if session.NextStep != apigen.NextStepActivate {
		t.Fatalf("nextStep = %q, want activate", session.NextStep)
	}
}

func TestUsersHTTP_MajorAdminSeesOnlyOwnMajor(t *testing.T) {
	env := newUsersEnv(t)

	env.createAdmin("admin.global", "Admin Global", "admin")
	env.createMajorAdmin("admin.rpl", "Kaprog RPL", "rpl")

	rplStudentID := env.createStudentInClass("1000000002", "siswa.rpl", "Siswa RPL", env.rplClassID)
	tkjStudentID := env.createStudentInClass("1000000003", "siswa.tkj", "Siswa TKJ", env.tkjClassID)

	majorClient, _ := env.loginAs("admin.rpl", usersTestPassword)

	res, body := env.do(majorClient, http.MethodGet, "/api/v1/admin/users", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /admin/users status = %d, body %s", res.StatusCode, body)
	}
	page := decodeUsersBody[apigen.AdminUserPage](t, body)
	if !pageHasUser(page, rplStudentID) {
		t.Fatal("the RPL student is missing from the major admin list")
	}
	if pageHasUser(page, tkjStudentID) {
		t.Fatal("the TKJ student appears in the RPL major admin list")
	}

	res, body = env.do(majorClient, http.MethodGet,
		fmt.Sprintf("/api/v1/admin/users/%s", rplStudentID), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET RPL user status = %d, body %s", res.StatusCode, body)
	}

	res, body = env.do(majorClient, http.MethodGet,
		fmt.Sprintf("/api/v1/admin/users/%s", tkjStudentID), nil, "", nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")

	// A plain student must not see the admin area at all.
	env.createActivatedStudent("1000000004", "siswa.biasa", usersTestPassword)
	studentClient, _ := env.loginAs("siswa.biasa", usersTestPassword)

	res, body = env.do(studentClient, http.MethodGet, "/api/v1/admin/users", nil, "", nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")

	res, body = env.do(studentClient, http.MethodGet,
		fmt.Sprintf("/api/v1/admin/users/%s", rplStudentID), nil, "", nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")
}

func TestUsersHTTP_SearchAndAccountTypeFilter(t *testing.T) {
	env := newUsersEnv(t)

	env.createAdmin("admin.cari", "Admin Pencari", "admin")
	adminClient, adminCSRF := env.loginAs("admin.cari", usersTestPassword)

	createStudent := func(nisn, displayName string) {
		t.Helper()
		res, body := env.do(adminClient, http.MethodPost, "/api/v1/admin/users", map[string]any{
			"accountType": "student",
			"displayName": displayName,
			"nisn":        nisn,
			"classId":     env.rplClassID.String(),
		}, adminCSRF, nil)
		if res.StatusCode != http.StatusCreated {
			t.Fatalf("create student %q status = %d, body %s", displayName, res.StatusCode, body)
		}
	}

	createStudent("1000000011", "Budi Santoso")
	createStudent("1000000012", "Siti Aminah")

	res, body := env.do(adminClient, http.MethodPost, "/api/v1/admin/users", map[string]any{
		"accountType":     "teacher",
		"displayName":     "Guru Uji",
		"staffNumber":     "1987654321",
		"teacherMajorIds": []string{env.majorID("rpl").String()},
	}, adminCSRF, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create teacher status = %d, body %s", res.StatusCode, body)
	}

	res, body = env.do(adminClient, http.MethodGet, "/api/v1/admin/users?q=Budi", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("search status = %d, body %s", res.StatusCode, body)
	}
	found := decodeUsersBody[apigen.AdminUserPage](t, body)
	if len(found.Items) != 1 || found.Items[0].DisplayName != "Budi Santoso" {
		t.Fatalf("search result = %+v, want only Budi Santoso", found.Items)
	}

	res, body = env.do(adminClient, http.MethodGet, "/api/v1/admin/users?accountType=student", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("accountType filter status = %d, body %s", res.StatusCode, body)
	}
	students := decodeUsersBody[apigen.AdminUserPage](t, body)
	if len(students.Items) != 2 {
		t.Fatalf("students = %d, want 2", len(students.Items))
	}
	for _, item := range students.Items {
		if item.AccountType != apigen.AccountTypeStudent {
			t.Errorf("accountType = %q, want student", item.AccountType)
		}
	}

	res, body = env.do(adminClient, http.MethodGet, "/api/v1/admin/users?accountType=teacher", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("teacher filter status = %d, body %s", res.StatusCode, body)
	}
	teachers := decodeUsersBody[apigen.AdminUserPage](t, body)
	if len(teachers.Items) != 1 || teachers.Items[0].DisplayName != "Guru Uji" {
		t.Fatalf("teachers = %+v, want only Guru Uji", teachers.Items)
	}
}

func TestUsersHTTP_ResetPasswordRevokesSessions(t *testing.T) {
	env := newUsersEnv(t)

	env.createAdmin("admin.reset", "Admin Reset", "admin")
	studentID := env.createActivatedStudent("1000000021", "siswa.reset", usersTestPassword)

	studentClient, _ := env.loginAs("siswa.reset", usersTestPassword)
	res, body := env.do(studentClient, http.MethodGet, "/api/v1/auth/session", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("student session status = %d, body %s", res.StatusCode, body)
	}

	adminClient, adminCSRF := env.loginAs("admin.reset", usersTestPassword)

	res, body = env.do(adminClient, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/users/%s/reset-password", studentID), nil, adminCSRF, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("reset password status = %d, body %s", res.StatusCode, body)
	}
	if got := res.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}

	issued := decodeUsersBody[apigen.IssuedPassword](t, body)
	requireInitialPassword(t, issued.Password)
	if time.Until(issued.ExpiresAt) <= 0 {
		t.Fatalf("expiresAt = %s, want a future time", issued.ExpiresAt)
	}

	// The old session is revoked.
	res, body = env.do(studentClient, http.MethodGet, "/api/v1/auth/session", nil, "", nil)
	requireProblem(t, res, body, http.StatusUnauthorized, "UNAUTHENTICATED")

	// The temporary password signs in but requires a password change.
	tempClient := env.client()
	tempCSRF := env.fetchCSRF(tempClient)
	res, body = env.login(tempClient, "siswa.reset", issued.Password, tempCSRF)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login with temporary password status = %d, body %s", res.StatusCode, body)
	}
	session := decodeSession(t, body)
	if session.NextStep != apigen.NextStepChangePassword {
		t.Fatalf("nextStep = %q, want change_password", session.NextStep)
	}
}

func TestUsersHTTP_DeactivateAndReactivate(t *testing.T) {
	env := newUsersEnv(t)

	env.createAdmin("admin.nonaktif", "Admin Nonaktif", "admin")
	studentID := env.createActivatedStudent("1000000031", "siswa.nonaktif", usersTestPassword)

	adminClient, adminCSRF := env.loginAs("admin.nonaktif", usersTestPassword)

	res, body := env.do(adminClient, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/users/%s/deactivate", studentID), nil, adminCSRF, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("deactivate status = %d, body %s", res.StatusCode, body)
	}

	client := env.client()
	csrf := env.fetchCSRF(client)
	res, body = env.login(client, "siswa.nonaktif", usersTestPassword, csrf)
	requireProblem(t, res, body, http.StatusForbidden, "ACCOUNT_INACTIVE")

	res, body = env.do(adminClient, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/users/%s/reactivate", studentID), nil, adminCSRF, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("reactivate status = %d, body %s", res.StatusCode, body)
	}

	client = env.client()
	csrf = env.fetchCSRF(client)
	res, body = env.login(client, "siswa.nonaktif", usersTestPassword, csrf)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login after reactivate status = %d, body %s", res.StatusCode, body)
	}
	if session := decodeSession(t, body); session.NextStep != apigen.NextStepNone {
		t.Fatalf("nextStep = %q, want none", session.NextStep)
	}
}

func TestUsersHTTP_ContentBlockRemovesContentCreateGrant(t *testing.T) {
	env := newUsersEnv(t)

	env.createAdmin("admin.blokir", "Admin Blokir", "admin")
	studentID := env.createActivatedStudent("1000000041", "siswa.blokir", usersTestPassword)

	studentClient, _ := env.loginAs("siswa.blokir", usersTestPassword)
	res, body := env.do(studentClient, http.MethodGet, "/api/v1/auth/session", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("student session status = %d, body %s", res.StatusCode, body)
	}
	if !hasGrant(decodeSession(t, body), "content.create") {
		t.Fatal("student session is missing content.create before the block")
	}

	adminClient, adminCSRF := env.loginAs("admin.blokir", usersTestPassword)

	res, body = env.do(adminClient, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/users/%s/content-block", studentID),
		map[string]any{"reason": "Uji blokir konten"}, adminCSRF, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("content block status = %d, body %s", res.StatusCode, body)
	}

	res, body = env.do(studentClient, http.MethodGet, "/api/v1/auth/session", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("student session after block status = %d, body %s", res.StatusCode, body)
	}
	if hasGrant(decodeSession(t, body), "content.create") {
		t.Fatal("student session still grants content.create after the block")
	}

	res, body = env.do(adminClient, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/users/%s/content-unblock", studentID), nil, adminCSRF, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("content unblock status = %d, body %s", res.StatusCode, body)
	}

	res, body = env.do(studentClient, http.MethodGet, "/api/v1/auth/session", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("student session after unblock status = %d, body %s", res.StatusCode, body)
	}
	if !hasGrant(decodeSession(t, body), "content.create") {
		t.Fatal("student session is missing content.create after the unblock")
	}
}

func TestUsersHTTP_DuplicateNISNReturnsConflict(t *testing.T) {
	env := newUsersEnv(t)

	env.createAdmin("admin.duplikat", "Admin Duplikat", "admin")
	adminClient, adminCSRF := env.loginAs("admin.duplikat", usersTestPassword)

	payload := map[string]any{
		"accountType": "student",
		"displayName": "Siswa Duplikat",
		"nisn":        "1000000051",
		"classId":     env.rplClassID.String(),
	}
	res, body := env.do(adminClient, http.MethodPost, "/api/v1/admin/users", payload, adminCSRF, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("first create status = %d, body %s", res.StatusCode, body)
	}

	res, body = env.do(adminClient, http.MethodPost, "/api/v1/admin/users", payload, adminCSRF, nil)
	requireProblem(t, res, body, http.StatusConflict, "CONFLICT")
}
