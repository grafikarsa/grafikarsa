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
	users, err := a.Q.ListUsers(r.Context(), repo.ListUsersParams{Limit: limit, Offset: offset})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat user.", r))
		return
	}
	total, _ := a.Q.CountUsersAll(r.Context())
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
	writeJSON(w, http.StatusCreated, ok(map[string]string{"status": "created"}, r))
}

func (a *App) handleAdminGetUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	u, err := a.Q.GetUserByID(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "User tidak ditemukan.", r))
		return
	}
	if UserRole(r) == "admin_jurusan" && uuidStr(u.MajorID) != UserScope(r) && UserScope(r) != "" {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Di luar jurusanmu.", r))
		return
	}
	writeJSON(w, http.StatusOK, ok(map[string]any{"id": u.Username, "username": u.Username, "name": u.Name}, r))
}

func (a *App) handleAdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "deleted"}, r))
}

func (a *App) handleAdminActivateUser(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusCreated, ok(map[string]string{"status": "created"}, r))
}

func (a *App) handleAssignRole(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleUnassignRole(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}
