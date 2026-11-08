package http

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/media"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/middleware"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/jobs"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/password"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/storage"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

const mediaTestPassword = "sandi-uji-123"

type mediaHTTPEnv struct {
	*flowEnv

	s3 *storage.Client
}

// newMediaHTTPEnv builds the real router with auth and media services on a
// seeded database, Valkey, and RustFS.
func newMediaHTTPEnv(t *testing.T) *mediaHTTPEnv {
	t.Helper()

	pool := testutil.NewSeededDB(t)
	rdb := testutil.NewValkey(t)
	s3Client := testutil.NewS3(t)
	hasher := password.NewHasher(cheapPasswordParams, 0)
	logger := testLogger()

	dataStore := store.NewStore(pool)
	authService, err := auth.NewService(
		dataStore,
		rdb,
		hasher,
		ratelimit.New(rdb),
		auth.Config{RecoverySecret: testRecoverySecret},
		logger,
	)
	if err != nil {
		t.Fatalf("NewService auth: %v", err)
	}
	jobRegistry := jobs.NewRegistry()
	jobs.Register(jobRegistry, &media.ProcessWorker{Logger: logger})
	jobClient, err := jobs.NewInsertClient(pool, jobRegistry, logger)
	if err != nil {
		t.Fatalf("NewInsertClient: %v", err)
	}

	handler := NewRouter(RouterDeps{
		Logger:         logger,
		Auth:           authService,
		Media:          media.NewService(dataStore, s3Client, jobClient, ratelimit.New(rdb), rdb),
		Limiter:        ratelimit.New(rdb),
		AllowedOrigins: []string{"http://localhost:3020"},
	})

	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	return &mediaHTTPEnv{
		flowEnv: &flowEnv{t: t, pool: pool, server: server, hasher: hasher},
		s3:      s3Client,
	}
}

func (e *mediaHTTPEnv) uploadMedia(c *http.Client, csrf, purpose, filename string, content []byte) (*http.Response, []byte) {
	e.t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writer.WriteField("purpose", purpose); err != nil {
		e.t.Fatalf("tulis field purpose: %v", err)
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		e.t.Fatalf("buat field file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		e.t.Fatalf("tulis isi file: %v", err)
	}
	if err := writer.Close(); err != nil {
		e.t.Fatalf("tutup multipart writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, e.server.URL+"/api/v1/media", &buf)
	if err != nil {
		e.t.Fatalf("buat request unggah: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if csrf != "" {
		req.Header.Set(middleware.CSRFHeaderName, csrf)
	}

	res, err := c.Do(req)
	if err != nil {
		e.t.Fatalf("POST /media: %v", err)
	}
	defer res.Body.Close()

	body := new(bytes.Buffer)
	if _, err := body.ReadFrom(res.Body); err != nil {
		e.t.Fatalf("baca respons unggah: %v", err)
	}
	return res, body.Bytes()
}

// loginAs signs in and returns the client plus the rotated CSRF token.
func (e *mediaHTTPEnv) loginAs(identifier, plain string) (*http.Client, string) {
	e.t.Helper()

	client := e.client()
	csrf := e.fetchCSRF(client)
	res, body := e.login(client, identifier, plain, csrf)
	if res.StatusCode != http.StatusOK {
		e.t.Fatalf("login %s status = %d, body %s", identifier, res.StatusCode, body)
	}
	return client, e.csrfFromJar(client)
}

func mediaTestPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestMediaHTTP_UploadReadAndDelete(t *testing.T) {
	env := newMediaHTTPEnv(t)
	env.createActivatedStudent("1000000011", "media.siswa", mediaTestPassword)
	content := mediaTestPNG(t)

	client, csrf := env.loginAs("media.siswa", mediaTestPassword)

	res, body := env.uploadMedia(client, csrf, "content", "foto.png", content)
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("POST /media status = %d, mau 202 (body %s)", res.StatusCode, body)
	}

	var asset apigen.MediaAsset
	if err := json.Unmarshal(body, &asset); err != nil {
		t.Fatalf("decode MediaAsset: %v (body %s)", err, body)
	}
	if asset.Kind != apigen.MediaKindImage || asset.Purpose != apigen.MediaPurposeContent {
		t.Errorf("asset kind/purpose = %s/%s, mau image/content", asset.Kind, asset.Purpose)
	}
	if asset.MimeType != "image/png" || asset.SizeBytes != int64(len(content)) {
		t.Errorf("asset mime/size = %s/%d, mau image/png/%d", asset.MimeType, asset.SizeBytes, len(content))
	}
	if asset.Status != apigen.MediaStatusUploaded {
		t.Errorf("status = %s, mau uploaded", asset.Status)
	}
	if asset.OriginalFilename == nil || *asset.OriginalFilename != "foto.png" {
		t.Errorf("originalFilename = %v, mau foto.png", asset.OriginalFilename)
	}

	path := "/api/v1/media/" + asset.Id.String()
	res, body = env.do(client, http.MethodGet, path, nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET media status = %d, mau 200 (body %s)", res.StatusCode, body)
	}
	var fetched apigen.MediaAsset
	if err := json.Unmarshal(body, &fetched); err != nil {
		t.Fatalf("decode GET media: %v (body %s)", err, body)
	}
	if fetched.Id != asset.Id {
		t.Errorf("GET media id = %s, mau %s", fetched.Id, asset.Id)
	}

	res, streamed := env.do(client, http.MethodGet, path+"/original", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET varian status = %d, mau 200 (body %s)", res.StatusCode, streamed)
	}
	if !bytes.Equal(streamed, content) {
		t.Error("isi varian original berbeda dari unggahan")
	}
	if got := res.Header.Get("Content-Type"); got != "image/png" {
		t.Errorf("Content-Type varian = %q, mau image/png", got)
	}
	if got := res.Header.Get("Cache-Control"); got != "private, max-age=300" {
		t.Errorf("Cache-Control varian = %q, mau private, max-age=300", got)
	}

	// Another user can neither read the asset nor its original.
	env.createActivatedStudent("1000000012", "media.lain", mediaTestPassword)
	otherClient, _ := env.loginAs("media.lain", mediaTestPassword)
	res, body = env.do(otherClient, http.MethodGet, path, nil, "", nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")
	res, body = env.do(otherClient, http.MethodGet, path+"/original", nil, "", nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")

	// The owner can delete an unreferenced asset.
	res, body = env.do(client, http.MethodDelete, path, nil, csrf, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE media status = %d, mau 204 (body %s)", res.StatusCode, body)
	}
	res, body = env.do(client, http.MethodGet, path, nil, "", nil)
	requireProblem(t, res, body, http.StatusNotFound, "NOT_FOUND")
}

func TestMediaHTTP_AnonymousUploadReturnsUnauthenticated(t *testing.T) {
	env := newMediaHTTPEnv(t)

	client := env.client()
	csrf := env.fetchCSRF(client)

	res, body := env.uploadMedia(client, csrf, "content", "foto.png", mediaTestPNG(t))
	requireProblem(t, res, body, http.StatusUnauthorized, "UNAUTHENTICATED")
}

func TestMediaHTTP_UploadWithoutCSRFIsRejected(t *testing.T) {
	env := newMediaHTTPEnv(t)
	env.createActivatedStudent("1000000013", "media.csrf", mediaTestPassword)

	client, _ := env.loginAs("media.csrf", mediaTestPassword)
	res, body := env.uploadMedia(client, "", "content", "foto.png", mediaTestPNG(t))
	requireProblem(t, res, body, http.StatusForbidden, "CSRF_FAILED")
}
