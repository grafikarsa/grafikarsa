package http

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"grafikarsa/api/internal/repo"
)

func (a *App) handleAdminListPortfolios(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := pageParams(r)
	status := r.URL.Query().Get("status")
	// Row-level scope: admin_jurusan hanya melihat portfolio jurusannya.
	scope := scopeFilter(r, "portfolios.view_all", "portfolios.moderate_jurusan")
	rows, err := a.Q.ListPortfoliosForAdmin(r.Context(), repo.ListPortfoliosForAdminParams{
		Column1: status, Column2: scope, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat.", r))
		return
	}
	out := []map[string]any{}
	for _, p := range rows {
		out = append(out, map[string]any{
			"id": uuidStr(p.ID), "title": p.Title, "slug": p.Slug, "status": p.Status,
			"username": p.Username, "owner_name": p.OwnerName,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	total, _ := a.Q.CountPortfoliosForAdmin(r.Context(), repo.CountPortfoliosForAdminParams{Column1: status, Column2: scope})
	writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, total), r))
}

func (a *App) handleAdminPendingPortfolios(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := pageParams(r)
	rows, err := a.Q.ListPendingPortfoliosScoped(r.Context(), repo.ListPendingPortfoliosScopedParams{
		Column1: scopeFilter(r, "portfolios.moderate", "portfolios.moderate_jurusan"),
		Limit:   limit, Offset: offset,
	})
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
	// Row-level scope: draft/pending jurusan lain tidak boleh dibaca
	// (balas 404 agar keberadaannya tidak bocor).
	if UserRole(r) != "admin" {
		owner, oerr := a.Q.GetUserByID(r.Context(), p.OwnerID)
		if oerr != nil || !owner.MajorID.Valid || UserScope(r) == "" || uuidStr(owner.MajorID) != UserScope(r) {
			writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
			return
		}
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
		if UserScope(r) == "" || scope == "" || scope != UserScope(r) {
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
	_ = a.Q.EnsurePortfolioStats(r.Context(), p.ID)
	auditLog(r, a, "portfolio.moderate."+status, "portfolios", id, req.Note)
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": status}, r))
}

func (a *App) handleApprovePortfolio(w http.ResponseWriter, r *http.Request) {
	a.moderate(w, r, "published", false)
}

func (a *App) handleBulkApprove(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.IDs) == 0 || len(req.IDs) > 20 {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Maksimal 20 ID.", r))
		return
	}
	okCount := 0
	for _, id := range req.IDs {
		p, err := a.Q.GetPortfolioByID(r.Context(), mustUUID(id))
		if err != nil || p.Status != "pending_review" {
			continue
		}
		if !canModerate(r, a, p) {
			continue
		}
		_ = a.Q.ModeratePortfolio(r.Context(), repo.ModeratePortfolioParams{
			ID: p.ID, Status: "published", ReviewedBy: mustUUID(UserID(r)),
		})
		_, _ = a.Q.EnqueueOutbox(r.Context(), repo.EnqueueOutboxParams{
			Topic:   "notify.portfolio_approved",
			Payload: notifPayload(map[string]string{"user_id": uuidStr(p.OwnerID), "portfolio_id": id}),
		})
		_ = a.Q.EnsurePortfolioStats(r.Context(), p.ID)
		auditLog(r, a, "portfolio.moderate.published", "portfolios", id, "bulk")
		okCount++
	}
	writeJSON(w, http.StatusOK, ok(map[string]int{"approved": okCount}, r))
}

func (a *App) handleRestorePortfolio(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_ = a.Q.RestorePortfolio(r.Context(), mustUUID(id))
	auditLog(r, a, "portfolio.restore", "portfolios", id, "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "restored"}, r))
}

func canModerate(r *http.Request, a *App, p repo.GetPortfolioByIDRow) bool {
	if UserRole(r) == "admin" || UserPerms(r)["portfolios.moderate"] {
		return true
	}
	if !UserPerms(r)["portfolios.moderate_jurusan"] {
		return false
	}
	owner, _ := a.Q.GetUserByID(r.Context(), p.OwnerID)
	scope := ""
	if owner.MajorID.Valid {
		scope = uuidStr(owner.MajorID)
	}
	return UserScope(r) != "" && scope != "" && scope == UserScope(r)
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

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func blockTypeAllowed(t string) string {
	switch t {
	case "text", "image", "table", "youtube", "button", "embed", "figma", "canva", "ppt", "pdf", "doc", "heading", "gallery", "code":
		return t
	}
	return ""
}

func (a *App) handleUpdateTag(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(strings.TrimSpace(req.Name)) < 2 {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Nama tag minimal 2 karakter.", r))
		return
	}
	_ = a.Q.UpdateTag(r.Context(), repo.UpdateTagParams{ID: mustUUID(id), Name: req.Name, Column3: nullIfEmpty(req.Slug)})
	auditLog(r, a, "tags.update", "tags", id, req.Name)
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleDeleteTag(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// Tag yang masih dipakai portfolio tidak dihapus (integritas katalog).
	deleted, err := a.Q.DeleteTagIfUnused(r.Context(), mustUUID(id))
	if err != nil || !deleted.Valid {
		writeJSON(w, http.StatusConflict, fail("TAG_IN_USE", "Tag masih dipakai portfolio.", r))
		return
	}
	auditLog(r, a, "tags.delete", "tags", id, "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "deleted"}, r))
}

func (a *App) handleCreateSeries(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Description string `json:"description"`
		Blocks      []struct {
			BlockType   string `json:"block_type"`
			Instruction string `json:"instruction"`
		} `json:"blocks"`
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
	// Template block wajib tersimpan, kalau tidak series tidak berguna.
	for i, b := range req.Blocks {
		bt := blockTypeAllowed(b.BlockType)
		if bt == "" {
			continue
		}
		_ = a.Q.AddSeriesBlock(r.Context(), repo.AddSeriesBlockParams{
			SeriesID: id, BlockType: bt, Instruction: b.Instruction, Position: int32(i),
		})
	}
	auditLog(r, a, "series.create", "series", uuidStr(id), req.Slug)
	writeJSON(w, http.StatusCreated, ok(map[string]string{"id": uuidStr(id)}, r))
}
