package taxonomy

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

func newTestService(t *testing.T) (*Service, *pgxpool.Pool) {
	t.Helper()
	return newTestServiceWithCache(t, nil)
}

func newTestServiceWithCache(t *testing.T, cache Cache) (*Service, *pgxpool.Pool) {
	t.Helper()

	pool := testutil.NewSeededDB(t)
	st := store.NewStore(pool)
	if cache == nil {
		cache = NewValkeyCache(testutil.NewValkey(t))
	}
	return NewService(st, cache), pool
}

func TestCategories_PortfolioReturnsSeededCategoriesWithMajorCodes(t *testing.T) {
	svc, _ := newTestService(t)

	categories, err := svc.Categories(context.Background(), ContentTypePortfolio)
	if err != nil {
		t.Fatalf("Categories: %v", err)
	}

	if len(categories) != 22 {
		t.Fatalf("len = %d, want 22", len(categories))
	}

	var webApp *Category
	for i := range categories {
		if categories[i].Slug == "web-app" {
			webApp = &categories[i]
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

func TestCategories_BlogReturnsSixSeededCategories(t *testing.T) {
	svc, _ := newTestService(t)

	categories, err := svc.Categories(context.Background(), ContentTypeBlog)
	if err != nil {
		t.Fatalf("Categories: %v", err)
	}

	if len(categories) != 6 {
		t.Fatalf("len = %d, want 6", len(categories))
	}
}

func TestCategories_InactiveCategoryIsHidden(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx,
		`UPDATE content_categories SET is_active = false WHERE content_type = 'blog' AND slug = 'tutorial'`,
	); err != nil {
		t.Fatalf("deactivate category: %v", err)
	}

	categories, err := svc.Categories(ctx, ContentTypeBlog)
	if err != nil {
		t.Fatalf("Categories: %v", err)
	}

	if len(categories) != 5 {
		t.Fatalf("len = %d, want 5", len(categories))
	}
	for _, category := range categories {
		if category.Slug == "tutorial" {
			t.Fatal("kategori nonaktif tutorial masih muncul")
		}
	}
}

func TestCategories_CachedResultSurvivesDatabaseChange(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()

	first, err := svc.Categories(ctx, ContentTypePortfolio)
	if err != nil {
		t.Fatalf("first Categories: %v", err)
	}

	if _, err := pool.Exec(ctx,
		`UPDATE content_categories SET is_active = false WHERE content_type = 'portfolio' AND slug = 'game'`,
	); err != nil {
		t.Fatalf("deactivate category: %v", err)
	}

	second, err := svc.Categories(ctx, ContentTypePortfolio)
	if err != nil {
		t.Fatalf("second Categories: %v", err)
	}

	if len(first) != len(second) {
		t.Errorf("cached len = %d, want %d", len(second), len(first))
	}
}

type failingCache struct{}

func (failingCache) Get(context.Context, string) ([]byte, error) {
	return nil, errors.New("valkey tidak tersedia")
}

func (failingCache) Set(context.Context, string, []byte, time.Duration) error {
	return errors.New("valkey tidak tersedia")
}

func TestCategories_CacheErrorFallsBackToDatabase(t *testing.T) {
	svc, _ := newTestServiceWithCache(t, failingCache{})

	categories, err := svc.Categories(context.Background(), ContentTypePortfolio)
	if err != nil {
		t.Fatalf("Categories: %v", err)
	}

	if len(categories) != 22 {
		t.Fatalf("len = %d, want 22", len(categories))
	}
}

func TestCategories_InvalidTypeIsRejected(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Categories(context.Background(), "galeri")
	if !apperr.Is(err, "BAD_REQUEST") {
		t.Fatalf("err = %v, want BAD_REQUEST", err)
	}
}

func TestTools_SearchPhotoRanksPhotoshopFirst(t *testing.T) {
	svc, _ := newTestService(t)

	tools, err := svc.Tools(context.Background(), "photo")
	if err != nil {
		t.Fatalf("Tools: %v", err)
	}

	if len(tools) == 0 {
		t.Fatal("tidak ada hasil untuk photo")
	}
	if tools[0].Slug != "adobe-photoshop" {
		t.Errorf("first = %q, want adobe-photoshop", tools[0].Slug)
	}
}

func TestTools_EmptyQueryReturnsAtMostTwentyOrderedByName(t *testing.T) {
	svc, _ := newTestService(t)

	tools, err := svc.Tools(context.Background(), "  ")
	if err != nil {
		t.Fatalf("Tools: %v", err)
	}

	if len(tools) == 0 || len(tools) > 20 {
		t.Fatalf("len = %d, want 1..20", len(tools))
	}
	for i := 1; i < len(tools); i++ {
		if tools[i-1].Name > tools[i].Name {
			t.Fatalf("hasil tidak urut nama: %q sebelum %q", tools[i-1].Name, tools[i].Name)
		}
	}
}

func TestTools_QueryTooLongIsRejected(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Tools(context.Background(), strings.Repeat("a", 61))
	if !apperr.Is(err, "BAD_REQUEST") {
		t.Fatalf("err = %v, want BAD_REQUEST", err)
	}
}

func TestTags_SearchExcludesBlockedTags(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `
		INSERT INTO tags (slug, name, is_blocked) VALUES
			('fotografi-digital', 'Fotografi Digital', false),
			('fotografi-blokir', 'Fotografi Blokir', true)`,
	); err != nil {
		t.Fatalf("insert tags: %v", err)
	}

	tags, err := svc.Tags(ctx, "fotografi")
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}

	var slugs []string
	for _, tag := range tags {
		slugs = append(slugs, tag.Slug)
	}
	if !slices.Contains(slugs, "fotografi-digital") {
		t.Errorf("slugs = %v, want contains fotografi-digital", slugs)
	}
	if slices.Contains(slugs, "fotografi-blokir") {
		t.Errorf("slugs = %v, tag terblokir tidak boleh muncul", slugs)
	}
}

func TestTags_EmptyQueryReturnsAtMostTwenty(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `
		INSERT INTO tags (slug, name, is_blocked) VALUES
			('tipografi', 'Tipografi', false),
			('tipografi-blokir', 'Tipografi Blokir', true)`,
	); err != nil {
		t.Fatalf("insert tags: %v", err)
	}

	tags, err := svc.Tags(ctx, "")
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}

	if len(tags) != 1 {
		t.Fatalf("len = %d, want 1 (hanya tag tidak terblokir)", len(tags))
	}
	if tags[0].Slug != "tipografi" {
		t.Errorf("slug = %q, want tipografi", tags[0].Slug)
	}
}

func TestTags_QueryTooLongIsRejected(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Tags(context.Background(), strings.Repeat("a", 41))
	if !apperr.Is(err, "BAD_REQUEST") {
		t.Fatalf("err = %v, want BAD_REQUEST", err)
	}
}
