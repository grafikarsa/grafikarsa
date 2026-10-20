package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"grafikarsa/api/internal/repo"
)

func (a *App) handlePromotionPreview(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromYearID string `json:"from_year_id"`
		ToYearID   string `json:"to_year_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	enrolls, err := a.Q.ListEnrollmentsByYear(r.Context(), mustUUID(req.FromYearID))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal preview.", r))
		return
	}
	counts := map[string]int{}
	lulus := 0
	warn := []string{}
	classSize := map[string]int{}
	for _, e := range enrolls {
		classSize[e.ClassName]++
	}
	for _, e := range enrolls {
		if e.Grade >= 12 {
			lulus++
			continue
		}
		key := e.ClassName + " -> " + classNameFor(int32(e.Grade+1), "", e.ClassGroup)
		counts[key]++
	}
	for name, n := range classSize {
		if n > 0 && n < 5 {
			warn = append(warn, name+" hanya "+itoa(n)+" siswa (pertimbangkan merge).")
		}
	}
	if len(enrolls) == 0 {
		warn = append(warn, "Tidak ada enrollment aktif di tahun asal.")
	}
	result := map[string]any{
		"naik": counts, "lulus": lulus, "total": len(enrolls),
		"kelas_baru_dibuat": len(counts), "warnings": warn,
	}
	b := toJSONBytes(result)
	id, err := a.Q.CreatePromotionJob(r.Context(), repo.CreatePromotionJobParams{
		FromYearID: mustUUID(req.FromYearID), ToYearID: mustUUID(req.ToYearID), DryrunResult: b,
	})
	if err != nil {
		writeJSON(w, http.StatusConflict, fail("DUPLICATE_JOB", "Job untuk pasangan tahun ini sudah ada.", r))
		return
	}
	auditLog(r, a, "promotion.preview", "promotion_jobs", uuidStr(id), "")
	writeJSON(w, http.StatusCreated, ok(map[string]any{"id": uuidStr(id), "preview": result}, r))
}

func (a *App) handlePromotionJobs(w http.ResponseWriter, r *http.Request) {
	rows, err := a.Q.ListPromotionJobs(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat.", r))
		return
	}
	out := []map[string]any{}
	for _, j := range rows {
		out = append(out, map[string]any{
			"id": uuidStr(j.ID), "status": j.Status,
			"executed_at": j.ExecutedAt.Time,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, ok(out, r))
}

func (a *App) handlePromotionApprove(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Exceptions []map[string]string `json:"exceptions"`
	}
	_ = req
	if !decodeJSON(w, r, &req) {
		return
	}
	b := toJSONBytes(req.Exceptions)
	if b == nil {
		b = []byte("[]")
	}
	_ = a.Q.ApprovePromotionJob(r.Context(), repo.ApprovePromotionJobParams{ID: mustUUID(id), Exceptions: b})
	auditLog(r, a, "promotion.approve", "promotion_jobs", id, "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "approved"}, r))
}

func (a *App) handlePromotionExecute(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Confirm string `json:"confirm"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	job, err := a.Q.GetPromotionJob(r.Context(), mustUUID(id))
	if err != nil || job.Status != "approved" {
		writeJSON(w, http.StatusUnprocessableEntity, fail("INVALID_STATUS_TRANSITION", "Job harus approved dulu.", r))
		return
	}
	// Konfirmasi wajib sama persis dengan tahun tujuan (bukan sekadar non-empty).
	target, err := a.Q.GetYearByID(r.Context(), job.ToYearID)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, fail("NOT_FOUND", "Tahun tujuan tidak ditemukan.", r))
		return
	}
	if req.Confirm != itoa(int(target.StartYear)) {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Konfirmasi salah. Ketik "+itoa(int(target.StartYear))+" untuk melanjutkan.", r))
		return
	}
	if err := a.Q.MarkPromotionExecuting(r.Context(), job.ID); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, fail("INVALID_STATUS_TRANSITION", "Job tidak bisa dieksekusi (sudah berjalan/selesai).", r))
		return
	}
	if err := a.executePromotion(r, job, req.Confirm); err != nil {
		_ = a.Q.MarkPromotionFailed(r.Context(), job.ID)
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Eksekusi kenaikan kelas gagal. Cek log server.", r))
		return
	}
	_ = a.Q.MarkPromotionDone(r.Context(), repo.MarkPromotionDoneParams{ID: job.ID, ExecutedBy: mustUUID(UserID(r))})
	auditLog(r, a, "promotion.execute", "promotion_jobs", uuidStr(job.ID), "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "done"}, r))
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (a *App) executePromotion(r *http.Request, job repo.GetPromotionJobRow, _ string) error {
	ctx := r.Context()
	pool := a.Tx.pool
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	qtx := a.Q.WithTx(tx)

	enrolls, err := qtx.ListEnrollmentsByYear(ctx, job.FromYearID)
	if err != nil {
		return err
	}
	var exceptions []map[string]string
	_ = json.Unmarshal(job.Exceptions, &exceptions)
	excMap := map[string]string{}
	for _, e := range exceptions {
		excMap[e["user_id"]] = e["action"]
	}

	majors, _ := qtx.ListMajorsFull(ctx)
	codeByID := map[string]string{}
	for _, m := range majors {
		codeByID[uuidStr(m.ID)] = m.Code
	}

	for _, e := range enrolls {
		uid := uuidStr(e.UserID)
		action := excMap[uid]
		grade := int(e.Grade)

		if action == "transfer" {
			// Pindahan: status 'transferred' (bukan graduated) dan tidak ikut naik.
			_ = qtx.SetEnrollmentStatus(ctx, repo.SetEnrollmentStatusParams{
				UserID: e.UserID, AcademicYearID: job.FromYearID, Status: "transferred",
			})
			continue
		}
		if action == "repeat" {
			// Tinggal kelas: grade sama, rombel dipertahankan.
			cid, cerr := qtx.CreateClassForYear(ctx, repo.CreateClassForYearParams{
				AcademicYearID: job.ToYearID, MajorID: e.MajorID,
				Grade: int16(grade), ClassGroup: e.ClassGroup,
				Name: classNameFor(int32(grade), codeByID[uuidStr(e.MajorID)], e.ClassGroup),
			})
			if cerr != nil {
				return cerr
			}
			_ = qtx.DeactivateEnrollments(ctx, repo.DeactivateEnrollmentsParams{UserID: e.UserID, AcademicYearID: job.FromYearID})
			_ = qtx.UpsertEnrollment(ctx, repo.UpsertEnrollmentParams{
				UserID: e.UserID, ClassID: cid, AcademicYearID: job.ToYearID, Status: "repeated",
			})
			_, _ = qtx.EnqueueOutbox(ctx, repo.EnqueueOutboxParams{
				Topic:   "notify.announcement",
				Payload: notifPayload(map[string]string{"user_id": uid, "kind": "promotion_result", "message": "Kamu tinggal kelas di " + classNameFor(int32(grade), codeByID[uuidStr(e.MajorID)], e.ClassGroup) + "."}),
			})
			continue
		}
		if grade >= 12 {
			// Lulus: enrollment lama jadi graduated & tidak current; TIDAK
			// dibuatkan enrollment tahun baru (alumni bukan siswa aktif).
			_ = qtx.GraduateEnrollment(ctx, repo.GraduateEnrollmentParams{UserID: e.UserID, AcademicYearID: job.FromYearID})
			_ = qtx.SetUserStatus(ctx, repo.SetUserStatusParams{ID: e.UserID, Status: "lulus"})
			_, _ = qtx.EnqueueOutbox(ctx, repo.EnqueueOutboxParams{
				Topic:   "notify.announcement",
				Payload: notifPayload(map[string]string{"user_id": uid, "kind": "promotion_result", "message": "Selamat! Akun alumnimu aktif."}),
			})
			continue
		}
		// Naik kelas: rombel (class_group) dipertahankan.
		cid, cerr := qtx.CreateClassForYear(ctx, repo.CreateClassForYearParams{
			AcademicYearID: job.ToYearID, MajorID: e.MajorID,
			Grade: int16(grade + 1), ClassGroup: e.ClassGroup,
			Name: classNameFor(int32(grade+1), codeByID[uuidStr(e.MajorID)], e.ClassGroup),
		})
		if cerr != nil {
			return cerr
		}
		_ = qtx.DeactivateEnrollments(ctx, repo.DeactivateEnrollmentsParams{UserID: e.UserID, AcademicYearID: job.FromYearID})
		_ = qtx.UpsertEnrollment(ctx, repo.UpsertEnrollmentParams{
			UserID: e.UserID, ClassID: cid, AcademicYearID: job.ToYearID, Status: "promoted",
		})
		_, _ = qtx.EnqueueOutbox(ctx, repo.EnqueueOutboxParams{
			Topic:   "notify.announcement",
			Payload: notifPayload(map[string]string{"user_id": uid, "kind": "promotion_result", "message": "Kamu naik ke " + classNameFor(int32(grade+1), codeByID[uuidStr(e.MajorID)], e.ClassGroup) + "."}),
		})
	}
	_ = qtx.SetActiveYear(ctx, job.ToYearID)
	return tx.Commit(ctx)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [16]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
