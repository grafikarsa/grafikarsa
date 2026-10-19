package http

import (
	"net/http"

	"grafikarsa/api/internal/repo"
)

// handleDashboardStats: angka ringkas dashboard admin. Scope mengikuti aturan
// yang sama dengan daftar user/portfolio: admin global melihat semua,
// admin_jurusan hanya jurusannya (fail-closed).
func (a *App) handleDashboardStats(w http.ResponseWriter, r *http.Request) {
	scope := scopeFilter(r, "dashboard.view_global", "dashboard.view_jurusan")
	if scope == "__none__" {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Kamu tidak memiliki akses ini.", r))
		return
	}
	if scope == "" && UserRole(r) == "admin_jurusan" {
		// admin_jurusan tanpa scope tidak boleh melihat data global.
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Scope jurusan belum diset.", r))
		return
	}
	s, err := a.Q.DashboardStats(r.Context(), scope)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat statistik.", r))
		return
	}
	writeJSON(w, http.StatusOK, ok(map[string]any{
		"scope": scope,
		"users": map[string]int64{
			"total": s.UsersTotal, "active": s.UsersActive, "alumni": s.UsersAlumni,
		},
		"portfolios": map[string]int64{
			"total": s.PortfoliosTotal, "published": s.PortfoliosPublished, "pending": s.PortfoliosPending,
		},
		"feedback_open":      s.FeedbackOpen,
		"jobs_queued":        s.JobsQueued,
		"jobs_failed":        s.JobsFailed,
		"blocks_quarantined": s.BlocksQuarantined,
	}, r))
}

// handleListJobs: status outbox (PRD: GET /admin/jobs).
func (a *App) handleListJobs(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := pageParams(r)
	status := r.URL.Query().Get("status")
	switch status {
	case "", "pending", "processing", "sent", "failed":
	default:
		status = ""
	}
	rows, err := a.Q.ListOutboxJobs(r.Context(), repo.ListOutboxJobsParams{
		Column1: status, Limit: limit, Offset: offset,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat jobs.", r))
		return
	}
	out := []map[string]any{}
	for _, j := range rows {
		out = append(out, map[string]any{
			"id": j.ID, "topic": j.Topic, "status": j.Status, "attempts": j.Attempts,
			"next_retry_at": j.NextRetryAt, "created_at": j.CreatedAt,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	total, _ := a.Q.CountOutboxJobs(r.Context(), status)
	writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, total), r))
}
