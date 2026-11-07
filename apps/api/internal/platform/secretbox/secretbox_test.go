package secretbox

import (
	"bytes"
	"testing"
)

var testSecret = []byte("test-secret-key-that-is-long-enough-for-hkdf")

func TestSealOpen_RoundTrip(t *testing.T) {
	t.Parallel()

	box, err := New(testSecret)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ciphertext, err := box.Seal([]byte("k7m2-pq9x-4hst"), []byte("user-1"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if bytes.Contains(ciphertext, []byte("k7m2-pq9x-4hst")) {
		t.Fatal("ciphertext contains the plaintext")
	}

	plaintext, err := box.Open(ciphertext, []byte("user-1"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if string(plaintext) != "k7m2-pq9x-4hst" {
		t.Fatalf("Open = %q, want the sealed plaintext", plaintext)
	}

	// A fresh seal must use a fresh nonce even for identical input.
	other, err := box.Seal([]byte("k7m2-pq9x-4hst"), []byte("user-1"))
	if err != nil {
		t.Fatalf("Seal second: %v", err)
	}
	if bytes.Equal(ciphertext, other) {
		t.Fatal("two seals of the same plaintext are identical, want a random nonce each time")
	}
}

func TestOpen_WrongAssociatedDataFails(t *testing.T) {
	t.Parallel()

	box, err := New(testSecret)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ciphertext, err := box.Seal([]byte("sandi-awal"), []byte("user-1"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	if _, err := box.Open(ciphertext, []byte("user-2")); err == nil {
		t.Fatal("Open with a different associated data succeeded, want authentication failure")
	}
}

func TestOpen_TamperedCiphertextFails(t *testing.T) {
	t.Parallel()

	box, err := New(testSecret)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ciphertext, err := box.Seal([]byte("sandi-awal"), []byte("user-1"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	tampered := append([]byte(nil), ciphertext...)
	tampered[len(tampered)-1] ^= 0x01
	if _, err := box.Open(tampered, []byte("user-1")); err == nil {
		t.Fatal("Open of a tampered ciphertext succeeded, want authentication failure")
	}

	short := ciphertext[:nonceSize-1]
	if _, err := box.Open(short, []byte("user-1")); err == nil {
		t.Fatal("Open of a short ciphertext succeeded, want an error")
	}
}

func TestOpen_DifferentSecretFails(t *testing.T) {
	t.Parallel()

	box, err := New(testSecret)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ciphertext, err := box.Seal([]byte("sandi-awal"), []byte("user-1"))
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	other, err := New([]byte("another-secret-key-that-is-also-long-enough"))
	if err != nil {
		t.Fatalf("New other: %v", err)
	}
	if _, err := other.Open(ciphertext, []byte("user-1")); err == nil {
		t.Fatal("Open with a different secret succeeded, want authentication failure")
	}
}

func TestNew_ShortSecretRejected(t *testing.T) {
	t.Parallel()

	if _, err := New([]byte("too-short")); err == nil {
		t.Fatal("New with a short secret succeeded, want an error")
	}
}
