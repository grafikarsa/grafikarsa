package http

import (
	"encoding/json"
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
		if page == 1 {
			if ids, err := a.Cache.GetStrings(r.Context(), "feed:smart:"+uid); err == nil && len(ids) > 0 {
				out := []map[string]any{}
				for _, pid := range ids {
					p, err := a.Q.GetPortfolioByID(r.Context(), mustUUID(pid))
					if err != nil || p.Status != "published" {
						continue
					}
					owner, _ := a.Q.GetUserByID(r.Context(), p.OwnerID)
					out = append(out, map[string]any{
						"id": uuidStr(p.ID), "title": p.Title, "slug": p.Slug,
						"username": owner.Username, "owner_name": owner.Name,
					})
					if len(out) >= int(limit) {
						break
					}
				}
				if len(out) > 0 {
					writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, int64(len(out))), r))
					return
				}
			}
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
		interest, _ := a.Q.GetUserInterest(r.Context(), mustUUID(uid))
		tagW := map[string]float64{}
		majorW := map[string]float64{}
		totalInterest := 0.0
		if interest.UserID.Valid {
			_ = json.Unmarshal(interest.TagWeights, &tagW)
			_ = json.Unmarshal(interest.MajorWeights, &majorW)
			totalInterest = float64(interest.TotalLikes)
		}
		type scored struct {
			item  repo.GetFeedCandidatesRow
			score float64
		}
		list := []scored{}
		// Batch tags sekali untuk 50 kandidat (hindari N+1).
		tagMap := map[string][]repo.ListPortfolioTagsRow{}
		for _, c := range cands {
			if uuidStr(c.OwnerID) == uid {
				continue // jangan tampilkan karya sendiri di smart feed
			}
			tags, _ := a.Q.ListPortfolioTags(r.Context(), c.ID)
			tagMap[uuidStr(c.ID)] = tags
		}
		for _, c := range cands {
			if _, skip := tagMap[uuidStr(c.ID)]; !skip {
				continue
			}
			reverse, _ := a.Q.IsFollowing(r.Context(), repo.IsFollowingParams{FollowerID: c.OwnerID, FollowingID: mustUUID(uid)})
			fScore := 0.0
			if followSet[uuidStr(c.OwnerID)] {
				if reverse > 0 {
					fScore = 1.0
				} else {
					fScore = 0.8
				}
			}
			ageH := time.Since(c.PublishedAt.Time).Hours()
			if ageH < 0 {
				ageH = 0
			}
			recency := math.Exp(-ageH / 168)
			eng := numericToFloat(c.Engagement)
			qual := numericToFloat(c.Quality)
			pTagVec := map[string]float64{}
			for _, t := range tagMap[uuidStr(c.ID)] {
				pTagVec[uuidStr(t.ID)] = 1
			}
			tagKeys := []string{}
			for k := range pTagVec {
				tagKeys = append(tagKeys, k)
			}
			for k := range tagW {
				found := false
				for _, ek := range tagKeys {
					if ek == k {
						found = true
					}
				}
				if !found {
					tagKeys = append(tagKeys, k)
				}
			}
			cos := cosineSimilarity(tagW, pTagVec, tagKeys)
			majorScore := 0.0
			if mw, ok := majorW[uuidStr(c.OwnerMajorID)]; ok && totalInterest > 0 {
				majorScore = log1p(mw) / 5
				if me.MajorID.Valid && c.OwnerMajorID.Valid && uuidStr(me.MajorID) == uuidStr(c.OwnerMajorID) {
					majorScore += 0.5
				}
				if majorScore > 1 {
					majorScore = 1
				}
			} else if me.MajorID.Valid && c.OwnerMajorID.Valid && uuidStr(me.MajorID) == uuidStr(c.OwnerMajorID) {
				majorScore = 0.5
			}
			classScore := 0.0
			if me.ClassID.Valid && c.OwnerClassID.Valid && uuidStr(me.ClassID) == uuidStr(c.OwnerClassID) {
				classScore = 1
			}
			relevance := 0.5*cos + 0.3*majorScore + 0.2*classScore
			final := 0.30*fScore + 0.25*recency + 0.20*eng + 0.15*relevance + 0.10*qual
			list = append(list, scored{c, final})
		}
		for i := 1; i < len(list); i++ {
			j := i
			for j > 0 && list[j].score > list[j-1].score {
				list[j], list[j-1] = list[j-1], list[j]
				j--
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
		ids := []string{}
		for _, s := range list[start:end] {
			out = append(out, map[string]any{
				"id": uuidStr(s.item.ID), "title": s.item.Title, "slug": s.item.Slug,
				"username": s.item.Username, "owner_name": s.item.OwnerName, "score": s.score,
			})
			ids = append(ids, uuidStr(s.item.ID))
		}
		if out == nil {
			out = []map[string]any{}
		}
		if page == 1 && len(ids) > 0 {
			_ = a.Cache.SetStrings(r.Context(), "feed:smart:"+uid, ids, 2*time.Minute)
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
	// Snapshot peringkat di-cache 1 jam (mahal & berubah harian).
	cacheKey := "lb:" + period + ":" + scope + ":" + kind
	if cached, err := a.Cache.GetBytes(r.Context(), cacheKey); err == nil && len(cached) > 0 {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(cached)
		return
	}
	rows, err := a.Q.GetLeaderboard(r.Context(), repo.GetLeaderboardParams{Period: period, Scope: scope, Kind: kind})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat leaderboard.", r))
		return
	}
	out := []map[string]any{}
	for _, e := range rows {
		item := map[string]any{
			"rank": e.Rank, "score": e.Score, "breakdown": e.Breakdown, "delta_rank": e.DeltaRank,
			"subject_user_id":      nullableUUID(e.SubjectUserID),
			"subject_portfolio_id": nullableUUID(e.SubjectPortfolioID),
		}
		// Enrichment untuk tampilan: nama user / judul+tampilan karya.
		if e.SubjectUserID.Valid {
			if u, uerr := a.Q.GetUserByID(r.Context(), e.SubjectUserID); uerr == nil {
				item["name"] = u.Name
				item["username"] = u.Username
				item["major"] = u.MajorID
			}
		}
		if e.SubjectPortfolioID.Valid {
			if p, perr := a.Q.GetPortfolioByID(r.Context(), e.SubjectPortfolioID); perr == nil {
				item["title"] = p.Title
				item["slug"] = p.Slug
				if o, oerr := a.Q.GetUserByID(r.Context(), p.OwnerID); oerr == nil {
					item["username"] = o.Username
					item["owner_name"] = o.Name
				}
			}
		}
		out = append(out, item)
	}
	if out == nil {
		out = []map[string]any{}
	}
	payload, _ := json.Marshal(okMeta(out, nil, r))
	// TTL pendek (10 menit) + invalidasi saat recompute selesai.
	_ = a.Cache.SetBytes(r.Context(), cacheKey, payload, 10*time.Minute)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(payload)
}

func (a *App) handleRecomputeLeaderboard(w http.ResponseWriter, r *http.Request) {
	n, _ := a.Cache.IncrWithExpire(r.Context(), "lb:recompute", time.Hour)
	if n > 1 {
		writeJSON(w, http.StatusTooManyRequests, fail("TOO_MANY_REQUESTS", "Recompute maksimal 1x per jam.", r))
		return
	}
	_, _ = a.Q.EnqueueOutbox(r.Context(), repo.EnqueueOutboxParams{
		Topic: "job.leaderboard_recompute", Payload: []byte("{}"),
	})
	auditLog(r, a, "leaderboards.recompute", "leaderboards", "", "manual")
	writeJSON(w, http.StatusAccepted, ok(map[string]string{"status": "queued"}, r))
}

func (a *App) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if len(q) < 2 {
		writeJSON(w, http.StatusBadRequest, fail("VALIDATION_ERROR", "Query minimal 2 karakter.", r))
		return
	}
	ports, _ := a.Q.SearchPortfolios(r.Context(), q)
	users, _ := a.Q.SearchUsers(r.Context(), q)
	suggest := []map[string]any{}
	if len(ports) == 0 && len(users) == 0 {
		tags, _ := a.Q.SuggestTags(r.Context(), q)
		for _, t := range tags {
			suggest = append(suggest, map[string]any{"slug": t.Slug, "name": t.Name})
		}
		if suggest == nil {
			suggest = []map[string]any{}
		}
	}
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
	writeJSON(w, http.StatusOK, ok(map[string]any{"portfolios": pout, "users": uout, "suggest": suggest}, r))
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
	uid := UserID(r)
	key := "notifcount:" + uid
	if cached, err := a.Cache.GetInt(r.Context(), key); err == nil && cached >= 0 {
		writeJSON(w, http.StatusOK, ok(map[string]int64{"unread": cached}, r))
		return
	}
	n, _ := a.Q.CountUnreadNotifications(r.Context(), mustUUID(uid))
	_ = a.Cache.SetInt(r.Context(), key, n, time.Minute)
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
