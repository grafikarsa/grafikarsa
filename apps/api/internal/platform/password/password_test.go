package password

import (
	"context"
	"encoding/base64"
	"errors"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

var testParams = Params{
	Memory:      1024,
	Iterations:  1,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

func newTestHasher(t *testing.T) *Hasher {
	t.Helper()

	return NewHasher(testParams, 2)
}

func TestHashAndVerify(t *testing.T) {
	t.Parallel()

	h := newTestHasher(t)
	ctx := context.Background()
	const plain = "Kopi-Susu-2026!"

	encoded, err := h.Hash(ctx, plain)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	ok, needsRehash, err := h.Verify(ctx, encoded, plain)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !ok {
		t.Fatal("Verify() = false, want true for the correct password")
	}
	if needsRehash {
		t.Fatal("Verify() needsRehash = true, want false for current parameters")
	}

	ok, needsRehash, err = h.Verify(ctx, encoded, "salah-Banget-2026!")
	if err != nil {
		t.Fatalf("Verify(wrong password) error = %v, want nil", err)
	}
	if ok {
		t.Fatal("Verify(wrong password) = true, want false")
	}
	if needsRehash {
		t.Fatal("Verify(wrong password) needsRehash = true, want false")
	}
}

func TestHashFormat(t *testing.T) {
	t.Parallel()

	h := newTestHasher(t)
	encoded, err := h.Hash(context.Background(), "Kopi-Susu-2026!")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	pattern := regexp.MustCompile(`^\$argon2id\$v=19\$m=1024,t=1,p=1\$[A-Za-z0-9+/]{22}\$[A-Za-z0-9+/]{43}$`)
	if !pattern.MatchString(encoded) {
		t.Fatalf("Hash() = %q, want PHC Argon2id format", encoded)
	}
}

func TestHashUsesRandomSalt(t *testing.T) {
	t.Parallel()

	h := newTestHasher(t)
	ctx := context.Background()

	first, err := h.Hash(ctx, "Kopi-Susu-2026!")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	second, err := h.Hash(ctx, "Kopi-Susu-2026!")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if first == second {
		t.Fatal("two hashes of the same password are identical, want different random salts")
	}
}

func TestVerifyBcrypt(t *testing.T) {
	t.Parallel()

	h := newTestHasher(t)
	ctx := context.Background()
	const plain = "Kopi-Susu-2026!"

	encoded, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt.GenerateFromPassword() error = %v", err)
	}

	ok, needsRehash, err := h.Verify(ctx, string(encoded), plain)
	if err != nil {
		t.Fatalf("Verify(bcrypt) error = %v", err)
	}
	if !ok {
		t.Fatal("Verify(bcrypt) = false, want true")
	}
	if !needsRehash {
		t.Fatal("Verify(bcrypt) needsRehash = false, want true")
	}

	ok, needsRehash, err = h.Verify(ctx, string(encoded), "salah-Banget-2026!")
	if err != nil {
		t.Fatalf("Verify(bcrypt wrong password) error = %v, want nil", err)
	}
	if ok || needsRehash {
		t.Fatalf("Verify(bcrypt wrong password) = (%v, %v), want (false, false)", ok, needsRehash)
	}
}

func TestVerifyArgon2idWithDifferentParamsNeedsRehash(t *testing.T) {
	t.Parallel()

	otherParams := testParams
	otherParams.Memory = 2048
	other := NewHasher(otherParams, 1)

	encoded, err := other.Hash(context.Background(), "Kopi-Susu-2026!")
	if err != nil {
		t.Fatalf("Hash() with other params error = %v", err)
	}

	ok, needsRehash, err := newTestHasher(t).Verify(context.Background(), encoded, "Kopi-Susu-2026!")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !ok {
		t.Fatal("Verify() = false, want true")
	}
	if !needsRehash {
		t.Fatal("Verify() needsRehash = false, want true for different stored parameters")
	}
}

func TestVerifyMalformedOrUnknownFormats(t *testing.T) {
	t.Parallel()

	salt := []byte("0123456789abcdef")
	key := make([]byte, testParams.KeyLength)
	valid := "$argon2id$v=19$m=1024,t=1,p=1$" +
		base64.RawStdEncoding.EncodeToString(salt) + "$" +
		base64.RawStdEncoding.EncodeToString(key)

	tests := []struct {
		name    string
		encoded string
		wantErr error
	}{
		{name: "empty", encoded: "", wantErr: ErrUnknownFormat},
		{name: "plain text", encoded: "not-a-hash", wantErr: ErrUnknownFormat},
		{name: "argon2i", encoded: strings.Replace(valid, "$argon2id$", "$argon2i$", 1), wantErr: ErrUnknownFormat},
		{name: "unknown bcrypt version", encoded: "$2x$10$abcdefghijklmnopqrstuv", wantErr: ErrUnknownFormat},
		{name: "argon2id truncated", encoded: "$argon2id$v=19", wantErr: ErrMalformedHash},
		{name: "argon2id old version", encoded: strings.Replace(valid, "v=19", "v=16", 1), wantErr: ErrMalformedHash},
		{name: "argon2id bad memory", encoded: strings.Replace(valid, "m=1024", "m=abc", 1), wantErr: ErrMalformedHash},
		{name: "argon2id missing param", encoded: strings.Replace(valid, ",p=1", "", 1), wantErr: ErrMalformedHash},
		{name: "argon2id zero iterations", encoded: strings.Replace(valid, "t=1", "t=0", 1), wantErr: ErrMalformedHash},
		{name: "argon2id bad salt base64", encoded: strings.Replace(valid, base64.RawStdEncoding.EncodeToString(salt), "***", 1), wantErr: ErrMalformedHash},
		{name: "argon2id bad key base64", encoded: strings.Replace(valid, base64.RawStdEncoding.EncodeToString(key), "***", 1), wantErr: ErrMalformedHash},
		{name: "argon2id short salt", encoded: strings.Replace(valid, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString([]byte("short")), 1), wantErr: ErrMalformedHash},
		{name: "bcrypt malformed", encoded: "$2a$not-a-real-hash", wantErr: ErrMalformedHash},
	}

	h := newTestHasher(t)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ok, needsRehash, err := h.Verify(context.Background(), test.encoded, "Kopi-Susu-2026!")
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Verify() error = %v, want %v", err, test.wantErr)
			}
			if ok || needsRehash {
				t.Fatalf("Verify() = (%v, %v), want (false, false)", ok, needsRehash)
			}
		})
	}
}

func TestHashCancelledWhileSemaphoreFull(t *testing.T) {
	t.Parallel()

	h := NewHasher(testParams, 1)

	// Occupy the only slot, then cancel the context before Hash can acquire.
	h.sem <- struct{}{}
	defer func() { <-h.sem }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := h.Hash(ctx, "Kopi-Susu-2026!")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Hash() error = %v, want context.Canceled", err)
	}
}

func TestDefaultParamsHashAndVerify(t *testing.T) {
	t.Parallel()

	h := NewHasher(DefaultParams, 1)
	ctx := context.Background()

	encoded, err := h.Hash(ctx, "Kopi-Susu-2026!")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	ok, needsRehash, err := h.Verify(ctx, encoded, "Kopi-Susu-2026!")
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !ok || needsRehash {
		t.Fatalf("Verify() = (%v, %v), want (true, false)", ok, needsRehash)
	}
}

func TestCheckPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		plain      string
		identities []string
		want       []Violation
	}{
		{name: "valid password", plain: "Kopi-Susu-2026!"},
		{name: "too short", plain: "Kopi1!", want: []Violation{TooShort}},
		{name: "empty", plain: "", want: []Violation{TooShort}},
		{name: "too short counts runes", plain: "åß∂ƒ©˙∆", want: []Violation{TooShort}},
		{name: "eight runes is long enough", plain: "åß∂ƒ©˙∆˚"},
		{name: "too long", plain: strings.Repeat("Kopi-Susu-2026!", 9), want: []Violation{TooLong}},
		{name: "common", plain: "password", want: []Violation{TooCommon}},
		{name: "common case insensitive", plain: "PASSWORD1", want: []Violation{TooCommon}},
		{name: "indonesian common", plain: "Bismillah", want: []Violation{TooCommon}},
		{name: "contains identity", plain: "Budi-Kopi-2026!", identities: []string{"budi"}, want: []Violation{ContainsIdentity}},
		{name: "identity case insensitive", plain: "Budi-Kopi-2026!", identities: []string{"BUDI"}, want: []Violation{ContainsIdentity}},
		{name: "short identity ignored", plain: "abc-Kopi-2026!", identities: []string{"abc"}},
		{name: "repetitive", plain: "aaaaaaaa", want: []Violation{Repetitive}},
		{name: "repetitive and identity", plain: "aaaaaaaa", identities: []string{"aaaa"}, want: []Violation{ContainsIdentity, Repetitive}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := CheckPolicy(test.plain, test.identities...)
			if len(got) != len(test.want) {
				t.Fatalf("CheckPolicy(%q, %v) = %v, want %v", test.plain, test.identities, got, test.want)
			}
			for i := range test.want {
				if got[i] != test.want[i] {
					t.Fatalf("CheckPolicy(%q, %v) = %v, want %v", test.plain, test.identities, got, test.want)
				}
			}
		})
	}
}

func TestViolationMessage(t *testing.T) {
	t.Parallel()

	all := []Violation{TooShort, TooLong, TooCommon, ContainsIdentity, Repetitive}
	for _, v := range all {
		if message := ViolationMessage(v); message == "" {
			t.Fatalf("ViolationMessage(%q) is empty", v)
		}
	}

	if got, want := ViolationMessage(TooShort), "Sandi minimal 8 karakter."; got != want {
		t.Fatalf("ViolationMessage(TooShort) = %q, want %q", got, want)
	}
}

func TestCommonPasswordsList(t *testing.T) {
	t.Parallel()

	if len(commonPasswords) < 200 {
		t.Fatalf("common password list has %d entries, want at least 200", len(commonPasswords))
	}

	required := []string{
		"password", "12345678", "123456789", "qwerty123", "bismillah",
		"indonesia", "sayangku", "rahasia123", "grafika", "grafikarsa",
		"smkn4malang", "malang123", "admin123", "iloveyou", "password1",
		"11111111", "00000000", "asdfghjk", "qwertyui", "cintaku",
	}
	for _, entry := range required {
		if _, ok := commonPasswords[entry]; !ok {
			t.Fatalf("common password list is missing %q", entry)
		}
	}

	for _, line := range strings.Split(commonPasswordsRaw, "\n") {
		if line == "" {
			continue
		}
		if line != strings.ToLower(line) {
			t.Fatalf("common password entry %q is not lowercase", line)
		}
	}
}
