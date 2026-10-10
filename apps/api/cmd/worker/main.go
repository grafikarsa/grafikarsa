package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

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
	default:
		fmt.Fprintln(os.Stderr, "usage: worker [outbox|stats|leaderboard]")
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
	for _, c := range cands {
		likes := toInt(c.Likes)
		comments := toInt(c.Comments)
		engagement := 0.0
		if likes > 0 || comments > 0 {
			engagement = float64(likes)/float64(likes+comments+1)*0.6 + float64(comments)/float64(likes+comments+1)*0.4
		}
		quality := 0.0
		if comments > 0 {
			quality = 0.5
		}
		ageDays := time.Since(c.PublishedAt.Time).Hours() / 24
		if ageDays < 0 {
			ageDays = 0
		}
		trending := 0.5*norm(float64(likes), 100) + 0.3*norm(float64(comments), 50) + 0.2*expDecay(ageDays, 7)
		_ = q.UpsertPortfolioStats(ctx, repo.UpsertPortfolioStatsParams{
			PortfolioID: c.ID, Likes: int32(likes), Views7d: 0, Views30d: 0,
			Comments: int32(comments), Quality: floatToNumeric(quality),
			Engagement: floatToNumeric(engagement), Trending: floatToNumeric(trending),
		})
	}
	slog.Info("stats done", "count", len(cands))
	return nil
}

func recomputeLeaderboard(ctx context.Context, q *repo.Queries) error {
	for _, period := range []string{"weekly", "monthly"} {
		for _, scope := range []string{"global"} {
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

func recomputeStudents(ctx context.Context, q *repo.Queries, period, scope string) error {
	cands, err := q.StudentLeaderboardCandidates(ctx)
	if err != nil {
		return err
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
		_ = q.InsertLeaderboardEntry(ctx, repo.InsertLeaderboardEntryParams{
			LeaderboardID: lbID, Rank: int32(rank), SubjectUserID: uuidOrNull(c.ID),
			Score: floatToNumeric(score), Breakdown: bd,
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
		score := 30*float64(toInt(c.Likes))/maxL + 20*expDecay(ageDays, 30) + 10*norm(float64(toInt(c.Comments)), 20)
		bd, _ := json.Marshal(map[string]any{"likes": toInt(c.Likes), "comments": toInt(c.Comments)})
		_ = q.InsertLeaderboardEntry(ctx, repo.InsertLeaderboardEntryParams{
			LeaderboardID: lbID, Rank: int32(rank), SubjectPortfolioID: uuidOrNull(c.ID),
			Score: floatToNumeric(score), Breakdown: bd,
		})
		rank++
		if rank > 20 {
			break
		}
	}
	slog.Info("leaderboard projects", "period", period, "count", rank-1)
	return nil
}
