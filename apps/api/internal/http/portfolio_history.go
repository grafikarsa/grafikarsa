package http

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"grafikarsa/api/internal/repo"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// auditLogTransition mencatat perubahan status sebuah entitas.
//
// PRD §8.1: "Semua transisi tercatat di portfolio_status_history (implisit
// via audit_logs + kolom reviewed_by/at/note)". Yang dipakai di v2.0 adalah
// audit_logs — jadi from/to harus benar-benar tersimpan, kalau tidak
// riwayatnya tidak bisa direkonstruksi.
//
// Bedanya dengan auditLog biasa: diff-nya menyimpan from/to, bukan cuma
// catatan bebas. Action tetap `portfolio.<status>` supaya filter berdasarkan
// action tetap bisa dipakai.
func auditLogTransition(r *http.Request, a *App, action, entity, entityID, from, to string) {
	var actor pgtype.UUID
	if uid := UserID(r); uid != "" {
		actor = mustUUID(uid)
	}
	db := toJSONBytes(map[string]string{"from": from, "to": to})
	_ = a.Q.InsertAuditLog(r.Context(), repo.InsertAuditLogParams{
		ActorID: actor, Action: action, Entity: entity, EntityID: entityID,
		Diff: db, Ip: toPgInet(clientIP(r)),
	})
}

// StatusHistoryEntry adalah satu baris riwayat status karya.
//
// Tag JSON pakai snake_case sesuai konvensi envelope API (media_id,
// object_key, dst) — bukan gaya Go.
type StatusHistoryEntry struct {
	Action     string    `json:"action"`
	From       string    `json:"from"`
	To         string    `json:"to"`
	ReviewNote string    `json:"review_note"`
	ActorID    string    `json:"actor_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// portfolioStatusHistory membaca riwayat transisi status sebuah karya dari
// audit_logs.
//
// Karena transisi dicatat sebagai action `portfolio.<status>` dengan diff
// berisi from/to, riwayatnya bisa dibaca ulang tanpa tabel terpisah. Dipakai
// halaman admin untuk melihat siapa yang mengubah status karya dan kapan.
//
// $1 = id portfolio, $2 = jumlah baris maksimum.
func (a *App) portfolioStatusHistory(ctx context.Context, portfolioID string, limit int32) ([]StatusHistoryEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := a.Q.ListPortfolioStatusHistory(ctx, repo.ListPortfolioStatusHistoryParams{
		EntityID: portfolioID,
		Limit:    limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]StatusHistoryEntry, 0, len(rows))
	for _, row := range rows {
		entry := StatusHistoryEntry{
			Action:     row.Action,
			From:       row.FromStatus,
			To:         row.ToStatus,
			ReviewNote: row.Note,
			CreatedAt:  row.CreatedAt.Time,
		}
		if row.ActorID.Valid {
			entry.ActorID = uuidStr(row.ActorID)
		}
		out = append(out, entry)
	}
	return out, nil
}

// handlePortfolioStatusHistory menampilkan riwayat transisi status sebuah
// karya.
//
// PRD §8.1: "Semua transisi tercatat di portfolio_status_history (implisit
// via audit_logs + kolom reviewed_by/at/note)". Endpoint ini adalah cara
// membacanya.
//
// GET /v1/admin/portfolios/{id}/status-history
func (a *App) handlePortfolioStatusHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	// Portofolio harus benar-benar ada — jangan bocorkan riwayat entitas
	// yang tidak ada.
	if _, err := a.Q.GetPortfolioByID(r.Context(), mustUUID(id)); err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
		return
	}
	limit := int32(50)
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 && n <= 200 {
			limit = int32(n)
		}
	}
	entries, err := a.portfolioStatusHistory(r.Context(), id, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat riwayat status.", r))
		return
	}
	writeJSON(w, http.StatusOK, ok(entries, r))
}
