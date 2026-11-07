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

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/academic"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/password"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

type academicEnv struct {
	*flowEnv
}

// newAcademicEnv builds the real router with auth and academic services on a
// seeded database, served over TLS so __Host- cookies are accepted.
func newAcademicEnv(t *testing.T) *academicEnv {
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
		Academic:       academic.NewService(st),
		Limiter:        ratelimit.New(rdb),
		AllowedOrigins: []string{"http://localhost:3020"},
	})

	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	return &academicEnv{flowEnv: &flowEnv{t: t, pool: pool, server: server, hasher: hasher}}
}

func academicStringPtr(value string) *string {
	return &value
}

func (e *academicEnv) assignGlobalRole(userID uuid.UUID, roleSlug string) {
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

func (e *academicEnv) assignMajorRole(userID uuid.UUID, roleSlug, majorCode string) {
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

func (e *academicEnv) majorID(code string) uuid.UUID {
	e.t.Helper()

	var id uuid.UUID
	if err := e.pool.QueryRow(context.Background(),
		`SELECT id FROM majors WHERE code = $1`, code,
	).Scan(&id); err != nil {
		e.t.Fatalf("load major %s: %v", code, err)
	}
	return id
}

// loginAs signs in with username/password and returns the client plus the
// rotated CSRF token.
func (e *academicEnv) loginAs(username, plain string) (*http.Client, string) {
	e.t.Helper()

	c := e.client()
	csrf := e.fetchCSRF(c)
	res, body := e.login(c, username, plain, csrf)
	if res.StatusCode != http.StatusOK {
		e.t.Fatalf("login %s status = %d, body %s", username, res.StatusCode, body)
	}
	return c, e.csrfFromJar(c)
}

func decodeAcademicBody[T any](t *testing.T, body []byte) T {
	t.Helper()

	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode response: %v (body %s)", err, body)
	}
	return out
}

func containsMajor(list apigen.MajorList, id uuid.UUID) bool {
	for _, major := range list.Items {
		if major.Id == id {
			return true
		}
	}
	return false
}

func TestAcademicHTTP_PublicMajorsListsActiveOnly(t *testing.T) {
	env := newAcademicEnv(t)
	c := env.client()

	res, body := env.do(c, http.MethodGet, "/api/v1/majors", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /majors status = %d, body %s", res.StatusCode, body)
	}
	if got := res.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	list := decodeAcademicBody[apigen.MajorList](t, body)
	if len(list.Items) != 9 {
		t.Fatalf("majors = %d, want 9", len(list.Items))
	}
	for _, major := range list.Items {
		if !major.IsActive {
			t.Errorf("major %s is inactive on the public list", major.Code)
		}
	}
}

func TestAcademicHTTP_AdminManagesMajorsYearsAndClasses(t *testing.T) {
	env := newAcademicEnv(t)
	now := time.Now().UTC()

	adminID := env.createUser(flowUserSpec{
		accountType: "staff",
		username:    academicStringPtr("admin.akademik"),
		displayName: "Admin Akademik",
		password:    "sandi-uji-123",
		activatedAt: &now,
	})
	env.assignGlobalRole(adminID, "admin")

	majorAdminID := env.createUser(flowUserSpec{
		accountType: "teacher",
		username:    academicStringPtr("admin.rpl"),
		displayName: "Kaprog RPL",
		password:    "sandi-uji-123",
		activatedAt: &now,
	})
	env.assignMajorRole(majorAdminID, "major_admin", "rpl")

	rplID := env.majorID("rpl")
	tkjID := env.majorID("tkj")

	adminClient, adminCSRF := env.loginAs("admin.akademik", "sandi-uji-123")

	res, body := env.do(adminClient, http.MethodPost, "/api/v1/admin/majors", map[string]any{
		"code":      "uji",
		"name":      "Jurusan Uji",
		"shortName": "UJI",
		"sortOrder": 10,
	}, adminCSRF, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /admin/majors status = %d, body %s", res.StatusCode, body)
	}
	createdMajor := decodeAcademicBody[apigen.Major](t, body)
	if createdMajor.Code != "uji" || createdMajor.Name != "Jurusan Uji" {
		t.Fatalf("created major = %+v", createdMajor)
	}
	if !createdMajor.IsActive {
		t.Error("created major is not active")
	}

	res, body = env.do(adminClient, http.MethodGet, "/api/v1/admin/majors", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /admin/majors status = %d, body %s", res.StatusCode, body)
	}
	adminMajors := decodeAcademicBody[apigen.MajorList](t, body)
	if len(adminMajors.Items) != 10 {
		t.Fatalf("admin majors = %d, want 10", len(adminMajors.Items))
	}
	if !containsMajor(adminMajors, createdMajor.Id) {
		t.Fatal("created major is missing from the admin list")
	}

	res, body = env.do(adminClient, http.MethodPatch,
		fmt.Sprintf("/api/v1/admin/majors/%s", createdMajor.Id),
		map[string]any{"isActive": false}, adminCSRF, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH /admin/majors/{id} status = %d, body %s", res.StatusCode, body)
	}
	if updated := decodeAcademicBody[apigen.Major](t, body); updated.IsActive {
		t.Error("major is still active after deactivation")
	}

	res, body = env.do(env.client(), http.MethodGet, "/api/v1/majors", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /majors status = %d, body %s", res.StatusCode, body)
	}
	publicMajors := decodeAcademicBody[apigen.MajorList](t, body)
	if containsMajor(publicMajors, createdMajor.Id) {
		t.Fatal("inactive major appears on the public list")
	}
	if len(publicMajors.Items) != 9 {
		t.Fatalf("public majors = %d, want 9", len(publicMajors.Items))
	}

	res, body = env.do(adminClient, http.MethodPost, "/api/v1/admin/academic-years", map[string]any{
		"startYear": 2026,
		"startsOn":  "2026-07-13",
		"endsOn":    "2027-06-30",
	}, adminCSRF, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /admin/academic-years status = %d, body %s", res.StatusCode, body)
	}
	year := decodeAcademicBody[apigen.AcademicYear](t, body)
	if year.Label != "2026/2027" {
		t.Errorf("label = %q, want 2026/2027", year.Label)
	}
	if year.Slug != "2026-2027" {
		t.Errorf("slug = %q, want 2026-2027", year.Slug)
	}
	if got := year.StartsOn.Time.Format("2006-01-02"); got != "2026-07-13" {
		t.Errorf("startsOn = %q, want 2026-07-13", got)
	}
	if got := year.EndsOn.Time.Format("2006-01-02"); got != "2027-06-30" {
		t.Errorf("endsOn = %q, want 2027-06-30", got)
	}

	res, body = env.do(adminClient, http.MethodPost,
		fmt.Sprintf("/api/v1/admin/academic-years/%s/set-current", year.Id), nil, adminCSRF, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("set-current status = %d, body %s", res.StatusCode, body)
	}

	res, body = env.do(adminClient, http.MethodGet, "/api/v1/admin/academic-years", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /admin/academic-years status = %d, body %s", res.StatusCode, body)
	}
	years := decodeAcademicBody[apigen.AcademicYearList](t, body)
	if len(years.Items) != 1 || years.Items[0].Id != year.Id || !years.Items[0].IsCurrent {
		t.Fatalf("academic years = %+v, want the created year marked current", years.Items)
	}

	majorClient, majorCSRF := env.loginAs("admin.rpl", "sandi-uji-123")

	classBody := map[string]any{
		"academicYearId": year.Id.String(),
		"majorId":        rplID.String(),
		"grade":          11,
		"section":        "b",
	}
	res, body = env.do(majorClient, http.MethodPost, "/api/v1/admin/classes", classBody, majorCSRF, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /admin/classes status = %d, body %s", res.StatusCode, body)
	}
	class := decodeAcademicBody[apigen.Class](t, body)
	if class.Name != "XI RPL B" {
		t.Errorf("class name = %q, want XI RPL B", class.Name)
	}
	if class.Slug != "xi-rpl-b" {
		t.Errorf("class slug = %q, want xi-rpl-b", class.Slug)
	}
	if class.MajorCode != "rpl" {
		t.Errorf("class majorCode = %q, want rpl", class.MajorCode)
	}
	if class.Grade != 11 || class.Section != "B" {
		t.Errorf("class grade/section = %d/%q, want 11/B", class.Grade, class.Section)
	}

	// A major admin cannot create classes for another major.
	res, body = env.do(majorClient, http.MethodPost, "/api/v1/admin/classes", map[string]any{
		"academicYearId": year.Id.String(),
		"majorId":        tkjID.String(),
		"grade":          11,
		"section":        "b",
	}, majorCSRF, nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")

	res, body = env.do(adminClient, http.MethodGet,
		"/api/v1/admin/classes?majorId="+rplID.String(), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /admin/classes status = %d, body %s", res.StatusCode, body)
	}
	classes := decodeAcademicBody[apigen.ClassList](t, body)
	if len(classes.Items) != 1 || classes.Items[0].Id != class.Id {
		t.Fatalf("classes = %+v, want only the created class", classes.Items)
	}

	// The same class cannot be created twice in one academic year.
	res, body = env.do(majorClient, http.MethodPost, "/api/v1/admin/classes", classBody, majorCSRF, nil)
	requireProblem(t, res, body, http.StatusConflict, "CONFLICT")

	// Grade 13 is outside the enum; the request must be rejected as a problem.
	res, body = env.do(majorClient, http.MethodPost, "/api/v1/admin/classes", map[string]any{
		"academicYearId": year.Id.String(),
		"majorId":        rplID.String(),
		"grade":          13,
		"section":        "c",
	}, majorCSRF, nil)
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		t.Fatalf("grade 13 status = %d, want a client error (body %s)", res.StatusCode, body)
	}
	if got := res.Header.Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("grade 13 Content-Type = %q, want application/problem+json", got)
	}
	var invalidProblem struct {
		Code   string `json:"code"`
		Status int    `json:"status"`
	}
	if err := json.Unmarshal(body, &invalidProblem); err != nil {
		t.Fatalf("decode grade 13 problem: %v (body %s)", err, body)
	}
	if invalidProblem.Code == "" || invalidProblem.Status != res.StatusCode {
		t.Fatalf("grade 13 problem = %+v, want status %d with a code", invalidProblem, res.StatusCode)
	}

	// Renaming the section recomputes the class name and slug.
	res, body = env.do(majorClient, http.MethodPatch,
		fmt.Sprintf("/api/v1/admin/classes/%s", class.Id),
		map[string]any{"section": "a"}, majorCSRF, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PATCH /admin/classes/{id} status = %d, body %s", res.StatusCode, body)
	}
	renamed := decodeAcademicBody[apigen.Class](t, body)
	if renamed.Name != "XI RPL A" || renamed.Slug != "xi-rpl-a" {
		t.Fatalf("renamed class = %q/%q, want XI RPL A/xi-rpl-a", renamed.Name, renamed.Slug)
	}

	res, body = env.do(majorClient, http.MethodDelete,
		fmt.Sprintf("/api/v1/admin/classes/%s", class.Id), nil, majorCSRF, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE /admin/classes/{id} status = %d, body %s", res.StatusCode, body)
	}

	res, body = env.do(adminClient, http.MethodGet,
		"/api/v1/admin/classes?majorId="+rplID.String(), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /admin/classes after delete status = %d, body %s", res.StatusCode, body)
	}
	if remaining := decodeAcademicBody[apigen.ClassList](t, body); len(remaining.Items) != 0 {
		t.Fatalf("classes after delete = %d, want 0", len(remaining.Items))
	}
}

func TestAcademicHTTP_StudentAndAnonymousCannotSeeAdminAcademic(t *testing.T) {
	env := newAcademicEnv(t)

	env.createActivatedStudent("1000000010", "siswa.akademik", "sandi-uji-123")
	studentClient, studentCSRF := env.loginAs("siswa.akademik", "sandi-uji-123")

	res, body := env.do(studentClient, http.MethodGet, "/api/v1/admin/majors", nil, "", nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")

	res, body = env.do(studentClient, http.MethodPost, "/api/v1/admin/majors", map[string]any{
		"code":      "nakal",
		"name":      "Jurusan Nakal",
		"shortName": "NKL",
	}, studentCSRF, nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")

	res, body = env.do(studentClient, http.MethodGet, "/api/v1/admin/academic-years", nil, "", nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")

	res, body = env.do(studentClient, http.MethodGet, "/api/v1/admin/classes", nil, "", nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")

	// Anonymous requests carry a valid CSRF token but no session.
	anon := env.client()
	anonCSRF := env.fetchCSRF(anon)
	res, body = env.do(anon, http.MethodPost, "/api/v1/admin/majors", map[string]any{
		"code":      "anon",
		"name":      "Jurusan Anonim",
		"shortName": "ANM",
	}, anonCSRF, nil)
	requireProblem(t, res, body, http.StatusUnauthorized, "UNAUTHENTICATED")
}
