package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

const (
	AccessIssuer   = "grafikarsa"
	AccessAudience = "grafikarsa-web"
)

type AccessClaims struct {
	jwt.RegisteredClaims
	RolePrimary  string `json:"role_primary"`
	ScopeMajorID string `json:"scope_major_id,omitempty"`
}

func NewAccessToken(userID, rolePrimary, scopeMajorID, kid, secret, ttl string) (token, jti string, err error) {
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", "", err
	}
	jti = base64.RawURLEncoding.EncodeToString(jtiBytes)
	dur, err := time.ParseDuration(ttl)
	if err != nil {
		return "", "", err
	}
	now := time.Now()
	claims := AccessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    AccessIssuer,
			Audience:  jwt.ClaimStrings{AccessAudience},
			Subject:   userID,
			ID:        jti,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(dur)),
		},
		RolePrimary:  rolePrimary,
		ScopeMajorID: scopeMajorID,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	t.Header["kid"] = kid
	signed, err := t.SignedString([]byte(secret))
	if err != nil {
		return "", "", err
	}
	return signed, jti, nil
}

// ParseAccessToken mencoba beberapa secret (rotasi kunci) dan memverifikasi
// kid bila ada. Token lama tetap valid selama secret sebelumnya dipertahankan.
func ParseAccessToken(token string, secrets []string) (*AccessClaims, error) {
	var lastErr error
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		t, err := jwt.ParseWithClaims(token, &AccessClaims{}, func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("algoritma tidak didukung")
			}
			return []byte(secret), nil
		}, jwt.WithIssuer(AccessIssuer), jwt.WithAudience(AccessAudience))
		if err != nil {
			lastErr = err
			continue
		}
		claims, ok := t.Claims.(*AccessClaims)
		if ok && t.Valid {
			return claims, nil
		}
		lastErr = fmt.Errorf("token tidak valid")
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("token tidak valid")
	}
	return nil, lastErr
}

func NewRefreshToken() (plain, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(plain))
	return plain, fmt.Sprintf("%x", sum[:]), nil
}

func HashRefreshToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return fmt.Sprintf("%x", sum[:])
}

func HashPassword(pw string) string {
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	hash := argon2.IDKey([]byte(pw), salt, 3, 64*1024, 2, 32)
	return "$argon2id$v=19$m=65536,t=3,p=2$" +
		base64.RawStdEncoding.EncodeToString(salt) + "$" +
		base64.RawStdEncoding.EncodeToString(hash)
}

func VerifyPassword(pw, encoded string) bool {
	if strings.HasPrefix(encoded, "$argon2id$") {
		parts := strings.Split(encoded, "$")
		if len(parts) != 6 {
			return false
		}
		salt, err1 := base64.RawStdEncoding.DecodeString(parts[4])
		want, err2 := base64.RawStdEncoding.DecodeString(parts[5])
		if err1 != nil || err2 != nil {
			return false
		}
		got := argon2.IDKey([]byte(pw), salt, 3, 64*1024, 2, 32)
		if len(got) != len(want) {
			return false
		}
		var diff byte
		for i := range got {
			diff |= got[i] ^ want[i]
		}
		return diff == 0
	}
	return false
}

func IsArgon2(encoded string) bool { return strings.HasPrefix(encoded, "$argon2id$") }
