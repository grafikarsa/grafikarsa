package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"grafikarsa/api/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
	"grafikarsa/api/internal/cache"
	"grafikarsa/api/internal/config"
	"grafikarsa/api/internal/repo"
)

// Worker Fase 2: outbox dispatcher + stats agregator + leaderboard recompute.
// Dijalankan sebagai proses terpisah (compose service worker, Fase 3 digabung).
// Fase 2: one-shot via `go run ./cmd/worker <job>` + cron host/docker.
func main() {
	job := "outbox"
	if len(os.Args) > 1 {
		job = os.Args[1]
	}
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL())
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	// Mode healthcheck untuk distroless: pastikan DB bisa dijangkau.
	if job == "-healthcheck" {
		if err := pool.Ping(ctx); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	defer pool.Close()
	q := repo.New(pool)
	c := cache.New(cfg)
	defer c.Close()

	switch job {
	case "outbox":
		if err := dispatchOutbox(ctx, q, c); err != nil {
			slog.Error("outbox", "err", err)
			os.Exit(1)
		}
	case "stats":
		if err := aggregateStats(ctx, q); err != nil {
			slog.Error("stats", "err", err)
			os.Exit(1)
		}
	case "leaderboard":
		if err := recomputeLeaderboard(ctx, q); err != nil {
			slog.Error("leaderboard", "err", err)
			os.Exit(1)
		}
	case "cleanup-media":
		if err := cleanupMedia(ctx, q, cfg); err != nil {
			slog.Error("cleanup-media", "err", err)
			os.Exit(1)
		}
	case "publish-changelogs":
		if err := publishChangelogs(ctx, q); err != nil {
			slog.Error("publish-changelogs", "err", err)
			os.Exit(1)
		}
	case "retention":
		if err := retention(ctx, q); err != nil {
			slog.Error("retention", "err", err)
			os.Exit(1)
		}
	case "flush-views":
		if err := flushViews(ctx, q, c); err != nil {
			slog.Error("flush-views", "err", err)
			os.Exit(1)
		}
	case "decay-interests":
		if err := decayInterests(ctx, q); err != nil {
			slog.Error("decay-interests", "err", err)
			os.Exit(1)
		}
	case "digest":
		if err := sendDailyDigest(ctx, q); err != nil {
			slog.Error("digest", "err", err)
			os.Exit(1)
		}
	case "promotion-draft":
		if err := createPromotionDraft(ctx, q); err != nil {
			slog.Error("promotion-draft", "err", err)
			os.Exit(1)
		}
	case "serve":
		runScheduler(ctx, q, c, cfg)
	default:
		fmt.Fprintln(os.Stderr, "usage: worker [serve|outbox|stats|leaderboard|cleanup-media|publish-changelogs|retention|promotion-draft]")
		os.Exit(1)
	}
}

func dispatchOutbox(ctx context.Context, q *repo.Queries, c *cache.Cache) error {
	// Kembalikan event yang nyangkut 'processing' (worker mati di tengah batch)
	// supaya tidak hilang permanen.
	_ = q.ReapStuckOutbox(ctx)
	items, err := q.ClaimOutbox(ctx)
	if err != nil {
		return err
	}
	for _, it := range items {
		var p map[string]string
		_ = json.Unmarshal(it.Payload, &p)
		if shouldSkipNotif(ctx, q, p["user_id"], it.Topic) {
			slog.Info("outbox skipped by prefs", "id", it.ID, "topic", it.Topic)
			_ = q.MarkOutboxSent(ctx, it.ID)
			continue
		}
		switch it.Topic {
		case "notify.new_follower":
			if err := emitNotif(ctx, q, c, repo.CreateNotificationParams{
				UserID: mustUUID(p["user_id"]), Type: "new_follower",
				Title: "Pengikut baru", Body: "Seseorang mulai mengikutimu.",
				Data: it.Payload,
			}); err != nil {
				slog.Error("notif gagal dibuat", "topic", it.Topic, "err", err)
				_ = q.MarkOutboxFailed(ctx, it.ID)
				continue
			}
		case "notify.portfolio_liked":
			if err := emitNotif(ctx, q, c, repo.CreateNotificationParams{
				UserID: mustUUID(p["user_id"]), Type: "portfolio_liked",
				Title: "Portfolio disukai", Body: "Seseorang menyukai portfoliomu.",
				Data: it.Payload,
			}); err != nil {
				slog.Error("notif gagal dibuat", "topic", it.Topic, "err", err)
				_ = q.MarkOutboxFailed(ctx, it.ID)
				continue
			}
		case "notify.new_comment", "notify.reply_comment":
			t := "new_comment"
			title := "Komentar baru"
			if it.Topic == "notify.reply_comment" {
				t = "reply_comment"
				title = "Balasan komentar"
			}
			if err := emitNotif(ctx, q, c, repo.CreateNotificationParams{
				UserID: mustUUID(p["user_id"]), Type: t, Title: title,
				Body: "Ada komentar baru.", Data: it.Payload,
			}); err != nil {
				slog.Error("notif gagal dibuat", "topic", it.Topic, "err", err)
				_ = q.MarkOutboxFailed(ctx, it.ID)
				continue
			}
		case "notify.portfolio_approved":
			if err := emitNotif(ctx, q, c, repo.CreateNotificationParams{
				UserID: mustUUID(p["user_id"]), Type: "portfolio_approved",
				Title: "Portfolio diterbitkan", Body: "Selamat! Portfoliomu sudah tayang.",
				Data: it.Payload,
			}); err != nil {
				slog.Error("notif gagal dibuat", "topic", it.Topic, "err", err)
				_ = q.MarkOutboxFailed(ctx, it.ID)
				continue
			}
		case "notify.portfolio_rejected":
			if err := emitNotif(ctx, q, c, repo.CreateNotificationParams{
				UserID: mustUUID(p["user_id"]), Type: "portfolio_rejected",
				Title: "Portfolio ditolak", Body: "Catatan: " + p["note"],
				Data: it.Payload,
			}); err != nil {
				slog.Error("notif gagal dibuat", "topic", it.Topic, "err", err)
				_ = q.MarkOutboxFailed(ctx, it.ID)
				continue
			}
		case "notify.portfolio_changes_requested":
			if err := emitNotif(ctx, q, c, repo.CreateNotificationParams{
				UserID: mustUUID(p["user_id"]), Type: "portfolio_changes_requested",
				Title: "Perlu perbaikan", Body: "Catatan: " + p["note"],
				Data: it.Payload,
			}); err != nil {
				slog.Error("notif gagal dibuat", "topic", it.Topic, "err", err)
				_ = q.MarkOutboxFailed(ctx, it.ID)
				continue
			}
		case "notify.feedback_updated":
			if err := emitNotif(ctx, q, c, repo.CreateNotificationParams{
				UserID: mustUUID(p["user_id"]), Type: "feedback_updated",
				Title: "Feedback diperbarui", Body: "Status: " + p["status"],
				Data: it.Payload,
			}); err != nil {
				slog.Error("notif gagal dibuat", "topic", it.Topic, "err", err)
				_ = q.MarkOutboxFailed(ctx, it.ID)
				continue
			}
		case "notify.announcement":
			if p["user_id"] != "" {
				if err := emitNotif(ctx, q, c, repo.CreateNotificationParams{
					UserID: mustUUID(p["user_id"]), Type: "announcement",
					Title: "Pengumuman", Body: p["message"],
					Data: it.Payload,
				}); err != nil {
					slog.Error("notif gagal dibuat", "topic", it.Topic, "err", err)
					_ = q.MarkOutboxFailed(ctx, it.ID)
					continue
				}
			}
		case "job.leaderboard_recompute":
			if err := recomputeLeaderboard(ctx, q); err != nil {
				slog.Error("leaderboard recompute", "err", err)
				_ = q.MarkOutboxFailed(ctx, it.ID)
				continue
			}
			// Snapshot berubah -> buang cache peringkat (key lb:*).
			_ = c.DelPattern(ctx, "lb:*")
		case "portfolio.submitted":
			notifyModerators(ctx, q, c, p["portfolio_id"])
		case "notify.changelog_published":
			fanoutChangelog(ctx, q, c, p)
		default:
			// Topic tak dikenal: tandai gagal (jangan hilang diam-diam).
			slog.Error("outbox topic tidak dikenal", "id", it.ID, "topic", it.Topic)
			_ = q.MarkOutboxFailed(ctx, it.ID)
			continue
		}
		_ = q.MarkOutboxSent(ctx, it.ID)
		slog.Info("outbox sent", "id", it.ID, "topic", it.Topic)
	}
	slog.Info("outbox done", "count", len(items))
	return nil
}

func aggregateStats(ctx context.Context, q *repo.Queries) error {
	cands, err := q.StatsPortfolioCandidates(ctx)
	if err != nil {
		return err
	}
	maxL, maxV, maxC := 1.0, 1.0, 1.0
	for _, c := range cands {
		likes := toInt(c.TotalLikes)
		comments := toInt(c.TotalComments)
		v7, _ := q.CountViewsSince(ctx, repo.CountViewsSinceParams{PortfolioID: c.ID, Column2: 7})
		v30, _ := q.CountViewsSince(ctx, repo.CountViewsSinceParams{PortfolioID: c.ID, Column2: 30})
		l7, _ := q.CountLikesSince(ctx, repo.CountLikesSinceParams{PortfolioID: c.ID, Column2: "7"})
		c7, _ := q.CountCommentsSince(ctx, repo.CountCommentsSinceParams{PortfolioID: c.ID, Column2: "7"})
		if float64(likes) > maxL {
			maxL = float64(likes)
		}
		if float64(v7) > maxV {
			maxV = float64(v7)
		}
		if float64(comments) > maxC {
			maxC = float64(comments)
		}
		ageDays := time.Since(c.PublishedAt.Time).Hours() / 24
		if ageDays < 0 {
			ageDays = 0
		}
		engagement := float64(l7)/maxL*0.6 + float64(v7)/maxV*0.4
		if engagement > 1 {
			engagement = 1
		}
		// Quality: 50% kelengkapan (thumbnail) + 50% kualitas (assessment bila ada,
		// selain itu sinyal interaksi). Skala 0-1.
		quality := 0.0
		if c.HasThumbnail {
			quality += 0.5
		}
		if c.AssessmentAvg > 0 {
			quality += 0.5 * c.AssessmentAvg / 10
		} else if likes > 0 || comments > 0 {
			quality += 0.25
		}
		trending := 0.5*norm(float64(l7), maxL) + 0.3*norm(float64(v7), maxV) + 0.2*norm(float64(c7), maxC)
		trending *= expNeg(ageDays / 7)
		if err := q.UpsertPortfolioStats(ctx, repo.UpsertPortfolioStatsParams{
			PortfolioID: c.ID, Likes: int32(likes), Views7d: int32(v7), Views30d: int32(v30),
			Comments: int32(comments), AssessmentAvg: floatToNumeric(c.AssessmentAvg),
			Quality:    floatToNumeric(quality),
			Engagement: floatToNumeric(engagement), Trending: floatToNumeric(trending),
		}); err != nil {
			slog.Error("stats upsert gagal", "portfolio", uuidStr(c.ID), "err", err)
		}
	}
	slog.Info("stats done", "count", len(cands))
	return nil
}

func recomputeLeaderboard(ctx context.Context, q *repo.Queries) error {
	scopes := []string{"global", "jurusan:rpl", "jurusan:tkj", "jurusan:mm", "jurusan:dkv", "jurusan:animasi"}
	for _, period := range []string{"weekly", "monthly"} {
		for _, scope := range scopes {
			if err := recomputeStudents(ctx, q, period, scope); err != nil {
				return err
			}
			if err := recomputeProjects(ctx, q, period, scope); err != nil {
				return err
			}
		}
	}
	return nil
}

func scopeMajorCode(scope string) string {
	if len(scope) > 8 && scope[:8] == "jurusan:" {
		return scope[8:]
	}
	return ""
}

func periodDays(period string) string {
	if period == "monthly" {
		return "30"
	}
	return "7"
}

func recomputeStudents(ctx context.Context, q *repo.Queries, period, scope string) error {
	cands, err := q.StudentLeaderboardCandidates(ctx, periodDays(period))
	if err != nil {
		return err
	}
	majorCode := scopeMajorCode(scope)
	filtered := cands[:0]
	for _, c := range cands {
		if majorCode != "" && c.MajorCode != majorCode {
			continue
		}
		filtered = append(filtered, c)
	}
	cands = filtered
	prev, _ := q.PrevLeaderboardRanks(ctx, repo.PrevLeaderboardRanksParams{Period: period, Scope: scope, Kind: "best_students"})
	prevRank := map[string]int{}
	for _, p := range prev {
		if p.SubjectUserID.Valid {
			prevRank[uuidStr(p.SubjectUserID)] = int(p.Rank)
		}
	}
	lbID, err := q.UpsertLeaderboard(ctx, repo.UpsertLeaderboardParams{Period: period, Scope: scope, Kind: "best_students"})
	if err != nil {
		return err
	}
	_ = q.ClearLeaderboardEntries(ctx, lbID)
	maxP, maxL, maxA, maxF := 1.0, 1.0, 0.0, 1.0
	for _, c := range cands {
		if f := float64(toInt(c.PortfolioCount)); f > maxP {
			maxP = f
		}
		if f := float64(toInt(c.TotalLikes)); f > maxL {
			maxL = f
		}
		if a := c.AssessmentAvg / 10; a > maxA {
			maxA = a
		}
		if f := float64(toInt(c.Followers)); f > maxF {
			maxF = f
		}
	}
	rank := 1
	for _, c := range cands {
		// 30 portfolio + 25 likes + 30 assessment + 15 followers (PRD §9.2).
		assess := 0.0
		if maxA > 0 {
			assess = 30 * (c.AssessmentAvg / 10) / maxA
		}
		score := 30*float64(toInt(c.PortfolioCount))/maxP +
			25*float64(toInt(c.TotalLikes))/maxL +
			assess +
			15*float64(toInt(c.Followers))/maxF
		bd, _ := json.Marshal(map[string]any{
			"portfolios": toInt(c.PortfolioCount), "likes": toInt(c.TotalLikes),
			"assessment": c.AssessmentAvg, "followers": toInt(c.Followers),
		})
		delta := 0
		if pr, ok := prevRank[uuidStr(c.ID)]; ok {
			delta = pr - rank
		}
		_ = q.InsertLeaderboardEntry(ctx, repo.InsertLeaderboardEntryParams{
			LeaderboardID: lbID, Rank: int32(rank), SubjectUserID: uuidOrNull(c.ID),
			Score: floatToNumeric(score), Breakdown: bd, DeltaRank: int32(delta),
		})
		// Notifikasi kenaikan peringkat (hanya global, 10 besar, naik).
		if scope == "global" && rank <= 10 && delta > 0 {
			if _, nerr := q.CreateNotification(ctx, repo.CreateNotificationParams{
				UserID: c.ID, Type: "leaderboard_ranked",
				Title: "Peringkat naik",
				Body:  fmt.Sprintf("Peringkatmu naik ke #%d (%s).", rank, period),
				Data:  notifPayload(map[string]string{"kind": "leaderboard_ranked", "rank": fmt.Sprint(rank), "period": period}),
			}); nerr != nil {
				slog.Warn("notif peringkat gagal", "user", uuidStr(c.ID), "err", nerr)
			}
		}
		rank++
		if rank > 20 {
			break
		}
	}
	slog.Info("leaderboard students", "period", period, "count", rank-1)
	return nil
}

func recomputeProjects(ctx context.Context, q *repo.Queries, period, scope string) error {
	cands, err := q.ProjectLeaderboardCandidates(ctx, periodDays(period))
	if err != nil {
		return err
	}
	majorCode := scopeMajorCode(scope)
	filtered := cands[:0]
	for _, c := range cands {
		if majorCode != "" && c.OwnerMajorCode != majorCode {
			continue
		}
		filtered = append(filtered, c)
	}
	cands = filtered
	topIDs := map[string]bool{}
	top, _ := q.TopLikedUnassessed(ctx, 100)
	for _, t := range top {
		topIDs[uuidStr(t)] = true
	}
	prev, _ := q.PrevLeaderboardRanks(ctx, repo.PrevLeaderboardRanksParams{Period: period, Scope: scope, Kind: "best_projects"})
	prevRank := map[string]int{}
	for _, p := range prev {
		if p.SubjectPortfolioID.Valid {
			prevRank[uuidStr(p.SubjectPortfolioID)] = int(p.Rank)
		}
	}
	lbID, err := q.UpsertLeaderboard(ctx, repo.UpsertLeaderboardParams{Period: period, Scope: scope, Kind: "best_projects"})
	if err != nil {
		return err
	}
	_ = q.ClearLeaderboardEntries(ctx, lbID)
	maxL := 1.0
	for _, c := range cands {
		if f := float64(toInt(c.Likes)); f > maxL {
			maxL = f
		}
	}
	rank := 1
	for _, c := range cands {
		ageDays := time.Since(c.PublishedAt.Time).Hours() / 24
		if ageDays < 0 {
			ageDays = 0
		}
		assessed := c.AssessmentCount > 0
		if !assessed && !topIDs[uuidStr(c.ID)] {
			continue // belum dinilai dan bukan top-5% likes -> skip (label Rising hanya untuk top)
		}
		// 50 assessment + 30 likes + 20 recency (PRD §9.2).
		score := 30*float64(toInt(c.Likes))/maxL + 20*expNeg(ageDays/30)
		if assessed {
			score += 50 * c.AssessmentAvg / 10
		}
		bd, _ := json.Marshal(map[string]any{"likes": toInt(c.Likes), "comments": toInt(c.Comments), "rising": !assessed})
		delta := 0
		if pr, ok := prevRank[uuidStr(c.ID)]; ok {
			delta = pr - rank
		}
		_ = q.InsertLeaderboardEntry(ctx, repo.InsertLeaderboardEntryParams{
			LeaderboardID: lbID, Rank: int32(rank), SubjectPortfolioID: uuidOrNull(c.ID),
			Score: floatToNumeric(score), Breakdown: bd, DeltaRank: int32(delta),
		})
		rank++
		if rank > 20 {
			break
		}
	}
	slog.Info("leaderboard projects", "period", period, "count", rank-1)
	return nil
}

func cleanupMedia(ctx context.Context, q *repo.Queries, cfg *config.Config) error {
	rows, err := q.ListPendingMedia(ctx)
	if err != nil {
		return err
	}
	st := newStorage(cfg)
	n := 0
	for _, m := range rows {
		_ = st.Delete(ctx, m.ObjectKey)
		_ = q.DeleteMediaRow(ctx, m.ID)
		n++
	}
	slog.Info("cleanup-media done", "count", n)
	return nil
}

func newStorage(cfg *config.Config) *storage.Storage { return storage.New(cfg) }

// emitNotif membuat notifikasi dan membatalkan cache badge unread user, serta
// mengembalikan error supaya kegagalan tidak hilang diam-diam (outbox di-retry).
func emitNotif(ctx context.Context, q *repo.Queries, c *cache.Cache, params repo.CreateNotificationParams) error {
	if _, err := q.CreateNotification(ctx, params); err != nil {
		return err
	}
	if c != nil {
		_ = c.Del(ctx, "notifcount:"+uuidStr(params.UserID))
	}
	return nil
}

// flushViews membaca HyperLogLog per portfolio/hari lalu menulis agregatnya ke
// portfolio_view_counts. Idempotent: nilai diambil GREATEST sehingga pengulangan
// tidak menurunkan angka.
func flushViews(ctx context.Context, q *repo.Queries, c *cache.Cache) error {
	day := time.Now().Format("2006-01-02")
	dirtyKey := "views:dirty:" + day
	ids, err := c.SMembers(ctx, dirtyKey)
	if err != nil || len(ids) == 0 {
		return nil
	}
	flushed := 0
	for _, pid := range ids {
		n, perr := c.PFCount(ctx, "views:hll:"+pid+":"+day)
		if perr != nil {
			continue
		}
		if uerr := q.UpsertViewCount(ctx, repo.UpsertViewCountParams{
			PortfolioID: mustUUID(pid), ViewedDate: toPgDate(day), UniqueViews: int32(n),
		}); uerr != nil {
			slog.Warn("view count gagal disimpan", "portfolio", pid, "err", uerr)
			continue
		}
		flushed++
	}
	_ = q.DeleteOldViewCounts(ctx)
	_ = c.Del(ctx, dirtyKey)
	slog.Info("view flush", "portfolio", flushed, "hari", day)
	return nil
}

// decayInterests meluruhkan bobot minat sekali sehari agar preferensi lama tidak
// mendominasi rekomendasi (0.98^hari).
func decayInterests(ctx context.Context, q *repo.Queries) error {
	factor := floatToNumeric(0.98)
	if err := q.DecayUserInterests(ctx, factor); err != nil {
		return err
	}
	slog.Info("interests decay dijalankan", "factor", 0.98)
	return nil
}

// sendDailyDigest mengirim ringkasan notifikasi belum dibaca sekali sehari
// untuk user yang mengaktifkan digest_daily (job harian 07:00 WIB).
func sendDailyDigest(ctx context.Context, q *repo.Queries) error {
	users, err := q.ListDigestUsers(ctx)
	if err != nil {
		return err
	}
	sent := 0
	for _, u := range users {
		sum, err := q.DigestSummaryForUser(ctx, u)
		if err != nil || sum.Unread == 0 {
			continue
		}
		body := fmt.Sprintf("Ada %d notifikasi belum dibaca.", sum.Unread)
		if sum.Sample != "" {
			body += " " + sum.Sample
		}
		if _, err := q.CreateNotification(ctx, repo.CreateNotificationParams{
			UserID: u, Type: "digest", Title: "Ringkasan harian", Body: body,
			Data: notifPayload(map[string]string{"kind": "digest"}),
		}); err != nil {
			slog.Error("digest gagal", "user", uuidStr(u), "err", err)
			continue
		}
		sent++
	}
	slog.Info("digest terkirim", "count", sent, "kandidat", len(users))
	return nil
}

// notifyModerators: beri tahu admin + admin_jurusan terkait saat portfolio
// masuk antrean review (bukan hanya menunggu mereka membuka dashboard).
func notifyModerators(ctx context.Context, q *repo.Queries, c *cache.Cache, portfolioID string) {
	p, err := q.GetPortfolioByID(ctx, mustUUID(portfolioID))
	if err != nil {
		return
	}
	owner, _ := q.GetUserByID(ctx, p.OwnerID)
	major := ""
	if owner.MajorID.Valid {
		major = uuidStr(owner.MajorID)
	}
	mods, err := q.ListModeratorIDs(ctx, major)
	if err != nil {
		return
	}
	data, _ := json.Marshal(map[string]string{"portfolio_id": portfolioID, "owner": owner.Username})
	for _, m := range mods {
		if err := emitNotif(ctx, q, c, repo.CreateNotificationParams{
			UserID: m, Type: "announcement",
			Title: "Portfolio menunggu review",
			Body:  owner.Name + " mengirim \"" + p.Title + "\" untuk direview.",
			Data:  data,
		}); err != nil {
			slog.Error("notif moderator gagal", "user", uuidStr(m), "err", err)
			continue
		}
	}
	slog.Info("moderator notified", "portfolio", portfolioID, "count", len(mods))
}

// fanoutChangelog: kirim pengumuman rilis ke user sesuai audience.
func fanoutChangelog(ctx context.Context, q *repo.Queries, c *cache.Cache, p map[string]string) {
	ids, err := q.ListAudienceUserIDs(ctx, p["audience"])
	if err != nil {
		return
	}
	data, _ := json.Marshal(map[string]string{"version": p["version"], "changelog_id": p["changelog_id"]})
	for _, uid := range ids {
		if err := emitNotif(ctx, q, c, repo.CreateNotificationParams{
			UserID: uid, Type: "announcement",
			Title: "Grafikarsa " + p["version"] + " rilis",
			Body:  "Ada pembaruan platform. Lihat changelog untuk detailnya.",
			Data:  data,
		}); err != nil {
			slog.Error("notif fanout gagal", "user", uuidStr(uid), "err", err)
			continue
		}
	}
	slog.Info("changelog fanout", "version", p["version"], "count", len(ids))
}

func notifPayload(m map[string]string) []byte {
	b, _ := json.Marshal(m)
	return b
}

func publishChangelogs(ctx context.Context, q *repo.Queries) error {
	rows, err := q.PublishDueChangelogs(ctx)
	if err != nil {
		return err
	}
	for _, c := range rows {
		_, _ = q.EnqueueOutbox(ctx, repo.EnqueueOutboxParams{
			Topic:   "notify.changelog_published",
			Payload: notifPayload(map[string]string{"kind": "changelog_published", "changelog_id": uuidStr(c.ID), "version": c.Version, "audience": c.Audience}),
		})
		slog.Info("changelog published", "version", c.Version)
	}
	// announcement fanout: kirim ke semua user aktif (batch via notif langsung, bukan per-user outbox meledak).
	// Fase 3: fanout sederhana — outbox dispatcher menangani per-user saat dibaca? Tidak.
	// Dipilih: notif broadcast dibuat saat user buka /changelogs (mark read) + badge unread count.
	// Jadi di sini cukup publish status; fanout push menyusul Fase 4 bila perlu.
	slog.Info("publish-changelogs done", "count", len(rows))
	return nil
}

func retention(ctx context.Context, q *repo.Queries) error {
	if err := q.DeleteOldReadNotifications(ctx); err != nil {
		return err
	}
	slog.Info("retention done")
	return nil
}
