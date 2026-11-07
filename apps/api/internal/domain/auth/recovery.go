package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/audit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/securetoken"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	recoverIdentifierLimit = 5
	recoverIPLimit         = 20
	recoverWindow          = time.Hour
)

// RecoverPassword sets a new password with a single recovery code and revokes
// every session of the account. It runs without a session.
func (s *Service) RecoverPassword(ctx context.Context, identifier, recoveryCode, newPassword, ip string) error {
	identifier = strings.TrimSpace(identifier)

	if err := s.allowRecover(ctx, identifier, ip); err != nil {
		return err
	}

	user, err := s.store.GetUserForLogin(ctx, identifier)
	if errors.Is(err, pgx.ErrNoRows) {
		s.verifyDummy(ctx, recoveryCode)
		return apperr.InvalidCredentials()
	}
	if err != nil {
		return fmt.Errorf("auth: get user for recovery: %w", err)
	}

	if user.ActivatedAt == nil {
		return apperr.InvalidCredentials()
	}

	if user.Status == "transferred" || user.Status == "inactive" || user.DeactivatedAt != nil {
		return &apperr.Error{
			Status: http.StatusForbidden,
			Code:   "ACCOUNT_INACTIVE",
			Title:  "Akun tidak aktif. Hubungi admin sekolah.",
		}
	}

	normalized, ok := securetoken.NormalizeRecoveryCode(recoveryCode)
	if !ok {
		return apperr.InvalidCredentials()
	}

	username := ""
	if user.Username != nil {
		username = *user.Username
	}
	// The password policy is checked before the one-time code is consumed, so a
	// weak password does not burn a recovery code.
	if err := s.checkPasswordPolicy(newPassword, username, user.Nisn, user.StaffNumber, user.DisplayName); err != nil {
		return err
	}

	newHash, err := s.hasher.Hash(ctx, newPassword)
	if err != nil {
		return fmt.Errorf("auth: hash password: %w", err)
	}

	codeHash := securetoken.HashRecoveryCode(s.cfg.RecoverySecret, normalized)

	var tokenHashes [][]byte
	err = s.store.WithTx(ctx, func(q *store.Queries) error {
		if _, err := q.UseRecoveryCode(ctx, store.UseRecoveryCodeParams{
			UserID:   user.ID,
			CodeHash: codeHash,
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.InvalidCredentials()
			}
			return fmt.Errorf("auth: use recovery code: %w", err)
		}

		if err := q.SetOwnPassword(ctx, store.SetOwnPasswordParams{
			PasswordHash: newHash,
			ID:           user.ID,
		}); err != nil {
			return fmt.Errorf("auth: set password: %w", err)
		}

		remaining, err := q.CountActiveRecoveryCodes(ctx, user.ID)
		if err != nil {
			return fmt.Errorf("auth: count recovery codes: %w", err)
		}

		tokenHashes, err = q.RevokeUserSessions(ctx, store.RevokeUserSessionsParams{
			UserID:   user.ID,
			ExceptID: uuid.Nil,
			Reason:   "password_reset",
		})
		if err != nil {
			return fmt.Errorf("auth: revoke sessions: %w", err)
		}

		return audit.Record(ctx, q, audit.Entry{
			ActorID:    &user.ID,
			ActorIP:    ip,
			Action:     "auth.password_recover",
			TargetType: "user",
			TargetID:   &user.ID,
			Data:       map[string]any{"remaining_codes": remaining},
		})
	})
	if err != nil {
		return err
	}

	for _, tokenHash := range tokenHashes {
		s.deleteSessionCache(ctx, tokenHash)
	}
	if err := s.InvalidateUser(ctx, user.ID); err != nil {
		s.logger.Warn("auth: invalidate user after password recovery", "user_id", user.ID, "error", err)
	}

	return nil
}

// ChangePassword sets a new password for the signed-in principal. The current
// password is not required while a temporary password still has to be
// replaced.
func (s *Service) ChangePassword(ctx context.Context, p *Principal, currentPassword *string, newPassword, ip string) (*Principal, error) {
	if p == nil {
		return nil, apperr.Unauthenticated()
	}
	if p.NextStep == NextStepActivate {
		return nil, apperr.InvalidTransition("Aktivasi akun dulu.")
	}

	identity, err := s.store.GetUserIdentity(ctx, p.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.Unauthenticated()
	}
	if err != nil {
		return nil, fmt.Errorf("auth: get user identity: %w", err)
	}

	if p.NextStep != NextStepChangePassword {
		if currentPassword == nil || strings.TrimSpace(*currentPassword) == "" {
			return nil, apperr.Validation(apperr.Field("currentPassword", "required", "Sandi saat ini wajib diisi."))
		}

		if err := s.allowPasswordConfirm(ctx, p.UserID); err != nil {
			return nil, err
		}

		ok, err := s.verifyPassword(ctx, identity.PasswordHash, *currentPassword, p.UserID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, apperr.Validation(apperr.Field("currentPassword", "wrong", "Sandi saat ini salah."))
		}
	}

	username := ""
	if identity.Username != nil {
		username = *identity.Username
	}
	if err := s.checkPasswordPolicy(newPassword, username, identity.Nisn, identity.StaffNumber, identity.DisplayName); err != nil {
		return nil, err
	}

	newHash, err := s.hasher.Hash(ctx, newPassword)
	if err != nil {
		return nil, fmt.Errorf("auth: hash password: %w", err)
	}

	var tokenHashes [][]byte
	err = s.store.WithTx(ctx, func(q *store.Queries) error {
		if err := q.SetOwnPassword(ctx, store.SetOwnPasswordParams{
			PasswordHash: newHash,
			ID:           p.UserID,
		}); err != nil {
			return fmt.Errorf("auth: set password: %w", err)
		}

		tokenHashes, err = q.RevokeUserSessions(ctx, store.RevokeUserSessionsParams{
			UserID:   p.UserID,
			ExceptID: p.SessionID,
			Reason:   "password_changed",
		})
		if err != nil {
			return fmt.Errorf("auth: revoke sessions: %w", err)
		}

		return audit.Record(ctx, q, audit.Entry{
			ActorID:    &p.UserID,
			ActorIP:    ip,
			Action:     "auth.password_change",
			TargetType: "user",
			TargetID:   &p.UserID,
		})
	})
	if err != nil {
		return nil, err
	}

	for _, tokenHash := range tokenHashes {
		s.deleteSessionCache(ctx, tokenHash)
	}
	if err := s.InvalidateUser(ctx, p.UserID); err != nil {
		s.logger.Warn("auth: invalidate user after password change", "user_id", p.UserID, "error", err)
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

// allowRecover applies the identifier and IP rate limits for password
// recovery. It fails closed when the limiter is unavailable.
func (s *Service) allowRecover(ctx context.Context, identifier, ip string) error {
	result, err := s.limiter.Allow(ctx, "recover:id:"+strings.ToLower(identifier), ratelimit.Rule{
		Limit:  recoverIdentifierLimit,
		Window: recoverWindow,
	})
	if err != nil {
		s.logger.Warn("auth: recovery identifier rate limiter failed", "error", err)
		return apperr.Unavailable("rate limiter")
	}
	if !result.Allowed {
		return apperr.RateLimited()
	}

	if ip == "" {
		return nil
	}

	result, err = s.limiter.Allow(ctx, "recover:ip:"+ip, ratelimit.Rule{
		Limit:  recoverIPLimit,
		Window: recoverWindow,
	})
	if err != nil {
		s.logger.Warn("auth: recovery ip rate limiter failed", "error", err)
		return apperr.Unavailable("rate limiter")
	}
	if !result.Allowed {
		return apperr.RateLimited()
	}

	return nil
}
