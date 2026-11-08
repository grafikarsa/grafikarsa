package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/taxonomy"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

type taxonomyEnv struct {
	t      *testing.T
	pool   *pgxpool.Pool
	server *httptest.Server
}

// newTaxonomyEnv builds the real router with the taxonomy service on a seeded
// database and Valkey. No session is needed for these public endpoints.
func newTaxonomyEnv(t *testing.T) *taxonomyEnv {
	t.Helper()

	pool := testutil.NewSeededDB(t)
	rdb := testutil.NewValkey(t)

	handler := NewRouter(RouterDeps{
		Logger:   testLogger(),
		Taxonomy: taxonomy.NewService(store.NewStore(pool), taxonomy.NewValkeyCache(rdb)),
	})

	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	return &taxonomyEnv{t: t, pool: pool, server: server}
}

func (e *taxonomyEnv) get(path string) (*http.Response, []byte) {
	e.t.Helper()

	resp, err := e.server.Client().Get(e.server.URL + path)
	if err != nil {
		e.t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		e.t.Fatalf("read body %s: %v", path, err)
	}
	return resp, body
}

func (e *taxonomyEnv) categories(t *testing.T, path string) apigen.CategoryList {
	t.Helper()

	resp, body := e.get(path)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Cache-Control"); got != "public, max-age=300" {
		t.Errorf("Cache-Control = %q, want public, max-age=300", got)
	}

	var list apigen.CategoryList
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode categories: %v", err)
	}
	return list
}

func TestTaxonomyCategories_PortfolioReturnsSeededCategories(t *testing.T) {
	env := newTaxonomyEnv(t)

	list := env.categories(t, "/api/v1/taxonomy/categories?type=portfolio")

	if len(list.Items) != 22 {
		t.Fatalf("len = %d, want 22", len(list.Items))
	}

	var webApp *apigen.CategoryOption
	for i := range list.Items {
		if list.Items[i].Slug == "web-app" {
			webApp = &list.Items[i]
			break
		}
	}
	if webApp == nil {
		t.Fatal("kategori web-app tidak ditemukan")
	}
	if !slices.Contains(webApp.MajorCodes, "rpl") {
		t.Errorf("majorCodes web-app = %v, want contains rpl", webApp.MajorCodes)
	}
}

func TestTaxonomyCategories_BlogReturnsSixSeededCategories(t *testing.T) {
	env := newTaxonomyEnv(t)

	list := env.categories(t, "/api/v1/taxonomy/categories?type=blog")

	if len(list.Items) != 6 {
		t.Fatalf("len = %d, want 6", len(list.Items))
	}
}

func TestTaxonomyCategories_InactiveCategoryIsHidden(t *testing.T) {
	env := newTaxonomyEnv(t)

	if _, err := env.pool.Exec(context.Background(),
		`UPDATE content_categories SET is_active = false WHERE content_type = 'blog' AND slug = 'tutorial'`,
	); err != nil {
		t.Fatalf("deactivate category: %v", err)
	}

	list := env.categories(t, "/api/v1/taxonomy/categories?type=blog")

	if len(list.Items) != 5 {
		t.Fatalf("len = %d, want 5", len(list.Items))
	}
	for _, item := range list.Items {
		if item.Slug == "tutorial" {
			t.Fatal("kategori nonaktif tutorial masih muncul")
		}
	}
}

func TestTaxonomyCategories_InvalidTypeIsBadRequest(t *testing.T) {
	env := newTaxonomyEnv(t)

	resp, body := env.get("/api/v1/taxonomy/categories?type=galeri")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %s)", resp.StatusCode, body)
	}
}

func TestTaxonomyTools_SearchPhotoRanksPhotoshopFirst(t *testing.T) {
	env := newTaxonomyEnv(t)

	resp, body := env.get("/api/v1/taxonomy/tools?q=photo")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Cache-Control"); got != "public, max-age=300" {
		t.Errorf("Cache-Control = %q, want public, max-age=300", got)
	}

	var list apigen.TaxonomyTermList
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode tools: %v", err)
	}
	if len(list.Items) == 0 {
		t.Fatal("tidak ada hasil untuk photo")
	}
	if list.Items[0].Slug != "adobe-photoshop" {
		t.Errorf("first = %q, want adobe-photoshop", list.Items[0].Slug)
	}
	if list.Items[0].Category == nil || *list.Items[0].Category != "design_software" {
		t.Errorf("category = %v, want design_software", list.Items[0].Category)
	}
}

func TestTaxonomyTools_EmptyQueryReturnsAtMostTwenty(t *testing.T) {
	env := newTaxonomyEnv(t)

	resp, body := env.get("/api/v1/taxonomy/tools")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", resp.StatusCode, body)
	}

	var list apigen.TaxonomyTermList
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode tools: %v", err)
	}
	if len(list.Items) == 0 || len(list.Items) > 20 {
		t.Fatalf("len = %d, want 1..20", len(list.Items))
	}
}

func TestTaxonomyTags_SearchExcludesBlockedTags(t *testing.T) {
	env := newTaxonomyEnv(t)

	if _, err := env.pool.Exec(context.Background(), `
		INSERT INTO tags (slug, name, is_blocked) VALUES
			('fotografi-digital', 'Fotografi Digital', false),
			('fotografi-blokir', 'Fotografi Blokir', true)`,
	); err != nil {
		t.Fatalf("insert tags: %v", err)
	}

	resp, body := env.get("/api/v1/taxonomy/tags?q=fotografi")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", resp.StatusCode, body)
	}
	if got := resp.Header.Get("Cache-Control"); got != "public, max-age=300" {
		t.Errorf("Cache-Control = %q, want public, max-age=300", got)
	}

	var list apigen.TaxonomyTermList
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode tags: %v", err)
	}

	var slugs []string
	for _, item := range list.Items {
		slugs = append(slugs, item.Slug)
		if item.Category != nil {
			t.Errorf("tag %q category = %v, want null", item.Slug, item.Category)
		}
	}
	if !slices.Contains(slugs, "fotografi-digital") {
		t.Errorf("slugs = %v, want contains fotografi-digital", slugs)
	}
	if slices.Contains(slugs, "fotografi-blokir") {
		t.Errorf("slugs = %v, tag terblokir tidak boleh muncul", slugs)
	}
}

func TestTaxonomyTags_EmptyQueryReturnsAtMostTwenty(t *testing.T) {
	env := newTaxonomyEnv(t)

	resp, body := env.get("/api/v1/taxonomy/tags")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", resp.StatusCode, body)
	}

	var list apigen.TaxonomyTermList
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode tags: %v", err)
	}
	if len(list.Items) > 20 {
		t.Fatalf("len = %d, want <= 20", len(list.Items))
	}
}
