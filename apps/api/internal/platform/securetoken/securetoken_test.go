package securetoken

import (
	"bytes"
	"crypto/sha256"
	"regexp"
	"strings"
	"testing"
)

var (
	base64URLTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)
	recoveryCodePattern   = regexp.MustCompile(`^[2-9a-hj-km-np-z]{5}-[2-9a-hj-km-np-z]{5}$`)
	readablePassword      = regexp.MustCompile(`^[2-9a-hj-km-np-z]{4}-[2-9a-hj-km-np-z]{4}-[2-9a-hj-km-np-z]{4}$`)
	hexHashPattern        = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

func TestSessionToken(t *testing.T) {
	t.Parallel()

	token, hash, err := SessionToken()
	if err != nil {
		t.Fatalf("SessionToken() error = %v", err)
	}

	if !base64URLTokenPattern.MatchString(token) {
		t.Fatalf("SessionToken() token = %q, want 43 base64url characters", token)
	}
	if len(hash) != sha256.Size {
		t.Fatalf("SessionToken() hash length = %d, want %d", len(hash), sha256.Size)
	}
	if !bytes.Equal(hash, HashSessionToken(token)) {
		t.Fatal("SessionToken() hash does not match HashSessionToken(token)")
	}

	other, _, err := SessionToken()
	if err != nil {
		t.Fatalf("SessionToken() second call error = %v", err)
	}
	if token == other {
		t.Fatal("two session tokens are identical, want unique tokens")
	}
}

func TestCSRFToken(t *testing.T) {
	t.Parallel()

	token, err := CSRFToken()
	if err != nil {
		t.Fatalf("CSRFToken() error = %v", err)
	}
	if !base64URLTokenPattern.MatchString(token) {
		t.Fatalf("CSRFToken() = %q, want 43 base64url characters", token)
	}

	other, err := CSRFToken()
	if err != nil {
		t.Fatalf("CSRFToken() second call error = %v", err)
	}
	if token == other {
		t.Fatal("two CSRF tokens are identical, want unique tokens")
	}
}

func TestEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    string
		b    string
		want bool
	}{
		{name: "equal", a: "token-value", b: "token-value", want: true},
		{name: "equal empty", a: "", b: "", want: true},
		{name: "different", a: "token-value", b: "token-valuf", want: false},
		{name: "different length", a: "token", b: "token-value", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := Equal(test.a, test.b); got != test.want {
				t.Fatalf("Equal(%q, %q) = %v, want %v", test.a, test.b, got, test.want)
			}
		})
	}
}

func TestRecoveryCodes(t *testing.T) {
	t.Parallel()

	codes, err := RecoveryCodes(8)
	if err != nil {
		t.Fatalf("RecoveryCodes(8) error = %v", err)
	}
	if len(codes) != 8 {
		t.Fatalf("RecoveryCodes(8) returned %d codes, want 8", len(codes))
	}

	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		if !recoveryCodePattern.MatchString(code) {
			t.Fatalf("RecoveryCodes() code = %q, want xxxxx-xxxxx format", code)
		}
		if _, exists := seen[code]; exists {
			t.Fatalf("RecoveryCodes() returned duplicate code %q", code)
		}
		seen[code] = struct{}{}
	}
}

func TestNormalizeRecoveryCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		want      string
		wantValid bool
	}{
		{name: "display form", input: "k7m2p-q9x4h", want: "k7m2pq9x4h", wantValid: true},
		{name: "mixed case and space", input: "K7M2P Q9X4H", want: "k7m2pq9x4h", wantValid: true},
		{name: "uppercase display form", input: "K7M2P-Q9X4H", want: "k7m2pq9x4h", wantValid: true},
		{name: "all letters and digits", input: "ABCDE-23456", want: "abcde23456", wantValid: true},
		{name: "too short", input: "k7m2pq9x4", wantValid: false},
		{name: "too long", input: "k7m2pq9x4hh", wantValid: false},
		{name: "contains zero", input: "0k7m2pq9x4", wantValid: false},
		{name: "contains one", input: "1k7m2pq9x4", wantValid: false},
		{name: "contains i", input: "ik7m2pq9x4", wantValid: false},
		{name: "contains l", input: "lk7m2pq9x4", wantValid: false},
		{name: "contains o", input: "ok7m2pq9x4", wantValid: false},
		{name: "contains punctuation", input: "k7m2_pq9x4", wantValid: false},
		{name: "empty", input: "", wantValid: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, ok := NormalizeRecoveryCode(test.input)
			if ok != test.wantValid {
				t.Fatalf("NormalizeRecoveryCode(%q) valid = %v, want %v", test.input, ok, test.wantValid)
			}
			if got != test.want {
				t.Fatalf("NormalizeRecoveryCode(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestHashRecoveryCode(t *testing.T) {
	t.Parallel()

	secret := []byte("recovery-secret")
	first := HashRecoveryCode(secret, "k7m2pq9x4h")
	second := HashRecoveryCode(secret, "k7m2pq9x4h")

	if first != second {
		t.Fatal("HashRecoveryCode() is not deterministic for the same input")
	}
	if !hexHashPattern.MatchString(first) {
		t.Fatalf("HashRecoveryCode() = %q, want 64 hex characters", first)
	}

	if other := HashRecoveryCode([]byte("different-secret"), "k7m2pq9x4h"); other == first {
		t.Fatal("HashRecoveryCode() produced the same hash for different secrets")
	}
	if other := HashRecoveryCode(secret, "h4stx9qp2m7k"); other == first {
		t.Fatal("HashRecoveryCode() produced the same hash for different codes")
	}
}

func TestReadablePassword(t *testing.T) {
	t.Parallel()

	password, err := ReadablePassword()
	if err != nil {
		t.Fatalf("ReadablePassword() error = %v", err)
	}
	if !readablePassword.MatchString(password) {
		t.Fatalf("ReadablePassword() = %q, want xxxx-xxxx-xxxx format", password)
	}

	other, err := ReadablePassword()
	if err != nil {
		t.Fatalf("ReadablePassword() second call error = %v", err)
	}
	if password == other {
		t.Fatal("two generated passwords are identical, want unique passwords")
	}
}

func TestRecoveryAlphabetDistribution(t *testing.T) {
	t.Parallel()

	const codeCount = 2000

	codes, err := RecoveryCodes(codeCount)
	if err != nil {
		t.Fatalf("RecoveryCodes(%d) error = %v", codeCount, err)
	}

	counts := make(map[rune]int, len(RecoveryAlphabet))
	for _, code := range codes {
		for _, r := range strings.ReplaceAll(code, "-", "") {
			counts[r]++
		}
	}

	if len(counts) != len(RecoveryAlphabet) {
		t.Fatalf("generated %d distinct characters, want %d", len(counts), len(RecoveryAlphabet))
	}

	total := codeCount * recoveryCodeLength
	mean := float64(total) / float64(len(RecoveryAlphabet))
	lower := mean * 0.75
	upper := mean * 1.25

	for _, r := range RecoveryAlphabet {
		frequency := float64(counts[r])
		if frequency < lower || frequency > upper {
			t.Fatalf("character %q frequency = %.0f, want between %.1f and %.1f", r, frequency, lower, upper)
		}
	}
}
