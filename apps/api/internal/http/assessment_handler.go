package http

import (
	"crypto/rand"
	"encoding/csv"
	"math/big"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"grafikarsa/api/internal/auth"
	"grafikarsa/api/internal/repo"
)

func (a *App) handleListMetrics(w http.ResponseWriter, r *http.Request) {
	rows, err := a.Q.ListMetrics(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat metrik.", r))
		return
	}
	out := []map[string]any{}
	for _, m := range rows {
		out = append(out, map[string]any{"id": uuidStr(m.ID), "name": m.Name, "sort": m.Sort, "is_active": m.IsActive})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, ok(out, r))
}

func (a *App) handleCreateMetric(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Sort int    `json:"sort"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Name) < 3 {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Nama minimal 3 karakter.", r))
		return
	}
	id, err := a.Q.CreateMetric(r.Context(), repo.CreateMetricParams{Name: req.Name, Sort: int32(req.Sort)})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal membuat metrik.", r))
		return
	}
	auditLog(r, a, "metrics.create", "assessment_metrics", uuidStr(id), req.Name)
	writeJSON(w, http.StatusCreated, ok(map[string]string{"id": uuidStr(id)}, r))
}

func (a *App) handleGetAssessment(w http.ResponseWriter, r *http.Request) {
	pid := chi.URLParam(r, "id")
	p, err := a.Q.GetPortfolioByID(r.Context(), mustUUID(pid))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
		return
	}
	ass, err := a.Q.GetAssessment(r.Context(), p.ID)
	if err != nil {
		writeJSON(w, http.StatusOK, ok(map[string]any{"portfolio_id": pid, "assessment": nil}, r))
		return
	}
	scores, _ := a.Q.ListScores(r.Context(), ass.ID)
	writeJSON(w, http.StatusOK, ok(map[string]any{
		"portfolio_id": pid, "total_score": ass.TotalScore,
		"comment": ass.Comment, "scores": scores,
	}, r))
}

func (a *App) handleScoreAssessment(w http.ResponseWriter, r *http.Request) {
	pid := chi.URLParam(r, "id")
	p, err := a.Q.GetPortfolioByID(r.Context(), mustUUID(pid))
	if err != nil || p.Status != "published" {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
		return
	}
	var req struct {
		Scores []struct {
			MetricID string `json:"metric_id"`
			Score    int    `json:"score"`
			Comment  string `json:"comment"`
		} `json:"scores"`
		Comment string `json:"comment"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Scores) == 0 {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Minimal 1 skor.", r))
		return
	}
	total := 0
	for _, s := range req.Scores {
		if s.Score < 1 || s.Score > 10 {
			writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Skor 1-10.", r))
			return
		}
		total += s.Score
	}
	avg := float64(total) / float64(len(req.Scores))
	assID, err := a.Q.UpsertAssessment(r.Context(), repo.UpsertAssessmentParams{
		PortfolioID: p.ID, AssessorID: mustUUID(UserID(r)),
		TotalScore: floatToNumeric(avg), Comment: req.Comment,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal menyimpan nilai.", r))
		return
	}
	_ = a.Q.DeleteScores(r.Context(), assID)
	for _, s := range req.Scores {
		_ = a.Q.AddScore(r.Context(), repo.AddScoreParams{
			AssessmentID: assID, MetricID: mustUUID(s.MetricID),
			Score: int16(s.Score), Comment: s.Comment,
		})
	}
	_ = a.Q.UpsertPortfolioStats(r.Context(), repo.UpsertPortfolioStatsParams{
		PortfolioID: p.ID, AssessmentAvg: floatToNumeric(avg),
	})
	auditLog(r, a, "assessments.score", "portfolios", pid, "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleImportStudents(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, fail("FILE_TOO_LARGE", "Maksimal 5MB.", r))
		return
	}
	f, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, fail("VALIDATION_ERROR", "Field file wajib.", r))
		return
	}
	defer f.Close()
	reader := csv.NewReader(f)
	rows, err := reader.ReadAll()
	if err != nil || len(rows) < 2 {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "CSV tidak valid (header + isi).", r))
		return
	}
	dryRun := r.URL.Query().Get("dry_run") == "true"
	created, errors := 0, []string{}
	for i, row := range rows[1:] {
		if len(row) < 4 {
			errors = append(errors, "baris "+itoa(i+2)+": kolom kurang")
			continue
		}
		username, email, name, role := strings.TrimSpace(row[0]), strings.TrimSpace(row[1]), strings.TrimSpace(row[2]), strings.TrimSpace(row[3])
		if !usernameRe.MatchString(username) || name == "" || usernameReserved(username) {
			errors = append(errors, "baris "+itoa(i+2)+": username/nama invalid atau dicadangkan")
			continue
		}
		if role != "siswa" && role != "guru" {
			role = "siswa"
		}
		if dryRun {
			created++
			continue
		}
		pw := randomPassword(12)
		_, err := a.Q.CreateUser(r.Context(), repo.CreateUserParams{
			Username: username, Email: email, PasswordHash: auth.HashPassword(pw),
			Name: name, RolePrimary: role, Status: "aktif", MustChangePassword: true,
		})
		if err != nil {
			errors = append(errors, "baris "+itoa(i+2)+": duplikat/gagal")
			continue
		}
		created++
	}
	writeJSON(w, http.StatusOK, ok(map[string]any{"created": created, "errors": errors, "dry_run": dryRun}, r))
}

// randomPassword: crypto/rand + pastikan ada huruf besar/kecil/angka
// agar lolos passwordStrong().
func randomPassword(n int) string {
	const lower = "abcdefghijklmnopqrstuvwxyz"
	const upper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const digits = "0123456789"
	const all = lower + upper + digits
	pick := func(set string) byte {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(set))))
		if err != nil {
			return set[0]
		}
		return set[idx.Int64()]
	}
	if n < 4 {
		n = 12
	}
	b := make([]byte, n)
	b[0], b[1], b[2] = pick(lower), pick(upper), pick(digits)
	for i := 3; i < n; i++ {
		b[i] = pick(all)
	}
	for i := len(b) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			break
		}
		k := j.Int64()
		b[i], b[k] = b[k], b[i]
	}
	return string(b)
}
