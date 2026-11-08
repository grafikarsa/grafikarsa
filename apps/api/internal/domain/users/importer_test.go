package users_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/users"
)

func (f *fixture) markCurrentYear() {
	f.t.Helper()
	if _, err := f.pool.Exec(f.ctx,
		"UPDATE academic_years SET is_current = true WHERE id = $1", f.yearID); err != nil {
		f.t.Fatalf("tandai tahun ajaran berjalan: %v", err)
	}
}

func optionalInt(value *int) string {
	if value == nil {
		return "kosong"
	}
	return fmt.Sprintf("%d", *value)
}

func importGlobalActor(f *fixture) users.Actor {
	return users.Actor{
		Principal: &auth.Principal{
			UserID: f.globalAdminID,
			Grants: []auth.Grant{
				{Permission: "academic.import", Global: true},
				{Permission: "user.create", Global: true},
			},
		},
		IP: testActorIP,
	}
}

func importScopedActor(f *fixture, majorIDs ...uuid.UUID) users.Actor {
	return users.Actor{
		Principal: &auth.Principal{
			UserID: f.scopedAdminID,
			Grants: []auth.Grant{
				{Permission: "academic.import", MajorIDs: majorIDs},
			},
		},
		IP: testActorIP,
	}
}

func (f *fixture) newImporter() *users.Importer {
	return users.NewImporter(f.service, nil)
}

func hasImportError(job users.ImportJob, row int, field, messagePart string) bool {
	for _, rowError := range job.Errors {
		if rowError.Row == row && rowError.Field == field &&
			strings.Contains(rowError.Message, messagePart) {
			return true
		}
	}
	return false
}

func (f *fixture) importStudentsCSV(importer *users.Importer, actor users.Actor, content string) users.ImportJob {
	f.t.Helper()
	job, err := importer.Create(f.ctx, actor, "students", []byte(content))
	if err != nil {
		f.t.Fatalf("Create impor siswa: %v", err)
	}
	return job
}

func (f *fixture) countUsersWithNISN(nisn string) int {
	f.t.Helper()
	var count int
	if err := f.pool.QueryRow(f.ctx, "SELECT count(*) FROM users WHERE nisn = $1", nisn).Scan(&count); err != nil {
		f.t.Fatalf("hitung user NISN %s: %v", nisn, err)
	}
	return count
}

func (f *fixture) commitImportJob(jobID uuid.UUID) {
	f.t.Helper()
	if _, err := f.pool.Exec(f.ctx,
		"UPDATE import_jobs SET status = 'committing' WHERE id = $1", jobID); err != nil {
		f.t.Fatalf("ubah status impor jadi committing: %v", err)
	}
	if err := f.newImporter().RunCommitJob(f.ctx, users.ImportCommitArgs{
		ImportJobID: jobID,
		ActorID:     f.globalAdminID,
		ActorIP:     testActorIP,
	}); err != nil {
		f.t.Fatalf("jalankan commit impor: %v", err)
	}
}

func TestImportStudents_HeaderAndSeparatorFormats(t *testing.T) {
	f := newFixture(t)
	f.markCurrentYear()
	importer := f.newImporter()
	actor := importGlobalActor(f)

	cases := []struct {
		name    string
		content string
	}{
		{
			name:    "comma",
			content: "nisn,nama,kelas\n1000009001,Budi Santoso,X RPL A\n",
		},
		{
			name:    "semicolon",
			content: "nisn;nama;kelas\n1000009002;Siti Aminah;X RPL A\n",
		},
		{
			name:    "bom and uppercase headers",
			content: "\xEF\xBB\xBFNISN,NAMA,KELAS\n1000009003,Ahmad Fauzi,X RPL A\n",
		},
		{
			name:    "extra columns are ignored",
			content: "nisn,nama,kelas,catatan\n1000009004,Rina Wati,X RPL A,baris baru\n",
		},
		{
			name:    "collapsed class spaces",
			content: "nisn,nama,kelas\n1000009005,Dewi Lestari,  x   rpl   a  \n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			job := f.importStudentsCSV(importer, actor, tc.content)
			if job.Status != "ready" {
				t.Fatalf("status = %q, errors %+v, mau ready", job.Status, job.Errors)
			}
			if job.TotalRows == nil || *job.TotalRows != 1 {
				t.Fatalf("totalRows = %v, mau 1", optionalInt(job.TotalRows))
			}
			if job.ValidRows == nil || *job.ValidRows != 1 {
				t.Fatalf("validRows = %v, mau 1", optionalInt(job.ValidRows))
			}
			if job.ErrorCountTotal != 0 || len(job.Errors) != 0 {
				t.Fatalf("errors = %+v, mau kosong", job.Errors)
			}
			if got := job.ByClass["X RPL A"]; got != 1 {
				t.Fatalf("byClass = %+v, mau X RPL A: 1", job.ByClass)
			}
		})
	}
}

func TestImportStudents_MissingHeaderReturnsSingleFileError(t *testing.T) {
	f := newFixture(t)
	f.markCurrentYear()

	job := f.importStudentsCSV(f.newImporter(), importGlobalActor(f), "nisn,nama\n1000009101,Budi\n")
	if job.Status != "invalid" {
		t.Fatalf("status = %q, mau invalid", job.Status)
	}
	if len(job.Errors) != 1 {
		t.Fatalf("errors = %+v, mau satu error", job.Errors)
	}
	rowError := job.Errors[0]
	if rowError.Row != 0 || rowError.Field != "header" ||
		!strings.Contains(rowError.Message, "kelas") {
		t.Fatalf("error = %+v, mau row 0 field header yang menyebut kelas", rowError)
	}
	if job.ErrorCountTotal != 1 {
		t.Fatalf("errorCountTotal = %d, mau 1", job.ErrorCountTotal)
	}
}

func TestImportStudents_CollectsEveryRowError(t *testing.T) {
	f := newFixture(t)
	f.markCurrentYear()
	importer := f.newImporter()
	actor := importGlobalActor(f)

	f.createStudent(actor, f.rplClassID, "Siswa Sudah Ada")
	var registeredNISN string
	if err := f.pool.QueryRow(f.ctx,
		"SELECT nisn FROM users WHERE display_name = 'Siswa Sudah Ada'").Scan(&registeredNISN); err != nil {
		f.t.Fatalf("ambil NISN terdaftar: %v", err)
	}

	csv := strings.Join([]string{
		"nisn,nama,kelas,tahun_masuk",
		"123,Budi,X RPL A,1999",                             // bad NISN + bad year
		"1000009202,,X RPL A,",                              // empty name
		"1000009203,Siti,Kelas Tidak Ada,",                  // class not found
		"1000009204,Ahmad,X RPL A,1899",                     // bad year
		"1000009205,Rina,X RPL A,",                          // valid
		fmt.Sprintf("%s,Duplikat,X RPL A,", registeredNISN), // already registered
		"", // trailing newline
	}, "\n")

	job := f.importStudentsCSV(importer, actor, csv)
	if job.Status != "invalid" {
		t.Fatalf("status = %q, mau invalid", job.Status)
	}
	if job.TotalRows == nil || *job.TotalRows != 6 {
		t.Fatalf("totalRows = %v, mau 6", optionalInt(job.TotalRows))
	}
	if job.ValidRows == nil || *job.ValidRows != 1 {
		t.Fatalf("validRows = %v, mau 1", optionalInt(job.ValidRows))
	}

	checks := []struct {
		row, field, message string
	}{
		{"1", "nisn", "10 digit"},
		{"1", "tahun_masuk", "2000 dan 2100"},
		{"2", "nama", "1–100"},
		{"3", "kelas", "tidak ditemukan"},
		{"4", "tahun_masuk", "2000 dan 2100"},
		{"6", "nisn", "sudah terdaftar"},
	}
	for _, check := range checks {
		row := 0
		fmt.Sscanf(check.row, "%d", &row)
		if !hasImportError(job, row, check.field, check.message) {
			t.Errorf("error baris %d field %s yang menyebut %q tidak ada: %+v",
				row, check.field, check.message, job.Errors)
		}
	}
	if job.ErrorCountTotal != len(job.Errors) {
		t.Errorf("errorCountTotal = %d, len(errors) = %d", job.ErrorCountTotal, len(job.Errors))
	}
}

func TestImportStudents_DuplicateWithinFile(t *testing.T) {
	f := newFixture(t)
	f.markCurrentYear()
	importer := f.newImporter()
	actor := importGlobalActor(f)

	csv := "nisn,nama,kelas\n1000009301,Budi,X RPL A\n1000009301,Siti,X RPL A\n"
	job := f.importStudentsCSV(importer, actor, csv)
	if job.Status != "invalid" {
		t.Fatalf("status = %q, mau invalid", job.Status)
	}
	if !hasImportError(job, 2, "nisn", "duplikat") {
		t.Fatalf("error duplikat tidak ada: %+v", job.Errors)
	}
}

func TestImportTeachers_ValidatesNIPAndMajors(t *testing.T) {
	f := newFixture(t)
	importer := f.newImporter()
	actor := importGlobalActor(f)

	csv := strings.Join([]string{
		"nip,nama,jurusan",
		"1987000001,Guru RPL,rpl",
		",Guru Tanpa NIP,rpl",
		"12,Guru NIP Pendek,rpl",
		"1987000002,Guru Jurusan Salah,tkj|xyz",
		"1987000003,Guru Multi,rpl|tkj|rpl",
	}, "\n")

	job, err := importer.Create(f.ctx, actor, "teachers", []byte(csv))
	if err != nil {
		t.Fatalf("Create impor guru: %v", err)
	}
	if job.Status != "invalid" {
		t.Fatalf("status = %q, mau invalid", job.Status)
	}
	if job.ValidRows == nil || *job.ValidRows != 2 {
		t.Fatalf("validRows = %v, mau 2", optionalInt(job.ValidRows))
	}
	if !hasImportError(job, 2, "nip", "wajib") {
		t.Errorf("error NIP wajib tidak ada: %+v", job.Errors)
	}
	if !hasImportError(job, 3, "nip", "3–30") {
		t.Errorf("error panjang NIP tidak ada: %+v", job.Errors)
	}
	if !hasImportError(job, 4, "jurusan", "tidak ditemukan") {
		t.Errorf("error jurusan tidak ada: %+v", job.Errors)
	}
}

func TestImportStaff_NIPIsOptional(t *testing.T) {
	f := newFixture(t)
	importer := f.newImporter()
	actor := importGlobalActor(f)

	csv := "nip,nama\n,Tendik Tanpa NIP\n1999000001,Tendik Dengan NIP\n"
	job, err := importer.Create(f.ctx, actor, "staff", []byte(csv))
	if err != nil {
		t.Fatalf("Create impor tendik: %v", err)
	}
	if job.Status != "ready" {
		t.Fatalf("status = %q, errors %+v, mau ready", job.Status, job.Errors)
	}
	if job.ValidRows == nil || *job.ValidRows != 2 {
		t.Fatalf("validRows = %v, mau 2", optionalInt(job.ValidRows))
	}
	if len(job.ByClass) != 0 {
		t.Fatalf("byClass = %+v, mau kosong", job.ByClass)
	}
}

func TestImportStudents_ScopedAdminCannotImportOtherMajor(t *testing.T) {
	f := newFixture(t)
	f.markCurrentYear()
	importer := f.newImporter()
	actor := importScopedActor(f, f.rplID)

	csv := "nisn,nama,kelas\n1000009401,Siswa RPL,X RPL A\n1000009402,Siswa TKJ,X TKJ A\n"
	job := f.importStudentsCSV(importer, actor, csv)
	if job.Status != "invalid" {
		t.Fatalf("status = %q, mau invalid", job.Status)
	}
	if !hasImportError(job, 2, "kelas", "Kamu tidak berwenang mengimpor ke kelas ini.") {
		t.Fatalf("error otorisasi tidak ada: %+v", job.Errors)
	}
	if job.ValidRows == nil || *job.ValidRows != 1 {
		t.Fatalf("validRows = %v, mau 1", optionalInt(job.ValidRows))
	}
	if got := job.ByClass["X RPL A"]; got != 1 {
		t.Fatalf("byClass = %+v, mau X RPL A: 1", job.ByClass)
	}
}

func TestImport_ScopedAdminCannotImportTeachersOrStaff(t *testing.T) {
	f := newFixture(t)
	importer := f.newImporter()
	actor := importScopedActor(f, f.rplID)

	for _, kind := range []string{"teachers", "staff"} {
		t.Run(kind, func(t *testing.T) {
			_, err := importer.Create(f.ctx, actor, kind, []byte("nama\nGuru RPL\n"))
			requireAppError(t, err, "NOT_FOUND")
		})
	}

	_, err := importer.Create(f.ctx, importScopedActor(f, f.rplID), "students",
		[]byte("nisn,nama,kelas\n1000009501,Budi,X RPL A\n"))
	// Uploading students in scope is allowed; the check above is only about
	// teacher/staff imports.
	if err != nil {
		t.Fatalf("impor siswa dalam scope gagal: %v", err)
	}
}

func TestImport_RequiresPermission(t *testing.T) {
	f := newFixture(t)
	importer := f.newImporter()

	_, err := importer.Create(f.ctx, users.Actor{
		Principal: &auth.Principal{UserID: f.globalAdminID},
	}, "students", []byte("nisn,nama,kelas\n1000009601,Budi,X RPL A\n"))
	requireAppError(t, err, "NOT_FOUND")
}

func TestImportCommit_CreatesAccountsWithInitialPasswords(t *testing.T) {
	f := newFixture(t)
	f.markCurrentYear()
	importer := f.newImporter()
	actor := importGlobalActor(f)

	csv := strings.Join([]string{
		"nisn,nama,kelas",
		"1000009701,Budi Santoso,X RPL A",
		"1000009702,Siti Aminah,X RPL A",
		"1000009703,Ahmad Fauzi,X RPL A",
	}, "\n")
	job := f.importStudentsCSV(importer, actor, csv)
	if job.Status != "ready" {
		t.Fatalf("status = %q, errors %+v", job.Status, job.Errors)
	}

	f.commitImportJob(job.ID)

	committed, err := importer.Get(f.ctx, actor, job.ID)
	if err != nil {
		t.Fatalf("Get impor: %v", err)
	}
	if committed.Status != "committed" {
		t.Fatalf("status = %q, mau committed", committed.Status)
	}
	if committed.Created == nil || *committed.Created != 3 {
		t.Fatalf("created = %v, mau 3", optionalInt(committed.Created))
	}
	if len(committed.Skipped) != 0 {
		t.Fatalf("skipped = %+v, mau kosong", committed.Skipped)
	}
	if committed.CommittedAt == nil {
		t.Fatal("committedAt kosong")
	}

	for _, nisn := range []string{"1000009701", "1000009702", "1000009703"} {
		var userID uuid.UUID
		var source string
		var vaultCount int
		err := f.pool.QueryRow(f.ctx, `
			SELECT u.id, e.source
			FROM users u
			JOIN enrollments e ON e.user_id = u.id
			WHERE u.nisn = $1 AND e.class_id = $2`,
			nisn, f.rplClassID,
		).Scan(&userID, &source)
		if err != nil {
			t.Fatalf("ambil user %s: %v", nisn, err)
		}
		if source != "import" {
			t.Errorf("source enrollment = %q, mau import", source)
		}
		if err := f.pool.QueryRow(f.ctx,
			"SELECT count(*) FROM initial_password_vault WHERE user_id = $1", userID).Scan(&vaultCount); err != nil {
			t.Fatalf("hitung brankas %s: %v", nisn, err)
		}
		if vaultCount != 1 {
			t.Errorf("entri brankas %s = %d, mau 1", nisn, vaultCount)
		}
		if _, ok := f.vaultPlaintext(userID); !ok {
			t.Errorf("sandi awal %s tidak ada di brankas", nisn)
		}
	}

	// Bulk imports must not write one user.create audit per row.
	var createAudits int
	if err := f.pool.QueryRow(f.ctx,
		"SELECT count(*) FROM audit_logs WHERE action = 'user.create'").Scan(&createAudits); err != nil {
		t.Fatalf("hitung audit user.create: %v", err)
	}
	if createAudits != 0 {
		t.Errorf("audit user.create = %d, mau 0", createAudits)
	}
	if !f.hasAuditAction("user.import_upload") || !f.hasAuditAction("user.import_commit") {
		t.Errorf("audit impor tidak lengkap: %v", f.auditActions())
	}
}

func TestImportCommit_IsIdempotent(t *testing.T) {
	f := newFixture(t)
	f.markCurrentYear()
	importer := f.newImporter()
	actor := importGlobalActor(f)

	job := f.importStudentsCSV(importer, actor,
		"nisn,nama,kelas\n1000009801,Budi Santoso,X RPL A\n")
	f.commitImportJob(job.ID)

	if got := f.countUsersWithNISN("1000009801"); got != 1 {
		t.Fatalf("user setelah commit pertama = %d, mau 1", got)
	}

	// Simulate a retry of the same job.
	f.commitImportJob(job.ID)
	if got := f.countUsersWithNISN("1000009801"); got != 1 {
		t.Fatalf("user setelah commit kedua = %d, mau 1", got)
	}

	var enrollmentCount int
	if err := f.pool.QueryRow(f.ctx, `
		SELECT count(*) FROM enrollments e
		JOIN users u ON u.id = e.user_id
		WHERE u.nisn = '1000009801' AND e.class_id = $1`, f.rplClassID).Scan(&enrollmentCount); err != nil {
		f.t.Fatalf("hitung enrollment: %v", err)
	}
	if enrollmentCount != 1 {
		t.Fatalf("enrollment = %d, mau 1", enrollmentCount)
	}

	var summary []byte
	if err := f.pool.QueryRow(f.ctx,
		"SELECT summary FROM import_jobs WHERE id = $1", job.ID).Scan(&summary); err != nil {
		f.t.Fatalf("ambil summary: %v", err)
	}
	if !strings.Contains(string(summary), `"created": 1`) {
		t.Fatalf("summary = %s, mau created 1", summary)
	}
}

func TestImportCommit_SkipsNISNTakenAfterValidation(t *testing.T) {
	f := newFixture(t)
	f.markCurrentYear()
	importer := f.newImporter()
	actor := importGlobalActor(f)

	job := f.importStudentsCSV(importer, actor,
		"nisn,nama,kelas\n1000009901,Budi Santoso,X RPL A\n")
	if job.Status != "ready" {
		t.Fatalf("status = %q, errors %+v", job.Status, job.Errors)
	}

	// Someone creates the same NISN after validation but before commit.
	nisn := "1000009901"
	_, err := f.service.Create(f.ctx, users.Actor{
		Principal: &auth.Principal{
			UserID: f.globalAdminID,
			Grants: []auth.Grant{{Permission: "user.create", Global: true}},
		},
		IP: testActorIP,
	}, users.CreateInput{
		AccountType: "student",
		DisplayName: "Budi Lain",
		NISN:        &nisn,
		ClassID:     &f.tkjClassID,
	})
	if err != nil {
		t.Fatalf("buat user pesaing: %v", err)
	}

	f.commitImportJob(job.ID)

	committed, err := importer.Get(f.ctx, actor, job.ID)
	if err != nil {
		t.Fatalf("Get impor: %v", err)
	}
	if committed.Status != "committed" {
		t.Fatalf("status = %q, mau committed", committed.Status)
	}
	if committed.Created == nil || *committed.Created != 0 {
		t.Fatalf("created = %v, mau 0", optionalInt(committed.Created))
	}
	if len(committed.Skipped) != 1 || committed.Skipped[0].Row != 1 {
		t.Fatalf("skipped = %+v, mau satu baris", committed.Skipped)
	}
	if !strings.Contains(committed.Skipped[0].Reason, "sudah terdaftar") {
		t.Fatalf("alasan skip = %q", committed.Skipped[0].Reason)
	}
	if got := f.countUsersWithNISN(nisn); got != 1 {
		t.Fatalf("user dengan NISN bentrok = %d, mau 1", got)
	}
}

func TestImportCommit_RequiresReadyStatus(t *testing.T) {
	f := newFixture(t)
	f.markCurrentYear()
	importer := f.newImporter()
	actor := importGlobalActor(f)

	job := f.importStudentsCSV(importer, actor,
		"nisn,nama,kelas\n1000010001,Budi,X RPL A\n")
	if _, err := f.pool.Exec(f.ctx,
		"UPDATE import_jobs SET status = 'invalid' WHERE id = $1", job.ID); err != nil {
		f.t.Fatalf("ubah status: %v", err)
	}

	if err := importer.RunCommitJob(f.ctx, users.ImportCommitArgs{
		ImportJobID: job.ID,
		ActorID:     f.globalAdminID,
	}); err != nil {
		t.Fatalf("RunCommitJob: %v", err)
	}

	var status string
	if err := f.pool.QueryRow(f.ctx,
		"SELECT status FROM import_jobs WHERE id = $1", job.ID).Scan(&status); err != nil {
		f.t.Fatalf("ambil status: %v", err)
	}
	if status != "invalid" {
		t.Fatalf("status = %q, mau tetap invalid", status)
	}
}

func TestImportList_ScopedActorOnlySeesOwnJobs(t *testing.T) {
	f := newFixture(t)
	f.markCurrentYear()
	importer := f.newImporter()

	globalActor := importGlobalActor(f)
	scopedActor := importScopedActor(f, f.rplID)

	globalJob := f.importStudentsCSV(importer, globalActor,
		"nisn,nama,kelas\n1000010101,Budi,X RPL A\n")
	scopedJob := f.importStudentsCSV(importer, scopedActor,
		"nisn,nama,kelas\n1000010102,Siti,X RPL A\n")

	scopedItems, err := importer.List(f.ctx, scopedActor)
	if err != nil {
		t.Fatalf("List scoped: %v", err)
	}
	if len(scopedItems) != 1 || scopedItems[0].ID != scopedJob.ID {
		t.Fatalf("scoped items = %+v, mau hanya %s", scopedItems, scopedJob.ID)
	}

	globalItems, err := importer.List(f.ctx, globalActor)
	if err != nil {
		t.Fatalf("List global: %v", err)
	}
	if len(globalItems) != 2 {
		t.Fatalf("global items = %d, mau 2", len(globalItems))
	}

	if _, err := importer.Get(f.ctx, scopedActor, globalJob.ID); err == nil {
		t.Fatal("scoped actor bisa melihat impor milik aktor lain")
	}
}

// TestImportErrorCountTotal_CapsStoredErrors ensures a large invalid file does
// not store more than 500 row errors while keeping the real total.
func TestImportErrorCountTotal_CapsStoredErrors(t *testing.T) {
	f := newFixture(t)
	f.markCurrentYear()
	importer := f.newImporter()
	actor := importGlobalActor(f)

	var builder strings.Builder
	builder.WriteString("nisn,nama,kelas\n")
	for i := 0; i < 600; i++ {
		fmt.Fprintf(&builder, "bad-%d,Nama %d,X RPL A\n", i, i)
	}

	job := f.importStudentsCSV(importer, actor, builder.String())
	if job.Status != "invalid" {
		t.Fatalf("status = %q, mau invalid", job.Status)
	}
	if len(job.Errors) != 500 {
		t.Fatalf("errors tersimpan = %d, mau 500", len(job.Errors))
	}
	if job.ErrorCountTotal != 600 {
		t.Fatalf("errorCountTotal = %d, mau 600", job.ErrorCountTotal)
	}
}
