package http

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"grafikarsa/api/internal/auth"
	"grafikarsa/api/internal/repo"
)

func pageParams(r *http.Request) (limit, offset int32, page int) {
	limit = 20
	if v, err := strconv.Atoi(r.URL.Query().Get("per_page")); err == nil && v >= 1 && v <= 50 {
		limit = int32(v)
	}
	page = 1
	if v, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && v >= 1 {
		page = v
	}
	offset = int32(page-1) * limit
	return limit, offset, page
}

func meta(page int, limit int32, total int64) map[string]any {
	pages := total / int64(limit)
	if total%int64(limit) > 0 {
		pages++
	}
	return map[string]any{"page": page, "per_page": limit, "total_pages": pages, "total": total}
}

func (a *App) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := pageParams(r)
	// Row-level scope: admin_jurusan hanya melihat user jurusannya.
	scope := scopeFilter(r, "users.view", "users.view_jurusan")
	users, err := a.Q.ListUsersScoped(r.Context(), repo.ListUsersScopedParams{
		Column1: scope, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat user.", r))
		return
	}
	total, _ := a.Q.CountUsersScoped(r.Context(), scope)
	out := []map[string]any{}
	for _, u := range users {
		out = append(out, map[string]any{
			"id": u.ID, "username": u.Username, "email": u.Email, "name": u.Name,
			"role_primary": u.RolePrimary, "status": u.Status, "is_active": u.IsActive,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, total), r))
}

type createUserRequest struct {
	Username string `json:"username" validate:"required,min=3,max=30"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=10,max=128"`
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Role     string `json:"role_primary" validate:"required,oneof=siswa guru admin_jurusan admin"`
}

func (a *App) handleAdminCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := validate.Struct(req); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Data user tidak valid.", r))
		return
	}
	if UserRole(r) == "admin_jurusan" && req.Role != "siswa" && req.Role != "guru" {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Admin jurusan hanya boleh buat siswa/guru.", r))
		return
	}
	if usernameReserved(req.Username) {
		writeJSON(w, http.StatusUnprocessableEntity, failDetails("VALIDATION_ERROR", "Username tidak tersedia.", []map[string]string{{"field": "username", "message": "Username ini dicadangkan sistem."}}, r))
		return
	}
	var majorID = nilUUID()
	if UserRole(r) == "admin_jurusan" && UserScope(r) != "" {
		majorID = mustUUID(UserScope(r))
	}
	id, err := a.Q.CreateUser(r.Context(), repo.CreateUserParams{
		Username: req.Username, Email: req.Email, PasswordHash: auth.HashPassword(req.Password),
		Name: req.Name, RolePrimary: req.Role, Status: "aktif", MajorID: majorID,
		MustChangePassword: true,
	})
	if err != nil {
		writeJSON(w, http.StatusConflict, fail("DUPLICATE_USER", "Username/email sudah dipakai.", r))
		return
	}
	role, _ := a.Q.GetRoleBySlug(r.Context(), req.Role)
	if role.Slug != "" {
		_ = a.Q.AssignUserRole(r.Context(), repo.AssignUserRoleParams{
			UserID: id, RoleID: role.ID, ScopeMajorID: majorID, AssignedBy: mustUUID(UserID(r)),
		})
	}
	invalidatePerms(r.Context(), a.Cache, uuidStr(id))
	auditLog(r, a, "users.create", "users", uuidStr(id), req.Role)
	writeJSON(w, http.StatusCreated, ok(map[string]string{"id": uuidStr(id)}, r))
}

func (a *App) handleAdminGetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, err := a.Q.GetUserByID(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "User tidak ditemukan.", r))
		return
	}
	if UserRole(r) == "admin_jurusan" && (UserScope(r) == "" || !u.MajorID.Valid || uuidStr(u.MajorID) != UserScope(r)) {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Di luar jurusanmu.", r))
		return
	}
	writeJSON(w, http.StatusOK, ok(map[string]any{"id": u.Username, "username": u.Username, "name": u.Name}, r))
}

func (a *App) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, err := a.Q.GetUserByID(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "User tidak ditemukan.", r))
		return
	}
	if UserRole(r) == "admin_jurusan" && (UserScope(r) == "" || !u.MajorID.Valid || uuidStr(u.MajorID) != UserScope(r)) {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Di luar jurusanmu.", r))
		return
	}
	var req struct {
		Name  *string `json:"name"`
		Email *string `json:"email"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	name := u.Name
	if req.Name != nil && *req.Name != "" {
		name = *req.Name
	}
	email := u.Email
	if req.Email != nil && *req.Email != "" {
		email = *req.Email
	}
	_ = a.Q.UpdateUser(r.Context(), repo.UpdateUserParams{ID: u.ID, Name: name, Email: email, MajorID: u.MajorID, ClassID: u.ClassID})
	auditLog(r, a, "users.update", "users", id, "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, err := a.Q.GetUserByID(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "User tidak ditemukan.", r))
		return
	}
	if uuidStr(u.ID) == UserID(r) {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Tidak bisa hapus diri sendiri.", r))
		return
	}
	_ = a.Q.SoftDeleteUser(r.Context(), u.ID)
	_ = a.Q.RevokeAllUserSessions(r.Context(), u.ID)
	invalidatePerms(r.Context(), a.Cache, id)
	auditLog(r, a, "users.delete", "users", id, "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "deleted"}, r))
}

func (a *App) handleAdminActivateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Active bool `json:"active"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	u, err := a.Q.GetUserByID(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "User tidak ditemukan.", r))
		return
	}
	_ = a.Q.SetUserActive(r.Context(), repo.SetUserActiveParams{ID: u.ID, IsActive: req.Active})
	auditLog(r, a, "users.activate", "users", id, "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleAdminResetPassword(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	newPw := uuid.NewString()[:12]
	u, err := a.Q.GetUserByID(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "User tidak ditemukan.", r))
		return
	}
	_ = a.Q.UpdatePasswordHash(r.Context(), repo.UpdatePasswordHashParams{ID: u.ID, PasswordHash: auth.HashPassword(newPw)})
	_ = a.Q.RevokeAllUserSessions(r.Context(), u.ID)
	// Password sementara: wajib diganti saat login pertama + cabut sesi lain
	// + catat audit (sebelumnya flag dan audit terlewat di jalur ini).
	_ = a.Q.SetMustChangePassword(r.Context(), repo.SetMustChangePasswordParams{ID: u.ID, MustChangePassword: true})
	invalidatePerms(r.Context(), a.Cache, uuidStr(u.ID))
	invalidateUserActive(r.Context(), a.Cache, uuidStr(u.ID))
	auditLog(r, a, "users.reset_password", "users", uuidStr(u.ID), "")
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, ok(map[string]string{"password": newPw}, r))
}

func (a *App) handleListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := a.Q.ListRoles(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat roles.", r))
		return
	}
	writeJSON(w, http.StatusOK, ok(roles, r))
}

type createRoleRequest struct {
	Slug  string   `json:"slug" validate:"required,min=2,max=50"`
	Name  string   `json:"name" validate:"required,min=2,max=100"`
	Perms []string `json:"permissions" validate:"required,min=1"`
}

func (a *App) handleCreateRole(w http.ResponseWriter, r *http.Request) {
	var req createRoleRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := validate.Struct(req); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Data role tidak valid.", r))
		return
	}
	if UserRole(r) != "admin" {
		for _, p := range req.Perms {
			if !UserPerms(r)[p] {
				writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Tidak boleh memberi permission yang kamu tidak punya: "+p, r))
				return
			}
		}
	}
	scope := "global"
	if UserRole(r) == "admin_jurusan" {
		scope = "jurusan"
	}
	rid, err := a.Q.CreateRole(r.Context(), repo.CreateRoleParams{Slug: req.Slug, Name: req.Name, ScopeType: scope})
	if err != nil {
		writeJSON(w, http.StatusConflict, fail("DUPLICATE_ROLE", "Slug role sudah dipakai.", r))
		return
	}
	for _, p := range req.Perms {
		_ = a.Q.AddRolePermission(r.Context(), repo.AddRolePermissionParams{RoleID: rid, Key: p})
	}
	auditLog(r, a, "roles.create", "roles", uuidStr(rid), req.Slug)
	writeJSON(w, http.StatusCreated, ok(map[string]string{"id": uuidStr(rid)}, r))
}

func (a *App) handleAssignRole(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID       string  `json:"user_id"`
		RoleSlug     string  `json:"role_slug"`
		ScopeMajorID *string `json:"scope_major_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	role, err := a.Q.GetRoleBySlug(r.Context(), req.RoleSlug)
	if err != nil || role.Slug == "" {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Role tidak ditemukan.", r))
		return
	}
	// Anti-eskalasi: hanya admin global yang boleh membagikan role global
	// (mis. 'admin'); role lain wajib subset permission pemberi.
	if UserRole(r) != "admin" {
		if role.ScopeType != "jurusan" {
			writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Role lintas jurusan hanya untuk admin.", r))
			return
		}
		perms, perr := a.Q.GetRolePermissions(r.Context(), role.ID)
		if perr != nil {
			writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memeriksa role.", r))
			return
		}
		mine := UserPerms(r)
		for _, p := range perms {
			if !mine[p] {
				writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Kamu tidak boleh memberikan permission yang tidak kamu miliki.", r))
				return
			}
		}
	}
	scope := nilUUID()
	if req.ScopeMajorID != nil && *req.ScopeMajorID != "" {
		scope = mustUUID(*req.ScopeMajorID)
	}
	if UserRole(r) == "admin_jurusan" {
		if UserScope(r) == "" || (scope.Valid && uuidStr(scope) != UserScope(r)) {
			writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Hanya dalam jurusanmu.", r))
			return
		}
		scope = mustUUID(UserScope(r))
	}
	u, err := a.Q.GetUserByID(r.Context(), mustUUID(req.UserID))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "User tidak ditemukan.", r))
		return
	}
	// Target harus di dalam scope pemberi (fail-closed).
	if UserRole(r) != "admin" && (!u.MajorID.Valid || uuidStr(u.MajorID) != UserScope(r)) {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "User di luar jurusanmu.", r))
		return
	}
	_ = a.Q.AssignUserRole(r.Context(), repo.AssignUserRoleParams{
		UserID: u.ID, RoleID: role.ID, ScopeMajorID: scope, AssignedBy: mustUUID(UserID(r)),
	})
	invalidatePerms(r.Context(), a.Cache, req.UserID)
	auditLog(r, a, "roles.assign", "user_roles", req.UserID, req.RoleSlug)
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleUnassignRole(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		RoleSlug string `json:"role_slug"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	role, err := a.Q.GetRoleBySlug(r.Context(), req.RoleSlug)
	if err != nil || role.Slug == "" || role.IsSystem {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Role sistem tidak bisa dilepas via ini.", r))
		return
	}
	_ = a.Q.UnassignUserRole(r.Context(), repo.UnassignUserRoleParams{UserID: mustUUID(req.UserID), RoleID: role.ID})
	invalidatePerms(r.Context(), a.Cache, req.UserID)
	auditLog(r, a, "roles.unassign", "user_roles", req.UserID, req.RoleSlug)
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}
