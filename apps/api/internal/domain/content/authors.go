package content

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/settings"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	maxRoleLabelRunes    = 60
	maxAuthorPosition    = 1000
	defaultInviteTTLDays = 14

	maxAuthorsPortfolioSetting = "collab.max_authors_portfolio"
	maxAuthorsBlogSetting      = "collab.max_authors_blog"
	inviteTTLDaysSetting       = "collab.invite_ttl_days"

	defaultMaxAuthorsPortfolio = 50
	defaultMaxAuthorsBlog      = 10
)

// AuthorInviteInput is the payload of InviteAuthor.
type AuthorInviteInput struct {
	UserID    uuid.UUID
	Level     string
	RoleLabel *string
}

// AuthorUpdateInput is the payload of UpdateAuthor. Nil level and position are
// left unchanged; a nil roleLabel clears the label (it is nullable in the
// contract).
type AuthorUpdateInput struct {
	Level     *string
	RoleLabel *string
	Position  *int
}

// ContentAuthorView is one author row shown in the Studio.
type ContentAuthorView struct {
	User            AuthorMini
	Level           string
	RoleLabel       *string
	Position        int
	InviteStatus    string
	InvitedAt       time.Time
	InviteExpiresAt *time.Time
}

// ListAuthors returns accepted authors and pending invitations of a content.
func (s *Service) ListAuthors(ctx context.Context, actor Actor, contentID uuid.UUID) ([]ContentAuthorView, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return nil, err
	}

	if _, err := s.authorAccess(ctx, s.st.Queries, contentID, principal.UserID); err != nil {
		return nil, err
	}

	rows, err := s.st.ListContentAuthors(ctx, contentID)
	if err != nil {
		return nil, fmt.Errorf("content: list authors: %w", err)
	}

	items := make([]ContentAuthorView, 0, len(rows))
	for _, row := range rows {
		items = append(items, authorViewFromListRow(row))
	}
	return items, nil
}

// InviteAuthor invites a user to collaborate on a content. Only the owner may
// invite, and the content must not be in a terminal status.
func (s *Service) InviteAuthor(ctx context.Context, actor Actor, contentID uuid.UUID, input AuthorInviteInput) (ContentAuthorView, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return ContentAuthorView{}, err
	}

	var result ContentAuthorView
	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		content, err := s.lockContent(ctx, q, contentID)
		if err != nil {
			return err
		}

		author, err := s.authorAccess(ctx, q, contentID, principal.UserID)
		if err != nil {
			return err
		}
		if author.Level != LevelOwner {
			return ownerOnlyForbidden()
		}
		if content.Status == StatusDeleted || content.Status == StatusRejected {
			return apperr.InvalidTransition("Konten ini tidak bisa menerima author.")
		}

		var fields []apperr.FieldError
		if input.Level != LevelEditor && input.Level != LevelContributor {
			fields = append(fields, apperr.Field(
				"level", "invalid_value", "Level author harus editor atau contributor.",
			))
		}
		roleLabel := normalizeRoleLabel(input.RoleLabel, &fields)

		if input.UserID == content.OwnerID {
			fields = append(fields, apperr.Field(
				"userId", "invalid_user", "Tidak bisa mengundang diri sendiri.",
			))
		} else {
			targetFields, err := inviteTargetErrors(ctx, q, input.UserID)
			if err != nil {
				return err
			}
			fields = append(fields, targetFields...)
		}
		if len(fields) > 0 {
			return apperr.Validation(fields...)
		}

		existing, err := q.GetContentAuthorRow(ctx, store.GetContentAuthorRowParams{
			ContentID: contentID,
			UserID:    input.UserID,
		})
		hasExisting := err == nil
		switch {
		case hasExisting:
			if existing.InviteStatus == "accepted" || existing.InviteStatus == "pending" {
				return apperr.Conflict("Pengguna sudah menjadi author atau sedang diundang.")
			}
		case !errors.Is(err, pgx.ErrNoRows):
			return fmt.Errorf("content: load invite row: %w", err)
		}

		rule := maxAuthorsFor(content.Type)
		limit := settings.Int(ctx, q, rule.setting, rule.def)
		count, err := q.CountActiveAuthors(ctx, contentID)
		if err != nil {
			return fmt.Errorf("content: count active authors: %w", err)
		}
		if int(count) >= limit {
			return apperr.LimitReached(fmt.Sprintf("Batas author tercapai (%d).", limit))
		}

		ttlDays := settings.Int(ctx, q, inviteTTLDaysSetting, defaultInviteTTLDays)
		if ttlDays <= 0 {
			ttlDays = defaultInviteTTLDays
		}
		now := s.now().UTC()
		expiresAt := now.Add(time.Duration(ttlDays) * 24 * time.Hour)
		invitedBy := principal.UserID

		if !hasExisting {
			position, err := q.NextAuthorPosition(ctx, contentID)
			if err != nil {
				return fmt.Errorf("content: next author position: %w", err)
			}
			if err := q.InsertContentAuthorInvite(ctx, store.InsertContentAuthorInviteParams{
				ContentID:       contentID,
				UserID:          input.UserID,
				Level:           input.Level,
				RoleLabel:       roleLabel,
				Position:        position,
				InvitedBy:       &invitedBy,
				InvitedAt:       now,
				InviteExpiresAt: &expiresAt,
			}); err != nil {
				return fmt.Errorf("content: insert author invite: %w", err)
			}
		} else {
			// Declined, expired, left, or removed rows are reused so a user
			// keeps a single author row per content.
			if err := q.ReinviteContentAuthor(ctx, store.ReinviteContentAuthorParams{
				Level:           input.Level,
				RoleLabel:       roleLabel,
				InvitedBy:       &invitedBy,
				InvitedAt:       now,
				InviteExpiresAt: &expiresAt,
				ContentID:       contentID,
				UserID:          input.UserID,
			}); err != nil {
				return fmt.Errorf("content: reinvite author: %w", err)
			}
		}

		// TODO(M6): notify the invitee about the collaboration invitation.

		result, err = authorByUser(ctx, q, contentID, input.UserID)
		return err
	})
	if err != nil {
		return ContentAuthorView{}, err
	}
	return result, nil
}

// UpdateAuthor changes the level, role label, or display position of an
// accepted or pending author. Only the owner may do it.
func (s *Service) UpdateAuthor(
	ctx context.Context,
	actor Actor,
	contentID, targetID uuid.UUID,
	input AuthorUpdateInput,
) (ContentAuthorView, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return ContentAuthorView{}, err
	}

	var result ContentAuthorView
	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		content, err := s.lockContent(ctx, q, contentID)
		if err != nil {
			return err
		}
		if err := s.requireOwner(ctx, q, contentID, principal.UserID); err != nil {
			return err
		}
		if targetID == content.OwnerID {
			return apperr.NotFound()
		}

		target, err := q.GetContentAuthorRow(ctx, store.GetContentAuthorRowParams{
			ContentID: contentID,
			UserID:    targetID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound()
		}
		if err != nil {
			return fmt.Errorf("content: load author: %w", err)
		}
		if target.InviteStatus != "accepted" && target.InviteStatus != "pending" {
			return apperr.NotFound()
		}

		var fields []apperr.FieldError
		level := target.Level
		if input.Level != nil {
			if *input.Level != LevelEditor && *input.Level != LevelContributor {
				fields = append(fields, apperr.Field(
					"level", "invalid_value", "Level author harus editor atau contributor.",
				))
			} else {
				level = *input.Level
			}
		}

		roleLabel := normalizeRoleLabel(input.RoleLabel, &fields)

		position := int(target.Position)
		if input.Position != nil {
			if *input.Position < 0 || *input.Position > maxAuthorPosition {
				fields = append(fields, apperr.Field(
					"position", "invalid_value", "Urutan author harus antara 0 dan 1000.",
				))
			} else {
				position = *input.Position
			}
		}
		if len(fields) > 0 {
			return apperr.Validation(fields...)
		}

		if err := q.UpdateContentAuthorFields(ctx, store.UpdateContentAuthorFieldsParams{
			Level:     level,
			RoleLabel: roleLabel,
			Position:  int16(position),
			ContentID: contentID,
			UserID:    targetID,
		}); err != nil {
			return fmt.Errorf("content: update author: %w", err)
		}

		result, err = authorByUser(ctx, q, contentID, targetID)
		return err
	})
	if err != nil {
		return ContentAuthorView{}, err
	}
	return result, nil
}

// RemoveAuthor removes an accepted or pending author, or cancels a pending
// invitation. Only the owner may do it.
func (s *Service) RemoveAuthor(ctx context.Context, actor Actor, contentID, targetID uuid.UUID) error {
	principal, err := s.principal(actor)
	if err != nil {
		return err
	}

	return s.st.WithTx(ctx, func(q *store.Queries) error {
		content, err := s.lockContent(ctx, q, contentID)
		if err != nil {
			return err
		}
		if err := s.requireOwner(ctx, q, contentID, principal.UserID); err != nil {
			return err
		}
		if targetID == principal.UserID || targetID == content.OwnerID {
			return apperr.Validation(apperr.Field(
				"userId", "forbidden", "Pindahkan kepemilikan dulu sebelum keluar.",
			))
		}

		target, err := q.GetContentAuthorRow(ctx, store.GetContentAuthorRowParams{
			ContentID: contentID,
			UserID:    targetID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound()
		}
		if err != nil {
			return fmt.Errorf("content: load author: %w", err)
		}
		if target.Level == LevelOwner ||
			(target.InviteStatus != "accepted" && target.InviteStatus != "pending") {
			return apperr.NotFound()
		}

		respondedAt := s.now().UTC()
		if err := q.RemoveContentAuthor(ctx, store.RemoveContentAuthorParams{
			RespondedAt: &respondedAt,
			ContentID:   contentID,
			UserID:      targetID,
		}); err != nil {
			return fmt.Errorf("content: remove author: %w", err)
		}
		return nil
	})
}

// TransferOwnership moves ownership to an accepted author. The former owner
// becomes an editor.
func (s *Service) TransferOwnership(ctx context.Context, actor Actor, contentID, targetID uuid.UUID) ([]ContentAuthorView, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return nil, err
	}

	var items []ContentAuthorView
	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		content, err := s.lockContent(ctx, q, contentID)
		if err != nil {
			return err
		}
		if err := s.requireOwner(ctx, q, contentID, principal.UserID); err != nil {
			return err
		}
		if targetID == content.OwnerID {
			return apperr.Validation(apperr.Field(
				"userId", "invalid_value", "Pengguna ini sudah menjadi pemilik.",
			))
		}

		target, err := q.GetContentAuthor(ctx, store.GetContentAuthorParams{
			ContentID: contentID,
			UserID:    targetID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound()
		}
		if err != nil {
			return fmt.Errorf("content: load transfer target: %w", err)
		}
		if target.Level == LevelOwner {
			return apperr.NotFound()
		}

		// Demote before promoting so uq_content_one_owner is never violated.
		if err := q.SetContentAuthorLevel(ctx, store.SetContentAuthorLevelParams{
			Level:     LevelEditor,
			ContentID: contentID,
			UserID:    content.OwnerID,
		}); err != nil {
			return fmt.Errorf("content: demote owner: %w", err)
		}
		if err := q.SetContentAuthorLevel(ctx, store.SetContentAuthorLevelParams{
			Level:     LevelOwner,
			ContentID: contentID,
			UserID:    targetID,
		}); err != nil {
			return fmt.Errorf("content: promote author: %w", err)
		}
		if err := q.UpdateContentOwner(ctx, store.UpdateContentOwnerParams{
			OwnerID: targetID,
			ID:      contentID,
		}); err != nil {
			return fmt.Errorf("content: update owner: %w", err)
		}

		// TODO(M6): notify both users about the ownership transfer.

		items, err = s.listAuthorsQ(ctx, q, contentID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

// LeaveContent lets an accepted editor or contributor leave a content. The
// owner must transfer ownership first.
func (s *Service) LeaveContent(ctx context.Context, actor Actor, contentID uuid.UUID) error {
	principal, err := s.principal(actor)
	if err != nil {
		return err
	}

	return s.st.WithTx(ctx, func(q *store.Queries) error {
		if _, err := s.lockContent(ctx, q, contentID); err != nil {
			return err
		}

		author, err := s.authorAccess(ctx, q, contentID, principal.UserID)
		if err != nil {
			return err
		}
		if author.Level == LevelOwner {
			return apperr.Validation(apperr.Field(
				"userId", "forbidden", "Pindahkan kepemilikan dulu sebelum keluar.",
			))
		}

		respondedAt := s.now().UTC()
		if err := q.LeaveContentAuthor(ctx, store.LeaveContentAuthorParams{
			RespondedAt: &respondedAt,
			ContentID:   contentID,
			UserID:      principal.UserID,
		}); err != nil {
			return fmt.Errorf("content: leave content: %w", err)
		}
		return nil
	})
}

// lockContent loads a content row and locks it for the rest of the
// transaction, keeping author limit checks race-free.
func (s *Service) lockContent(ctx context.Context, q *store.Queries, id uuid.UUID) (store.Content, error) {
	row, err := q.LockContent(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Content{}, apperr.NotFound()
	}
	if err != nil {
		return store.Content{}, fmt.Errorf("content: lock content %s: %w", id, err)
	}
	return row, nil
}

// requireOwner resolves the actor's accepted author row and requires the
// owner level.
func (s *Service) requireOwner(ctx context.Context, q *store.Queries, contentID, userID uuid.UUID) error {
	author, err := s.authorAccess(ctx, q, contentID, userID)
	if err != nil {
		return err
	}
	if author.Level != LevelOwner {
		return ownerOnlyForbidden()
	}
	return nil
}

// listAuthorsQ is ListAuthors bound to an arbitrary query handle so callers
// inside a transaction see their own writes.
func (s *Service) listAuthorsQ(ctx context.Context, q *store.Queries, contentID uuid.UUID) ([]ContentAuthorView, error) {
	rows, err := q.ListContentAuthors(ctx, contentID)
	if err != nil {
		return nil, fmt.Errorf("content: list authors: %w", err)
	}
	items := make([]ContentAuthorView, 0, len(rows))
	for _, row := range rows {
		items = append(items, authorViewFromListRow(row))
	}
	return items, nil
}

func authorByUser(ctx context.Context, q *store.Queries, contentID, userID uuid.UUID) (ContentAuthorView, error) {
	row, err := q.GetContentAuthorWithUser(ctx, store.GetContentAuthorWithUserParams{
		ContentID: contentID,
		UserID:    userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ContentAuthorView{}, apperr.NotFound()
	}
	if err != nil {
		return ContentAuthorView{}, fmt.Errorf("content: load author: %w", err)
	}
	return authorViewFromWithUserRow(row), nil
}

func authorViewFromListRow(row store.ListContentAuthorsRow) ContentAuthorView {
	return ContentAuthorView{
		User: AuthorMini{
			UserID:      row.UserID,
			Username:    row.Username,
			DisplayName: row.DisplayName,
		},
		Level:           row.Level,
		RoleLabel:       row.RoleLabel,
		Position:        int(row.Position),
		InviteStatus:    row.InviteStatus,
		InvitedAt:       row.InvitedAt,
		InviteExpiresAt: row.InviteExpiresAt,
	}
}

func authorViewFromWithUserRow(row store.GetContentAuthorWithUserRow) ContentAuthorView {
	return ContentAuthorView{
		User: AuthorMini{
			UserID:      row.UserID,
			Username:    row.Username,
			DisplayName: row.DisplayName,
		},
		Level:           row.Level,
		RoleLabel:       row.RoleLabel,
		Position:        int(row.Position),
		InviteStatus:    row.InviteStatus,
		InvitedAt:       row.InvitedAt,
		InviteExpiresAt: row.InviteExpiresAt,
	}
}

type maxAuthorsRule struct {
	setting string
	def     int
}

// maxAuthorsFor returns the setting key and default limit of a content type.
func maxAuthorsFor(contentType string) maxAuthorsRule {
	if contentType == TypeBlog {
		return maxAuthorsRule{setting: maxAuthorsBlogSetting, def: defaultMaxAuthorsBlog}
	}
	return maxAuthorsRule{setting: maxAuthorsPortfolioSetting, def: defaultMaxAuthorsPortfolio}
}

// inviteTargetErrors reports why a user cannot be invited.
func inviteTargetErrors(ctx context.Context, q *store.Queries, userID uuid.UUID) ([]apperr.FieldError, error) {
	target, err := q.GetInviteTargetUser(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return []apperr.FieldError{inviteTargetField()}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("content: load invite target: %w", err)
	}
	if target.Status != "active" && target.Status != "alumni" {
		return []apperr.FieldError{inviteTargetField()}, nil
	}
	if target.ActivatedAt == nil || target.ContentBlockedAt != nil {
		return []apperr.FieldError{inviteTargetField()}, nil
	}
	return nil, nil
}

func inviteTargetField() apperr.FieldError {
	return apperr.Field("userId", "invalid_user", "Pengguna ini tidak bisa diundang.")
}

// normalizeRoleLabel trims the optional role label and clears it when empty.
func normalizeRoleLabel(raw *string, fields *[]apperr.FieldError) *string {
	if raw == nil {
		return nil
	}
	label := strings.TrimSpace(*raw)
	if label == "" {
		return nil
	}
	if utf8.RuneCountInString(label) > maxRoleLabelRunes {
		*fields = append(*fields, apperr.Field(
			"roleLabel", "too_long", "Label peran maksimal 60 karakter.",
		))
		return nil
	}
	return &label
}

// ownerOnlyForbidden is the 403 returned when an accepted non-owner author
// tries an owner-only action.
func ownerOnlyForbidden() error {
	return &apperr.Error{
		Status: http.StatusForbidden,
		Code:   "FORBIDDEN_LEVEL",
		Title:  "Hanya pemilik yang bisa melakukan ini.",
	}
}
