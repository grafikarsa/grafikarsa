package http

import (
	"math"

	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"grafikarsa/api/internal/repo"
)

func recencyScore(publishedAt interface{ GetTime() bool }) float64 { return 0 }

func followingScore(isMutual, isOneWay bool) float64 {
	if isMutual {
		return 1.0
	}
	if isOneWay {
		return 0.8
	}
	return 0
}

func (a *App) handleFeed(w http.ResponseWriter, r *http.Request) {
	algo := r.URL.Query().Get("algorithm")
	if algo == "" {
		algo = "recent"
		if UserID(r) != "" {
			algo = "smart"
		}
	}
	limit, offset, page := pageParams(r)
	if limit > 15 {
		limit = 15
	}
	switch algo {
	case "recent":
		rows, err := a.Q.GetRecentFeed(r.Context(), repo.GetRecentFeedParams{Limit: limit, Offset: offset})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat feed.", r))
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
		writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, int64(len(out))), r))
	case "following":
		uid := UserID(r)
		if uid == "" {
			writeJSON(w, http.StatusUnauthorized, fail("UNAUTHORIZED", "Login untuk feed following.", r))
			return
		}
		rows, err := a.Q.GetFollowingFeed(r.Context(), repo.GetFollowingFeedParams{FollowerID: mustUUID(uid), Limit: limit, Offset: offset})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat feed.", r))
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
		writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, int64(len(out))), r))
	default: // smart
		uid := UserID(r)
		if uid == "" {
			writeJSON(w, http.StatusUnauthorized, fail("UNAUTHORIZED", "Login untuk feed smart.", r))
			return
		}
		cacheKey := "feed:" + uid + ":smart:" + r.URL.Query().Get("page")
		if cached, err := a.Cache.GetStrings(r.Context(), cacheKey); err == nil && len(cached) > 0 {
			writeJSON(w, http.StatusOK, okMeta([]map[string]any{}, meta(page, limit, 0), r))
			return
		}
		cands, err := a.Q.GetFeedCandidates(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat feed.", r))
			return
		}
		following, _ := a.Q.ListFollowingIDs(r.Context(), mustUUID(uid))
		followSet := map[string]bool{}
		for _, f := range following {
			followSet[uuidStr(f)] = true
		}
		me, _ := a.Q.GetUserByID(r.Context(), mustUUID(uid))
		type scored struct {
			item  repo.GetFeedCandidatesRow
			score float64
		}
		list := []scored{}
		for _, c := range cands {
			fScore := 0.0
			if followSet[uuidStr(c.ID)] {
				fScore = 0.8
			}
			ageH := time.Since(c.PublishedAt.Time).Hours()
			if ageH < 0 {
				ageH = 0
			}
			recency := math.Exp(-ageH / 168)
			eng := numericToFloat(c.Engagement)
			qual := numericToFloat(c.Quality)
			relevance := 0.0
			if me.MajorID.Valid && c.OwnerMajorID.Valid && uuidStr(me.MajorID) == uuidStr(c.OwnerMajorID) {
				relevance += 0.15
			}
			if me.ClassID.Valid && c.OwnerClassID.Valid && uuidStr(me.ClassID) == uuidStr(c.OwnerClassID) {
				relevance += 0.2
			}
			final := 0.30*fScore + 0.25*recency + 0.20*eng + 0.15*relevance + 0.10*qual
			list = append(list, scored{c, final})
		}
		for i := 0; i < len(list); i++ {
			for j := i + 1; j < len(list); j++ {
				if list[j].score > list[i].score {
					list[i], list[j] = list[j], list[i]
				}
			}
		}
		start := int(offset)
		if start > len(list) {
			start = len(list)
		}
		end := start + int(limit)
		if end > len(list) {
			end = len(list)
		}
		out := []map[string]any{}
		for _, s := range list[start:end] {
			out = append(out, map[string]any{
				"id": uuidStr(s.item.ID), "title": s.item.Title, "slug": s.item.Slug,
				"username": s.item.Username, "owner_name": s.item.OwnerName, "score": s.score,
			})
		}
		if out == nil {
			out = []map[string]any{}
		}
		writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, int64(len(list))), r))
	}
}

func (a *App) handleGetFeedPrefs(w http.ResponseWriter, r *http.Request) {
	uid := UserID(r)
	pref, err := a.Q.GetFeedPreference(r.Context(), mustUUID(uid))
	if err != nil {
		writeJSON(w, http.StatusOK, ok(map[string]string{"algorithm": "smart"}, r))
		return
	}
	writeJSON(w, http.StatusOK, ok(map[string]string{"algorithm": pref.Algorithm}, r))
}

func (a *App) handleUpdateFeedPrefs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Algorithm string `json:"algorithm"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Algorithm != "smart" && req.Algorithm != "recent" && req.Algorithm != "following" {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Algoritma tidak valid.", r))
		return
	}
	_ = a.Q.SaveFeedPreference(r.Context(), repo.SaveFeedPreferenceParams{UserID: mustUUID(UserID(r)), Algorithm: req.Algorithm})
	writeJSON(w, http.StatusOK, ok(map[string]string{"algorithm": req.Algorithm}, r))
}

func (a *App) handleLeaderboards(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "weekly"
	}
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		scope = "global"
	}
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		kind = "best_projects"
	}
	rows, err := a.Q.GetLeaderboard(r.Context(), repo.GetLeaderboardParams{Period: period, Scope: scope, Kind: kind})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat leaderboard.", r))
		return
	}
	out := []map[string]any{}
	for _, e := range rows {
		out = append(out, map[string]any{
			"rank": e.Rank, "score": e.Score, "breakdown": e.Breakdown, "delta_rank": e.DeltaRank,
			"subject_user_id":      nullableUUID(e.SubjectUserID),
			"subject_portfolio_id": nullableUUID(e.SubjectPortfolioID),
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, ok(out, r))
}

func (a *App) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if len(q) < 2 {
		writeJSON(w, http.StatusBadRequest, fail("VALIDATION_ERROR", "Query minimal 2 karakter.", r))
		return
	}
	ports, _ := a.Q.SearchPortfolios(r.Context(), q)
	users, _ := a.Q.SearchUsers(r.Context(), q)
	pout := []map[string]any{}
	for _, p := range ports {
		pout = append(pout, map[string]any{"id": uuidStr(p.ID), "title": p.Title, "slug": p.Slug, "username": p.Username})
	}
	uout := []map[string]any{}
	for _, u := range users {
		uout = append(uout, map[string]any{"id": uuidStr(u.ID), "username": u.Username, "name": u.Name})
	}
	if pout == nil {
		pout = []map[string]any{}
	}
	if uout == nil {
		uout = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, ok(map[string]any{"portfolios": pout, "users": uout}, r))
}

func (a *App) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := pageParams(r)
	rows, err := a.Q.ListNotifications(r.Context(), repo.ListNotificationsParams{UserID: mustUUID(UserID(r)), Limit: limit, Offset: offset})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat notifikasi.", r))
		return
	}
	total, _ := a.Q.CountUnreadNotifications(r.Context(), mustUUID(UserID(r)))
	out := []map[string]any{}
	for _, n := range rows {
		out = append(out, map[string]any{
			"id": n.ID, "type": n.Type, "title": n.Title, "body": n.Body,
			"data": n.Data, "is_read": n.IsRead, "created_at": n.CreatedAt.Time,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, total), r))
}

func (a *App) handleNotifCount(w http.ResponseWriter, r *http.Request) {
	n, _ := a.Q.CountUnreadNotifications(r.Context(), mustUUID(UserID(r)))
	writeJSON(w, http.StatusOK, ok(map[string]int64{"unread": n}, r))
}

func (a *App) handleNotifRead(w http.ResponseWriter, r *http.Request) {
	id := chiURLParam(r, "id")
	_ = a.Q.MarkNotificationRead(r.Context(), repo.MarkNotificationReadParams{ID: mustUUID(id), UserID: mustUUID(UserID(r))})
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleNotifReadAll(w http.ResponseWriter, r *http.Request) {
	_ = a.Q.MarkAllNotificationsRead(r.Context(), mustUUID(UserID(r)))
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleNotifDelete(w http.ResponseWriter, r *http.Request) {
	id := chiURLParam(r, "id")
	_ = a.Q.DeleteNotification(r.Context(), repo.DeleteNotificationParams{ID: mustUUID(id), UserID: mustUUID(UserID(r))})
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "deleted"}, r))
}

func chiURLParam(r *http.Request, key string) string { return chi.URLParam(r, key) }
