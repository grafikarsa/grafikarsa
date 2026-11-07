package academic_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/academic"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

const testActorIP = "203.0.113.7"

type fixture struct {
	t       *testing.T
	ctx     context.Context
	pool    *pgxpool.Pool
	store   *store.Store
	service *academic.Service
	actorID uuid.UUID
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	pool := testutil.NewSeededDB(t)
	f := &fixture{
		t:       t,
		ctx:     context.Background(),
		pool:    pool,
		store:   store.NewStore(pool),
		service: academic.NewService(store.NewStore(pool)),
	}
	err := pool.QueryRow(f.ctx, `
		INSERT INTO users (account_type, display_name, password_hash, password_kind)
		VALUES ('staff', 'Pengelola Akademik', 'x', 'user')
		RETURNING id`,
	).Scan(&f.actorID)
	if err != nil {
		t.Fatalf("buat user aktor: %v", err)
	}
	return f
}

func (f *fixture) majorID(code string) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	if err := f.pool.QueryRow(f.ctx, "SELECT id FROM majors WHERE code = $1", code).Scan(&id); err != nil {
		f.t.Fatalf("ambil jurusan %s: %v", code, err)
	}
	return id
}

func (f *fixture) actor(grants ...auth.Grant) academic.Actor {
	return academic.Actor{
		Principal: &auth.Principal{UserID: f.actorID, Grants: grants},
		IP:        testActorIP,
	}
}

func (f *fixture) globalActor(permission string) academic.Actor {
	return f.actor(auth.Grant{Permission: permission, Global: true})
}

func (f *fixture) scopedActor(permission string, majorID uuid.UUID) academic.Actor {
	return f.actor(auth.Grant{Permission: permission, MajorIDs: []uuid.UUID{majorID}})
}

// insertYear creates an academic year directly, bypassing the service, for
// tests that need specific dates.
func (f *fixture) insertYear(startYear int16, startsOn, endsOn string) uuid.UUID {
	f.t.Helper()
	var id uuid.UUID
	err := f.pool.QueryRow(f.ctx, `
		INSERT INTO academic_years (start_year, starts_on, ends_on)
		VALUES ($1, $2, $3)
		RETURNING id`,
		startYear, startsOn, endsOn,
	).Scan(&id)
	if err != nil {
		f.t.Fatalf("buat tahun ajaran: %v", err)
	}
	return id
}

func (f *fixture) majorGroupStatus(majorID uuid.UUID) (string, *time.Time) {
	f.t.Helper()
	row, err := f.store.GetMajorGroupByMajorID(f.ctx, majorID)
	if err != nil {
		f.t.Fatalf("ambil grup jurusan: %v", err)
	}
	return row.Status, row.HistorySince
}

func (f *fixture) classGroupStatus(classID uuid.UUID) (string, *time.Time) {
	f.t.Helper()
	row, err := f.store.GetClassGroupByClassID(f.ctx, classID)
	if err != nil {
		f.t.Fatalf("ambil grup kelas: %v", err)
	}
	return row.Status, row.HistorySince
}

func (f *fixture) auditActions() []string {
	f.t.Helper()
	rows, err := f.pool.Query(f.ctx, "SELECT action FROM audit_logs ORDER BY id")
	if err != nil {
		f.t.Fatalf("ambil audit log: %v", err)
	}
	defer rows.Close()

	var actions []string
	for rows.Next() {
		var action string
		if err := rows.Scan(&action); err != nil {
			f.t.Fatalf("baca audit log: %v", err)
		}
		actions = append(actions, action)
	}
	if err := rows.Err(); err != nil {
		f.t.Fatalf("iterasi audit log: %v", err)
	}
	return actions
}

func (f *fixture) hasAuditAction(action string) bool {
	f.t.Helper()
	for _, got := range f.auditActions() {
		if got == action {
			return true
		}
	}
	return false
}

func requireAppError(t *testing.T, err error, code string) *apperr.Error {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, mau kode %s", code)
	}
	e, ok := apperr.As(err)
	if !ok {
		t.Fatalf("error = %v, mau *apperr.Error dengan kode %s", err, code)
	}
	if e.Code != code {
		t.Fatalf("kode error = %s (%v), mau %s", e.Code, err, code)
	}
	return e
}

func requireFieldError(t *testing.T, err error, field string) apperr.FieldError {
	t.Helper()
	e := requireAppError(t, err, "VALIDATION_FAILED")
	for _, f := range e.Fields {
		if f.Field == field {
			return f
		}
	}
	t.Fatalf("field %q tidak ada di error validasi: %+v", field, e.Fields)
	return apperr.FieldError{}
}

func TestClassName(t *testing.T) {
	tests := []struct {
		grade   int16
		short   string
		section string
		want    string
	}{
		{10, "RPL", "A", "X RPL A"},
		{11, "RPL", "B", "XI RPL B"},
		{12, "DKV", "C", "XII DKV C"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := academic.ClassName(tt.grade, tt.short, tt.section); got != tt.want {
				t.Fatalf("ClassName(%d, %q, %q) = %q, mau %q", tt.grade, tt.short, tt.section, got, tt.want)
			}
		})
	}
}

func TestClassSlug(t *testing.T) {
	tests := []struct {
		grade   int16
		short   string
		section string
		want    string
	}{
		{10, "RPL", "A", "x-rpl-a"},
		{11, "RPL", "B", "xi-rpl-b"},
		{12, "DKV", "C", "xii-dkv-c"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := academic.ClassSlug(tt.grade, tt.short, tt.section); got != tt.want {
				t.Fatalf("ClassSlug(%d, %q, %q) = %q, mau %q", tt.grade, tt.short, tt.section, got, tt.want)
			}
		})
	}
}

func TestYearLabel(t *testing.T) {
	if got := academic.YearLabel(2026); got != "2026/2027" {
		t.Fatalf("YearLabel(2026) = %q, mau %q", got, "2026/2027")
	}
}

func TestYearSlug(t *testing.T) {
	if got := academic.YearSlug(2026); got != "2026-2027" {
		t.Fatalf("YearSlug(2026) = %q, mau %q", got, "2026-2027")
	}
}

func TestCreateMajor_CreatesGroupAndAudit(t *testing.T) {
	f := newFixture(t)

	major, err := f.service.CreateMajor(f.ctx, f.globalActor("academic.manage_majors"), academic.CreateMajorInput{
		Code:      " TEST-MAJOR ",
		Name:      "Jurusan Uji",
		ShortName: "uji",
		SortOrder: 10,
	})
	if err != nil {
		t.Fatalf("CreateMajor: %v", err)
	}
	if major.Code != "test-major" {
		t.Errorf("Code = %q, mau %q", major.Code, "test-major")
	}
	if major.ShortName != "UJI" {
		t.Errorf("ShortName = %q, mau %q", major.ShortName, "UJI")
	}
	if major.GroupID == uuid.Nil {
		t.Fatal("GroupID kosong, grup jurusan tidak dibuat")
	}

	status, historySince := f.majorGroupStatus(major.ID)
	if status != "active" {
		t.Errorf("status grup = %q, mau active", status)
	}
	if historySince != nil {
		t.Errorf("history_since = %v, mau nil", historySince)
	}
	if !f.hasAuditAction("academic.major_create") {
		t.Error("audit academic.major_create tidak tercatat")
	}
}

func TestCreateMajor_DuplicateCodeReturnsConflict(t *testing.T) {
	f := newFixture(t)
	actor := f.globalActor("academic.manage_majors")
	in := academic.CreateMajorInput{Code: "dua-jurusan", Name: "Jurusan Satu", ShortName: "JS"}

	if _, err := f.service.CreateMajor(f.ctx, actor, in); err != nil {
		t.Fatalf("CreateMajor pertama: %v", err)
	}
	_, err := f.service.CreateMajor(f.ctx, actor, in)
	e := requireAppError(t, err, "CONFLICT")
	if e.Detail != "Kode jurusan sudah dipakai." {
		t.Errorf("Detail = %q, mau %q", e.Detail, "Kode jurusan sudah dipakai.")
	}
}

func TestCreateMajor_WithoutPermissionReturnsNotFound(t *testing.T) {
	f := newFixture(t)

	_, err := f.service.CreateMajor(f.ctx, f.actor(), academic.CreateMajorInput{
		Code: "tanpa-izin", Name: "Jurusan Tanpa Izin", ShortName: "TI",
	})
	requireAppError(t, err, "NOT_FOUND")
}

func TestUpdateMajor_DeactivateAndReactivateTogglesGroupStatus(t *testing.T) {
	f := newFixture(t)
	actor := f.globalActor("academic.manage_majors")
	majorID := f.majorID("rpl")

	inactive := false
	major, err := f.service.UpdateMajor(f.ctx, actor, majorID, academic.UpdateMajorInput{IsActive: &inactive})
	if err != nil {
		t.Fatalf("UpdateMajor nonaktif: %v", err)
	}
	if major.IsActive {
		t.Error("IsActive = true, mau false")
	}
	status, historySince := f.majorGroupStatus(majorID)
	if status != "history" {
		t.Errorf("status grup = %q, mau history", status)
	}
	if historySince == nil {
		t.Error("history_since = nil, mau terisi")
	}

	active := true
	major, err = f.service.UpdateMajor(f.ctx, actor, majorID, academic.UpdateMajorInput{IsActive: &active})
	if err != nil {
		t.Fatalf("UpdateMajor aktif: %v", err)
	}
	if !major.IsActive {
		t.Error("IsActive = false, mau true")
	}
	status, historySince = f.majorGroupStatus(majorID)
	if status != "active" {
		t.Errorf("status grup = %q, mau active", status)
	}
	if historySince != nil {
		t.Errorf("history_since = %v, mau nil", historySince)
	}
	if !f.hasAuditAction("academic.major_update") {
		t.Error("audit academic.major_update tidak tercatat")
	}
}

func TestCreateYear_DuplicateAndSetCurrent(t *testing.T) {
	f := newFixture(t)
	actor := f.globalActor("academic.manage_years")

	year, err := f.service.CreateYear(f.ctx, actor, academic.CreateYearInput{
		StartYear: 2030,
		StartsOn:  time.Date(2030, time.July, 1, 0, 0, 0, 0, time.UTC),
		EndsOn:    time.Date(2031, time.June, 30, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateYear: %v", err)
	}
	if year.Label != "2030/2031" || year.Slug != "2030-2031" {
		t.Errorf("Label/Slug = %q/%q, mau 2030/2031 dan 2030-2031", year.Label, year.Slug)
	}

	_, err = f.service.CreateYear(f.ctx, actor, academic.CreateYearInput{
		StartYear: 2030,
		StartsOn:  time.Date(2030, time.July, 2, 0, 0, 0, 0, time.UTC),
		EndsOn:    time.Date(2031, time.June, 30, 0, 0, 0, 0, time.UTC),
	})
	e := requireAppError(t, err, "CONFLICT")
	if e.Detail != "Tahun ajaran sudah ada." {
		t.Errorf("Detail = %q, mau %q", e.Detail, "Tahun ajaran sudah ada.")
	}

	later, err := f.service.CreateYear(f.ctx, actor, academic.CreateYearInput{
		StartYear: 2031,
		StartsOn:  time.Date(2031, time.July, 1, 0, 0, 0, 0, time.UTC),
		EndsOn:    time.Date(2032, time.June, 30, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateYear kedua: %v", err)
	}

	if err := f.service.SetCurrentYear(f.ctx, actor, year.ID); err != nil {
		t.Fatalf("SetCurrentYear: %v", err)
	}
	assertCurrentYear(t, f, year.ID)

	if err := f.service.SetCurrentYear(f.ctx, actor, later.ID); err != nil {
		t.Fatalf("SetCurrentYear kedua: %v", err)
	}
	assertCurrentYear(t, f, later.ID)

	if !f.hasAuditAction("academic.year_create") {
		t.Error("audit academic.year_create tidak tercatat")
	}
	if !f.hasAuditAction("academic.year_set_current") {
		t.Error("audit academic.year_set_current tidak tercatat")
	}
}

func assertCurrentYear(t *testing.T, f *fixture, want uuid.UUID) {
	t.Helper()
	rows, err := f.pool.Query(f.ctx, "SELECT id FROM academic_years WHERE is_current")
	if err != nil {
		t.Fatalf("ambil tahun ajaran aktif: %v", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("baca tahun ajaran aktif: %v", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterasi tahun ajaran aktif: %v", err)
	}
	if len(ids) != 1 || ids[0] != want {
		t.Fatalf("tahun ajaran aktif = %v, mau tepat satu %s", ids, want)
	}
}

func TestCreateClass_ScopedActorAndValidation(t *testing.T) {
	f := newFixture(t)
	yearID := f.insertYear(2030, "2030-07-01", "2031-06-30")
	rplID := f.majorID("rpl")
	tkjID := f.majorID("tkj")
	actor := f.scopedActor("academic.manage_classes", rplID)

	var class academic.Class
	t.Run("scoped actor creates RPL class", func(t *testing.T) {
		var err error
		class, err = f.service.CreateClass(f.ctx, actor, academic.CreateClassInput{
			AcademicYearID: yearID,
			MajorID:        rplID,
			Grade:          11,
			Section:        "b",
		})
		if err != nil {
			t.Fatalf("CreateClass: %v", err)
		}
		if class.Name != "XI RPL B" {
			t.Errorf("Name = %q, mau %q", class.Name, "XI RPL B")
		}
		if class.Slug != "xi-rpl-b" {
			t.Errorf("Slug = %q, mau %q", class.Slug, "xi-rpl-b")
		}
		if class.GroupID == uuid.Nil {
			t.Fatal("GroupID kosong, grup kelas tidak dibuat")
		}
		status, historySince := f.classGroupStatus(class.ID)
		if status != "active" || historySince != nil {
			t.Errorf("grup kelas = %q/%v, mau active/nil", status, historySince)
		}
		if !f.hasAuditAction("academic.class_create") {
			t.Error("audit academic.class_create tidak tercatat")
		}
	})

	t.Run("scoped actor cannot create another major", func(t *testing.T) {
		_, err := f.service.CreateClass(f.ctx, actor, academic.CreateClassInput{
			AcademicYearID: yearID,
			MajorID:        tkjID,
			Grade:          11,
			Section:        "A",
		})
		requireAppError(t, err, "NOT_FOUND")
	})

	t.Run("duplicate class returns conflict", func(t *testing.T) {
		_, err := f.service.CreateClass(f.ctx, actor, academic.CreateClassInput{
			AcademicYearID: yearID,
			MajorID:        rplID,
			Grade:          11,
			Section:        "B",
		})
		e := requireAppError(t, err, "CONFLICT")
		if e.Detail != "Kelas ini sudah ada di tahun ajaran tersebut." {
			t.Errorf("Detail = %q, mau %q", e.Detail, "Kelas ini sudah ada di tahun ajaran tersebut.")
		}
	})

	t.Run("grade 13 is rejected", func(t *testing.T) {
		_, err := f.service.CreateClass(f.ctx, f.scopedActor("academic.manage_classes", rplID), academic.CreateClassInput{
			AcademicYearID: yearID,
			MajorID:        rplID,
			Grade:          13,
			Section:        "A",
		})
		requireFieldError(t, err, "grade")
	})

	t.Run("validation section", func(t *testing.T) {
		_, err := f.service.CreateClass(f.ctx, f.scopedActor("academic.manage_classes", rplID), academic.CreateClassInput{
			AcademicYearID: yearID,
			MajorID:        rplID,
			Grade:          11,
			Section:        "b1",
		})
		requireFieldError(t, err, "section")
	})
}

func TestCreateClass_EndedYearMarksGroupHistory(t *testing.T) {
	f := newFixture(t)
	yearID := f.insertYear(2020, "2020-07-01", "2021-06-30")
	rplID := f.majorID("rpl")

	class, err := f.service.CreateClass(f.ctx, f.scopedActor("academic.manage_classes", rplID), academic.CreateClassInput{
		AcademicYearID: yearID,
		MajorID:        rplID,
		Grade:          10,
		Section:        "A",
	})
	if err != nil {
		t.Fatalf("CreateClass: %v", err)
	}

	status, historySince := f.classGroupStatus(class.ID)
	if status != "history" {
		t.Errorf("status grup = %q, mau history", status)
	}
	if historySince == nil {
		t.Error("history_since = nil, mau terisi")
	}
}

func TestDeleteClass_WithEnrollmentReturnsConflict(t *testing.T) {
	f := newFixture(t)
	yearID := f.insertYear(2030, "2030-07-01", "2031-06-30")
	rplID := f.majorID("rpl")
	actor := f.scopedActor("academic.manage_classes", rplID)

	class, err := f.service.CreateClass(f.ctx, actor, academic.CreateClassInput{
		AcademicYearID: yearID,
		MajorID:        rplID,
		Grade:          10,
		Section:        "A",
	})
	if err != nil {
		t.Fatalf("CreateClass: %v", err)
	}

	var studentID uuid.UUID
	err = f.pool.QueryRow(f.ctx, `
		INSERT INTO users (account_type, nisn, display_name, password_hash, password_kind)
		VALUES ('student', '1234567890', 'Siswa Uji', 'x', 'user')
		RETURNING id`,
	).Scan(&studentID)
	if err != nil {
		t.Fatalf("buat user siswa: %v", err)
	}
	if _, err := f.pool.Exec(f.ctx, `
		INSERT INTO enrollments (user_id, class_id, academic_year_id)
		VALUES ($1, $2, $3)`,
		studentID, class.ID, yearID,
	); err != nil {
		t.Fatalf("buat enrollment: %v", err)
	}

	err = f.service.DeleteClass(f.ctx, actor, class.ID)
	e := requireAppError(t, err, "CONFLICT")
	if e.Detail != "Kelas sudah punya siswa, tidak bisa dihapus." {
		t.Errorf("Detail = %q, mau %q", e.Detail, "Kelas sudah punya siswa, tidak bisa dihapus.")
	}

	if _, err := f.store.GetClassByID(f.ctx, class.ID); err != nil {
		t.Fatalf("kelas terhapus padahal punya siswa: %v", err)
	}

	if _, err := f.pool.Exec(f.ctx, "DELETE FROM enrollments WHERE class_id = $1", class.ID); err != nil {
		t.Fatalf("hapus enrollment: %v", err)
	}
	if err := f.service.DeleteClass(f.ctx, actor, class.ID); err != nil {
		t.Fatalf("DeleteClass: %v", err)
	}

	if _, err := f.store.GetClassByID(f.ctx, class.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetClassByID setelah hapus = %v, mau tidak ditemukan", err)
	}
	if _, err := f.store.GetClassGroupByClassID(f.ctx, class.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetClassGroupByClassID setelah hapus = %v, mau tidak ditemukan", err)
	}
	if !f.hasAuditAction("academic.class_delete") {
		t.Error("audit academic.class_delete tidak tercatat")
	}
}

func TestListClasses_Filters(t *testing.T) {
	f := newFixture(t)
	actor := f.globalActor("academic.manage_classes")
	yearOne := f.insertYear(2030, "2030-07-01", "2031-06-30")
	yearTwo := f.insertYear(2031, "2031-07-01", "2032-06-30")
	rplID := f.majorID("rpl")
	tkjID := f.majorID("tkj")

	create := func(yearID, majorID uuid.UUID, section string) academic.Class {
		t.Helper()
		class, err := f.service.CreateClass(f.ctx, actor, academic.CreateClassInput{
			AcademicYearID: yearID,
			MajorID:        majorID,
			Grade:          10,
			Section:        section,
		})
		if err != nil {
			t.Fatalf("CreateClass: %v", err)
		}
		return class
	}
	create(yearOne, rplID, "A")
	create(yearOne, tkjID, "B")
	create(yearTwo, rplID, "A")

	all, err := f.service.ListClasses(f.ctx, academic.ClassFilter{})
	if err != nil {
		t.Fatalf("ListClasses: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("ListClasses semua = %d, mau 3", len(all))
	}

	byYear, err := f.service.ListClasses(f.ctx, academic.ClassFilter{AcademicYearID: &yearOne})
	if err != nil {
		t.Fatalf("ListClasses tahun: %v", err)
	}
	if len(byYear) != 2 {
		t.Fatalf("ListClasses tahun = %d, mau 2", len(byYear))
	}

	byMajor, err := f.service.ListClasses(f.ctx, academic.ClassFilter{MajorID: &rplID})
	if err != nil {
		t.Fatalf("ListClasses jurusan: %v", err)
	}
	if len(byMajor) != 2 {
		t.Fatalf("ListClasses jurusan = %d, mau 2", len(byMajor))
	}

	byBoth, err := f.service.ListClasses(f.ctx, academic.ClassFilter{AcademicYearID: &yearOne, MajorID: &rplID})
	if err != nil {
		t.Fatalf("ListClasses tahun+jurusan: %v", err)
	}
	if len(byBoth) != 1 || byBoth[0].MajorCode != "rpl" {
		t.Fatalf("ListClasses tahun+jurusan = %+v, mau satu kelas RPL", byBoth)
	}
}

func TestAuditLog_RecordsAcademicActions(t *testing.T) {
	f := newFixture(t)
	majorActor := f.globalActor("academic.manage_majors")
	yearActor := f.globalActor("academic.manage_years")

	major, err := f.service.CreateMajor(f.ctx, majorActor, academic.CreateMajorInput{
		Code: "audit-major", Name: "Jurusan Audit", ShortName: "AM",
	})
	if err != nil {
		t.Fatalf("CreateMajor: %v", err)
	}
	newName := "Jurusan Audit Baru"
	if _, err := f.service.UpdateMajor(f.ctx, majorActor, major.ID, academic.UpdateMajorInput{Name: &newName}); err != nil {
		t.Fatalf("UpdateMajor: %v", err)
	}

	year, err := f.service.CreateYear(f.ctx, yearActor, academic.CreateYearInput{
		StartYear: 2032,
		StartsOn:  time.Date(2032, time.July, 1, 0, 0, 0, 0, time.UTC),
		EndsOn:    time.Date(2033, time.June, 30, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateYear: %v", err)
	}
	if err := f.service.SetCurrentYear(f.ctx, yearActor, year.ID); err != nil {
		t.Fatalf("SetCurrentYear: %v", err)
	}

	classActor := f.scopedActor("academic.manage_classes", major.ID)
	class, err := f.service.CreateClass(f.ctx, classActor, academic.CreateClassInput{
		AcademicYearID: year.ID,
		MajorID:        major.ID,
		Grade:          10,
		Section:        "A",
	})
	if err != nil {
		t.Fatalf("CreateClass: %v", err)
	}
	if _, err := f.service.UpdateClassSection(f.ctx, classActor, class.ID, "B"); err != nil {
		t.Fatalf("UpdateClassSection: %v", err)
	}
	if err := f.service.DeleteClass(f.ctx, classActor, class.ID); err != nil {
		t.Fatalf("DeleteClass: %v", err)
	}

	want := []string{
		"academic.major_create",
		"academic.major_update",
		"academic.year_create",
		"academic.year_set_current",
		"academic.class_create",
		"academic.class_update",
		"academic.class_delete",
	}
	actions := make(map[string]bool, len(want))
	for _, action := range f.auditActions() {
		actions[action] = true
	}
	for _, action := range want {
		if !actions[action] {
			t.Errorf("audit %s tidak tercatat (tercatat: %v)", action, f.auditActions())
		}
	}

	var actorID uuid.UUID
	var actorIP string
	err = f.pool.QueryRow(f.ctx, `
		SELECT actor_id, host(actor_ip)
		FROM audit_logs
		WHERE action = 'academic.major_create'`,
	).Scan(&actorID, &actorIP)
	if err != nil {
		t.Fatalf("baca audit actor: %v", err)
	}
	if actorID != f.actorID {
		t.Errorf("actor_id = %s, mau %s", actorID, f.actorID)
	}
	if actorIP != testActorIP {
		t.Errorf("actor_ip = %q, mau %q", actorIP, testActorIP)
	}
}

func TestCreateYear_Validation(t *testing.T) {
	f := newFixture(t)
	actor := f.globalActor("academic.manage_years")

	t.Run("start year outside range", func(t *testing.T) {
		_, err := f.service.CreateYear(f.ctx, actor, academic.CreateYearInput{
			StartYear: 1999,
			StartsOn:  time.Date(1999, time.July, 1, 0, 0, 0, 0, time.UTC),
			EndsOn:    time.Date(2000, time.June, 30, 0, 0, 0, 0, time.UTC),
		})
		requireFieldError(t, err, "start_year")
	})

	t.Run("ends before starts", func(t *testing.T) {
		_, err := f.service.CreateYear(f.ctx, actor, academic.CreateYearInput{
			StartYear: 2045,
			StartsOn:  time.Date(2045, time.July, 1, 0, 0, 0, 0, time.UTC),
			EndsOn:    time.Date(2045, time.June, 30, 0, 0, 0, 0, time.UTC),
		})
		requireFieldError(t, err, "ends_on")
	})

	t.Run("starts on different year", func(t *testing.T) {
		_, err := f.service.CreateYear(f.ctx, actor, academic.CreateYearInput{
			StartYear: 2045,
			StartsOn:  time.Date(2046, time.July, 1, 0, 0, 0, 0, time.UTC),
			EndsOn:    time.Date(2047, time.June, 30, 0, 0, 0, 0, time.UTC),
		})
		requireFieldError(t, err, "starts_on")
	})
}

func TestServiceMethods_WithoutPermissionReturnNotFound(t *testing.T) {
	f := newFixture(t)
	noGrants := f.actor()

	if _, err := f.service.CreateYear(f.ctx, noGrants, academic.CreateYearInput{
		StartYear: 2050,
		StartsOn:  time.Date(2050, time.July, 1, 0, 0, 0, 0, time.UTC),
		EndsOn:    time.Date(2051, time.June, 30, 0, 0, 0, 0, time.UTC),
	}); err == nil {
		t.Fatal("CreateYear tanpa izin berhasil")
	} else {
		requireAppError(t, err, "NOT_FOUND")
	}

	yearID := f.insertYear(2050, "2050-07-01", "2051-06-30")
	if err := f.service.SetCurrentYear(f.ctx, noGrants, yearID); err == nil {
		t.Fatal("SetCurrentYear tanpa izin berhasil")
	} else {
		requireAppError(t, err, "NOT_FOUND")
	}

	rplID := f.majorID("rpl")
	if _, err := f.service.CreateClass(f.ctx, noGrants, academic.CreateClassInput{
		AcademicYearID: yearID,
		MajorID:        rplID,
		Grade:          10,
		Section:        "A",
	}); err == nil {
		t.Fatal("CreateClass tanpa izin berhasil")
	} else {
		requireAppError(t, err, "NOT_FOUND")
	}
}
