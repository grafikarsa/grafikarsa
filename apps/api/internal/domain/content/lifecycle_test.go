package content

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

func (f *fixture) insertMediaWithStatus(ownerID uuid.UUID, status string) uuid.UUID {
	f.t.Helper()

	var id uuid.UUID
	err := f.pool.QueryRow(f.ctx, `
		INSERT INTO media_assets (
			owner_id, kind, purpose, original_key, original_filename, mime_type,
			size_bytes, width, height, status, blurhash, dominant_color
		) VALUES ($1, 'image', 'content', $2, 'gambar.png', 'image/png', 1024, 1200, 675,
			$3, 'LEHV6nWB2yk8pyo0adR*.7kCMdnj', '#1e293b')
		RETURNING id`,
		ownerID, "test/"+uuid.NewString()+".png", status,
	).Scan(&id)
	if err != nil {
		f.t.Fatalf("insert media %s: %v", status, err)
	}
	return id
}

// enrollOwnerInMajor inserts an academic year, class, and enrollment for the
// user, returning the major id used as the moderation queue major.
func (f *fixture) enrollUserInMajor(userID uuid.UUID, majorCode string) uuid.UUID {
	f.t.Helper()

	var majorID uuid.UUID
	if err := f.pool.QueryRow(f.ctx,
		`SELECT id FROM majors WHERE code = $1`, majorCode,
	).Scan(&majorID); err != nil {
		f.t.Fatalf("load major %s: %v", majorCode, err)
	}

	var yearID uuid.UUID
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO academic_years (start_year, starts_on, ends_on, is_current)
		VALUES (2026, '2026-07-01', '2027-06-30', false)
		ON CONFLICT (start_year) DO UPDATE SET starts_on = EXCLUDED.starts_on
		RETURNING id`,
	).Scan(&yearID); err != nil {
		f.t.Fatalf("insert academic year: %v", err)
	}

	var classID uuid.UUID
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO classes (academic_year_id, major_id, grade, section, name, slug)
		VALUES ($1, $2, 10, 'A', 'X RPL A', 'x-rpl-a-2026')
		ON CONFLICT (academic_year_id, major_id, grade, section)
		DO UPDATE SET name = EXCLUDED.name
		RETURNING id`,
		yearID, majorID,
	).Scan(&classID); err != nil {
		f.t.Fatalf("insert class: %v", err)
	}

	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO enrollments (user_id, class_id, academic_year_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, academic_year_id) DO UPDATE SET class_id = EXCLUDED.class_id`,
		userID, classID, yearID,
	); err != nil {
		f.t.Fatalf("insert enrollment: %v", err)
	}
	return majorID
}

func (f *fixture) draftLock(contentID, authorID uuid.UUID) int {
	f.t.Helper()

	studio, err := f.service.GetStudio(f.ctx, f.actor(authorID), contentID)
	if err != nil {
		f.t.Fatalf("GetStudio: %v", err)
	}
	return studio.Draft.LockVersion
}

type portfolioMedia struct {
	BodyMediaID uuid.UUID
	CoverID     uuid.UUID
}

func (f *fixture) saveCompletePortfolioDraft(contentID, authorID uuid.UUID, tags ...string) portfolioMedia {
	f.t.Helper()

	bodyMedia := f.insertMediaWithStatus(authorID, "ready")
	cover := f.insertMediaWithStatus(authorID, "ready")
	excerpt := "Karya poster untuk ujian akhir semester."
	input := DraftInput{
		LockVersion:  f.draftLock(contentID, authorID),
		Title:        "Poster Ujian Akhir",
		Excerpt:      &excerpt,
		CategoryID:   &f.portfolioCategoryID,
		CoverMediaID: &cover,
		Body:         portfolioBody(bodyMedia),
		Meta:         map[string]any{"context": "competition"},
		Tags:         tags,
		Tools:        []string{"figma"},
		Skills:       []string{"ui-design"},
	}
	if _, err := f.service.SaveDraft(f.ctx, f.actor(authorID), contentID, input); err != nil {
		f.t.Fatalf("SaveDraft portfolio: %v", err)
	}
	return portfolioMedia{BodyMediaID: bodyMedia, CoverID: cover}
}

func (f *fixture) saveCompleteBlogDraft(contentID, authorID uuid.UUID) {
	f.t.Helper()

	excerpt := "Ringkasan proyek uji coba yang cukup panjang untuk memenuhi batas minimal blog."
	input := DraftInput{
		LockVersion: f.draftLock(contentID, authorID),
		Title:       "Belajar Menulis Blog",
		Excerpt:     &excerpt,
		CategoryID:  &f.blogCategoryID,
		Body:        blogBody(160),
		Meta:        map[string]any{},
	}
	if _, err := f.service.SaveDraft(f.ctx, f.actor(authorID), contentID, input); err != nil {
		f.t.Fatalf("SaveDraft blog: %v", err)
	}
}

func portfolioBody(mediaID uuid.UUID) map[string]any {
	return map[string]any{"blocks": []any{
		map[string]any{
			"id":   uuid.NewString(),
			"type": "image",
			"data": map[string]any{"media_id": mediaID.String(), "fit": "contain"},
		},
	}}
}

func blogBody(words int) map[string]any {
	text := strings.TrimSpace(strings.Repeat("kata ", words))
	return map[string]any{
		"type": "doc",
		"content": []any{
			map[string]any{
				"type":  "paragraph",
				"attrs": map[string]any{"id": uuid.NewString()},
				"content": []any{
					map[string]any{"type": "text", "text": text},
				},
			},
		},
	}
}

func (f *fixture) setInReviewLimit(limit int) {
	f.t.Helper()

	raw := []byte(fmt.Sprintf("%d", limit))
	if _, err := f.store.UpsertSetting(f.ctx, store.UpsertSettingParams{
		Key:   maxInReviewSetting,
		Value: raw,
	}); err != nil {
		f.t.Fatalf("set in-review limit: %v", err)
	}
}

func (f *fixture) versionByNumber(contentID uuid.UUID, number int) store.ContentVersion {
	f.t.Helper()

	var id uuid.UUID
	if err := f.pool.QueryRow(f.ctx, `
		SELECT id FROM content_versions WHERE content_id = $1 AND number = $2`,
		contentID, number,
	).Scan(&id); err != nil {
		f.t.Fatalf("load version %d: %v", number, err)
	}
	version, err := f.store.GetContentVersionByID(f.ctx, store.GetContentVersionByIDParams{
		ID:        id,
		ContentID: contentID,
	})
	if err != nil {
		f.t.Fatalf("get version %d: %v", number, err)
	}
	return version
}

func (f *fixture) approveVersion(contentID uuid.UUID, number int) store.ContentVersion {
	f.t.Helper()

	version := f.versionByNumber(contentID, number)
	if _, err := f.pool.Exec(f.ctx, `
		UPDATE content_versions SET state = 'approved', decided_at = now() WHERE id = $1`,
		version.ID,
	); err != nil {
		f.t.Fatalf("approve version: %v", err)
	}
	if _, err := f.pool.Exec(f.ctx, `
		UPDATE contents
		SET status = 'published', live_version_id = $2, revision_status = NULL,
		    first_published_at = COALESCE(first_published_at, now())
		WHERE id = $1`,
		contentID, version.ID,
	); err != nil {
		f.t.Fatalf("publish content: %v", err)
	}
	version.State = "approved"
	return version
}

func (f *fixture) moderationCase(contentID uuid.UUID) store.ModerationCase {
	f.t.Helper()

	row, err := f.store.GetModerationCaseByVersion(f.ctx, f.versionByNumber(contentID, f.latestVersionNumber(contentID)).ID)
	if err != nil {
		f.t.Fatalf("load moderation case: %v", err)
	}
	return row
}

func (f *fixture) latestVersionNumber(contentID uuid.UUID) int {
	f.t.Helper()

	var number int
	if err := f.pool.QueryRow(f.ctx, `
		SELECT max(number) FROM content_versions WHERE content_id = $1`,
		contentID,
	).Scan(&number); err != nil {
		f.t.Fatalf("load latest version number: %v", err)
	}
	return number
}

func (f *fixture) setCaseClaimed(contentID uuid.UUID) {
	f.t.Helper()

	if _, err := f.pool.Exec(f.ctx, `
		UPDATE moderation_cases
		SET status = 'claimed', claimed_by = $2, claimed_at = now(),
		    claim_expires_at = now() + interval '30 minutes'
		WHERE content_id = $1`,
		contentID, f.editorID,
	); err != nil {
		f.t.Fatalf("claim moderation case: %v", err)
	}
}

func (f *fixture) countContentVersionMedia(versionID uuid.UUID) int {
	f.t.Helper()

	return countRows(f.t, f.pool,
		`SELECT count(*) FROM content_version_media WHERE version_id = $1`, versionID)
}

func TestSubmit_PortfolioCreatesPendingVersionAndOpenCase(t *testing.T) {
	f := newFixture(t)
	majorID := f.enrollUserInMajor(f.ownerID, "rpl")
	created := f.create(TypePortfolio)
	media := f.saveCompletePortfolioDraft(created.ID, f.ownerID, "poster")

	submitted, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID)
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if submitted.Status != StatusInReview {
		t.Fatalf("status = %s, mau in_review", submitted.Status)
	}
	if submitted.RevisionStatus != nil {
		t.Errorf("revisionStatus = %v, mau nil", submitted.RevisionStatus)
	}

	content := f.contentRow(created.ID)
	if content.Status != StatusInReview {
		t.Fatalf("status konten = %s, mau in_review", content.Status)
	}

	version := f.versionByNumber(created.ID, 1)
	if version.Kind != versionKindInitial || version.State != "pending" {
		t.Errorf("kind/state = %s/%s, mau initial/pending", version.Kind, version.State)
	}
	if version.Number != 1 {
		t.Errorf("number = %d, mau 1", version.Number)
	}
	if len(version.ContentHash) != 32 {
		t.Errorf("contentHash = %d byte, mau 32", len(version.ContentHash))
	}
	if version.Title != "Poster Ujian Akhir" {
		t.Errorf("title = %q, mau Poster Ujian Akhir", version.Title)
	}
	if got := f.countContentVersionMedia(version.ID); got != 2 {
		t.Errorf("version media = %d, mau 2 (body + cover)", got)
	}
	rows, err := f.pool.Query(f.ctx,
		`SELECT media_id FROM content_version_media WHERE version_id = $1`, version.ID)
	if err != nil {
		t.Fatalf("load version media: %v", err)
	}
	defer rows.Close()
	mediaIDs := map[uuid.UUID]struct{}{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan version media: %v", err)
		}
		mediaIDs[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate version media: %v", err)
	}
	for _, id := range []uuid.UUID{media.BodyMediaID, media.CoverID} {
		if _, ok := mediaIDs[id]; !ok {
			t.Errorf("media %s tidak ada di version media", id)
		}
	}

	moderationCase := f.moderationCase(created.ID)
	if moderationCase.Status != "open" || moderationCase.Kind != versionKindInitial {
		t.Errorf("case status/kind = %s/%s, mau open/initial", moderationCase.Status, moderationCase.Kind)
	}
	if moderationCase.QueueMajorID == nil || *moderationCase.QueueMajorID != majorID {
		t.Errorf("queueMajorId = %v, mau %s", moderationCase.QueueMajorID, majorID)
	}
	if moderationCase.VersionID != version.ID {
		t.Errorf("case version = %s, mau %s", moderationCase.VersionID, version.ID)
	}
}

func TestSubmit_BlogCreatesPendingVersion(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypeBlog)
	f.saveCompleteBlogDraft(created.ID, f.ownerID)

	submitted, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID)
	if err != nil {
		t.Fatalf("Submit blog: %v", err)
	}
	if submitted.Status != StatusInReview {
		t.Errorf("status = %s, mau in_review", submitted.Status)
	}
	version := f.versionByNumber(created.ID, 1)
	if version.Kind != versionKindInitial || version.State != "pending" {
		t.Errorf("kind/state = %s/%s, mau initial/pending", version.Kind, version.State)
	}
	if version.Excerpt == nil || *version.Excerpt == "" {
		t.Errorf("excerpt snapshot kosong")
	}
	if len(version.ContentHash) != 32 {
		t.Errorf("contentHash = %d byte, mau 32", len(version.ContentHash))
	}
	// Staff users have no enrollment, so the case goes to the global queue.
	if moderationCase := f.moderationCase(created.ID); moderationCase.QueueMajorID != nil {
		t.Errorf("queueMajorId = %v, mau nil untuk user tanpa enrollment", moderationCase.QueueMajorID)
	}
}

func TestSubmit_SecondSubmitReturnsInvalidTransition(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.saveCompletePortfolioDraft(created.ID, f.ownerID)

	if _, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID); err != nil {
		t.Fatalf("Submit pertama: %v", err)
	}
	_, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID)
	appErr := requireAppError(t, err, "INVALID_TRANSITION")
	if !strings.Contains(appErr.Detail, "ditinjau") {
		t.Errorf("detail = %q, mau menyebut sedang ditinjau", appErr.Detail)
	}
}

func TestSubmit_IncompleteDraftReturnsAllValidationErrors(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)

	input := DraftInput{
		LockVersion: f.draftLock(created.ID, f.ownerID),
		Title:       "",
		Body:        map[string]any{"blocks": []any{}},
		Meta:        map[string]any{},
	}
	if _, err := f.service.SaveDraft(f.ctx, f.actor(f.ownerID), created.ID, input); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}

	_, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID)
	appErr := requireAppError(t, err, "VALIDATION_FAILED")
	requireFields(t, appErr, "title", "categoryId", "coverMediaId", "body.blocks")
}

func TestSubmit_ProcessingMediaReturnsMediaProcessing(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)

	processing := f.insertMediaWithStatus(f.ownerID, "processing")
	excerpt := "Karya dengan media yang masih diproses."
	input := DraftInput{
		LockVersion:  f.draftLock(created.ID, f.ownerID),
		Title:        "Poster Sedang Proses",
		Excerpt:      &excerpt,
		CategoryID:   &f.portfolioCategoryID,
		CoverMediaID: &processing,
		Body:         portfolioBody(processing),
		Meta:         map[string]any{"context": "personal"},
	}
	if _, err := f.service.SaveDraft(f.ctx, f.actor(f.ownerID), created.ID, input); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}

	_, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID)
	appErr := requireAppError(t, err, "VALIDATION_FAILED")
	if !hasFieldError(appErr, "media") {
		t.Fatalf("error media tidak ada: %+v", appErr.Fields)
	}
	for _, field := range appErr.Fields {
		if field.Field == "media" {
			if field.Code != "media_processing" {
				t.Errorf("code = %s, mau media_processing", field.Code)
			}
			if field.Message != "Masih ada media yang belum selesai diproses." {
				t.Errorf("message = %q", field.Message)
			}
		}
	}
}

func TestSubmit_InReviewLimitReturnsLimitReached(t *testing.T) {
	f := newFixture(t)
	f.setInReviewLimit(1)

	first := f.create(TypePortfolio)
	f.saveCompletePortfolioDraft(first.ID, f.ownerID)
	if _, err := f.service.Submit(f.ctx, f.actor(f.ownerID), first.ID); err != nil {
		t.Fatalf("Submit pertama: %v", err)
	}

	second := f.create(TypePortfolio)
	f.saveCompletePortfolioDraft(second.ID, f.ownerID)
	_, err := f.service.Submit(f.ctx, f.actor(f.ownerID), second.ID)
	requireAppError(t, err, "LIMIT_REACHED")
}

func TestSubmit_NonOwnerReturnsForbiddenLevel(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.saveCompletePortfolioDraft(created.ID, f.ownerID)
	f.addAuthor(created.ID, f.editorID, LevelEditor)
	f.addAuthor(created.ID, f.contributorID, LevelContributor)

	for _, userID := range []uuid.UUID{f.editorID, f.contributorID} {
		_, err := f.service.Submit(f.ctx, f.actor(userID), created.ID)
		appErr := requireAppError(t, err, "FORBIDDEN_LEVEL")
		if appErr.Status != 403 {
			t.Errorf("status = %d, mau 403", appErr.Status)
		}
		if appErr.Title != "Hanya pemilik yang bisa melakukan ini." {
			t.Errorf("title = %q", appErr.Title)
		}
	}
}

func TestSubmit_OutsiderReturnsNotFound(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.saveCompletePortfolioDraft(created.ID, f.ownerID)

	_, err := f.service.Submit(f.ctx, f.actor(f.outsiderID), created.ID)
	requireAppError(t, err, "NOT_FOUND")
}

func TestWithdraw_ReturnsContentToDraftAndVersionWithdrawn(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.saveCompletePortfolioDraft(created.ID, f.ownerID)
	if _, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	withdrawn, err := f.service.Withdraw(f.ctx, f.actor(f.ownerID), created.ID)
	if err != nil {
		t.Fatalf("Withdraw: %v", err)
	}
	if withdrawn.Status != StatusDraft {
		t.Errorf("status = %s, mau draft", withdrawn.Status)
	}
	version := f.versionByNumber(created.ID, 1)
	if version.State != "withdrawn" {
		t.Errorf("state versi = %s, mau withdrawn", version.State)
	}
	if moderationCase := f.moderationCase(created.ID); moderationCase.Status != "withdrawn" {
		t.Errorf("status kasus = %s, mau withdrawn", moderationCase.Status)
	}

	// A draft that already has a withdrawn version resubmits as resubmission.
	if _, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID); err != nil {
		t.Fatalf("Submit ulang: %v", err)
	}
	if version := f.versionByNumber(created.ID, 2); version.Kind != versionKindResubmission {
		t.Errorf("kind ulang = %s, mau resubmission", version.Kind)
	}
}

func TestWithdraw_AfterModeratorClaimReturnsInvalidTransition(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.saveCompletePortfolioDraft(created.ID, f.ownerID)
	if _, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	f.setCaseClaimed(created.ID)

	_, err := f.service.Withdraw(f.ctx, f.actor(f.ownerID), created.ID)
	appErr := requireAppError(t, err, "INVALID_TRANSITION")
	if appErr.Status != 409 {
		t.Errorf("status = %d, mau 409", appErr.Status)
	}
	if !strings.Contains(appErr.Detail, "Moderator sudah mulai meninjau") {
		t.Errorf("detail = %q", appErr.Detail)
	}
}

func TestRevision_StartCopiesLiveSnapshotToDraft(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.saveCompletePortfolioDraft(created.ID, f.ownerID)
	if _, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	live := f.approveVersion(created.ID, 1)

	// Diverge the draft directly so we can see the copy overwrite it.
	if _, err := f.pool.Exec(f.ctx, `
		UPDATE content_drafts SET title = 'Judul Draf Berbeda', lock_version = lock_version + 1
		WHERE content_id = $1`,
		created.ID,
	); err != nil {
		t.Fatalf("diverge draft: %v", err)
	}
	before := f.draftLock(created.ID, f.ownerID)

	started, err := f.service.StartRevision(f.ctx, f.actor(f.ownerID), created.ID)
	if err != nil {
		t.Fatalf("StartRevision: %v", err)
	}
	if started.RevisionStatus == nil || *started.RevisionStatus != RevisionEditing {
		t.Fatalf("revisionStatus = %v, mau editing", started.RevisionStatus)
	}
	if started.Draft.Title != live.Title {
		t.Errorf("draft title = %q, mau %q", started.Draft.Title, live.Title)
	}
	if started.Draft.LockVersion <= before {
		t.Errorf("lockVersion = %d, mau > %d", started.Draft.LockVersion, before)
	}

	var baseVersionID *uuid.UUID
	if err := f.pool.QueryRow(f.ctx,
		`SELECT base_version_id FROM content_drafts WHERE content_id = $1`, created.ID,
	).Scan(&baseVersionID); err != nil {
		t.Fatalf("load base version: %v", err)
	}
	if baseVersionID == nil || *baseVersionID != live.ID {
		t.Errorf("baseVersionId = %v, mau %s", baseVersionID, live.ID)
	}
	if got := countRows(t, f.pool,
		`SELECT count(*) FROM content_draft_media WHERE content_id = $1`, created.ID,
	); got != 2 {
		t.Errorf("draft media = %d, mau 2 (dari versi tayang)", got)
	}
}

func TestRevision_SubmitWithdrawAndDiscard(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.saveCompletePortfolioDraft(created.ID, f.ownerID)
	if _, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	live := f.approveVersion(created.ID, 1)
	if _, err := f.service.StartRevision(f.ctx, f.actor(f.ownerID), created.ID); err != nil {
		t.Fatalf("StartRevision: %v", err)
	}

	submitted, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID)
	if err != nil {
		t.Fatalf("Submit revisi: %v", err)
	}
	if submitted.Status != StatusPublished {
		t.Errorf("status = %s, mau published", submitted.Status)
	}
	if submitted.RevisionStatus == nil || *submitted.RevisionStatus != RevisionInReview {
		t.Fatalf("revisionStatus = %v, mau in_review", submitted.RevisionStatus)
	}
	version := f.versionByNumber(created.ID, 2)
	if version.Kind != versionKindRevision || version.State != "pending" {
		t.Errorf("kind/state = %s/%s, mau revision/pending", version.Kind, version.State)
	}

	withdrawn, err := f.service.Withdraw(f.ctx, f.actor(f.ownerID), created.ID)
	if err != nil {
		t.Fatalf("Withdraw revisi: %v", err)
	}
	if withdrawn.Status != StatusPublished {
		t.Errorf("status setelah tarik = %s, mau published", withdrawn.Status)
	}
	if withdrawn.RevisionStatus == nil || *withdrawn.RevisionStatus != RevisionEditing {
		t.Fatalf("revisionStatus setelah tarik = %v, mau editing", withdrawn.RevisionStatus)
	}
	if version := f.versionByNumber(created.ID, 2); version.State != "withdrawn" {
		t.Errorf("state revisi = %s, mau withdrawn", version.State)
	}

	discarded, err := f.service.DiscardRevision(f.ctx, f.actor(f.ownerID), created.ID)
	if err != nil {
		t.Fatalf("DiscardRevision: %v", err)
	}
	if discarded.RevisionStatus != nil {
		t.Errorf("revisionStatus = %v, mau nil", discarded.RevisionStatus)
	}
	if discarded.Draft.Title != live.Title {
		t.Errorf("draft title = %q, mau %q", discarded.Draft.Title, live.Title)
	}
}

func TestArchive_WithEditingRevisionDiscardsRevisionAndArchives(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.saveCompletePortfolioDraft(created.ID, f.ownerID)
	if _, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	live := f.approveVersion(created.ID, 1)
	if _, err := f.service.StartRevision(f.ctx, f.actor(f.ownerID), created.ID); err != nil {
		t.Fatalf("StartRevision: %v", err)
	}

	excerpt := "Draf revisi yang akan dibuang saat arsip."
	input := DraftInput{
		LockVersion: f.draftLock(created.ID, f.ownerID),
		Title:       "Draf Revisi Arsip",
		Excerpt:     &excerpt,
		CategoryID:  &f.portfolioCategoryID,
		Body:        map[string]any{"blocks": []any{}},
		Meta:        map[string]any{"context": "personal"},
	}
	if _, err := f.service.SaveDraft(f.ctx, f.actor(f.ownerID), created.ID, input); err != nil {
		t.Fatalf("SaveDraft revisi: %v", err)
	}

	archived, err := f.service.Archive(f.ctx, f.actor(f.ownerID), created.ID)
	if err != nil {
		t.Fatalf("Archive: %v", err)
	}
	if archived.Status != StatusArchived {
		t.Fatalf("status = %s, mau archived", archived.Status)
	}
	if archived.RevisionStatus != nil {
		t.Errorf("revisionStatus = %v, mau nil", archived.RevisionStatus)
	}
	if archived.ArchivedAt == nil {
		t.Error("archivedAt nil, mau terisi")
	}
	if archived.Draft.Title != live.Title {
		t.Errorf("draft title = %q, mau %q (dikembalikan ke versi tayang)", archived.Draft.Title, live.Title)
	}

	unarchived, err := f.service.Unarchive(f.ctx, f.actor(f.ownerID), created.ID)
	if err != nil {
		t.Fatalf("Unarchive: %v", err)
	}
	if unarchived.Status != StatusPublished {
		t.Errorf("status = %s, mau published", unarchived.Status)
	}
	if unarchived.ArchivedAt != nil {
		t.Errorf("archivedAt = %v, mau nil", unarchived.ArchivedAt)
	}
}

func TestDeleteAndRestore_PublishedContent(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.saveCompletePortfolioDraft(created.ID, f.ownerID)
	if _, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	f.approveVersion(created.ID, 1)

	if err := f.service.Delete(f.ctx, f.actor(f.ownerID), created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	deleted := f.contentRow(created.ID)
	if deleted.Status != StatusDeleted {
		t.Fatalf("status = %s, mau deleted", deleted.Status)
	}
	if deleted.StatusBeforeDelete == nil || *deleted.StatusBeforeDelete != StatusPublished {
		t.Errorf("statusBeforeDelete = %v, mau published", deleted.StatusBeforeDelete)
	}
	if deleted.DeletedAt == nil || deleted.PurgeAfter == nil {
		t.Fatalf("deletedAt/purgeAfter = %v/%v, mau terisi", deleted.DeletedAt, deleted.PurgeAfter)
	}
	if !deleted.PurgeAfter.After(time.Now().UTC()) {
		t.Errorf("purgeAfter = %v, mau di masa depan", deleted.PurgeAfter)
	}

	if err := f.service.Delete(f.ctx, f.actor(f.ownerID), created.ID); err == nil {
		t.Fatal("Delete kedua berhasil, mau INVALID_TRANSITION")
	} else {
		requireAppError(t, err, "INVALID_TRANSITION")
	}

	restored, err := f.service.Restore(f.ctx, f.actor(f.ownerID), created.ID)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if restored.Status != StatusPublished {
		t.Errorf("status = %s, mau published", restored.Status)
	}
	if restored.DeletedAt != nil || restored.PurgeAfter != nil {
		t.Errorf("deletedAt/purgeAfter = %v/%v, mau nil", restored.DeletedAt, restored.PurgeAfter)
	}
	if restored.RevisionStatus != nil {
		t.Errorf("revisionStatus = %v, mau nil", restored.RevisionStatus)
	}
}

func TestDelete_InReviewReturnsInvalidTransition(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.saveCompletePortfolioDraft(created.ID, f.ownerID)
	if _, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	err := f.service.Delete(f.ctx, f.actor(f.ownerID), created.ID)
	appErr := requireAppError(t, err, "INVALID_TRANSITION")
	if !strings.Contains(appErr.Detail, "Tarik kembali dari moderasi dulu.") {
		t.Errorf("detail = %q", appErr.Detail)
	}
}

func TestPurge_DeletesOnlyExpiredTrashedContents(t *testing.T) {
	f := newFixture(t)
	expired := f.create(TypePortfolio)
	kept := f.create(TypePortfolio)

	markDeleted := func(contentID uuid.UUID, purgeAfter time.Time) {
		if _, err := f.pool.Exec(f.ctx, `
			UPDATE contents
			SET status = 'deleted', status_before_delete = 'draft', deleted_at = now(),
			    purge_after = $2
			WHERE id = $1`,
			contentID, purgeAfter,
		); err != nil {
			t.Fatalf("mark deleted: %v", err)
		}
	}
	markDeleted(expired.ID, time.Now().UTC().Add(-time.Hour))
	markDeleted(kept.ID, time.Now().UTC().Add(24*time.Hour))

	deleted, err := Purge(f.ctx, f.store)
	if err != nil {
		t.Fatalf("Purge: %v", err)
	}
	if deleted != 1 {
		t.Errorf("purged = %d, mau 1", deleted)
	}
	if countRows(t, f.pool, `SELECT count(*) FROM contents WHERE id = $1`, expired.ID) != 0 {
		t.Error("konten kedaluwarsa masih ada")
	}
	if countRows(t, f.pool, `SELECT count(*) FROM contents WHERE id = $1`, kept.ID) != 1 {
		t.Error("konten yang belum kedaluwarsa ikut terhapus")
	}
}

func TestSubmit_ConcurrentSubmitsOnlyOneSucceeds(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.saveCompletePortfolioDraft(created.ID, f.ownerID)

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID)
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	succeeded, invalid := 0, 0
	for err := range errs {
		if err == nil {
			succeeded++
			continue
		}
		appErr, ok := apperr.As(err)
		if ok && appErr.Code == "INVALID_TRANSITION" {
			invalid++
			continue
		}
		t.Fatalf("error tak terduga: %v", err)
	}
	if succeeded != 1 || invalid != 1 {
		t.Fatalf("sukses/gagal = %d/%d, mau 1/1", succeeded, invalid)
	}
	if got := countRows(t, f.pool,
		`SELECT count(*) FROM content_versions WHERE content_id = $1`, created.ID,
	); got != 1 {
		t.Errorf("jumlah versi = %d, mau 1", got)
	}
}
