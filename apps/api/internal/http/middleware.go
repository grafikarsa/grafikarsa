package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"grafikarsa/api/internal/observability"

	"github.com/go-chi/chi/v5/middleware"
	"grafikarsa/api/internal/auth"
	"grafikarsa/api/internal/cache"
	"grafikarsa/api/internal/config"
	"grafikarsa/api/internal/repo"
	"grafikarsa/api/internal/storage"
)

type App struct {
	Cfg   *config.Config
	Q     *repo.Queries
	Cache *cache.Cache
	// Storage dipakai lewat interface (PRD §7.5 ayat 3): handler tidak
	// boleh tahu apa pun soal kode S3 di baliknya.
	Storage storage.Store
	Secure  bool
	Tx      *txRunner
}

func NewApp(cfg *config.Config, q *repo.Queries, c *cache.Cache, st storage.Store) *App {
	return &App{Cfg: cfg, Q: q, Cache: c, Storage: st, Secure: cfg.AppEnv == "production"}
}

func (a *App) authOptional(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := parseAccess(r, a.Cfg.JWT.VerificationKeys())
		if claims == nil {
			next.ServeHTTP(w, r)
			return
		}
		if blacklisted(r.Context(), a.Cache, claims.ID) {
			next.ServeHTTP(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID, claims.Subject)
		ctx = context.WithValue(ctx, ctxRole, claims.RolePrimary)
		ctx = context.WithValue(ctx, ctxJTI, claims.ID)
		ctx = context.WithValue(ctx, ctxScope, claims.ScopeMajorID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *App) authRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := parseAccess(r, a.Cfg.JWT.VerificationKeys())
		if claims == nil {
			writeJSON(w, http.StatusUnauthorized, fail("UNAUTHORIZED", "Sesi berakhir. Silakan login ulang.", r))
			return
		}
		if blacklisted(r.Context(), a.Cache, claims.ID) {
			writeJSON(w, http.StatusUnauthorized, fail("TOKEN_REVOKED", "Sesi sudah diakhiri.", r))
			return
		}
		// Status user dicek dari cache pendek (60 detik): user yang dinonaktifkan
		// atau dihapus langsung kehilangan akses walau access token belum expired.
		if !a.userActive(r.Context(), claims.Subject) {
			writeJSON(w, http.StatusUnauthorized, fail("UNAUTHORIZED", "Akun tidak aktif. Hubungi admin.", r))
			return
		}
		perms := loadPerms(r.Context(), a.Cache, a.Q, claims.Subject)
		ctx := context.WithValue(r.Context(), ctxUserID, claims.Subject)
		ctx = context.WithValue(ctx, ctxRole, claims.RolePrimary)
		ctx = context.WithValue(ctx, ctxJTI, claims.ID)
		ctx = context.WithValue(ctx, ctxScope, claims.ScopeMajorID)
		ctx = context.WithValue(ctx, ctxPerms, perms)
		// Wajib ganti password: blokir endpoint lain sampai diganti.
		if claims.MustChangePassword && !passwordChangeAllowlist[r.URL.Path] {
			writeJSON(w, http.StatusForbidden, fail("PASSWORD_CHANGE_REQUIRED", "Ganti password dulu sebelum melanjutkan.", r))
			return
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// withAnyPerm: lolos bila user punya SALAH SATU permission.
// Dipakai agar admin_jurusan (permission *_jurusan) bisa memakai route yang
// sama dengan admin global, dengan filter scope di query.
func (a *App) withAnyPerm(perms []string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if UserRole(r) == "admin" {
			h(w, r)
			return
		}
		up := UserPerms(r)
		for _, p := range perms {
			if up[p] {
				h(w, r)
				return
			}
		}
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Kamu tidak memiliki akses ini.", r))
	}
}

func (a *App) requireAnyPerm(perms ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return a.withAnyPerm(perms, next.ServeHTTP)
	}
}

// scopeFilter: "" bila user punya permission global (lihat semua),
// atau scope jurusan user untuk permission *_jurusan.
func scopeFilter(r *http.Request, globalPerm, scopedPerm string) string {
	if UserRole(r) == "admin" || UserPerms(r)[globalPerm] {
		return ""
	}
	if UserPerms(r)[scopedPerm] {
		return UserScope(r)
	}
	return "__none__"
}

func (a *App) requirePerm(perm string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if UserRole(r) == "admin" {
				next.ServeHTTP(w, r)
				return
			}
			if !UserPerms(r)[perm] {
				writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Kamu tidak memiliki akses ini.", r))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (a *App) csrf(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next(w, r)
			return
		}
		header := r.Header.Get("X-CSRF-Token")
		cookie, _ := r.Cookie(csrfCookieName(a.Secure))
		if header == "" || cookie == nil || header != cookie.Value {
			writeJSON(w, http.StatusForbidden, fail("CSRF_INVALID", "Token keamanan tidak valid.", r))
			return
		}
		if !verifyCSRFToken(a.Cfg.JWT.CSRFSecret(), UserID(r), header) {
			writeJSON(w, http.StatusForbidden, fail("CSRF_INVALID", "Token keamanan kedaluwarsa.", r))
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), ctxCSRF, true)))
	}
}

// rateLimit: batas request per IP per menit (default 300) via Dragonfly.
// Degradasi aman: bila cache error, request dilewatkan.
func (a *App) rateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := a.Cfg.RateLimit.Requests
		if limit <= 0 {
			limit = 100
		}
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		// Kunci per user bila login (satu sekolah bisa berbagi satu IP publik),
		// selain itu per IP nyata dari BFF.
		id := UserID(r)
		if id == "" {
			id = clientIP(r)
		}
		key := "rl:" + id + ":" + time.Now().UTC().Format("2006-01-02T15:04")
		if n, err := a.Cache.IncrWithExpire(r.Context(), key, time.Minute); err == nil && n > int64(limit) {
			writeJSON(w, http.StatusTooManyRequests, fail("TOO_MANY_REQUESTS", "Terlalu banyak permintaan. Coba lagi sebentar.", r))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// passwordChangeAllowlist: endpoint yang tetap boleh diakses saat user
// wajib ganti password. Dipakai INLINE di authRequired (bukan sebagai
// middleware terpisah) supaya check berlaku seragam di semua route
// ber-token — termasuk route yang menambah/mengubah middleware sendiri.
var passwordChangeAllowlist = map[string]bool{
	"/v1/me": true, "/v1/me/password": true, "/v1/me/permissions": true,
	"/v1/auth/logout": true, "/v1/auth/logout-all": true, "/v1/auth/sessions": true,
}

// handleMetrics: hanya untuk pemegang permission system.jobs (Prometheus
// internal). Endpoint ini tidak boleh terbuka: memuat pola trafik internal.
func (a *App) handleMetrics(w http.ResponseWriter, r *http.Request) {
	claims := parseAccess(r, a.Cfg.JWT.VerificationKeys())
	if claims == nil {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Tidak diizinkan.", r))
		return
	}
	perms := loadPerms(r.Context(), a.Cache, a.Q, claims.Subject)
	if claims.RolePrimary != "admin" && !perms["system.jobs"] {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Tidak diizinkan.", r))
		return
	}
	observability.MetricsHandler()(w, r)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, fail("VALIDATION_ERROR", "Body JSON tidak valid.", r))
		return false
	}
	return true
}

var _ = middleware.GetReqID
var _ = auth.AccessIssuer
