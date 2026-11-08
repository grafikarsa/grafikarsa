package content

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

func (f *fixture) setAuthorLimit(contentType string, limit int) {
	f.t.Helper()

	rule := maxAuthorsFor(contentType)
	raw, err := json.Marshal(limit)
	if err != nil {
		f.t.Fatalf("marshal setting: %v", err)
	}
	if _, err := f.store.UpsertSetting(f.ctx, store.UpsertSettingParams{
		Key:   rule.setting,
		Value: raw,
	}); err != nil {
		f.t.Fatalf("set author limit: %v", err)
	}
}

func (f *fixture) authorRow(contentID, userID uuid.UUID) store.ContentAuthor {
	f.t.Helper()

	row, err := f.store.GetContentAuthorRow(f.ctx, store.GetContentAuthorRowParams{
		ContentID: contentID,
		UserID:    userID,
	})
	if err != nil {
		f.t.Fatalf("load author row: %v", err)
	}
	return row
}

func TestInviteAuthor_CreatesPendingInvitation(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)

	invited, err := f.service.InviteAuthor(f.ctx, f.actor(f.ownerID), created.ID, AuthorInviteInput{
		UserID:    f.editorID,
		Level:     LevelEditor,
		RoleLabel: strPtr("  Backend  "),
	})
	if err != nil {
		t.Fatalf("InviteAuthor: %v", err)
	}

	if invited.InviteStatus != "pending" || invited.Level != LevelEditor {
		t.Fatalf("invite = %+v, mau pending/editor", invited)
	}
	if invited.RoleLabel == nil || *invited.RoleLabel != "Backend" {
		t.Fatalf("roleLabel = %v, mau Backend tanpa spasi", invited.RoleLabel)
	}
	if invited.Position != 1 {
		t.Errorf("position = %d, mau 1", invited.Position)
	}
	if invited.User.UserID != f.editorID || invited.User.DisplayName != "Editor Karya" {
		t.Errorf("user = %+v, mau Editor Karya", invited.User)
	}
	if invited.InviteExpiresAt == nil {
		t.Fatal("inviteExpiresAt nil")
	}
	wantExpiry := time.Now().UTC().Add(14 * 24 * time.Hour)
	if diff := invited.InviteExpiresAt.Sub(wantExpiry); diff > time.Minute || diff < -time.Minute {
		t.Errorf("inviteExpiresAt = %s, mau sekitar %s", invited.InviteExpiresAt, wantExpiry)
	}

	row := f.authorRow(created.ID, f.editorID)
	if row.InvitedBy == nil || *row.InvitedBy != f.ownerID {
		t.Errorf("invitedBy = %v, mau %s", row.InvitedBy, f.ownerID)
	}
	if row.RespondedAt != nil {
		t.Errorf("respondedAt = %v, mau NULL", row.RespondedAt)
	}
}

func TestInviteAuthor_BlockedInactiveAndUnactivatedUsersReturnFieldError(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)

	blockedID := f.insertUser("terblokir", "Terblokir")
	if _, err := f.pool.Exec(f.ctx, `UPDATE users SET content_blocked_at = now() WHERE id = $1`, blockedID); err != nil {
		t.Fatalf("block user: %v", err)
	}

	inactiveID := f.insertUser("nonaktif", "Nonaktif")
	if _, err := f.pool.Exec(f.ctx, `UPDATE users SET status = 'inactive' WHERE id = $1`, inactiveID); err != nil {
		t.Fatalf("deactivate user: %v", err)
	}

	var unactivatedID uuid.UUID
	if err := f.pool.QueryRow(f.ctx, `
		INSERT INTO users (account_type, display_name, username, password_hash, password_kind)
		VALUES ('staff', 'Belum Aktif', 'belumaktif', 'x', 'user')
		RETURNING id`,
	).Scan(&unactivatedID); err != nil {
		t.Fatalf("insert unactivated user: %v", err)
	}

	for name, userID := range map[string]uuid.UUID{
		"blocked":     blockedID,
		"inactive":    inactiveID,
		"unactivated": unactivatedID,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := f.service.InviteAuthor(f.ctx, f.actor(f.ownerID), created.ID, AuthorInviteInput{
				UserID: userID,
				Level:  LevelEditor,
			})
			appErr := requireAppError(t, err, "VALIDATION_FAILED")
			if got := fieldErrorMessage(appErr, "userId"); got != "Pengguna ini tidak bisa diundang." {
				t.Errorf("pesan userId = %q", got)
			}
		})
	}
}

func TestInviteAuthor_ExistingAuthorReturnsConflict(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)

	if _, err := f.service.InviteAuthor(f.ctx, f.actor(f.ownerID), created.ID, AuthorInviteInput{
		UserID: f.editorID,
		Level:  LevelEditor,
	}); err != nil {
		t.Fatalf("invite pertama: %v", err)
	}

	_, err := f.service.InviteAuthor(f.ctx, f.actor(f.ownerID), created.ID, AuthorInviteInput{
		UserID: f.editorID,
		Level:  LevelContributor,
	})
	requireAppError(t, err, "CONFLICT")
}

func TestInviteAuthor_LimitReached(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.setAuthorLimit(TypePortfolio, 2)

	if _, err := f.service.InviteAuthor(f.ctx, f.actor(f.ownerID), created.ID, AuthorInviteInput{
		UserID: f.editorID,
		Level:  LevelEditor,
	}); err != nil {
		t.Fatalf("invite kedua: %v", err)
	}

	_, err := f.service.InviteAuthor(f.ctx, f.actor(f.ownerID), created.ID, AuthorInviteInput{
		UserID: f.outsiderID,
		Level:  LevelEditor,
	})
	appErr := requireAppError(t, err, "LIMIT_REACHED")
	if appErr.Detail != "Batas author tercapai (2)." {
		t.Errorf("detail = %q", appErr.Detail)
	}
}

func TestInviteAuthor_ReinviteAfterDeclineReusesRow(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)

	if _, err := f.service.InviteAuthor(f.ctx, f.actor(f.ownerID), created.ID, AuthorInviteInput{
		UserID: f.editorID,
		Level:  LevelEditor,
	}); err != nil {
		t.Fatalf("invite: %v", err)
	}
	if err := f.service.DeclineInvitation(f.ctx, f.actor(f.editorID), created.ID); err != nil {
		t.Fatalf("decline: %v", err)
	}
	before := f.authorRow(created.ID, f.editorID)

	time.Sleep(2 * time.Millisecond)
	reinvited, err := f.service.InviteAuthor(f.ctx, f.actor(f.ownerID), created.ID, AuthorInviteInput{
		UserID: f.editorID,
		Level:  LevelContributor,
	})
	if err != nil {
		t.Fatalf("reinvite: %v", err)
	}

	if got := countRows(t, f.pool, `SELECT count(*) FROM content_authors WHERE content_id = $1`, created.ID); got != 2 {
		t.Fatalf("jumlah author = %d, mau 2 (owner + editor)", got)
	}
	if reinvited.InviteStatus != "pending" || reinvited.Level != LevelContributor {
		t.Errorf("reinvite = %+v, mau pending/contributor", reinvited)
	}
	row := f.authorRow(created.ID, f.editorID)
	if row.RespondedAt != nil {
		t.Errorf("respondedAt = %v, mau NULL", row.RespondedAt)
	}
	if !row.InvitedAt.After(before.InvitedAt) {
		t.Errorf("invitedAt tidak diperbarui: %s <= %s", row.InvitedAt, before.InvitedAt)
	}
}

func TestInviteAuthor_EditorReturnsForbiddenLevel(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.addAuthor(created.ID, f.editorID, LevelEditor)

	_, err := f.service.InviteAuthor(f.ctx, f.actor(f.editorID), created.ID, AuthorInviteInput{
		UserID: f.outsiderID,
		Level:  LevelEditor,
	})
	appErr := requireAppError(t, err, "FORBIDDEN_LEVEL")
	if appErr.Status != 403 || appErr.Title != "Hanya pemilik yang bisa melakukan ini." {
		t.Errorf("error = %d %q", appErr.Status, appErr.Title)
	}
}

func TestInviteAuthor_NonAuthorReturnsNotFound(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)

	_, err := f.service.InviteAuthor(f.ctx, f.actor(f.outsiderID), created.ID, AuthorInviteInput{
		UserID: f.editorID,
		Level:  LevelEditor,
	})
	requireAppError(t, err, "NOT_FOUND")
}

func TestInviteAuthor_ConcurrentInvitesRespectLimit(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.setAuthorLimit(TypePortfolio, 2)

	userIDs := []uuid.UUID{f.editorID, f.contributorID}
	errs := make([]error, len(userIDs))

	var wg sync.WaitGroup
	for i, userID := range userIDs {
		wg.Add(1)
		go func(index int, id uuid.UUID) {
			defer wg.Done()
			_, errs[index] = f.service.InviteAuthor(context.Background(), f.actor(f.ownerID), created.ID, AuthorInviteInput{
				UserID: id,
				Level:  LevelEditor,
			})
		}(i, userID)
	}
	wg.Wait()

	succeeded, limited := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			succeeded++
		case apperr.Is(err, "LIMIT_REACHED"):
			limited++
		default:
			t.Errorf("error tak terduga: %v", err)
		}
	}
	if succeeded != 1 || limited != 1 {
		t.Fatalf("sukses/gagal = %d/%d, mau 1/1", succeeded, limited)
	}
	if got := countRows(t, f.pool, `
		SELECT count(*) FROM content_authors
		WHERE content_id = $1 AND invite_status IN ('accepted', 'pending')`, created.ID); got != 2 {
		t.Errorf("jumlah author aktif = %d, mau 2", got)
	}
}

func TestUpdateAuthor_UpdatesLevelLabelAndPosition(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.addAuthor(created.ID, f.editorID, LevelEditor)

	updated, err := f.service.UpdateAuthor(f.ctx, f.actor(f.ownerID), created.ID, f.editorID, AuthorUpdateInput{
		Level:     strPtr(LevelContributor),
		RoleLabel: strPtr("Animator"),
		Position:  intPtr(7),
	})
	if err != nil {
		t.Fatalf("UpdateAuthor: %v", err)
	}
	if updated.Level != LevelContributor || updated.Position != 7 {
		t.Errorf("author = %+v, mau contributor/7", updated)
	}
	if updated.RoleLabel == nil || *updated.RoleLabel != "Animator" {
		t.Errorf("roleLabel = %v", updated.RoleLabel)
	}

	cleared, err := f.service.UpdateAuthor(f.ctx, f.actor(f.ownerID), created.ID, f.editorID, AuthorUpdateInput{
		Level:     strPtr(LevelContributor),
		RoleLabel: nil,
		Position:  intPtr(7),
	})
	if err != nil {
		t.Fatalf("UpdateAuthor clear label: %v", err)
	}
	if cleared.RoleLabel != nil {
		t.Errorf("roleLabel = %v, mau NULL", cleared.RoleLabel)
	}
}

func TestUpdateAuthor_EditorReturnsForbiddenLevel(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.addAuthor(created.ID, f.editorID, LevelEditor)
	f.addAuthor(created.ID, f.contributorID, LevelContributor)

	_, err := f.service.UpdateAuthor(f.ctx, f.actor(f.editorID), created.ID, f.contributorID, AuthorUpdateInput{
		Level: strPtr(LevelEditor),
	})
	requireAppError(t, err, "FORBIDDEN_LEVEL")
}

func TestRemoveAuthor_RemovesAcceptedAuthor(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.addAuthor(created.ID, f.editorID, LevelEditor)

	if err := f.service.RemoveAuthor(f.ctx, f.actor(f.ownerID), created.ID, f.editorID); err != nil {
		t.Fatalf("RemoveAuthor: %v", err)
	}

	row := f.authorRow(created.ID, f.editorID)
	if row.InviteStatus != "removed" || row.RespondedAt == nil {
		t.Errorf("row = %+v, mau removed dengan respondedAt", row)
	}
}

func TestRemoveAuthor_OwnerRemoveSelfReturnsValidation(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)

	err := f.service.RemoveAuthor(f.ctx, f.actor(f.ownerID), created.ID, f.ownerID)
	appErr := requireAppError(t, err, "VALIDATION_FAILED")
	if got := fieldErrorMessage(appErr, "userId"); got != "Pindahkan kepemilikan dulu sebelum keluar." {
		t.Errorf("pesan = %q", got)
	}
}

func TestTransferOwnership_MovesOwnerAndDemotesFormerOwner(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.addAuthor(created.ID, f.editorID, LevelEditor)

	items, err := f.service.TransferOwnership(f.ctx, f.actor(f.ownerID), created.ID, f.editorID)
	if err != nil {
		t.Fatalf("TransferOwnership: %v", err)
	}

	if got := f.contentRow(created.ID).OwnerID; got != f.editorID {
		t.Errorf("owner_id = %s, mau %s", got, f.editorID)
	}
	if row := f.authorRow(created.ID, f.ownerID); row.Level != LevelEditor {
		t.Errorf("level pemilik lama = %s, mau editor", row.Level)
	}
	if row := f.authorRow(created.ID, f.editorID); row.Level != LevelOwner {
		t.Errorf("level pemilik baru = %s, mau owner", row.Level)
	}
	if got := countRows(t, f.pool, `
		SELECT count(*) FROM content_authors
		WHERE content_id = $1 AND level = 'owner'`, created.ID); got != 1 {
		t.Errorf("jumlah owner = %d, mau 1", got)
	}
	if len(items) != 2 {
		t.Errorf("items = %d, mau 2", len(items))
	}
}

func TestTransferOwnership_PendingAuthorReturnsNotFound(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.addPendingInvite(created.ID, f.editorID, time.Now().UTC().Add(24*time.Hour))

	_, err := f.service.TransferOwnership(f.ctx, f.actor(f.ownerID), created.ID, f.editorID)
	requireAppError(t, err, "NOT_FOUND")
}

func TestLeaveContent_EditorLeaves(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)
	f.addAuthor(created.ID, f.editorID, LevelEditor)

	if err := f.service.LeaveContent(f.ctx, f.actor(f.editorID), created.ID); err != nil {
		t.Fatalf("LeaveContent: %v", err)
	}

	row := f.authorRow(created.ID, f.editorID)
	if row.InviteStatus != "left" || row.RespondedAt == nil {
		t.Errorf("row = %+v, mau left dengan respondedAt", row)
	}
}

func TestLeaveContent_OwnerReturnsValidation(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)

	err := f.service.LeaveContent(f.ctx, f.actor(f.ownerID), created.ID)
	appErr := requireAppError(t, err, "VALIDATION_FAILED")
	if got := fieldErrorMessage(appErr, "userId"); got != "Pindahkan kepemilikan dulu sebelum keluar." {
		t.Errorf("pesan = %q", got)
	}
}

func TestLeaveContent_NonAuthorReturnsNotFound(t *testing.T) {
	f := newFixture(t)
	created := f.create(TypePortfolio)

	err := f.service.LeaveContent(f.ctx, f.actor(f.outsiderID), created.ID)
	requireAppError(t, err, "NOT_FOUND")
}

func intPtr(value int) *int { return &value }
