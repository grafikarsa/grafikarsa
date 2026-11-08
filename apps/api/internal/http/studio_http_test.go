package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/content"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/password"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

const studioTestPassword = "sandi-studio-123"

type studioEnv struct {
	*flowEnv
	content *content.Service
}

// newStudioEnv builds the real router with auth and the content service on a
// seeded database, served over TLS so __Host- cookies are accepted.
func newStudioEnv(t *testing.T) *studioEnv {
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

	contentService := content.NewService(st)
	handler := NewRouter(RouterDeps{
		Logger:         logger,
		Auth:           authService,
		Content:        contentService,
		Limiter:        ratelimit.New(rdb),
		AllowedOrigins: []string{"http://localhost:3020"},
	})

	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	return &studioEnv{
		flowEnv: &flowEnv{t: t, pool: pool, server: server, hasher: hasher},
		content: contentService,
	}
}

func (e *studioEnv) createStudent(nisn, username, displayName string) uuid.UUID {
	e.t.Helper()

	now := time.Now().UTC()
	return e.createUser(flowUserSpec{
		nisn:        &nisn,
		username:    &username,
		displayName: displayName,
		password:    studioTestPassword,
		activatedAt: &now,
	})
}

func (e *studioEnv) loginAs(identifier string) (*http.Client, string) {
	e.t.Helper()

	client := e.client()
	csrf := e.fetchCSRF(client)
	res, body := e.login(client, identifier, studioTestPassword, csrf)
	if res.StatusCode != http.StatusOK {
		e.t.Fatalf("login %s status = %d, body %s", identifier, res.StatusCode, body)
	}
	return client, e.csrfFromJar(client)
}

func TestStudioContent_CreateSaveListSummaryAndAccess(t *testing.T) {
	env := newStudioEnv(t)
	env.createStudent("1000000001", "siswastudio", "Siswa Studio")
	client, csrf := env.loginAs("1000000001")

	res, body := env.do(client, http.MethodPost, "/api/v1/studio/contents", map[string]any{
		"type": "portfolio",
	}, csrf, nil)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /studio/contents status = %d, body %s", res.StatusCode, body)
	}
	var created apigen.StudioContent
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if len(created.PublicId) != 8 || created.Status != apigen.ContentStatusDraft {
		t.Fatalf("publicId/status = %q/%s, mau 8 karakter/draft", created.PublicId, created.Status)
	}
	if created.Draft.LockVersion != 1 {
		t.Fatalf("lockVersion = %d, mau 1", created.Draft.LockVersion)
	}

	draftPath := "/api/v1/studio/contents/" + created.Id.String() + "/draft"
	draftBody := map[string]any{
		"lockVersion": 1,
		"title":       "Poster Uji",
		"body":        map[string]any{"blocks": []any{}},
		"meta":        map[string]any{"context": "personal"},
		"tags":        []string{"poster"},
	}
	res, body = env.do(client, http.MethodPut, draftPath, draftBody, csrf, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("PUT draft status = %d, body %s", res.StatusCode, body)
	}
	var saved apigen.StudioContent
	if err := json.Unmarshal(body, &saved); err != nil {
		t.Fatalf("decode saved: %v", err)
	}
	if saved.Draft.LockVersion != 2 {
		t.Fatalf("lockVersion = %d, mau 2", saved.Draft.LockVersion)
	}
	if saved.Draft.Title != "Poster Uji" {
		t.Errorf("draft title = %q, mau Poster Uji", saved.Draft.Title)
	}

	res, body = env.do(client, http.MethodPut, draftPath, draftBody, csrf, nil)
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("PUT draft dengan lock lama status = %d, body %s", res.StatusCode, body)
	}
	var problem apigen.Problem
	if err := json.Unmarshal(body, &problem); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if problem.Code != "LOCK_CONFLICT" {
		t.Errorf("code = %q, mau LOCK_CONFLICT", problem.Code)
	}

	res, body = env.do(client, http.MethodGet, "/api/v1/studio/contents", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /studio/contents status = %d, body %s", res.StatusCode, body)
	}
	var page apigen.StudioContentPage
	if err := json.Unmarshal(body, &page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Id != created.Id {
		t.Fatalf("items = %+v, mau hanya %s", page.Items, created.Id)
	}
	if page.Items[0].Title != "Poster Uji" {
		t.Errorf("item title = %q, mau Poster Uji", page.Items[0].Title)
	}

	res, body = env.do(client, http.MethodGet, "/api/v1/studio/summary", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /studio/summary status = %d, body %s", res.StatusCode, body)
	}
	var summary apigen.StudioSummary
	if err := json.Unmarshal(body, &summary); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
	if summary.Counts[string(apigen.ContentStatusDraft)] != 1 {
		t.Errorf("counts draft = %d, mau 1 (%+v)", summary.Counts[string(apigen.ContentStatusDraft)], summary.Counts)
	}
	if summary.PendingInvitations != 0 {
		t.Errorf("pendingInvitations = %d, mau 0", summary.PendingInvitations)
	}

	env.createStudent("1000000002", "siswalain", "Siswa Lain")
	otherClient, _ := env.loginAs("1000000002")
	res, body = env.do(otherClient, http.MethodGet, "/api/v1/studio/contents/"+created.Id.String(), nil, "", nil)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("GET studio content oleh siswa lain status = %d, body %s", res.StatusCode, body)
	}

	res, body = env.do(env.server.Client(), http.MethodGet, "/api/v1/studio/contents", nil, "", nil)
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /studio/contents anonim status = %d, body %s", res.StatusCode, body)
	}
}
