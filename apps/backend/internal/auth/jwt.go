package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/config"
)

type JWTService struct {
	accessSecret  string
	refreshSecret string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

type AccessTokenClaims struct {
	Sub  string `json:"sub"`
	Role string `json:"role"`
	JTI  string `json:"jti"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func NewJWTService(cfg *config.Config) *JWTService {
	return &JWTService{
		accessSecret:  cfg.JWT.AccessSecret,
		refreshSecret: cfg.JWT.RefreshSecret,
		accessExpiry:  cfg.JWT.AccessExpiry,
		refreshExpiry: cfg.JWT.RefreshExpiry,
	}
}

func (j *JWTService) GenerateAccessToken(userID uuid.UUID, role string) (string, string, error) {
	jti := uuid.New().String()
	now := time.Now()
	expiresAt := now.Add(j.accessExpiry)

	claims := AccessTokenClaims{
		Sub:  userID.String(),
		Role: role,
		JTI:  jti,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "grafikarsa",
			Audience:  jwt.ClaimStrings{"grafikarsa-api"},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(j.accessSecret))
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return signedToken, jti, nil
}

func (j *JWTService) GenerateRefreshToken() (string, string, time.Time) {
	token := uuid.New().String() + uuid.New().String()
	hash := HashToken(token)
	expiresAt := time.Now().Add(j.refreshExpiry)
	return token, hash, expiresAt
}

func (j *JWTService) ValidateAccessToken(tokenString string) (*AccessTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.accessSecret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*AccessTokenClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	if claims.Issuer != "grafikarsa" {
		return nil, fmt.Errorf("invalid issuer")
	}

	return claims, nil
}

func (j *JWTService) GetAccessExpiry() time.Duration {
	return j.accessExpiry
}

func (j *JWTService) GetRefreshExpiry() time.Duration {
	return j.refreshExpiry
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
