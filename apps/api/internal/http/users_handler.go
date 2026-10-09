package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"grafikarsa/api/internal/auth"
	"grafikarsa/api/internal/repo"
)

var validate = validator.New()

type updateMeRequest struct {
	Name     *string `json:"name" validate:"omitempty,min=2,max=100"`
	Bio      *string `json:"bio" validate:"omitempty,max=500"`
	Username *string `json:"username" validate:"omitempty,min=3,max=30"`
}

type updatePasswordRequest struct {
	Current string `json:"current_password" validate:"required"`
	New     string `json:"new_password" validate:"required,min=10,max=128"`
}

func (a *App) handleMe(w http.ResponseWriter, r *http.Request) {
	u, err := a.Q.GetUserByID(r.Context(), mustUUID(UserID(r)))
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, fail("UNAUTHORIZED", "Sesi berakhir.", r))
		return
	}
	writeJSON(w, http.StatusOK, ok(map[string]any{
		"id": u.Username, "username": u.Username, "email": u.Email,
		"name": u.Name, "role_primary": u.RolePrimary, "status": u.Status,
		"must_change_password": u.MustChangePassword,
	}, r))
}

func (a *App) handleMyPermissions(w http.ResponseWriter, r *http.Request) {
	perms := UserPerms(r)
	out := []string{}
	for p := range perms {
		out = append(out, p)
	}
	writeJSON(w, http.StatusOK, ok(out, r))
}

func (a *App) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	var req updateMeRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := validate.Struct(req); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Data tidak valid.", r))
		return
	}
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleUpdatePassword(w http.ResponseWriter, r *http.Request) {
	var req updatePasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := validate.Struct(req); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, failDetails("VALIDATION_ERROR", "Password tidak valid.", []map[string]string{{"field": "new_password", "message": "Minimal 10 karakter."}}, r))
		return
	}
	u, err := a.Q.GetUserByID(r.Context(), mustUUID(UserID(r)))
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, fail("UNAUTHORIZED", "Sesi berakhir.", r))
		return
	}
	if !auth.VerifyPassword(req.Current, u.PasswordHash) && !checkBcrypt(req.Current, u.PasswordHash) {
		writeJSON(w, http.StatusUnauthorized, fail("INVALID_CREDENTIALS", "Password lama salah.", r))
		return
	}
	_ = a.Q.UpdatePasswordHash(r.Context(), updatePwParams(u.ID, auth.HashPassword(req.New)))
	_ = a.Q.RevokeAllUserSessions(r.Context(), u.ID)
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleListUsers(w http.ResponseWriter, r *http.Request) {
	a.handleAdminListUsers(w, r)
}

func (a *App) handleGetUser(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	u, err := a.Q.GetUserByUsername(r.Context(), username)
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "User tidak ditemukan.", r))
		return
	}
	if UserRole(r) != "admin" && !UserPerms(r)["users.view"] && !UserPerms(r)["users.view_jurusan"] {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Kamu tidak memiliki akses ini.", r))
		return
	}
	writeJSON(w, http.StatusOK, ok(publicUser(repo.GetUserByUsernameRow(u)), r))
}
