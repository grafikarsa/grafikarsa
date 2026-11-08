package content

import (
	"testing"

	"github.com/google/uuid"
)

func TestListVersions_ReturnsSubmitterAndDecision(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.saveCompletePortfolioDraft(created.ID, f.ownerID)
	if _, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID); err != nil {
		t.Fatalf("Submit: %v", err)
	}

	version := f.versionByNumber(created.ID, 1)

	var reasonID uuid.UUID
	if err := f.pool.QueryRow(f.ctx,
		`SELECT id FROM moderation_reasons WHERE slug = 'plagiarisme'`,
	).Scan(&reasonID); err != nil {
		t.Fatalf("load reason: %v", err)
	}
	if _, err := f.pool.Exec(f.ctx, `
		UPDATE moderation_cases
		SET status = 'decided', decision = 'rejected', reason_id = $2,
		    decision_note = 'Karya ini milik orang lain.', decided_by = $3, decided_at = now()
		WHERE content_id = $1`,
		created.ID, reasonID, f.outsiderID,
	); err != nil {
		t.Fatalf("decide case: %v", err)
	}
	if _, err := f.pool.Exec(f.ctx, `
		UPDATE content_versions SET state = 'rejected', decided_at = now() WHERE id = $1`,
		version.ID,
	); err != nil {
		t.Fatalf("reject version: %v", err)
	}
	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO moderation_annotations (case_id, target_ref, note, created_by)
		SELECT id, 'field:title', 'Ganti judul.', $2 FROM moderation_cases WHERE content_id = $1`,
		created.ID, f.outsiderID,
	); err != nil {
		t.Fatalf("insert annotation: %v", err)
	}

	versions, err := f.service.ListVersions(f.ctx, f.actor(f.ownerID), created.ID)
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(versions) != 1 {
		t.Fatalf("jumlah versi = %d, mau 1", len(versions))
	}
	got := versions[0]
	if got.ID != version.ID || got.Number != 1 || got.Kind != versionKindInitial {
		t.Errorf("versi = %+v", got)
	}
	if got.State != "rejected" {
		t.Errorf("state = %s, mau rejected", got.State)
	}
	if got.SubmittedBy.UserID != f.ownerID || got.SubmittedBy.DisplayName != "Pemilik Karya" {
		t.Errorf("submittedBy = %+v", got.SubmittedBy)
	}
	if got.DecidedAt == nil {
		t.Error("decidedAt versi nil")
	}
	if got.Decision == nil {
		t.Fatal("decision nil, mau keputusan penolakan")
	}
	if got.Decision.Decision != "rejected" {
		t.Errorf("decision = %s, mau rejected", got.Decision.Decision)
	}
	if got.Decision.ReasonLabel == nil || *got.Decision.ReasonLabel == "" {
		t.Errorf("reasonLabel = %v", got.Decision.ReasonLabel)
	}
	if got.Decision.Note == nil || *got.Decision.Note != "Karya ini milik orang lain." {
		t.Errorf("note = %v", got.Decision.Note)
	}
	if len(got.Decision.Annotations) != 1 || got.Decision.Annotations[0].TargetRef != "field:title" {
		t.Errorf("annotations = %+v", got.Decision.Annotations)
	}
	if !got.Decision.CanAppeal {
		t.Error("canAppeal = false, mau true untuk pemilik tanpa banding")
	}

	f.addAuthor(created.ID, f.editorID, LevelEditor)
	editorVersions, err := f.service.ListVersions(f.ctx, f.actor(f.editorID), created.ID)
	if err != nil {
		t.Fatalf("ListVersions editor: %v", err)
	}
	if editorVersions[0].Decision == nil || editorVersions[0].Decision.CanAppeal {
		t.Error("canAppeal editor = true, mau false")
	}

	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO appeals (case_id, content_id, submitted_by, message)
		SELECT id, $1, $2, 'Mohon ditinjau kembali, ini karya sendiri.'
		FROM moderation_cases WHERE content_id = $1`,
		created.ID, f.ownerID,
	); err != nil {
		t.Fatalf("insert appeal: %v", err)
	}
	versions, err = f.service.ListVersions(f.ctx, f.actor(f.ownerID), created.ID)
	if err != nil {
		t.Fatalf("ListVersions setelah banding: %v", err)
	}
	if versions[0].Decision == nil {
		t.Fatal("decision nil setelah banding")
	}
	if versions[0].Decision.CanAppeal {
		t.Error("canAppeal = true, mau false setelah banding")
	}
	if versions[0].Decision.AppealStatus == nil || *versions[0].Decision.AppealStatus != "open" {
		t.Errorf("appealStatus = %v, mau open", versions[0].Decision.AppealStatus)
	}
}

func TestGetVersion_ExpandsSnapshot(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	media := f.saveCompletePortfolioDraft(created.ID, f.ownerID, "poster")
	if _, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	version := f.versionByNumber(created.ID, 1)

	detail, err := f.service.GetVersion(f.ctx, f.actor(f.ownerID), created.ID, version.ID)
	if err != nil {
		t.Fatalf("GetVersion: %v", err)
	}
	if detail.Version.ID != version.ID || detail.Version.Number != 1 {
		t.Errorf("version = %+v", detail.Version)
	}
	if detail.Title != "Poster Ujian Akhir" {
		t.Errorf("title = %q", detail.Title)
	}
	if detail.Excerpt == nil || *detail.Excerpt == "" {
		t.Errorf("excerpt = %v, mau terisi", detail.Excerpt)
	}
	if detail.Category == nil || detail.Category.Slug != "web-app" || detail.Category.Name == "" {
		t.Errorf("category = %+v, mau web-app", detail.Category)
	}
	if detail.Cover == nil || detail.Cover.MediaID != media.CoverID {
		t.Errorf("cover = %+v, mau %s", detail.Cover, media.CoverID)
	}
	blocks, ok := detail.Body["blocks"].([]any)
	if !ok || len(blocks) != 1 {
		t.Errorf("body blocks = %#v, mau satu blok", detail.Body["blocks"])
	}
	if len(detail.Tags) != 1 || detail.Tags[0].Slug != "poster" {
		t.Errorf("tags = %+v, mau poster", detail.Tags)
	}
	if len(detail.Tools) != 1 || detail.Tools[0].Slug != "figma" {
		t.Errorf("tools = %+v, mau figma", detail.Tools)
	}
	if len(detail.Skills) != 1 || detail.Skills[0].Slug != "ui-design" {
		t.Errorf("skills = %+v, mau ui-design", detail.Skills)
	}
}

func TestVersions_OutsiderReturnsNotFound(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.saveCompletePortfolioDraft(created.ID, f.ownerID)
	if _, err := f.service.Submit(f.ctx, f.actor(f.ownerID), created.ID); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	version := f.versionByNumber(created.ID, 1)

	_, err := f.service.ListVersions(f.ctx, f.actor(f.outsiderID), created.ID)
	requireAppError(t, err, "NOT_FOUND")
	_, err = f.service.GetVersion(f.ctx, f.actor(f.outsiderID), created.ID, version.ID)
	requireAppError(t, err, "NOT_FOUND")
}
