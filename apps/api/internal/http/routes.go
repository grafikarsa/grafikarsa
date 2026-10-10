package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"grafikarsa/api/internal/cache"
	"grafikarsa/api/internal/config"
	"grafikarsa/api/internal/db"
	"grafikarsa/api/internal/observability"
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
	r.Get("/metrics", observability.MetricsHandler())
	r.Route("/v1", func(r chi.Router) {
		// Rate limit per IP (gate ganti-password ada di authRequired).
		r.Use(a.rateLimit)
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
		})
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", a.handleLogin)
			r.Post("/refresh", a.handleRefresh)
			r.Group(func(r chi.Router) {
				r.Use(a.authRequired)
				r.Post("/logout", a.csrf(a.handleLogout))
				r.Post("/logout-all", a.csrf(a.handleLogoutAll))
				r.Get("/sessions", a.handleSessions)
				r.Delete("/sessions/{id}", a.csrf(a.handleDeleteSession))
			})
		})
		r.Group(func(r chi.Router) {
			r.Use(a.authRequired)
			r.Get("/me", a.handleMe)
			r.Patch("/me", a.csrf(a.handleUpdateMe))
			r.Patch("/me/password", a.csrf(a.handleUpdatePassword))
			r.Get("/me/permissions", a.handleMyPermissions)
			r.Get("/me/check-username", a.handleCheckUsername)
			r.Put("/me/social-links", a.csrf(a.handleUpdateSocialLinks))
			r.Get("/me/notification-prefs", a.handleNotifPrefs)
			r.Put("/me/notification-prefs", a.csrf(a.handleSaveNotifPrefs))
		})
		r.Group(func(r chi.Router) {
			// Direktori & profil publik; field sensitif digate di handler.
			r.Use(a.authOptional)
			r.Get("/users", a.handleListUsers)
			r.Get("/users/{username}", a.handleGetUser)
			r.Get("/users/{username}/followers", a.handleListFollowers)
			r.Get("/users/{username}/following", a.handleListFollowing)
		})
		r.Route("/admin", func(r chi.Router) {
			r.Use(a.authRequired)
			r.Route("/portfolios", func(r chi.Router) {
				r.Use(a.requireAnyPerm("portfolios.view_all", "portfolios.moderate_jurusan"))
				r.Get("/", a.handleAdminListPortfolios)
				r.Get("/pending", a.handleAdminPendingPortfolios)
				r.With(a.requireAnyPerm("portfolios.moderate")).Post("/bulk-approve", a.csrf(a.handleBulkApprove))
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", a.handleAdminGetPortfolio)
					r.With(a.requireAnyPerm("portfolios.moderate", "portfolios.moderate_jurusan")).Post("/approve", a.csrf(a.handleApprovePortfolio))
					r.With(a.requireAnyPerm("portfolios.moderate", "portfolios.moderate_jurusan")).Post("/request-changes", a.csrf(a.handleRequestChanges))
					r.With(a.requireAnyPerm("portfolios.moderate", "portfolios.moderate_jurusan")).Post("/reject", a.csrf(a.handleRejectPortfolio))
					r.With(a.requireAnyPerm("portfolios.moderate")).Post("/restore", a.csrf(a.handleRestorePortfolio))
				})
			})
			r.Route("/tags", func(r chi.Router) {
				r.Use(a.requirePerm("tags.manage"))
				r.Get("/", a.handleListTags)
				r.Post("/", a.csrf(a.handleCreateTag))
				r.Patch("/{id}", a.csrf(a.handleUpdateTag))
				r.Delete("/{id}", a.csrf(a.handleDeleteTag))
			})
			r.Route("/series", func(r chi.Router) {
				r.Use(a.requirePerm("series.manage"))
				r.Get("/", a.handleListSeries)
				r.Post("/", a.csrf(a.handleCreateSeries))
			})
			r.Route("/users", func(r chi.Router) {
				r.Use(a.requireAnyPerm("users.view", "users.view_jurusan"))
				r.Get("/", a.handleAdminListUsers)
				r.With(a.requireAnyPerm("users.create", "users.create_jurusan")).Post("/", a.csrf(a.handleAdminCreateUser))
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", a.handleAdminGetUser)
					r.With(a.requireAnyPerm("users.edit", "users.edit_jurusan")).Patch("/", a.csrf(a.handleAdminUpdateUser))
					r.With(a.requireAnyPerm("users.delete")).Delete("/", a.csrf(a.handleAdminDeleteUser))
					r.With(a.requireAnyPerm("users.activate")).Post("/activate", a.csrf(a.handleAdminActivateUser))
					r.With(a.requireAnyPerm("users.reset_password")).Post("/reset-password", a.csrf(a.handleAdminResetPassword))
				})
			})
			r.Route("/roles", func(r chi.Router) {
				r.Use(a.requirePerm("roles.view"))
				r.Get("/", a.handleListRoles)
				r.With(a.requirePerm("roles.create")).Post("/", a.csrf(a.handleCreateRole))
				r.With(a.requireAnyPerm("roles.assign", "roles.assign_jurusan")).Post("/assign", a.csrf(a.handleAssignRole))
				r.With(a.requireAnyPerm("roles.assign", "roles.assign_jurusan")).Delete("/assign", a.csrf(a.handleUnassignRole))
			})
			r.With(a.requirePerm("users.view")).Get("/audit-logs", a.handleAuditLogs)
			r.Route("/majors", func(r chi.Router) {
				r.Use(a.requirePerm("academic.view"))
				r.Get("/", a.handleListMajors)
				r.With(a.requirePerm("academic.manage_majors")).Post("/", a.csrf(a.handleCreateMajor))
			})
			r.Route("/years", func(r chi.Router) {
				r.Use(a.requirePerm("academic.view"))
				r.Get("/", a.handleListYears)
				r.With(a.requirePerm("academic.manage_years")).Post("/", a.csrf(a.handleCreateYear))
				r.With(a.requirePerm("academic.manage_years")).Post("/{id}/activate", a.csrf(a.handleActivateYear))
			})
			r.Route("/classes", func(r chi.Router) {
				r.Use(a.requirePerm("academic.view"))
				r.Get("/", a.handleListClasses)
				r.With(a.requirePerm("academic.manage_classes")).Post("/", a.csrf(a.handleCreateClass))
				r.With(a.requirePerm("academic.manage_classes")).Delete("/{id}", a.csrf(a.handleDeleteClass))
				r.Get("/{id}/students", a.handleClassStudents)
				r.With(a.requirePerm("academic.manage_classes")).Post("/enroll", a.csrf(a.handleEnrollStudent))
			})
			r.Route("/promotion", func(r chi.Router) {
				r.Use(a.requirePerm("academic.promote_dryrun"))
				r.Get("/jobs", a.handlePromotionJobs)
				r.Post("/jobs/preview", a.csrf(a.handlePromotionPreview))
				r.With(a.requirePerm("academic.promote_execute")).Post("/jobs/{id}/approve", a.csrf(a.handlePromotionApprove))
				r.With(a.requirePerm("academic.promote_execute")).Post("/jobs/{id}/execute", a.csrf(a.handlePromotionExecute))
			})
			r.Route("/assessments", func(r chi.Router) {
				r.Use(a.requirePerm("assessments.view"))
				r.Get("/metrics", a.handleListMetrics)
				r.With(a.requirePerm("assessments.manage_metrics")).Post("/metrics", a.csrf(a.handleCreateMetric))
				r.With(a.requirePerm("assessments.score")).Get("/portfolio/{id}", a.handleGetAssessment)
				r.With(a.requirePerm("assessments.score")).Post("/portfolio/{id}", a.csrf(a.handleScoreAssessment))
			})
			r.With(a.requirePerm("academic.import_students")).Post("/import/students", a.csrf(a.handleImportStudents))
			r.Route("/feedback", func(r chi.Router) {
				r.Use(a.requirePerm("feedback.triage"))
				r.Get("/", a.handleListFeedback)
				r.Get("/{id}", a.handleGetFeedback)
				r.Post("/{id}/reply", a.csrf(a.handleReplyFeedback))
				r.Patch("/{id}/triage", a.csrf(a.handleFeedbackTriage))
			})
			r.Route("/changelogs", func(r chi.Router) {
				r.Use(a.requirePerm("changelog.manage"))
				r.Get("/", a.handleListChangelogs)
				r.Post("/", a.csrf(a.handleCreateChangelog))
				r.With(a.requirePerm("changelog.publish")).Post("/{id}/publish", a.csrf(a.handlePublishChangelog))
				r.With(a.requirePerm("changelog.publish")).Post("/{id}/unpublish", a.csrf(a.handleUnpublishChangelog))
				r.With(a.requirePerm("changelog.publish")).Post("/{id}/archive", a.csrf(a.handleArchiveChangelog))
			})
			r.With(a.requirePerm("system.jobs")).Post("/leaderboards/recompute", a.csrf(a.handleRecomputeLeaderboard))
		})
		r.Group(func(r chi.Router) {
			r.Use(a.authRequired)
			r.With(a.requirePerm("media.upload")).Post("/media/upload", a.csrf(a.handleUpload))
			r.Post("/media/{id}/confirm", a.csrf(a.handleConfirmMedia))
			r.Delete("/media/{id}", a.csrf(a.handleDeleteMedia))
		})
		r.Group(func(r chi.Router) {
			r.Use(a.authOptional)
			r.Get("/media/{id}", a.handleServeMedia)
		})
		r.Route("/portfolios", func(r chi.Router) {
			r.With(a.authOptional).Get("/", a.handleListPublished)
			r.With(a.authOptional).Get("/by-slug", a.handleGetPortfolioBySlug)
			r.With(a.authOptional).Get("/id/{id}", a.handleGetPortfolio)
			r.Group(func(r chi.Router) {
				r.Use(a.authRequired)
				r.With(a.requirePerm("portfolios.create")).Post("/", a.csrf(a.handleCreatePortfolio))
				r.Patch("/id/{id}", a.csrf(a.handleUpdatePortfolio))
				r.Delete("/id/{id}", a.csrf(a.handleDeletePortfolio))
				r.Post("/id/{id}/submit", a.csrf(a.handleSubmitPortfolio))
				r.Post("/id/{id}/cancel", a.csrf(a.handleCancelPortfolio))
				r.Post("/id/{id}/archive", a.csrf(a.handleArchivePortfolio))
				r.Post("/id/{id}/unarchive", a.csrf(a.handleUnarchivePortfolio))
				r.Get("/me", a.handleMyPortfolios)
				r.Post("/{portfolio_id}/blocks", a.csrf(a.handleCreateBlock))
				r.Put("/{portfolio_id}/blocks/reorder", a.csrf(a.handleReorderBlocks))
			})
		})
		r.Group(func(r chi.Router) {
			r.Use(a.authRequired)
			r.Patch("/blocks/{blockId}", a.csrf(a.handleUpdateBlock))
			r.Delete("/blocks/{blockId}", a.csrf(a.handleDeleteBlock))
		})
		r.Group(func(r chi.Router) {
			r.Use(a.authOptional)
			r.Get("/tags", a.handleListTags)
			r.Get("/series", a.handleListSeries)
			r.Get("/series/{id}", a.handleGetSeries)
		})
		r.Group(func(r chi.Router) {
			r.Use(a.authRequired)
			r.Post("/users/{username}/follow", a.csrf(a.handleFollow))
			r.Delete("/users/{username}/follow", a.csrf(a.handleUnfollow))
			r.Post("/portfolios/id/{id}/like", a.csrf(a.handleLike))
			r.Delete("/portfolios/id/{id}/like", a.csrf(a.handleUnlike))
			r.Post("/portfolios/id/{id}/comments", a.csrf(a.handleCreateComment))
			r.Delete("/comments/{id}", a.csrf(a.handleDeleteComment))
		})
		r.Group(func(r chi.Router) {
			r.Use(a.authOptional)
			r.Get("/portfolios/id/{id}/comments", a.handleListComments)
		})
		r.Group(func(r chi.Router) {
			r.Use(a.authOptional)
			r.Get("/feed", a.handleFeed)
		})
		r.Group(func(r chi.Router) {
			r.Use(a.authRequired)
			r.Get("/feed/preferences", a.handleGetFeedPrefs)
			r.Put("/feed/preferences", a.csrf(a.handleUpdateFeedPrefs))
		})
		r.Group(func(r chi.Router) {
			r.Use(a.authOptional)
			r.Get("/leaderboards", a.handleLeaderboards)
			r.Get("/search", a.handleSearch)
			r.Get("/changelogs", a.handlePublishedChangelogs)
			r.Get("/changelogs/{id}", a.handleChangelogDetail)
		})
		r.Group(func(r chi.Router) {
			r.Use(a.authOptional)
			r.Post("/feedback", a.handleCreateFeedback)
		})
		r.Group(func(r chi.Router) {
			r.Use(a.authRequired)
			r.Get("/feedback", a.handleListFeedback)
			r.Get("/feedback/{id}", a.handleGetFeedback)
			r.Post("/feedback/{id}/reply", a.csrf(a.handleReplyFeedback))
			r.Post("/feedback/{id}/vote", a.csrf(a.handleVoteFeedback))
			r.Post("/changelogs/{id}/read", a.csrf(a.handleReadChangelog))
			r.Get("/changelogs/unread-count", a.handleUnreadChangelogCount)
			r.Post("/changelogs/mark-all-read", a.csrf(a.handleMarkAllChangelogsRead))
		})
		r.Group(func(r chi.Router) {
			r.Use(a.authRequired)
			r.Get("/notifications", a.handleListNotifications)
			r.Get("/notifications/count", a.handleNotifCount)
			r.Patch("/notifications/{id}/read", a.csrf(a.handleNotifRead))
			r.Post("/notifications/read-all", a.csrf(a.handleNotifReadAll))
			r.Delete("/notifications/{id}", a.csrf(a.handleNotifDelete))
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
