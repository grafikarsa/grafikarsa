package content

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestListStudio_OrdersByUpdatedAtAndPaginatesWithCursor(t *testing.T) {
	f := newFixture(t)

	first := f.create(TypePortfolio)
	second := f.create(TypeBlog)
	third := f.create(TypePortfolio)

	page, err := f.service.ListStudio(f.ctx, f.actor(f.ownerID), ListFilter{Limit: 2})
	if err != nil {
		t.Fatalf("ListStudio page 1: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("page 1 items = %d, mau 2", len(page.Items))
	}
	if page.NextCursor == "" {
		t.Fatal("nextCursor kosong, mau ada halaman berikutnya")
	}
	if page.Items[0].ID != third.ID || page.Items[1].ID != second.ID {
		t.Errorf("page 1 ids = %s, %s, mau %s, %s", page.Items[0].ID, page.Items[1].ID, third.ID, second.ID)
	}
	if page.Items[0].Title != "Tanpa judul" {
		t.Errorf("title = %q, mau Tanpa judul", page.Items[0].Title)
	}
	if page.Items[0].MyLevel != LevelOwner || page.Items[0].AuthorsCount != 1 {
		t.Errorf("level/authors = %s/%d", page.Items[0].MyLevel, page.Items[0].AuthorsCount)
	}

	next, err := f.service.ListStudio(f.ctx, f.actor(f.ownerID), ListFilter{Limit: 2, Cursor: page.NextCursor})
	if err != nil {
		t.Fatalf("ListStudio page 2: %v", err)
	}
	if len(next.Items) != 1 || next.Items[0].ID != first.ID {
		t.Fatalf("page 2 items = %+v, mau hanya %s", next.Items, first.ID)
	}
	if next.NextCursor != "" {
		t.Errorf("nextCursor = %q, mau kosong di halaman terakhir", next.NextCursor)
	}
}

func TestListStudio_FiltersByStatusAndType(t *testing.T) {
	f := newFixture(t)

	portfolio := f.create(TypePortfolio)
	blog := f.create(TypeBlog)
	f.setStatus(blog.ID, StatusChangesRequested)
	f.addAuthor(portfolio.ID, f.editorID, LevelEditor)

	page, err := f.service.ListStudio(f.ctx, f.actor(f.ownerID), ListFilter{Status: StatusChangesRequested})
	if err != nil {
		t.Fatalf("ListStudio status: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != blog.ID {
		t.Fatalf("items = %+v, mau satu blog changes_requested", page.Items)
	}

	page, err = f.service.ListStudio(f.ctx, f.actor(f.editorID), ListFilter{Type: TypePortfolio})
	if err != nil {
		t.Fatalf("ListStudio type: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != portfolio.ID || page.Items[0].MyLevel != LevelEditor {
		t.Fatalf("items = %+v, mau portfolio sebagai editor", page.Items)
	}
	if page.Items[0].AuthorsCount != 2 {
		t.Errorf("authorsCount = %d, mau 2", page.Items[0].AuthorsCount)
	}
}

func TestSummary_CountsEveryStatusAndPendingInvitations(t *testing.T) {
	f := newFixture(t)

	f.create(TypePortfolio)
	second := f.create(TypeBlog)
	f.setStatus(second.ID, StatusChangesRequested)

	outsiderContent, err := f.service.Create(f.ctx, f.actor(f.outsiderID), CreateInput{Type: TypePortfolio})
	if err != nil {
		t.Fatalf("Create outsider: %v", err)
	}
	f.addPendingInvite(outsiderContent.ID, f.ownerID, time.Now().UTC().Add(14*24*time.Hour))

	summary, err := f.service.Summary(f.ctx, f.actor(f.ownerID))
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if summary.Counts[StatusDraft] != 1 {
		t.Errorf("draft = %d, mau 1", summary.Counts[StatusDraft])
	}
	if summary.Counts[StatusChangesRequested] != 1 {
		t.Errorf("changes_requested = %d, mau 1", summary.Counts[StatusChangesRequested])
	}
	for _, status := range statuses {
		if _, ok := summary.Counts[status]; !ok {
			t.Errorf("counts tidak punya status %s", status)
		}
	}
	if summary.PendingInvitations != 1 {
		t.Errorf("pendingInvitations = %d, mau 1", summary.PendingInvitations)
	}
}

func TestGetStudio_NonAuthorReturnsNotFound(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)

	_, err := f.service.GetStudio(f.ctx, f.actor(f.outsiderID), created.ID)

	requireAppError(t, err, "NOT_FOUND")
}

func TestGetStudio_ReturnsLatestModerationFeedbackAndAppealState(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.publish(created.ID, nil)

	content := f.contentRow(created.ID)
	var reasonID uuid.UUID
	if err := f.pool.QueryRow(f.ctx,
		`SELECT id FROM moderation_reasons WHERE slug = 'plagiarisme'`,
	).Scan(&reasonID); err != nil {
		t.Fatalf("load reason: %v", err)
	}

	var caseID uuid.UUID
	err := f.pool.QueryRow(f.ctx, `
		INSERT INTO moderation_cases (
			content_id, version_id, kind, status, decision, reason_id,
			decision_note, decided_by, decided_at
		) VALUES ($1, $2, 'initial', 'decided', 'rejected', $3, 'Karya orang lain.', $4, now())
		RETURNING id`,
		created.ID, *content.LiveVersionID, reasonID, f.outsiderID,
	).Scan(&caseID)
	if err != nil {
		t.Fatalf("insert moderation case: %v", err)
	}
	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO moderation_annotations (case_id, target_ref, note, created_by)
		VALUES ($1, 'field:title', 'Ganti judul.', $2)`,
		caseID, f.outsiderID,
	); err != nil {
		t.Fatalf("insert annotation: %v", err)
	}

	result, err := f.service.GetStudio(f.ctx, f.actor(f.ownerID), created.ID)
	if err != nil {
		t.Fatalf("GetStudio: %v", err)
	}
	if result.LastModeration == nil {
		t.Fatal("lastModeration nil, mau keputusan terakhir")
	}
	if result.LastModeration.Decision != "rejected" {
		t.Errorf("decision = %s, mau rejected", result.LastModeration.Decision)
	}
	if result.LastModeration.ReasonLabel == nil || *result.LastModeration.ReasonLabel == "" {
		t.Errorf("reasonLabel = %v, mau terisi", result.LastModeration.ReasonLabel)
	}
	if result.LastModeration.Note == nil || *result.LastModeration.Note != "Karya orang lain." {
		t.Errorf("note = %v", result.LastModeration.Note)
	}
	if len(result.LastModeration.Annotations) != 1 || result.LastModeration.Annotations[0].TargetRef != "field:title" {
		t.Errorf("annotations = %+v", result.LastModeration.Annotations)
	}
	if !result.LastModeration.CanAppeal || result.LastModeration.AppealStatus != nil {
		t.Errorf("canAppeal/appealStatus = %v/%v, mau true/nil", result.LastModeration.CanAppeal, result.LastModeration.AppealStatus)
	}

	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO appeals (case_id, content_id, submitted_by, message)
		VALUES ($1, $2, $3, 'Mohon ditinjau kembali, ini karya sendiri.')`,
		caseID, created.ID, f.ownerID,
	); err != nil {
		t.Fatalf("insert appeal: %v", err)
	}

	result, err = f.service.GetStudio(f.ctx, f.actor(f.ownerID), created.ID)
	if err != nil {
		t.Fatalf("GetStudio setelah banding: %v", err)
	}
	if result.LastModeration.CanAppeal {
		t.Error("canAppeal = true, mau false setelah banding")
	}
	if result.LastModeration.AppealStatus == nil || *result.LastModeration.AppealStatus != "open" {
		t.Errorf("appealStatus = %v, mau open", result.LastModeration.AppealStatus)
	}
}

func TestGetStudio_ContributorCanReadButMyLevelIsContributor(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.addAuthor(created.ID, f.contributorID, LevelContributor)

	result, err := f.service.GetStudio(f.ctx, f.actor(f.contributorID), created.ID)
	if err != nil {
		t.Fatalf("GetStudio contributor: %v", err)
	}
	if result.MyLevel != LevelContributor {
		t.Errorf("myLevel = %s, mau contributor", result.MyLevel)
	}
	if result.Draft.LockVersion != 1 {
		t.Errorf("lockVersion = %d, mau 1", result.Draft.LockVersion)
	}
}
