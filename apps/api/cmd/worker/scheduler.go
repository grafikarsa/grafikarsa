package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"grafikarsa/api/internal/cache"
	"grafikarsa/api/internal/config"
	"grafikarsa/api/internal/repo"
)

// Scheduler: menjalankan semua job berkala dalam satu proses.
// Interval dipilih agar: outbox cepat (30 dtk), stats segar (5 mnt),
// leaderboard harian 02:05 WIB, promotion draft harian 00:10 WIB.
func runScheduler(ctx context.Context, q *repo.Queries, c *cache.Cache, cfg *config.Config) {
	slog.Info("scheduler start")

	go every(ctx, 30*time.Second, "outbox", func() error { return dispatchOutbox(ctx, q, c) })
	go every(ctx, 5*time.Minute, "stats", func() error { return aggregateStats(ctx, q) })
	go every(ctx, 5*time.Minute, "publish-changelogs", func() error { return publishChangelogs(ctx, q) })
	go every(ctx, time.Hour, "cleanup-media", func() error { return cleanupMedia(ctx, q, cfg) })
	go every(ctx, 24*time.Hour, "retention", func() error { return retention(ctx, q) })
	go dailyAt(ctx, 2, 5, "leaderboard", func() error { return recomputeLeaderboard(ctx, q) })
	go dailyAt(ctx, 0, 10, "promotion-draft", func() error { return createPromotionDraft(ctx, q) })
	go dailyAt(ctx, 7, 0, "digest", func() error { return sendDailyDigest(ctx, q) })

	<-ctx.Done()
	slog.Info("scheduler stop")
}

func safeRun(name string, fn func() error) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("job panic", "job", name, "panic", rec)
		}
	}()
	if err := fn(); err != nil {
		slog.Error("job failed", "job", name, "err", err)
	} else {
		slog.Info("job ok", "job", name)
	}
}

func every(ctx context.Context, d time.Duration, name string, fn func() error) {
	run := func() {
		if err := fn(); err != nil {
			slog.Error("job failed", "job", name, "err", err)
			return
		}
		slog.Info("job ok", "job", name)
	}
	run()
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			run()
		}
	}
}

func dailyAt(ctx context.Context, hour, minute int, name string, fn func() error) {
	loc := time.FixedZone("WIB", 7*3600) // hindari ketergantungan tzdata di distroless
	next := func() time.Duration {
		now := time.Now().In(loc)
		n := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
		if !n.After(now) {
			n = n.Add(24 * time.Hour)
		}
		return time.Until(n)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(next()):
			if err := fn(); err != nil {
				slog.Error("job failed", "job", name, "err", err)
			} else {
				slog.Info("job ok", "job", name)
			}
		}
	}
}

// createPromotionDraft: saat tanggal kenaikan kelas tiba, buat job draft +
// preview otomatis + notifikasi admin. Eksekusi TETAP manual (approve).
func createPromotionDraft(ctx context.Context, q *repo.Queries) error {
	ay, err := q.GetActiveYearFull(ctx)
	if err != nil {
		return nil // belum ada tahun aktif
	}
	loc := time.FixedZone("WIB", 7*3600)
	now := time.Now().In(loc)
	if int(now.Month()) != int(ay.PromotionMonth) || now.Day() != int(ay.PromotionDay) {
		return nil
	}
	next, err := q.FindYearByStartYear(ctx, ay.StartYear+1)
	if err != nil {
		slog.Info("promotion draft dilewati: tahun berikutnya belum dibuat")
		return nil
	}
	if _, err := q.GetPromotionJobByYears(ctx, repo.GetPromotionJobByYearsParams{
		FromYearID: ay.ID, ToYearID: next.ID,
	}); err == nil {
		return nil // sudah ada
	}
	enrolls, _ := q.ListEnrollmentsByYear(ctx, ay.ID)
	naik := map[string]int{}
	lulus := 0
	for _, e := range enrolls {
		if e.Grade >= 12 {
			lulus++
		} else {
			naik["grade-"+strconv.Itoa(int(e.Grade))+"-to-"+strconv.Itoa(int(e.Grade)+1)]++
		}
	}
	result, _ := json.Marshal(map[string]any{
		"naik": naik, "lulus": lulus, "total": len(enrolls),
		"warnings": []string{"Dibuat otomatis oleh cron tanggal kenaikan kelas."},
	})
	id, err := q.CreatePromotionJob(ctx, repo.CreatePromotionJobParams{
		FromYearID: ay.ID, ToYearID: next.ID, DryrunResult: result,
	})
	if err != nil {
		return err
	}
	admins, _ := q.ListAdminUserIDs(ctx)
	body, _ := json.Marshal(map[string]any{"job_id": uuidStr(id), "from": ay.StartYear, "to": next.StartYear})
	for _, adminID := range admins {
		_, _ = q.CreateNotification(ctx, repo.CreateNotificationParams{
			UserID: adminID, Type: "announcement",
			Title: "Draft kenaikan kelas siap",
			Body:  "Preview kenaikan kelas dibuat otomatis. Tinjau lalu setujui di halaman Promosi.",
			Data:  body,
		})
	}
	slog.Info("promotion draft dibuat", "job", uuidStr(id), "admins", len(admins))
	return nil
}
