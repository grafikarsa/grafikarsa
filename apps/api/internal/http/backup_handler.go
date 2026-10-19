package http

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// handleBackupDump: backup database manual (permission system.backup).
//
// Menjalankan pg_dump ke direktori backup yang persisten. Dipakai saat
// admin butuh titik restore tambahan di luar jadwal cron harian.
//
// Hanya database yang dibackup di sini. Object storage ikut dicadangkan
// oleh tools/backup.sh di host (rclone sync) karena mirror-nya berjalan di
// level host, bukan di dalam container API.
func (a *App) handleBackupDump(w http.ResponseWriter, r *http.Request) {
	dir := a.Cfg.Backup.Dir
	if dir == "" {
		writeJSON(w, http.StatusServiceUnavailable, fail("SERVICE_UNAVAILABLE", "Backup tidak dikonfigurasi.", r))
		return
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal menyiapkan direktori backup.", r))
		return
	}
	stamp := time.Now().UTC().Format("20060102-150405")
	out := filepath.Join(dir, fmt.Sprintf("db-%s.dump", stamp))

	timeout := a.Cfg.Backup.TimeoutPgDump
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	// PGPASSWORD dibaca dari env DB_PASSWORD supaya password tidak muncul
	// di daftar proses (ps) — pg_dump tidak punya flag password.
	cmd := exec.CommandContext(ctx, "pg_dump",
		"-Fc", // format custom (compressed)
		"-h", a.Cfg.DB.Host,
		"-p", fmt.Sprintf("%d", a.Cfg.DB.Port),
		"-U", a.Cfg.DB.User,
		"-d", a.Cfg.DB.Name,
		"-f", out,
	)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+a.Cfg.DB.Password)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		_ = os.Remove(out) // jangan sisakan file setengah jadi
		msg := strings.TrimSpace(stderr.String())
		if len(msg) > 400 {
			msg = msg[:400]
		}
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Backup gagal: "+msg, r))
		return
	}

	info, err := os.Stat(out)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Backup gagal: file tidak ditemukan.", r))
		return
	}
	auditLog(r, a, "system.backup", "database", a.Cfg.DB.Name, out)
	writeJSON(w, http.StatusOK, ok(map[string]any{
		"status":   "ok",
		"file":     filepath.Base(out),
		"path":     out,
		"bytes":    info.Size(),
		"database": a.Cfg.DB.Name,
	}, r))
}
