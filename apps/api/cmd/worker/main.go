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
	defer pool.Close()
	q := repo.New(pool)
	c := cache.New(cfg)
	defer c.Close()

	switch job {
	case "outbox":
		if err := dispatchOutbox(ctx, q); err != nil {
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
	default:
		fmt.Fprintln(os.Stderr, "usage: worker [outbox|stats|leaderboard|cleanup-media]")
		os.Exit(1)
	}
}

func dispatchOutbox(ctx context.Context, q *repo.Queries) error {
	items, err := q.ClaimOutbox(ctx)
	if err != nil {
		return err
	}
	for _, it := range items {
		var p map[string]string
		_ = json.Unmarshal(it.Payload, &p)
		if shouldSkipNotif(ctx, q, p["user_id"], it.Topic) {
			slog.Info("outbox skipped by prefs", "id", it.ID, "topic", it.Topic)
			continue
		}
		switch it.Topic {
		case "notify.new_follower":
			_, _ = q.CreateNotification(ctx, repo.CreateNotificationParams{
				UserID: mustUUID(p["user_id"]), Type: "new_follower",
				Title: "Pengikut baru", Body: "Seseorang mulai mengikutimu.",
				Data: it.Payload,
			})
		case "notify.portfolio_liked":
			_, _ = q.CreateNotification(ctx, repo.CreateNotificationParams{
				UserID: mustUUID(p["user_id"]), Type: "portfolio_liked",
				Title: "Portfolio disukai", Body: "Seseorang menyukai portfoliomu.",
				Data: it.Payload,
			})
		case "notify.new_comment", "notify.reply_comment":
			t := "new_comment"
			title := "Komentar baru"
			if it.Topic == "notify.reply_comment" {
				t = "reply_comment"
				title = "Balasan komentar"
			}
			_, _ = q.CreateNotification(ctx, repo.CreateNotificationParams{
				UserID: mustUUID(p["user_id"]), Type: t, Title: title,
				Body: "Ada komentar baru.", Data: it.Payload,
			})
		case "notify.portfolio_approved":
			_, _ = q.CreateNotification(ctx, repo.CreateNotificationParams{
				UserID: mustUUID(p["user_id"]), Type: "portfolio_approved",
				Title: "Portfolio diterbitkan", Body: "Selamat! Portfoliomu sudah tayang.",
				Data: it.Payload,
			})
		case "notify.portfolio_rejected":
			_, _ = q.CreateNotification(ctx, repo.CreateNotificationParams{
				UserID: mustUUID(p["user_id"]), Type: "portfolio_rejected",
				Title: "Portfolio ditolak", Body: "Catatan: " + p["note"],
				Data: it.Payload,
			})
		case "notify.portfolio_changes_requested":
			_, _ = q.CreateNotification(ctx, repo.CreateNotificationParams{
				UserID: mustUUID(p["user_id"]), Type: "portfolio_changes_requested",
				Title: "Perlu perbaikan", Body: "Catatan: " + p["note"],
				Data: it.Payload,
			})
		}
		slog.Info("outbox sent", "id", it.ID, "topic", it.Topic)
	}
	slog.Info("outbox done", "count", len(items))
	return nil
}

func aggregateStats(ctx context.Context, q *repo.Queries) error {
	cands, err := q.ProjectLeaderboardCandidates(ctx)
	if err != nil {
		return err
	}
	maxL, maxV, maxC := 1.0, 1.0, 1.0
	for _, c := range cands {
		likes := toInt(c.Likes)
		comments := toInt(c.Comments)
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
		quality := 0.5
		if comments == 0 && likes == 0 {
			quality = 0
		}
		trending := 0.5*norm(float64(l7), maxL) + 0.3*norm(float64(v7), maxV) + 0.2*norm(float64(c7), maxC)
		trending *= expNeg(ageDays / 7)
		_ = q.UpsertPortfolioStats(ctx, repo.UpsertPortfolioStatsParams{
			PortfolioID: c.ID, Likes: int32(likes), Views7d: int32(v7), Views30d: int32(v30),
			Comments: int32(comments), Quality: floatToNumeric(quality),
			Engagement: floatToNumeric(engagement), Trending: floatToNumeric(trending),
		})
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

func recomputeStudents(ctx context.Context, q *repo.Queries, period, scope string) error {
	cands, err := q.StudentLeaderboardCandidates(ctx)
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
	maxP, maxL, maxF := 1.0, 1.0, 1.0
	for _, c := range cands {
		if f := float64(toInt(c.PortfolioCount)); f > maxP {
			maxP = f
		}
		if f := float64(toInt(c.TotalLikes)); f > maxL {
			maxL = f
		}
		if f := float64(toInt(c.Followers)); f > maxF {
			maxF = f
		}
	}
	rank := 1
	for _, c := range cands {
		score := 30*float64(toInt(c.PortfolioCount))/maxP + 25*float64(toInt(c.TotalLikes))/maxL + 15*float64(toInt(c.Followers))/maxF
		bd, _ := json.Marshal(map[string]any{"portfolios": toInt(c.PortfolioCount), "likes": toInt(c.TotalLikes), "followers": toInt(c.Followers)})
		delta := 0
		if pr, ok := prevRank[uuidStr(c.ID)]; ok {
			delta = pr - rank
		}
		_ = q.InsertLeaderboardEntry(ctx, repo.InsertLeaderboardEntryParams{
			LeaderboardID: lbID, Rank: int32(rank), SubjectUserID: uuidOrNull(c.ID),
			Score: floatToNumeric(score), Breakdown: bd, DeltaRank: int32(delta),
		})
		rank++
		if rank > 20 {
			break
		}
	}
	slog.Info("leaderboard students", "period", period, "count", rank-1)
	return nil
}

func recomputeProjects(ctx context.Context, q *repo.Queries, period, scope string) error {
	cands, err := q.ProjectLeaderboardCandidates(ctx)
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
		score := 30*float64(toInt(c.Likes))/maxL + 20*expNeg(ageDays/30) + 10*norm(float64(toInt(c.Comments)), 20)
		if assessed {
			score += 50 * float64(c.AssessmentAvg) / 10
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
