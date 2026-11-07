// Package password hashes and verifies user passwords with Argon2id while
// remaining compatible with bcrypt hashes from the legacy system.
package password

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// Errors returned by Verify for hashes that cannot be processed.
var (
	ErrUnknownFormat = errors.New("password: unknown hash format")
	ErrMalformedHash = errors.New("password: malformed hash")
)

// Params configures Argon2id. Memory is expressed in KiB.
type Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultParams is the OWASP 2024 minimum profile, chosen to keep RAM low on a
// small VPS.
var DefaultParams = Params{
	Memory:      19 * 1024,
	Iterations:  2,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

const argon2idVersion = "v=19"

// Upper bounds accepted when parsing stored hashes.
const (
	maxMemoryKiB   = 256 * 1024
	maxIterations  = 10
	maxParallelism = 8
)

// Hasher hashes and verifies passwords. A semaphore caps the number of
// concurrent Argon2 computations so memory use stays predictable.
type Hasher struct {
	params Params
	sem    chan struct{}
}

// NewHasher creates a Hasher. A maxConcurrent value of zero or less uses
// runtime.NumCPU().
func NewHasher(p Params, maxConcurrent int) *Hasher {
	if maxConcurrent <= 0 {
		maxConcurrent = runtime.NumCPU()
	}

	return &Hasher{
		params: p,
		sem:    make(chan struct{}, maxConcurrent),
	}
}

// Hash derives a PHC-formatted Argon2id hash with a random salt.
func (h *Hasher) Hash(ctx context.Context, plain string) (string, error) {
	salt := make([]byte, h.params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("password: generate salt: %w", err)
	}

	key, err := h.derive(ctx, plain, salt, h.params, h.params.KeyLength)
	if err != nil {
		return "", err
	}

	return encodeArgon2id(h.params, salt, key), nil
}

// Verify checks plain against encoded. It reports whether the password
// matched and whether the stored hash should be replaced with one produced by
// this Hasher's parameters. A wrong password is not an error.
func (h *Hasher) Verify(ctx context.Context, encoded, plain string) (ok bool, needsRehash bool, err error) {
	switch {
	case strings.HasPrefix(encoded, "$argon2id$"):
		params, salt, key, parseErr := parseArgon2id(encoded)
		if parseErr != nil {
			return false, false, parseErr
		}

		computed, deriveErr := h.derive(ctx, plain, salt, params, uint32(len(key)))
		if deriveErr != nil {
			return false, false, deriveErr
		}

		if subtle.ConstantTimeCompare(computed, key) != 1 {
			return false, false, nil
		}

		return true, params != h.params, nil
	case strings.HasPrefix(encoded, "$2a$"), strings.HasPrefix(encoded, "$2b$"), strings.HasPrefix(encoded, "$2y$"):
		if ctxErr := ctx.Err(); ctxErr != nil {
			return false, false, ctxErr
		}

		if compareErr := bcrypt.CompareHashAndPassword([]byte(encoded), []byte(plain)); compareErr != nil {
			if errors.Is(compareErr, bcrypt.ErrMismatchedHashAndPassword) {
				return false, false, nil
			}

			return false, false, ErrMalformedHash
		}

		// bcrypt hashes always need to be replaced with Argon2id.
		return true, true, nil
	default:
		return false, false, ErrUnknownFormat
	}
}

func (h *Hasher) derive(ctx context.Context, plain string, salt []byte, p Params, keyLength uint32) ([]byte, error) {
	if err := h.acquire(ctx); err != nil {
		return nil, err
	}
	defer h.release()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return argon2.IDKey([]byte(plain), salt, p.Iterations, p.Memory, p.Parallelism, keyLength), nil
}

func (h *Hasher) acquire(ctx context.Context) error {
	select {
	case h.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *Hasher) release() {
	<-h.sem
}

func encodeArgon2id(p Params, salt, key []byte) string {
	return fmt.Sprintf(
		"$argon2id$%s$m=%d,t=%d,p=%d$%s$%s",
		argon2idVersion,
		p.Memory,
		p.Iterations,
		p.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
}

func parseArgon2id(encoded string) (Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != argon2idVersion {
		return Params{}, nil, nil, ErrMalformedHash
	}

	params, err := parseArgon2Params(parts[3])
	if err != nil {
		return Params{}, nil, nil, err
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Params{}, nil, nil, ErrMalformedHash
	}

	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return Params{}, nil, nil, ErrMalformedHash
	}

	if len(salt) < 8 || len(key) < 16 {
		return Params{}, nil, nil, ErrMalformedHash
	}

	params.SaltLength = uint32(len(salt))
	params.KeyLength = uint32(len(key))

	return params, salt, key, nil
}

func parseArgon2Params(encoded string) (Params, error) {
	fields := strings.Split(encoded, ",")
	if len(fields) != 3 {
		return Params{}, ErrMalformedHash
	}

	var (
		params         Params
		hasMemory      bool
		hasIterations  bool
		hasParallelism bool
	)

	for _, field := range fields {
		name, value, found := strings.Cut(field, "=")
		if !found {
			return Params{}, ErrMalformedHash
		}

		switch name {
		case "m":
			parsed, err := strconv.ParseUint(value, 10, 32)
			if err != nil || hasMemory {
				return Params{}, ErrMalformedHash
			}
			params.Memory = uint32(parsed)
			hasMemory = true
		case "t":
			parsed, err := strconv.ParseUint(value, 10, 32)
			if err != nil || hasIterations {
				return Params{}, ErrMalformedHash
			}
			params.Iterations = uint32(parsed)
			hasIterations = true
		case "p":
			parsed, err := strconv.ParseUint(value, 10, 8)
			if err != nil || hasParallelism {
				return Params{}, ErrMalformedHash
			}
			params.Parallelism = uint8(parsed)
			hasParallelism = true
		default:
			return Params{}, ErrMalformedHash
		}
	}

	if !hasMemory || !hasIterations || !hasParallelism {
		return Params{}, ErrMalformedHash
	}

	if params.Memory == 0 || params.Iterations == 0 || params.Parallelism == 0 {
		return Params{}, ErrMalformedHash
	}
	// Refuse absurd parameters so a tampered hash cannot exhaust memory or CPU.
	if params.Memory > maxMemoryKiB || params.Iterations > maxIterations || params.Parallelism > maxParallelism {
		return Params{}, ErrMalformedHash
	}

	return params, nil
}
