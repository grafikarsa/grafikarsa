package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/middleware"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/password"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

// cheapPasswordParams keeps Argon2 fast in tests.
var cheapPasswordParams = password.Params{
	Memory:      1024,
	Iterations:  1,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

type flowEnv struct {
	t      *testing.T
	pool   *pgxpool.Pool
	server *httptest.Server
	hasher *password.Hasher
}

// newFlowEnv builds the real router with a real auth service, seeded database,
// and Valkey, served over TLS so __Host- cookies are accepted.
func newFlowEnv(t *testing.T) *flowEnv {
	t.Helper()

	pool := testutil.NewSeededDB(t)
	rdb := testutil.NewValkey(t)
	hasher := password.NewHasher(cheapPasswordParams, 0)
	logger := testLogger()

	authService, err := auth.NewService(
		store.NewStore(pool),
		rdb,
		hasher,
		ratelimit.New(rdb),
		auth.Config{},
		logger,
	)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	handler := NewRouter(RouterDeps{
		Logger:         logger,
		Auth:           authService,
		Limiter:        ratelimit.New(rdb),
		AllowedOrigins: []string{"http://localhost:3020"},
	})

	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	return &flowEnv{t: t, pool: pool, server: server, hasher: hasher}
}

type flowUserSpec struct {
	accountType       string
	nisn              *string
	username          *string
	displayName       string
	password          string
	passwordKind      string
	passwordExpiresAt *time.Time
	activatedAt       *time.Time
}

func (e *flowEnv) createUser(spec flowUserSpec) uuid.UUID {
	e.t.Helper()

	if spec.accountType == "" {
		spec.accountType = "student"
	}
	if spec.displayName == "" {
		spec.displayName = "Pengguna Uji"
	}
	if spec.passwordKind == "" {
		spec.passwordKind = "user"
	}

	hash, err := e.hasher.Hash(context.Background(), spec.password)
	if err != nil {
		e.t.Fatalf("hash password: %v", err)
	}

	var id uuid.UUID
	err = e.pool.QueryRow(context.Background(), `
		INSERT INTO users (
			account_type, status, nisn, username, display_name,
			password_hash, password_kind, password_expires_at, activated_at
		) VALUES ($1, 'active', $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		spec.accountType, spec.nisn, spec.username, spec.displayName,
		hash, spec.passwordKind, spec.passwordExpiresAt, spec.activatedAt,
	).Scan(&id)
	if err != nil {
		e.t.Fatalf("insert user: %v", err)
	}

	return id
}

func (e *flowEnv) createActivatedStudent(nisn, username, plain string) uuid.UUID {
	e.t.Helper()

	now := time.Now().UTC()
	return e.createUser(flowUserSpec{
		nisn:        &nisn,
		username:    &username,
		displayName: "Siswa Aktif",
		password:    plain,
		activatedAt: &now,
	})
}

func (e *flowEnv) createPendingStudent(nisn, displayName, plain string) uuid.UUID {
	e.t.Helper()

	expires := time.Now().UTC().Add(24 * time.Hour)
	return e.createUser(flowUserSpec{
		nisn:              &nisn,
		displayName:       displayName,
		password:          plain,
		passwordKind:      "initial",
		passwordExpiresAt: &expires,
	})
}

func (e *flowEnv) createTemporaryPasswordStudent(nisn, username, displayName, plain string) uuid.UUID {
	e.t.Helper()

	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)
	return e.createUser(flowUserSpec{
		nisn:              &nisn,
		username:          &username,
		displayName:       displayName,
		password:          plain,
		passwordKind:      "temporary",
		passwordExpiresAt: &expires,
		activatedAt:       &now,
	})
}

func (e *flowEnv) client() *http.Client {
	e.t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		e.t.Fatalf("create cookie jar: %v", err)
	}

	return &http.Client{
		Transport: e.server.Client().Transport,
		Jar:       jar,
		Timeout:   15 * time.Second,
	}
}

func (e *flowEnv) do(c *http.Client, method, path string, body any, csrf string, headers map[string]string) (*http.Response, []byte) {
	e.t.Helper()

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			e.t.Fatalf("encode request body: %v", err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, e.server.URL+path, reader)
	if err != nil {
		e.t.Fatalf("create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if csrf != "" {
		req.Header.Set(middleware.CSRFHeaderName, csrf)
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}

	res, err := c.Do(req)
	if err != nil {
		e.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		e.t.Fatalf("read response body: %v", err)
	}

	return res, data
}

func (e *flowEnv) fetchCSRF(c *http.Client) string {
	e.t.Helper()

	res, body := e.do(c, http.MethodGet, "/api/v1/auth/csrf", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		e.t.Fatalf("GET /auth/csrf status = %d, body %s", res.StatusCode, body)
	}

	var payload struct {
		CsrfToken string `json:"csrfToken"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		e.t.Fatalf("decode csrf body: %v", err)
	}
	if payload.CsrfToken == "" {
		e.t.Fatal("csrfToken is empty")
	}

	return payload.CsrfToken
}

// csrfFromJar reads the current CSRF cookie, which the login handler rotates.
func (e *flowEnv) csrfFromJar(c *http.Client) string {
	e.t.Helper()

	base, err := url.Parse(e.server.URL)
	if err != nil {
		e.t.Fatalf("parse server URL: %v", err)
	}
	for _, cookie := range c.Jar.Cookies(base) {
		if cookie.Name == middleware.CSRFCookieName {
			return cookie.Value
		}
	}

	e.t.Fatal("csrf cookie not found in jar")
	return ""
}

func (e *flowEnv) login(c *http.Client, identifier, plain, csrf string) (*http.Response, []byte) {
	e.t.Helper()

	return e.do(c, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"identifier": identifier,
		"password":   plain,
	}, csrf, nil)
}

func decodeSession(t *testing.T, body []byte) apigen.Session {
	t.Helper()

	var session apigen.Session
	if err := json.Unmarshal(body, &session); err != nil {
		t.Fatalf("decode session: %v (body %s)", err, body)
	}
	return session
}

func findCookie(res *http.Response, name string) *http.Cookie {
	for _, cookie := range res.Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func requireProblem(t *testing.T, res *http.Response, body []byte, status int, code string) {
	t.Helper()

	if res.StatusCode != status {
		t.Fatalf("status = %d, want %d (body %s)", res.StatusCode, status, body)
	}
	if got := res.Header.Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want application/problem+json", got)
	}

	var problemBody struct {
		Code   string `json:"code"`
		Status int    `json:"status"`
	}
	if err := json.Unmarshal(body, &problemBody); err != nil {
		t.Fatalf("decode problem body: %v (body %s)", err, body)
	}
	if problemBody.Code != code {
		t.Fatalf("code = %q, want %q", problemBody.Code, code)
	}
	if problemBody.Status != status {
		t.Errorf("problem status = %d, want %d", problemBody.Status, status)
	}
}

func TestAuthFlow_LoginSessionLogout(t *testing.T) {
	env := newFlowEnv(t)
	userID := env.createActivatedStudent("1000000001", "siswa.aktif", "sandi-uji-123")

	c := env.client()
	csrf := env.fetchCSRF(c)

	res, body := env.login(c, "1000000001", "sandi-uji-123", csrf)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, body %s", res.StatusCode, body)
	}
	if got := res.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("login Content-Type = %q, want application/json", got)
	}

	session := decodeSession(t, body)
	if session.NextStep != apigen.NextStepNone {
		t.Errorf("nextStep = %q, want none", session.NextStep)
	}
	if session.User.Id != userID {
		t.Errorf("user id = %s, want %s", session.User.Id, userID)
	}
	if session.User.Username == nil || *session.User.Username != "siswa.aktif" {
		t.Errorf("username = %v, want siswa.aktif", session.User.Username)
	}
	if session.IsMigratedAccount == nil || *session.IsMigratedAccount {
		t.Errorf("isMigratedAccount = %v, want false", session.IsMigratedAccount)
	}

	sessionCookie := findCookie(res, middleware.SessionCookieName)
	if sessionCookie == nil {
		t.Fatal("session cookie was not set")
	}
	if !sessionCookie.HttpOnly {
		t.Error("session cookie is not HttpOnly")
	}
	if !sessionCookie.Secure {
		t.Error("session cookie is not Secure")
	}

	res, body = env.do(c, http.MethodGet, "/api/v1/auth/session", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET session status = %d, body %s", res.StatusCode, body)
	}
	if got := decodeSession(t, body); got.User.Id != userID {
		t.Errorf("session user id = %s, want %s", got.User.Id, userID)
	}

	res, body = env.do(c, http.MethodGet, "/api/v1/auth/sessions", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET sessions status = %d, body %s", res.StatusCode, body)
	}
	var devices apigen.SessionDeviceList
	if err := json.Unmarshal(body, &devices); err != nil {
		t.Fatalf("decode sessions: %v", err)
	}
	if len(devices.Items) != 1 {
		t.Fatalf("sessions = %d, want 1", len(devices.Items))
	}
	if !devices.Items[0].Current {
		t.Error("current session is not marked as current")
	}

	csrf = env.csrfFromJar(c)
	res, body = env.do(c, http.MethodPost, "/api/v1/auth/logout", nil, csrf, nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("logout status = %d, body %s", res.StatusCode, body)
	}

	res, body = env.do(c, http.MethodGet, "/api/v1/auth/session", nil, "", nil)
	requireProblem(t, res, body, http.StatusUnauthorized, "UNAUTHENTICATED")
}

func TestAuthFlow_LoginRequiresCSRF(t *testing.T) {
	env := newFlowEnv(t)
	env.createActivatedStudent("1000000002", "siswa.csrf", "sandi-uji-123")

	c := env.client()
	csrf := env.fetchCSRF(c)

	res, body := env.login(c, "1000000002", "sandi-uji-123", "")
	requireProblem(t, res, body, http.StatusForbidden, "CSRF_FAILED")

	res, body = env.do(c, http.MethodPost, "/api/v1/auth/login", map[string]string{
		"identifier": "1000000002",
		"password":   "sandi-uji-123",
	}, csrf, map[string]string{"Origin": "https://evil.example"})
	requireProblem(t, res, body, http.StatusForbidden, "CSRF_FAILED")
}

func TestAuthFlow_WrongPasswordReturnsInvalidCredentials(t *testing.T) {
	env := newFlowEnv(t)
	env.createActivatedStudent("1000000003", "siswa.salah", "sandi-uji-123")

	c := env.client()
	csrf := env.fetchCSRF(c)

	res, body := env.login(c, "1000000003", "sandi-salah", csrf)
	requireProblem(t, res, body, http.StatusUnauthorized, "INVALID_CREDENTIALS")
}

func TestAuthFlow_NotActivatedUserIsGated(t *testing.T) {
	env := newFlowEnv(t)
	env.createPendingStudent("1000000004", "Siswa Baru", "sandi-awal-123")

	c := env.client()
	csrf := env.fetchCSRF(c)

	res, body := env.login(c, "1000000004", "sandi-awal-123", csrf)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, body %s", res.StatusCode, body)
	}
	session := decodeSession(t, body)
	if session.NextStep != apigen.NextStepActivate {
		t.Fatalf("nextStep = %q, want activate", session.NextStep)
	}
	if session.User.Username != nil {
		t.Errorf("username = %v, want nil before activation", session.User.Username)
	}

	res, body = env.do(c, http.MethodGet, "/api/v1/auth/sessions", nil, "", nil)
	requireProblem(t, res, body, http.StatusForbidden, "ACTIVATION_REQUIRED")

	res, body = env.do(c, http.MethodGet, "/api/v1/auth/session", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET session status = %d, body %s", res.StatusCode, body)
	}
}

func TestAuthFlow_TemporaryPasswordUserIsGated(t *testing.T) {
	env := newFlowEnv(t)
	env.createTemporaryPasswordStudent("1000000005", "siswa.sementara", "Siswa Sementara", "sandi-temp-123")

	c := env.client()
	csrf := env.fetchCSRF(c)

	res, body := env.login(c, "1000000005", "sandi-temp-123", csrf)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, body %s", res.StatusCode, body)
	}
	session := decodeSession(t, body)
	if session.NextStep != apigen.NextStepChangePassword {
		t.Fatalf("nextStep = %q, want change_password", session.NextStep)
	}

	res, body = env.do(c, http.MethodGet, "/api/v1/auth/sessions", nil, "", nil)
	requireProblem(t, res, body, http.StatusForbidden, "PASSWORD_CHANGE_REQUIRED")
}

func TestAuthFlow_GarbageSessionCookieIsCleared(t *testing.T) {
	env := newFlowEnv(t)
	c := env.client()

	base, err := url.Parse(env.server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	c.Jar.SetCookies(base, []*http.Cookie{{
		Name:   middleware.SessionCookieName,
		Value:  "not-a-real-session",
		Path:   "/",
		Secure: true,
	}})

	res, body := env.do(c, http.MethodGet, "/api/v1/auth/session", nil, "", nil)
	requireProblem(t, res, body, http.StatusUnauthorized, "UNAUTHENTICATED")

	var clearHeader string
	for _, header := range res.Header.Values("Set-Cookie") {
		if strings.HasPrefix(header, middleware.SessionCookieName+"=") {
			clearHeader = header
			break
		}
	}
	if clearHeader == "" {
		t.Fatal("session cookie was not cleared")
	}
	if !strings.Contains(clearHeader, "Max-Age=0") {
		t.Errorf("clear cookie = %q, want Max-Age=0", clearHeader)
	}
}

func TestAuthFlow_RevokeOthersLogsOutOtherClients(t *testing.T) {
	env := newFlowEnv(t)
	env.createActivatedStudent("1000000006", "siswa.multidevice", "sandi-uji-123")

	clientA := env.client()
	csrfA := env.fetchCSRF(clientA)
	if res, body := env.login(clientA, "1000000006", "sandi-uji-123", csrfA); res.StatusCode != http.StatusOK {
		t.Fatalf("login A status = %d, body %s", res.StatusCode, body)
	}

	clientB := env.client()
	csrfB := env.fetchCSRF(clientB)
	if res, body := env.login(clientB, "1000000006", "sandi-uji-123", csrfB); res.StatusCode != http.StatusOK {
		t.Fatalf("login B status = %d, body %s", res.StatusCode, body)
	}

	res, body := env.do(clientA, http.MethodPost, "/api/v1/auth/sessions/revoke-others", nil, env.csrfFromJar(clientA), nil)
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke-others status = %d, body %s", res.StatusCode, body)
	}

	res, body = env.do(clientB, http.MethodGet, "/api/v1/auth/session", nil, "", nil)
	requireProblem(t, res, body, http.StatusUnauthorized, "UNAUTHENTICATED")

	res, body = env.do(clientA, http.MethodGet, "/api/v1/auth/session", nil, "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("client A session status = %d, body %s", res.StatusCode, body)
	}
}

func TestAuthFlow_PlaceholderEndpointStillReturns501(t *testing.T) {
	env := newFlowEnv(t)
	env.createPendingStudent("1000000007", "Siswa Placeholder", "sandi-awal-123")

	c := env.client()
	csrf := env.fetchCSRF(c)
	if res, body := env.login(c, "1000000007", "sandi-awal-123", csrf); res.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, body %s", res.StatusCode, body)
	}

	res, body := env.do(c, http.MethodPost, "/api/v1/auth/activate",
		map[string]string{"username": "siswa.baru"}, env.csrfFromJar(c), nil)
	requireProblem(t, res, body, http.StatusNotImplemented, "NOT_IMPLEMENTED")
}
