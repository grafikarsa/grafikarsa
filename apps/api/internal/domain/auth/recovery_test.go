package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/securetoken"
)

// insertRecoveryCodes seeds a recovery-code batch directly, which keeps these
// tests focused on recovery and password changes.
func (e *testEnv) insertRecoveryCodes(userID uuid.UUID, count int) []string {
	e.t.Helper()

	codes, err := securetoken.RecoveryCodes(count)
	if err != nil {
		e.t.Fatalf("generate recovery codes: %v", err)
	}
	batchID, err := uuid.NewV7()
	if err != nil {
		e.t.Fatalf("generate batch id: %v", err)
	}

	for _, code := range codes {
		normalized, ok := securetoken.NormalizeRecoveryCode(code)
		if !ok {
			e.t.Fatalf("generated code %q is not valid", code)
		}
		if _, err := e.pool.Exec(context.Background(), `
			INSERT INTO recovery_codes (user_id, batch_id, code_hash)
			VALUES ($1, $2, $3)`,
			userID, batchID, securetoken.HashRecoveryCode(testRecoverySecret, normalized)); err != nil {
			e.t.Fatalf("insert recovery code: %v", err)
		}
	}

	return codes
}

func TestRecoverPassword_ValidCodeChangesPasswordAndRevokesSessions(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	userID := env.createUser(userSpec{
		Nisn:        ptr("1234567890"),
		Username:    ptr("siswa.satu"),
		DisplayName: "Siswa Satu",
		Password:    "sandi-awal-123",
		ActivatedAt: &activated,
	})

	first := env.login("1234567890", "sandi-awal-123")
	second := env.login("1234567890", "sandi-awal-123")
	codes := env.insertRecoveryCodes(userID, recoveryCodeCount)

	// A weak password is rejected before the one-time code is consumed.
	err := env.svc.RecoverPassword(context.Background(), "1234567890", codes[0], "abc123", "192.0.2.10")
	requireFieldError(t, err, "newPassword", "too_short")

	if err := env.svc.RecoverPassword(context.Background(), "1234567890", codes[0], "Melon-Manis-77!", "192.0.2.10"); err != nil {
		t.Fatalf("RecoverPassword: %v", err)
	}

	result, err := env.svc.Login(context.Background(), LoginInput{
		Identifier: "siswa.satu",
		Password:   "Melon-Manis-77!",
	})
	if err != nil {
		t.Fatalf("Login with recovered password: %v", err)
	}
	if result.Principal.NextStep != NextStepNone {
		t.Fatalf("NextStep = %q, want none", result.Principal.NextStep)
	}

	for _, token := range []string{first.Token, second.Token} {
		_, err := env.svc.Authenticate(context.Background(), token)
		requireAppError(t, err, "UNAUTHENTICATED")
	}

	// The same code cannot be used twice.
	err = env.svc.RecoverPassword(context.Background(), "1234567890", codes[0], "Mangga-Manis-88!", "192.0.2.10")
	requireAppError(t, err, "INVALID_CREDENTIALS")

	var remaining int64
	if err := env.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM recovery_codes
		WHERE user_id = $1 AND used_at IS NULL AND revoked_at IS NULL`, userID).
		Scan(&remaining); err != nil {
		t.Fatalf("count remaining codes: %v", err)
	}
	if remaining != recoveryCodeCount-1 {
		t.Fatalf("remaining codes = %d, want %d", remaining, recoveryCodeCount-1)
	}
}

func TestRecoverPassword_WrongCodeAndUnknownIdentifierReturnInvalidCredentials(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	userID := env.createUser(userSpec{
		Nisn:        ptr("1234567890"),
		Username:    ptr("siswa.satu"),
		DisplayName: "Siswa Satu",
		Password:    "sandi-awal-123",
		ActivatedAt: &activated,
	})
	env.insertRecoveryCodes(userID, recoveryCodeCount)

	err := env.svc.RecoverPassword(context.Background(), "1234567890", "aaaaa-bbbbb", "Melon-Manis-77!", "192.0.2.11")
	requireAppError(t, err, "INVALID_CREDENTIALS")

	err = env.svc.RecoverPassword(context.Background(), "tidak-ada", "aaaaa-bbbbb", "Melon-Manis-77!", "192.0.2.11")
	requireAppError(t, err, "INVALID_CREDENTIALS")

	err = env.svc.RecoverPassword(context.Background(), "1234567890", "bukan-kode", "Melon-Manis-77!", "192.0.2.11")
	requireAppError(t, err, "INVALID_CREDENTIALS")

	// None of the failures consumed a code.
	var remaining int64
	if err := env.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM recovery_codes
		WHERE user_id = $1 AND used_at IS NULL AND revoked_at IS NULL`, userID).
		Scan(&remaining); err != nil {
		t.Fatalf("count remaining codes: %v", err)
	}
	if remaining != recoveryCodeCount {
		t.Fatalf("remaining codes = %d, want %d", remaining, recoveryCodeCount)
	}
}

func TestRecoverPassword_RateLimitsAfterFiveAttempts(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	userID := env.createUser(userSpec{
		Nisn:        ptr("1234567890"),
		Username:    ptr("siswa.satu"),
		DisplayName: "Siswa Satu",
		Password:    "sandi-awal-123",
		ActivatedAt: &activated,
	})
	env.insertRecoveryCodes(userID, recoveryCodeCount)

	for attempt := 1; attempt <= 5; attempt++ {
		err := env.svc.RecoverPassword(context.Background(), "1234567890", "aaaaa-bbbbb", "Melon-Manis-77!", "192.0.2.12")
		requireAppError(t, err, "INVALID_CREDENTIALS")
	}

	err := env.svc.RecoverPassword(context.Background(), "1234567890", "aaaaa-bbbbb", "Melon-Manis-77!", "192.0.2.12")
	requireAppError(t, err, "RATE_LIMITED")
}

func TestChangePassword_TemporaryPasswordNeedsNoCurrentPassword(t *testing.T) {
	env := newTestEnv(t)
	now := env.clock.Now()
	expires := now.Add(24 * time.Hour)

	env.createUser(userSpec{
		Nisn:              ptr("1234567890"),
		Username:          ptr("siswa.sementara"),
		DisplayName:       "Siswa Sementara",
		Password:          "sandi-temp-123",
		PasswordKind:      "temporary",
		PasswordExpiresAt: &expires,
		ActivatedAt:       &now,
	})

	current := env.login("1234567890", "sandi-temp-123")
	other := env.login("1234567890", "sandi-temp-123")
	if current.Principal.NextStep != NextStepChangePassword {
		t.Fatalf("NextStep = %q, want %q", current.Principal.NextStep, NextStepChangePassword)
	}

	updated, err := env.svc.ChangePassword(context.Background(), current.Principal, nil, "Melon-Manis-77!", "192.0.2.13")
	if err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
	if updated.NextStep != NextStepNone {
		t.Fatalf("NextStep after change = %q, want none", updated.NextStep)
	}

	_, err = env.svc.Authenticate(context.Background(), other.Token)
	requireAppError(t, err, "UNAUTHENTICATED")
	if _, err := env.svc.Authenticate(context.Background(), current.Token); err != nil {
		t.Fatalf("Authenticate current session: %v", err)
	}

	if _, err := env.svc.Login(context.Background(), LoginInput{
		Identifier: "siswa.sementara",
		Password:   "Melon-Manis-77!",
	}); err != nil {
		t.Fatalf("Login with new password: %v", err)
	}
}

func TestChangePassword_WrongCurrentPasswordReturnsFieldErrors(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	env.createUser(userSpec{
		Nisn:        ptr("1234567890"),
		Username:    ptr("siswa.satu"),
		DisplayName: "Siswa Satu",
		Password:    "sandi-awal-123",
		ActivatedAt: &activated,
	})

	login := env.login("1234567890", "sandi-awal-123")

	_, err := env.svc.ChangePassword(context.Background(), login.Principal, nil, "Melon-Manis-77!", "")
	requireFieldError(t, err, "currentPassword", "required")

	_, err = env.svc.ChangePassword(context.Background(), login.Principal, ptr("sandi-salah"), "Melon-Manis-77!", "")
	requireFieldError(t, err, "currentPassword", "wrong")

	updated, err := env.svc.ChangePassword(context.Background(), login.Principal, ptr("sandi-awal-123"), "Melon-Manis-77!", "")
	if err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
	if updated.NextStep != NextStepNone {
		t.Fatalf("NextStep = %q, want none", updated.NextStep)
	}

	if _, err := env.svc.Login(context.Background(), LoginInput{
		Identifier: "siswa.satu",
		Password:   "Melon-Manis-77!",
	}); err != nil {
		t.Fatalf("Login with new password: %v", err)
	}

	_, err = env.svc.Login(context.Background(), LoginInput{
		Identifier: "siswa.satu",
		Password:   "sandi-awal-123",
	})
	requireAppError(t, err, "INVALID_CREDENTIALS")
}

func TestRegenerateRecoveryCodes_InvalidatesOldCodes(t *testing.T) {
	env := newTestEnv(t)
	activated := env.clock.Now()

	userID := env.createUser(userSpec{
		Nisn:        ptr("1234567890"),
		Username:    ptr("siswa.satu"),
		DisplayName: "Siswa Satu",
		Password:    "sandi-awal-123",
		ActivatedAt: &activated,
	})
	oldCodes := env.insertRecoveryCodes(userID, recoveryCodeCount)

	login := env.login("1234567890", "sandi-awal-123")

	_, err := env.svc.RegenerateRecoveryCodes(context.Background(), login.Principal, "sandi-salah", "192.0.2.14")
	requireAppError(t, err, "INVALID_CREDENTIALS")

	newCodes, err := env.svc.RegenerateRecoveryCodes(context.Background(), login.Principal, "sandi-awal-123", "192.0.2.14")
	if err != nil {
		t.Fatalf("RegenerateRecoveryCodes: %v", err)
	}
	if len(newCodes) != recoveryCodeCount {
		t.Fatalf("new codes = %d, want %d", len(newCodes), recoveryCodeCount)
	}
	for _, oldCode := range oldCodes {
		for _, newCode := range newCodes {
			if oldCode == newCode {
				t.Fatalf("regenerate reused code %q", oldCode)
			}
		}
	}

	var active int64
	if err := env.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM recovery_codes
		WHERE user_id = $1 AND used_at IS NULL AND revoked_at IS NULL`, userID).
		Scan(&active); err != nil {
		t.Fatalf("count active codes: %v", err)
	}
	if active != recoveryCodeCount {
		t.Fatalf("active codes = %d, want %d", active, recoveryCodeCount)
	}

	// An old code is dead, a new one still works.
	err = env.svc.RecoverPassword(context.Background(), "1234567890", oldCodes[0], "Mangga-Manis-88!", "192.0.2.14")
	requireAppError(t, err, "INVALID_CREDENTIALS")

	if err := env.svc.RecoverPassword(context.Background(), "1234567890", newCodes[0], "Mangga-Manis-88!", "192.0.2.14"); err != nil {
		t.Fatalf("RecoverPassword with a new code: %v", err)
	}
}
