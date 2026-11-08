package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/content/body"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	maxTitleRunes   = 150
	maxExcerptRunes = 300
	maxTags         = 10
	maxTermSlugs    = 20
	maxTagSlugRunes = 40
	maxSlugRunes    = 120
)

// SaveDraft replaces the editable draft of a content.
func (s *Service) SaveDraft(ctx context.Context, actor Actor, contentID uuid.UUID, input DraftInput) (StudioContent, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return StudioContent{}, err
	}

	var result StudioContent
	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		content, err := s.loadContent(ctx, q, contentID)
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

		if err := editStatusError(content); err != nil {
			return err
		}

		draft, err := q.GetContentDraft(ctx, contentID)
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("content: draft missing for %s", contentID)
		}
		if err != nil {
			return fmt.Errorf("content: load draft: %w", err)
		}
		if int(draft.LockVersion) != input.LockVersion {
			return s.lockConflict(ctx, q, draft)
		}

		var fields []apperr.FieldError

		title := strings.TrimSpace(input.Title)
		if utf8.RuneCountInString(title) > maxTitleRunes {
			fields = append(fields, apperr.Field("title", "too_long", "Judul maksimal 150 karakter."))
		}

		excerpt := normalizeExcerpt(input.Excerpt, &fields)

		normalized, bodyErrs := normalizeDraftBody(content.Type, input.Body)
		if len(bodyErrs) > 0 {
			return apperr.Validation(bodyErrs...)
		}

		categoryID := input.CategoryID
		if categoryID != nil {
			category, err := q.GetContentCategoryByID(ctx, *categoryID)
			if errors.Is(err, pgx.ErrNoRows) {
				fields = append(fields, invalidCategoryField())
			} else if err != nil {
				return fmt.Errorf("content: load category: %w", err)
			} else if !category.IsActive || category.ContentType != content.Type {
				fields = append(fields, invalidCategoryField())
			}
		}

		coverID := input.CoverMediaID
		if coverID != nil {
			if _, err := q.GetAuthorOwnedImage(ctx, store.GetAuthorOwnedImageParams{
				MediaID:   *coverID,
				ContentID: contentID,
			}); errors.Is(err, pgx.ErrNoRows) {
				fields = append(fields, apperr.Field(
					"coverMediaId", "invalid_media", "Sampul harus gambar milik salah satu author konten ini.",
				))
			} else if err != nil {
				return fmt.Errorf("content: validate cover media: %w", err)
			}
		}

		tagIDs, _, tagErrs := resolveTags(ctx, q, input.Tags)
		fields = append(fields, tagErrs...)

		toolIDs, _, toolErrs := resolveTerms(ctx, q, "tools", input.Tools)
		fields = append(fields, toolErrs...)

		skillIDs, _, skillErrs := resolveTerms(ctx, q, "skills", input.Skills)
		fields = append(fields, skillErrs...)

		meta, seriesID, metaErrs := validateMeta(content.Type, input.Meta)
		fields = append(fields, metaErrs...)
		if seriesID != nil {
			ownerID, err := q.GetBlogSeriesOwner(ctx, *seriesID)
			switch {
			case errors.Is(err, pgx.ErrNoRows):
				fields = append(fields, apperr.Field(
					"meta.seriesId", "invalid_series", "Seri tidak ditemukan atau bukan milik pemilik konten.",
				))
			case err != nil:
				return fmt.Errorf("content: load blog series: %w", err)
			case ownerID != content.OwnerID:
				fields = append(fields, apperr.Field(
					"meta.seriesId", "invalid_series", "Seri tidak ditemukan atau bukan milik pemilik konten.",
				))
			}
		}

		if len(fields) > 0 {
			return apperr.Validation(fields...)
		}

		metaJSON, err := json.Marshal(meta)
		if err != nil {
			return fmt.Errorf("content: encode meta: %w", err)
		}

		rows, err := q.UpdateContentDraft(ctx, store.UpdateContentDraftParams{
			Title:        title,
			CategoryID:   categoryID,
			CoverMediaID: coverID,
			Excerpt:      excerpt,
			Body:         normalized.Body,
			Meta:         metaJSON,
			TagIds:       tagIDs,
			ToolIds:      toolIDs,
			SkillIds:     skillIDs,
			UpdatedBy:    principal.UserID,
			ContentID:    contentID,
			LockVersion:  draft.LockVersion,
		})
		if err != nil {
			return fmt.Errorf("content: update draft: %w", err)
		}
		if rows == 0 {
			current, err := q.GetContentDraft(ctx, contentID)
			if err != nil {
				return fmt.Errorf("content: reload draft after conflict: %w", err)
			}
			return s.lockConflict(ctx, q, current)
		}

		if err := replaceDraftMedia(ctx, q, contentID, normalized.MediaIDs, coverID); err != nil {
			return err
		}

		if content.LiveVersionID == nil {
			contentTitle := title
			if contentTitle == "" {
				contentTitle = "Tanpa judul"
			}
			slug := Slugify(contentTitle, maxSlugRunes)
			if slug == "" {
				slug = "tanpa-judul"
			}
			if err := q.UpdateContentFromDraft(ctx, store.UpdateContentFromDraftParams{
				Title:        contentTitle,
				Slug:         slug,
				CategoryID:   categoryID,
				CoverMediaID: coverID,
				Excerpt:      excerpt,
				ID:           contentID,
			}); err != nil {
				return fmt.Errorf("content: update content from draft: %w", err)
			}
		} else if err := q.TouchContent(ctx, contentID); err != nil {
			return fmt.Errorf("content: touch content: %w", err)
		}

		result, err = s.getStudioQ(ctx, q, contentID, principal.UserID)
		return err
	})
	if err != nil {
		return StudioContent{}, err
	}
	return result, nil
}

func normalizeExcerpt(raw *string, fields *[]apperr.FieldError) *string {
	if raw == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*raw)
	if utf8.RuneCountInString(trimmed) > maxExcerptRunes {
		*fields = append(*fields, apperr.Field("excerpt", "too_long", "Ringkasan maksimal 300 karakter."))
		return nil
	}
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func invalidCategoryField() apperr.FieldError {
	return apperr.Field("categoryId", "invalid_category", "Kategori tidak ditemukan, nonaktif, atau tidak sesuai tipe konten.")
}

func normalizeDraftBody(contentType string, input map[string]any) (body.Result, []apperr.FieldError) {
	bodyType := body.ContentType(contentType)
	raw := body.EmptyBody(bodyType)
	if len(input) > 0 {
		encoded, err := json.Marshal(input)
		if err != nil {
			return body.Result{}, []apperr.FieldError{
				apperr.Field("body", "invalid_json", "Body konten bukan JSON yang valid."),
			}
		}
		raw = encoded
	}

	result, errs := body.Normalize(bodyType, raw)
	for i := range errs {
		errs[i].Field = prefixBodyField(errs[i].Field)
	}
	if result.Body == nil && len(errs) == 0 {
		result.Body = body.EmptyBody(bodyType)
	}
	return result, errs
}

func prefixBodyField(field string) string {
	if field == "body" || strings.HasPrefix(field, "body.") {
		return field
	}
	return "body." + field
}

func replaceDraftMedia(ctx context.Context, q *store.Queries, contentID uuid.UUID, mediaIDs []uuid.UUID, coverID *uuid.UUID) error {
	if err := q.DeleteContentDraftMedia(ctx, contentID); err != nil {
		return fmt.Errorf("content: clear draft media: %w", err)
	}

	seen := make(map[uuid.UUID]struct{}, len(mediaIDs)+1)
	insert := func(id uuid.UUID) error {
		if _, exists := seen[id]; exists {
			return nil
		}
		seen[id] = struct{}{}
		return q.InsertContentDraftMedia(ctx, store.InsertContentDraftMediaParams{
			ContentID: contentID,
			MediaID:   id,
		})
	}
	for _, id := range mediaIDs {
		if err := insert(id); err != nil {
			return fmt.Errorf("content: insert draft media: %w", err)
		}
	}
	if coverID != nil {
		if err := insert(*coverID); err != nil {
			return fmt.Errorf("content: insert cover media: %w", err)
		}
	}
	return nil
}

func editStatusError(content store.Content) error {
	switch content.Status {
	case StatusDraft, StatusChangesRequested, StatusTakenDown:
		return nil
	case StatusInReview:
		return apperr.InvalidTransition("Konten sedang ditinjau, tarik kembali dulu untuk mengedit.")
	case StatusPublished:
		if content.RevisionStatus != nil {
			switch *content.RevisionStatus {
			case RevisionEditing, RevisionChangesRequested:
				return nil
			case RevisionInReview:
				return apperr.InvalidTransition("Revisi sedang ditinjau, tarik kembali dulu untuk mengedit.")
			}
		}
		return apperr.InvalidTransition("Mulai revisi untuk mengubah konten yang sudah tayang.")
	case StatusRejected:
		return apperr.InvalidTransition("Konten ditolak dan tidak bisa diubah.")
	case StatusArchived:
		return apperr.InvalidTransition("Buka arsip dulu.")
	case StatusDeleted:
		return apperr.InvalidTransition("Pulihkan dulu dari sampah.")
	default:
		return apperr.InvalidTransition("Konten tidak bisa diedit pada status ini.")
	}
}

func contributorForbidden() error {
	return &apperr.Error{
		Status: http.StatusForbidden,
		Code:   "FORBIDDEN_LEVEL",
		Title:  "Kamu hanya kontributor, tidak bisa mengedit.",
	}
}

func (s *Service) lockConflict(ctx context.Context, q *store.Queries, draft store.ContentDraft) error {
	name := "pengguna lain"
	if author, err := q.GetAuthorMini(ctx, draft.UpdatedBy); err == nil {
		name = author.DisplayName
	}
	return apperr.LockConflict().WithDetail(fmt.Sprintf(
		"Draf sudah diubah oleh %s pada %s.",
		name,
		draft.UpdatedAt.UTC().Format("2006-01-02 15:04"),
	))
}

func resolveTags(ctx context.Context, q *store.Queries, names []string) ([]uuid.UUID, []TaxonomyTerm, []apperr.FieldError) {
	var errs []apperr.FieldError
	if len(names) > maxTags {
		errs = append(errs, apperr.Field("tags", "too_many", "Tag maksimal 10."))
	}

	ids := make([]uuid.UUID, 0, len(names))
	terms := make([]TaxonomyTerm, 0, len(names))
	seen := make(map[string]struct{}, len(names))

	for i, name := range names {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" || utf8.RuneCountInString(trimmed) > maxTagSlugRunes {
			errs = append(errs, apperr.Field(tagField(i), "invalid_length", "Nama tag harus 1–40 karakter."))
			continue
		}
		slug := Slugify(trimmed, maxTagSlugRunes)
		if slug == "" {
			errs = append(errs, apperr.Field(tagField(i), "invalid_length", "Nama tag harus 1–40 karakter."))
			continue
		}
		if _, exists := seen[slug]; exists {
			continue
		}
		seen[slug] = struct{}{}

		tag, err := q.GetTagBySlug(ctx, slug)
		if errors.Is(err, pgx.ErrNoRows) {
			if err := q.InsertTag(ctx, store.InsertTagParams{Slug: slug, Name: trimmed}); err != nil {
				errs = append(errs, apperr.Field(tagField(i), "invalid_tag", "Tag tidak dapat disimpan."))
				continue
			}
			tag, err = q.GetTagBySlug(ctx, slug)
		}
		if err != nil {
			errs = append(errs, apperr.Field(tagField(i), "invalid_tag", "Tag tidak dapat disimpan."))
			continue
		}
		if tag.IsBlocked {
			errs = append(errs, apperr.Field(tagField(i), "blocked", "Tag ini tidak diizinkan."))
			continue
		}
		ids = append(ids, tag.ID)
		terms = append(terms, TaxonomyTerm{Slug: tag.Slug, Name: tag.Name})
	}

	return ids, terms, errs
}

func tagField(index int) string {
	return "tags[" + strconv.Itoa(index) + "]"
}

// resolveTerms resolves tool or skill slugs (kind is "tools" or "skills") to
// their ids and display terms. Slugs are matched case-insensitively by the
// input normalizer below, and inactive or unknown slugs are field errors.
func resolveTerms(ctx context.Context, q *store.Queries, kind string, slugs []string) ([]uuid.UUID, []TaxonomyTerm, []apperr.FieldError) {
	var errs []apperr.FieldError
	if len(slugs) > maxTermSlugs {
		errs = append(errs, apperr.Field(kind, "too_many", "Maksimal 20 item."))
	}

	normalized := make([]string, 0, len(slugs))
	seenInput := make(map[string]struct{}, len(slugs))
	for _, slug := range slugs {
		trimmed := strings.ToLower(strings.TrimSpace(slug))
		if trimmed == "" {
			continue
		}
		if _, exists := seenInput[trimmed]; exists {
			continue
		}
		seenInput[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	type termRow struct {
		id       uuid.UUID
		slug     string
		name     string
		category string
	}
	found := make(map[string]termRow, len(normalized))
	switch kind {
	case "tools":
		rows, err := q.GetToolsBySlugs(ctx, normalized)
		if err != nil {
			return nil, nil, []apperr.FieldError{apperr.Field(kind, "lookup_failed", "Tool tidak dapat dimuat.")}
		}
		for _, row := range rows {
			found[row.Slug] = termRow{id: row.ID, slug: row.Slug, name: row.Name, category: row.Category}
		}
	case "skills":
		rows, err := q.GetSkillsBySlugs(ctx, normalized)
		if err != nil {
			return nil, nil, []apperr.FieldError{apperr.Field(kind, "lookup_failed", "Skill tidak dapat dimuat.")}
		}
		for _, row := range rows {
			found[row.Slug] = termRow{id: row.ID, slug: row.Slug, name: row.Name, category: row.Category}
		}
	default:
		return nil, nil, nil
	}

	ids := make([]uuid.UUID, 0, len(normalized))
	terms := make([]TaxonomyTerm, 0, len(normalized))
	seen := make(map[uuid.UUID]struct{}, len(normalized))
	for i, slug := range slugs {
		trimmed := strings.ToLower(strings.TrimSpace(slug))
		if trimmed == "" {
			continue
		}
		row, ok := found[trimmed]
		if !ok {
			message := "Tool tidak ditemukan atau tidak aktif."
			if kind == "skills" {
				message = "Skill tidak ditemukan atau tidak aktif."
			}
			errs = append(errs, apperr.Field(indexedField(kind, i), "not_found", message))
			continue
		}
		if _, exists := seen[row.id]; exists {
			continue
		}
		seen[row.id] = struct{}{}
		ids = append(ids, row.id)
		category := row.category
		terms = append(terms, TaxonomyTerm{Slug: row.slug, Name: row.name, Category: &category})
	}

	return ids, terms, errs
}

func indexedField(kind string, index int) string {
	return kind + "[" + strconv.Itoa(index) + "]"
}
