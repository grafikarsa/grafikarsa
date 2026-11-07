package roles_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

const missingUserID = "00000000-0000-0000-0000-0000000000ff"

func TestAssign_GlobalAdminGrantsModeratorGlobally(t *testing.T) {
	f := newFixture(t)

	assignment, err := f.service.Assign(f.ctx, f.globalActor(), f.targetID, "moderator", nil, nil)
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}

	if assignment.UserID != f.targetID || assignment.RoleSlug != "moderator" {
		t.Errorf("assignment = %+v, mau user %s role moderator", assignment, f.targetID)
	}
	if assignment.MajorID != nil {
		t.Errorf("MajorID = %v, mau nil (global)", assignment.MajorID)
	}
	if assignment.AssignedBy == nil || *assignment.AssignedBy != f.globalAdminID {
		t.Errorf("AssignedBy = %v, mau %s", assignment.AssignedBy, f.globalAdminID)
	}
	if assignment.AssignedAt.IsZero() {
		t.Error("AssignedAt kosong")
	}
	if f.inv.count() != 1 || f.inv.last() != f.targetID {
		t.Errorf("invalidasi = %d kali, terakhir %s; mau 1 kali untuk %s",
			f.inv.count(), f.inv.last(), f.targetID)
	}
	if !f.hasAuditAction("role.assign") {
		t.Error("audit role.assign tidak tercatat")
	}

	list, err := f.service.ListAssignments(f.ctx, f.globalActor(), f.targetID)
	if err != nil {
		t.Fatalf("ListAssignments: %v", err)
	}
	if len(list) != 1 || list[0].ID != assignment.ID {
		t.Fatalf("assignments = %+v, mau satu dengan id %s", list, assignment.ID)
	}
}

func TestAssign_ScopedMajorAdminGrantsModeratorInOwnMajor(t *testing.T) {
	f := newFixture(t)

	assignment, err := f.service.Assign(f.ctx, f.scopedActor(), f.targetID, "moderator", &f.rplID, nil)
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if assignment.MajorID == nil || *assignment.MajorID != f.rplID {
		t.Fatalf("MajorID = %v, mau %s", assignment.MajorID, f.rplID)
	}
	if assignment.AssignedBy == nil || *assignment.AssignedBy != f.scopedAdminID {
		t.Errorf("AssignedBy = %v, mau %s", assignment.AssignedBy, f.scopedAdminID)
	}
	if f.inv.count() != 1 || f.inv.last() != f.targetID {
		t.Errorf("invalidasi = %d kali, terakhir %s; mau 1 kali untuk %s",
			f.inv.count(), f.inv.last(), f.targetID)
	}

	list, err := f.service.ListAssignments(f.ctx, f.scopedActor(), f.targetID)
	if err != nil {
		t.Fatalf("ListAssignments: %v", err)
	}
	if len(list) != 1 || list[0].ID != assignment.ID {
		t.Fatalf("assignments = %+v, mau satu dengan id %s", list, assignment.ID)
	}
}

func TestAssign_ScopedMajorAdminCannotGrantOutsideScope(t *testing.T) {
	f := newFixture(t)
	actor := f.scopedActor()

	t.Run("other major", func(t *testing.T) {
		_, err := f.service.Assign(f.ctx, actor, f.targetID, "moderator", &f.tkjID, nil)
		requireAppError(t, err, "NOT_FOUND")
	})

	t.Run("global target", func(t *testing.T) {
		_, err := f.service.Assign(f.ctx, actor, f.targetID, "moderator", nil, nil)
		requireAppError(t, err, "NOT_FOUND")
	})

	t.Run("role with permissions the actor does not hold", func(t *testing.T) {
		// admin includes settings.manage, which major_admin does not have.
		_, err := f.service.Assign(f.ctx, actor, f.targetID, "admin", &f.rplID, nil)
		requireAppError(t, err, "NOT_FOUND")
	})

	t.Run("unknown role", func(t *testing.T) {
		_, err := f.service.Assign(f.ctx, actor, f.targetID, "does_not_exist", &f.rplID, nil)
		requireAppError(t, err, "NOT_FOUND")
	})

	if f.inv.count() != 0 {
		t.Errorf("invalidasi = %d kali, mau 0 karena semua percobaan ditolak", f.inv.count())
	}
}

func TestAssign_ScopedMajorAdminCanGrantMajorAdminInOwnMajor(t *testing.T) {
	f := newFixture(t)

	// The rule is permission-based: major_admin holds every permission of the
	// major_admin role scoped to RPL, including role.assign, so it may grant
	// that same role in its own major.
	assignment, err := f.service.Assign(f.ctx, f.scopedActor(), f.targetID, "major_admin", &f.rplID, nil)
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if assignment.MajorID == nil || *assignment.MajorID != f.rplID {
		t.Fatalf("MajorID = %v, mau %s", assignment.MajorID, f.rplID)
	}
}

func TestAssign_NonScopableRoleWithMajorScopeReturnsValidation(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.Assign(f.ctx, f.globalActor(), f.targetID, "admin", &f.rplID, nil)
	e := requireAppError(t, err, "VALIDATION_FAILED")
	requireField(t, e, "majorId", "not_scopable")

	if f.inv.count() != 0 {
		t.Errorf("invalidasi = %d kali, mau 0", f.inv.count())
	}
}

func TestAssign_ImplicitRoleReturnsValidation(t *testing.T) {
	f := newFixture(t)

	for _, slug := range []string{"member", "teacher"} {
		t.Run(slug, func(t *testing.T) {
			_, err := f.service.Assign(f.ctx, f.globalActor(), f.targetID, slug, nil, nil)
			e := requireAppError(t, err, "VALIDATION_FAILED")
			requireField(t, e, "role", "implicit")
		})
	}

	if f.inv.count() != 0 {
		t.Errorf("invalidasi = %d kali, mau 0", f.inv.count())
	}
}

func TestAssign_DuplicateReturnsConflict(t *testing.T) {
	f := newFixture(t)

	if _, err := f.service.Assign(f.ctx, f.globalActor(), f.targetID, "moderator", nil, nil); err != nil {
		t.Fatalf("Assign pertama: %v", err)
	}

	_, err := f.service.Assign(f.ctx, f.globalActor(), f.targetID, "moderator", nil, nil)
	e := requireAppError(t, err, "CONFLICT")
	if e.Detail != "Role ini sudah dimiliki pengguna." {
		t.Errorf("Detail = %q", e.Detail)
	}
	if f.inv.count() != 1 {
		t.Errorf("invalidasi = %d kali, mau 1", f.inv.count())
	}
}

func TestAssign_ValidatesExpiryAndTarget(t *testing.T) {
	f := newFixture(t)
	actor := f.globalActor()

	t.Run("past expiry", func(t *testing.T) {
		past := time.Now().Add(-time.Hour)
		_, err := f.service.Assign(f.ctx, actor, f.targetID, "moderator", nil, &past)
		e := requireAppError(t, err, "VALIDATION_FAILED")
		requireField(t, e, "expiresAt", "future")
	})

	t.Run("future expiry", func(t *testing.T) {
		future := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
		assignment, err := f.service.Assign(f.ctx, actor, f.targetID, "moderator", nil, &future)
		if err != nil {
			t.Fatalf("Assign: %v", err)
		}
		if assignment.ExpiresAt == nil || !assignment.ExpiresAt.Equal(future) {
			t.Errorf("ExpiresAt = %v, mau %v", assignment.ExpiresAt, future)
		}
	})

	t.Run("inactive target", func(t *testing.T) {
		_, err := f.service.Assign(f.ctx, actor, f.inactiveID, "moderator", nil, nil)
		e := requireAppError(t, err, "VALIDATION_FAILED")
		requireField(t, e, "user", "invalid")
	})

	t.Run("missing target", func(t *testing.T) {
		_, err := f.service.Assign(f.ctx, actor, uuid.MustParse(missingUserID), "moderator", nil, nil)
		requireAppError(t, err, "NOT_FOUND")
	})

	t.Run("unknown major", func(t *testing.T) {
		missingMajor := uuid.MustParse("00000000-0000-0000-0000-0000000000fe")
		_, err := f.service.Assign(f.ctx, actor, f.targetID, "moderator", &missingMajor, nil)
		e := requireAppError(t, err, "VALIDATION_FAILED")
		requireField(t, e, "majorId", "not_found")
	})

	if f.inv.count() != 1 {
		t.Errorf("invalidasi = %d kali, mau 1", f.inv.count())
	}
}

func TestListAssignments_ScopedActorSeesOnlyOwnScope(t *testing.T) {
	f := newFixture(t)

	if _, err := f.service.Assign(f.ctx, f.globalActor(), f.targetID, "moderator", nil, nil); err != nil {
		t.Fatalf("Assign global: %v", err)
	}
	scoped, err := f.service.Assign(f.ctx, f.globalActor(), f.targetID, "moderator", &f.rplID, nil)
	if err != nil {
		t.Fatalf("Assign RPL: %v", err)
	}

	list, err := f.service.ListAssignments(f.ctx, f.scopedActor(), f.targetID)
	if err != nil {
		t.Fatalf("ListAssignments scoped: %v", err)
	}
	if len(list) != 1 || list[0].ID != scoped.ID {
		t.Fatalf("assignments scoped = %+v, mau hanya %s", list, scoped.ID)
	}

	all, err := f.service.ListAssignments(f.ctx, f.globalActor(), f.targetID)
	if err != nil {
		t.Fatalf("ListAssignments global: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("assignments global = %d, mau 2", len(all))
	}

	if _, err := f.service.ListAssignments(f.ctx, f.plainActor(), f.targetID); err == nil {
		t.Fatal("ListAssignments plain = nil, mau NOT_FOUND")
	} else {
		requireAppError(t, err, "NOT_FOUND")
	}
}

func TestRevoke_RemovesAssignmentAndAudits(t *testing.T) {
	f := newFixture(t)
	actor := f.scopedActor()

	assignment, err := f.service.Assign(f.ctx, actor, f.targetID, "moderator", &f.rplID, nil)
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}

	if err := f.service.Revoke(f.ctx, actor, assignment.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	if f.inv.count() != 2 || f.inv.last() != f.targetID {
		t.Errorf("invalidasi = %d kali, terakhir %s; mau 2 kali untuk %s",
			f.inv.count(), f.inv.last(), f.targetID)
	}

	revokedAt, revokedBy := f.assignmentRevocation(assignment.ID)
	if revokedAt == nil {
		t.Error("revoked_at kosong, mau terisi")
	}
	if revokedBy == nil || *revokedBy != f.scopedAdminID {
		t.Errorf("revoked_by = %v, mau %s", revokedBy, f.scopedAdminID)
	}
	if !f.hasAuditAction("role.revoke") {
		t.Error("audit role.revoke tidak tercatat")
	}

	list, err := f.service.ListAssignments(f.ctx, f.globalActor(), f.targetID)
	if err != nil {
		t.Fatalf("ListAssignments: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("assignments = %+v, mau kosong setelah dicabut", list)
	}

	err = f.service.Revoke(f.ctx, actor, assignment.ID)
	requireAppError(t, err, "NOT_FOUND")
}

func TestRevoke_ScopedActorCannotRevokeOutsideScope(t *testing.T) {
	f := newFixture(t)

	global, err := f.service.Assign(f.ctx, f.globalActor(), f.targetID, "moderator", nil, nil)
	if err != nil {
		t.Fatalf("Assign global: %v", err)
	}
	otherMajor, err := f.service.Assign(f.ctx, f.globalActor(), f.targetID, "moderator", &f.tkjID, nil)
	if err != nil {
		t.Fatalf("Assign TKJ: %v", err)
	}

	if err := f.service.Revoke(f.ctx, f.scopedActor(), global.ID); err == nil {
		t.Fatal("Revoke global = nil, mau NOT_FOUND")
	} else {
		requireAppError(t, err, "NOT_FOUND")
	}
	if err := f.service.Revoke(f.ctx, f.scopedActor(), otherMajor.ID); err == nil {
		t.Fatal("Revoke TKJ = nil, mau NOT_FOUND")
	} else {
		requireAppError(t, err, "NOT_FOUND")
	}

	list, err := f.service.ListAssignments(f.ctx, f.globalActor(), f.targetID)
	if err != nil {
		t.Fatalf("ListAssignments: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("assignments = %d, mau 2 (tidak ada yang tercabut)", len(list))
	}
	if f.inv.count() != 2 {
		t.Errorf("invalidasi = %d kali, mau 2 (hanya dari assign)", f.inv.count())
	}
}
