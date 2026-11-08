package content

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/content/body"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/settings"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const publicIDAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"

var (
	defaultPortfolioMeta = json.RawMessage(`{"context":"personal","links":[]}`)
	defaultBlogMeta      = json.RawMessage(`{}`)
)

// Create creates an empty draft content owned by the actor.
func (s *Service) Create(ctx context.Context, actor Actor, input CreateInput) (StudioContent, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return StudioContent{}, err
	}
	if !principal.Can(permissionContentCreate) {
		return StudioContent{}, apperr.ContentBlocked()
	}
	if input.Type != TypePortfolio && input.Type != TypeBlog {
		return StudioContent{}, apperr.Validation(apperr.Field(
			"type", "invalid_value", "Tipe konten harus portfolio atau blog.",
		))
	}

	var result StudioContent
	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		limit := settings.Int(ctx, q, maxActiveDraftsSetting, defaultMaxActiveDrafts)
		active, err := q.CountActiveDraftsByOwner(ctx, principal.UserID)
		if err != nil {
			return fmt.Errorf("content: count active drafts: %w", err)
		}
		if int(active) >= limit {
			return apperr.LimitReached(fmt.Sprintf(
				"Batas draf aktif tercapai (%d). Selesaikan atau hapus draf lain.", limit,
			))
		}

		content, err := insertContentWithPublicID(ctx, q, input.Type, principal.UserID)
		if err != nil {
			return err
		}

		meta := defaultBlogMeta
		if input.Type == TypePortfolio {
			meta = defaultPortfolioMeta
		}
		if _, err := q.InsertContentDraft(ctx, store.InsertContentDraftParams{
			ContentID: content.ID,
			Body:      body.EmptyBody(body.ContentType(input.Type)),
			Meta:      meta,
			UpdatedBy: principal.UserID,
		}); err != nil {
			return fmt.Errorf("content: insert draft: %w", err)
		}

		respondedAt := s.now().UTC()
		if err := q.InsertContentOwner(ctx, store.InsertContentOwnerParams{
			ContentID:   content.ID,
			UserID:      principal.UserID,
			RespondedAt: &respondedAt,
		}); err != nil {
			return fmt.Errorf("content: insert owner: %w", err)
		}

		if err := q.InsertContentStats(ctx, content.ID); err != nil {
			return fmt.Errorf("content: insert stats: %w", err)
		}

		result, err = s.getStudioQ(ctx, q, content.ID, principal.UserID)
		return err
	})
	if err != nil {
		return StudioContent{}, err
	}
	return result, nil
}

// insertContentWithPublicID allocates a random public id and retries on
// collision. The ON CONFLICT DO NOTHING in the insert keeps the transaction
// usable after a collision, so every retry runs in the same transaction.
func insertContentWithPublicID(ctx context.Context, q *store.Queries, contentType string, ownerID uuid.UUID) (store.Content, error) {
	for attempt := 0; attempt < maxPublicIDAttempts; attempt++ {
		publicID, err := randomPublicID()
		if err != nil {
			return store.Content{}, fmt.Errorf("content: generate public id: %w", err)
		}
		content, err := q.InsertContent(ctx, store.InsertContentParams{
			PublicID:    publicID,
			ContentType: contentType,
			OwnerID:     ownerID,
		})
		if err == nil {
			return content, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return store.Content{}, fmt.Errorf("content: insert content: %w", err)
		}
	}
	return store.Content{}, fmt.Errorf("content: exhausted public id attempts")
}

func randomPublicID() (string, error) {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	out := make([]byte, len(raw))
	for i, b := range raw {
		out[i] = publicIDAlphabet[int(b)%len(publicIDAlphabet)]
	}
	return string(out), nil
}
