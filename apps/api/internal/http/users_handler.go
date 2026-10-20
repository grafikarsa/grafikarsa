package http

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/trustelem/zxcvbn"
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

// passwordStrong: minimal 10 karakter + zxcvbn score >= 3 (PRD §6.2).
//
// Aturan lama ("harus ada huruf besar, kecil, dan angka") jauh terlalu lemah:
// "Password1" dan "password123" lolos padahalamong paling mudah ditebak.
// zxcvbn mengukur kekuatan sebenarnya: membandingkan dengan kamus password
// umum, pola keyboard (qwerty, asdf), urutan/number, dan pengulangan karakter.
//
// userInputs dipakai untuk menolak password yang memuat username/email user
// sendiri ("budi2024" untuk user "budi") — pola yang paling sering dipakai
// siswa.
//
// Catatan performa: ~0,3-1,2 ms per panggilan, hanya dipanggil di jalur
// ganti/set password, bukan di hot path.
//
// Keterbatasan yang perlu diketahui: kamus zxcvbn berbahasa Inggris, jadi
// password umum berbahasa Indonesia (mis. "Pendidikan1") tidak dikenali
// sebagai lemah. Aturan ini karena itu tidak menggantikan zxcvbn atau
// daftar password lemah berbahasa Indonesia sendiri.
func passwordStrong(pw string, userInputs ...string) (bool, string) {
	if len(pw) < 10 {
		return false, "Minimal 10 karakter."
	}
	if len(pw) > 128 {
		return false, "Maksimal 128 karakter."
	}
	res := zxcvbn.PasswordStrength(pw, userInputs)
	if res.Score < 3 {
		return false, "Password terlalu lemah. Hindari pola umum (mis. 'password123', 'qwerty') dan gabungkan beberapa kata unik."
	}
	return true, ""
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
		if usernameReserved(*req.Username) {
			writeJSON(w, http.StatusUnprocessableEntity, failDetails("VALIDATION_ERROR", "Username tidak tersedia.", []map[string]string{{"field": "username", "message": "Username ini dicadangkan sistem."}}, r))
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
	if n, _ := a.Cache.IncrWithExpire(r.Context(), "chkuser:"+UserID(r), time.Minute); n > 30 {
		writeJSON(w, http.StatusTooManyRequests, fail("TOO_MANY_REQUESTS", "Terlalu sering. Coba lagi sebentar.", r))
		return
	}
	q := r.URL.Query().Get("username")
	if !usernameRe.MatchString(q) || usernameReserved(q) {
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
	action := strings.TrimSpace(r.URL.Query().Get("action"))
	entity := strings.TrimSpace(r.URL.Query().Get("entity"))
	rows, err := a.Q.GetAuditLogsFiltered(r.Context(), repo.GetAuditLogsFilteredParams{
		Column1: action, Column2: entity, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat audit.", r))
		return
	}
	out := []map[string]any{}
	for _, l := range rows {
		note := ""
		var diff map[string]any
		if len(l.Diff) > 0 && json.Unmarshal(l.Diff, &diff) == nil {
			if v, ok := diff["note"].(string); ok {
				note = v
			}
		}
		out = append(out, map[string]any{
			"id": l.ID, "actor_id": nullableUUID(l.ActorID), "action": l.Action,
			"entity": l.Entity, "entity_id": l.EntityID, "note": note,
			"ip": l.Ip, "created_at": l.CreatedAt.Time,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	total, _ := a.Q.CountAuditLogs(r.Context(), repo.CountAuditLogsParams{Column1: action, Column2: entity})
	writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, total), r))
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
	// User diambil lebih dulu supaya username & email-nya bisa dipakai sebagai
	// userInput zxcvbn: password yang memuat username sendiri ditolak.
	u, err := a.Q.GetUserByID(r.Context(), mustUUID(UserID(r)))
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, fail("UNAUTHORIZED", "Sesi berakhir.", r))
		return
	}
	if ok, msg := passwordStrong(req.New, u.Username, u.Email); !ok {
		writeJSON(w, http.StatusUnprocessableEntity, failDetails("VALIDATION_ERROR", "Password lemah.", []map[string]string{{"field": "new_password", "message": msg}}, r))
		return
	}
	if !auth.VerifyPassword(req.Current, u.PasswordHash) && !checkBcrypt(req.Current, u.PasswordHash) {
		writeJSON(w, http.StatusUnauthorized, fail("INVALID_CREDENTIALS", "Password lama salah.", r))
		return
	}
	_ = a.Q.UpdatePasswordHash(r.Context(), updatePwParams(u.ID, auth.HashPassword(req.New)))
	// Password sudah diganti: cabut flag wajib-ganti + semua sesi lain.
	_ = a.Q.SetMustChangePassword(r.Context(), repo.SetMustChangePasswordParams{ID: u.ID, MustChangePassword: false})
	_ = a.Q.RevokeAllUserSessions(r.Context(), u.ID)
	auditLog(r, a, "profile.password_change", "users", uuidStr(u.ID), "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

// handleListUsers: direktori publik (tanpa email).
func (a *App) handleListUsers(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := pageParams(r)
	rows, err := a.Q.ListUsersScoped(r.Context(), repo.ListUsersScopedParams{
		Column1: "", Limit: limit, Offset: offset,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat user.", r))
		return
	}
	search := strings.ToLower(r.URL.Query().Get("search"))
	out := []map[string]any{}
	for _, u := range rows {
		if u.RolePrimary == "admin" {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(u.Username), search) && !strings.Contains(strings.ToLower(u.Name), search) {
			continue
		}
		out = append(out, map[string]any{
			"id": uuidStr(u.ID), "username": u.Username, "name": u.Name,
			"role_primary": u.RolePrimary, "status": u.Status,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	total, _ := a.Q.CountUsersScoped(r.Context(), "")
	writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, total), r))
}

func (a *App) handleGetUser(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	u, err := a.Q.GetUserByUsername(r.Context(), username)
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "User tidak ditemukan.", r))
		return
	}
	// Profil publik: email hanya untuk diri sendiri, admin, atau bila
	// user mengaktifkan show_email.
	isSelf := UserID(r) == uuidStr(u.ID)
	seeEmail := isSelf || UserRole(r) == "admin" || u.ShowEmail
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
	if seeEmail {
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
