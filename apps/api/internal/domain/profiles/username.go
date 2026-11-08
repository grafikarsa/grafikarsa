package profiles

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/audit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

// ChangeUsername renames the authenticated user, records the old username for
// redirects, writes an audit entry, and invalidates the cached principal.
func (s *Service) ChangeUsername(ctx context.Context, p *auth.Principal, raw string) (*MyProfile, error) {
	if p == nil {
		return nil, apperr.Unauthenticated()
	}
	if s.auth == nil {
		return nil, fmt.Errorf("profiles: username checker is not configured")
	}

	normalized := auth.NormalizeUsername(raw)
	check, err := s.auth.CheckUsername(ctx, p, normalized)
	if err != nil {
		return nil, fmt.Errorf("profiles: check username: %w", err)
	}
	if !check.Available {
		return nil, usernameError(check.Reason)
	}

	state, err := s.st.GetUsernameState(ctx, p.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound()
		}
		return nil, fmt.Errorf("profiles: get username state: %w", err)
	}
	if state.Username == normalized {
		return s.GetMe(ctx, p)
	}

	now := s.now().UTC()
	if state.UsernameChangedAt != nil {
		days := s.settingInt(ctx, "profile.username_change_days", defaultUsernameChangeDays)
		available := state.UsernameChangedAt.Add(time.Duration(days) * 24 * time.Hour)
		if now.Before(available) {
			return nil, apperr.LimitReached(fmt.Sprintf(
				"Username baru bisa diganti lagi pada %s.", formatDateID(available),
			))
		}
	}

	// Releasing the old username after redirect_until (setting released_at)
	// will be a periodic job in a later PR; this API only sets the window.
	redirectDays := s.settingInt(ctx, "profile.username_redirect_days", defaultUsernameRedirectDays)
	redirectUntil := now.Add(time.Duration(redirectDays) * 24 * time.Hour)
	oldUsername := state.Username

	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		if err := q.InsertUsernameHistory(ctx, store.InsertUsernameHistoryParams{
			UserID:        p.UserID,
			OldUsername:   oldUsername,
			RedirectUntil: redirectUntil,
		}); err != nil {
			return fmt.Errorf("profiles: insert username history: %w", err)
		}
		if err := q.SetUsername(ctx, store.SetUsernameParams{
			ID:       p.UserID,
			Username: normalized,
		}); err != nil {
			if isUniqueViolation(err) {
				return apperr.Validation(apperr.Field(
					"username", "taken", "Username sudah dipakai.",
				))
			}
			return fmt.Errorf("profiles: set username: %w", err)
		}
		actorID := p.UserID
		if err := audit.Record(ctx, q, audit.Entry{
			ActorID:    &actorID,
			Action:     "user.change_username",
			TargetType: targetTypeUser,
			TargetID:   &actorID,
			Data:       map[string]any{"old": oldUsername, "new": normalized},
		}); err != nil {
			return fmt.Errorf("profiles: record username change: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := s.auth.InvalidateUser(ctx, p.UserID); err != nil {
		return nil, fmt.Errorf("profiles: invalidate principal: %w", err)
	}
	return s.GetMe(ctx, p)
}

// usernameError maps an auth username reason to a validation error.
func usernameError(reason string) *apperr.Error {
	switch reason {
	case "reserved":
		return apperr.Validation(apperr.Field("username", "reserved", "Username ini tidak boleh dipakai."))
	case "taken":
		return apperr.Validation(apperr.Field("username", "taken", "Username sudah dipakai."))
	default:
		return apperr.Validation(apperr.Field(
			"username", "invalid_format",
			"Username 3–30 karakter: huruf kecil, angka, titik, atau garis bawah.",
		))
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// formatDateID renders a date with Indonesian month names.
func formatDateID(t time.Time) string {
	months := [...]string{
		"Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember",
	}
	return fmt.Sprintf("%d %s %d", t.Day(), months[t.Month()-1], t.Year())
}
