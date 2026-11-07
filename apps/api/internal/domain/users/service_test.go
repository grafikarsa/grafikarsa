package users_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/users"
)

func TestCreateStudent_ScopedAdminCreatesEnrollmentExperienceAndInitialPassword(t *testing.T) {
	f := newFixture(t)
	nisn := f.nextNISN()

	created, err := f.service.Create(f.ctx, f.scopedActor(), users.CreateInput{
		AccountType: "student",
		DisplayName: "Siswa RPL Baru",
		NISN:        &nisn,
		ClassID:     &f.rplClassID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if created.InitialPassword == "" {
		t.Fatal("InitialPassword kosong, mau sandi awal dikembalikan sekali")
	}
	detail := created.User
	if detail.AccountType != "student" || detail.Status != "active" {
		t.Errorf("tipe/status = %s/%s, mau student/active", detail.AccountType, detail.Status)
	}
	if detail.Activated {
		t.Error("Activated = true, mau false untuk akun baru")
	}
	if detail.PasswordKind != "initial" {
		t.Errorf("PasswordKind = %q, mau initial", detail.PasswordKind)
	}
	if detail.CurrentClass == nil || detail.CurrentClass.ID != f.rplClassID {
		t.Fatalf("CurrentClass = %+v, mau kelas RPL %s", detail.CurrentClass, f.rplClassID)
	}
	if detail.CurrentClass.MajorCode != "rpl" || detail.CurrentClass.AcademicYearLabel != "2026/2027" {
		t.Errorf("CurrentClass = %+v, mau major rpl dan label 2026/2027", detail.CurrentClass)
	}
	if detail.EntryYear == nil || *detail.EntryYear != 2026 {
		t.Errorf("EntryYear = %v, mau 2026", detail.EntryYear)
	}
	if len(detail.Enrollments) != 1 {
		t.Fatalf("Enrollments = %d, mau 1", len(detail.Enrollments))
	}
	if detail.Enrollments[0].ClassID != f.rplClassID || detail.Enrollments[0].Outcome != nil {
		t.Errorf("Enrollment = %+v, mau kelas RPL tanpa outcome", detail.Enrollments[0])
	}

	var title, organization string
	var isSystem, isCurrent bool
	var startMonth time.Time
	err = f.pool.QueryRow(f.ctx, `
		SELECT title, organization, is_system, is_current, start_month
		FROM experiences
		WHERE user_id = $1`,
		detail.ID,
	).Scan(&title, &organization, &isSystem, &isCurrent, &startMonth)
	if err != nil {
		t.Fatalf("ambil experience sistem: %v", err)
	}
	if title != "Siswa Pengembangan Perangkat Lunak dan Gim (RPL)" {
		t.Errorf("judul experience = %q", title)
	}
	if organization != "SMKN 4 Malang" || !isSystem || !isCurrent {
		t.Errorf("experience = %q/%q system=%v current=%v", organization, organization, isSystem, isCurrent)
	}
	if got := startMonth.Format("2006-01-02"); got != "2026-07-01" {
		t.Errorf("start_month = %s, mau 2026-07-01", got)
	}

	hash := f.passwordHash(detail.ID)
	ok, _, err := f.hasher.Verify(f.ctx, hash, created.InitialPassword)
	if err != nil {
		t.Fatalf("verifikasi sandi awal: %v", err)
	}
	if !ok {
		t.Fatal("sandi awal yang dikembalikan tidak cocok dengan hash tersimpan")
	}

	kind, expiresAt := f.passwordInfo(detail.ID)
	if kind != "initial" {
		t.Errorf("password_kind = %q, mau initial", kind)
	}
	if d := time.Until(expiresAt); d < 29*24*time.Hour || d > 31*24*time.Hour {
		t.Errorf("masa berlaku sandi awal = %v, mau sekitar 30 hari", d)
	}
	if !f.hasAuditAction("user.create") {
		t.Error("audit user.create tidak tercatat")
	}
}

func TestCreate_ScopedAdminCannotCreateOtherMajorOrTeacher(t *testing.T) {
	f := newFixture(t)
	actor := f.scopedActor()

	t.Run("student in TKJ is not found", func(t *testing.T) {
		nisn := f.nextNISN()
		_, err := f.service.Create(f.ctx, actor, users.CreateInput{
			AccountType: "student",
			DisplayName: "Siswa TKJ",
			NISN:        &nisn,
			ClassID:     &f.tkjClassID,
		})
		requireAppError(t, err, "NOT_FOUND")
	})

	t.Run("teacher requires global permission", func(t *testing.T) {
		_, err := f.service.Create(f.ctx, actor, users.CreateInput{
			AccountType:     "teacher",
			DisplayName:     "Guru RPL",
			TeacherMajorIDs: []uuid.UUID{f.rplID},
		})
		requireAppError(t, err, "NOT_FOUND")
	})
}

func TestCreate_DuplicateNISNReturnsConflict(t *testing.T) {
	f := newFixture(t)
	nisn := f.nextNISN()

	if _, err := f.service.Create(f.ctx, f.globalActor(), users.CreateInput{
		AccountType: "student",
		DisplayName: "Siswa Pertama",
		NISN:        &nisn,
		ClassID:     &f.rplClassID,
	}); err != nil {
		t.Fatalf("Create pertama: %v", err)
	}

	_, err := f.service.Create(f.ctx, f.globalActor(), users.CreateInput{
		AccountType: "student",
		DisplayName: "Siswa Kedua",
		NISN:        &nisn,
		ClassID:     &f.tkjClassID,
	})
	e := requireAppError(t, err, "CONFLICT")
	if e.Detail != "NISN sudah terdaftar." {
		t.Errorf("Detail = %q, mau %q", e.Detail, "NISN sudah terdaftar.")
	}
}

func TestCreateTeacher_GlobalAdminStoresMajors(t *testing.T) {
	f := newFixture(t)

	created := f.createTeacher(f.globalActor(), "Guru RPL", []uuid.UUID{f.rplID, f.tkjID})
	if len(created.User.TeacherMajorIDs) != 2 {
		t.Fatalf("TeacherMajorIDs = %v, mau 2 jurusan", created.User.TeacherMajorIDs)
	}
	if created.User.PasswordKind != "initial" || created.User.EntryYear != nil {
		t.Errorf("detail guru = kind %q entry %v", created.User.PasswordKind, created.User.EntryYear)
	}

	var count int
	if err := f.pool.QueryRow(f.ctx, "SELECT count(*) FROM teacher_majors WHERE user_id = $1", created.User.ID).Scan(&count); err != nil {
		t.Fatalf("hitung teacher_majors: %v", err)
	}
	if count != 2 {
		t.Fatalf("teacher_majors = %d, mau 2", count)
	}
}

func TestList_ScopedAdminOnlySeesUsersInScope(t *testing.T) {
	f := newFixture(t)

	rpl := f.createStudent(f.scopedActor(), f.rplClassID, "Siswa RPL")
	tkj := f.createStudent(f.globalActor(), f.tkjClassID, "Siswa TKJ")

	page, err := f.service.List(f.ctx, f.scopedActor(), users.ListFilter{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("List scoped mengembalikan %d user, mau 1: %+v", len(page.Items), page.Items)
	}
	if page.Items[0].ID != rpl.User.ID {
		t.Errorf("user = %s, mau %s", page.Items[0].ID, rpl.User.ID)
	}
	if page.Items[0].CurrentClass == nil || page.Items[0].CurrentClass.ID != f.rplClassID {
		t.Errorf("CurrentClass = %+v, mau kelas RPL", page.Items[0].CurrentClass)
	}

	if _, err := f.service.Get(f.ctx, f.scopedActor(), tkj.User.ID); err == nil {
		t.Fatal("Get user TKJ oleh admin scoped = nil, mau NOT_FOUND")
	} else {
		requireAppError(t, err, "NOT_FOUND")
	}
	if _, err := f.service.Get(f.ctx, f.scopedActor(), rpl.User.ID); err != nil {
		t.Fatalf("Get user RPL oleh admin scoped: %v", err)
	}
}

func TestList_SearchByNameAndExactNISN(t *testing.T) {
	f := newFixture(t)
	actor := f.scopedActor()

	budi := f.createStudent(actor, f.rplClassID, "Budi Santoso")
	siti := f.createStudent(actor, f.rplClassID, "Siti Aminah")

	byName, err := f.service.List(f.ctx, actor, users.ListFilter{Query: "  budi  "})
	if err != nil {
		t.Fatalf("List by name: %v", err)
	}
	if len(byName.Items) != 1 || byName.Items[0].ID != budi.User.ID {
		t.Fatalf("hasil cari nama = %+v, mau hanya Budi", byName.Items)
	}

	byNISN, err := f.service.List(f.ctx, actor, users.ListFilter{Query: *siti.User.NISN})
	if err != nil {
		t.Fatalf("List by NISN: %v", err)
	}
	if len(byNISN.Items) != 1 || byNISN.Items[0].ID != siti.User.ID {
		t.Fatalf("hasil cari NISN = %+v, mau hanya Siti", byNISN.Items)
	}

	empty, err := f.service.List(f.ctx, actor, users.ListFilter{Query: "tidak ada"})
	if err != nil {
		t.Fatalf("List tanpa hasil: %v", err)
	}
	if len(empty.Items) != 0 {
		t.Fatalf("hasil = %d, mau 0", len(empty.Items))
	}
}

func TestList_Pagination(t *testing.T) {
	f := newFixture(t)
	actor := f.scopedActor()

	seen := map[uuid.UUID]struct{}{}
	for i := 0; i < 5; i++ {
		created := f.createStudent(actor, f.rplClassID, "Siswa Halaman")
		seen[created.User.ID] = struct{}{}
	}

	first, err := f.service.List(f.ctx, actor, users.ListFilter{Limit: 2})
	if err != nil {
		t.Fatalf("List halaman 1: %v", err)
	}
	if len(first.Items) != 2 || first.NextCursor == "" {
		t.Fatalf("halaman 1 = %d item, cursor %q; mau 2 dan cursor", len(first.Items), first.NextCursor)
	}

	second, err := f.service.List(f.ctx, actor, users.ListFilter{Limit: 2, Cursor: first.NextCursor})
	if err != nil {
		t.Fatalf("List halaman 2: %v", err)
	}
	if len(second.Items) != 2 || second.NextCursor == "" {
		t.Fatalf("halaman 2 = %d item, cursor %q; mau 2 dan cursor", len(second.Items), second.NextCursor)
	}

	third, err := f.service.List(f.ctx, actor, users.ListFilter{Limit: 2, Cursor: second.NextCursor})
	if err != nil {
		t.Fatalf("List halaman 3: %v", err)
	}
	if len(third.Items) != 1 || third.NextCursor != "" {
		t.Fatalf("halaman 3 = %d item, cursor %q; mau 1 dan tanpa cursor", len(third.Items), third.NextCursor)
	}

	for _, page := range [][]users.UserSummary{first.Items, second.Items, third.Items} {
		for _, item := range page {
			if _, exists := seen[item.ID]; !exists {
				t.Fatalf("user %s tidak dikenal di halaman", item.ID)
			}
			delete(seen, item.ID)
		}
	}
	if len(seen) != 0 {
		t.Fatalf("user belum terlihat: %v", seen)
	}
}

func TestUpdate_ScopedAdminUpdatesStudentAndAuditsFieldNames(t *testing.T) {
	f := newFixture(t)
	actor := f.scopedActor()
	created := f.createStudent(actor, f.rplClassID, "Nama Lama")

	newName := "Nama Baru"
	newNISN := f.nextNISN()
	updated, err := f.service.Update(f.ctx, actor, created.User.ID, users.UpdateInput{
		DisplayName: &newName,
		NISN:        &newNISN,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.DisplayName != newName || updated.NISN == nil || *updated.NISN != newNISN {
		t.Fatalf("detail setelah update = %q/%v", updated.DisplayName, updated.NISN)
	}
	if f.invalidator.last() != created.User.ID {
		t.Errorf("invalidator menerima %s, mau %s", f.invalidator.last(), created.User.ID)
	}

	var data []byte
	if err := f.pool.QueryRow(f.ctx, `
		SELECT data FROM audit_logs
		WHERE action = 'user.update' AND target_id = $1
		ORDER BY id DESC LIMIT 1`,
		created.User.ID,
	).Scan(&data); err != nil {
		t.Fatalf("ambil audit update: %v", err)
	}
	text := string(data)
	if !contains(text, "display_name") || !contains(text, "nisn") {
		t.Errorf("audit data = %s, mau memuat display_name dan nisn", text)
	}
	if contains(text, newNISN) {
		t.Errorf("audit data = %s, tidak boleh memuat nilai NISN", text)
	}
}

func TestUpdate_ScopedAdminCannotUpdateTeacher(t *testing.T) {
	f := newFixture(t)
	teacher := f.createTeacher(f.globalActor(), "Guru RPL", []uuid.UUID{f.rplID})

	newName := "Guru Diubah"
	_, err := f.service.Update(f.ctx, f.scopedActor(), teacher.User.ID, users.UpdateInput{
		DisplayName: &newName,
	})
	requireAppError(t, err, "NOT_FOUND")
}

func TestResetPassword_ActivatedUserGetsTemporaryPasswordAndSessionsRevoked(t *testing.T) {
	f := newFixture(t)
	actor := f.scopedActor()
	created := f.createStudent(actor, f.rplClassID, "Siswa Reset")
	f.activate(created.User.ID)
	f.insertSession(created.User.ID)

	before := f.invalidator.count()
	plain, expiresAt, err := f.service.ResetPassword(f.ctx, actor, created.User.ID)
	if err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	if plain == "" {
		t.Fatal("sandi sementara kosong")
	}

	kind, storedExpires := f.passwordInfo(created.User.ID)
	if kind != "temporary" {
		t.Errorf("password_kind = %q, mau temporary", kind)
	}
	if storedExpires.Sub(expiresAt).Abs() > time.Second {
		t.Errorf("expiresAt = %v, tersimpan %v", expiresAt, storedExpires)
	}
	if d := time.Until(expiresAt); d < 71*time.Hour || d > 73*time.Hour {
		t.Errorf("masa berlaku sandi sementara = %v, mau sekitar 72 jam", d)
	}

	hash := f.passwordHash(created.User.ID)
	ok, _, err := f.hasher.Verify(f.ctx, hash, plain)
	if err != nil {
		t.Fatalf("verifikasi sandi sementara: %v", err)
	}
	if !ok {
		t.Fatal("sandi sementara yang dikembalikan tidak cocok dengan hash")
	}

	if got := f.revokedSessionCount(created.User.ID, "admin_reset"); got != 1 {
		t.Errorf("sesi dicabut = %d, mau 1", got)
	}
	if f.invalidator.count() != before+1 || f.invalidator.last() != created.User.ID {
		t.Errorf("invalidator = %d panggilan, terakhir %s", f.invalidator.count(), f.invalidator.last())
	}
	if !f.hasAuditAction("user.reset_password") {
		t.Error("audit user.reset_password tidak tercatat")
	}
}

func TestResetPassword_NotActivatedReturnsInvalidTransition(t *testing.T) {
	f := newFixture(t)
	created := f.createStudent(f.scopedActor(), f.rplClassID, "Belum Aktivasi")

	_, _, err := f.service.ResetPassword(f.ctx, f.scopedActor(), created.User.ID)
	e := requireAppError(t, err, "INVALID_TRANSITION")
	if e.Detail != "Akun belum aktivasi, gunakan terbitkan ulang sandi awal." {
		t.Errorf("Detail = %q", e.Detail)
	}
}

func TestResetPassword_SelfReturnsConflict(t *testing.T) {
	f := newFixture(t)

	_, _, err := f.service.ResetPassword(f.ctx, f.globalActor(), f.globalAdminID)
	e := requireAppError(t, err, "CONFLICT")
	if e.Detail != "Tidak bisa dilakukan pada akunmu sendiri." {
		t.Errorf("Detail = %q", e.Detail)
	}
}

func TestReissueInitialPassword_NotActivatedChangesHash(t *testing.T) {
	f := newFixture(t)
	actor := f.scopedActor()
	created := f.createStudent(actor, f.rplClassID, "Siswa Terbit Ulang")

	oldHash := f.passwordHash(created.User.ID)
	plain, expiresAt, err := f.service.ReissueInitialPassword(f.ctx, actor, created.User.ID)
	if err != nil {
		t.Fatalf("ReissueInitialPassword: %v", err)
	}
	if plain == "" {
		t.Fatal("sandi awal baru kosong")
	}
	newHash := f.passwordHash(created.User.ID)
	if newHash == oldHash {
		t.Fatal("hash sandi tidak berubah setelah terbitkan ulang")
	}

	ok, _, err := f.hasher.Verify(f.ctx, newHash, plain)
	if err != nil {
		t.Fatalf("verifikasi sandi baru: %v", err)
	}
	if !ok {
		t.Fatal("sandi awal baru tidak cocok dengan hash")
	}
	kind, _ := f.passwordInfo(created.User.ID)
	if kind != "initial" {
		t.Errorf("password_kind = %q, mau initial", kind)
	}
	if d := time.Until(expiresAt); d < 29*24*time.Hour || d > 31*24*time.Hour {
		t.Errorf("masa berlaku = %v, mau sekitar 30 hari", d)
	}
	if !f.hasAuditAction("user.reissue_initial_password") {
		t.Error("audit user.reissue_initial_password tidak tercatat")
	}
}

func TestReissueInitialPassword_ActivatedReturnsInvalidTransition(t *testing.T) {
	f := newFixture(t)
	created := f.createStudent(f.scopedActor(), f.rplClassID, "Sudah Aktivasi")
	f.activate(created.User.ID)

	_, _, err := f.service.ReissueInitialPassword(f.ctx, f.scopedActor(), created.User.ID)
	e := requireAppError(t, err, "INVALID_TRANSITION")
	if e.Detail != "Akun sudah aktivasi, gunakan reset sandi." {
		t.Errorf("Detail = %q", e.Detail)
	}
}

func TestDeactivate_SelfReturnsConflict(t *testing.T) {
	f := newFixture(t)

	err := f.service.Deactivate(f.ctx, f.globalActor(), f.globalAdminID)
	e := requireAppError(t, err, "CONFLICT")
	if e.Detail != "Tidak bisa dilakukan pada akunmu sendiri." {
		t.Errorf("Detail = %q", e.Detail)
	}
}

func TestDeactivate_OtherUserBecomesInactiveAndSessionsRevoked(t *testing.T) {
	f := newFixture(t)
	actor := f.scopedActor()
	created := f.createStudent(actor, f.rplClassID, "Siswa Nonaktif")
	f.activate(created.User.ID)
	f.insertSession(created.User.ID)

	if err := f.service.Deactivate(f.ctx, actor, created.User.ID); err != nil {
		t.Fatalf("Deactivate: %v", err)
	}

	status, deactivatedAt := f.userStatus(created.User.ID)
	if status != "inactive" {
		t.Errorf("status = %q, mau inactive", status)
	}
	if deactivatedAt == nil {
		t.Error("deactivated_at = nil, mau terisi")
	}
	if got := f.revokedSessionCount(created.User.ID, "deactivated"); got != 1 {
		t.Errorf("sesi dicabut = %d, mau 1", got)
	}
	if f.invalidator.last() != created.User.ID {
		t.Errorf("invalidator terakhir = %s, mau %s", f.invalidator.last(), created.User.ID)
	}
	if !f.hasAuditAction("user.deactivate") {
		t.Error("audit user.deactivate tidak tercatat")
	}
}

func TestReactivate_AlumniFlaggedUserReturnsToAlumni(t *testing.T) {
	f := newFixture(t)
	created := f.createStudent(f.scopedActor(), f.rplClassID, "Alumni Nonaktif")

	if _, err := f.pool.Exec(f.ctx, `
		UPDATE users
		SET status = 'inactive',
		    graduation_year = 2025,
		    deactivated_at = now()
		WHERE id = $1`,
		created.User.ID,
	); err != nil {
		t.Fatalf("siapkan alumni nonaktif: %v", err)
	}

	if err := f.service.Reactivate(f.ctx, f.globalActor(), created.User.ID); err != nil {
		t.Fatalf("Reactivate: %v", err)
	}

	status, deactivatedAt := f.userStatus(created.User.ID)
	if status != "alumni" {
		t.Errorf("status = %q, mau alumni", status)
	}
	if deactivatedAt != nil {
		t.Errorf("deactivated_at = %v, mau nil", deactivatedAt)
	}
	if f.invalidator.last() != created.User.ID {
		t.Errorf("invalidator terakhir = %s, mau %s", f.invalidator.last(), created.User.ID)
	}
	if !f.hasAuditAction("user.reactivate") {
		t.Error("audit user.reactivate tidak tercatat")
	}
}

func TestBlockContent_RequiresGlobalPermission(t *testing.T) {
	f := newFixture(t)
	created := f.createStudent(f.scopedActor(), f.rplClassID, "Siswa Diblokir")

	err := f.service.BlockContent(f.ctx, f.scopedActor(), created.User.ID, "alasan valid")
	requireAppError(t, err, "NOT_FOUND")

	if err := f.service.BlockContent(f.ctx, f.globalActor(), created.User.ID, "  melanggar pedoman  "); err != nil {
		t.Fatalf("BlockContent global: %v", err)
	}

	var blockedAt *time.Time
	var reason *string
	if err := f.pool.QueryRow(f.ctx,
		"SELECT content_blocked_at, content_blocked_reason FROM users WHERE id = $1",
		created.User.ID,
	).Scan(&blockedAt, &reason); err != nil {
		t.Fatalf("ambil status blokir: %v", err)
	}
	if blockedAt == nil || reason == nil || *reason != "melanggar pedoman" {
		t.Errorf("blokir = %v/%v, mau terisi dan alasan terpangkas", blockedAt, reason)
	}
	if f.invalidator.last() != created.User.ID {
		t.Errorf("invalidator terakhir = %s, mau %s", f.invalidator.last(), created.User.ID)
	}

	if err := f.service.UnblockContent(f.ctx, f.globalActor(), created.User.ID); err != nil {
		t.Fatalf("UnblockContent: %v", err)
	}
	if err := f.pool.QueryRow(f.ctx,
		"SELECT content_blocked_at, content_blocked_reason FROM users WHERE id = $1",
		created.User.ID,
	).Scan(&blockedAt, &reason); err != nil {
		t.Fatalf("ambil status blokir setelah buka: %v", err)
	}
	if blockedAt != nil || reason != nil {
		t.Errorf("blokir setelah buka = %v/%v, mau nil", blockedAt, reason)
	}
	if !f.hasAuditAction("user.block_content") || !f.hasAuditAction("user.unblock_content") {
		t.Error("audit blokir/buka blokir tidak lengkap")
	}
}

func TestBlockContent_InvalidReasonReturnsValidationError(t *testing.T) {
	f := newFixture(t)
	created := f.createStudent(f.scopedActor(), f.rplClassID, "Siswa Alasan Pendek")

	err := f.service.BlockContent(f.ctx, f.globalActor(), created.User.ID, "x")
	requireAppError(t, err, "VALIDATION_FAILED")
}

func TestAllActionsWriteAuditRows(t *testing.T) {
	f := newFixture(t)
	actor := f.scopedActor()

	student := f.createStudent(actor, f.rplClassID, "Siswa Audit")
	newName := "Siswa Audit Diubah"
	if _, err := f.service.Update(f.ctx, actor, student.User.ID, users.UpdateInput{DisplayName: &newName}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if _, _, err := f.service.ReissueInitialPassword(f.ctx, actor, student.User.ID); err != nil {
		t.Fatalf("ReissueInitialPassword: %v", err)
	}

	activated := f.createStudent(actor, f.rplClassID, "Siswa Audit Aktif")
	f.activate(activated.User.ID)
	if _, _, err := f.service.ResetPassword(f.ctx, actor, activated.User.ID); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	if err := f.service.Deactivate(f.ctx, actor, activated.User.ID); err != nil {
		t.Fatalf("Deactivate: %v", err)
	}
	if err := f.service.Reactivate(f.ctx, actor, activated.User.ID); err != nil {
		t.Fatalf("Reactivate: %v", err)
	}
	if err := f.service.BlockContent(f.ctx, f.globalActor(), activated.User.ID, "alasan audit"); err != nil {
		t.Fatalf("BlockContent: %v", err)
	}
	if err := f.service.UnblockContent(f.ctx, f.globalActor(), activated.User.ID); err != nil {
		t.Fatalf("UnblockContent: %v", err)
	}

	for _, action := range []string{
		"user.create",
		"user.update",
		"user.reissue_initial_password",
		"user.reset_password",
		"user.deactivate",
		"user.reactivate",
		"user.block_content",
		"user.unblock_content",
	} {
		if !f.hasAuditAction(action) {
			t.Errorf("audit %s tidak tercatat", action)
		}
	}
}

func contains(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func TestCreate_StoresInitialPasswordInVault(t *testing.T) {
	f := newFixture(t)

	created := f.createStudent(f.scopedActor(), f.rplClassID, "Siswa Brankas")

	plain, ok := f.vaultPlaintext(created.User.ID)
	if !ok {
		t.Fatal("entri brankas tidak ada setelah Create")
	}
	if plain != created.InitialPassword {
		t.Errorf("sandi di brankas = %q, mau sama dengan sandi awal yang dikembalikan", plain)
	}
}

func TestCreate_StoresInitialPasswordInVaultForTeacher(t *testing.T) {
	f := newFixture(t)

	created := f.createTeacher(f.globalActor(), "Guru Brankas", []uuid.UUID{f.rplID})

	plain, ok := f.vaultPlaintext(created.User.ID)
	if !ok {
		t.Fatal("entri brankas tidak ada setelah Create guru")
	}
	if plain != created.InitialPassword {
		t.Errorf("sandi guru di brankas = %q, mau sama dengan sandi awal", plain)
	}
}

func TestReissueInitialPassword_RotatesVaultEntry(t *testing.T) {
	f := newFixture(t)

	created := f.createStudent(f.scopedActor(), f.rplClassID, "Siswa Terbit Ulang")

	reissued, _, err := f.service.ReissueInitialPassword(f.ctx, f.scopedActor(), created.User.ID)
	if err != nil {
		t.Fatalf("ReissueInitialPassword: %v", err)
	}
	if reissued == created.InitialPassword {
		t.Fatal("sandi terbitan ulang sama dengan sandi awal sebelumnya")
	}

	plain, ok := f.vaultPlaintext(created.User.ID)
	if !ok {
		t.Fatal("entri brankas hilang setelah terbit ulang")
	}
	if plain != reissued {
		t.Errorf("sandi di brankas = %q, mau sandi terbitan ulang", plain)
	}
}

func TestResetPassword_DeletesVaultEntry(t *testing.T) {
	f := newFixture(t)

	created := f.createStudent(f.scopedActor(), f.rplClassID, "Siswa Reset")
	f.activate(created.User.ID)

	if _, _, err := f.service.ResetPassword(f.ctx, f.scopedActor(), created.User.ID); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}

	if _, ok := f.vaultPlaintext(created.User.ID); ok {
		t.Fatal("entri brankas masih ada setelah ResetPassword")
	}
}

func TestDeactivate_DeletesVaultEntry(t *testing.T) {
	f := newFixture(t)

	created := f.createStudent(f.scopedActor(), f.rplClassID, "Siswa Nonaktif")

	if err := f.service.Deactivate(f.ctx, f.scopedActor(), created.User.ID); err != nil {
		t.Fatalf("Deactivate: %v", err)
	}

	if _, ok := f.vaultPlaintext(created.User.ID); ok {
		t.Fatal("entri brankas masih ada setelah Deactivate")
	}
}
