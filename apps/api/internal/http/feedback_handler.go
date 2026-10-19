package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

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
		Category       string `json:"category"`
		Title          string `json:"title"`
		Body           string `json:"body"`
		Email          string `json:"email"`
		TurnstileToken string `json:"captcha_token"`
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
	// Guest wajib Turnstile (bila dikonfigurasi) agar tidak spam.
	if UserID(r) == "" && !verifyTurnstile(r.Context(), a.Cfg, req.TurnstileToken) {
		writeJSON(w, http.StatusTooManyRequests, fail("CAPTCHA_REQUIRED", "Verifikasi keamanan diperlukan.", r))
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
	resp := map[string]any{"id": uuidStr(id)}
	// Guest tidak punya sesi: beri token follow-up (?t=) agar bisa membuka
	// kembali thread-nya tanpa login. Token = HMAC(secret, thread_id).
	if uid == "" {
		resp["follow_up_token"] = feedbackFollowUpToken(a.Cfg.JWT.CSRFSecret(), uuidStr(id))
		resp["follow_up_url"] = "/feedback?t=" + feedbackFollowUpToken(a.Cfg.JWT.CSRFSecret(), uuidStr(id))
	}
	writeJSON(w, http.StatusCreated, ok(resp, r))
}

// feedbackFollowUpToken: token akses guest untuk satu thread, berlaku 6 bulan
// (format "<exp>.<hmac>"). Guest memakainya lewat ?t= untuk membuka thread.
func feedbackFollowUpToken(secret, threadID string) string {
	exp := time.Now().AddDate(0, 6, 0).Unix()
	return strconv.FormatInt(exp, 10) + "." + feedbackTokenMAC(secret, threadID, exp)
}

func feedbackTokenMAC(secret, threadID string, exp int64) string {
	mac := hmac.New(sha256.New, []byte(secret+"|feedback"))
	mac.Write([]byte(threadID + "|" + strconv.FormatInt(exp, 10)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func feedbackFollowUpValid(secret, threadID, token string) bool {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return false
	}
	exp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	return hmac.Equal([]byte(parts[1]), []byte(feedbackTokenMAC(secret, threadID, exp)))
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
	isTriageList := UserRole(r) == "admin" || UserPerms(r)["feedback.triage"]
	for _, t := range rows {
		// Guest & non-triage hanya melihat thread publik.
		if !isTriageList && !t.IsPublic {
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
	// Thread privat hanya untuk pelapor, moderator, atau pemegang token guest.
	canTriage := UserRole(r) == "admin" || UserPerms(r)["feedback.triage"]
	isReporter := t.ReporterID.Valid && uuidStr(t.ReporterID) == UserID(r)
	hasGuestToken := feedbackFollowUpValid(a.Cfg.JWT.CSRFSecret(), uuidStr(t.ID), r.URL.Query().Get("t"))
	if !t.IsPublic && !canTriage && !isReporter && !hasGuestToken {
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
		"sla_hours":    slaHours(t.Priority, t.Category),
		"age_hours":    time.Since(t.CreatedAt.Time).Hours(),
		"sla_breached": slaBreached(t),
	}, r))
}

// allowedFeedbackStatus: state machine sederhana sesuai PRD §11.1.
var allowedFeedbackStatus = map[string]bool{
	"open": true, "triaged": true, "in_progress": true,
	"resolved": true, "closed": true, "duplicate": true,
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func (a *App) handleReplyFeedback(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	t, err := a.Q.GetFeedbackThread(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Thread tidak ditemukan.", r))
		return
	}
	// Thread privat hanya untuk pelapor, moderator, atau pemegang token guest.
	canTriage := UserRole(r) == "admin" || UserPerms(r)["feedback.triage"]
	isReporter := t.ReporterID.Valid && uuidStr(t.ReporterID) == UserID(r)
	hasGuestToken := feedbackFollowUpValid(a.Cfg.JWT.CSRFSecret(), uuidStr(t.ID), r.URL.Query().Get("t"))
	if !t.IsPublic && !canTriage && !isReporter && !hasGuestToken {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Thread tidak ditemukan.", r))
		return
	}
	// Anti-abuse: balasan guest dibatasi 10/jam per thread+IP. Tanpa ini
	// thread publik bisa dispambalas tanpa perlu akun.
	if UserID(r) == "" && hasGuestToken {
		key := "feedback:reply:" + uuidStr(t.ID) + ":" + clientIP(r)
		if n, _ := a.Cache.IncrWithExpire(r.Context(), key, time.Hour); n > 10 {
			writeJSON(w, http.StatusTooManyRequests, fail("TOO_MANY_REQUESTS", "Terlalu banyak balasan. Coba lagi nanti.", r))
			return
		}
	}
	var req struct {
		Body       string `json:"body"`
		IsInternal bool   `json:"is_internal"`
		Status     string `json:"status"`
		Priority   string `json:"priority"`
		Token      string `json:"t"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Body) < 2 {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Isi balasan minimal 2 karakter.", r))
		return
	}
	// Otorisasi: pelapor boleh membalas thread-nya; perubahan status dan
	// catatan internal hanya untuk pemegang permission triage/admin.
	if !canTriage && !isReporter {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Bukan thread milikmu.", r))
		return
	}
	if req.Status != "" {
		if !canTriage {
			writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Hanya moderator yang bisa mengubah status.", r))
			return
		}
		if !allowedFeedbackStatus[req.Status] {
			writeJSON(w, http.StatusUnprocessableEntity, failDetails("VALIDATION_ERROR", "Status tidak valid.", []map[string]string{{"field": "status", "message": "open/triaged/in_progress/resolved/closed/duplicate."}}, r))
			return
		}
	}
	if req.Priority != "" && !canTriage {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Hanya moderator yang bisa mengubah prioritas.", r))
		return
	}

	var authorID = nilUUID()
	if uid := UserID(r); uid != "" {
		authorID = mustUUID(uid)
	}
	_, _ = a.Q.AddFeedbackMessage(r.Context(), repo.AddFeedbackMessageParams{
		ThreadID: t.ID, AuthorID: authorID, BodyMd: req.Body, IsInternal: req.IsInternal && canTriage,
	})
	priority := coalesce(req.Priority, t.Priority)
	_ = a.Q.UpdateFeedbackThread(r.Context(), repo.UpdateFeedbackThreadParams{
		ID: t.ID, Status: coalesce(req.Status, t.Status), Priority: priority,
		AssigneeID: t.AssigneeID, IsPublic: t.IsPublic,
	})
	if t.ReporterID.Valid && canTriage {
		_, _ = a.Q.EnqueueOutbox(r.Context(), repo.EnqueueOutboxParams{
			Topic:   "notify.feedback_updated",
			Payload: notifPayload(map[string]string{"user_id": uuidStr(t.ReporterID), "thread_id": id, "status": coalesce(req.Status, t.Status)}),
		})
	}
	auditLog(r, a, "feedback.reply", "feedback_threads", id, coalesce(req.Status, ""))
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
	_ = a.Q.RefreshFeedbackUpvotes(r.Context(), mustUUID(id))
	n, _ := a.Q.CountFeedbackVotes(r.Context(), mustUUID(id))
	writeJSON(w, http.StatusOK, ok(map[string]int64{"upvotes": n}, r))
}

func (a *App) handleCreateChangelog(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Version   string `json:"version"`
		Title     string `json:"title"`
		Summary   string `json:"summary"`
		Channel   string `json:"channel"`
		Audience  string `json:"audience"`
		PublishAt string `json:"publish_at"`
		Sections  []struct {
			Kind   string `json:"kind"`
			Title  string `json:"title"`
			BodyMd string `json:"body_md"`
		} `json:"sections"`
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
	status := "draft"
	var publishAt pgtype.Timestamptz
	if req.PublishAt != "" {
		t, perr := time.Parse(time.RFC3339, req.PublishAt)
		if perr != nil {
			writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "publish_at harus RFC3339.", r))
			return
		}
		publishAt = pgtype.Timestamptz{Time: t, Valid: true}
		if t.After(time.Now()) {
			status = "scheduled"
		}
	}
	id, err := a.Q.CreateChangelog(r.Context(), repo.CreateChangelogParams{
		Version: req.Version, Title: req.Title, Summary: req.Summary,
		Channel: req.Channel, Audience: req.Audience, Status: status,
		PublishAt: publishAt, CreatedBy: mustUUID(UserID(r)),
	})
	if err != nil {
		writeJSON(w, http.StatusConflict, fail("DUPLICATE_VERSION", "Version sudah dipakai.", r))
		return
	}
	for i, sec := range req.Sections {
		kind := sheetKind(sec.Kind)
		if kind == "" {
			continue
		}
		_ = a.Q.AddChangelogSection(r.Context(), repo.AddChangelogSectionParams{
			ChangelogID: id, Kind: kind, Title: sec.Title, BodyMd: sec.BodyMd, Sort: int32(i),
		})
	}
	auditLog(r, a, "changelog.create", "changelogs", uuidStr(id), req.Version)
	writeJSON(w, http.StatusCreated, ok(map[string]string{"id": uuidStr(id), "status": status}, r))
}

func sheetKind(k string) string {
	switch k {
	case "added", "improved", "fixed", "removed":
		return k
	}
	return ""
}

// handleChangelogDetail: detail + sections (publik untuk yang terbit).
func (a *App) handleChangelogDetail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, err := a.Q.GetChangelog(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Changelog tidak ditemukan.", r))
		return
	}
	isManager := UserRole(r) == "admin" || UserPerms(r)["changelog.manage"]
	visible := c.Status == "published" && (!c.PublishAt.Valid || !c.PublishAt.Time.After(time.Now()))
	if !visible && !isManager {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Changelog tidak ditemukan.", r))
		return
	}
	secs, _ := a.Q.ListChangelogSections(r.Context(), c.ID)
	out := []map[string]any{}
	for _, sc := range secs {
		out = append(out, map[string]any{"kind": sc.Kind, "title": sc.Title, "body_md": sc.BodyMd})
	}
	if out == nil {
		out = []map[string]any{}
	}
	reads, _ := a.Q.CountChangelogReads(r.Context(), c.ID)
	contribs := []map[string]any{}
	if rows, cerr := a.Q.ListChangelogContributors(r.Context(), c.ID); cerr == nil {
		for _, x := range rows {
			contribs = append(contribs, map[string]any{
				"user_id": uuidStr(x.ID), "username": x.Username, "name": x.Name, "role": x.Role,
			})
		}
	}
	writeJSON(w, http.StatusOK, ok(map[string]any{
		"id": uuidStr(c.ID), "version": c.Version, "title": c.Title, "summary": c.Summary,
		"channel": c.Channel, "audience": c.Audience, "status": c.Status, "reads": reads,
		"sections": out, "contributors": contribs,
	}, r))
}

// handleUnreadChangelogCount: rilis terbit yang belum dibaca user.
func (a *App) handleUnreadChangelogCount(w http.ResponseWriter, r *http.Request) {
	n, err := a.Q.CountUnreadChangelogs(r.Context(), mustUUID(UserID(r)))
	if err != nil {
		n = 0
	}
	writeJSON(w, http.StatusOK, ok(map[string]int64{"unread": n}, r))
}

// handleMarkAllChangelogsRead: tandai semua rilis terbit sudah dibaca.
func (a *App) handleMarkAllChangelogsRead(w http.ResponseWriter, r *http.Request) {
	_ = a.Q.MarkAllChangelogsRead(r.Context(), mustUUID(UserID(r)))
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

// handleArchiveChangelog: arsipkan rilis.
func (a *App) handleArchiveChangelog(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_ = a.Q.SetChangelogStatus(r.Context(), repo.SetChangelogStatusParams{ID: mustUUID(id), Status: "archived"})
	auditLog(r, a, "changelog.archive", "changelogs", id, "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "archived"}, r))
}

// handleUnpublishChangelog: tarik dari publik; >100 pembaca butuh admin.
func (a *App) handleUnpublishChangelog(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, _ := a.Q.CountChangelogReads(r.Context(), mustUUID(id))
	if n > 100 && UserRole(r) != "admin" {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Sudah dibaca 100+ user, perlu admin.", r))
		return
	}
	_ = a.Q.SetChangelogStatus(r.Context(), repo.SetChangelogStatusParams{ID: mustUUID(id), Status: "draft"})
	auditLog(r, a, "changelog.unpublish", "changelogs", id, "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "draft"}, r))
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
		Topic:   "notify.changelog_published",
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

// slaHours: target penyelesaian sesuai PRD §11.1.
func slaHours(priority, category string) int {
	if priority == "critical" || category == "bug" {
		return 24
	}
	if category == "idea" || category == "question" {
		return 14 * 24
	}
	return 72
}

func slaBreached(t repo.GetFeedbackThreadRow) bool {
	if t.Status == "resolved" || t.Status == "closed" {
		return false
	}
	limit := time.Duration(slaHours(t.Priority, t.Category)) * time.Hour
	return time.Since(t.CreatedAt.Time) > limit
}

// handleFeedbackTriage: prioritas / publikasi / spam (butuh triage).
func (a *App) handleFeedbackTriage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Priority        *string `json:"priority"`
		IsPublic        *bool   `json:"is_public"`
		IsSpam          *bool   `json:"is_spam"`
		LinkedChangelog string  `json:"linked_changelog_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	uid := mustUUID(id)
	if req.Priority != nil {
		switch *req.Priority {
		case "low", "medium", "high", "critical":
			_ = a.Q.UpdateFeedbackPriority(r.Context(), repo.UpdateFeedbackPriorityParams{
				ID: uid, Priority: *req.Priority, AssigneeID: mustUUID(UserID(r)),
			})
		default:
			writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Prioritas tidak valid.", r))
			return
		}
	}
	if req.IsPublic != nil {
		_ = a.Q.SetFeedbackPublic(r.Context(), repo.SetFeedbackPublicParams{ID: uid, IsPublic: *req.IsPublic})
	}
	if req.IsSpam != nil {
		_ = a.Q.SetFeedbackSpam(r.Context(), repo.SetFeedbackSpamParams{ID: uid, IsSpam: *req.IsSpam})
	}
	// Tautkan ke changelog (roadmap linkage) + beri tahu voters.
	if req.LinkedChangelog != "" {
		cl, cerr := a.Q.GetChangelog(r.Context(), mustUUID(req.LinkedChangelog))
		if cerr != nil {
			writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Changelog tidak ditemukan.", r))
			return
		}
		_ = a.Q.SetFeedbackLinkedChangelog(r.Context(), repo.SetFeedbackLinkedChangelogParams{
			ID: uid, LinkedChangelogID: mustUUID(req.LinkedChangelog),
		})
		if voters, verr := a.Q.ListFeedbackVoterIDs(r.Context(), uid); verr == nil {
			for _, v := range voters {
				_, _ = a.Q.EnqueueOutbox(r.Context(), repo.EnqueueOutboxParams{
					Topic: "notify.announcement",
					Payload: notifPayload(map[string]string{
						"user_id": uuidStr(v), "kind": "feedback_released",
						"message": "Masukanmu dirilis di Grafikarsa v" + cl.Version + ".",
					}),
				})
			}
		}
	}
	auditLog(r, a, "feedback.triage", "feedback_threads", id, "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

// handleLatestChangelog: rilis terbaru yang sudah terbit (untuk widget/landing).
func (a *App) handleLatestChangelog(w http.ResponseWriter, r *http.Request) {
	c, err := a.Q.LatestChangelog(r.Context())
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Belum ada changelog.", r))
		return
	}
	secs, _ := a.Q.ListChangelogSections(r.Context(), c.ID)
	out := []map[string]any{}
	for _, sc := range secs {
		out = append(out, map[string]any{"kind": sc.Kind, "title": sc.Title, "body_md": sc.BodyMd})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, ok(map[string]any{
		"id": uuidStr(c.ID), "version": c.Version, "title": c.Title, "summary": c.Summary,
		"channel": c.Channel, "sections": out,
	}, r))
}

// handleChangelogByVersion: halaman SEO /changelog/{version} (PRD §11.2).
// Draft hanya terlihat manager; versi tak ada balas 404.
func (a *App) handleChangelogByVersion(w http.ResponseWriter, r *http.Request) {
	ver := chi.URLParam(r, "version")
	c, err := a.Q.GetChangelogByVersion(r.Context(), ver)
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Changelog tidak ditemukan.", r))
		return
	}
	isManager := UserRole(r) == "admin" || UserPerms(r)["changelog.manage"]
	visible := c.Status == "published" && (!c.PublishAt.Valid || !c.PublishAt.Time.After(time.Now()))
	if !visible && !isManager {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Changelog tidak ditemukan.", r))
		return
	}
	secs, _ := a.Q.ListChangelogSections(r.Context(), c.ID)
	out := []map[string]any{}
	for _, sc := range secs {
		out = append(out, map[string]any{"kind": sc.Kind, "title": sc.Title, "body_md": sc.BodyMd})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, ok(map[string]any{
		"id": uuidStr(c.ID), "version": c.Version, "title": c.Title, "summary": c.Summary,
		"status": c.Status, "sections": out,
	}, r))
}

// handleListPublicMajors / handleListPublicClasses: referensi untuk filter FE,
// tidak butuh login (PRD §5) dan hanya menampilkan yang aktif.
func (a *App) handleListPublicMajors(w http.ResponseWriter, r *http.Request) {
	rows, err := a.Q.ListPublicMajors(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat jurusan.", r))
		return
	}
	out := []map[string]any{}
	for _, m := range rows {
		out = append(out, map[string]any{
			"id": uuidStr(m.ID), "code": m.Code, "name": m.Name, "require_series": m.RequireSeries,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, ok(out, r))
}

func (a *App) handleListPublicClasses(w http.ResponseWriter, r *http.Request) {
	rows, err := a.Q.ListPublicClasses(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat kelas.", r))
		return
	}
	out := []map[string]any{}
	for _, c := range rows {
		out = append(out, map[string]any{
			"id": uuidStr(c.ID), "name": c.Name, "grade": c.Grade, "group": c.ClassGroup,
			"major_code": c.MajorCode, "start_year": c.StartYear,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, ok(out, r))
}

// handleListChangelogContributors: daftar kontributor satu rilis.
func (a *App) handleListChangelogContributors(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := a.Q.ListChangelogContributors(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat kontributor.", r))
		return
	}
	out := []map[string]any{}
	for _, x := range rows {
		out = append(out, map[string]any{
			"user_id": uuidStr(x.ID), "username": x.Username, "name": x.Name, "role": x.Role,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, ok(out, r))
}

// handleAddChangelogContributor: tambah/ubah kontributor (dev/design/qa/...).
func (a *App) handleAddChangelogContributor(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.UserID == "" {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "user_id wajib.", r))
		return
	}
	role := strings.ToLower(strings.TrimSpace(req.Role))
	switch role {
	case "dev", "design", "qa", "pm", "docs":
	default:
		role = "dev"
	}
	if _, err := a.Q.GetUserByID(r.Context(), mustUUID(req.UserID)); err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "User tidak ditemukan.", r))
		return
	}
	_ = a.Q.AddChangelogContributor(r.Context(), repo.AddChangelogContributorParams{
		ChangelogID: mustUUID(id), UserID: mustUUID(req.UserID), Role: role,
	})
	auditLog(r, a, "changelog.contributor_add", "changelogs", id, role)
	writeJSON(w, http.StatusCreated, ok(map[string]string{"status": "ok"}, r))
}

// handleRemoveChangelogContributor: hapus kontributor.
func (a *App) handleRemoveChangelogContributor(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID := chi.URLParam(r, "userId")
	_ = a.Q.RemoveChangelogContributor(r.Context(), repo.RemoveChangelogContributorParams{
		ChangelogID: mustUUID(id), UserID: mustUUID(userID),
	})
	auditLog(r, a, "changelog.contributor_remove", "changelogs", id, "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}
