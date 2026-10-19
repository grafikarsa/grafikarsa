package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/microcosm-cc/bluemonday"
	"grafikarsa/api/internal/repo"
)

var sanitizer = bluemonday.UGCPolicy()

func (a *App) handleFollow(w http.ResponseWriter, r *http.Request) {
	target := chi.URLParam(r, "username")
	uid := UserID(r)
	u, err := a.Q.GetUserByUsername(r.Context(), target)
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "User tidak ditemukan.", r))
		return
	}
	if uuidStr(u.ID) == uid {
		writeJSON(w, http.StatusUnprocessableEntity, fail("CANNOT_FOLLOW_SELF", "Tidak bisa follow diri sendiri.", r))
		return
	}
	if u.RolePrimary == "admin" {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Tidak bisa follow admin.", r))
		return
	}
	// Anti-abuse: maksimal 60 follow/jam per akun.
	if n, _ := a.Cache.IncrWithExpire(r.Context(), "follow:hour:"+uid, time.Hour); n > 60 {
		writeJSON(w, http.StatusTooManyRequests, fail("TOO_MANY_REQUESTS", "Terlalu banyak mengikuti dalam satu jam.", r))
		return
	}
	_ = a.Q.FollowUser(r.Context(), repo.FollowUserParams{FollowerID: mustUUID(uid), FollowingID: u.ID})
	_, _ = a.Q.EnqueueOutbox(r.Context(), repo.EnqueueOutboxParams{
		Topic:   "notify.new_follower",
		Payload: []byte(`{"user_id":"` + uuidStr(u.ID) + `","actor_id":"` + uid + `"}`),
	})
	auditLog(r, a, "social.follow", "users", uuidStr(u.ID), "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "following"}, r))
}

func (a *App) handleUnfollow(w http.ResponseWriter, r *http.Request) {
	target := chi.URLParam(r, "username")
	uid := UserID(r)
	u, err := a.Q.GetUserByUsername(r.Context(), target)
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "User tidak ditemukan.", r))
		return
	}
	_ = a.Q.UnfollowUser(r.Context(), repo.UnfollowUserParams{FollowerID: mustUUID(uid), FollowingID: u.ID})
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "unfollowed"}, r))
}

func (a *App) handleLike(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid := UserID(r)
	if UserRole(r) == "admin" {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Admin tidak bisa like.", r))
		return
	}
	p, err := a.Q.GetPortfolioByID(r.Context(), mustUUID(id))
	if err != nil || p.Status != "published" {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
		return
	}
	// Anti-gaming: like massal (>20/jam) tidak dihitung dan dicatat untuk audit,
	// supaya leaderboard tidak bisa dinaikkan dengan spam like.
	if n, cerr := a.Q.CountLikesByUserSince(r.Context(), mustUUID(uid)); cerr == nil && n >= 20 {
		auditLog(r, a, "like.rate_limited", "portfolios", id, ">20/jam")
		writeJSON(w, http.StatusTooManyRequests, fail("LIKE_RATE_LIMITED", "Terlalu banyak suka dalam satu jam. Coba lagi nanti.", r))
		return
	}
	_ = a.Q.LikePortfolio(r.Context(), repo.LikePortfolioParams{UserID: mustUUID(uid), PortfolioID: p.ID})
	trackLikeInterest(r.Context(), a, uid, p.ID)
	if uuidStr(p.OwnerID) != uid {
		_, _ = a.Q.EnqueueOutbox(r.Context(), repo.EnqueueOutboxParams{
			Topic:   "notify.portfolio_liked",
			Payload: []byte(`{"user_id":"` + uuidStr(p.OwnerID) + `","actor_id":"` + uid + `","portfolio_id":"` + id + `"}`),
		})
	}
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "liked"}, r))
}

func (a *App) handleUnlike(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_ = a.Q.UnlikePortfolio(r.Context(), repo.UnlikePortfolioParams{UserID: mustUUID(UserID(r)), PortfolioID: mustUUID(id)})
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "unliked"}, r))
}

func (a *App) handleCreateComment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid := UserID(r)
	p, err := a.Q.GetPortfolioByID(r.Context(), mustUUID(id))
	if err != nil || p.Status != "published" {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
		return
	}
	var req struct {
		ParentID *string `json:"parent_id"`
		Content  string  `json:"content"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Content) < 1 || len(req.Content) > 2000 {
		writeJSON(w, http.StatusUnprocessableEntity, failDetails("VALIDATION_ERROR", "Komentar tidak valid.", []map[string]string{{"field": "content", "message": "1-2000 karakter."}}, r))
		return
	}
	// Anti-spam komentar: 5/menit dan 30/jam per akun.
	if n, _ := a.Cache.IncrWithExpire(r.Context(), "comment:min:"+uid, time.Minute); n > 5 {
		writeJSON(w, http.StatusTooManyRequests, fail("TOO_MANY_REQUESTS", "Terlalu cepat. Tunggu sebentar.", r))
		return
	}
	if n, _ := a.Cache.IncrWithExpire(r.Context(), "comment:hour:"+uid, time.Hour); n > 30 {
		writeJSON(w, http.StatusTooManyRequests, fail("TOO_MANY_REQUESTS", "Terlalu banyak komentar dalam satu jam.", r))
		return
	}
	var parentID = nilUUID()
	if req.ParentID != nil && *req.ParentID != "" {
		parent, err := a.Q.GetComment(r.Context(), mustUUID(*req.ParentID))
		if err != nil || uuidStr(parent.PortfolioID) != id {
			writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Parent komentar tidak valid.", r))
			return
		}
		parentID = parent.ID
		_, _ = a.Q.EnqueueOutbox(r.Context(), repo.EnqueueOutboxParams{
			Topic:   "notify.reply_comment",
			Payload: []byte(`{"user_id":"` + uuidStr(parent.AuthorID) + `","actor_id":"` + uid + `","portfolio_id":"` + id + `"}`),
		})
	}
	html := sanitizer.Sanitize(req.Content)
	cid, err := a.Q.CreateComment(r.Context(), repo.CreateCommentParams{
		PortfolioID: p.ID, AuthorID: mustUUID(uid), ParentID: parentID,
		BodyMd: req.Content, BodyHtml: html,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal komentar.", r))
		return
	}
	if uuidStr(p.OwnerID) != uid {
		_, _ = a.Q.EnqueueOutbox(r.Context(), repo.EnqueueOutboxParams{
			Topic:   "notify.new_comment",
			Payload: []byte(`{"user_id":"` + uuidStr(p.OwnerID) + `","actor_id":"` + uid + `","portfolio_id":"` + id + `","comment_id":"` + uuidStr(cid) + `"}`),
		})
	}
	writeJSON(w, http.StatusCreated, ok(map[string]string{"id": uuidStr(cid)}, r))
}

func (a *App) handleListComments(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	limit, offset, page := pageParams(r)
	rows, err := a.Q.ListComments(r.Context(), repo.ListCommentsParams{PortfolioID: mustUUID(id), Limit: limit, Offset: offset})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat komentar.", r))
		return
	}
	byID := map[string]map[string]any{}
	roots := []map[string]any{}
	for _, c := range rows {
		node := map[string]any{
			"id": c.ID, "body_md": c.BodyMd, "body_html": c.BodyHtml,
			"username": c.Username, "author_name": c.AuthorName,
			"created_at": c.CreatedAt.Time, "children": []map[string]any{},
		}
		byID[uuidStr(c.ID)] = node
	}
	for _, c := range rows {
		node := byID[uuidStr(c.ID)]
		if c.ParentID.Valid {
			if parent, ok := byID[uuidStr(c.ParentID)]; ok {
				parent["children"] = append(parent["children"].([]map[string]any), node)
				continue
			}
		}
		roots = append(roots, node)
	}
	if roots == nil {
		roots = []map[string]any{}
	}
	total, _ := a.Q.CountComments(r.Context(), mustUUID(id))
	writeJSON(w, http.StatusOK, okMeta(roots, meta(page, limit, total), r))
}

func (a *App) handleDeleteComment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, err := a.Q.GetComment(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Komentar tidak ditemukan.", r))
		return
	}
	p, _ := a.Q.GetPortfolioByID(r.Context(), c.PortfolioID)
	uid := UserID(r)
	if uuidStr(c.AuthorID) != uid && !isOwnerOrAdmin(r, p.OwnerID) {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Tidak bisa hapus komentar ini.", r))
		return
	}
	_ = a.Q.DeleteComment(r.Context(), c.ID)
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "deleted"}, r))
}

func notifPayload(m map[string]string) []byte {
	b := toJSONBytes(m)
	return b
}
