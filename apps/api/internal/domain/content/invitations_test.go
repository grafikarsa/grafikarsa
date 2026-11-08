package content

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestInvitation_AcceptMakesAuthorVisible(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)

	if _, err := f.service.InviteAuthor(f.ctx, f.actor(f.ownerID), created.ID, AuthorInviteInput{
		UserID: f.editorID,
		Level:  LevelEditor,
	}); err != nil {
		t.Fatalf("invite: %v", err)
	}

	page, err := f.service.ListInvitations(f.ctx, f.actor(f.editorID), "")
	if err != nil {
		t.Fatalf("ListInvitations: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Content.ID != created.ID {
		t.Fatalf("invitations = %+v, mau satu untuk %s", page.Items, created.ID)
	}
	if page.Items[0].Level != LevelEditor {
		t.Errorf("level = %s, mau editor", page.Items[0].Level)
	}

	if err := f.service.AcceptInvitation(f.ctx, f.actor(f.editorID), created.ID); err != nil {
		t.Fatalf("AcceptInvitation: %v", err)
	}

	row := f.authorRow(created.ID, f.editorID)
	if row.InviteStatus != "accepted" || row.RespondedAt == nil {
		t.Fatalf("row = %+v, mau accepted dengan respondedAt", row)
	}

	authors, err := f.service.ListAuthors(f.ctx, f.actor(f.ownerID), created.ID)
	if err != nil {
		t.Fatalf("ListAuthors: %v", err)
	}
	found := false
	for _, author := range authors {
		if author.User.UserID == f.editorID && author.InviteStatus == "accepted" {
			found = true
		}
	}
	if !found {
		t.Fatalf("author accepted tidak muncul: %+v", authors)
	}

	studio, err := f.service.GetStudio(f.ctx, f.actor(f.editorID), created.ID)
	if err != nil {
		t.Fatalf("GetStudio invitee: %v", err)
	}
	if studio.MyLevel != LevelEditor {
		t.Errorf("myLevel = %s, mau editor", studio.MyLevel)
	}

	page, err = f.service.ListInvitations(f.ctx, f.actor(f.editorID), "")
	if err != nil {
		t.Fatalf("ListInvitations setelah accept: %v", err)
	}
	if len(page.Items) != 0 {
		t.Errorf("invitations setelah accept = %+v, mau kosong", page.Items)
	}
}

func TestInvitation_AcceptExpiredReturnsInvalidTransition(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.addPendingInvite(created.ID, f.editorID, time.Now().UTC().Add(-time.Minute))

	err := f.service.AcceptInvitation(f.ctx, f.actor(f.editorID), created.ID)
	requireAppError(t, err, "INVALID_TRANSITION")

	if row := f.authorRow(created.ID, f.editorID); row.InviteStatus != "pending" {
		t.Errorf("status = %s, mau tetap pending", row.InviteStatus)
	}
}

func TestInvitation_AcceptDeletedContentReturnsInvalidTransition(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	if _, err := f.service.InviteAuthor(f.ctx, f.actor(f.ownerID), created.ID, AuthorInviteInput{
		UserID: f.editorID,
		Level:  LevelEditor,
	}); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if _, err := f.pool.Exec(f.ctx, `
		UPDATE contents
		SET status = 'deleted', deleted_at = now(),
		    status_before_delete = 'draft', purge_after = now() + interval '30 days'
		WHERE id = $1`, created.ID); err != nil {
		t.Fatalf("mark deleted: %v", err)
	}
	err := f.service.AcceptInvitation(f.ctx, f.actor(f.editorID), created.ID)
	requireAppError(t, err, "INVALID_TRANSITION")
}

func TestInvitation_Decline(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.addPendingInvite(created.ID, f.editorID, time.Now().UTC().Add(time.Hour))

	if err := f.service.DeclineInvitation(f.ctx, f.actor(f.editorID), created.ID); err != nil {
		t.Fatalf("DeclineInvitation: %v", err)
	}

	row := f.authorRow(created.ID, f.editorID)
	if row.InviteStatus != "declined" || row.RespondedAt == nil {
		t.Errorf("row = %+v, mau declined dengan respondedAt", row)
	}
}

func TestInvitation_DeclineExpiredReturnsInvalidTransition(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.addPendingInvite(created.ID, f.editorID, time.Now().UTC().Add(-time.Hour))

	err := f.service.DeclineInvitation(f.ctx, f.actor(f.editorID), created.ID)
	requireAppError(t, err, "INVALID_TRANSITION")
}

func TestInvitation_NonInviteeReturnsNotFound(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.addPendingInvite(created.ID, f.editorID, time.Now().UTC().Add(time.Hour))

	err := f.service.AcceptInvitation(f.ctx, f.actor(f.outsiderID), created.ID)
	requireAppError(t, err, "NOT_FOUND")
}

func TestListInvitations_ExcludesExpiredPendingRows(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.addPendingInvite(created.ID, f.editorID, time.Now().UTC().Add(-time.Minute))

	page, err := f.service.ListInvitations(f.ctx, f.actor(f.editorID), "")
	if err != nil {
		t.Fatalf("ListInvitations: %v", err)
	}
	if len(page.Items) != 0 {
		t.Errorf("invitations = %+v, mau kosong", page.Items)
	}
}

func TestListInvitations_OrdersNewestFirst(t *testing.T) {
	f := newFixture(t)
	first := f.create(TypePortfolio)
	second := f.create(TypePortfolio)

	for _, contentID := range []uuid.UUID{first.ID, second.ID} {
		if _, err := f.service.InviteAuthor(f.ctx, f.actor(f.ownerID), contentID, AuthorInviteInput{
			UserID: f.editorID,
			Level:  LevelEditor,
		}); err != nil {
			t.Fatalf("invite: %v", err)
		}
	}
	if _, err := f.pool.Exec(f.ctx, `
		UPDATE content_authors SET invited_at = '2026-01-01T00:00:00Z' WHERE content_id = $1`, first.ID); err != nil {
		t.Fatalf("set invited_at: %v", err)
	}
	if _, err := f.pool.Exec(f.ctx, `
		UPDATE content_authors SET invited_at = '2026-01-02T00:00:00Z' WHERE content_id = $1`, second.ID); err != nil {
		t.Fatalf("set invited_at: %v", err)
	}

	page, err := f.service.ListInvitations(f.ctx, f.actor(f.editorID), "")
	if err != nil {
		t.Fatalf("ListInvitations: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("items = %d, mau 2", len(page.Items))
	}
	if page.Items[0].Content.ID != second.ID {
		t.Errorf("item pertama = %s, mau terbaru %s", page.Items[0].Content.ID, second.ID)
	}
}

func TestInvitationCursor_RoundTrips(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 678901234, time.UTC)
	contentID := uuid.MustParse("018f0000-0000-7000-8000-000000000001")

	cursor := encodeInvitationCursor(now, contentID)
	gotTime, gotID, err := decodeInvitationCursor(cursor)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !gotTime.Equal(now) || gotID != contentID {
		t.Errorf("decode = %s/%s, mau %s/%s", gotTime, gotID, now, contentID)
	}

	if _, _, err := decodeInvitationCursor("bukan-cursor"); err == nil {
		t.Error("decode cursor invalid = nil, mau error")
	}
}
