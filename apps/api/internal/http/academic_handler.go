package http

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"grafikarsa/api/internal/repo"
)

func romanGrade(g int) string {
	switch g {
	case 10:
		return "X"
	case 11:
		return "XI"
	case 12:
		return "XII"
	}
	return ""
}

func (a *App) handleListMajors(w http.ResponseWriter, r *http.Request) {
	rows, err := a.Q.ListMajorsFull(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat jurusan.", r))
		return
	}
	out := []map[string]any{}
	for _, m := range rows {
		out = append(out, map[string]any{"id": uuidStr(m.ID), "code": m.Code, "name": m.Name, "require_series": m.RequireSeries})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, ok(out, r))
}

func (a *App) handleCreateMajor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Code = strings.ToLower(req.Code)
	if len(req.Code) < 2 || len(req.Name) < 3 {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Kode + nama tidak valid.", r))
		return
	}
	id, err := a.Q.CreateMajor(r.Context(), repo.CreateMajorParams{Code: req.Code, Name: req.Name})
	if err != nil {
		writeJSON(w, http.StatusConflict, fail("DUPLICATE_MAJOR", "Kode jurusan sudah dipakai.", r))
		return
	}
	auditLog(r, a, "majors.create", "majors", uuidStr(id), req.Code)
	writeJSON(w, http.StatusCreated, ok(map[string]string{"id": uuidStr(id)}, r))
}

func (a *App) handleListYears(w http.ResponseWriter, r *http.Request) {
	rows, err := a.Q.ListYears(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat tahun.", r))
		return
	}
	out := []map[string]any{}
	for _, y := range rows {
		out = append(out, map[string]any{"id": uuidStr(y.ID), "start_year": y.StartYear, "is_active": y.IsActive})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, ok(out, r))
}

func (a *App) handleCreateYear(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StartYear int `json:"start_year"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.StartYear < 2000 || req.StartYear > 2100 {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Tahun tidak valid.", r))
		return
	}
	id, err := a.Q.CreateYear(r.Context(), repo.CreateYearParams{StartYear: int32(req.StartYear)})
	if err != nil {
		writeJSON(w, http.StatusConflict, fail("DUPLICATE_YEAR", "Tahun sudah ada.", r))
		return
	}
	auditLog(r, a, "years.create", "academic_years", uuidStr(id), "")
	writeJSON(w, http.StatusCreated, ok(map[string]string{"id": uuidStr(id)}, r))
}

func (a *App) handleActivateYear(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_ = a.Q.SetActiveYear(r.Context(), mustUUID(id))
	auditLog(r, a, "years.activate", "academic_years", id, "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleListClasses(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, offset, page := pageParams(r)
	year, major := q.Get("year_id"), q.Get("major_id")
	rows, err := a.Q.ListClasses(r.Context(), repo.ListClassesParams{Column1: year, Column2: major, Limit: limit, Offset: offset})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat kelas.", r))
		return
	}
	// Scope admin_jurusan: WAJIB punya scope (fail-closed), lalu filter hanya
	// kelas di jurusannya. Tanpa filter, admin_jurusan melihat semua kelas.
	if UserRole(r) == "admin_jurusan" {
		scope := UserScope(r)
		if scope == "" {
			writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Scope jurusan belum diset.", r))
			return
		}
		filtered := []repo.ListClassesRow{}
		for _, c := range rows {
			if uuidStr(c.MajorID) == scope {
				filtered = append(filtered, c)
			}
		}
		rows = filtered
	}
	out := []map[string]any{}
	for _, c := range rows {
		n, _ := a.Q.CountClassStudents(r.Context(), c.ID)
		out = append(out, map[string]any{
			"id": uuidStr(c.ID), "name": c.Name, "grade": c.Grade,
			"major_code": c.MajorCode, "start_year": c.StartYear, "students": n,
		})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, int64(len(out))), r))
}

func (a *App) handleCreateClass(w http.ResponseWriter, r *http.Request) {
	var req struct {
		YearID  string `json:"year_id"`
		MajorID string `json:"major_id"`
		Grade   int    `json:"grade"`
		Group   string `json:"group"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Grade < 10 || req.Grade > 12 || len(req.Group) != 1 {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Grade 10-12 + rombel A-Z.", r))
		return
	}
	if UserRole(r) == "admin_jurusan" {
		// Fail-closed: wajib punya scope DAN kelas harus di jurusan itu.
		if UserScope(r) == "" || req.MajorID != UserScope(r) {
			writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Di luar jurusanmu.", r))
			return
		}
	}
	majors, _ := a.Q.ListMajorsFull(r.Context())
	code := ""
	for _, m := range majors {
		if uuidStr(m.ID) == req.MajorID {
			code = m.Code
		}
	}
	name := romanGrade(req.Grade) + "-" + strings.ToUpper(code) + "-" + strings.ToUpper(req.Group)
	id, err := a.Q.CreateClass(r.Context(), repo.CreateClassParams{
		AcademicYearID: mustUUID(req.YearID), MajorID: mustUUID(req.MajorID),
		Grade: int16(req.Grade), ClassGroup: req.Group, Name: name,
	})
	if err != nil {
		writeJSON(w, http.StatusConflict, fail("DUPLICATE_CLASS", "Kelas sudah ada.", r))
		return
	}
	auditLog(r, a, "classes.create", "classes", uuidStr(id), name)
	writeJSON(w, http.StatusCreated, ok(map[string]string{"id": uuidStr(id), "name": name}, r))
}

func (a *App) handleDeleteClass(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	n, _ := a.Q.CountClassStudents(r.Context(), mustUUID(id))
	if n > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, fail("CLASS_NOT_EMPTY", "Kelas masih ada siswa.", r))
		return
	}
	_ = a.Q.DeleteClass(r.Context(), mustUUID(id))
	auditLog(r, a, "classes.delete", "classes", id, "")
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "deleted"}, r))
}

func (a *App) handleClassStudents(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := a.Q.ListClassStudents(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat.", r))
		return
	}
	out := []map[string]any{}
	for _, u := range rows {
		out = append(out, map[string]any{"id": uuidStr(u.ID), "username": u.Username, "name": u.Name})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, ok(out, r))
}

func (a *App) handleEnrollStudent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID  string `json:"user_id"`
		ClassID string `json:"class_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	u, err := a.Q.GetUserByID(r.Context(), mustUUID(req.UserID))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "User tidak ditemukan.", r))
		return
	}
	classes, _ := a.Q.ListClasses(r.Context(), repo.ListClassesParams{Limit: 1, Offset: 0})
	_ = classes
	// Ambil tahun dari kelas target via ListClassesRow? Sederhana: enroll ke tahun aktif.
	year, err := a.Q.GetActiveYear(r.Context())
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, fail("NO_ACTIVE_YEAR", "Belum ada tahun aktif.", r))
		return
	}
	_ = u
	_ = a.Q.UpsertEnrollment(r.Context(), repo.UpsertEnrollmentParams{
		UserID: mustUUID(req.UserID), ClassID: mustUUID(req.ClassID),
		AcademicYearID: year.ID, Status: "enrolled",
	})
	auditLog(r, a, "enrollments.upsert", "enrollments", req.UserID, req.ClassID)
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func classNameFor(grade int32, code, group string) string {
	return romanGrade(int(grade)) + "-" + strings.ToUpper(code) + "-" + strings.ToUpper(group)
}
