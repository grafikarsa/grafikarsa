package content

import (
	"testing"
	"time"
)

func TestExpireInvitations_MarksOnlyExpiredPendingRows(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)

	f.addPendingInvite(created.ID, f.editorID, time.Now().UTC().Add(-time.Hour))
	f.addPendingInvite(created.ID, f.outsiderID, time.Now().UTC().Add(time.Hour))
	f.addAuthor(created.ID, f.contributorID, LevelContributor)

	expired, err := f.service.ExpireInvitations(f.ctx)
	if err != nil {
		t.Fatalf("ExpireInvitations: %v", err)
	}
	if expired != 1 {
		t.Errorf("expired = %d, mau 1", expired)
	}

	if row := f.authorRow(created.ID, f.editorID); row.InviteStatus != "expired" {
		t.Errorf("status undangan lewat = %s, mau expired", row.InviteStatus)
	}
	if row := f.authorRow(created.ID, f.outsiderID); row.InviteStatus != "pending" {
		t.Errorf("status undangan aktif = %s, mau pending", row.InviteStatus)
	}
	if row := f.authorRow(created.ID, f.contributorID); row.InviteStatus != "accepted" {
		t.Errorf("status author = %s, mau accepted", row.InviteStatus)
	}
}

func TestExpireInvitations_LeavesDeclinedRowsAlone(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.addPendingInvite(created.ID, f.editorID, time.Now().UTC().Add(-time.Hour))

	if _, err := f.pool.Exec(f.ctx, `
		UPDATE content_authors
		SET invite_status = 'declined', responded_at = now() - interval '2 hours'
		WHERE content_id = $1 AND user_id = $2`, created.ID, f.editorID); err != nil {
		t.Fatalf("mark declined: %v", err)
	}

	expired, err := f.service.ExpireInvitations(f.ctx)
	if err != nil {
		t.Fatalf("ExpireInvitations: %v", err)
	}
	if expired != 0 {
		t.Errorf("expired = %d, mau 0", expired)
	}
	if row := f.authorRow(created.ID, f.editorID); row.InviteStatus != "declined" {
		t.Errorf("status = %s, mau declined", row.InviteStatus)
	}
}
