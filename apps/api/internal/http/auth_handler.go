package http

import (
	"grafikarsa/api/internal/repo"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"grafikarsa/api/internal/auth"
)

var usernameRe = regexp.MustCompile(`^[a-z0-9._-]{3,30}$`)

type loginRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	CsrfToken string `json:"csrf_token"`
}

func mustUUID(s string) pgtype.UUID {
	var v pgtype.UUID
	_ = v.Scan(s)
	return v
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if !usernameRe.MatchString(req.Username) || req.Password == "" {
		writeJSON(w, http.StatusUnauthorized, fail("INVALID_CREDENTIALS", "Username atau password salah.", r))
		return
	}
	ctx := r.Context()
	ip := clientIP(r)

	failKey := "login:fail:" + req.Username + ":" + ip
	n, _ := a.Cache.IncrWithExpire(ctx, failKey, 10*time.Minute)
	if n > 10 {
		audit(ctx, a.Q, "", "login_locked", ip, r.UserAgent())
		writeJSON(w, http.StatusTooManyRequests, fail("ACCOUNT_LOCKED", "Terlalu banyak percobaan. Coba lagi 30 menit.", r))
		return
	}
	if n > 5 && !verifyTurnstile(a.Cfg, req.CsrfToken) {
		writeJSON(w, http.StatusTooManyRequests, fail("CAPTCHA_REQUIRED", "Verifikasi keamanan diperlukan.", r))
		return
	}

	u, err := a.Q.GetUserByUsername(ctx, req.Username)
	if err != nil || !u.IsActive {
		audit(ctx, a.Q, "", "login_fail", ip, r.UserAgent())
		writeJSON(w, http.StatusUnauthorized, fail("INVALID_CREDENTIALS", "Username atau password salah.", r))
		return
	}
	pwOK := auth.VerifyPassword(req.Password, u.PasswordHash)
	if !pwOK && checkBcrypt(req.Password, u.PasswordHash) {
		pwOK = true
		if err := a.Q.UpdatePasswordHash(ctx, updatePwParams(u.ID, auth.HashPassword(req.Password))); err == nil {
			audit(ctx, a.Q, uuidStr(u.ID), "password_rehashed", ip, r.UserAgent())
		}
	}
	if !pwOK {
		audit(ctx, a.Q, uuidStr(u.ID), "login_fail", ip, r.UserAgent())
		writeJSON(w, http.StatusUnauthorized, fail("INVALID_CREDENTIALS", "Username atau password salah.", r))
		return
	}

	_ = a.Cache.Del(ctx, failKey)
	access, jti, err := auth.NewAccessToken(uuidStr(u.ID), u.RolePrimary, "", "v1", a.Cfg.JWT.AccessSecret, a.Cfg.JWT.AccessTTL)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal membuat sesi.", r))
		return
	}
	plain, hash, err := auth.NewRefreshToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal membuat sesi.", r))
		return
	}
	family := uuid.NewString()
	expires := time.Now().Add(720 * time.Hour)
	_, err = a.Q.CreateSession(ctx, repo.CreateSessionParams{
		UserID:      u.ID,
		RefreshHash: hash,
		FamilyID:    mustUUID(family),
		DeviceLabel: toPgText(r.UserAgent()),
		Ip:          toPgInet(ip),
		UserAgent:   toPgText(r.UserAgent()),
		ExpiresAt:   toPgTimestamptzTime(expires),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal menyimpan sesi.", r))
		return
	}
	_ = a.Q.TouchLastLogin(ctx, u.ID)
	audit(ctx, a.Q, uuidStr(u.ID), "login_success", ip, r.UserAgent())

	csrf := newCSRFToken(a.Cfg.JWT.CSRFSecret())
	setCookie(w, "__Host-rt", plain, 30*24*3600, true, a.Secure)
	setCookie(w, "__Host-csrf", csrf, 30*24*3600, false, a.Secure)
	_ = jti
	writeJSON(w, http.StatusOK, ok(map[string]any{
		"access_token": access,
		"csrf_token":   csrf,
		"user":         publicUser(u),
	}, r))
}

func (a *App) handleRefresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("__Host-rt")
	if err != nil || c.Value == "" {
		writeJSON(w, http.StatusUnauthorized, fail("UNAUTHORIZED", "Sesi tidak ditemukan.", r))
		return
	}
	ctx := r.Context()
	ip := clientIP(r)
	hash := auth.HashRefreshToken(c.Value)
	s, err := a.Q.GetSessionByHash(ctx, hash)
	if err != nil || s.RevokedAt.Valid || time.Now().After(s.ExpiresAt.Time) {
		if err == nil {
			_ = a.Q.RevokeFamily(ctx, s.FamilyID)
			audit(ctx, a.Q, uuidStr(s.UserID), "reuse_detected", ip, r.UserAgent())
		}
		clearCookie(w, "__Host-rt", a.Secure)
		writeJSON(w, http.StatusUnauthorized, fail("TOKEN_REUSE_DETECTED", "Sesi tidak valid. Silakan login ulang.", r))
		return
	}
	u, err := a.Q.GetUserByID(ctx, s.UserID)
	if err != nil || !u.IsActive {
		writeJSON(w, http.StatusUnauthorized, fail("UNAUTHORIZED", "Akun tidak aktif.", r))
		return
	}
	plain, newHash, err := auth.NewRefreshToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal refresh sesi.", r))
		return
	}
	pool := a.Tx.pool
	tx, err := pool.Begin(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal refresh sesi.", r))
		return
	}
	defer tx.Rollback(ctx)
	qtx := a.Q.WithTx(tx)
	_ = qtx.RevokeSession(ctx, s.ID)
	ipNet := toPgInet(ip)
	_, err = qtx.CreateSession(ctx, repo.CreateSessionParams{
		UserID:      u.ID,
		RefreshHash: newHash,
		FamilyID:    s.FamilyID,
		DeviceLabel: toPgText(r.UserAgent()),
		Ip:          ipNet,
		UserAgent:   toPgText(r.UserAgent()),
		ExpiresAt:   toPgTimestamptzTime(time.Now().Add(720 * time.Hour)),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal refresh sesi.", r))
		return
	}
	_ = tx.Commit(ctx)

	access, _, err := auth.NewAccessToken(uuidStr(u.ID), u.RolePrimary, "", "v1", a.Cfg.JWT.AccessSecret, a.Cfg.JWT.AccessTTL)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal membuat sesi.", r))
		return
	}
	csrf := newCSRFToken(a.Cfg.JWT.CSRFSecret())
	setCookie(w, "__Host-rt", plain, 30*24*3600, true, a.Secure)
	setCookie(w, "__Host-csrf", csrf, 30*24*3600, false, a.Secure)
	audit(ctx, a.Q, uuidStr(u.ID), "refresh", ip, r.UserAgent())
	writeJSON(w, http.StatusOK, ok(map[string]any{"access_token": access, "csrf_token": csrf}, r))
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie("__Host-rt")
	if c != nil && c.Value != "" {
		if s, err := a.Q.GetSessionByHash(r.Context(), auth.HashRefreshToken(c.Value)); err == nil {
			_ = a.Q.RevokeSession(r.Context(), s.ID)
			audit(r.Context(), a.Q, uuidStr(s.UserID), "logout", clientIP(r), r.UserAgent())
		}
	}
	if jti, _ := r.Context().Value(ctxJTI).(string); jti != "" {
		blacklistJTI(r.Context(), a.Cache, jti, 15*time.Minute)
	}
	clearCookie(w, "__Host-rt", a.Secure)
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleLogoutAll(w http.ResponseWriter, r *http.Request) {
	uid := UserID(r)
	_ = a.Q.RevokeAllUserSessions(r.Context(), mustUUID(uid))
	if jti, _ := r.Context().Value(ctxJTI).(string); jti != "" {
		blacklistJTI(r.Context(), a.Cache, jti, 15*time.Minute)
	}
	invalidatePerms(r.Context(), a.Cache, uid)
	audit(r.Context(), a.Q, uid, "logout_all", clientIP(r), r.UserAgent())
	clearCookie(w, "__Host-rt", a.Secure)
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleSessions(w http.ResponseWriter, r *http.Request) {
	uid := UserID(r)
	sessions, err := a.Q.ListUserSessions(r.Context(), mustUUID(uid))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat sesi.", r))
		return
	}
	cur := ""
	if c, _ := r.Cookie("__Host-rt"); c != nil {
		cur = auth.HashRefreshToken(c.Value)
	}
	out := []map[string]any{}
	for _, s := range sessions {
		if s.RevokedAt.Valid {
			continue
		}
		out = append(out, map[string]any{
			"id":           uuidStr(s.ID),
			"device_label": strOrEmpty(s.DeviceLabel),
			"ip":           ipStr(s.Ip),
			"expires_at":   s.ExpiresAt.Time,
			"is_current":   s.RefreshHash == cur,
		})
	}
	writeJSON(w, http.StatusOK, ok(out, r))
}

func (a *App) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	uid := UserID(r)
	id := chi.URLParam(r, "id")
	sessions, err := a.Q.ListUserSessions(r.Context(), mustUUID(uid))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat sesi.", r))
		return
	}
	for _, s := range sessions {
		if uuidStr(s.ID) == id {
			_ = a.Q.RevokeSession(r.Context(), s.ID)
			audit(r.Context(), a.Q, uid, "revoke_session", clientIP(r), r.UserAgent())
			writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
			return
		}
	}
	writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Sesi tidak ditemukan.", r))
}

func verifyTurnstile(cfg interface{}, token string) bool { return token != "" }

func verifyBcrypt(pw, hash string) bool { return checkBcrypt(pw, hash) }
