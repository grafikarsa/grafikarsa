package content

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

// Service implements the content editor domain: creating contents, saving
// drafts, and the Studio reads. Lifecycle transitions and collaboration are
// implemented by separate services that reuse the access helpers here.
type Service struct {
	st  *store.Store
	now func() time.Time
}

// NewService builds a content Service.
func NewService(st *store.Store) *Service {
	return &Service{st: st, now: time.Now}
}

func (s *Service) principal(a Actor) (*auth.Principal, error) {
	if a.Principal == nil {
		return nil, apperr.Unauthenticated()
	}
	return a.Principal, nil
}

// loadContent returns a content or NotFound when it does not exist.
func (s *Service) loadContent(ctx context.Context, q *store.Queries, id uuid.UUID) (store.Content, error) {
	c, err := q.GetContent(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Content{}, apperr.NotFound()
	}
	if err != nil {
		return store.Content{}, fmt.Errorf("content: load content %s: %w", id, err)
	}
	return c, nil
}

// authorAccess returns the accepted author row of userID on contentID.
// Callers that only need existence (read access) use it directly; editing
// callers additionally check the level. Any other invite status, or no row,
// is reported as NotFound so the API never leaks a content's existence.
func (s *Service) authorAccess(ctx context.Context, q *store.Queries, contentID, userID uuid.UUID) (store.ContentAuthor, error) {
	author, err := q.GetContentAuthor(ctx, store.GetContentAuthorParams{
		ContentID: contentID,
		UserID:    userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.ContentAuthor{}, apperr.NotFound()
	}
	if err != nil {
		return store.ContentAuthor{}, fmt.Errorf("content: load author: %w", err)
	}
	return author, nil
}
