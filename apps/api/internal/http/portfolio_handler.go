package http

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"grafikarsa/api/internal/repo"
)

var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func slugify(title string) string {
	s := strings.ToLower(title)
	var b strings.Builder
	prevDash := true
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "portfolio-" + uuid.NewString()[:8]
	}
	if len(out) > 80 {
		out = out[:80]
	}
	return strings.Trim(out, "-")
}

type createPortfolioRequest struct {
	Title    string   `json:"title" validate:"required,min=3,max=200"`
	TagIDs   []string `json:"tag_ids"`
	SeriesID *string  `json:"series_id"`
}

func (a *App) handleCreatePortfolio(w http.ResponseWriter, r *http.Request) {
	var req createPortfolioRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := validate.Struct(req); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Judul wajib 3-200 karakter.", r))
		return
	}
	uid := UserID(r)
	slug := slugify(req.Title)
	var seriesID = nilUUID()
	if req.SeriesID != nil && *req.SeriesID != "" {
		seriesID = mustUUID(*req.SeriesID)
	}
	id, err := a.Q.CreatePortfolio(r.Context(), repo.CreatePortfolioParams{
		OwnerID: mustUUID(uid), Title: req.Title, Slug: slug, SeriesID: seriesID,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal membuat portfolio.", r))
		return
	}
	for _, tid := range req.TagIDs {
		_ = a.Q.AddPortfolioTag(r.Context(), repo.AddPortfolioTagParams{PortfolioID: id, TagID: mustUUID(tid)})
	}
	auditLog(r, a, "portfolio.create", "portfolios", uuidStr(id), "")
	writeJSON(w, http.StatusCreated, ok(map[string]any{"id": uuidStr(id), "slug": slug}, r))
}

func (a *App) canViewPortfolio(r *http.Request, p repo.GetPortfolioByIDRow, ownerScope string) bool {
	if p.Status == "published" {
		return true
	}
	uid := UserID(r)
	if uid != "" && (uid == uuidStr(p.OwnerID) || UserRole(r) == "admin") {
		return true
	}
	if uid != "" && UserPerms(r)["portfolios.moderate"] {
		return true
	}
	if uid != "" && UserPerms(r)["portfolios.moderate_jurusan"] && ownerScope != "" && ownerScope == UserScope(r) {
		return true
	}
	return false
}

func (a *App) handleGetPortfolio(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := a.Q.GetPortfolioByID(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
		return
	}
	owner, _ := a.Q.GetUserByID(r.Context(), p.OwnerID)
	ownerScope := ""
	if owner.MajorID.Valid {
		ownerScope = uuidStr(owner.MajorID)
	}
	if !a.canViewPortfolio(r, p, ownerScope) {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
		return
	}
	writeJSON(w, http.StatusOK, ok(portfolioDetail(r, a, p), r))
}

func (a *App) handleGetPortfolioBySlug(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	slug := r.URL.Query().Get("slug")
	if username == "" || slug == "" {
		writeJSON(w, http.StatusBadRequest, fail("VALIDATION_ERROR", "username + slug wajib.", r))
		return
	}
	owner, err := a.Q.GetUserByUsername(r.Context(), username)
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
		return
	}
	p, err := a.Q.GetPortfolioBySlug(r.Context(), repo.GetPortfolioBySlugParams{OwnerID: owner.ID, Slug: slug})
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
		return
	}
	byID := repo.GetPortfolioByIDRow{
		ID: p.ID, OwnerID: p.OwnerID, Title: p.Title, Slug: p.Slug, Status: p.Status,
		ThumbnailMediaID: p.ThumbnailMediaID, SeriesID: p.SeriesID, DraftPayload: p.DraftPayload,
		PublishedAt: p.PublishedAt, SubmittedAt: p.SubmittedAt, ReviewedBy: p.ReviewedBy,
		ReviewedAt: p.ReviewedAt, ReviewNote: p.ReviewNote, CreatedAt: p.CreatedAt,
	}
	ownerScope := ""
	if owner.MajorID.Valid {
		ownerScope = uuidStr(owner.MajorID)
	}
	if !a.canViewPortfolio(r, byID, ownerScope) {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
		return
	}
	if p.Status == "published" {
		// View dihitung di HyperLogLog (Dragonfly) lalu di-flush berkala oleh
		// worker: tidak ada INSERT per request, unik per pengunjung/hari.
		viewer := UserID(r)
		if viewer == "" {
			viewer = r.Header.Get("X-Session-ID")
		}
		if viewer == "" {
			viewer = "anon-" + uuid.NewString()
		}
		a.recordViewHLL(r.Context(), uuidStr(p.ID), viewer)
	}
	writeJSON(w, http.StatusOK, ok(portfolioDetail(r, a, byID), r))
}

func portfolioDetail(r *http.Request, a *App, p repo.GetPortfolioByIDRow) map[string]any {
	ctx := r.Context()
	blocks, _ := a.Q.ListBlocks(ctx, p.ID)
	tags, _ := a.Q.ListPortfolioTags(ctx, p.ID)
	likes, _ := a.Q.CountLikes(ctx, p.ID)
	comments, _ := a.Q.CountComments(ctx, p.ID)
	isLiked := false
	if uid := UserID(r); uid != "" {
		n, _ := a.Q.IsLiked(ctx, repo.IsLikedParams{UserID: mustUUID(uid), PortfolioID: p.ID})
		isLiked = n > 0
	}
	blockOut := []map[string]any{}
	for _, b := range blocks {
		blockOut = append(blockOut, map[string]any{
			"id": b.ID, "block_type": b.BlockType, "position": b.Position,
			"payload": b.Payload, "is_from_series": b.IsFromSeries,
		})
	}
	if blockOut == nil {
		blockOut = []map[string]any{}
	}
	tagOut := []map[string]any{}
	for _, t := range tags {
		tagOut = append(tagOut, map[string]any{"id": uuidStr(t.ID), "slug": t.Slug, "name": t.Name})
	}
	if tagOut == nil {
		tagOut = []map[string]any{}
	}
	return map[string]any{
		"id": uuidStr(p.ID), "title": p.Title, "slug": p.Slug, "status": p.Status,
		"thumbnail_media_id": nullableUUID(p.ThumbnailMediaID),
		"series_id":          nullableUUID(p.SeriesID),
		"published_at":       p.PublishedAt.Time,
		"submitted_at":       p.SubmittedAt.Time,
		"review_note":        p.ReviewNote,
		"blocks":             blockOut, "tags": tagOut,
		"like_count": likes, "comment_count": comments, "is_liked": isLiked,
	}
}

func (a *App) handleUpdatePortfolio(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := a.Q.GetPortfolioByID(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
		return
	}
	if !isOwnerOrAdmin(r, p.OwnerID) {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Bukan milikmu.", r))
		return
	}
	if p.Status == "pending_review" {
		writeJSON(w, http.StatusUnprocessableEntity, fail("INVALID_STATUS_TRANSITION", "Portfolio sedang direview, batalkan dulu.", r))
		return
	}
	var req struct {
		Title        *string        `json:"title"`
		SeriesID     *string        `json:"series_id"`
		TagIDs       []string       `json:"tag_ids"`
		Thumbnail    *string        `json:"thumbnail_media_id"`
		DraftPayload map[string]any `json:"draft_payload"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Title != nil {
		if len(*req.Title) < 3 || len(*req.Title) > 200 {
			writeJSON(w, http.StatusUnprocessableEntity, failDetails("VALIDATION_ERROR", "Judul tidak valid.", []map[string]string{{"field": "title", "message": "3-200 karakter."}}, r))
			return
		}
		_ = a.Q.UpdatePortfolioTitle(r.Context(), repo.UpdatePortfolioTitleParams{ID: p.ID, Title: *req.Title})
	}
	if req.Thumbnail != nil && *req.Thumbnail != "" {
		thumbID := mustUUID(*req.Thumbnail)
		// Wajib media milik sendiri: mencegah menempel media orang lain lalu
		// mempublikasikannya lewat portfolio sendiri (IDOR).
		if !a.ownsMedia(r.Context(), thumbID, UserID(r)) {
			writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Media bukan milikmu.", r))
			return
		}
		_ = a.Q.UpdatePortfolioThumbnail(r.Context(), repo.UpdatePortfolioThumbnailParams{ID: p.ID, ThumbnailMediaID: thumbID})
		// Media dipakai -> tandai confirmed agar tidak dibersihkan job pending.
		_ = a.Q.ConfirmMedia(r.Context(), thumbID)
	}
	if req.SeriesID != nil {
		sid := nilUUID()
		if *req.SeriesID != "" {
			sid = mustUUID(*req.SeriesID)
		}
		_ = a.Q.UpdatePortfolioSeries(r.Context(), repo.UpdatePortfolioSeriesParams{ID: p.ID, SeriesID: sid})
	}
	if req.TagIDs != nil {
		_ = a.Q.DeletePortfolioTags(r.Context(), p.ID)
		for _, tid := range req.TagIDs {
			_ = a.Q.AddPortfolioTag(r.Context(), repo.AddPortfolioTagParams{PortfolioID: p.ID, TagID: mustUUID(tid)})
		}
	}
	// Autosave draft editor (maks 64KB, hanya untuk draft milik sendiri).
	if req.DraftPayload != nil {
		raw, merr := json.Marshal(req.DraftPayload)
		if merr != nil || len(raw) > 64*1024 {
			writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Draft terlalu besar (maks 64KB).", r))
			return
		}
		title := ""
		if req.Title != nil {
			title = *req.Title
		}
		_ = a.Q.UpdatePortfolioDraft(r.Context(), repo.UpdatePortfolioDraftParams{
			ID: p.ID, Column2: toPgText(title), DraftPayload: raw,
		})
	}
	auditLog(r, a, "portfolio.update", "portfolios", id, "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleDeletePortfolio(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := a.Q.GetPortfolioByID(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
		return
	}
	if !isOwnerOrAdmin(r, p.OwnerID) {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Bukan milikmu.", r))
		return
	}
	_ = a.Q.SoftDeletePortfolio(r.Context(), p.ID)
	auditLog(r, a, "portfolio.delete", "portfolios", id, "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "deleted"}, r))
}

func (a *App) handleSubmitPortfolio(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := a.Q.GetPortfolioByID(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
		return
	}
	if !isOwnerOrAdmin(r, p.OwnerID) {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Bukan milikmu.", r))
		return
	}
	if p.Status != "draft" && p.Status != "rejected" && p.Status != "changes_requested" {
		writeJSON(w, http.StatusUnprocessableEntity, fail("INVALID_STATUS_TRANSITION", "Hanya draft/rejected/changes yang bisa disubmit.", r))
		return
	}
	details := []map[string]string{}
	if !p.ThumbnailMediaID.Valid {
		details = append(details, map[string]string{"field": "thumbnail_media_id", "message": "Thumbnail wajib diisi."})
	}
	if len(p.Title) < 10 {
		details = append(details, map[string]string{"field": "title", "message": "Judul minimal 10 karakter."})
	}
	nb, _ := a.Q.CountPortfolioBlocks(r.Context(), p.ID)
	if nb == 0 {
		details = append(details, map[string]string{"field": "blocks", "message": "Minimal 1 block."})
	}
	nt, _ := a.Q.CountPortfolioTags(r.Context(), p.ID)
	if nt == 0 {
		details = append(details, map[string]string{"field": "tag_ids", "message": "Minimal 1 tag."})
	}
	if p.SeriesID.Valid == false {
		if owner, err := a.Q.GetUserByID(r.Context(), p.OwnerID); err == nil && owner.MajorID.Valid {
			if req, err := a.Q.GetMajorRequireSeries(r.Context(), owner.MajorID); err == nil && req {
				details = append(details, map[string]string{"field": "series_id", "message": "Jurusanmu mewajibkan series."})
			}
		}
	}
	if len(details) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, failDetails("INCOMPLETE_PORTFOLIO", "Portfolio belum lengkap.", details, r))
		return
	}
	snap := buildSnapshot(r, a, p)
	_ = a.Q.SubmitPortfolio(r.Context(), repo.SubmitPortfolioParams{ID: p.ID, SubmittedSnapshot: snap})
	_, _ = a.Q.EnqueueOutbox(r.Context(), repo.EnqueueOutboxParams{Topic: "portfolio.submitted", Payload: []byte(`{"portfolio_id":"` + id + `"}`)})
	auditLog(r, a, "portfolio.submit", "portfolios", id, "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "pending_review"}, r))
}

func (a *App) handleArchivePortfolio(w http.ResponseWriter, r *http.Request) {
	a.transitionOwn(w, r, "archived", "portfolio.archive")
}

func (a *App) handleUnarchivePortfolio(w http.ResponseWriter, r *http.Request) {
	a.transitionOwn(w, r, "draft", "portfolio.unarchive")
}

func (a *App) handleCancelPortfolio(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := a.Q.GetPortfolioByID(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
		return
	}
	if !isOwnerOrAdmin(r, p.OwnerID) || p.Status != "pending_review" {
		writeJSON(w, http.StatusUnprocessableEntity, fail("INVALID_STATUS_TRANSITION", "Hanya pending_review yang bisa dibatalkan.", r))
		return
	}
	_ = a.Q.CancelSubmit(r.Context(), p.ID)
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "draft"}, r))
}

func (a *App) transitionOwn(w http.ResponseWriter, r *http.Request, to, action string) {
	id := chi.URLParam(r, "id")
	p, err := a.Q.GetPortfolioByID(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
		return
	}
	if !isOwnerOrAdmin(r, p.OwnerID) {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Bukan milikmu.", r))
		return
	}
	switch to {
	case "archived":
		_ = a.Q.ArchivePortfolio(r.Context(), p.ID)
	default:
		_ = a.Q.UnarchivePortfolio(r.Context(), p.ID)
	}
	auditLog(r, a, action, "portfolios", id, "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": to}, r))
}

func (a *App) handleMyPortfolios(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := pageParams(r)
	rows, err := a.Q.ListMyPortfolios(r.Context(), repo.ListMyPortfoliosParams{OwnerID: mustUUID(UserID(r)), Limit: limit, Offset: offset})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat.", r))
		return
	}
	out := []map[string]any{}
	for _, p := range rows {
		out = append(out, map[string]any{"id": uuidStr(p.ID), "title": p.Title, "slug": p.Slug, "status": p.Status})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, int64(len(out))), r))
}

func (a *App) handleListPublished(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := pageParams(r)
	q := r.URL.Query()
	search := q.Get("search")
	tagID := q.Get("tag_id")
	majorID := q.Get("major_id")
	var rows []repo.FilterPublishedPortfoliosRow
	var err error
	var total int64
	switch {
	case search != "":
		srows, e := a.Q.FilterBySearch(r.Context(), repo.FilterBySearchParams{Column1: toPgText(search), Limit: limit, Offset: offset})
		err = e
		for _, t := range srows {
			rows = append(rows, repo.FilterPublishedPortfoliosRow(t))
		}
		total = int64(len(rows))
	case tagID != "":
		trows, e := a.Q.FilterByTag(r.Context(), repo.FilterByTagParams{TagID: mustUUID(tagID), Limit: limit, Offset: offset})
		err = e
		for _, t := range trows {
			rows = append(rows, repo.FilterPublishedPortfoliosRow(t))
		}
		total = int64(len(rows))
	case majorID != "":
		mrows, e := a.Q.FilterByMajor(r.Context(), repo.FilterByMajorParams{MajorID: mustUUID(majorID), Limit: limit, Offset: offset})
		err = e
		for _, t := range mrows {
			rows = append(rows, repo.FilterPublishedPortfoliosRow{
				ID: t.ID, OwnerID: t.OwnerID, Title: t.Title, Slug: t.Slug,
				ThumbnailMediaID: t.ThumbnailMediaID, PublishedAt: t.PublishedAt,
				Username: t.Username, OwnerName: t.OwnerName,
			})
		}
		total = int64(len(rows))
	default:
		rows, err = a.Q.FilterPublishedPortfolios(r.Context(), repo.FilterPublishedPortfoliosParams{
			Column1: search, Limit: limit, Offset: offset,
		})
		total, _ = a.Q.CountFilteredPortfolios(r.Context(), search)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat.", r))
		return
	}
	out := []map[string]any{}
	for _, p := range rows {
		out = append(out, map[string]any{
			"id": uuidStr(p.ID), "title": p.Title, "slug": p.Slug,
			"username": p.Username, "owner_name": p.OwnerName,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, total), r))
}
