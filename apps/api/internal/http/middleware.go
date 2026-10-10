package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"grafikarsa/api/internal/auth"
	"grafikarsa/api/internal/cache"
	"grafikarsa/api/internal/config"
	"grafikarsa/api/internal/repo"
	"grafikarsa/api/internal/storage"
)

type App struct {
	Cfg     *config.Config
	Q       *repo.Queries
	Cache   *cache.Cache
	Storage *storage.Storage
	Secure  bool
	Tx      *txRunner
}

func NewApp(cfg *config.Config, q *repo.Queries, c *cache.Cache, st *storage.Storage) *App {
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
		perms := loadPerms(r.Context(), a.Cache, a.Q, claims.Subject)
		ctx := context.WithValue(r.Context(), ctxUserID, claims.Subject)
		ctx = context.WithValue(ctx, ctxRole, claims.RolePrimary)
		ctx = context.WithValue(ctx, ctxJTI, claims.ID)
		ctx = context.WithValue(ctx, ctxScope, claims.ScopeMajorID)
		ctx = context.WithValue(ctx, ctxPerms, perms)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a *App) withPerm(perm string, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if UserRole(r) == "admin" {
			h(w, r)
			return
		}
		if !UserPerms(r)[perm] {
			writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Kamu tidak memiliki akses ini.", r))
			return
		}
		h(w, r)
	}
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
		return http.HandlerFunc(a.withAnyPerm(perms, next.ServeHTTP))
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
		cookie, _ := r.Cookie("__Host-csrf")
		if header == "" || cookie == nil || header != cookie.Value {
			writeJSON(w, http.StatusForbidden, fail("CSRF_INVALID", "Token keamanan tidak valid.", r))
			return
		}
		if !verifyCSRFToken(a.Cfg.JWT.CSRFSecret(), header) {
			writeJSON(w, http.StatusForbidden, fail("CSRF_INVALID", "Token keamanan kedaluwarsa.", r))
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), ctxCSRF, true)))
	}
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
