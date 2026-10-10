package http

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"grafikarsa/api/internal/repo"
)

func autoPriority(title, body string) string {
	lower := strings.ToLower(title + " " + body)
	for _, kw := range []string{"error", "gagal", "crash", "tidak bisa", "500", "keamanan", "xss"} {
		if strings.Contains(lower, kw) {
			return "high"
		}
	}
	return "medium"
}

func (a *App) handleCreateFeedback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Category string `json:"category"`
		Title    string `json:"title"`
		Body     string `json:"body"`
		Email    string `json:"email"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Category != "bug" && req.Category != "idea" && req.Category != "question" && req.Category != "praise" {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Kategori tidak valid.", r))
		return
	}
	if len(req.Title) < 5 || len(req.Body) < 10 {
		writeJSON(w, http.StatusUnprocessableEntity, failDetails("VALIDATION_ERROR", "Terlalu pendek.", []map[string]string{{"field": "title", "message": "Judul 5+, isi 10+."}}, r))
		return
	}
	// Rate-limit reporter: 3 thread/hari.
	uid := UserID(r)
	if uid != "" {
		if n, _ := a.Cache.IncrWithExpire(r.Context(), "feedback:day:"+uid, 24*3600*1000000000); n > 3 {
			writeJSON(w, http.StatusTooManyRequests, fail("TOO_MANY_REQUESTS", "Maksimal 3 laporan per hari.", r))
			return
		}
	}
	var reporterID = nilUUID()
	if uid != "" {
		reporterID = mustUUID(uid)
	}
	id, err := a.Q.CreateFeedbackThread(r.Context(), repo.CreateFeedbackThreadParams{
		ReporterID: reporterID, ReporterEmail: req.Email, Category: req.Category,
		Title: req.Title, BodyMd: req.Body, Priority: autoPriority(req.Title, req.Body),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal mengirim.", r))
		return
	}
	writeJSON(w, http.StatusCreated, ok(map[string]string{"id": uuidStr(id)}, r))
}

func (a *App) handleListFeedback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, offset, page := pageParams(r)
	rows, err := a.Q.ListFeedbackThreads(r.Context(), repo.ListFeedbackThreadsParams{
		Column1: q.Get("status"), Column2: q.Get("category"), Limit: limit, Offset: offset,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat.", r))
		return
	}
	out := []map[string]any{}
	for _, t := range rows {
		if !t.IsPublic && UserID(r) == "" {
			continue
		}
		out = append(out, map[string]any{
			"id": uuidStr(t.ID), "category": t.Category, "title": t.Title,
			"status": t.Status, "upvotes": t.UpvotesCached,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, int64(len(out))), r))
}

func (a *App) handleGetFeedback(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	t, err := a.Q.GetFeedbackThread(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Thread tidak ditemukan.", r))
		return
	}
	showInternal := UserRole(r) == "admin" || UserPerms(r)["feedback.triage"]
	msgs, _ := a.Q.ListFeedbackMessages(r.Context(), repo.ListFeedbackMessagesParams{ThreadID: t.ID, Column2: !showInternal})
	mout := []map[string]any{}
	for _, m := range msgs {
		mout = append(mout, map[string]any{"body": m.BodyMd, "is_internal": m.IsInternal, "created_at": m.CreatedAt.Time})
	}
	if mout == nil {
		mout = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, ok(map[string]any{
		"id": uuidStr(t.ID), "category": t.Category, "title": t.Title, "body": t.BodyMd,
		"status": t.Status, "priority": t.Priority, "messages": mout,
	}, r))
}

func (a *App) handleReplyFeedback(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	t, err := a.Q.GetFeedbackThread(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Thread tidak ditemukan.", r))
		return
	}
	var req struct {
		Body       string `json:"body"`
		IsInternal bool   `json:"is_internal"`
		Status     string `json:"status"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Body) < 2 {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Isi balasan minimal 2 karakter.", r))
		return
	}
	var authorID = nilUUID()
	if uid := UserID(r); uid != "" {
		authorID = mustUUID(uid)
	}
	_, _ = a.Q.AddFeedbackMessage(r.Context(), repo.AddFeedbackMessageParams{
		ThreadID: t.ID, AuthorID: authorID, BodyMd: req.Body, IsInternal: req.IsInternal && UserPerms(r)["feedback.triage"],
	})
	if req.Status != "" {
		_ = a.Q.UpdateFeedbackThread(r.Context(), repo.UpdateFeedbackThreadParams{
			ID: t.ID, Status: req.Status, Priority: t.Priority,
			AssigneeID: t.AssigneeID, IsPublic: t.IsPublic,
		})
		if t.ReporterID.Valid {
			_, _ = a.Q.EnqueueOutbox(r.Context(), repo.EnqueueOutboxParams{
				Topic:   "notify.feedback_updated",
				Payload: notifPayload(map[string]string{"user_id": uuidStr(t.ReporterID), "thread_id": id, "status": req.Status}),
			})
		}
	}
	auditLog(r, a, "feedback.reply", "feedback_threads", id, req.Status)
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleVoteFeedback(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid := UserID(r)
	if uid == "" {
		writeJSON(w, http.StatusUnauthorized, fail("UNAUTHORIZED", "Login untuk vote.", r))
		return
	}
	_ = a.Q.VoteFeedback(r.Context(), repo.VoteFeedbackParams{ThreadID: mustUUID(id), UserID: mustUUID(uid)})
	n, _ := a.Q.CountFeedbackVotes(r.Context(), mustUUID(id))
	writeJSON(w, http.StatusOK, ok(map[string]int64{"upvotes": n}, r))
}

func (a *App) handleCreateChangelog(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Version  string `json:"version"`
		Title    string `json:"title"`
		Summary  string `json:"summary"`
		Channel  string `json:"channel"`
		Audience string `json:"audience"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Version) < 3 || len(req.Title) < 5 {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Version + judul tidak valid.", r))
		return
	}
	if req.Channel == "" {
		req.Channel = "stable"
	}
	if req.Audience == "" {
		req.Audience = "all"
	}
	id, err := a.Q.CreateChangelog(r.Context(), repo.CreateChangelogParams{
		Version: req.Version, Title: req.Title, Summary: req.Summary,
		Channel: req.Channel, Audience: req.Audience, Status: "draft",
		CreatedBy: mustUUID(UserID(r)),
	})
	if err != nil {
		writeJSON(w, http.StatusConflict, fail("DUPLICATE_VERSION", "Version sudah dipakai.", r))
		return
	}
	auditLog(r, a, "changelog.create", "changelogs", uuidStr(id), req.Version)
	writeJSON(w, http.StatusCreated, ok(map[string]string{"id": uuidStr(id)}, r))
}

func (a *App) handleListChangelogs(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := pageParams(r)
	rows, err := a.Q.ListChangelogs(r.Context(), repo.ListChangelogsParams{Limit: limit, Offset: offset})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat.", r))
		return
	}
	out := []map[string]any{}
	for _, c := range rows {
		out = append(out, map[string]any{
			"id": uuidStr(c.ID), "version": c.Version, "title": c.Title,
			"status": c.Status, "audience": c.Audience,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, int64(len(out))), r))
}

func (a *App) handlePublishedChangelogs(w http.ResponseWriter, r *http.Request) {
	limit, offset, _ := pageParams(r)
	rows, err := a.Q.ListPublishedChangelogs(r.Context(), repo.ListPublishedChangelogsParams{Limit: limit, Offset: offset})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat.", r))
		return
	}
	out := []map[string]any{}
	for _, c := range rows {
		out = append(out, map[string]any{
			"id": uuidStr(c.ID), "version": c.Version, "title": c.Title, "summary": c.Summary,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, ok(out, r))
}

func (a *App) handlePublishChangelog(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, err := a.Q.GetChangelog(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Changelog tidak ditemukan.", r))
		return
	}
	n, _ := a.Q.CountChangelogReads(r.Context(), c.ID)
	if n > 100 && UserRole(r) != "admin" {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Sudah dibaca 100+ user, perlu admin.", r))
		return
	}
	_ = a.Q.SetChangelogStatus(r.Context(), repo.SetChangelogStatusParams{ID: c.ID, Status: "published"})
	_ = a.Q.MarkChangelogPublished(r.Context(), c.ID)
	_, _ = a.Q.EnqueueOutbox(r.Context(), repo.EnqueueOutboxParams{
		Topic:   "notify.announcement",
		Payload: notifPayload(map[string]string{"kind": "changelog_published", "changelog_id": id, "version": c.Version, "audience": c.Audience}),
	})
	auditLog(r, a, "changelog.publish", "changelogs", id, c.Version)
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "published"}, r))
}

func (a *App) handleReadChangelog(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if uid := UserID(r); uid != "" {
		_ = a.Q.MarkChangelogRead(r.Context(), repo.MarkChangelogReadParams{ChangelogID: mustUUID(id), UserID: mustUUID(uid)})
	}
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}
