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
	naik := map[string]int{}
	lulus := 0
	warn := []string{}
	for _, e := range enrolls {
		grade := e.Grade
		if grade == 12 {
			lulus++
		} else {
			key := "grade-" + itoa(int(grade)) + "-to-" + itoa(int(grade)+1)
			naik[key]++
		}
	}
	if len(enrolls) == 0 {
		warn = append(warn, "Tidak ada enrollment aktif di tahun asal.")
	}
	result := map[string]any{"naik": naik, "lulus": lulus, "total": len(enrolls), "warnings": warn}
	b, _ := json.Marshal(result)
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
	b, _ := json.Marshal(req.Exceptions)
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
	if req.Confirm == "" {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Ketik konfirmasi tahun.", r))
		return
	}
	if err := a.executePromotion(r, job); err != nil {
		_ = err
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Eksekusi gagal.", r))
		return
	}
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "done"}, r))
}

func (a *App) executePromotion(r *http.Request, job repo.GetPromotionJobRow) error {
	ctx := r.Context()
	if err := func() error {
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
		classCache := map[string]interface{ ID() string }{}
		_ = classCache
		newClassID := map[string]string{}
		getClass := func(majorID string, grade int, group string) string {
			key := majorID + "|" + itoa(grade) + "|" + group
			if id, ok := newClassID[key]; ok {
				return id
			}
			return ""
		}
		_ = getClass
		for _, e := range enrolls {
			uid := uuidStr(e.UserID)
			action := excMap[uid]
			grade := int(e.Grade)
			if action == "repeat" {
				// tinggal kelas: grade sama, tahun baru
				cid, err := qtx.CreateClassForYear(ctx, repo.CreateClassForYearParams{
					AcademicYearID: job.ToYearID, MajorID: e.MajorID,
					Grade: int16(grade), ClassGroup: "A",
					Name: classNameFor(int32(grade), codeByID[uuidStr(e.MajorID)], "A"),
				})
				if err != nil {
					return err
				}
				_ = qtx.DeactivateEnrollments(ctx, repo.DeactivateEnrollmentsParams{UserID: e.UserID, AcademicYearID: job.FromYearID})
				_ = qtx.UpsertEnrollment(ctx, repo.UpsertEnrollmentParams{
					UserID: e.UserID, ClassID: cid, AcademicYearID: job.ToYearID, Status: "repeated",
				})
				continue
			}
			if grade >= 12 {
				_ = qtx.DeactivateEnrollments(ctx, repo.DeactivateEnrollmentsParams{UserID: e.UserID, AcademicYearID: job.FromYearID})
				_ = qtx.UpsertEnrollment(ctx, repo.UpsertEnrollmentParams{
					UserID: e.UserID, ClassID: e.ClassID, AcademicYearID: job.ToYearID, Status: "graduated",
				})
				_ = qtx.SetUserStatus(ctx, repo.SetUserStatusParams{ID: e.UserID, Status: "lulus"})
				_, _ = qtx.EnqueueOutbox(ctx, repo.EnqueueOutboxParams{
					Topic:   "notify.announcement",
					Payload: notifPayload(map[string]string{"user_id": uid, "kind": "promotion_result", "message": "Akun alumni aktif."}),
				})
				continue
			}
			cid, err := qtx.CreateClassForYear(ctx, repo.CreateClassForYearParams{
				AcademicYearID: job.ToYearID, MajorID: e.MajorID,
				Grade: int16(grade + 1), ClassGroup: "A",
				Name: classNameFor(int32(grade+1), codeByID[uuidStr(e.MajorID)], "A"),
			})
			if err != nil {
				return err
			}
			_ = qtx.DeactivateEnrollments(ctx, repo.DeactivateEnrollmentsParams{UserID: e.UserID, AcademicYearID: job.FromYearID})
			_ = qtx.UpsertEnrollment(ctx, repo.UpsertEnrollmentParams{
				UserID: e.UserID, ClassID: cid, AcademicYearID: job.ToYearID, Status: "promoted",
			})
			_, _ = qtx.EnqueueOutbox(ctx, repo.EnqueueOutboxParams{
				Topic:   "notify.announcement",
				Payload: notifPayload(map[string]string{"user_id": uid, "kind": "promotion_result", "message": "Kamu naik kelas."}),
			})
		}
		_ = qtx.SetActiveYear(ctx, job.ToYearID)
		return tx.Commit(ctx)
	}(); err != nil {
		return err
	}
	_ = a.Q.MarkPromotionDone(r.Context(), repo.MarkPromotionDoneParams{ID: job.ID, ExecutedBy: mustUUID(UserID(r))})
	auditLog(r, a, "promotion.execute", "promotion_jobs", uuidStr(job.ID), "")
	return nil
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
