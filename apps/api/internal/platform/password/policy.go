package password

import (
	_ "embed"
	"strings"
)

// Violation identifies a password policy rule that was not satisfied.
type Violation string

// Password policy violations.
const (
	TooShort         Violation = "too_short"
	TooLong          Violation = "too_long"
	TooCommon        Violation = "too_common"
	ContainsIdentity Violation = "contains_identity"
	Repetitive       Violation = "repetitive"
)

const (
	minPasswordLength = 8
	maxPasswordLength = 128
	minIdentityLength = 4
)

//go:embed common.txt
var commonPasswordsRaw string

var commonPasswords = parseCommonPasswords(commonPasswordsRaw)

func parseCommonPasswords(raw string) map[string]struct{} {
	entries := make(map[string]struct{})
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entries[strings.ToLower(line)] = struct{}{}
	}

	return entries
}

// CheckPolicy returns every policy violation found in plain. An empty slice
// means the password is acceptable. Identities are compared case-insensitively
// and only entries with at least 4 characters are considered.
func CheckPolicy(plain string, identities ...string) []Violation {
	violations := make([]Violation, 0, 2)
	runes := []rune(plain)

	if len(runes) < minPasswordLength {
		violations = append(violations, TooShort)
	}

	if len(runes) > maxPasswordLength {
		violations = append(violations, TooLong)
	}

	lower := strings.ToLower(plain)
	if _, common := commonPasswords[lower]; common {
		violations = append(violations, TooCommon)
	}

	for _, identity := range identities {
		identity = strings.TrimSpace(identity)
		if len([]rune(identity)) < minIdentityLength {
			continue
		}
		if strings.Contains(lower, strings.ToLower(identity)) {
			violations = append(violations, ContainsIdentity)
			break
		}
	}

	if isRepetitive(runes) {
		violations = append(violations, Repetitive)
	}

	return violations
}

// ViolationMessage returns a user-facing Indonesian explanation for v.
func ViolationMessage(v Violation) string {
	switch v {
	case TooShort:
		return "Sandi minimal 8 karakter."
	case TooLong:
		return "Sandi maksimal 128 karakter."
	case TooCommon:
		return "Sandi terlalu umum dan mudah ditebak."
	case ContainsIdentity:
		return "Sandi tidak boleh memuat nama, email, atau username Anda."
	case Repetitive:
		return "Sandi tidak boleh berisi karakter yang sama berulang."
	default:
		return "Sandi tidak valid."
	}
}

func isRepetitive(runes []rune) bool {
	if len(runes) < 2 {
		return false
	}

	for _, r := range runes[1:] {
		if r != runes[0] {
			return false
		}
	}

	return true
}
