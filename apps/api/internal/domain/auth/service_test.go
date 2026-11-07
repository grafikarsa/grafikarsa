package auth

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/password"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/securetoken"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

// cheapPasswordParams keeps Argon2 fast in tests. NewService and the users
// created by these tests share the same parameters, so no rehash happens
// unless a test explicitly stores a bcrypt hash.
var cheapPasswordParams = password.Params{
	Memory:      1024,
	Iterations:  1,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

// testRecoverySecret keys the HMAC of recovery codes in tests.
var testRecoverySecret = []byte("test-recovery-secret-0123456789abcdef")

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type testEnv struct {
	t      *testing.T
	pool   *pgxpool.Pool
	rdb    *redis.Client
	svc    *Service
	hasher *password.Hasher
	clock  *testClock
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	pool := testutil.NewSeededDB(t)
	rdb := testutil.NewValkey(t)
	hasher := password.NewHasher(cheapPasswordParams, 0)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	svc, err := NewService(store.NewStore(pool), rdb, hasher, ratelimit.New(rdb), Config{
		RecoverySecret: testRecoverySecret,
	}, logger)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	clock := &testClock{now: time.Now().UTC().Truncate(time.Second)}
	svc.now = clock.Now

	return &testEnv{t: t, pool: pool, rdb: rdb, svc: svc, hasher: hasher, clock: clock}
}

type userSpec struct {
	AccountType       string
	Status            string
	Nisn              *string
	StaffNumber       *string
	Username          *string
	DisplayName       string
	Password          string
	PasswordHash      string
	PasswordKind      string
	PasswordExpiresAt *time.Time
	ActivatedAt       *time.Time
	ContentBlockedAt  *time.Time
	DeactivatedAt     *time.Time
	LegacyV1ID        *uuid.UUID
}

func (e *testEnv) createUser(spec userSpec) uuid.UUID {
	e.t.Helper()

	if spec.AccountType == "" {
		spec.AccountType = "student"
	}
	if spec.Status == "" {
		spec.Status = "active"
	}
	if spec.DisplayName == "" {
		spec.DisplayName = "Pengguna Uji"
	}
	if spec.PasswordKind == "" {
		spec.PasswordKind = "user"
	}
	if spec.PasswordHash == "" {
		hash, err := e.hasher.Hash(context.Background(), spec.Password)
		if err != nil {
			e.t.Fatalf("hash password: %v", err)
		}
		spec.PasswordHash = hash
	}

	var id uuid.UUID
	err := e.pool.QueryRow(context.Background(), `
		INSERT INTO users (
			account_type, status, nisn, staff_number, username, display_name,
			password_hash, password_kind, password_expires_at, activated_at,
			content_blocked_at, deactivated_at, legacy_v1_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id`,
		spec.AccountType, spec.Status, spec.Nisn, spec.StaffNumber, spec.Username,
		spec.DisplayName, spec.PasswordHash, spec.PasswordKind,
		spec.PasswordExpiresAt, spec.ActivatedAt, spec.ContentBlockedAt,
		spec.DeactivatedAt, spec.LegacyV1ID,
	).Scan(&id)
	if err != nil {
		e.t.Fatalf("insert user: %v", err)
	}
	return id
}

func (e *testEnv) majorID(code string) uuid.UUID {
	e.t.Helper()

	var id uuid.UUID
	if err := e.pool.QueryRow(context.Background(),
		"SELECT id FROM majors WHERE code = $1", code).Scan(&id); err != nil {
		e.t.Fatalf("get major %s: %v", code, err)
	}
	return id
}

func (e *testEnv) assignRole(userID uuid.UUID, slug string, majorID *uuid.UUID) {
	e.t.Helper()

	_, err := e.pool.Exec(context.Background(), `
		INSERT INTO user_roles (user_id, role_id, major_id)
		SELECT $1, r.id, $2 FROM roles r WHERE r.slug = $3`,
		userID, majorID, slug)
	if err != nil {
		e.t.Fatalf("assign role %s: %v", slug, err)
	}
}

func (e *testEnv) login(identifier, plain string) *LoginResult {
	e.t.Helper()

	result, err := e.svc.Login(context.Background(), LoginInput{
		Identifier: identifier,
		Password:   plain,
		UserAgent:  "uji/1.0",
	})
	if err != nil {
		e.t.Fatalf("Login(%q): %v", identifier, err)
	}
	return result
}

func (e *testEnv) sessionLastSeen(sessionID uuid.UUID) time.Time {
	e.t.Helper()

	var lastSeen time.Time
	if err := e.pool.QueryRow(context.Background(),
		"SELECT last_seen_at FROM sessions WHERE id = $1", sessionID).Scan(&lastSeen); err != nil {
		e.t.Fatalf("get session last_seen_at: %v", err)
	}
	return lastSeen
}

func requireAppError(t *testing.T, err error, code string) *apperr.Error {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error %s, got nil", code)
	}
	appErr, ok := apperr.As(err)
	if !ok {
		t.Fatalf("expected app error %s, got %T: %v", code, err, err)
	}
	if appErr.Code != code {
		t.Fatalf("error code = %s, want %s", appErr.Code, code)
	}
	return appErr
}

func ptr[T any](v T) *T { return &v }

func TestLogin_ByNISNStaffNumberAndUsernameSucceeds(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	studentID := env.createUser(userSpec{
		Nisn:        ptr("1234567890"),
		Username:    ptr("siswa.satu"),
		DisplayName: "Siswa Satu",
		Password:    "sandi-uji-123",
		ActivatedAt: &activated,
	})
	teacherID := env.createUser(userSpec{
		AccountType: "teacher",
		StaffNumber: ptr("1987654321"),
		Username:    ptr("guru.satu"),
		DisplayName: "Guru Satu",
		Password:    "sandi-uji-123",
		ActivatedAt: &activated,
	})
	tests := []struct {
		name       string
		identifier string
		wantID     uuid.UUID
		wantType   string
	}{
		{name: "by nisn", identifier: "1234567890", wantID: studentID, wantType: "student"},
		{name: "by staff number", identifier: "1987654321", wantID: teacherID, wantType: "teacher"},
		{name: "by username case-insensitively", identifier: "SISWA.SATU", wantID: studentID, wantType: "student"},
		{name: "by username", identifier: "guru.satu", wantID: teacherID, wantType: "teacher"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := env.login(tt.identifier, "sandi-uji-123")
			if result.Token == "" {
				t.Fatal("Login returned an empty token")
			}

			p := result.Principal
			if p.UserID != tt.wantID {
				t.Fatalf("UserID = %s, want %s", p.UserID, tt.wantID)
			}
			if p.AccountType != tt.wantType {
				t.Fatalf("AccountType = %q, want %q", p.AccountType, tt.wantType)
			}
			if p.Status != "active" {
				t.Fatalf("Status = %q, want active", p.Status)
			}
			if p.NextStep != NextStepNone {
				t.Fatalf("NextStep = %q, want %q", p.NextStep, NextStepNone)
			}
			if p.Username == nil {
				t.Fatal("Username = nil, want set")
			}
			if p.DisplayName == "" {
				t.Fatal("DisplayName is empty")
			}
			if p.IsMigrated {
				t.Fatal("IsMigrated = true, want false")
			}
			wantExpiry := env.clock.Now().Add(defaultSessionAbsolute)
			if !p.ExpiresAt.Equal(wantExpiry) {
				t.Fatalf("ExpiresAt = %v, want %v", p.ExpiresAt, wantExpiry)
			}
			if p.SessionID == uuid.Nil {
				t.Fatal("SessionID = uuid.Nil")
			}
		})
	}
}

func TestLogin_WrongPasswordAndUnknownIdentifierReturnInvalidCredentials(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	env.createUser(userSpec{
		Nisn:        ptr("1234567890"),
		Username:    ptr("siswa.satu"),
		DisplayName: "Siswa Satu",
		Password:    "sandi-benar",
		ActivatedAt: &activated,
	})

	_, err := env.svc.Login(context.Background(), LoginInput{
		Identifier: "1234567890",
		Password:   "sandi-salah",
	})
	requireAppError(t, err, "INVALID_CREDENTIALS")

	_, err = env.svc.Login(context.Background(), LoginInput{
		Identifier: "tidak-ada",
		Password:   "apa-saja",
	})
	requireAppError(t, err, "INVALID_CREDENTIALS")

	_, err = env.svc.Login(context.Background(), LoginInput{
		Identifier: "   ",
		Password:   "apa-saja",
	})
	requireAppError(t, err, "INVALID_CREDENTIALS")
}

func TestLogin_RateLimitsIdentifierAfterFiveAttempts(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	env.createUser(userSpec{
		Nisn:        ptr("1234567890"),
		Username:    ptr("siswa.satu"),
		DisplayName: "Siswa Satu",
		Password:    "sandi-benar",
		ActivatedAt: &activated,
	})

	for attempt := 1; attempt <= 5; attempt++ {
		_, err := env.svc.Login(context.Background(), LoginInput{
			Identifier: "1234567890",
			Password:   "sandi-salah",
		})
		requireAppError(t, err, "INVALID_CREDENTIALS")
	}

	_, err := env.svc.Login(context.Background(), LoginInput{
		Identifier: "1234567890",
		Password:   "sandi-salah",
	})
	requireAppError(t, err, "RATE_LIMITED")
}

func TestLogin_InactiveUserReturnsAccountInactive(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	env.createUser(userSpec{
		Status:      "inactive",
		Nisn:        ptr("1234567890"),
		Username:    ptr("siswa.nonaktif"),
		DisplayName: "Siswa Nonaktif",
		Password:    "sandi-benar",
		ActivatedAt: &activated,
	})
	deactivatedAt := env.clock.Now()
	env.createUser(userSpec{
		Nisn:          ptr("1234567891"),
		Username:      ptr("siswa.deactivated"),
		DisplayName:   "Siswa Deactivated",
		Password:      "sandi-benar",
		ActivatedAt:   &activated,
		DeactivatedAt: &deactivatedAt,
	})

	tests := []struct {
		name       string
		identifier string
	}{
		{name: "inactive status", identifier: "1234567890"},
		{name: "deactivated_at set", identifier: "1234567891"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := env.svc.Login(context.Background(), LoginInput{
				Identifier: tt.identifier,
				Password:   "sandi-benar",
			})
			requireAppError(t, err, "ACCOUNT_INACTIVE")
		})
	}
}

func TestLogin_ExpiredInitialPasswordReturnsPasswordExpired(t *testing.T) {
	env := newTestEnv(t)
	expired := env.clock.Now().Add(-time.Hour)

	env.createUser(userSpec{
		Nisn:              ptr("1234567890"),
		DisplayName:       "Siswa Baru",
		Password:          "sandi-awal",
		PasswordKind:      "initial",
		PasswordExpiresAt: &expired,
	})

	_, err := env.svc.Login(context.Background(), LoginInput{
		Identifier: "1234567890",
		Password:   "sandi-awal",
	})
	requireAppError(t, err, "PASSWORD_EXPIRED")
}

func TestLogin_NotActivatedUserNeedsActivation(t *testing.T) {
	env := newTestEnv(t)
	future := env.clock.Now().Add(24 * time.Hour)

	env.createUser(userSpec{
		Nisn:              ptr("1234567890"),
		DisplayName:       "Siswa Baru",
		Password:          "sandi-awal",
		PasswordKind:      "initial",
		PasswordExpiresAt: &future,
	})

	result := env.login("1234567890", "sandi-awal")
	if result.Principal.NextStep != NextStepActivate {
		t.Fatalf("NextStep = %q, want %q", result.Principal.NextStep, NextStepActivate)
	}
	if result.Principal.Username != nil {
		t.Fatalf("Username = %v, want nil", result.Principal.Username)
	}
}

func TestLogin_TemporaryPasswordNeedsChange(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()
	future := env.clock.Now().Add(24 * time.Hour)

	env.createUser(userSpec{
		Nisn:              ptr("1234567890"),
		Username:          ptr("siswa.satu"),
		DisplayName:       "Siswa Satu",
		Password:          "sandi-sementara",
		PasswordKind:      "temporary",
		PasswordExpiresAt: &future,
		ActivatedAt:       &activated,
	})

	result := env.login("1234567890", "sandi-sementara")
	if result.Principal.NextStep != NextStepChangePassword {
		t.Fatalf("NextStep = %q, want %q", result.Principal.NextStep, NextStepChangePassword)
	}
}

func TestLogin_BcryptHashIsRehashedToArgon2id(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	bcryptHash, err := bcrypt.GenerateFromPassword([]byte("sandi-lama"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}

	userID := env.createUser(userSpec{
		Nisn:         ptr("1234567890"),
		Username:     ptr("siswa.lama"),
		DisplayName:  "Siswa Lama",
		PasswordHash: string(bcryptHash),
		ActivatedAt:  &activated,
	})

	env.login("1234567890", "sandi-lama")

	var stored string
	if err := env.pool.QueryRow(context.Background(),
		"SELECT password_hash FROM users WHERE id = $1", userID).Scan(&stored); err != nil {
		t.Fatalf("get password hash: %v", err)
	}
	if !strings.HasPrefix(stored, "$argon2id$") {
		t.Fatalf("password_hash = %q, want an Argon2id hash", stored)
	}
}

func TestAuthenticate_ReturnsPrincipalAndRejectsAfterLogout(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	env.createUser(userSpec{
		Nisn:        ptr("1234567890"),
		Username:    ptr("siswa.satu"),
		DisplayName: "Siswa Satu",
		Password:    "sandi-benar",
		ActivatedAt: &activated,
	})

	login := env.login("1234567890", "sandi-benar")

	principal, err := env.svc.Authenticate(context.Background(), login.Token)
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}
	if principal.UserID != login.Principal.UserID || principal.SessionID != login.Principal.SessionID {
		t.Fatalf("Authenticate principal = %+v, want session %s", principal, login.Principal.SessionID)
	}

	// The second call is served from the Valkey cache; its JSON round trip must
	// preserve the principal and grants.
	cached, err := env.svc.Authenticate(context.Background(), login.Token)
	if err != nil {
		t.Fatalf("Authenticate from cache: %v", err)
	}
	if cached.UserID != principal.UserID || cached.SessionID != principal.SessionID || !cached.Can("content.create") {
		t.Fatalf("cached principal = %+v, want user %s with content.create", cached, principal.UserID)
	}

	_, err = env.svc.Authenticate(context.Background(), "token-palsu")
	requireAppError(t, err, "UNAUTHENTICATED")

	if err := env.svc.Logout(context.Background(), principal); err != nil {
		t.Fatalf("Logout: %v", err)
	}

	_, err = env.svc.Authenticate(context.Background(), login.Token)
	requireAppError(t, err, "UNAUTHENTICATED")
}

func TestAuthenticate_ExpiredIdleSessionIsRevoked(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	env.createUser(userSpec{
		Nisn:        ptr("1234567890"),
		Username:    ptr("siswa.satu"),
		DisplayName: "Siswa Satu",
		Password:    "sandi-benar",
		ActivatedAt: &activated,
	})

	login := env.login("1234567890", "sandi-benar")

	if _, err := env.pool.Exec(context.Background(),
		"UPDATE sessions SET idle_expires_at = $2 WHERE id = $1",
		login.Principal.SessionID, env.clock.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("expire idle session: %v", err)
	}
	if err := env.rdb.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("flush valkey: %v", err)
	}

	_, err := env.svc.Authenticate(context.Background(), login.Token)
	requireAppError(t, err, "UNAUTHENTICATED")

	var reason *string
	if err := env.pool.QueryRow(context.Background(),
		"SELECT revoked_reason FROM sessions WHERE id = $1",
		login.Principal.SessionID).Scan(&reason); err != nil {
		t.Fatalf("get revoked_reason: %v", err)
	}
	if reason == nil || *reason != "expired" {
		t.Fatalf("revoked_reason = %v, want expired", reason)
	}
}

func TestPrincipal_TeacherHasImplicitRolePermissions(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	env.createUser(userSpec{
		AccountType: "teacher",
		Nisn:        ptr("1234567890"),
		Username:    ptr("guru.satu"),
		DisplayName: "Guru Satu",
		Password:    "sandi-benar",
		ActivatedAt: &activated,
	})

	principal := env.login("1234567890", "sandi-benar").Principal

	for _, permission := range []string{
		"content.create",
		"curation.manage",
		"assignment.manage",
		"assessment.score",
	} {
		if !principal.CanGlobally(permission) {
			t.Errorf("CanGlobally(%q) = false, want true", permission)
		}
	}
	if principal.Can("settings.manage") {
		t.Error("teacher unexpectedly has settings.manage")
	}

	if !slices.IsSortedFunc(principal.Grants, func(a, b Grant) int {
		return strings.Compare(a.Permission, b.Permission)
	}) {
		t.Errorf("Grants are not sorted by permission: %+v", principal.Grants)
	}
}

func TestPrincipal_ScopedModeratorGrant(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	userID := env.createUser(userSpec{
		Nisn:        ptr("1234567890"),
		Username:    ptr("siswa.moderator"),
		DisplayName: "Siswa Moderator",
		Password:    "sandi-benar",
		ActivatedAt: &activated,
	})

	majorA := env.majorID("rpl")
	majorB := env.majorID("dkv")
	env.assignRole(userID, "moderator", &majorA)

	principal := env.login("1234567890", "sandi-benar").Principal

	if !principal.Can("content.moderate") {
		t.Fatal("Can(content.moderate) = false, want true")
	}
	if principal.CanGlobally("content.moderate") {
		t.Fatal("CanGlobally(content.moderate) = true, want false")
	}
	if !principal.CanForMajor("content.moderate", majorA) {
		t.Fatal("CanForMajor(content.moderate, majorA) = false, want true")
	}
	if principal.CanForMajor("content.moderate", majorB) {
		t.Fatal("CanForMajor(content.moderate, majorB) = true, want false")
	}
}

func TestPrincipal_ContentBlockRemovesCreatePermission(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()
	blockedAt := env.clock.Now()

	userID := env.createUser(userSpec{
		AccountType:      "teacher",
		Nisn:             ptr("1234567890"),
		Username:         ptr("guru.blocked"),
		DisplayName:      "Guru Diblokir",
		Password:         "sandi-benar",
		ActivatedAt:      &activated,
		ContentBlockedAt: &blockedAt,
	})
	// An explicit global role also grants content.create; blocking must still
	// remove it.
	env.assignRole(userID, "admin", nil)

	principal := env.login("1234567890", "sandi-benar").Principal

	if principal.Can("content.create") {
		t.Fatal("Can(content.create) = true, want false for a blocked user")
	}
	if !principal.CanGlobally("settings.manage") {
		t.Fatal("CanGlobally(settings.manage) = false, want true from the admin role")
	}
}

func TestListSessions_MarksCurrentAndMasksIP(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	env.createUser(userSpec{
		Nisn:        ptr("1234567890"),
		Username:    ptr("siswa.satu"),
		DisplayName: "Siswa Satu",
		Password:    "sandi-benar",
		ActivatedAt: &activated,
	})

	first, err := env.svc.Login(context.Background(), LoginInput{
		Identifier: "1234567890",
		Password:   "sandi-benar",
		IP:         "192.168.1.23",
	})
	if err != nil {
		t.Fatalf("Login first: %v", err)
	}
	second, err := env.svc.Login(context.Background(), LoginInput{
		Identifier: "1234567890",
		Password:   "sandi-benar",
		IP:         "2001:db8:85a3:8d3:1234::1",
	})
	if err != nil {
		t.Fatalf("Login second: %v", err)
	}

	devices, err := env.svc.ListSessions(context.Background(), second.Principal)
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("ListSessions returned %d devices, want 2", len(devices))
	}

	prefixByID := make(map[uuid.UUID]*string, len(devices))
	for _, device := range devices {
		prefixByID[device.ID] = device.IPPrefix
		if device.Current != (device.ID == second.Principal.SessionID) {
			t.Errorf("device %s Current = %v, want %v", device.ID, device.Current, device.ID == second.Principal.SessionID)
		}
	}

	if got := prefixByID[first.Principal.SessionID]; got == nil || *got != "192.168.1.x" {
		t.Errorf("IPv4 prefix = %v, want 192.168.1.x", got)
	}
	if got := prefixByID[second.Principal.SessionID]; got == nil || *got != "2001:db8:85a3:8d3::" {
		t.Errorf("IPv6 prefix = %v, want 2001:db8:85a3:8d3::", got)
	}
}

func TestRevokeSession_OtherUsersSessionReturnsNotFound(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	env.createUser(userSpec{
		Nisn:        ptr("1234567890"),
		Username:    ptr("siswa.satu"),
		DisplayName: "Siswa Satu",
		Password:    "sandi-benar",
		ActivatedAt: &activated,
	})
	env.createUser(userSpec{
		Nisn:        ptr("1234567891"),
		Username:    ptr("siswa.dua"),
		DisplayName: "Siswa Dua",
		Password:    "sandi-benar",
		ActivatedAt: &activated,
	})

	first := env.login("1234567890", "sandi-benar")
	second := env.login("1234567891", "sandi-benar")

	err := env.svc.RevokeSession(context.Background(), first.Principal, second.Principal.SessionID)
	requireAppError(t, err, "NOT_FOUND")

	// The victim session must still be active.
	if _, err := env.svc.Authenticate(context.Background(), second.Token); err != nil {
		t.Fatalf("Authenticate after foreign revoke attempt: %v", err)
	}

	// Revoking one's own session works and invalidates the token.
	if err := env.svc.RevokeSession(context.Background(), first.Principal, first.Principal.SessionID); err != nil {
		t.Fatalf("RevokeSession own: %v", err)
	}
	_, err = env.svc.Authenticate(context.Background(), first.Token)
	requireAppError(t, err, "UNAUTHENTICATED")
}

func TestRevokeOtherSessions_KeepsCurrentSession(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	env.createUser(userSpec{
		Nisn:        ptr("1234567890"),
		Username:    ptr("siswa.satu"),
		DisplayName: "Siswa Satu",
		Password:    "sandi-benar",
		ActivatedAt: &activated,
	})

	current := env.login("1234567890", "sandi-benar")
	other := env.login("1234567890", "sandi-benar")

	if err := env.svc.RevokeOtherSessions(context.Background(), current.Principal); err != nil {
		t.Fatalf("RevokeOtherSessions: %v", err)
	}

	if _, err := env.svc.Authenticate(context.Background(), current.Token); err != nil {
		t.Fatalf("Authenticate current session: %v", err)
	}
	_, err := env.svc.Authenticate(context.Background(), other.Token)
	requireAppError(t, err, "UNAUTHENTICATED")
}

func TestAuthenticate_TouchIsThrottled(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	env.createUser(userSpec{
		Nisn:        ptr("1234567890"),
		Username:    ptr("siswa.satu"),
		DisplayName: "Siswa Satu",
		Password:    "sandi-benar",
		ActivatedAt: &activated,
	})

	login := env.login("1234567890", "sandi-benar")
	sessionID := login.Principal.SessionID

	// Make the session look untouched for longer than TouchInterval.
	if _, err := env.pool.Exec(context.Background(),
		"UPDATE sessions SET last_seen_at = $2 WHERE id = $1",
		sessionID, env.clock.Now().Add(-2*defaultTouchInterval)); err != nil {
		t.Fatalf("age session: %v", err)
	}
	if err := env.rdb.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("flush valkey: %v", err)
	}

	if _, err := env.svc.Authenticate(context.Background(), login.Token); err != nil {
		t.Fatalf("Authenticate first: %v", err)
	}
	afterTouch := env.sessionLastSeen(sessionID)

	// A second call within TouchInterval must not update last_seen_at, even
	// when the cache is bypassed.
	env.clock.Advance(defaultTouchInterval / 5)
	if err := env.rdb.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("flush valkey: %v", err)
	}
	if _, err := env.svc.Authenticate(context.Background(), login.Token); err != nil {
		t.Fatalf("Authenticate second: %v", err)
	}
	afterSecond := env.sessionLastSeen(sessionID)

	if !afterSecond.Equal(afterTouch) {
		t.Fatalf("last_seen_at changed within TouchInterval: %v -> %v", afterTouch, afterSecond)
	}
}

func TestInvalidateUser_DropsCachedPrincipal(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	userID := env.createUser(userSpec{
		Nisn:        ptr("1234567890"),
		Username:    ptr("siswa.satu"),
		DisplayName: "Siswa Satu",
		Password:    "sandi-benar",
		ActivatedAt: &activated,
	})

	login := env.login("1234567890", "sandi-benar")
	if _, err := env.svc.Authenticate(context.Background(), login.Token); err != nil {
		t.Fatalf("Authenticate: %v", err)
	}

	cacheKey := sessionCacheKey(securetoken.HashSessionToken(login.Token))
	exists, err := env.rdb.Exists(context.Background(), cacheKey).Result()
	if err != nil {
		t.Fatalf("check cache: %v", err)
	}
	if exists != 1 {
		t.Fatalf("cache key %s exists = %d, want 1", cacheKey, exists)
	}

	if err := env.svc.InvalidateUser(context.Background(), userID); err != nil {
		t.Fatalf("InvalidateUser: %v", err)
	}

	exists, err = env.rdb.Exists(context.Background(), cacheKey).Result()
	if err != nil {
		t.Fatalf("check cache after invalidate: %v", err)
	}
	if exists != 0 {
		t.Fatalf("cache key %s still exists after InvalidateUser", cacheKey)
	}
	if _, err := env.svc.Authenticate(context.Background(), login.Token); err != nil {
		t.Fatalf("Authenticate after InvalidateUser: %v", err)
	}
}

func TestNewService_RejectsMissingDependencies(t *testing.T) {
	if _, err := NewService(nil, nil, nil, nil, Config{}, nil); err == nil {
		t.Fatal("NewService with nil dependencies returned no error")
	}
}
