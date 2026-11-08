package http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/audit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/users"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/password"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

// newAuditEnv builds the real router with auth, user administration, and audit
// services on a seeded database, served over TLS so __Host- cookies are
// accepted.
func newAuditEnv(t *testing.T) *usersEnv {
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
		Audit:          audit.NewService(st),
		Limiter:        ratelimit.New(rdb),
		AllowedOrigins: []string{"http://localhost:3020"},
	})

	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	env := &usersEnv{flowEnv: &flowEnv{t: t, pool: pool, server: server, hasher: hasher}}
	env.seedAcademics()
	return env
}

// createStudent posts one student through the admin API and returns the
// created account.
func createStudentThroughAPI(
	t *testing.T,
	env *usersEnv,
	client *http.Client,
	csrf, nisn, displayName string,
	classID uuid.UUID,
) apigen.AdminUserCreated {
	t.Helper()

	res, body := env.do(client, http.MethodPost, "/api/v1/admin/users", map[string]any{
		"accountType": "student",
		"displayName": displayName,
		"nisn":        nisn,
		"classId":     classID.String(),
	}, csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /admin/users status = %d, body %s", res.StatusCode, body)
	}
	return decodeUsersBody[apigen.AdminUserCreated](t, body)
}

func decodeAuditPage(t *testing.T, body []byte) apigen.AuditLogPage {
	t.Helper()

	return decodeUsersBody[apigen.AuditLogPage](t, body)
}

func TestAuditHTTP_GlobalAdminSeesEntriesWithActorIdentity(t *testing.T) {
	env := newAuditEnv(t)

	adminID := env.createAdmin("admin.audit", "Admin Audit", "admin")
	client, csrf := env.loginAs("admin.audit", usersTestPassword)

	created := createStudentThroughAPI(t, env, client, csrf,
		"1000000101", "Siswa Audit RPL", env.rplClassID)

	res, body := env.do(client, http.MethodGet, "/api/v1/admin/audit-logs", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /admin/audit-logs status = %d, body %s", res.StatusCode, body)
	}
	page := decodeAuditPage(t, body)
	if len(page.Items) != 1 {
		t.Fatalf("items = %d, want 1 (%s)", len(page.Items), body)
	}

	item := page.Items[0]
	if item.Action != "user.create" {
		t.Errorf("action = %q, want user.create", item.Action)
	}
	if item.ActorId == nil || uuid.UUID(*item.ActorId) != adminID {
		t.Errorf("actorId = %v, want %s", item.ActorId, adminID)
	}
	if item.ActorName == nil || *item.ActorName != "Admin Audit" {
		t.Errorf("actorName = %v, want Admin Audit", item.ActorName)
	}
	if item.ActorUsername == nil || *item.ActorUsername != "admin.audit" {
		t.Errorf("actorUsername = %v, want admin.audit", item.ActorUsername)
	}
	if item.ActorIp == nil {
		t.Error("actorIp is nil, want the request IP for a global audit admin")
	}
	if item.TargetId == nil || uuid.UUID(*item.TargetId) != created.User.Id {
		t.Errorf("targetId = %v, want %s", item.TargetId, created.User.Id)
	}
	if item.ScopeMajorId == nil || uuid.UUID(*item.ScopeMajorId) != env.majorID("rpl") {
		t.Errorf("scopeMajorId = %v, want the RPL major", item.ScopeMajorId)
	}
	if item.Data["account_type"] != "student" {
		t.Errorf("data.account_type = %v, want student", item.Data["account_type"])
	}
}

func TestAuditHTTP_MajorAdminSeesOnlyOwnMajorAndNoIP(t *testing.T) {
	env := newAuditEnv(t)

	env.createAdmin("admin.global", "Admin Global", "admin")
	env.createMajorAdmin("admin.rpl", "Kaprog RPL", "rpl")
	adminClient, adminCSRF := env.loginAs("admin.global", usersTestPassword)

	createStudentThroughAPI(t, env, adminClient, adminCSRF,
		"1000000102", "Siswa RPL", env.rplClassID)
	createStudentThroughAPI(t, env, adminClient, adminCSRF,
		"1000000103", "Siswa TKJ", env.tkjClassID)

	// A staff account has no major scope, so its entry must stay hidden from
	// scoped admins.
	res, body := env.do(adminClient, http.MethodPost, "/api/v1/admin/users", map[string]any{
		"accountType": "staff",
		"displayName": "Staf tanpa jurusan",
	}, adminCSRF, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST staff status = %d, body %s", res.StatusCode, body)
	}

	rplAdmin, _ := env.loginAs("admin.rpl", usersTestPassword)
	res, body = env.do(rplAdmin, http.MethodGet, "/api/v1/admin/audit-logs", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /admin/audit-logs status = %d, body %s", res.StatusCode, body)
	}
	page := decodeAuditPage(t, body)
	if len(page.Items) == 0 {
		t.Fatal("the RPL admin sees no audit entries")
	}

	rplMajorID := env.majorID("rpl")
	for _, item := range page.Items {
		if item.ScopeMajorId == nil || uuid.UUID(*item.ScopeMajorId) != rplMajorID {
			t.Errorf("action %q has scopeMajorId %v, want the RPL major", item.Action, item.ScopeMajorId)
		}
		if item.ActorIp != nil {
			t.Errorf("actorIp = %v, want nil for a scoped audit admin", item.ActorIp)
		}
	}
}

func TestAuditHTTP_StudentForbidden(t *testing.T) {
	env := newAuditEnv(t)

	env.createActivatedStudent("1000000104", "siswa.audit", usersTestPassword)
	client, _ := env.loginAs("siswa.audit", usersTestPassword)

	res, body := env.do(client, http.MethodGet, "/api/v1/admin/audit-logs", nil, "", nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")
}

func TestAuditHTTP_ActionPrefixFilter(t *testing.T) {
	env := newAuditEnv(t)

	env.createAdmin("admin.prefix", "Admin Prefix", "admin")
	client, csrf := env.loginAs("admin.prefix", usersTestPassword)
	createStudentThroughAPI(t, env, client, csrf,
		"1000000105", "Siswa Filter", env.rplClassID)

	res, body := env.do(client, http.MethodGet,
		"/api/v1/admin/audit-logs?actionPrefix=user.", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET with actionPrefix status = %d, body %s", res.StatusCode, body)
	}
	page := decodeAuditPage(t, body)
	if len(page.Items) == 0 {
		t.Fatal("the user. prefix returned no entries")
	}
	for _, item := range page.Items {
		if !strings.HasPrefix(item.Action, "user.") {
			t.Errorf("action = %q, want the user. prefix", item.Action)
		}
	}

	res, body = env.do(client, http.MethodGet,
		"/api/v1/admin/audit-logs?actionPrefix=auth.", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET with unused prefix status = %d, body %s", res.StatusCode, body)
	}
	if page := decodeAuditPage(t, body); len(page.Items) != 0 {
		t.Fatalf("the auth. prefix returned %d entries, want 0", len(page.Items))
	}
}

func TestAuditHTTP_PaginationWithLimitOne(t *testing.T) {
	env := newAuditEnv(t)

	env.createAdmin("admin.page", "Admin Halaman", "admin")
	client, csrf := env.loginAs("admin.page", usersTestPassword)
	createStudentThroughAPI(t, env, client, csrf,
		"1000000106", "Siswa Halaman Satu", env.rplClassID)
	createStudentThroughAPI(t, env, client, csrf,
		"1000000107", "Siswa Halaman Dua", env.rplClassID)

	res, body := env.do(client, http.MethodGet, "/api/v1/admin/audit-logs?limit=1", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET first page status = %d, body %s", res.StatusCode, body)
	}
	first := decodeAuditPage(t, body)
	if len(first.Items) != 1 {
		t.Fatalf("first page has %d items, want 1", len(first.Items))
	}
	if first.NextCursor == nil || *first.NextCursor == "" {
		t.Fatal("first page nextCursor is empty")
	}

	res, body = env.do(client, http.MethodGet,
		"/api/v1/admin/audit-logs?limit=1&cursor="+url.QueryEscape(*first.NextCursor), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET second page status = %d, body %s", res.StatusCode, body)
	}
	second := decodeAuditPage(t, body)
	if len(second.Items) != 1 {
		t.Fatalf("second page has %d items, want 1", len(second.Items))
	}
	if second.Items[0].Id == first.Items[0].Id {
		t.Fatalf("second page repeats id %d", second.Items[0].Id)
	}
}
