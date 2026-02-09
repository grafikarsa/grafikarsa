package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"grafikarsa/internal/domain"
)

// AuthRepository handles authentication data access
type AuthRepository struct {
	db *pgxpool.Pool
}

// NewAuthRepository creates a new AuthRepository
func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{db: db}
}

// CreateRefreshToken stores a new refresh token
func (r *AuthRepository) CreateRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, family_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Exec(ctx, query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.FamilyID,
		token.ExpiresAt,
		token.CreatedAt,
	)

	return err
}

// GetRefreshTokenByHash retrieves a refresh token by its hash
func (r *AuthRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, family_id, is_revoked, revoked_at, revoked_reason, expires_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	var token domain.RefreshToken
	err := r.db.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.FamilyID,
		&token.IsRevoked,
		&token.RevokedAt,
		&token.RevokedReason,
		&token.ExpiresAt,
		&token.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &token, nil
}

// GetRefreshTokenByID retrieves a refresh token by ID
func (r *AuthRepository) GetRefreshTokenByID(ctx context.Context, id uuid.UUID) (*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, family_id, is_revoked, revoked_at, revoked_reason, expires_at, created_at
		FROM refresh_tokens
		WHERE id = $1
	`

	var token domain.RefreshToken
	err := r.db.QueryRow(ctx, query, id).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.FamilyID,
		&token.IsRevoked,
		&token.RevokedAt,
		&token.RevokedReason,
		&token.ExpiresAt,
		&token.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &token, nil
}

// RevokeRefreshToken revokes a specific refresh token
func (r *AuthRepository) RevokeRefreshToken(ctx context.Context, id uuid.UUID, reason string) error {
	query := `
		UPDATE refresh_tokens
		SET is_revoked = true, revoked_at = $2, revoked_reason = $3
		WHERE id = $1 AND is_revoked = false
	`

	_, err := r.db.Exec(ctx, query, id, time.Now(), reason)
	return err
}

// RevokeTokenFamily revokes all tokens in a family (for reuse detection)
func (r *AuthRepository) RevokeTokenFamily(ctx context.Context, familyID uuid.UUID, reason string) error {
	query := `
		UPDATE refresh_tokens
		SET is_revoked = true, revoked_at = $2, revoked_reason = $3
		WHERE family_id = $1 AND is_revoked = false
	`

	_, err := r.db.Exec(ctx, query, familyID, time.Now(), reason)
	return err
}

// RevokeAllUserRefreshTokens revokes all refresh tokens for a user
func (r *AuthRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID, reason string) error {
	query := `
		UPDATE refresh_tokens
		SET is_revoked = true, revoked_at = $2, revoked_reason = $3
		WHERE user_id = $1 AND is_revoked = false
	`

	_, err := r.db.Exec(ctx, query, userID, time.Now(), reason)
	return err
}

// CreateAuthSession creates a new authentication session
func (r *AuthRepository) CreateAuthSession(ctx context.Context, session *domain.AuthSession) error {
	query := `
		INSERT INTO auth_sessions (id, user_id, refresh_token_id, user_agent, ip_address, last_used_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(ctx, query,
		session.ID,
		session.UserID,
		session.RefreshTokenID,
		session.UserAgent,
		session.IPAddress,
		session.LastUsedAt,
		session.CreatedAt,
	)

	return err
}

// GetSessionByRefreshTokenID gets a session by refresh token ID
func (r *AuthRepository) GetSessionByRefreshTokenID(ctx context.Context, refreshTokenID uuid.UUID) (*domain.AuthSession, error) {
	query := `
		SELECT id, user_id, refresh_token_id, user_agent, ip_address, is_revoked, revoked_at, last_used_at, created_at
		FROM auth_sessions
		WHERE refresh_token_id = $1
	`

	var session domain.AuthSession
	err := r.db.QueryRow(ctx, query, refreshTokenID).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshTokenID,
		&session.UserAgent,
		&session.IPAddress,
		&session.IsRevoked,
		&session.RevokedAt,
		&session.LastUsedAt,
		&session.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &session, nil
}

// GetSessionByID gets a session by ID
func (r *AuthRepository) GetSessionByID(ctx context.Context, id uuid.UUID) (*domain.AuthSession, error) {
	query := `
		SELECT id, user_id, refresh_token_id, user_agent, ip_address, is_revoked, revoked_at, last_used_at, created_at
		FROM auth_sessions
		WHERE id = $1
	`

	var session domain.AuthSession
	err := r.db.QueryRow(ctx, query, id).Scan(
		&session.ID,
		&session.UserID,
		&session.RefreshTokenID,
		&session.UserAgent,
		&session.IPAddress,
		&session.IsRevoked,
		&session.RevokedAt,
		&session.LastUsedAt,
		&session.CreatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &session, nil
}

// GetUserActiveSessions gets all active sessions for a user
func (r *AuthRepository) GetUserActiveSessions(ctx context.Context, userID uuid.UUID) ([]domain.AuthSession, error) {
	query := `
		SELECT s.id, s.user_id, s.refresh_token_id, s.user_agent, s.ip_address, 
		       s.is_revoked, s.revoked_at, s.last_used_at, s.created_at
		FROM auth_sessions s
		INNER JOIN refresh_tokens rt ON s.refresh_token_id = rt.id
		WHERE s.user_id = $1 
		  AND s.is_revoked = false 
		  AND rt.is_revoked = false
		  AND rt.expires_at > NOW()
		ORDER BY s.last_used_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []domain.AuthSession
	for rows.Next() {
		var session domain.AuthSession
		if err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.RefreshTokenID,
			&session.UserAgent,
			&session.IPAddress,
			&session.IsRevoked,
			&session.RevokedAt,
			&session.LastUsedAt,
			&session.CreatedAt,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}

// RevokeSession revokes a specific session
func (r *AuthRepository) RevokeSession(ctx context.Context, sessionID uuid.UUID) error {
	query := `
		UPDATE auth_sessions
		SET is_revoked = true, revoked_at = $2
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, sessionID, time.Now())
	return err
}

// RevokeAllUserSessions revokes all sessions for a user
func (r *AuthRepository) RevokeAllUserSessions(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `
		UPDATE auth_sessions
		SET is_revoked = true, revoked_at = $2
		WHERE user_id = $1 AND is_revoked = false
	`

	result, err := r.db.Exec(ctx, query, userID, time.Now())
	if err != nil {
		return 0, err
	}

	return int(result.RowsAffected()), nil
}

// UpdateSessionLastUsed updates the last_used_at timestamp for a session
func (r *AuthRepository) UpdateSessionLastUsed(ctx context.Context, sessionID uuid.UUID) error {
	query := `
		UPDATE auth_sessions
		SET last_used_at = $2
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, sessionID, time.Now())
	return err
}

// UpdateSessionRefreshToken updates the refresh token ID and last used time
func (r *AuthRepository) UpdateSessionRefreshToken(ctx context.Context, sessionID, newRefreshTokenID uuid.UUID) error {
	query := `
		UPDATE auth_sessions
		SET refresh_token_id = $2, last_used_at = $3
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, sessionID, newRefreshTokenID, time.Now())
	return err
}

// CountActiveSessionsForUser counts active sessions for a user
func (r *AuthRepository) CountActiveSessionsForUser(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM auth_sessions s
		INNER JOIN refresh_tokens rt ON s.refresh_token_id = rt.id
		WHERE s.user_id = $1 
		  AND s.is_revoked = false 
		  AND rt.is_revoked = false
		  AND rt.expires_at > NOW()
	`

	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

// CleanupExpiredTokens removes expired tokens (called periodically)
func (r *AuthRepository) CleanupExpiredTokens(ctx context.Context) error {
	query := `
		DELETE FROM refresh_tokens
		WHERE expires_at < NOW() - INTERVAL '7 days'
	`

	_, err := r.db.Exec(ctx, query)
	return err
}
