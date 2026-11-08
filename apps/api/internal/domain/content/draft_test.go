package content

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestSaveDraft_HappyPathStoresNormalizedDraftAndDenormalizesContent(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)

	bodyMedia := f.insertImage(f.ownerID)
	coverMedia := f.insertImage(f.ownerID)
	blockID := uuid.NewString()
	input := DraftInput{
		LockVersion:  1,
		Title:        "Poster HUT ke-80 SMKN 4 Malang",
		Excerpt:      strPtr("Ringkasan karya"),
		CategoryID:   &f.portfolioCategoryID,
		CoverMediaID: &coverMedia,
		Body: map[string]any{
			"blocks": []any{
				map[string]any{
					"id":   blockID,
					"type": "image",
					"data": map[string]any{"media_id": bodyMedia.String(), "alt": "Poster"},
				},
			},
		},
		Meta: map[string]any{
			"context": "competition",
			"period":  map[string]any{"start": "2026-08", "end": "2026-10"},
			"links": []any{
				map[string]any{"label": "Demo", "url": "https://example.com/demo"},
			},
		},
		Tags:   []string{"Desain Grafis", "desain grafis", "Poster"},
		Tools:  []string{"adobe-photoshop"},
		Skills: []string{"fotografi"},
	}

	result, err := f.service.SaveDraft(f.ctx, f.actor(f.ownerID), created.ID, input)
	if err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}

	if result.Draft.LockVersion != 2 {
		t.Errorf("lockVersion = %d, mau 2", result.Draft.LockVersion)
	}
	if result.Draft.Title != input.Title {
		t.Errorf("draft title = %q, mau %q", result.Draft.Title, input.Title)
	}
	if len(result.Draft.Tags) != 2 {
		t.Fatalf("tags = %+v, mau 2 setelah dedupe", result.Draft.Tags)
	}
	names := map[string]bool{}
	for _, tag := range result.Draft.Tags {
		names[tag.Name] = true
	}
	if !names["Desain Grafis"] || !names["Poster"] {
		t.Errorf("tags = %+v, mau Desain Grafis dan Poster", result.Draft.Tags)
	}
	if len(result.Draft.Tools) != 1 || result.Draft.Tools[0].Slug != "adobe-photoshop" {
		t.Errorf("tools = %+v, mau adobe-photoshop", result.Draft.Tools)
	}
	if result.Draft.Tools[0].Category == nil || *result.Draft.Tools[0].Category != "design_software" {
		t.Errorf("tool category = %v, mau design_software", result.Draft.Tools[0].Category)
	}
	if len(result.Draft.Skills) != 1 || result.Draft.Skills[0].Slug != "fotografi" {
		t.Errorf("skills = %+v, mau fotografi", result.Draft.Skills)
	}
	if result.Draft.Cover == nil || result.Draft.Cover.MediaID != coverMedia {
		t.Errorf("cover = %+v, mau %s", result.Draft.Cover, coverMedia)
	}
	if result.Draft.UpdatedBy == nil || result.Draft.UpdatedBy.UserID != f.ownerID {
		t.Errorf("updatedBy = %+v, mau owner", result.Draft.UpdatedBy)
	}

	content := f.contentRow(created.ID)
	if content.Title != input.Title {
		t.Errorf("contents.title = %q, mau %q", content.Title, input.Title)
	}
	if content.Slug != "poster-hut-ke-80-smkn-4-malang" {
		t.Errorf("contents.slug = %q, mau poster-hut-ke-80-smkn-4-malang", content.Slug)
	}
	if content.CategoryID == nil || *content.CategoryID != f.portfolioCategoryID {
		t.Errorf("contents.category_id = %v, mau %s", content.CategoryID, f.portfolioCategoryID)
	}
	if content.CoverMediaID == nil || *content.CoverMediaID != coverMedia {
		t.Errorf("contents.cover_media_id = %v, mau %s", content.CoverMediaID, coverMedia)
	}
	if content.Excerpt == nil || *content.Excerpt != "Ringkasan karya" {
		t.Errorf("contents.excerpt = %v, mau Ringkasan karya", content.Excerpt)
	}

	if got := countRows(t, f.pool, `SELECT count(*) FROM content_draft_media WHERE content_id = $1`, created.ID); got != 2 {
		t.Errorf("content_draft_media rows = %d, mau 2 (body + cover)", got)
	}
	if got := countRows(t, f.pool, `SELECT count(*) FROM tags WHERE slug = 'poster' AND name = 'Poster'`); got != 1 {
		t.Errorf("tag poster rows = %d, mau 1", got)
	}

	draft, err := f.store.GetContentDraft(f.ctx, created.ID)
	if err != nil {
		t.Fatalf("load draft: %v", err)
	}
	if len(draft.TagIds) != 2 || len(draft.ToolIds) != 1 || len(draft.SkillIds) != 1 {
		t.Errorf("array ids = tags %d tools %d skills %d", len(draft.TagIds), len(draft.ToolIds), len(draft.SkillIds))
	}

	// A second save replaces the draft media set instead of appending to it.
	input.LockVersion = 2
	input.CoverMediaID = nil
	if _, err := f.service.SaveDraft(f.ctx, f.actor(f.ownerID), created.ID, input); err != nil {
		t.Fatalf("SaveDraft kedua: %v", err)
	}
	if got := countRows(t, f.pool, `SELECT count(*) FROM content_draft_media WHERE content_id = $1`, created.ID); got != 1 {
		t.Errorf("content_draft_media rows setelah simpan ulang = %d, mau 1", got)
	}
}

func TestSaveDraft_StaleLockVersionReturnsConflictWithEditorName(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.addAuthor(created.ID, f.editorID, LevelEditor)

	saved, err := f.service.SaveDraft(f.ctx, f.actor(f.editorID), created.ID, f.draftInput("Judul Editor"))
	if err != nil {
		t.Fatalf("editor SaveDraft: %v", err)
	}
	if saved.Draft.LockVersion != 2 || saved.Draft.UpdatedBy == nil || saved.Draft.UpdatedBy.UserID != f.editorID {
		t.Fatalf("draft setelah editor = lock %d updatedBy %+v", saved.Draft.LockVersion, saved.Draft.UpdatedBy)
	}

	_, err = f.service.SaveDraft(f.ctx, f.actor(f.ownerID), created.ID, f.draftInput("Judul Pemilik"))

	appErr := requireAppError(t, err, "LOCK_CONFLICT")
	if !strings.Contains(appErr.Detail, "Editor Karya") {
		t.Errorf("detail = %q, mau menyebut nama editor", appErr.Detail)
	}
}

func TestSaveDraft_InvalidBodyReturnsValidationWithBodyPrefix(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)

	input := f.draftInput("Judul")
	input.Body = map[string]any{"blocks": "bukan array"}

	_, err := f.service.SaveDraft(f.ctx, f.actor(f.ownerID), created.ID, input)

	appErr := requireAppError(t, err, "VALIDATION_FAILED")
	found := false
	for _, field := range appErr.Fields {
		if strings.HasPrefix(field.Field, "body.") {
			found = true
		}
	}
	if !found {
		t.Fatalf("fields = %+v, mau ada field berawalan body.", appErr.Fields)
	}
}

func TestSaveDraft_CategoryOfAnotherTypeReturnsFieldError(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)

	input := f.draftInput("Judul")
	input.CategoryID = &f.blogCategoryID

	_, err := f.service.SaveDraft(f.ctx, f.actor(f.ownerID), created.ID, input)

	appErr := requireAppError(t, err, "VALIDATION_FAILED")
	requireFields(t, appErr, "categoryId")
}

func TestSaveDraft_CoverMediaOfNonAuthorReturnsFieldError(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	otherMedia := f.insertImage(f.outsiderID)

	input := f.draftInput("Judul")
	input.CoverMediaID = &otherMedia

	_, err := f.service.SaveDraft(f.ctx, f.actor(f.ownerID), created.ID, input)

	appErr := requireAppError(t, err, "VALIDATION_FAILED")
	requireFields(t, appErr, "coverMediaId")
}

func TestSaveDraft_BlockedTagReturnsFieldError(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.blockedTag("terlarang", "Terlarang")

	input := f.draftInput("Judul")
	input.Tags = []string{"Terlarang"}

	_, err := f.service.SaveDraft(f.ctx, f.actor(f.ownerID), created.ID, input)

	appErr := requireAppError(t, err, "VALIDATION_FAILED")
	requireFields(t, appErr, "tags[0]")
	if got := fieldErrorMessage(appErr, "tags[0]"); got != "Tag ini tidak diizinkan." {
		t.Errorf("message = %q", got)
	}
}

func TestSaveDraft_PortfolioMetaErrors(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)

	input := f.draftInput("Judul")
	input.Meta = map[string]any{
		"context": "personal",
		"period":  map[string]any{"start": "2026-10", "end": "2026-08"},
		"links": []any{
			map[string]any{"label": "", "url": "http://example.com"},
		},
	}

	_, err := f.service.SaveDraft(f.ctx, f.actor(f.ownerID), created.ID, input)

	appErr := requireAppError(t, err, "VALIDATION_FAILED")
	requireFields(t, appErr, "meta.links[0].label", "meta.links[0].url", "meta.period.end")
}

func TestSaveDraft_BlogSeriesOfAnotherOwnerReturnsFieldError(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypeBlog)

	var seriesID uuid.UUID
	err := f.pool.QueryRow(f.ctx, `
		INSERT INTO blog_series (owner_id, title, slug)
		VALUES ($1, 'Seri Orang Lain', 'seri-orang-lain')
		RETURNING id`, f.outsiderID,
	).Scan(&seriesID)
	if err != nil {
		t.Fatalf("insert blog series: %v", err)
	}

	input := f.draftInput("Judul")
	input.Body = map[string]any{"type": "doc", "content": []any{}}
	input.Meta = map[string]any{"seriesId": seriesID.String()}

	_, err = f.service.SaveDraft(f.ctx, f.actor(f.ownerID), created.ID, input)

	appErr := requireAppError(t, err, "VALIDATION_FAILED")
	requireFields(t, appErr, "meta.seriesId")
}

func TestSaveDraft_ContributorReturnsForbiddenLevel(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.addAuthor(created.ID, f.contributorID, LevelContributor)

	_, err := f.service.SaveDraft(f.ctx, f.actor(f.contributorID), created.ID, f.draftInput("Judul"))

	requireAppError(t, err, "FORBIDDEN_LEVEL")
}

func TestSaveDraft_InReviewReturnsInvalidTransition(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.setStatus(created.ID, StatusInReview)

	_, err := f.service.SaveDraft(f.ctx, f.actor(f.ownerID), created.ID, f.draftInput("Judul"))

	appErr := requireAppError(t, err, "INVALID_TRANSITION")
	if !strings.Contains(appErr.Detail, "ditinjau") {
		t.Errorf("detail = %q, mau menyebut sedang ditinjau", appErr.Detail)
	}
}

func TestSaveDraft_PublishedWithoutRevisionReturnsInvalidTransition(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.publish(created.ID, nil)

	_, err := f.service.SaveDraft(f.ctx, f.actor(f.ownerID), created.ID, f.draftInput("Judul"))

	appErr := requireAppError(t, err, "INVALID_TRANSITION")
	if !strings.Contains(appErr.Detail, "Mulai revisi") {
		t.Errorf("detail = %q, mau menyuruh mulai revisi", appErr.Detail)
	}
}

func TestSaveDraft_PublishedWithEditingRevisionDoesNotChangeContentTitle(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	revision := RevisionEditing
	f.publish(created.ID, &revision)

	input := f.draftInput("Judul Revisi")
	input.Title = "Judul Revisi"

	result, err := f.service.SaveDraft(f.ctx, f.actor(f.ownerID), created.ID, input)
	if err != nil {
		t.Fatalf("SaveDraft revisi: %v", err)
	}

	if result.Draft.Title != "Judul Revisi" || result.Draft.LockVersion != 2 {
		t.Errorf("draft = %q lock %d, mau Judul Revisi lock 2", result.Draft.Title, result.Draft.LockVersion)
	}
	content := f.contentRow(created.ID)
	if content.Title != "Tanpa judul" {
		t.Errorf("contents.title = %q, mau tetap Tanpa judul", content.Title)
	}
	if content.Status != StatusPublished || content.RevisionStatus == nil || *content.RevisionStatus != RevisionEditing {
		t.Errorf("status/revision = %s/%v, mau published/editing", content.Status, content.RevisionStatus)
	}

	// Saving the revision again uses the incremented lock version.
	input.LockVersion = 2
	if _, err := f.service.SaveDraft(f.ctx, f.actor(f.ownerID), created.ID, input); err != nil {
		t.Fatalf("SaveDraft revisi kedua: %v", err)
	}

	// A stale save against the previous lock version still conflicts.
	_, err = f.service.SaveDraft(f.ctx, f.actor(f.ownerID), created.ID, input)
	requireAppError(t, err, "LOCK_CONFLICT")
}

func TestSaveDraft_NonAuthorReturnsNotFound(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)

	_, err := f.service.SaveDraft(f.ctx, f.actor(f.outsiderID), created.ID, f.draftInput("Judul"))

	requireAppError(t, err, "NOT_FOUND")
}

func strPtr(value string) *string {
	return &value
}
