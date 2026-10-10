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
	Name      *string `json:"name" validate:"omitempty,min=2,max=100"`
	Bio       *string `json:"bio" validate:"omitempty,max=500"`
	Username  *string `json:"username" validate:"omitempty,min=3,max=30"`
	ShowEmail *bool   `json:"show_email"`
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
	links, _ := a.Q.ListSocialLinks(r.Context(), u.ID)
	linkOut := []map[string]string{}
	for _, l := range links {
		linkOut = append(linkOut, map[string]string{"platform": l.Platform, "url": l.Url})
	}
	if linkOut == nil {
		linkOut = []map[string]string{}
	}
	writeJSON(w, http.StatusOK, ok(map[string]any{
		"id": uuidStr(u.ID), "username": u.Username, "email": u.Email,
		"name": u.Name, "bio": u.Bio, "role_primary": u.RolePrimary, "status": u.Status,
		"must_change_password": u.MustChangePassword,
		"avatar_media_id":      nullableUUID(u.AvatarMediaID),
		"banner_media_id":      nullableUUID(u.BannerMediaID),
		"social_links":         linkOut,
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
	uid := mustUUID(UserID(r))
	u, err := a.Q.GetUserByID(r.Context(), uid)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, fail("UNAUTHORIZED", "Sesi berakhir.", r))
		return
	}
	name, bio, username := u.Name, u.Bio, u.Username
	if req.Name != nil {
		name = *req.Name
	}
	if req.Bio != nil {
		bio = *req.Bio
	}
	if req.Username != nil && *req.Username != u.Username {
		if !usernameRe.MatchString(*req.Username) {
			writeJSON(w, http.StatusUnprocessableEntity, failDetails("VALIDATION_ERROR", "Username tidak valid.", []map[string]string{{"field": "username", "message": "3-30 karakter alfanumerik."}}, r))
			return
		}
		n, _ := a.Q.CheckUsernameTaken(r.Context(), repo.CheckUsernameTakenParams{Username: *req.Username, ID: uid})
		if n > 0 {
			writeJSON(w, http.StatusConflict, fail("USERNAME_TAKEN", "Username sudah dipakai.", r))
			return
		}
		username = *req.Username
	}
	showEmail := false
	if req.ShowEmail != nil {
		showEmail = *req.ShowEmail
	}
	_ = a.Q.UpdateMe(r.Context(), repo.UpdateMeParams{ID: uid, Name: name, Bio: bio, Username: username, ShowEmail: showEmail})
	invalidatePerms(r.Context(), a.Cache, UserID(r))
	auditLog(r, a, "profile.update", "users", UserID(r), "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleCheckUsername(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("username")
	if !usernameRe.MatchString(q) {
		writeJSON(w, http.StatusOK, ok(map[string]bool{"available": false}, r))
		return
	}
	n, _ := a.Q.CheckUsernameTaken(r.Context(), repo.CheckUsernameTakenParams{Username: q, ID: mustUUID(UserID(r))})
	writeJSON(w, http.StatusOK, ok(map[string]bool{"available": n == 0}, r))
}

func (a *App) handleUpdateSocialLinks(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Links []struct {
			Platform string `json:"platform"`
			URL      string `json:"url"`
		} `json:"links"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Links) > 14 {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Maksimal 14 tautan.", r))
		return
	}
	uid := mustUUID(UserID(r))
	_ = a.Q.DeleteSocialLinks(r.Context(), uid)
	for _, l := range req.Links {
		if l.Platform == "" || !isHTTPS(l.URL) {
			continue
		}
		_ = a.Q.AddSocialLink(r.Context(), repo.AddSocialLinkParams{UserID: uid, Platform: l.Platform, Url: l.URL})
	}
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleNotifPrefs(w http.ResponseWriter, r *http.Request) {
	uid := mustUUID(UserID(r))
	p, err := a.Q.GetNotifPrefs(r.Context(), uid)
	if err != nil {
		writeJSON(w, http.StatusOK, ok(map[string]bool{"portfolio_liked": true, "new_comment": true, "new_follower": true, "digest_daily": false}, r))
		return
	}
	writeJSON(w, http.StatusOK, ok(map[string]bool{
		"portfolio_liked": p.PortfolioLiked, "new_comment": p.NewComment,
		"new_follower": p.NewFollower, "digest_daily": p.DigestDaily,
	}, r))
}

func (a *App) handleSaveNotifPrefs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PortfolioLiked bool `json:"portfolio_liked"`
		NewComment     bool `json:"new_comment"`
		NewFollower    bool `json:"new_follower"`
		DigestDaily    bool `json:"digest_daily"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	_ = a.Q.SaveNotifPrefs(r.Context(), repo.SaveNotifPrefsParams{
		UserID: mustUUID(UserID(r)), PortfolioLiked: req.PortfolioLiked,
		NewComment: req.NewComment, NewFollower: req.NewFollower, DigestDaily: req.DigestDaily,
	})
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleAuditLogs(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := pageParams(r)
	rows, err := a.Q.GetAuditLogs(r.Context(), repo.GetAuditLogsParams{Limit: limit, Offset: offset})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat audit.", r))
		return
	}
	out := []map[string]any{}
	for _, l := range rows {
		out = append(out, map[string]any{
			"id": l.ID, "actor_id": nullableUUID(l.ActorID), "action": l.Action,
			"entity": l.Entity, "entity_id": l.EntityID, "created_at": l.CreatedAt.Time,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, int64(len(out))), r))
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
	links, _ := a.Q.ListSocialLinks(r.Context(), u.ID)
	linkOut := []map[string]string{}
	for _, l := range links {
		linkOut = append(linkOut, map[string]string{"platform": l.Platform, "url": l.Url})
	}
	if linkOut == nil {
		linkOut = []map[string]string{}
	}
	followers, _ := a.Q.CountFollowers(r.Context(), u.ID)
	following, _ := a.Q.CountFollowing(r.Context(), u.ID)
	published, _ := a.Q.CountUserPublished(r.Context(), u.ID)
	isFollowing := false
	if uid := UserID(r); uid != "" {
		n, _ := a.Q.IsFollowing(r.Context(), repo.IsFollowingParams{FollowerID: mustUUID(uid), FollowingID: u.ID})
		isFollowing = n > 0
	}
	email := ""
	if UserID(r) == uuidStr(u.ID) {
		email = u.Email
	}
	writeJSON(w, http.StatusOK, ok(map[string]any{
		"id": uuidStr(u.ID), "username": u.Username, "email": email,
		"name": u.Name, "bio": u.Bio, "role_primary": u.RolePrimary, "status": u.Status,
		"avatar_media_id": nullableUUID(u.AvatarMediaID), "banner_media_id": nullableUUID(u.BannerMediaID),
		"social_links": linkOut, "followers": followers, "following": following,
		"published_count": published, "is_following": isFollowing,
	}, r))
}

func (a *App) handleListFollowers(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	u, err := a.Q.GetUserByUsername(r.Context(), username)
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "User tidak ditemukan.", r))
		return
	}
	limit, offset, page := pageParams(r)
	rows, err := a.Q.ListFollowers(r.Context(), repo.ListFollowersParams{FollowingID: u.ID, Limit: limit, Offset: offset})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat.", r))
		return
	}
	out := []map[string]any{}
	for _, f := range rows {
		out = append(out, map[string]any{"id": uuidStr(f.ID), "username": f.Username, "name": f.Name})
	}
	if out == nil {
		out = []map[string]any{}
	}
	total, _ := a.Q.CountFollowers(r.Context(), u.ID)
	writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, total), r))
}

func (a *App) handleListFollowing(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	u, err := a.Q.GetUserByUsername(r.Context(), username)
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "User tidak ditemukan.", r))
		return
	}
	limit, offset, page := pageParams(r)
	rows, err := a.Q.ListFollowing(r.Context(), repo.ListFollowingParams{FollowerID: u.ID, Limit: limit, Offset: offset})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat.", r))
		return
	}
	out := []map[string]any{}
	for _, f := range rows {
		out = append(out, map[string]any{"id": uuidStr(f.ID), "username": f.Username, "name": f.Name})
	}
	if out == nil {
		out = []map[string]any{}
	}
	total, _ := a.Q.CountFollowing(r.Context(), u.ID)
	writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, total), r))
}
