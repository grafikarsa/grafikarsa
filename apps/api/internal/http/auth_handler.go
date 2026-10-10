package http

import (
	"context"
	"encoding/json"
	"grafikarsa/api/internal/config"
	"grafikarsa/api/internal/repo"
	"net/http"
	"net/url"
	"regexp"
	"strings"
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
	lockKey := "login:lock:" + req.Username + ":" + ip
	if locked, _ := a.Cache.Exists(ctx, lockKey); locked {
		writeJSON(w, http.StatusTooManyRequests, fail("ACCOUNT_LOCKED", "Terkunci 30 menit karena terlalu banyak percobaan.", r))
		return
	}
	n, _ := a.Cache.IncrWithExpire(ctx, failKey, 10*time.Minute)
	if n > 10 {
		_ = a.Cache.Set(ctx, lockKey, "1", 30*time.Minute)
		audit(ctx, a.Q, "", "login_locked", ip, r.UserAgent())
		writeJSON(w, http.StatusTooManyRequests, fail("ACCOUNT_LOCKED", "Terlalu banyak percobaan. Coba lagi 30 menit.", r))
		return
	}
	if n > 5 && !verifyTurnstile(r.Context(), a.Cfg, req.CsrfToken) {
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
	scope := a.effectiveScope(ctx, u.ID, u.RolePrimary)
	access, jti, err := auth.NewAccessToken(uuidStr(u.ID), u.RolePrimary, scope, a.Cfg.JWT.KeyID(), a.Cfg.JWT.AccessSecret, a.Cfg.JWT.AccessTTL)
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

	scope := a.effectiveScope(ctx, u.ID, u.RolePrimary)
	access, _, err := auth.NewAccessToken(uuidStr(u.ID), u.RolePrimary, scope, a.Cfg.JWT.KeyID(), a.Cfg.JWT.AccessSecret, a.Cfg.JWT.AccessTTL)
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

func verifyTurnstile(ctx context.Context, cfg *config.Config, token string) bool {
	secret := cfg.TurnstileSecret()
	if secret == "" {
		return token != ""
	}
	if token == "" {
		return false
	}
	form := url.Values{"secret": {secret}, "response": {token}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://challenges.cloudflare.com/turnstile/v0/siteverify", strings.NewReader(form.Encode()))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	var out struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return false
	}
	return out.Success
}

func verifyBcrypt(pw, hash string) bool { return checkBcrypt(pw, hash) }

// effectiveScope: scope jurusan efektif (user_roles berscope, atau major user
// untuk admin_jurusan) — dipakai untuk row-level filtering, bukan otoritas.
func (a *App) effectiveScope(ctx context.Context, userID pgtype.UUID, role string) string {
	if role == "admin" {
		return ""
	}
	sc, err := a.Q.GetUserScope(ctx, userID)
	if err != nil || !sc.Valid {
		return ""
	}
	return uuidStr(sc)
}
