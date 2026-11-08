package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/users"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/jobs"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/password"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

const importsPath = "/api/v1/admin/imports"

type importsEnv struct {
	*usersEnv

	importer     *users.Importer
	insertClient *river.Client[pgx.Tx]
	pool         *pgxpool.Pool
}

// newImportsEnv builds the real router with auth, user administration, and CSV
// imports on a seeded database, served over TLS so __Host- cookies work.
func newImportsEnv(t *testing.T) *importsEnv {
	t.Helper()

	pool := testutil.NewSeededDB(t)
	rdb := testutil.NewValkey(t)
	hasher := password.NewHasher(cheapPasswordParams, 0)
	logger := testLogger()

	st := store.NewStore(pool)
	authService, err := auth.NewService(
		st,
		rdb,
		hasher,
		ratelimit.New(rdb),
		auth.Config{RecoverySecret: testRecoverySecret},
		logger,
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	usersService := users.NewService(st, authService, hasher, newUsersTestBox(t))
	registry := jobs.NewRegistry()
	jobs.Register(registry, &users.ImportCommitWorker{})
	insertClient, err := jobs.NewInsertClient(pool, registry, logger)
	if err != nil {
		t.Fatalf("NewInsertClient: %v", err)
	}
	importer := users.NewImporter(usersService, insertClient)

	handler := NewRouter(RouterDeps{
		Logger:         logger,
		Auth:           authService,
		Users:          usersService,
		Imports:        importer,
		Limiter:        ratelimit.New(rdb),
		AllowedOrigins: []string{"http://localhost:3020"},
	})

	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	env := &importsEnv{
		usersEnv: &usersEnv{flowEnv: &flowEnv{
			t: t, pool: pool, server: server, hasher: hasher,
		}},
		importer:     importer,
		insertClient: insertClient,
		pool:         pool,
	}
	env.seedAcademics()
	if _, err := pool.Exec(context.Background(),
		"UPDATE academic_years SET is_current = true WHERE id = $1", env.yearID); err != nil {
		t.Fatalf("tandai tahun ajaran berjalan: %v", err)
	}
	return env
}

// uploadImport posts a multipart CSV with the requested declared content type.
func (e *importsEnv) uploadImport(
	client *http.Client,
	csrf, kind, filename, content, contentType string,
) (*http.Response, []byte) {
	e.t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("kind", kind); err != nil {
		e.t.Fatalf("tulis field kind: %v", err)
	}
	if contentType == "" {
		contentType = "text/csv"
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		e.t.Fatalf("buat part file: %v", err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		e.t.Fatalf("tulis isi CSV: %v", err)
	}
	if err := writer.Close(); err != nil {
		e.t.Fatalf("tutup multipart: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, e.server.URL+importsPath, &body)
	if err != nil {
		e.t.Fatalf("buat request impor: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-CSRF-Token", csrf)

	res, err := client.Do(req)
	if err != nil {
		e.t.Fatalf("POST %s: %v", importsPath, err)
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		e.t.Fatalf("baca respons impor: %v", err)
	}
	return res, data
}

func decodeImportJob(t *testing.T, body []byte) apigen.ImportJob {
	t.Helper()

	var job apigen.ImportJob
	if err := json.Unmarshal(body, &job); err != nil {
		t.Fatalf("decode ImportJob: %v (body %s)", err, body)
	}
	return job
}

func (e *importsEnv) runImportCommit(t *testing.T, jobID uuid.UUID, actorID uuid.UUID) {
	t.Helper()

	worker := &users.ImportCommitWorker{Importer: e.importer, Logger: testLogger()}
	job := &river.Job[users.ImportCommitArgs]{
		Args: users.ImportCommitArgs{
			ImportJobID: jobID,
			ActorID:     actorID,
			ActorIP:     "203.0.113.7",
		},
	}
	if err := worker.Work(context.Background(), job); err != nil {
		t.Fatalf("Work import commit: %v", err)
	}
}

func (e *importsEnv) countClassVaultEntries(t *testing.T, classID uuid.UUID) int {
	t.Helper()

	var count int
	if err := e.pool.QueryRow(context.Background(), `
		SELECT count(*)
		FROM initial_password_vault v
		JOIN enrollments en ON en.user_id = v.user_id
		WHERE en.class_id = $1`, classID).Scan(&count); err != nil {
		t.Fatalf("hitung brankas kelas: %v", err)
	}
	return count
}

func hasImportRowError(job apigen.ImportJob, row int, field, messagePart string) bool {
	for _, rowError := range job.Errors {
		if rowError.Row == row && rowError.Field == field &&
			strings.Contains(rowError.Message, messagePart) {
			return true
		}
	}
	return false
}

func TestImportsHTTP_AdminUploadsValidCSVThenCommits(t *testing.T) {
	env := newImportsEnv(t)

	adminID := env.createAdmin("admin.impor", "Admin Impor", "admin")
	adminClient, adminCSRF := env.loginAs("admin.impor", usersTestPassword)

	content := strings.Join([]string{
		"nisn,nama,kelas",
		"1000011001,Budi Santoso,X RPL A",
		"1000011002,Siti Aminah,X RPL A",
		"1000011003,Ahmad Fauzi,X RPL A",
	}, "\n")

	res, body := env.uploadImport(adminClient, adminCSRF, "students", "siswa.csv", content, "")
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("POST /admin/imports status = %d, body %s", res.StatusCode, body)
	}

	job := decodeImportJob(t, body)
	if job.Status != apigen.ImportStatusReady {
		t.Fatalf("status = %q, errors %+v, mau ready", job.Status, job.Errors)
	}
	if job.Kind != apigen.ImportKindStudents {
		t.Errorf("kind = %q, mau students", job.Kind)
	}
	if job.TotalRows == nil || *job.TotalRows != 3 {
		t.Fatalf("totalRows = %v, mau 3", job.TotalRows)
	}
	if job.ValidRows == nil || *job.ValidRows != 3 {
		t.Fatalf("validRows = %v, mau 3", job.ValidRows)
	}
	if job.ErrorCountTotal != 0 || len(job.Errors) != 0 {
		t.Fatalf("errors = %+v, mau kosong", job.Errors)
	}
	if got := job.ByClass["X RPL A"]; got != 3 {
		t.Fatalf("byClass = %+v, mau X RPL A: 3", job.ByClass)
	}
	if job.Created != nil {
		t.Fatalf("created = %v, mau kosong sebelum commit", *job.Created)
	}
	if job.CreatedBy != adminID {
		t.Errorf("createdBy = %s, mau %s", job.CreatedBy, adminID)
	}

	// The detail endpoint exposes the same preview.
	res, body = env.do(adminClient, http.MethodGet,
		importsPath+"/"+job.Id.String(), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /admin/imports/{id} status = %d, body %s", res.StatusCode, body)
	}
	detail := decodeImportJob(t, body)
	if got := detail.ByClass["X RPL A"]; got != 3 {
		t.Fatalf("detail byClass = %+v, mau X RPL A: 3", detail.ByClass)
	}

	res, body = env.do(adminClient, http.MethodPost,
		importsPath+"/"+job.Id.String()+"/commit", nil, adminCSRF, nil)
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("POST commit status = %d, body %s", res.StatusCode, body)
	}
	committing := decodeImportJob(t, body)
	if committing.Status != apigen.ImportStatusCommitting {
		t.Fatalf("status = %q, mau committing", committing.Status)
	}

	env.runImportCommit(t, job.Id, adminID)

	res, body = env.do(adminClient, http.MethodGet,
		importsPath+"/"+job.Id.String(), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET setelah commit status = %d, body %s", res.StatusCode, body)
	}
	committed := decodeImportJob(t, body)
	if committed.Status != apigen.ImportStatusCommitted {
		t.Fatalf("status = %q, mau committed", committed.Status)
	}
	if committed.Created == nil || *committed.Created != 3 {
		t.Fatalf("created = %v, mau 3", committed.Created)
	}
	if committed.CommittedAt == nil {
		t.Fatal("committedAt kosong")
	}
	if len(committed.Skipped) != 0 {
		t.Fatalf("skipped = %+v, mau kosong", committed.Skipped)
	}

	if got := env.countClassVaultEntries(t, env.rplClassID); got != 3 {
		t.Fatalf("entri brankas kelas = %d, mau 3", got)
	}

	// The account cards endpoint proves the imported accounts can receive
	// printed initial passwords.
	res, body = env.do(adminClient, http.MethodGet,
		accountCardsPath+"?classId="+env.rplClassID.String(), nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET account cards status = %d, body %s", res.StatusCode, body)
	}
	if got := res.Header.Get("Content-Type"); got != "application/pdf" {
		t.Errorf("Content-Type = %q, mau application/pdf", got)
	}
	if !bytes.HasPrefix(body, []byte("%PDF")) {
		t.Fatalf("kartu akun bukan PDF: %q", body[:min(len(body), 16)])
	}

	// The list endpoint shows the job for its creator.
	res, body = env.do(adminClient, http.MethodGet, importsPath, nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET /admin/imports status = %d, body %s", res.StatusCode, body)
	}
	var list apigen.ImportJobList
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode daftar impor: %v (body %s)", err, body)
	}
	if len(list.Items) != 1 || list.Items[0].Id != job.Id {
		t.Fatalf("daftar impor = %+v, mau satu job %s", list.Items, job.Id)
	}
}

func TestImportsHTTP_InvalidRowsAndCommitTransition(t *testing.T) {
	env := newImportsEnv(t)

	env.createAdmin("admin.impor.invalid", "Admin Impor Invalid", "admin")
	adminClient, adminCSRF := env.loginAs("admin.impor.invalid", usersTestPassword)

	content := strings.Join([]string{
		"nisn,nama,kelas",
		"123,Budi,X RPL A",
		"1000012001,Siti,X RPL A",
	}, "\n")

	res, body := env.uploadImport(adminClient, adminCSRF, "students", "siswa.csv", content, "")
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("upload status = %d, body %s", res.StatusCode, body)
	}
	job := decodeImportJob(t, body)
	if job.Status != apigen.ImportStatusInvalid {
		t.Fatalf("status = %q, mau invalid", job.Status)
	}
	if job.ValidRows == nil || *job.ValidRows != 1 {
		t.Fatalf("validRows = %v, mau 1", job.ValidRows)
	}
	if job.ErrorCountTotal != 1 {
		t.Fatalf("errorCountTotal = %d, mau 1", job.ErrorCountTotal)
	}
	if !hasImportRowError(job, 1, "nisn", "10 digit") {
		t.Fatalf("error NISN tidak ada: %+v", job.Errors)
	}

	res, body = env.do(adminClient, http.MethodPost,
		importsPath+"/"+job.Id.String()+"/commit", nil, adminCSRF, nil)
	requireProblem(t, res, body, http.StatusConflict, "INVALID_TRANSITION")
}

func TestImportsHTTP_ScopedAdminOnlyImportsOwnMajor(t *testing.T) {
	env := newImportsEnv(t)

	env.createAdmin("admin.impor.global", "Admin Impor Global", "admin")
	globalClient, globalCSRF := env.loginAs("admin.impor.global", usersTestPassword)

	content := strings.Join([]string{
		"nisn,nama,kelas",
		"1000013001,Siswa RPL,X RPL A",
		"1000013002,Siswa TKJ,X TKJ A",
	}, "\n")
	res, body := env.uploadImport(globalClient, globalCSRF, "students", "siswa.csv", content, "")
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("upload global status = %d, body %s", res.StatusCode, body)
	}
	globalJob := decodeImportJob(t, body)
	if globalJob.Status != apigen.ImportStatusReady {
		t.Fatalf("status global = %q, errors %+v", globalJob.Status, globalJob.Errors)
	}

	env.createMajorAdmin("admin.impor.rpl", "Admin Impor RPL", "rpl")
	scopedClient, scopedCSRF := env.loginAs("admin.impor.rpl", usersTestPassword)

	res, body = env.uploadImport(scopedClient, scopedCSRF, "students", "siswa.csv", content, "")
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("upload scoped status = %d, body %s", res.StatusCode, body)
	}
	scopedJob := decodeImportJob(t, body)
	if scopedJob.Status != apigen.ImportStatusInvalid {
		t.Fatalf("status scoped = %q, mau invalid", scopedJob.Status)
	}
	if !hasImportRowError(scopedJob, 2, "kelas", "Kamu tidak berwenang mengimpor ke kelas ini.") {
		t.Fatalf("error otorisasi tidak ada: %+v", scopedJob.Errors)
	}

	// A scoped admin cannot see another actor's job.
	res, body = env.do(scopedClient, http.MethodGet,
		importsPath+"/"+globalJob.Id.String(), nil, "", nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")

	res, body = env.do(scopedClient, http.MethodGet, importsPath, nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list scoped status = %d, body %s", res.StatusCode, body)
	}
	var list apigen.ImportJobList
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("decode daftar impor: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].Id != scopedJob.Id {
		t.Fatalf("daftar scoped = %+v, mau hanya %s", list.Items, scopedJob.Id)
	}

	// Teachers require a global grant, so a scoped upload is not found.
	res, body = env.uploadImport(scopedClient, scopedCSRF, "teachers", "guru.csv",
		"nip,nama\n1987000001,Guru RPL\n", "")
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")
}

func TestImportsHTTP_StudentCannotAccessImports(t *testing.T) {
	env := newImportsEnv(t)

	env.createActivatedStudent("1000014001", "siswa.impor", usersTestPassword)
	studentClient, studentCSRF := env.loginAs("siswa.impor", usersTestPassword)

	res, body := env.uploadImport(studentClient, studentCSRF, "students", "siswa.csv",
		"nisn,nama,kelas\n1000014002,Budi,X RPL A\n", "")
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")

	res, body = env.do(studentClient, http.MethodGet, importsPath, nil, "", nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")
}

func TestImportsHTTP_RejectsBinaryAndOversizedFiles(t *testing.T) {
	env := newImportsEnv(t)

	env.createAdmin("admin.impor.berkas", "Admin Impor Berkas", "admin")
	adminClient, adminCSRF := env.loginAs("admin.impor.berkas", usersTestPassword)

	res, body := env.uploadImport(adminClient, adminCSRF, "students", "siswa.pdf",
		"nisn,nama,kelas\n1000015001,Budi,X RPL A\n", "application/pdf")
	requireProblem(t, res, body, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE")

	large := "nisn,nama,kelas\n" + strings.Repeat("1", 2<<20)
	res, body = env.uploadImport(adminClient, adminCSRF, "students", "besar.csv", large, "text/csv")
	requireProblem(t, res, body, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE")
}
