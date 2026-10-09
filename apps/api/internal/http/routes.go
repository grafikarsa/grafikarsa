package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"grafikarsa/api/internal/cache"
	"grafikarsa/api/internal/config"
	"grafikarsa/api/internal/db"
	"grafikarsa/api/internal/repo"
	"grafikarsa/api/internal/storage"
)

type Deps struct {
	Pool    *pgxpool.Pool
	Cache   *cache.Cache
	Storage *storage.Storage
}

func (a *App) routes() http.Handler {
	r := baseMiddleware()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
	})
	r.Get("/ready", a.handleReady)
	r.Route("/v1", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
		})
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", a.handleLogin)
			r.Post("/refresh", a.handleRefresh)
			r.Group(func(r chi.Router) {
				r.Use(a.authRequired)
				r.Post("/logout", a.handleLogout)
				r.Post("/logout-all", a.handleLogoutAll)
				r.Get("/sessions", a.handleSessions)
				r.Delete("/sessions/{id}", a.handleDeleteSession)
			})
		})
		r.Group(func(r chi.Router) {
			r.Use(a.authRequired)
			r.Get("/me", a.handleMe)
			r.Patch("/me", a.csrf(a.handleUpdateMe))
			r.Patch("/me/password", a.csrf(a.handleUpdatePassword))
			r.Get("/me/permissions", a.handleMyPermissions)
		})
		r.Group(func(r chi.Router) {
			r.Use(a.authRequired)
			r.Get("/users", a.withPerm("users.view", a.handleListUsers))
			r.Get("/users/{username}", a.withPerm("users.view", a.handleGetUser))
		})
		r.Route("/admin", func(r chi.Router) {
			r.Use(a.authRequired)
			r.Route("/users", func(r chi.Router) {
				r.Use(a.requirePerm("users.view"))
				r.Get("/", a.handleAdminListUsers)
				r.With(a.requirePerm("users.create")).Post("/", a.csrf(a.handleAdminCreateUser))
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", a.handleAdminGetUser)
					r.With(a.requirePerm("users.edit")).Patch("/", a.csrf(a.handleAdminUpdateUser))
					r.With(a.requirePerm("users.delete")).Delete("/", a.csrf(a.handleAdminDeleteUser))
					r.With(a.requirePerm("users.activate")).Post("/activate", a.csrf(a.handleAdminActivateUser))
					r.With(a.requirePerm("users.reset_password")).Post("/reset-password", a.csrf(a.handleAdminResetPassword))
				})
			})
			r.Route("/roles", func(r chi.Router) {
				r.Use(a.requirePerm("roles.view"))
				r.Get("/", a.handleListRoles)
				r.With(a.requirePerm("roles.create")).Post("/", a.csrf(a.handleCreateRole))
				r.With(a.requirePerm("roles.assign")).Post("/assign", a.csrf(a.handleAssignRole))
				r.With(a.requirePerm("roles.assign")).Delete("/assign", a.csrf(a.handleUnassignRole))
			})
		})
		r.Group(func(r chi.Router) {
			r.Use(a.authRequired)
			r.Post("/media/upload", a.csrf(a.handleUpload))
			r.Post("/media/{id}/confirm", a.csrf(a.handleConfirmMedia))
			r.Delete("/media/{id}", a.csrf(a.handleDeleteMedia))
		})
		r.Group(func(r chi.Router) {
			r.Use(a.authOptional)
			r.Get("/media/{id}", a.handleServeMedia)
		})
	})
	return r
}

func (a *App) handleReady(w http.ResponseWriter, r *http.Request) {
	checks := map[string]string{}
	status := http.StatusOK
	if err := db.Health(r.Context(), a.Tx.pool); err != nil {
		checks["db"] = "error"
		status = http.StatusServiceUnavailable
	} else {
		checks["db"] = "ok"
	}
	if err := a.Cache.Ping(r.Context()); err != nil {
		checks["dragonfly"] = "error"
		status = http.StatusServiceUnavailable
	} else {
		checks["dragonfly"] = "ok"
	}
	if err := a.Storage.Health(r.Context()); err != nil {
		checks["storage"] = "error"
		status = http.StatusServiceUnavailable
	} else {
		checks["storage"] = "ok"
	}
	if status != http.StatusOK {
		writeJSON(w, status, fail("SERVICE_UNAVAILABLE", "Salah satu dependensi belum siap.", r))
		return
	}
	writeJSON(w, status, ok(checks, r))
}

func NewRouter(d Deps) http.Handler {
	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}
	app := NewApp(cfg, repo.New(d.Pool), d.Cache, d.Storage)
	app.Tx = newTxRunner(d.Pool)
	return withBase(app.routes())
}

func withBase(h http.Handler) http.Handler {
	return h
}
