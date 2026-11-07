// Package securetoken provides the security primitives used by the
// authentication flows: session tokens, CSRF tokens, recovery codes, and
// generated initial passwords.
//
// Only the standard library is used so the package stays easy to audit.
package securetoken

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"
)

const (
	sessionTokenBytes = 32
	csrfTokenBytes    = 32

	// RecoveryAlphabet avoids visually ambiguous characters (0, 1, i, l, o).
	RecoveryAlphabet = "23456789abcdefghjkmnpqrstuvwxyz"

	recoveryCodeLength  = 10
	recoveryGroupLength = 5
	readableGroups      = 3
	readableGroupLength = 4
)

// SessionToken returns a new random session token (32 bytes, base64url without
// padding) and its SHA-256 hash (raw 32 bytes) for storage. Never store the
// token itself.
func SessionToken() (token string, hash []byte, err error) {
	raw := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("securetoken: generate session token: %w", err)
	}

	token = base64.RawURLEncoding.EncodeToString(raw)

	return token, HashSessionToken(token), nil
}

// HashSessionToken returns the SHA-256 hash of a session token, used for
// database lookups.
func HashSessionToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))

	return sum[:]
}

// CSRFToken returns a random 32-byte base64url token.
func CSRFToken() (string, error) {
	raw := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("securetoken: generate CSRF token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// Equal compares two strings in constant time.
func Equal(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// RecoveryCodes returns n distinct recovery codes in display form
// ("xxxxx-xxxxx").
func RecoveryCodes(n int) ([]string, error) {
	if n <= 0 {
		return []string{}, nil
	}

	codes := make([]string, 0, n)
	seen := make(map[string]struct{}, n)

	for len(codes) < n {
		raw, err := randomAlphabetString(recoveryCodeLength)
		if err != nil {
			return nil, err
		}

		code := raw[:recoveryGroupLength] + "-" + raw[recoveryGroupLength:]
		if _, exists := seen[code]; exists {
			continue
		}

		seen[code] = struct{}{}
		codes = append(codes, code)
	}

	return codes, nil
}

// NormalizeRecoveryCode lowercases input, removes spaces and dashes, and
// validates the result. It returns the 10-character normalized code on
// success.
func NormalizeRecoveryCode(input string) (string, bool) {
	var normalized strings.Builder
	normalized.Grow(recoveryCodeLength)

	for _, r := range strings.ToLower(input) {
		if r == '-' || unicode.IsSpace(r) {
			continue
		}
		normalized.WriteRune(r)
	}

	code := normalized.String()
	if len(code) != recoveryCodeLength {
		return "", false
	}

	for _, r := range code {
		if !strings.ContainsRune(RecoveryAlphabet, r) {
			return "", false
		}
	}

	return code, true
}

// HashRecoveryCode returns hex(HMAC-SHA256(secret, normalizedCode)). Recovery
// codes are high-entropy, so a keyed fast hash is sufficient and allows
// indexed lookup.
func HashRecoveryCode(secret []byte, normalized string) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(normalized))

	return hex.EncodeToString(mac.Sum(nil))
}

// ReadablePassword returns a random initial/temporary password made of three
// groups of four characters joined by dashes, for example "k7m2-pq9x-4hst".
func ReadablePassword() (string, error) {
	raw, err := randomAlphabetString(readableGroups * readableGroupLength)
	if err != nil {
		return "", err
	}

	groups := make([]string, 0, readableGroups)
	for i := 0; i < readableGroups; i++ {
		start := i * readableGroupLength
		groups = append(groups, raw[start:start+readableGroupLength])
	}

	return strings.Join(groups, "-"), nil
}

// randomAlphabetString returns n uniformly distributed characters from
// RecoveryAlphabet. Bytes outside the unbiased range are rejected so every
// character has exactly the same probability.
func randomAlphabetString(n int) (string, error) {
	const maxUnbiasedByte = 256 - (256 % len(RecoveryAlphabet))

	out := make([]byte, 0, n)
	buf := make([]byte, n)

	for len(out) < n {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("securetoken: generate random characters: %w", err)
		}

		for _, b := range buf {
			if int(b) >= maxUnbiasedByte {
				continue
			}

			out = append(out, RecoveryAlphabet[int(b)%len(RecoveryAlphabet)])
			if len(out) == n {
				break
			}
		}
	}

	return string(out), nil
}
