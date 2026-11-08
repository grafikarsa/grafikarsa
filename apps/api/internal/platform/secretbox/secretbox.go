// Package secretbox encrypts small secrets at rest with AES-256-GCM. The
// encryption key is derived from the application secret with HKDF-SHA256, so
// rotating GK_SECRET_KEY makes old ciphertexts unreadable.
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"
)

const (
	keyLength = 32
	nonceSize = 12
	// keyInfo separates this key from other uses of the application secret.
	keyInfo = "grafikarsa/secretbox/v1"
)

// Box seals and opens small secrets with AES-256-GCM.
type Box struct {
	aead cipher.AEAD
}

// New derives a 32-byte AES key from secret with HKDF-SHA256 (nil salt, fixed
// info) and returns a ready Box. Secrets shorter than 32 bytes are rejected.
func New(secret []byte) (*Box, error) {
	if len(secret) < keyLength {
		return nil, fmt.Errorf("secretbox: secret must be at least %d bytes", keyLength)
	}

	key := make([]byte, keyLength)
	if _, err := io.ReadFull(hkdf.New(sha256.New, secret, nil, []byte(keyInfo)), key); err != nil {
		return nil, fmt.Errorf("secretbox: derive key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("secretbox: new cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("secretbox: new GCM: %w", err)
	}
	return &Box{aead: aead}, nil
}

// Seal encrypts plaintext and authenticates associatedData. The result is the
// random 12-byte nonce followed by the sealed ciphertext.
func (b *Box) Seal(plaintext, associatedData []byte) ([]byte, error) {
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("secretbox: random nonce: %w", err)
	}
	return b.aead.Seal(nonce, nonce, plaintext, associatedData), nil
}

// Open decrypts a ciphertext produced by Seal and verifies associatedData.
func (b *Box) Open(ciphertext, associatedData []byte) ([]byte, error) {
	if len(ciphertext) < nonceSize {
		return nil, errors.New("secretbox: ciphertext is too short")
	}
	nonce, sealed := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := b.aead.Open(nil, nonce, sealed, associatedData)
	if err != nil {
		return nil, fmt.Errorf("secretbox: open: %w", err)
	}
	return plaintext, nil
}
