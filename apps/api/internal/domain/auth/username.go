package auth

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	usernameMinLength = 3
	usernameMaxLength = 30

	usernameInvalidFormat = "invalid_format"
	usernameReserved      = "reserved"
	usernameTaken         = "taken"
)

// usernamePattern mirrors the users.username CHECK constraint. It requires an
// alphanumeric first character and forbids a trailing dot.
var usernamePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9._]{1,28})[a-z0-9_]$`)

// NormalizeUsername lowercases and trims; it does not validate.
func NormalizeUsername(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

// ValidateUsernameFormat returns "invalid_format" when username is not an
// acceptable username, or an empty string when it is valid.
func ValidateUsernameFormat(username string) string {
	if len(username) < usernameMinLength || len(username) > usernameMaxLength {
		return usernameInvalidFormat
	}
	if strings.Contains(username, "..") {
		return usernameInvalidFormat
	}
	if !usernamePattern.MatchString(username) {
		return usernameInvalidFormat
	}
	return ""
}

// UsernameCheck is the result of checking whether a username can be used.
// Reason is empty when the username is available.
type UsernameCheck struct {
	Username  string
	Available bool
	Reason    string
}

// CheckUsername validates the format, reserved list, and uniqueness of raw.
// The principal's own current username counts as available.
func (s *Service) CheckUsername(ctx context.Context, p *Principal, raw string) (UsernameCheck, error) {
	if p == nil {
		return UsernameCheck{}, apperr.Unauthenticated()
	}

	normalized := NormalizeUsername(raw)
	check := UsernameCheck{Username: normalized}

	if reason := ValidateUsernameFormat(normalized); reason != "" {
		check.Reason = reason
		return check, nil
	}

	if p.Username != nil && NormalizeUsername(*p.Username) == normalized {
		check.Available = true
		return check, nil
	}

	reserved, err := s.store.IsUsernameReserved(ctx, normalized)
	if err != nil {
		return UsernameCheck{}, fmt.Errorf("auth: check reserved username: %w", err)
	}
	if reserved {
		check.Reason = usernameReserved
		return check, nil
	}

	taken, err := s.store.IsUsernameTaken(ctx, store.IsUsernameTakenParams{
		Username: normalized,
		UserID:   p.UserID,
	})
	if err != nil {
		return UsernameCheck{}, fmt.Errorf("auth: check taken username: %w", err)
	}
	if taken {
		check.Reason = usernameTaken
		return check, nil
	}

	check.Available = true
	return check, nil
}

// usernameError maps a UsernameCheck reason to a validation error on the
// username field.
func usernameError(reason string) *apperr.Error {
	switch reason {
	case usernameReserved:
		return apperr.Validation(apperr.Field("username", usernameReserved,
			"Username ini tidak boleh dipakai."))
	case usernameTaken:
		return apperr.Validation(apperr.Field("username", usernameTaken,
			"Username sudah dipakai."))
	default:
		return apperr.Validation(apperr.Field("username", usernameInvalidFormat,
			"Username 3–30 karakter: huruf kecil, angka, titik, atau garis bawah."))
	}
}
