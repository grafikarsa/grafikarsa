package roles_test

import (
	"testing"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/roles"
)

func TestCreateRole_GlobalAdminCreatesCustomRole(t *testing.T) {
	f := newFixture(t)
	description := "Menyunting konten sekolah"

	role, err := f.service.CreateRole(
		f.ctx,
		f.globalActor(),
		"editor",
		"Editor",
		&description,
		[]string{"report.handle", "content.moderate", "content.moderate"},
	)
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	if role.Slug != "editor" || role.Name != "Editor" {
		t.Errorf("role = %s/%s, mau editor/Editor", role.Slug, role.Name)
	}
	if role.Description == nil || *role.Description != description {
		t.Errorf("Description = %v, mau %q", role.Description, description)
	}
	if role.IsSystem {
		t.Error("IsSystem = true, mau false untuk role kustom")
	}
	if role.HolderCount != 0 {
		t.Errorf("HolderCount = %d, mau 0", role.HolderCount)
	}
	want := []string{"content.moderate", "report.handle"}
	if len(role.Permissions) != len(want) {
		t.Fatalf("Permissions = %v, mau %v", role.Permissions, want)
	}
	for i := range want {
		if role.Permissions[i] != want[i] {
			t.Fatalf("Permissions = %v, mau %v", role.Permissions, want)
		}
	}

	createdBy := f.roleCreatedBy(role.ID)
	if createdBy == nil || *createdBy != f.globalAdminID {
		t.Errorf("created_by = %v, mau %s", createdBy, f.globalAdminID)
	}
	if f.auditCount("role.create") != 1 {
		t.Errorf("audit role.create = %d, mau 1", f.auditCount("role.create"))
	}

	list, err := f.service.ListRoles(f.ctx, f.globalActor())
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	got := findRole(t, list, "editor")
	if len(got.Permissions) != 2 {
		t.Errorf("Permissions di ListRoles = %v, mau 2 izin", got.Permissions)
	}
}

func TestCreateRole_DeniesActorsWithoutManageOrPermission(t *testing.T) {
	f := newFixture(t)

	t.Run("scoped role.assign only", func(t *testing.T) {
		_, err := f.service.CreateRole(f.ctx, f.scopedActor(), "editor_a", "Editor A", nil, nil)
		requireAppError(t, err, "NOT_FOUND")
	})

	t.Run("plain user", func(t *testing.T) {
		_, err := f.service.CreateRole(f.ctx, f.plainActor(), "editor_b", "Editor B", nil, nil)
		requireAppError(t, err, "NOT_FOUND")
	})

	t.Run("permission not held globally", func(t *testing.T) {
		_, err := f.service.CreateRole(
			f.ctx, f.limitedActor(), "editor_c", "Editor C", nil, []string{"content.moderate"},
		)
		requireAppError(t, err, "NOT_FOUND")
	})

	t.Run("permission held globally", func(t *testing.T) {
		if _, err := f.service.CreateRole(
			f.ctx, f.limitedActor(), "editor_d", "Editor D", nil, []string{"content.create"},
		); err != nil {
			t.Fatalf("CreateRole: %v", err)
		}
	})

	if f.inv.count() != 0 {
		t.Errorf("invalidasi = %d kali, mau 0", f.inv.count())
	}
}

func TestCreateRole_ValidatesInput(t *testing.T) {
	f := newFixture(t)
	actor := f.globalActor()

	t.Run("invalid slug", func(t *testing.T) {
		_, err := f.service.CreateRole(f.ctx, actor, "Bad Slug", "Nama Valid", nil, nil)
		e := requireAppError(t, err, "VALIDATION_FAILED")
		requireField(t, e, "slug", "format")
	})

	t.Run("short name", func(t *testing.T) {
		_, err := f.service.CreateRole(f.ctx, actor, "valid_slug", "A", nil, nil)
		e := requireAppError(t, err, "VALIDATION_FAILED")
		requireField(t, e, "name", "length")
	})

	t.Run("unknown permission", func(t *testing.T) {
		_, err := f.service.CreateRole(f.ctx, actor, "valid_slug", "Nama Valid", nil, []string{"does.not_exist"})
		e := requireAppError(t, err, "VALIDATION_FAILED")
		requireField(t, e, "permissions", "not_found")
	})

	t.Run("duplicate slug", func(t *testing.T) {
		_, err := f.service.CreateRole(f.ctx, actor, "moderator", "Moderator Baru", nil, nil)
		e := requireAppError(t, err, "CONFLICT")
		if e.Detail != "Slug role sudah digunakan." {
			t.Errorf("Detail = %q", e.Detail)
		}
	})
}

func TestUpdateRole_SystemRoleNameAndDescription(t *testing.T) {
	f := newFixture(t)
	moderatorID := f.roleID("moderator")
	before := f.rolePermissionKeys("moderator")

	name := "Moderator Sekolah"
	description := "Pengurus moderasi sekolah"
	role, err := f.service.UpdateRole(f.ctx, f.globalActor(), moderatorID, &name, &description)
	if err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}

	if role.Name != name || role.Description == nil || *role.Description != description {
		t.Errorf("role = %q/%v, mau %q/%q", role.Name, role.Description, name, description)
	}
	if !role.IsSystem || role.Slug != "moderator" {
		t.Errorf("role = system %v slug %q, mau system true slug moderator", role.IsSystem, role.Slug)
	}
	if len(role.Permissions) != len(before) {
		t.Errorf("Permissions berubah: %v, mau %v", role.Permissions, before)
	}
	if f.auditCount("role.update") != 1 {
		t.Errorf("audit role.update = %d, mau 1", f.auditCount("role.update"))
	}

	empty := "   "
	cleared, err := f.service.UpdateRole(f.ctx, f.globalActor(), moderatorID, nil, &empty)
	if err != nil {
		t.Fatalf("UpdateRole clear description: %v", err)
	}
	if cleared.Description != nil {
		t.Errorf("Description = %v, mau nil setelah dikosongkan", cleared.Description)
	}
}

func TestSetRolePermissions_SystemRoleReturnsConflict(t *testing.T) {
	f := newFixture(t)
	moderatorID := f.roleID("moderator")

	_, err := f.service.SetRolePermissions(f.ctx, f.globalActor(), moderatorID, []string{"content.moderate"})
	e := requireAppError(t, err, "CONFLICT")
	if e.Detail != "Role sistem tidak bisa diubah." {
		t.Errorf("Detail = %q", e.Detail)
	}

	err = f.service.DeleteRole(f.ctx, f.globalActor(), moderatorID)
	e = requireAppError(t, err, "CONFLICT")
	if e.Detail != "Role sistem tidak bisa diubah." {
		t.Errorf("Detail hapus = %q", e.Detail)
	}
}

func TestSetRolePermissions_UpdatesHoldersAndInvalidates(t *testing.T) {
	f := newFixture(t)
	role := f.createRole(f.globalActor(), "editor", "Editor", []string{"content.moderate"})

	if _, err := f.service.Assign(f.ctx, f.globalActor(), f.targetID, "editor", nil, nil); err != nil {
		t.Fatalf("Assign: %v", err)
	}

	updated, err := f.service.SetRolePermissions(
		f.ctx, f.globalActor(), role.ID, []string{"content.moderate", "report.handle"},
	)
	if err != nil {
		t.Fatalf("SetRolePermissions: %v", err)
	}

	want := []string{"content.moderate", "report.handle"}
	if len(updated.Permissions) != len(want) {
		t.Fatalf("Permissions = %v, mau %v", updated.Permissions, want)
	}
	for i := range want {
		if updated.Permissions[i] != want[i] {
			t.Fatalf("Permissions = %v, mau %v", updated.Permissions, want)
		}
	}
	if updated.HolderCount != 1 {
		t.Errorf("HolderCount = %d, mau 1", updated.HolderCount)
	}
	if f.inv.count() != 2 || !f.inv.contains(f.targetID) {
		t.Errorf("invalidasi = %d kali, mau 2 termasuk %s", f.inv.count(), f.targetID)
	}
	if f.auditCount("role.permissions_update") != 1 {
		t.Errorf("audit role.permissions_update = %d, mau 1", f.auditCount("role.permissions_update"))
	}
}

func TestSetRolePermissions_DeniesAddedPermissionActorDoesNotHold(t *testing.T) {
	f := newFixture(t)
	limited := f.limitedActor()

	role, err := f.service.CreateRole(f.ctx, limited, "limited_role", "Role Terbatas", nil, []string{"content.create"})
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	_, err = f.service.SetRolePermissions(
		f.ctx, limited, role.ID, []string{"content.create", "content.moderate"},
	)
	requireAppError(t, err, "NOT_FOUND")

	list, err := f.service.ListRoles(f.ctx, f.globalActor())
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	got := findRole(t, list, "limited_role")
	if len(got.Permissions) != 1 || got.Permissions[0] != "content.create" {
		t.Fatalf("Permissions = %v, mau tetap [content.create]", got.Permissions)
	}

	if _, err := f.service.SetRolePermissions(f.ctx, limited, role.ID, nil); err != nil {
		t.Fatalf("SetRolePermissions kosong: %v", err)
	}
}

func TestSetRolePermissions_AllowsKeepingPermissionActorDoesNotHold(t *testing.T) {
	f := newFixture(t)
	role := f.createRole(f.globalActor(), "reporting", "Reporting", []string{"report.handle"})

	// The actor does not hold report.handle, but it is not being added, so it
	// may keep it while adding a permission it does hold.
	updated, err := f.service.SetRolePermissions(
		f.ctx, f.limitedActor(), role.ID, []string{"report.handle", "content.create"},
	)
	if err != nil {
		t.Fatalf("SetRolePermissions: %v", err)
	}
	want := []string{"content.create", "report.handle"}
	if len(updated.Permissions) != len(want) {
		t.Fatalf("Permissions = %v, mau %v", updated.Permissions, want)
	}
	for i := range want {
		if updated.Permissions[i] != want[i] {
			t.Fatalf("Permissions = %v, mau %v", updated.Permissions, want)
		}
	}
}

func TestDeleteRole_WithActiveHoldersReturnsConflict(t *testing.T) {
	f := newFixture(t)
	role := f.createRole(f.globalActor(), "temporary", "Sementara", nil)

	if _, err := f.service.Assign(f.ctx, f.globalActor(), f.targetID, "temporary", nil, nil); err != nil {
		t.Fatalf("Assign: %v", err)
	}

	err := f.service.DeleteRole(f.ctx, f.globalActor(), role.ID)
	e := requireAppError(t, err, "CONFLICT")
	if e.Detail != "Role ini masih dimiliki pengguna." {
		t.Errorf("Detail = %q", e.Detail)
	}
	if !f.roleExists(role.ID) {
		t.Error("role terhapus padahal masih dimiliki pengguna")
	}
	if f.auditCount("role.delete") != 0 {
		t.Errorf("audit role.delete = %d, mau 0", f.auditCount("role.delete"))
	}
}

func TestDeleteRole_AfterRevokeSucceedsAndAudits(t *testing.T) {
	f := newFixture(t)
	role := f.createRole(f.globalActor(), "temporary", "Sementara", nil)

	assignment, err := f.service.Assign(f.ctx, f.globalActor(), f.targetID, "temporary", nil, nil)
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if err := f.service.Revoke(f.ctx, f.globalActor(), assignment.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}

	if err := f.service.DeleteRole(f.ctx, f.globalActor(), role.ID); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
	if f.roleExists(role.ID) {
		t.Error("role masih ada setelah dihapus")
	}
	if f.auditCount("role.delete") != 1 {
		t.Errorf("audit role.delete = %d, mau 1", f.auditCount("role.delete"))
	}
}

func TestDeleteRole_WithExpiredHolderSucceeds(t *testing.T) {
	f := newFixture(t)
	role := f.createRole(f.globalActor(), "expired", "Kedaluwarsa", nil)

	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO user_roles (user_id, role_id, assigned_by, expires_at)
		VALUES ($1, $2, $3, now() - interval '1 hour')`,
		f.targetID, role.ID, f.globalAdminID,
	); err != nil {
		t.Fatalf("buat assignment kedaluwarsa: %v", err)
	}

	if err := f.service.DeleteRole(f.ctx, f.globalActor(), role.ID); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}
	if f.roleExists(role.ID) {
		t.Error("role masih ada setelah dihapus")
	}
}

func TestListPermissionsAndRoles_RequireVisibility(t *testing.T) {
	f := newFixture(t)

	if _, err := f.service.ListPermissions(f.ctx, f.plainActor()); err == nil {
		t.Fatal("ListPermissions plain = nil, mau NOT_FOUND")
	} else {
		requireAppError(t, err, "NOT_FOUND")
	}
	if _, err := f.service.ListRoles(f.ctx, f.plainActor()); err == nil {
		t.Fatal("ListRoles plain = nil, mau NOT_FOUND")
	} else {
		requireAppError(t, err, "NOT_FOUND")
	}

	permissions, err := f.service.ListPermissions(f.ctx, f.scopedActor())
	if err != nil {
		t.Fatalf("ListPermissions scoped: %v", err)
	}
	if len(permissions) != len(f.allPermissions) {
		t.Errorf("jumlah izin = %d, mau %d", len(permissions), len(f.allPermissions))
	}
	var assignPermission *roles.Permission
	for i := range permissions {
		if permissions[i].Key == "role.assign" {
			assignPermission = &permissions[i]
		}
	}
	if assignPermission == nil || !assignPermission.Scopable {
		t.Fatalf("role.assign = %+v, mau ada dan scopable", assignPermission)
	}

	list, err := f.service.ListRoles(f.ctx, f.scopedActor())
	if err != nil {
		t.Fatalf("ListRoles scoped: %v", err)
	}
	moderator := findRole(t, list, "moderator")
	if !moderator.IsSystem || moderator.HolderCount != 0 {
		t.Errorf("moderator = system %v holders %d, mau system true holders 0",
			moderator.IsSystem, moderator.HolderCount)
	}
	if len(moderator.Permissions) != len(f.rolePermissionKeys("moderator")) {
		t.Errorf("Permissions moderator = %v", moderator.Permissions)
	}

	if _, err := f.service.Assign(f.ctx, f.globalActor(), f.targetID, "moderator", nil, nil); err != nil {
		t.Fatalf("Assign: %v", err)
	}
	list, err = f.service.ListRoles(f.ctx, f.scopedActor())
	if err != nil {
		t.Fatalf("ListRoles setelah assign: %v", err)
	}
	if got := findRole(t, list, "moderator"); got.HolderCount != 1 {
		t.Errorf("HolderCount moderator = %d, mau 1", got.HolderCount)
	}
}

func findRole(t *testing.T, list []roles.Role, slug string) roles.Role {
	t.Helper()
	for _, role := range list {
		if role.Slug == slug {
			return role
		}
	}
	t.Fatalf("role %s tidak ada di %d role", slug, len(list))
	return roles.Role{}
}
