package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"grafikarsa/api/internal/repo"
)

func (a *App) handleAdminListPortfolios(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := pageParams(r)
	rows, err := a.Q.ListPublishedPortfolios(r.Context(), repo.ListPublishedPortfoliosParams{Limit: limit, Offset: offset})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat.", r))
		return
	}
	out := []map[string]any{}
	for _, p := range rows {
		out = append(out, map[string]any{"id": uuidStr(p.ID), "title": p.Title, "slug": p.Slug, "status": "published"})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, int64(len(out))), r))
}

func (a *App) handleAdminPendingPortfolios(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := pageParams(r)
	rows, err := a.Q.ListPendingPortfolios(r.Context(), repo.ListPendingPortfoliosParams{Limit: limit, Offset: offset})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat.", r))
		return
	}
	out := []map[string]any{}
	for _, p := range rows {
		out = append(out, map[string]any{
			"id": uuidStr(p.ID), "title": p.Title, "slug": p.Slug,
			"username": p.Username, "submitted_at": p.SubmittedAt.Time,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, int64(len(out))), r))
}

func (a *App) handleAdminGetPortfolio(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := a.Q.GetPortfolioByID(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
		return
	}
	writeJSON(w, http.StatusOK, ok(portfolioDetail(r, a, p), r))
}

func (a *App) moderate(w http.ResponseWriter, r *http.Request, status string, needNote bool) {
	id := chi.URLParam(r, "id")
	var req struct {
		Note string `json:"note"`
	}
	_ = req
	if needNote {
		if !decodeJSON(w, r, &req) {
			return
		}
		if len(req.Note) < 5 {
			writeJSON(w, http.StatusUnprocessableEntity, failDetails("VALIDATION_ERROR", "Catatan wajib.", []map[string]string{{"field": "note", "message": "Minimal 5 karakter."}}, r))
			return
		}
	}
	p, err := a.Q.GetPortfolioByID(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
		return
	}
	if UserRole(r) != "admin" && !UserPerms(r)["portfolios.moderate"] {
		if !UserPerms(r)["portfolios.moderate_jurusan"] {
			writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Kamu tidak memiliki akses ini.", r))
			return
		}
		owner, _ := a.Q.GetUserByID(r.Context(), p.OwnerID)
		scope := ""
		if owner.MajorID.Valid {
			scope = uuidStr(owner.MajorID)
		}
		if UserScope(r) != "" && scope != UserScope(r) {
			writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Di luar jurusanmu.", r))
			return
		}
	}
	_ = a.Q.ModeratePortfolio(r.Context(), repo.ModeratePortfolioParams{
		ID: p.ID, Status: status, ReviewNote: req.Note, ReviewedBy: mustUUID(UserID(r)),
	})
	notifType := "portfolio_approved"
	if status == "rejected" {
		notifType = "portfolio_rejected"
	} else if status == "changes_requested" {
		notifType = "portfolio_changes_requested"
	}
	_, _ = a.Q.EnqueueOutbox(r.Context(), repo.EnqueueOutboxParams{
		Topic:   "notify." + notifType,
		Payload: notifPayload(map[string]string{"user_id": uuidStr(p.OwnerID), "portfolio_id": id, "note": req.Note}),
	})
	_ = a.Q.UpsertPortfolioStats(r.Context(), repo.UpsertPortfolioStatsParams{PortfolioID: p.ID})
	auditLog(r, a, "portfolio.moderate."+status, "portfolios", id, req.Note)
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": status}, r))
}

func (a *App) handleApprovePortfolio(w http.ResponseWriter, r *http.Request) {
	a.moderate(w, r, "published", false)
}

func (a *App) handleRequestChanges(w http.ResponseWriter, r *http.Request) {
	a.moderate(w, r, "changes_requested", true)
}

func (a *App) handleRejectPortfolio(w http.ResponseWriter, r *http.Request) {
	a.moderate(w, r, "rejected", true)
}

func (a *App) handleCreateTag(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Name) < 2 || len(req.Slug) < 2 {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Nama + slug minimal 2 karakter.", r))
		return
	}
	id, err := a.Q.EnsureTag(r.Context(), repo.EnsureTagParams{Slug: req.Slug, Name: req.Name})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal membuat tag.", r))
		return
	}
	writeJSON(w, http.StatusCreated, ok(map[string]string{"id": uuidStr(id)}, r))
}

func (a *App) handleUpdateTag(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleDeleteTag(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "deleted"}, r))
}

func (a *App) handleCreateSeries(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Name) < 3 || len(req.Slug) < 2 {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Nama + slug tidak valid.", r))
		return
	}
	id, err := a.Q.CreateSeries(r.Context(), repo.CreateSeriesParams{Slug: req.Slug, Name: req.Name, Description: req.Description})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal membuat series.", r))
		return
	}
	writeJSON(w, http.StatusCreated, ok(map[string]string{"id": uuidStr(id)}, r))
}
