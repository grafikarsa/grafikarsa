-- ============================================================================
-- GRAFIKARSA AUTHENTICATION SCHEMA EXTENSION
-- Dual-Token JWT Authentication Support
-- PostgreSQL 16+ Compatible
-- ============================================================================

-- ============================================================================
-- A. AUTHENTICATION TABLES OVERVIEW
-- ============================================================================
/*
NEW TABLES FOR DUAL-TOKEN JWT AUTHENTICATION:

1. refresh_tokens
   - Stores hashed refresh tokens for token rotation
   - Supports token revocation and family tracking
   - Enables "logout all devices" functionality

2. auth_sessions
   - Tracks active user sessions with device info
   - Supports session management and security auditing
   - Links to refresh tokens for session-based revocation

SECURITY FEATURES:
- Refresh tokens are NEVER stored in plaintext (SHA-256 hashed)
- Token rotation invalidates previous tokens
- Token family tracking detects token reuse attacks
- Automatic session creation on login
*/

-- ============================================================================
-- B. TABLE DEFINITIONS
-- ============================================================================

-- ==========================================================================
-- TABLE: refresh_tokens
-- Stores hashed refresh tokens for secure token rotation
-- ==========================================================================
CREATE TABLE refresh_tokens (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                 UUID NOT NULL,
    token_hash              VARCHAR(64) NOT NULL,  -- SHA-256 hash (64 hex chars)
    token_family            UUID NOT NULL,          -- Groups tokens for rotation detection
    issued_at               TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at              TIMESTAMP WITH TIME ZONE NOT NULL,
    revoked_at              TIMESTAMP WITH TIME ZONE,
    revoke_reason           VARCHAR(50),            -- 'logout', 'rotation', 'security', 'password_change'
    replaced_by_token_id    UUID,                   -- Points to the new token after rotation
    session_id              UUID,                   -- Links to auth_sessions
    created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Foreign Keys
    CONSTRAINT refresh_tokens_user_fk FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
    CONSTRAINT refresh_tokens_replaced_by_fk FOREIGN KEY (replaced_by_token_id)
        REFERENCES refresh_tokens(id) ON DELETE SET NULL ON UPDATE CASCADE,

    -- Constraints
    CONSTRAINT refresh_tokens_hash_unique UNIQUE (token_hash),
    CONSTRAINT refresh_tokens_expires_after_issued CHECK (expires_at > issued_at)
);

COMMENT ON TABLE refresh_tokens IS 'Hashed refresh tokens for dual-token JWT authentication';
COMMENT ON COLUMN refresh_tokens.token_hash IS 'SHA-256 hash of the refresh token (never store plaintext)';
COMMENT ON COLUMN refresh_tokens.token_family IS 'UUID grouping tokens in a rotation chain; used to detect reuse attacks';
COMMENT ON COLUMN refresh_tokens.revoke_reason IS 'Reason for revocation: logout, rotation, security, password_change';
COMMENT ON COLUMN refresh_tokens.replaced_by_token_id IS 'ID of new token created during rotation';

-- ==========================================================================
-- TABLE: auth_sessions
-- Tracks active user sessions with device/client information
-- ==========================================================================
CREATE TABLE auth_sessions (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                 UUID NOT NULL,
    user_agent              VARCHAR(500),
    ip_address              INET,
    device_fingerprint      VARCHAR(64),            -- Optional client-side fingerprint
    last_used_at            TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    revoked_at              TIMESTAMP WITH TIME ZONE,
    revoke_reason           VARCHAR(50),
    created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Foreign Keys
    CONSTRAINT auth_sessions_user_fk FOREIGN KEY (user_id)
        REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);

COMMENT ON TABLE auth_sessions IS 'Active user sessions with device/client metadata';
COMMENT ON COLUMN auth_sessions.user_agent IS 'Browser/client user agent string';
COMMENT ON COLUMN auth_sessions.ip_address IS 'Client IP address (IPv4 or IPv6)';
COMMENT ON COLUMN auth_sessions.device_fingerprint IS 'Optional client-generated device fingerprint';

-- Add session foreign key to refresh_tokens now that auth_sessions exists
ALTER TABLE refresh_tokens
    ADD CONSTRAINT refresh_tokens_session_fk FOREIGN KEY (session_id)
        REFERENCES auth_sessions(id) ON DELETE SET NULL ON UPDATE CASCADE;

-- ============================================================================
-- C. INDEXES
-- ============================================================================

-- Refresh Tokens
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_family ON refresh_tokens(token_family);
CREATE INDEX idx_refresh_tokens_session ON refresh_tokens(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX idx_refresh_tokens_active ON refresh_tokens(user_id) 
    WHERE revoked_at IS NULL;

-- Auth Sessions
CREATE INDEX idx_auth_sessions_user ON auth_sessions(user_id);
CREATE INDEX idx_auth_sessions_active ON auth_sessions(user_id) 
    WHERE revoked_at IS NULL;
CREATE INDEX idx_auth_sessions_last_used ON auth_sessions(last_used_at);

-- ============================================================================
-- D. TRIGGERS
-- ============================================================================

-- Auto-update updated_at for refresh_tokens
CREATE TRIGGER trigger_refresh_tokens_updated_at
    BEFORE UPDATE ON refresh_tokens
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Auto-update updated_at for auth_sessions
CREATE TRIGGER trigger_auth_sessions_updated_at
    BEFORE UPDATE ON auth_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- E. HELPER FUNCTIONS
-- ============================================================================

-- ==========================================================================
-- Function: Revoke all refresh tokens for a user
-- Called on logout-all, password change, or security events
-- ==========================================================================
CREATE OR REPLACE FUNCTION revoke_all_user_refresh_tokens(
    p_user_id UUID,
    p_reason VARCHAR(50) DEFAULT 'security'
)
RETURNS INTEGER AS $$
DECLARE
    revoked_count INTEGER;
BEGIN
    UPDATE refresh_tokens
    SET revoked_at = NOW(),
        revoke_reason = p_reason,
        updated_at = NOW()
    WHERE user_id = p_user_id
      AND revoked_at IS NULL;
    
    GET DIAGNOSTICS revoked_count = ROW_COUNT;
    RETURN revoked_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION revoke_all_user_refresh_tokens IS 'Revokes all active refresh tokens for a user';

-- ==========================================================================
-- Function: Revoke token family (on reuse detection)
-- Security measure: if a rotated token is reused, revoke entire family
-- ==========================================================================
CREATE OR REPLACE FUNCTION revoke_token_family(
    p_token_family UUID
)
RETURNS INTEGER AS $$
DECLARE
    revoked_count INTEGER;
BEGIN
    UPDATE refresh_tokens
    SET revoked_at = NOW(),
        revoke_reason = 'security',
        updated_at = NOW()
    WHERE token_family = p_token_family
      AND revoked_at IS NULL;
    
    GET DIAGNOSTICS revoked_count = ROW_COUNT;
    RETURN revoked_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION revoke_token_family IS 'Revokes all tokens in a family when reuse attack is detected';

-- ==========================================================================
-- Function: Revoke all user sessions
-- Called on logout-all or security events
-- ==========================================================================
CREATE OR REPLACE FUNCTION revoke_all_user_sessions(
    p_user_id UUID,
    p_reason VARCHAR(50) DEFAULT 'logout'
)
RETURNS INTEGER AS $$
DECLARE
    revoked_count INTEGER;
BEGIN
    UPDATE auth_sessions
    SET revoked_at = NOW(),
        revoke_reason = p_reason,
        updated_at = NOW()
    WHERE user_id = p_user_id
      AND revoked_at IS NULL;
    
    GET DIAGNOSTICS revoked_count = ROW_COUNT;
    RETURN revoked_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION revoke_all_user_sessions IS 'Revokes all active sessions for a user';

-- ==========================================================================
-- Function: Clean up expired tokens (for scheduled maintenance)
-- ==========================================================================
CREATE OR REPLACE FUNCTION cleanup_expired_tokens(
    p_retention_days INTEGER DEFAULT 30
)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM refresh_tokens
    WHERE expires_at < NOW() - (p_retention_days || ' days')::INTERVAL;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION cleanup_expired_tokens IS 'Deletes tokens expired more than N days ago';

-- ============================================================================
-- END OF AUTHENTICATION SCHEMA EXTENSION
-- ============================================================================