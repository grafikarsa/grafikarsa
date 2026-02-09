package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"grafikarsa/internal/config"
	"grafikarsa/internal/domain"
	"grafikarsa/internal/repository"
	"grafikarsa/internal/utils"
)

// AuthService handles authentication business logic
type AuthService struct {
	authRepo *repository.AuthRepository
	userRepo *repository.UserRepository
	config   *config.Config
}

// NewAuthService creates a new AuthService
func NewAuthService(authRepo *repository.AuthRepository, userRepo *repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{
		authRepo: authRepo,
		userRepo: userRepo,
		config:   cfg,
	}
}

// LoginResult contains the result of a login attempt
type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	User         *domain.User
	SessionID    uuid.UUID
}

// Login authenticates a user and creates a session
func (s *AuthService) Login(ctx context.Context, info domain.LoginInfo) (*LoginResult, error) {
	// Get user by username
	user, err := s.userRepo.GetByUsername(ctx, info.Username)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Check password
	if !utils.CheckPassword(info.Password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// Check if user can login
	if !user.CanLogin() {
		return nil, ErrAccountDisabled
	}

	// Generate token IDs
	tokenID := utils.NewTokenID()
	familyID := utils.NewFamilyID()
	sessionID := uuid.New()

	// Generate tokens
	jwtConfig := utils.JWTConfig{
		AccessSecret:      s.config.JWTAccessSecret,
		RefreshSecret:     s.config.JWTRefreshSecret,
		AccessExpiration:  s.config.JWTAccessExpiration,
		RefreshExpiration: s.config.JWTRefreshExpiration,
	}

	tokenPair, err := utils.GenerateTokenPair(jwtConfig, user.ID, user.Username, string(user.Role), tokenID, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Store refresh token (hashed)
	refreshToken := &domain.RefreshToken{
		ID:        tokenID,
		UserID:    user.ID,
		TokenHash: utils.HashToken(tokenPair.RefreshToken),
		FamilyID:  familyID,
		ExpiresAt: time.Now().Add(s.config.JWTRefreshExpiration),
		CreatedAt: time.Now(),
	}

	if err := s.authRepo.CreateRefreshToken(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Create session
	session := &domain.AuthSession{
		ID:             sessionID,
		UserID:         user.ID,
		RefreshTokenID: tokenID,
		UserAgent:      info.UserAgent,
		IPAddress:      info.IPAddress,
		LastUsedAt:     time.Now(),
		CreatedAt:      time.Now(),
	}

	if err := s.authRepo.CreateAuthSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return &LoginResult{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
		User:         user,
		SessionID:    sessionID,
	}, nil
}

// RefreshResult contains the result of a token refresh
type RefreshResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// RefreshToken validates and rotates a refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenString string) (*RefreshResult, error) {
	// Validate token
	jwtConfig := utils.JWTConfig{
		RefreshSecret: s.config.JWTRefreshSecret,
	}

	claims, err := utils.ValidateRefreshToken(refreshTokenString, jwtConfig.RefreshSecret)
	if err != nil {
		return nil, ErrTokenExpired
	}

	// Get stored token by hash
	tokenHash := utils.HashToken(refreshTokenString)
	storedToken, err := s.authRepo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Token not found - might be already used/rotated
	if storedToken == nil {
		// Check if this token family still exists (possible reuse attack)
		existingToken, _ := s.authRepo.GetRefreshTokenByID(ctx, claims.TokenID)
		if existingToken != nil && existingToken.IsRevoked {
			// Token was already rotated but someone tried to use old token
			// This is a potential reuse attack - revoke entire family
			_ = s.authRepo.RevokeTokenFamily(ctx, claims.FamilyID, domain.RevokedReasonTokenReuse)
			_, _ = s.authRepo.RevokeAllUserSessions(ctx, claims.UserID)
			return nil, ErrTokenReuseDetected
		}
		return nil, ErrTokenExpired
	}

	// Check if token is valid
	if !storedToken.IsValid() {
		// Token reuse detected if revoked but not expired
		if storedToken.IsRevoked && !storedToken.IsExpired() {
			_ = s.authRepo.RevokeTokenFamily(ctx, storedToken.FamilyID, domain.RevokedReasonTokenReuse)
			_, _ = s.authRepo.RevokeAllUserSessions(ctx, storedToken.UserID)
			return nil, ErrTokenReuseDetected
		}
		return nil, ErrTokenExpired
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if user == nil || !user.CanLogin() {
		return nil, ErrAccountDisabled
	}

	// Revoke old token
	if err := s.authRepo.RevokeRefreshToken(ctx, storedToken.ID, domain.RevokedReasonRotated); err != nil {
		return nil, fmt.Errorf("failed to revoke old token: %w", err)
	}

	// Generate new token pair with same family
	newTokenID := utils.NewTokenID()
	newJWTConfig := utils.JWTConfig{
		AccessSecret:      s.config.JWTAccessSecret,
		RefreshSecret:     s.config.JWTRefreshSecret,
		AccessExpiration:  s.config.JWTAccessExpiration,
		RefreshExpiration: s.config.JWTRefreshExpiration,
	}

	tokenPair, err := utils.GenerateTokenPair(newJWTConfig, user.ID, user.Username, string(user.Role), newTokenID, storedToken.FamilyID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Store new refresh token
	newRefreshToken := &domain.RefreshToken{
		ID:        newTokenID,
		UserID:    user.ID,
		TokenHash: utils.HashToken(tokenPair.RefreshToken),
		FamilyID:  storedToken.FamilyID,
		ExpiresAt: time.Now().Add(s.config.JWTRefreshExpiration),
		CreatedAt: time.Now(),
	}

	if err := s.authRepo.CreateRefreshToken(ctx, newRefreshToken); err != nil {
		return nil, fmt.Errorf("failed to store new refresh token: %w", err)
	}

	// Update session with new refresh token
	session, _ := s.authRepo.GetSessionByRefreshTokenID(ctx, storedToken.ID)
	if session != nil {
		_ = s.authRepo.UpdateSessionRefreshToken(ctx, session.ID, newTokenID)
	}

	return &RefreshResult{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

// Logout revokes the current session
func (s *AuthService) Logout(ctx context.Context, refreshTokenString string) error {
	if refreshTokenString == "" {
		return nil // No token to revoke
	}

	tokenHash := utils.HashToken(refreshTokenString)
	storedToken, err := s.authRepo.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	if storedToken == nil {
		return nil // Token not found, consider already logged out
	}

	// Revoke the token
	if err := s.authRepo.RevokeRefreshToken(ctx, storedToken.ID, domain.RevokedReasonLogout); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	// Revoke associated session
	session, _ := s.authRepo.GetSessionByRefreshTokenID(ctx, storedToken.ID)
	if session != nil {
		_ = s.authRepo.RevokeSession(ctx, session.ID)
	}

	return nil
}

// LogoutAll revokes all sessions for a user
func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) (int, error) {
	// Revoke all refresh tokens
	if err := s.authRepo.RevokeAllUserRefreshTokens(ctx, userID, domain.RevokedReasonLogoutAll); err != nil {
		return 0, fmt.Errorf("failed to revoke tokens: %w", err)
	}

	// Revoke all sessions
	count, err := s.authRepo.RevokeAllUserSessions(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to revoke sessions: %w", err)
	}

	return count, nil
}

// GetActiveSessions returns all active sessions for a user
func (s *AuthService) GetActiveSessions(ctx context.Context, userID uuid.UUID, currentRefreshToken string) ([]domain.SessionInfo, error) {
	sessions, err := s.authRepo.GetUserActiveSessions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions: %w", err)
	}

	// Identify current session
	var currentTokenID uuid.UUID
	if currentRefreshToken != "" {
		tokenHash := utils.HashToken(currentRefreshToken)
		storedToken, _ := s.authRepo.GetRefreshTokenByHash(ctx, tokenHash)
		if storedToken != nil {
			currentTokenID = storedToken.ID
		}
	}

	result := make([]domain.SessionInfo, 0, len(sessions))
	for _, session := range sessions {
		session.IsCurrent = session.RefreshTokenID == currentTokenID
		result = append(result, session.ToSessionInfo())
	}

	return result, nil
}

// RevokeSession revokes a specific session
func (s *AuthService) RevokeSession(ctx context.Context, userID, sessionID uuid.UUID) error {
	session, err := s.authRepo.GetSessionByID(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}

	if session == nil {
		return ErrSessionNotFound
	}

	// Ensure user owns the session
	if session.UserID != userID {
		return ErrForbidden
	}

	// Revoke the associated refresh token
	if err := s.authRepo.RevokeRefreshToken(ctx, session.RefreshTokenID, domain.RevokedReasonLogout); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	// Revoke the session
	if err := s.authRepo.RevokeSession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	return nil
}

// ValidateAccessToken validates an access token and returns the claims
func (s *AuthService) ValidateAccessToken(tokenString string) (*utils.AccessTokenClaims, error) {
	return utils.ValidateAccessToken(tokenString, s.config.JWTAccessSecret)
}

// Service errors
var (
	ErrInvalidCredentials = fmt.Errorf("invalid credentials")
	ErrAccountDisabled    = fmt.Errorf("account disabled")
	ErrTokenExpired       = fmt.Errorf("token expired")
	ErrTokenReuseDetected = fmt.Errorf("token reuse detected")
	ErrSessionNotFound    = fmt.Errorf("session not found")
	ErrForbidden          = fmt.Errorf("forbidden")
)
