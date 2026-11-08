package content

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	defaultInvitationLimit = 20
)

// InvitationContent is the content summary shown on an invitation card.
type InvitationContent struct {
	ID       uuid.UUID
	PublicID string
	Type     string
	Title    string
	Cover    *ImageRef
}

// Invitation is one pending collaboration invitation for the caller.
type Invitation struct {
	Content   InvitationContent
	InvitedBy AuthorMini
	Level     string
	RoleLabel *string
	InvitedAt time.Time
	ExpiresAt time.Time
}

// InvitationPage is one page of pending invitations.
type InvitationPage struct {
	Items      []Invitation
	NextCursor string
}

// ListInvitations returns pending, unexpired invitations of the caller.
func (s *Service) ListInvitations(ctx context.Context, actor Actor, cursor string) (InvitationPage, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return InvitationPage{}, err
	}

	now := s.now().UTC()
	params := store.ListPendingInvitationsParams{
		UserID:     principal.UserID,
		Now:        &now,
		LimitCount: defaultInvitationLimit + 1,
	}
	if cursor != "" {
		cursorTime, cursorID, err := decodeInvitationCursor(cursor)
		if err != nil {
			return InvitationPage{}, apperr.BadRequest("cursor tidak valid")
		}
		params.CursorTime = &cursorTime
		params.CursorID = &cursorID
	}

	rows, err := s.st.ListPendingInvitations(ctx, params)
	if err != nil {
		return InvitationPage{}, fmt.Errorf("content: list invitations: %w", err)
	}

	page := InvitationPage{Items: make([]Invitation, 0, min(len(rows), defaultInvitationLimit))}
	if len(rows) > defaultInvitationLimit {
		rows = rows[:defaultInvitationLimit]
		last := rows[len(rows)-1]
		page.NextCursor = encodeInvitationCursor(last.InvitedAt, last.ContentID)
	}
	for _, row := range rows {
		page.Items = append(page.Items, invitationFromRow(row))
	}
	return page, nil
}

// AcceptInvitation accepts a pending invitation of the caller.
func (s *Service) AcceptInvitation(ctx context.Context, actor Actor, contentID uuid.UUID) error {
	principal, err := s.principal(actor)
	if err != nil {
		return err
	}

	return s.st.WithTx(ctx, func(q *store.Queries) error {
		content, err := s.lockContent(ctx, q, contentID)
		if err != nil {
			return err
		}

		row, err := q.GetContentAuthorRow(ctx, store.GetContentAuthorRowParams{
			ContentID: contentID,
			UserID:    principal.UserID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound()
		}
		if err != nil {
			return fmt.Errorf("content: load invitation: %w", err)
		}

		if err := checkPendingInvitation(row, s.now().UTC()); err != nil {
			return err
		}
		if content.Status == StatusDeleted {
			return apperr.InvalidTransition("Konten sudah dihapus.")
		}

		target, err := q.GetInviteTargetUser(ctx, principal.UserID)
		if err != nil {
			return fmt.Errorf("content: load invitee: %w", err)
		}
		if target.ContentBlockedAt != nil {
			return apperr.ContentBlocked()
		}

		respondedAt := s.now().UTC()
		if err := q.AcceptContentAuthorInvite(ctx, store.AcceptContentAuthorInviteParams{
			RespondedAt: &respondedAt,
			ContentID:   contentID,
			UserID:      principal.UserID,
		}); err != nil {
			return fmt.Errorf("content: accept invitation: %w", err)
		}

		// TODO(M6): notify the owner that the invitation was accepted.
		return nil
	})
}

// DeclineInvitation declines a pending invitation of the caller.
func (s *Service) DeclineInvitation(ctx context.Context, actor Actor, contentID uuid.UUID) error {
	principal, err := s.principal(actor)
	if err != nil {
		return err
	}

	return s.st.WithTx(ctx, func(q *store.Queries) error {
		row, err := q.GetContentAuthorRow(ctx, store.GetContentAuthorRowParams{
			ContentID: contentID,
			UserID:    principal.UserID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound()
		}
		if err != nil {
			return fmt.Errorf("content: load invitation: %w", err)
		}

		if err := checkPendingInvitation(row, s.now().UTC()); err != nil {
			return err
		}

		respondedAt := s.now().UTC()
		if err := q.DeclineContentAuthorInvite(ctx, store.DeclineContentAuthorInviteParams{
			RespondedAt: &respondedAt,
			ContentID:   contentID,
			UserID:      principal.UserID,
		}); err != nil {
			return fmt.Errorf("content: decline invitation: %w", err)
		}
		return nil
	})
}

// checkPendingInvitation verifies the row is a pending, unexpired invitation.
func checkPendingInvitation(row store.ContentAuthor, now time.Time) error {
	if row.InviteStatus == "expired" {
		return apperr.InvalidTransition("Undangan sudah kedaluwarsa.")
	}
	if row.InviteStatus != "pending" {
		return apperr.NotFound()
	}
	if row.InviteExpiresAt == nil || !row.InviteExpiresAt.After(now) {
		return apperr.InvalidTransition("Undangan sudah kedaluwarsa.")
	}
	return nil
}

func invitationFromRow(row store.ListPendingInvitationsRow) Invitation {
	item := Invitation{
		Content: InvitationContent{
			ID:       row.ContentID,
			PublicID: row.PublicID,
			Type:     row.Type,
			Title:    row.Title,
		},
		InvitedBy: AuthorMini{
			UserID:      row.InviterID,
			Username:    row.InviterUsername,
			DisplayName: row.InviterDisplayName,
		},
		Level:     row.Level,
		RoleLabel: row.RoleLabel,
		InvitedAt: row.InvitedAt,
		ExpiresAt: row.InvitedAt,
	}
	if row.InviteExpiresAt != nil {
		item.ExpiresAt = *row.InviteExpiresAt
	}
	if row.CoverMediaID != nil {
		item.Content.Cover = newImageRef(
			*row.CoverMediaID,
			row.CoverWidth,
			row.CoverHeight,
			row.CoverBlurhash,
			row.CoverDominantColor,
		)
	}
	return item
}

type invitationCursor struct {
	Time string `json:"t"`
	ID   string `json:"i"`
}

func encodeInvitationCursor(invitedAt time.Time, contentID uuid.UUID) string {
	raw, err := json.Marshal(invitationCursor{
		Time: invitedAt.UTC().Format(time.RFC3339Nano),
		ID:   contentID.String(),
	})
	if err != nil {
		panic("content: encode invitation cursor: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeInvitationCursor(value string) (time.Time, uuid.UUID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	var payload invitationCursor
	if err := json.Unmarshal(raw, &payload); err != nil {
		return time.Time{}, uuid.Nil, err
	}
	invitedAt, err := time.Parse(time.RFC3339Nano, payload.Time)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	id, err := uuid.Parse(payload.ID)
	if err != nil || id == uuid.Nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("content: invalid invitation cursor id")
	}
	return invitedAt, id, nil
}
