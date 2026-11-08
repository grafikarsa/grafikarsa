package users

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const selfActionMessage = "Tidak bisa dilakukan pada akunmu sendiri."

// Deactivate marks an account inactive and revokes all of its sessions.
func (s *Service) Deactivate(ctx context.Context, a Actor, id uuid.UUID) error {
	scope, ok := a.permissionScope(permissionDeactivate)
	if !ok {
		return apperr.NotFound()
	}
	if a.Principal.UserID == id {
		return apperr.Conflict(selfActionMessage)
	}

	err := s.st.WithTx(ctx, func(q *store.Queries) error {
		target, err := s.loadTarget(ctx, q, id)
		if err != nil {
			return err
		}
		if !scope.allows(target.majorIDs()) {
			return apperr.NotFound()
		}

		if err := q.DeactivateUser(ctx, id); err != nil {
			return fmt.Errorf("users: deactivate: %w", err)
		}
		if err := revokeUserSessions(ctx, q, id, "deactivated"); err != nil {
			return err
		}
		if err := deleteVaultEntry(ctx, q, id); err != nil {
			return err
		}
		return s.recordAudit(ctx, q, a, "user.deactivate", id, target.auditScopeMajorID(), nil)
	})
	if err != nil {
		return err
	}
	return s.invalidate(ctx, id)
}

// Reactivate restores an inactive account. Accounts with a graduation year
// become alumni again; everyone else becomes active.
func (s *Service) Reactivate(ctx context.Context, a Actor, id uuid.UUID) error {
	scope, ok := a.permissionScope(permissionDeactivate)
	if !ok {
		return apperr.NotFound()
	}

	err := s.st.WithTx(ctx, func(q *store.Queries) error {
		target, err := s.loadTarget(ctx, q, id)
		if err != nil {
			return err
		}
		if !scope.allows(target.majorIDs()) {
			return apperr.NotFound()
		}

		if err := q.ReactivateUser(ctx, id); err != nil {
			return fmt.Errorf("users: reactivate: %w", err)
		}
		return s.recordAudit(ctx, q, a, "user.reactivate", id, target.auditScopeMajorID(), nil)
	})
	if err != nil {
		return err
	}
	return s.invalidate(ctx, id)
}

// ResetPassword issues a temporary password for an activated account and
// revokes every existing session. The plain password is returned once.
func (s *Service) ResetPassword(ctx context.Context, a Actor, id uuid.UUID) (string, time.Time, error) {
	scope, ok := a.permissionScope(permissionReset)
	if !ok {
		return "", time.Time{}, apperr.NotFound()
	}
	if a.Principal.UserID == id {
		return "", time.Time{}, apperr.Conflict(selfActionMessage)
	}

	plain, hash, err := s.generatePassword(ctx)
	if err != nil {
		return "", time.Time{}, err
	}

	ttlHours := s.settingInt(ctx, temporaryPasswordTTLKey, temporaryPasswordTTLHours)
	expiresAt := s.now().UTC().Add(time.Duration(ttlHours) * time.Hour)

	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		target, err := s.loadTarget(ctx, q, id)
		if err != nil {
			return err
		}
		if !scope.allows(target.majorIDs()) {
			return apperr.NotFound()
		}
		if !target.row.Activated {
			return apperr.InvalidTransition("Akun belum aktivasi, gunakan terbitkan ulang sandi awal.")
		}

		if err := q.SetUserPassword(ctx, store.SetUserPasswordParams{
			PasswordHash:      hash,
			PasswordKind:      "temporary",
			PasswordExpiresAt: expiresAt,
			ID:                id,
		}); err != nil {
			return fmt.Errorf("users: set temporary password: %w", err)
		}
		if err := revokeUserSessions(ctx, q, id, "admin_reset"); err != nil {
			return err
		}
		if err := deleteVaultEntry(ctx, q, id); err != nil {
			return err
		}
		return s.recordAudit(ctx, q, a, "user.reset_password", id, target.auditScopeMajorID(), nil)
	})
	if err != nil {
		return "", time.Time{}, err
	}

	if err := s.invalidate(ctx, id); err != nil {
		return "", time.Time{}, err
	}
	return plain, expiresAt, nil
}

// ReissueInitialPassword replaces the initial password of an account that has
// not been activated yet. The plain password is returned once.
func (s *Service) ReissueInitialPassword(ctx context.Context, a Actor, id uuid.UUID) (string, time.Time, error) {
	scope, ok := a.permissionScope(permissionReset)
	if !ok {
		return "", time.Time{}, apperr.NotFound()
	}

	plain, hash, err := s.generatePassword(ctx)
	if err != nil {
		return "", time.Time{}, err
	}

	ttlDays := s.settingInt(ctx, initialPasswordTTLKey, initialPasswordTTLDays)
	expiresAt := s.now().UTC().Add(time.Duration(ttlDays) * 24 * time.Hour)

	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		target, err := s.loadTarget(ctx, q, id)
		if err != nil {
			return err
		}
		if !scope.allows(target.majorIDs()) {
			return apperr.NotFound()
		}
		if target.row.Activated {
			return apperr.InvalidTransition("Akun sudah aktivasi, gunakan reset sandi.")
		}

		if err := q.SetUserPassword(ctx, store.SetUserPasswordParams{
			PasswordHash:      hash,
			PasswordKind:      "initial",
			PasswordExpiresAt: expiresAt,
			ID:                id,
		}); err != nil {
			return fmt.Errorf("users: set initial password: %w", err)
		}
		if err := s.putVaultEntry(ctx, q, id, plain, expiresAt); err != nil {
			return err
		}
		return s.recordAudit(ctx, q, a, "user.reissue_initial_password", id, target.auditScopeMajorID(), nil)
	})
	if err != nil {
		return "", time.Time{}, err
	}
	return plain, expiresAt, nil
}

// BlockContent revokes content creation for one account. This is a global
// permission; scoped actors are treated as not found.
func (s *Service) BlockContent(ctx context.Context, a Actor, id uuid.UUID, reason string) error {
	scope, ok := a.permissionScope(permissionBlockContent)
	if !ok || !scope.global {
		return apperr.NotFound()
	}

	reason = strings.TrimSpace(reason)
	if err := validateBlockReason(reason); err != nil {
		return err
	}

	err := s.st.WithTx(ctx, func(q *store.Queries) error {
		target, err := s.loadTarget(ctx, q, id)
		if err != nil {
			return err
		}
		if err := q.BlockUserContent(ctx, store.BlockUserContentParams{
			Reason: reason,
			ID:     id,
		}); err != nil {
			return fmt.Errorf("users: block content: %w", err)
		}
		return s.recordAudit(ctx, q, a, "user.block_content", id, target.auditScopeMajorID(), map[string]any{
			"reason": reason,
		})
	})
	if err != nil {
		return err
	}
	return s.invalidate(ctx, id)
}

// UnblockContent restores content creation for one account.
func (s *Service) UnblockContent(ctx context.Context, a Actor, id uuid.UUID) error {
	scope, ok := a.permissionScope(permissionBlockContent)
	if !ok || !scope.global {
		return apperr.NotFound()
	}

	err := s.st.WithTx(ctx, func(q *store.Queries) error {
		target, err := s.loadTarget(ctx, q, id)
		if err != nil {
			return err
		}
		if err := q.UnblockUserContent(ctx, id); err != nil {
			return fmt.Errorf("users: unblock content: %w", err)
		}
		return s.recordAudit(ctx, q, a, "user.unblock_content", id, target.auditScopeMajorID(), nil)
	})
	if err != nil {
		return err
	}
	return s.invalidate(ctx, id)
}

func revokeUserSessions(ctx context.Context, q *store.Queries, userID uuid.UUID, reason string) error {
	if _, err := q.RevokeUserSessions(ctx, store.RevokeUserSessionsParams{
		UserID:   userID,
		ExceptID: uuid.Nil,
		Reason:   reason,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("users: revoke sessions: %w", err)
	}
	return nil
}
