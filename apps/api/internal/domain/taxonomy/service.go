// Package taxonomy implements the public taxonomy lookups used by the content
// editor autocompletes: active categories, tools, and tags. HTTP handlers live
// elsewhere; this package only exposes the domain service.
package taxonomy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	// ContentTypePortfolio and ContentTypeBlog are the category content types.
	ContentTypePortfolio = "portfolio"
	ContentTypeBlog      = "blog"

	categoryCachePrefix = "tax:categories:"
	categoryCacheTTL    = 5 * time.Minute

	maxToolsQueryLength = 60
	maxTagsQueryLength  = 40
)

// Category is one active content category with the majors it is suggested for.
type Category struct {
	ID          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	MajorCodes  []string  `json:"majorCodes"`
}

// Term is a tool or tag suggestion for the content editor autocomplete.
type Term struct {
	Slug     string  `json:"slug"`
	Name     string  `json:"name"`
	Category *string `json:"category,omitempty"`
}

// Cache is the subset of the Valkey API the service needs.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// ErrCacheMiss reports that the requested key is absent.
var ErrCacheMiss = errors.New("taxonomy: cache miss")

// ValkeyCache adapts a Valkey (Redis-compatible) client to Cache.
type ValkeyCache struct {
	client *redis.Client
}

// NewValkeyCache returns a Cache backed by client.
func NewValkeyCache(client *redis.Client) *ValkeyCache {
	return &ValkeyCache{client: client}
}

// Get returns the cached bytes for key, or ErrCacheMiss when it is absent.
func (c *ValkeyCache) Get(ctx context.Context, key string) ([]byte, error) {
	raw, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrCacheMiss
	}
	return raw, err
}

// Set stores value under key for ttl.
func (c *ValkeyCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Service implements the public taxonomy lookups.
type Service struct {
	st    *store.Store
	cache Cache
}

// NewService builds a taxonomy Service. A nil cache disables caching.
func NewService(st *store.Store, cache Cache) *Service {
	return &Service{st: st, cache: cache}
}

// Categories returns the active categories of contentType ordered by
// sort_order then name. Results are cached for five minutes per content type;
// cache failures fall back to the database.
func (s *Service) Categories(ctx context.Context, contentType string) ([]Category, error) {
	if contentType != ContentTypePortfolio && contentType != ContentTypeBlog {
		return nil, apperr.BadRequest("type harus portfolio atau blog")
	}

	if cached, ok := s.cachedCategories(ctx, contentType); ok {
		return cached, nil
	}

	rows, err := s.st.ListContentCategories(ctx, contentType)
	if err != nil {
		return nil, fmt.Errorf("taxonomy: list categories: %w", err)
	}

	categories := make([]Category, 0, len(rows))
	for _, row := range rows {
		majorCodes := row.MajorCodes
		if majorCodes == nil {
			majorCodes = []string{}
		}
		categories = append(categories, Category{
			ID:          row.ID,
			Slug:        row.Slug,
			Name:        row.Name,
			Description: row.Description,
			MajorCodes:  majorCodes,
		})
	}

	s.cacheCategories(ctx, contentType, categories)

	return categories, nil
}

func (s *Service) cachedCategories(ctx context.Context, contentType string) ([]Category, bool) {
	if s.cache == nil {
		return nil, false
	}
	raw, err := s.cache.Get(ctx, categoryCachePrefix+contentType)
	if err != nil {
		return nil, false
	}
	var categories []Category
	if err := json.Unmarshal(raw, &categories); err != nil {
		return nil, false
	}
	for i := range categories {
		if categories[i].MajorCodes == nil {
			categories[i].MajorCodes = []string{}
		}
	}
	return categories, true
}

func (s *Service) cacheCategories(ctx context.Context, contentType string, categories []Category) {
	if s.cache == nil {
		return
	}
	raw, err := json.Marshal(categories)
	if err != nil {
		return
	}
	// A cache write failure must not fail the request: the database already
	// answered it.
	_ = s.cache.Set(ctx, categoryCachePrefix+contentType, raw, categoryCacheTTL)
}

// Tools returns up to 20 active tools. An empty q lists them by name; otherwise
// it searches name and slug, ranking exact and prefix matches first.
func (s *Service) Tools(ctx context.Context, q string) ([]Term, error) {
	q = strings.TrimSpace(q)
	if utf8.RuneCountInString(q) > maxToolsQueryLength {
		return nil, apperr.BadRequest("q maksimal 60 karakter")
	}

	if q == "" {
		rows, err := s.st.ListActiveTools(ctx)
		if err != nil {
			return nil, fmt.Errorf("taxonomy: list tools: %w", err)
		}
		terms := make([]Term, 0, len(rows))
		for _, row := range rows {
			terms = append(terms, Term{Slug: row.Slug, Name: row.Name, Category: &row.Category})
		}
		return terms, nil
	}

	rows, err := s.st.SearchTools(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("taxonomy: search tools: %w", err)
	}
	terms := make([]Term, 0, len(rows))
	for _, row := range rows {
		terms = append(terms, Term{Slug: row.Slug, Name: row.Name, Category: &row.Category})
	}
	return terms, nil
}

// Tags returns up to 20 tags. An empty q lists the most used tags first;
// otherwise it searches name and slug. Blocked tags never appear.
func (s *Service) Tags(ctx context.Context, q string) ([]Term, error) {
	q = strings.TrimSpace(q)
	if utf8.RuneCountInString(q) > maxTagsQueryLength {
		return nil, apperr.BadRequest("q maksimal 40 karakter")
	}

	if q == "" {
		rows, err := s.st.ListPopularTags(ctx)
		if err != nil {
			return nil, fmt.Errorf("taxonomy: list tags: %w", err)
		}
		terms := make([]Term, 0, len(rows))
		for _, row := range rows {
			terms = append(terms, Term{Slug: row.Slug, Name: row.Name})
		}
		return terms, nil
	}

	rows, err := s.st.SearchTags(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("taxonomy: search tags: %w", err)
	}
	terms := make([]Term, 0, len(rows))
	for _, row := range rows {
		terms = append(terms, Term{Slug: row.Slug, Name: row.Name})
	}
	return terms, nil
}
