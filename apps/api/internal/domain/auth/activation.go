package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/audit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/password"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/securetoken"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	recoveryCodeCount = 8
	usernameRedirect  = 90 * 24 * time.Hour

	passwordConfirmLimit  = 10
	passwordConfirmWindow = time.Hour
)

// Activate sets the username and (unless kept from a migrated account) the
// password of a not-yet-activated user, then issues a new batch of recovery
// codes. Activation is only complete after ConfirmRecoveryCodes.
func (s *Service) Activate(ctx context.Context, p *Principal, rawUsername string, newPassword *string, ip string) ([]string, error) {
	if p == nil {
		return nil, apperr.Unauthenticated()
	}
	if p.NextStep != NextStepActivate {
		return nil, apperr.InvalidTransition("Akun sudah aktif.")
	}

	check, err := s.CheckUsername(ctx, p, rawUsername)
	if err != nil {
		return nil, err
	}
	if check.Reason != "" {
		return nil, usernameError(check.Reason)
	}

	identity, err := s.store.GetUserIdentity(ctx, p.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.Unauthenticated()
	}
	if err != nil {
		return nil, fmt.Errorf("auth: get user identity: %w", err)
	}

	// Migrated v1 accounts may keep the password they already have.
	keepPassword := identity.LegacyV1ID != nil && identity.PasswordKind == "user"
	hasNewPassword := newPassword != nil && *newPassword != ""
	if !hasNewPassword && !keepPassword {
		return nil, apperr.Validation(apperr.Field("newPassword", "required", "Sandi baru wajib diisi."))
	}

	var newHash string
	if hasNewPassword {
		if err := s.checkPasswordPolicy(*newPassword, check.Username, identity.Nisn, identity.StaffNumber, identity.DisplayName); err != nil {
			return nil, err
		}
		newHash, err = s.hasher.Hash(ctx, *newPassword)
		if err != nil {
			return nil, fmt.Errorf("auth: hash password: %w", err)
		}
	}

	codes, batchID, err := newRecoveryCodeBatch()
	if err != nil {
		return nil, err
	}

	usernameChanged := identity.Username == nil || NormalizeUsername(*identity.Username) != check.Username
	oldUsername := identity.Username

	err = s.store.WithTx(ctx, func(q *store.Queries) error {
		if oldUsername != nil && usernameChanged {
			if err := q.InsertUsernameHistory(ctx, store.InsertUsernameHistoryParams{
				UserID:        p.UserID,
				OldUsername:   *oldUsername,
				RedirectUntil: s.now().UTC().Add(usernameRedirect),
			}); err != nil {
				return fmt.Errorf("auth: insert username history: %w", err)
			}
		}

		if err := q.SetUsername(ctx, store.SetUsernameParams{
			Username: check.Username,
			ID:       p.UserID,
		}); err != nil {
			return fmt.Errorf("auth: set username: %w", err)
		}

		if hasNewPassword {
			if err := q.SetOwnPassword(ctx, store.SetOwnPasswordParams{
				PasswordHash: newHash,
				ID:           p.UserID,
			}); err != nil {
				return fmt.Errorf("auth: set password: %w", err)
			}
		}

		if err := s.storeRecoveryCodes(ctx, q, p.UserID, codes, batchID); err != nil {
			return err
		}

		return audit.Record(ctx, q, audit.Entry{
			ActorID:    &p.UserID,
			ActorIP:    ip,
			Action:     "auth.activate",
			TargetType: "user",
			TargetID:   &p.UserID,
			Data: map[string]any{
				"username_changed": usernameChanged,
				"password_changed": hasNewPassword,
			},
		})
	})
	if err != nil {
		return nil, err
	}

	if err := s.InvalidateUser(ctx, p.UserID); err != nil {
		s.logger.Warn("auth: invalidate user after activation", "user_id", p.UserID, "error", err)
	}

	return codes, nil
}

// ConfirmRecoveryCodes marks activation complete when the user has stored
// their recovery codes, and returns the refreshed principal of the same
// session.
func (s *Service) ConfirmRecoveryCodes(ctx context.Context, p *Principal) (*Principal, error) {
	if p == nil {
		return nil, apperr.Unauthenticated()
	}
	if p.NextStep != NextStepActivate {
		return nil, apperr.InvalidTransition("Akun sudah aktif.")
	}

	count, err := s.store.CountActiveRecoveryCodes(ctx, p.UserID)
	if err != nil {
		return nil, fmt.Errorf("auth: count recovery codes: %w", err)
	}
	if count == 0 {
		return nil, apperr.InvalidTransition("Buat recovery code terlebih dahulu.")
	}

	err = s.store.WithTx(ctx, func(q *store.Queries) error {
		if err := q.CompleteActivation(ctx, p.UserID); err != nil {
			return fmt.Errorf("auth: complete activation: %w", err)
		}
		return audit.Record(ctx, q, audit.Entry{
			ActorID:    &p.UserID,
			Action:     "auth.activation_complete",
			TargetType: "user",
			TargetID:   &p.UserID,
		})
	})
	if err != nil {
		return nil, err
	}

	if err := s.InvalidateUser(ctx, p.UserID); err != nil {
		s.logger.Warn("auth: invalidate user after activation", "user_id", p.UserID, "error", err)
	}

	principal, err := s.loadPrincipal(ctx, store.Session{
		ID:        p.SessionID,
		UserID:    p.UserID,
		ExpiresAt: p.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}
	return principal, nil
}

// RegenerateRecoveryCodes replaces every recovery code after confirming the
// user's current password.
func (s *Service) RegenerateRecoveryCodes(ctx context.Context, p *Principal, currentPassword, ip string) ([]string, error) {
	if p == nil {
		return nil, apperr.Unauthenticated()
	}
	if p.NextStep != NextStepNone {
		return nil, apperr.InvalidTransition("Selesaikan proses akun terlebih dahulu.")
	}

	if err := s.allowPasswordConfirm(ctx, p.UserID); err != nil {
		return nil, err
	}

	identity, err := s.store.GetUserIdentity(ctx, p.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.Unauthenticated()
	}
	if err != nil {
		return nil, fmt.Errorf("auth: get user identity: %w", err)
	}

	ok, err := s.verifyPassword(ctx, identity.PasswordHash, currentPassword, p.UserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, apperr.InvalidCredentials()
	}

	codes, batchID, err := newRecoveryCodeBatch()
	if err != nil {
		return nil, err
	}

	err = s.store.WithTx(ctx, func(q *store.Queries) error {
		if err := s.storeRecoveryCodes(ctx, q, p.UserID, codes, batchID); err != nil {
			return err
		}
		return audit.Record(ctx, q, audit.Entry{
			ActorID:    &p.UserID,
			ActorIP:    ip,
			Action:     "auth.recovery_codes_regenerate",
			TargetType: "user",
			TargetID:   &p.UserID,
		})
	})
	if err != nil {
		return nil, err
	}

	return codes, nil
}

// allowPasswordConfirm applies the shared "confirm with the current password"
// rate limit. It fails closed when the limiter is unavailable.
func (s *Service) allowPasswordConfirm(ctx context.Context, userID uuid.UUID) error {
	result, err := s.limiter.Allow(ctx, "pwconfirm:"+userID.String(), ratelimit.Rule{
		Limit:  passwordConfirmLimit,
		Window: passwordConfirmWindow,
	})
	if err != nil {
		s.logger.Warn("auth: password confirmation rate limiter failed", "error", err)
		return apperr.Unavailable("rate limiter")
	}
	if !result.Allowed {
		return apperr.RateLimited()
	}
	return nil
}

// checkPasswordPolicy validates a new password against the account identities
// and returns one field error per violation.
func (s *Service) checkPasswordPolicy(plain, username string, nisn, staffNumber *string, displayName string) error {
	identities := []string{username}
	if nisn != nil {
		identities = append(identities, *nisn)
	}
	if staffNumber != nil {
		identities = append(identities, *staffNumber)
	}
	identities = append(identities, strings.Fields(displayName)...)

	violations := password.CheckPolicy(plain, identities...)
	if len(violations) == 0 {
		return nil
	}

	fields := make([]apperr.FieldError, 0, len(violations))
	for _, violation := range violations {
		fields = append(fields, apperr.Field("newPassword", string(violation), password.ViolationMessage(violation)))
	}
	return apperr.Validation(fields...)
}

// verifyPassword checks plain against an encoded hash. A hash that cannot be
// processed counts as a wrong password and is logged.
func (s *Service) verifyPassword(ctx context.Context, encoded, plain string, userID uuid.UUID) (bool, error) {
	ok, _, err := s.hasher.Verify(ctx, encoded, plain)
	if err != nil {
		if errors.Is(err, password.ErrUnknownFormat) || errors.Is(err, password.ErrMalformedHash) {
			s.logger.Warn("auth: stored password hash is not usable", "user_id", userID)
			return false, nil
		}
		return false, fmt.Errorf("auth: verify password: %w", err)
	}
	return ok, nil
}

// newRecoveryCodeBatch generates a fresh set of display-formatted codes and
// their shared batch id.
func newRecoveryCodeBatch() ([]string, uuid.UUID, error) {
	codes, err := securetoken.RecoveryCodes(recoveryCodeCount)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("auth: generate recovery codes: %w", err)
	}
	batchID, err := uuid.NewV7()
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("auth: generate recovery code batch: %w", err)
	}
	return codes, batchID, nil
}

// storeRecoveryCodes revokes the user's active codes and inserts the new batch.
func (s *Service) storeRecoveryCodes(ctx context.Context, q *store.Queries, userID uuid.UUID, codes []string, batchID uuid.UUID) error {
	if err := q.RevokeActiveRecoveryCodes(ctx, userID); err != nil {
		return fmt.Errorf("auth: revoke recovery codes: %w", err)
	}

	for _, code := range codes {
		normalized, ok := securetoken.NormalizeRecoveryCode(code)
		if !ok {
			return fmt.Errorf("auth: generated recovery code is not valid")
		}
		if err := q.InsertRecoveryCode(ctx, store.InsertRecoveryCodeParams{
			UserID:   userID,
			BatchID:  batchID,
			CodeHash: securetoken.HashRecoveryCode(s.cfg.RecoverySecret, normalized),
		}); err != nil {
			return fmt.Errorf("auth: insert recovery code: %w", err)
		}
	}
	return nil
}
