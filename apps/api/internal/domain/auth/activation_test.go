package auth

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/securetoken"
)

func requireFieldError(t *testing.T, err error, field, code string) {
	t.Helper()

	appErr := requireAppError(t, err, "VALIDATION_FAILED")
	for _, fieldErr := range appErr.Fields {
		if fieldErr.Field == field && fieldErr.Code == code {
			return
		}
	}
	t.Fatalf("field errors = %+v, want %s=%s", appErr.Fields, field, code)
}

func TestActivate_HappyPathCompletesAfterConfirm(t *testing.T) {
	env := newTestEnv(t)
	expires := env.clock.Now().Add(24 * time.Hour)

	userID := env.createUser(userSpec{
		Nisn:              ptr("1234567890"),
		DisplayName:       "Siswa Baru",
		Password:          "sandi-awal-123",
		PasswordKind:      "initial",
		PasswordExpiresAt: &expires,
	})

	login := env.login("1234567890", "sandi-awal-123")
	if login.Principal.NextStep != NextStepActivate {
		t.Fatalf("NextStep = %q, want %q", login.Principal.NextStep, NextStepActivate)
	}

	codes, err := env.svc.Activate(context.Background(), login.Principal, "  Siswa.Baru ", ptr("Kopi-Susu-2026!"), "192.0.2.10")
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if len(codes) != recoveryCodeCount {
		t.Fatalf("recovery codes = %d, want %d", len(codes), recoveryCodeCount)
	}

	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		normalized, ok := securetoken.NormalizeRecoveryCode(code)
		if !ok {
			t.Fatalf("recovery code %q is not in display form", code)
		}
		if len(code) != 11 || code[5] != '-' {
			t.Fatalf("recovery code %q is not shaped xxxxx-xxxxx", code)
		}
		if _, duplicate := seen[normalized]; duplicate {
			t.Fatalf("duplicate recovery code %q", code)
		}
		seen[normalized] = struct{}{}
	}

	// Activation is not complete until the codes are confirmed.
	afterActivate, err := env.svc.Authenticate(context.Background(), login.Token)
	if err != nil {
		t.Fatalf("Authenticate after Activate: %v", err)
	}
	if afterActivate.NextStep != NextStepActivate {
		t.Fatalf("NextStep after Activate = %q, want %q", afterActivate.NextStep, NextStepActivate)
	}
	if afterActivate.Username == nil || *afterActivate.Username != "siswa.baru" {
		t.Fatalf("Username = %v, want siswa.baru", afterActivate.Username)
	}

	// The freshly chosen username counts as available to its owner.
	own, err := env.svc.CheckUsername(context.Background(), afterActivate, "SISWA.BARU")
	if err != nil {
		t.Fatalf("CheckUsername own: %v", err)
	}
	if !own.Available || own.Username != "siswa.baru" {
		t.Fatalf("CheckUsername own = %+v, want available siswa.baru", own)
	}

	confirmed, err := env.svc.ConfirmRecoveryCodes(context.Background(), afterActivate)
	if err != nil {
		t.Fatalf("ConfirmRecoveryCodes: %v", err)
	}
	if confirmed.NextStep != NextStepNone {
		t.Fatalf("NextStep after confirm = %q, want %q", confirmed.NextStep, NextStepNone)
	}
	if confirmed.SessionID != login.Principal.SessionID {
		t.Fatalf("SessionID = %s, want %s", confirmed.SessionID, login.Principal.SessionID)
	}

	var (
		username    string
		kind        string
		passwordExp *time.Time
		activatedAt *time.Time
		confirmedAt *time.Time
	)
	if err := env.pool.QueryRow(context.Background(), `
		SELECT username, password_kind, password_expires_at, activated_at, recovery_codes_confirmed_at
		FROM users WHERE id = $1`, userID).
		Scan(&username, &kind, &passwordExp, &activatedAt, &confirmedAt); err != nil {
		t.Fatalf("get activated user: %v", err)
	}
	if username != "siswa.baru" || kind != "user" || passwordExp != nil || activatedAt == nil || confirmedAt == nil {
		t.Fatalf("activated user = username %q kind %q expires %v activated %v confirmed %v",
			username, kind, passwordExp, activatedAt, confirmedAt)
	}

	var activeCodes, batches int
	if err := env.pool.QueryRow(context.Background(), `
		SELECT count(*), count(DISTINCT batch_id)
		FROM recovery_codes
		WHERE user_id = $1 AND used_at IS NULL AND revoked_at IS NULL`, userID).
		Scan(&activeCodes, &batches); err != nil {
		t.Fatalf("count recovery codes: %v", err)
	}
	if activeCodes != recoveryCodeCount || batches != 1 {
		t.Fatalf("active recovery codes = %d in %d batches, want %d in 1", activeCodes, batches, recoveryCodeCount)
	}

	rows, err := env.pool.Query(context.Background(),
		"SELECT action FROM audit_logs WHERE actor_id = $1 ORDER BY id", userID)
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	defer rows.Close()
	var actions []string
	for rows.Next() {
		var action string
		if err := rows.Scan(&action); err != nil {
			t.Fatalf("scan audit action: %v", err)
		}
		actions = append(actions, action)
	}
	for _, want := range []string{"auth.activate", "auth.activation_complete"} {
		if !slices.Contains(actions, want) {
			t.Errorf("audit actions = %v, want %q", actions, want)
		}
	}

	// The new password works and the old one is gone.
	result, err := env.svc.Login(context.Background(), LoginInput{
		Identifier: "siswa.baru",
		Password:   "Kopi-Susu-2026!",
	})
	if err != nil {
		t.Fatalf("Login with new password: %v", err)
	}
	if result.Principal.NextStep != NextStepNone {
		t.Fatalf("NextStep after login = %q, want none", result.Principal.NextStep)
	}

	_, err = env.svc.Login(context.Background(), LoginInput{
		Identifier: "1234567890",
		Password:   "sandi-awal-123",
	})
	requireAppError(t, err, "INVALID_CREDENTIALS")

	// Activation cannot run again once confirmed.
	_, err = env.svc.Activate(context.Background(), confirmed, "siswa.lain", ptr("Kopi-Susu-2026!"), "")
	requireAppError(t, err, "INVALID_TRANSITION")
}

func TestActivate_RejectsInvalidReservedAndTakenUsernames(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()
	expires := env.clock.Now().Add(24 * time.Hour)

	env.createUser(userSpec{
		Nisn:        ptr("1234567891"),
		Username:    ptr("dipakai"),
		DisplayName: "Siswa Aktif",
		Password:    "sandi-awal-123",
		ActivatedAt: &activated,
	})
	env.createUser(userSpec{
		Nisn:              ptr("1234567890"),
		DisplayName:       "Siswa Baru",
		Password:          "sandi-awal-123",
		PasswordKind:      "initial",
		PasswordExpiresAt: &expires,
	})

	login := env.login("1234567890", "sandi-awal-123")

	tests := []struct {
		name     string
		username string
		code     string
	}{
		{name: "too short", username: "ab", code: "invalid_format"},
		{name: "double dot", username: "siswa..baru", code: "invalid_format"},
		{name: "trailing dot", username: "siswa.baru.", code: "invalid_format"},
		{name: "reserved", username: "admin", code: "reserved"},
		{name: "taken", username: "dipakai", code: "taken"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := env.svc.Activate(context.Background(), login.Principal, tt.username, ptr("Kopi-Susu-2026!"), "")
			requireFieldError(t, err, "username", tt.code)

			check, err := env.svc.CheckUsername(context.Background(), login.Principal, tt.username)
			if err != nil {
				t.Fatalf("CheckUsername: %v", err)
			}
			if check.Available || check.Reason != tt.code {
				t.Fatalf("CheckUsername = %+v, want unavailable %s", check, tt.code)
			}
		})
	}
}

func TestActivate_RejectsWeakPassword(t *testing.T) {
	env := newTestEnv(t)
	expires := env.clock.Now().Add(24 * time.Hour)

	env.createUser(userSpec{
		Nisn:              ptr("1234567890"),
		DisplayName:       "Siswa Baru",
		Password:          "sandi-awal-123",
		PasswordKind:      "initial",
		PasswordExpiresAt: &expires,
	})
	login := env.login("1234567890", "sandi-awal-123")

	_, err := env.svc.Activate(context.Background(), login.Principal, "siswa.baru", ptr("abc123"), "")
	requireFieldError(t, err, "newPassword", "too_short")

	_, err = env.svc.Activate(context.Background(), login.Principal, "siswa.baru", ptr("siswa.baru-2026"), "")
	requireFieldError(t, err, "newPassword", "contains_identity")

	_, err = env.svc.Activate(context.Background(), login.Principal, "siswa.baru", nil, "")
	requireFieldError(t, err, "newPassword", "required")
}

func TestActivate_MigratedUserKeepsPasswordAndRedirectsOldUsername(t *testing.T) {
	env := newTestEnv(t)
	legacyID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("uuid.NewV7: %v", err)
	}

	userID := env.createUser(userSpec{
		Nisn:         ptr("1234567890"),
		Username:     ptr("lama.siswa"),
		DisplayName:  "Siswa Lama",
		Password:     "sandi-lama-123",
		PasswordKind: "user",
		LegacyV1ID:   &legacyID,
	})

	login := env.login("1234567890", "sandi-lama-123")
	if login.Principal.NextStep != NextStepActivate || !login.Principal.IsMigrated {
		t.Fatalf("principal = %+v, want migrated account needing activation", login.Principal)
	}

	codes, err := env.svc.Activate(context.Background(), login.Principal, "baru.siswa", nil, "")
	if err != nil {
		t.Fatalf("Activate without new password: %v", err)
	}
	if len(codes) != recoveryCodeCount {
		t.Fatalf("recovery codes = %d, want %d", len(codes), recoveryCodeCount)
	}

	if _, err := env.svc.ConfirmRecoveryCodes(context.Background(), login.Principal); err != nil {
		t.Fatalf("ConfirmRecoveryCodes: %v", err)
	}

	// The migrated password is kept.
	if _, err := env.svc.Login(context.Background(), LoginInput{
		Identifier: "baru.siswa",
		Password:   "sandi-lama-123",
	}); err != nil {
		t.Fatalf("Login with kept password: %v", err)
	}

	var (
		oldUsername   string
		redirectUntil time.Time
		releasedAt    *time.Time
	)
	if err := env.pool.QueryRow(context.Background(), `
		SELECT old_username, redirect_until, released_at
		FROM username_history WHERE user_id = $1`, userID).
		Scan(&oldUsername, &redirectUntil, &releasedAt); err != nil {
		t.Fatalf("get username history: %v", err)
	}
	if oldUsername != "lama.siswa" {
		t.Errorf("old_username = %q, want lama.siswa", oldUsername)
	}
	if releasedAt != nil {
		t.Error("released_at is set, want NULL")
	}
	wantRedirect := env.clock.Now().Add(usernameRedirect)
	if !redirectUntil.Equal(wantRedirect) {
		t.Errorf("redirect_until = %v, want %v", redirectUntil, wantRedirect)
	}

	// The old username now belongs to its owner during the redirect window, so
	// another user cannot take it.
	otherID := env.createUser(userSpec{
		Nisn:        ptr("1234567891"),
		Username:    ptr("siswa.lain"),
		DisplayName: "Siswa Lain",
		Password:    "sandi-lain-123",
	})
	check, err := env.svc.CheckUsername(context.Background(), &Principal{
		UserID: otherID,
	}, "lama.siswa")
	if err != nil {
		t.Fatalf("CheckUsername other user: %v", err)
	}
	if check.Available || check.Reason != usernameTaken {
		t.Fatalf("CheckUsername old username = %+v, want taken", check)
	}
}

func TestActivate_SecondCallReplacesCodesAndConfirmNeedsCodes(t *testing.T) {
	env := newTestEnv(t)
	expires := env.clock.Now().Add(24 * time.Hour)

	userID := env.createUser(userSpec{
		Nisn:              ptr("1234567890"),
		DisplayName:       "Siswa Baru",
		Password:          "sandi-awal-123",
		PasswordKind:      "initial",
		PasswordExpiresAt: &expires,
	})
	login := env.login("1234567890", "sandi-awal-123")

	_, err := env.svc.ConfirmRecoveryCodes(context.Background(), login.Principal)
	requireAppError(t, err, "INVALID_TRANSITION")

	first, err := env.svc.Activate(context.Background(), login.Principal, "siswa.baru", ptr("Kopi-Susu-2026!"), "")
	if err != nil {
		t.Fatalf("Activate first: %v", err)
	}
	second, err := env.svc.Activate(context.Background(), login.Principal, "siswa.baru", ptr("Kopi-Susu-2026!"), "")
	if err != nil {
		t.Fatalf("Activate second: %v", err)
	}
	if slices.Equal(first, second) {
		t.Fatal("the second Activate returned the same recovery codes")
	}

	var active int64
	if err := env.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM recovery_codes
		WHERE user_id = $1 AND used_at IS NULL AND revoked_at IS NULL`, userID).
		Scan(&active); err != nil {
		t.Fatalf("count recovery codes: %v", err)
	}
	if active != recoveryCodeCount {
		t.Fatalf("active recovery codes = %d, want %d", active, recoveryCodeCount)
	}

	if _, err := env.svc.ConfirmRecoveryCodes(context.Background(), login.Principal); err != nil {
		t.Fatalf("ConfirmRecoveryCodes: %v", err)
	}
}

func TestNewService_RejectsShortRecoverySecret(t *testing.T) {
	env := newTestEnv(t)

	_, err := NewService(env.svc.store, env.rdb, env.hasher, env.svc.limiter, Config{
		RecoverySecret: []byte("too-short"),
	}, nil)
	if err == nil {
		t.Fatal("NewService with a short recovery secret returned no error")
	}
}

func TestConfirmRecoveryCodes_DeletesInitialPasswordVaultEntry(t *testing.T) {
	env := newTestEnv(t)
	expires := env.clock.Now().Add(24 * time.Hour)

	userID := env.createUser(userSpec{
		Nisn:              ptr("1234567891"),
		DisplayName:       "Siswa Brankas",
		Password:          "sandi-awal-123",
		PasswordKind:      "initial",
		PasswordExpiresAt: &expires,
	})

	if _, err := env.pool.Exec(context.Background(), `
		INSERT INTO initial_password_vault (user_id, ciphertext, expires_at)
		VALUES ($1, $2, $3)`, userID, []byte("nonce-and-sealed"), expires); err != nil {
		t.Fatalf("insert vault entry: %v", err)
	}

	login := env.login("1234567891", "sandi-awal-123")
	if _, err := env.svc.Activate(context.Background(), login.Principal, "siswa.brankas", ptr("Kopi-Susu-2026!"), "192.0.2.10"); err != nil {
		t.Fatalf("Activate: %v", err)
	}

	pending, err := env.svc.Authenticate(context.Background(), login.Token)
	if err != nil {
		t.Fatalf("Authenticate after Activate: %v", err)
	}
	if _, err := env.svc.ConfirmRecoveryCodes(context.Background(), pending); err != nil {
		t.Fatalf("ConfirmRecoveryCodes: %v", err)
	}

	var count int
	if err := env.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM initial_password_vault WHERE user_id = $1", userID,
	).Scan(&count); err != nil {
		t.Fatalf("count vault entries: %v", err)
	}
	if count != 0 {
		t.Fatalf("vault entries after activation = %d, want 0", count)
	}
}
