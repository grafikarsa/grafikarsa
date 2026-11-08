package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/content/body"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/settings"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	minTitleRunes       = 3
	minBlogExcerptRunes = 50

	versionKindInitial      = "initial"
	versionKindResubmission = "resubmission"
	versionKindRevision     = "revision"
)

// Submit sends the current draft to moderation. It creates the immutable
// snapshot version and the open moderation case in the same transaction.
func (s *Service) Submit(ctx context.Context, actor Actor, contentID uuid.UUID) (StudioContent, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return StudioContent{}, err
	}

	var result StudioContent
	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		content, err := s.lockContent(ctx, q, contentID)
		if err != nil {
			return err
		}
		author, err := s.authorAccess(ctx, q, contentID, principal.UserID)
		if err != nil {
			return err
		}
		if err := requireOwnerLevel(author); err != nil {
			return err
		}

		kind, err := submitKind(ctx, q, content)
		if err != nil {
			return err
		}

		limit := settings.Int(ctx, q, maxInReviewSetting, defaultMaxInReview)
		inReview, err := q.CountInReviewByOwner(ctx, content.OwnerID)
		if err != nil {
			return fmt.Errorf("content: count in-review contents: %w", err)
		}
		if int(inReview) >= limit {
			return apperr.LimitReached(fmt.Sprintf(
				"Batas konten dalam tinjauan tercapai (%d). Tunggu hasil moderasi konten lain.", limit,
			))
		}

		draft, err := q.GetContentDraft(ctx, contentID)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("content: draft missing for %s", contentID)
		}
		if err != nil {
			return fmt.Errorf("content: load draft: %w", err)
		}

		snapshot, fields, err := s.buildSubmitSnapshot(ctx, q, content, draft)
		if err != nil {
			return err
		}
		if len(fields) > 0 {
			return apperr.Validation(fields...)
		}

		number, err := q.MaxContentVersionNumber(ctx, contentID)
		if err != nil {
			return fmt.Errorf("content: load max version number: %w", err)
		}

		version, err := q.InsertContentVersion(ctx, store.InsertContentVersionParams{
			ContentID:    contentID,
			Number:       number + 1,
			Kind:         kind,
			Title:        snapshot.Title,
			Slug:         snapshot.Slug,
			CategoryID:   snapshot.CategoryID,
			CoverMediaID: snapshot.CoverMediaID,
			Excerpt:      snapshot.Excerpt,
			Body:         snapshot.Body,
			Meta:         snapshot.Meta,
			TagIds:       snapshot.TagIDs,
			ToolIds:      snapshot.ToolIDs,
			SkillIds:     snapshot.SkillIDs,
			BodySchema:   snapshot.BodySchema,
			ContentHash:  snapshot.Hash,
			SubmittedBy:  principal.UserID,
		})
		if err != nil {
			return fmt.Errorf("content: insert version: %w", err)
		}

		for _, mediaID := range snapshot.MediaIDs {
			if err := q.InsertContentVersionMedia(ctx, store.InsertContentVersionMediaParams{
				VersionID: version.ID,
				MediaID:   mediaID,
			}); err != nil {
				return fmt.Errorf("content: insert version media: %w", err)
			}
		}

		queueMajor, err := ownerQueueMajor(ctx, q, content.OwnerID)
		if err != nil {
			return err
		}
		if _, err := q.InsertModerationCase(ctx, store.InsertModerationCaseParams{
			ContentID:    contentID,
			VersionID:    version.ID,
			Kind:         kind,
			QueueMajorID: queueMajor,
		}); err != nil {
			return fmt.Errorf("content: insert moderation case: %w", err)
		}

		if kind == versionKindRevision {
			if err := q.SetContentRevisionInReview(ctx, contentID); err != nil {
				return fmt.Errorf("content: set revision in review: %w", err)
			}
		} else if err := q.SetContentInReview(ctx, contentID); err != nil {
			return fmt.Errorf("content: set content in review: %w", err)
		}

		result, err = s.getStudioQ(ctx, q, contentID, principal.UserID)
		return err
	})
	if err != nil {
		return StudioContent{}, err
	}
	return result, nil
}

// Withdraw pulls a pending submission back while no moderator has claimed it.
func (s *Service) Withdraw(ctx context.Context, actor Actor, contentID uuid.UUID) (StudioContent, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return StudioContent{}, err
	}

	var result StudioContent
	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		content, err := s.lockContent(ctx, q, contentID)
		if err != nil {
			return err
		}
		author, err := s.authorAccess(ctx, q, contentID, principal.UserID)
		if err != nil {
			return err
		}
		if err := requireOwnerLevel(author); err != nil {
			return err
		}

		isRevision := content.Status == StatusPublished &&
			content.RevisionStatus != nil && *content.RevisionStatus == RevisionInReview
		if content.Status != StatusInReview && !isRevision {
			return apperr.InvalidTransition("Tidak ada pengiriman yang sedang ditinjau.")
		}

		version, err := q.GetPendingVersionForUpdate(ctx, contentID)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.InvalidTransition("Tidak ada versi yang sedang ditinjau.")
		}
		if err != nil {
			return fmt.Errorf("content: load pending version: %w", err)
		}

		moderationCase, err := q.GetModerationCaseByVersion(ctx, version.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("content: moderation case missing for version %s", version.ID)
		}
		if err != nil {
			return fmt.Errorf("content: load moderation case: %w", err)
		}
		if moderationCase.Status != "open" {
			return apperr.InvalidTransition("Moderator sudah mulai meninjau, tidak bisa ditarik.")
		}

		if err := q.SetContentVersionWithdrawn(ctx, version.ID); err != nil {
			return fmt.Errorf("content: withdraw version: %w", err)
		}
		if err := q.SetModerationCaseWithdrawn(ctx, moderationCase.ID); err != nil {
			return fmt.Errorf("content: withdraw moderation case: %w", err)
		}

		switch {
		case isRevision:
			if err := q.SetContentRevisionEditing(ctx, contentID); err != nil {
				return fmt.Errorf("content: restore revision editing: %w", err)
			}
		case content.LiveVersionID != nil && content.TakenDownAt != nil:
			if err := q.SetContentWithdrawnToTakenDown(ctx, contentID); err != nil {
				return fmt.Errorf("content: withdraw to taken down: %w", err)
			}
		default:
			if err := q.SetContentWithdrawnToDraft(ctx, contentID); err != nil {
				return fmt.Errorf("content: withdraw to draft: %w", err)
			}
		}

		result, err = s.getStudioQ(ctx, q, contentID, principal.UserID)
		return err
	})
	if err != nil {
		return StudioContent{}, err
	}
	return result, nil
}

// StartRevision copies the live version into the working draft so the owner
// (or an editor) can prepare a revision without touching the published
// snapshot.
func (s *Service) StartRevision(ctx context.Context, actor Actor, contentID uuid.UUID) (StudioContent, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return StudioContent{}, err
	}

	var result StudioContent
	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		content, err := s.lockContent(ctx, q, contentID)
		if err != nil {
			return err
		}
		author, err := s.authorAccess(ctx, q, contentID, principal.UserID)
		if err != nil {
			return err
		}
		switch author.Level {
		case LevelOwner, LevelEditor:
		case LevelContributor:
			return contributorForbidden()
		default:
			return apperr.NotFound()
		}

		if content.Status != StatusPublished {
			return apperr.InvalidTransition("Hanya konten tayang yang bisa direvisi.")
		}
		if content.RevisionStatus != nil {
			return apperr.InvalidTransition("Sudah ada revisi berjalan untuk konten ini.")
		}
		if content.LiveVersionID == nil {
			return apperr.InvalidTransition("Konten belum pernah tayang.")
		}

		version, err := q.GetContentVersionByID(ctx, store.GetContentVersionByIDParams{
			ID:        *content.LiveVersionID,
			ContentID: contentID,
		})
		if err != nil {
			return fmt.Errorf("content: load live version: %w", err)
		}
		if err := restoreDraftFromVersion(ctx, q, contentID, version, principal.UserID); err != nil {
			return err
		}
		if err := q.SetContentRevisionEditing(ctx, contentID); err != nil {
			return fmt.Errorf("content: set revision editing: %w", err)
		}

		result, err = s.getStudioQ(ctx, q, contentID, principal.UserID)
		return err
	})
	if err != nil {
		return StudioContent{}, err
	}
	return result, nil
}

// DiscardRevision drops an unsent revision and restores the draft from the
// live version.
func (s *Service) DiscardRevision(ctx context.Context, actor Actor, contentID uuid.UUID) (StudioContent, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return StudioContent{}, err
	}

	var result StudioContent
	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		content, err := s.lockContent(ctx, q, contentID)
		if err != nil {
			return err
		}
		author, err := s.authorAccess(ctx, q, contentID, principal.UserID)
		if err != nil {
			return err
		}
		if err := requireOwnerLevel(author); err != nil {
			return err
		}

		if content.Status != StatusPublished || content.RevisionStatus == nil {
			return apperr.InvalidTransition("Tidak ada revisi yang bisa dibuang.")
		}
		switch *content.RevisionStatus {
		case RevisionEditing, RevisionChangesRequested:
		case RevisionInReview:
			return apperr.InvalidTransition("Revisi sedang ditinjau, tarik kembali dulu.")
		default:
			return apperr.InvalidTransition("Tidak ada revisi yang bisa dibuang.")
		}
		if content.LiveVersionID == nil {
			return apperr.InvalidTransition("Konten belum pernah tayang.")
		}

		version, err := q.GetContentVersionByID(ctx, store.GetContentVersionByIDParams{
			ID:        *content.LiveVersionID,
			ContentID: contentID,
		})
		if err != nil {
			return fmt.Errorf("content: load live version: %w", err)
		}
		if err := restoreDraftFromVersion(ctx, q, contentID, version, principal.UserID); err != nil {
			return err
		}
		if err := q.ClearContentRevisionStatus(ctx, contentID); err != nil {
			return fmt.Errorf("content: clear revision status: %w", err)
		}

		result, err = s.getStudioQ(ctx, q, contentID, principal.UserID)
		return err
	})
	if err != nil {
		return StudioContent{}, err
	}
	return result, nil
}

// Archive hides a published content from every public surface. An open
// revision draft is discarded first because the DB requires revision_status to
// be NULL outside published.
func (s *Service) Archive(ctx context.Context, actor Actor, contentID uuid.UUID) (StudioContent, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return StudioContent{}, err
	}

	var result StudioContent
	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		content, err := s.lockContent(ctx, q, contentID)
		if err != nil {
			return err
		}
		author, err := s.authorAccess(ctx, q, contentID, principal.UserID)
		if err != nil {
			return err
		}
		if err := requireOwnerLevel(author); err != nil {
			return err
		}

		if content.Status != StatusPublished {
			return apperr.InvalidTransition("Hanya konten tayang yang bisa diarsipkan.")
		}
		if content.RevisionStatus != nil && *content.RevisionStatus == RevisionInReview {
			return apperr.InvalidTransition("Tarik kembali revisi dari moderasi dulu.")
		}
		if content.RevisionStatus != nil {
			if content.LiveVersionID == nil {
				return apperr.InvalidTransition("Konten belum pernah tayang.")
			}
			version, err := q.GetContentVersionByID(ctx, store.GetContentVersionByIDParams{
				ID:        *content.LiveVersionID,
				ContentID: contentID,
			})
			if err != nil {
				return fmt.Errorf("content: load live version: %w", err)
			}
			if err := restoreDraftFromVersion(ctx, q, contentID, version, principal.UserID); err != nil {
				return err
			}
		}

		if err := q.ArchiveContent(ctx, store.ArchiveContentParams{
			ArchivedAt: timePtr(s.now().UTC()),
			ID:         contentID,
		}); err != nil {
			return fmt.Errorf("content: archive content: %w", err)
		}

		result, err = s.getStudioQ(ctx, q, contentID, principal.UserID)
		return err
	})
	if err != nil {
		return StudioContent{}, err
	}
	return result, nil
}

// Unarchive republishes an archived content without review because its live
// version is immutable and was already approved.
func (s *Service) Unarchive(ctx context.Context, actor Actor, contentID uuid.UUID) (StudioContent, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return StudioContent{}, err
	}

	var result StudioContent
	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		content, err := s.lockContent(ctx, q, contentID)
		if err != nil {
			return err
		}
		author, err := s.authorAccess(ctx, q, contentID, principal.UserID)
		if err != nil {
			return err
		}
		if err := requireOwnerLevel(author); err != nil {
			return err
		}

		if content.Status != StatusArchived {
			return apperr.InvalidTransition("Hanya konten arsip yang bisa dibuka.")
		}
		if content.LiveVersionID == nil {
			return apperr.InvalidTransition("Konten belum pernah tayang.")
		}
		version, err := q.GetContentVersionByID(ctx, store.GetContentVersionByIDParams{
			ID:        *content.LiveVersionID,
			ContentID: contentID,
		})
		if err != nil {
			return fmt.Errorf("content: load live version: %w", err)
		}
		if version.State != "approved" {
			return apperr.InvalidTransition("Versi tayang konten ini tidak valid.")
		}

		if err := q.UnarchiveContent(ctx, contentID); err != nil {
			return fmt.Errorf("content: unarchive content: %w", err)
		}

		result, err = s.getStudioQ(ctx, q, contentID, principal.UserID)
		return err
	})
	if err != nil {
		return StudioContent{}, err
	}
	return result, nil
}

// Delete moves a content to the trash for the configured retention period.
func (s *Service) Delete(ctx context.Context, actor Actor, contentID uuid.UUID) error {
	principal, err := s.principal(actor)
	if err != nil {
		return err
	}

	return s.st.WithTx(ctx, func(q *store.Queries) error {
		content, err := s.lockContent(ctx, q, contentID)
		if err != nil {
			return err
		}
		author, err := s.authorAccess(ctx, q, contentID, principal.UserID)
		if err != nil {
			return err
		}
		if err := requireOwnerLevel(author); err != nil {
			return err
		}

		switch {
		case content.Status == StatusDeleted:
			return apperr.InvalidTransition("Konten sudah ada di sampah.")
		case content.Status == StatusInReview:
			return apperr.InvalidTransition("Tarik kembali dari moderasi dulu.")
		case content.Status == StatusPublished && content.RevisionStatus != nil &&
			*content.RevisionStatus == RevisionInReview:
			return apperr.InvalidTransition("Tarik kembali dari moderasi dulu.")
		}

		days := settings.Int(ctx, q, trashRetentionSetting, defaultTrashRetention)
		now := s.now().UTC()
		if err := q.SoftDeleteContent(ctx, store.SoftDeleteContentParams{
			DeletedAt:  timePtr(now),
			PurgeAfter: timePtr(now.AddDate(0, 0, days)),
			ID:         contentID,
		}); err != nil {
			return fmt.Errorf("content: soft delete content: %w", err)
		}
		return nil
	})
}

// Restore brings a trashed content back to its previous status while the
// retention window is still open.
func (s *Service) Restore(ctx context.Context, actor Actor, contentID uuid.UUID) (StudioContent, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return StudioContent{}, err
	}

	var result StudioContent
	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		content, err := s.lockContent(ctx, q, contentID)
		if err != nil {
			return err
		}
		author, err := s.authorAccess(ctx, q, contentID, principal.UserID)
		if err != nil {
			return err
		}
		if err := requireOwnerLevel(author); err != nil {
			return err
		}

		if content.Status != StatusDeleted {
			return apperr.InvalidTransition("Konten tidak ada di sampah.")
		}
		if content.PurgeAfter == nil || !content.PurgeAfter.After(s.now().UTC()) {
			return apperr.InvalidTransition("Masa pemulihan konten sudah lewat.")
		}

		rows, err := q.RestoreContent(ctx, contentID)
		if err != nil {
			return fmt.Errorf("content: restore content: %w", err)
		}
		if rows == 0 {
			return apperr.InvalidTransition("Masa pemulihan konten sudah lewat.")
		}

		result, err = s.getStudioQ(ctx, q, contentID, principal.UserID)
		return err
	})
	if err != nil {
		return StudioContent{}, err
	}
	return result, nil
}

type submitSnapshot struct {
	Title        string
	Slug         string
	CategoryID   *uuid.UUID
	CoverMediaID *uuid.UUID
	Excerpt      *string
	Body         json.RawMessage
	Meta         json.RawMessage
	TagIDs       []uuid.UUID
	ToolIDs      []uuid.UUID
	SkillIDs     []uuid.UUID
	BodySchema   int16
	MediaIDs     []uuid.UUID
	Hash         []byte
}

// buildSubmitSnapshot validates the draft completeness and builds the
// immutable values stored on the version. It collects every validation error
// instead of returning on the first one.
func (s *Service) buildSubmitSnapshot(
	ctx context.Context,
	q *store.Queries,
	content store.Content,
	draft store.ContentDraft,
) (submitSnapshot, []apperr.FieldError, error) {
	var fields []apperr.FieldError

	bodyResult, bodyFields := validateDraftBodyForSubmit(content.Type, draft.Body)
	fields = append(fields, bodyFields...)
	if bodyResult.Body == nil {
		bodyResult.Body = body.EmptyBody(body.ContentType(content.Type))
	}

	title := strings.TrimSpace(draft.Title)
	if runes := utf8.RuneCountInString(title); runes < minTitleRunes || runes > maxTitleRunes {
		fields = append(fields, apperr.Field(
			"title", "invalid_length", "Judul harus 3–150 karakter.",
		))
	}

	categoryID := draft.CategoryID
	if categoryID == nil {
		fields = append(fields, apperr.Field(
			"categoryId", "required", "Pilih kategori konten.",
		))
	} else {
		category, err := q.GetContentCategoryByID(ctx, *categoryID)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			fields = append(fields, invalidCategoryField())
		case err != nil:
			return submitSnapshot{}, nil, fmt.Errorf("content: load category: %w", err)
		case !category.IsActive || category.ContentType != content.Type:
			fields = append(fields, invalidCategoryField())
		}
	}

	coverID := draft.CoverMediaID
	if content.Type == TypePortfolio && coverID == nil {
		fields = append(fields, apperr.Field(
			"coverMediaId", "required", "Sampul wajib dipilih untuk portfolio.",
		))
	}

	excerpt := normalizeSubmitExcerpt(content.Type, draft.Excerpt, &fields)

	rawMeta := decodeObject(draft.Meta)
	meta, _, metaFields := validateMeta(content.Type, rawMeta)
	fields = append(fields, metaFields...)
	if content.Type == TypePortfolio {
		contextValue, _ := rawMeta["context"].(string)
		if strings.TrimSpace(contextValue) == "" {
			fields = append(fields, apperr.Field(
				"meta.context", "required", "Konteks karya wajib diisi.",
			))
		}
	}

	mediaIDs, err := collectSubmitMedia(ctx, q, draft, bodyResult.MediaIDs)
	if err != nil {
		return submitSnapshot{}, nil, err
	}
	mediaFields, err := validateSubmitMedia(ctx, q, mediaIDs)
	if err != nil {
		return submitSnapshot{}, nil, err
	}
	fields = append(fields, mediaFields...)

	if len(fields) > 0 {
		return submitSnapshot{}, fields, nil
	}

	slug := Slugify(title, maxSlugRunes)
	if slug == "" {
		slug = "tanpa-judul"
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return submitSnapshot{}, nil, fmt.Errorf("content: encode meta: %w", err)
	}

	hashSnapshot := map[string]any{
		"title":          title,
		"slug":           slug,
		"category_id":    categoryID,
		"cover_media_id": coverID,
		"excerpt":        excerpt,
		"body":           json.RawMessage(bodyResult.Body),
		"meta":           json.RawMessage(metaJSON),
		"tag_ids":        draft.TagIds,
		"tool_ids":       draft.ToolIds,
		"skill_ids":      draft.SkillIds,
	}
	hash, err := body.CanonicalHash(hashSnapshot)
	if err != nil {
		return submitSnapshot{}, nil, fmt.Errorf("content: hash snapshot: %w", err)
	}

	return submitSnapshot{
		Title:        title,
		Slug:         slug,
		CategoryID:   categoryID,
		CoverMediaID: coverID,
		Excerpt:      excerpt,
		Body:         bodyResult.Body,
		Meta:         metaJSON,
		TagIDs:       draft.TagIds,
		ToolIDs:      draft.ToolIds,
		SkillIDs:     draft.SkillIds,
		BodySchema:   draft.BodySchema,
		MediaIDs:     mediaIDs,
		Hash:         hash,
	}, nil, nil
}

func submitKind(ctx context.Context, q *store.Queries, content store.Content) (string, error) {
	switch content.Status {
	case StatusDraft:
		count, err := q.CountContentVersions(ctx, content.ID)
		if err != nil {
			return "", fmt.Errorf("content: count versions: %w", err)
		}
		if count == 0 {
			return versionKindInitial, nil
		}
		return versionKindResubmission, nil
	case StatusChangesRequested, StatusTakenDown:
		return versionKindResubmission, nil
	case StatusPublished:
		if content.RevisionStatus != nil {
			switch *content.RevisionStatus {
			case RevisionEditing, RevisionChangesRequested:
				return versionKindRevision, nil
			case RevisionInReview:
				return "", apperr.InvalidTransition("Revisi sedang ditinjau.")
			}
		}
		return "", apperr.InvalidTransition("Mulai revisi untuk mengirim perubahan konten yang sudah tayang.")
	case StatusInReview:
		return "", apperr.InvalidTransition("Konten sedang ditinjau.")
	case StatusRejected:
		return "", apperr.InvalidTransition("Konten ditolak dan tidak bisa dikirim ulang.")
	case StatusArchived:
		return "", apperr.InvalidTransition("Buka arsip dulu.")
	case StatusDeleted:
		return "", apperr.InvalidTransition("Pulihkan dulu dari sampah.")
	default:
		return "", apperr.InvalidTransition("Konten tidak bisa dikirim pada status ini.")
	}
}

func normalizeSubmitExcerpt(contentType string, raw *string, fields *[]apperr.FieldError) *string {
	if contentType == TypeBlog {
		value := ""
		if raw != nil {
			value = strings.TrimSpace(*raw)
		}
		runes := utf8.RuneCountInString(value)
		switch {
		case runes == 0:
			*fields = append(*fields, apperr.Field(
				"excerpt", "required", "Ringkasan blog wajib diisi (50–300 karakter).",
			))
			return nil
		case runes < minBlogExcerptRunes || runes > maxExcerptRunes:
			*fields = append(*fields, apperr.Field(
				"excerpt", "invalid_length", "Ringkasan harus 50–300 karakter.",
			))
			return nil
		}
		return &value
	}

	if raw == nil {
		return nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil
	}
	if utf8.RuneCountInString(value) > maxExcerptRunes {
		*fields = append(*fields, apperr.Field("excerpt", "too_long", "Ringkasan maksimal 300 karakter."))
		return nil
	}
	return &value
}

func validateDraftBodyForSubmit(contentType string, raw json.RawMessage) (body.Result, []apperr.FieldError) {
	bodyType := body.ContentType(contentType)
	if len(raw) == 0 {
		raw = body.EmptyBody(bodyType)
	}

	result, errs := body.ValidateForSubmit(bodyType, raw)
	for i := range errs {
		errs[i].Field = prefixBodyField(errs[i].Field)
	}
	if result.Body == nil && len(errs) == 0 {
		result.Body = body.EmptyBody(bodyType)
	}
	return result, errs
}

// collectSubmitMedia unions the draft media table, the cover, and the media
// referenced by the normalized body, then returns a deterministic deduplicated
// list.
func collectSubmitMedia(
	ctx context.Context,
	q *store.Queries,
	draft store.ContentDraft,
	bodyMediaIDs []uuid.UUID,
) ([]uuid.UUID, error) {
	draftMediaIDs, err := q.ListContentDraftMedia(ctx, draft.ContentID)
	if err != nil {
		return nil, fmt.Errorf("content: load draft media: %w", err)
	}

	seen := make(map[uuid.UUID]struct{}, len(draftMediaIDs)+len(bodyMediaIDs)+1)
	mediaIDs := make([]uuid.UUID, 0, len(draftMediaIDs)+len(bodyMediaIDs)+1)
	add := func(id uuid.UUID) {
		if _, exists := seen[id]; exists {
			return
		}
		seen[id] = struct{}{}
		mediaIDs = append(mediaIDs, id)
	}
	for _, id := range draftMediaIDs {
		add(id)
	}
	for _, id := range bodyMediaIDs {
		add(id)
	}
	if draft.CoverMediaID != nil {
		add(*draft.CoverMediaID)
	}
	return mediaIDs, nil
}

func validateSubmitMedia(ctx context.Context, q *store.Queries, mediaIDs []uuid.UUID) ([]apperr.FieldError, error) {
	if len(mediaIDs) == 0 {
		return nil, nil
	}

	rows, err := q.ListMediaStatuses(ctx, mediaIDs)
	if err != nil {
		return nil, fmt.Errorf("content: load media statuses: %w", err)
	}
	found := make(map[uuid.UUID]store.ListMediaStatusesRow, len(rows))
	for _, row := range rows {
		found[row.ID] = row
	}

	processing := false
	for _, id := range mediaIDs {
		row, ok := found[id]
		if !ok || row.DeletedAt != nil || row.Status == "failed" {
			return []apperr.FieldError{apperr.Field(
				"media", "media_failed", "Ada media yang gagal diproses, hapus atau unggah ulang.",
			)}, nil
		}
		if row.Status != "ready" {
			processing = true
		}
	}
	if processing {
		return []apperr.FieldError{apperr.Field(
			"media", "media_processing", "Masih ada media yang belum selesai diproses.",
		)}, nil
	}
	return nil, nil
}

func ownerQueueMajor(ctx context.Context, q *store.Queries, ownerID uuid.UUID) (*uuid.UUID, error) {
	majorID, err := q.GetOwnerQueueMajor(ctx, ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("content: load owner major: %w", err)
	}
	return &majorID, nil
}

func (s *Service) lockContent(ctx context.Context, q *store.Queries, id uuid.UUID) (store.Content, error) {
	content, err := q.LockContent(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Content{}, apperr.NotFound()
	}
	if err != nil {
		return store.Content{}, fmt.Errorf("content: lock content %s: %w", id, err)
	}
	return content, nil
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func requireOwnerLevel(author store.ContentAuthor) error {
	if author.Level == LevelOwner {
		return nil
	}
	return ownerForbidden()
}

func ownerForbidden() error {
	return &apperr.Error{
		Status: http.StatusForbidden,
		Code:   "FORBIDDEN_LEVEL",
		Title:  "Hanya pemilik yang bisa melakukan ini.",
	}
}

func restoreDraftFromVersion(
	ctx context.Context,
	q *store.Queries,
	contentID uuid.UUID,
	version store.ContentVersion,
	updatedBy uuid.UUID,
) error {
	baseVersionID := version.ID
	if err := q.UpdateContentDraftFromVersion(ctx, store.UpdateContentDraftFromVersionParams{
		BaseVersionID: &baseVersionID,
		Title:         version.Title,
		CategoryID:    version.CategoryID,
		CoverMediaID:  version.CoverMediaID,
		Excerpt:       version.Excerpt,
		Body:          version.Body,
		Meta:          version.Meta,
		TagIds:        version.TagIds,
		ToolIds:       version.ToolIds,
		SkillIds:      version.SkillIds,
		BodySchema:    version.BodySchema,
		UpdatedBy:     updatedBy,
		ContentID:     contentID,
	}); err != nil {
		return fmt.Errorf("content: restore draft from version: %w", err)
	}
	if err := q.DeleteContentDraftMedia(ctx, contentID); err != nil {
		return fmt.Errorf("content: clear draft media: %w", err)
	}
	if err := q.CopyContentVersionMediaToDraft(ctx, store.CopyContentVersionMediaToDraftParams{
		ContentID: contentID,
		VersionID: version.ID,
	}); err != nil {
		return fmt.Errorf("content: copy version media to draft: %w", err)
	}
	return nil
}
