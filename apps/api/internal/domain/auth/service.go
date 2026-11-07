package auth

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/password"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/securetoken"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	defaultSessionIdle     = 7 * 24 * time.Hour
	defaultSessionAbsolute = 30 * 24 * time.Hour
	defaultTouchInterval   = 5 * time.Minute
	defaultCacheTTL        = 60 * time.Second

	loginIdentifierLimit = 5
	loginIPLimit         = 20
	loginWindow          = 15 * time.Minute

	maxUserAgentLength = 512

	sessionCachePrefix = "sess:"
	userSessionsPrefix = "usess:"
)

// Config tunes session lifetimes and how often session activity is cached or
// written back to the database. Zero values use the defaults.
type Config struct {
	SessionIdle     time.Duration
	SessionAbsolute time.Duration
	TouchInterval   time.Duration
	CacheTTL        time.Duration
}

func (c Config) withDefaults() Config {
	if c.SessionIdle <= 0 {
		c.SessionIdle = defaultSessionIdle
	}
	if c.SessionAbsolute <= 0 {
		c.SessionAbsolute = defaultSessionAbsolute
	}
	if c.TouchInterval <= 0 {
		c.TouchInterval = defaultTouchInterval
	}
	if c.CacheTTL <= 0 {
		c.CacheTTL = defaultCacheTTL
	}
	return c
}

// Service implements login, session validation, and session management.
type Service struct {
	store   *store.Store
	rdb     *redis.Client
	hasher  *password.Hasher
	limiter *ratelimit.Limiter
	cfg     Config
	logger  *slog.Logger
	now     func() time.Time
	// dummyHash is verified for unknown identifiers so login timing does not
	// reveal whether an account exists.
	dummyHash string
}

// NewService builds an auth Service. The dummy password hash is computed once
// here because it is used on every login for an unknown identifier.
func NewService(
	st *store.Store,
	rdb *redis.Client,
	hasher *password.Hasher,
	limiter *ratelimit.Limiter,
	cfg Config,
	logger *slog.Logger,
) (*Service, error) {
	switch {
	case st == nil:
		return nil, errors.New("auth: store is required")
	case rdb == nil:
		return nil, errors.New("auth: valkey client is required")
	case hasher == nil:
		return nil, errors.New("auth: password hasher is required")
	case limiter == nil:
		return nil, errors.New("auth: rate limiter is required")
	}
	if logger == nil {
		logger = slog.Default()
	}

	dummyPassword, _, err := securetoken.SessionToken()
	if err != nil {
		return nil, fmt.Errorf("auth: generate dummy password: %w", err)
	}
	dummyHash, err := hasher.Hash(context.Background(), dummyPassword)
	if err != nil {
		return nil, fmt.Errorf("auth: hash dummy password: %w", err)
	}

	return &Service{
		store:     st,
		rdb:       rdb,
		hasher:    hasher,
		limiter:   limiter,
		cfg:       cfg.withDefaults(),
		logger:    logger,
		now:       time.Now,
		dummyHash: dummyHash,
	}, nil
}

// LoginInput contains the credentials submitted to Login.
type LoginInput struct {
	Identifier string
	Password   string
	IP         string
	UserAgent  string
}

// LoginResult is a successful login: the opaque token to put in the session
// cookie and the authenticated principal.
type LoginResult struct {
	Token     string
	Principal *Principal
}

// Device describes one active session for the session list.
type Device struct {
	ID         uuid.UUID
	Current    bool
	CreatedAt  time.Time
	LastSeenAt time.Time
	UserAgent  *string
	IPPrefix   *string
}

// Login verifies credentials, applies login rate limits, creates a session,
// and returns its principal.
func (s *Service) Login(ctx context.Context, in LoginInput) (*LoginResult, error) {
	identifier := strings.TrimSpace(in.Identifier)
	if identifier == "" || in.Password == "" {
		return nil, apperr.InvalidCredentials()
	}

	now := s.now().UTC()

	if err := s.allowLogin(ctx, identifier, in.IP); err != nil {
		return nil, err
	}

	user, err := s.store.GetUserForLogin(ctx, identifier)
	if errors.Is(err, pgx.ErrNoRows) {
		s.verifyDummy(ctx, in.Password)
		return nil, apperr.InvalidCredentials()
	}
	if err != nil {
		return nil, fmt.Errorf("auth: get user for login: %w", err)
	}

	ok, needsRehash, err := s.hasher.Verify(ctx, user.PasswordHash, in.Password)
	if err != nil {
		if errors.Is(err, password.ErrUnknownFormat) || errors.Is(err, password.ErrMalformedHash) {
			s.logger.Warn("auth: stored password hash is not usable", "user_id", user.ID)
			return nil, apperr.InvalidCredentials()
		}
		return nil, fmt.Errorf("auth: verify password: %w", err)
	}
	if !ok {
		return nil, apperr.InvalidCredentials()
	}

	if user.Status == "transferred" || user.Status == "inactive" || user.DeactivatedAt != nil {
		return nil, &apperr.Error{
			Status: http.StatusForbidden,
			Code:   "ACCOUNT_INACTIVE",
			Title:  "Akun tidak aktif. Hubungi admin sekolah.",
		}
	}

	if (user.PasswordKind == "initial" || user.PasswordKind == "temporary") &&
		user.PasswordExpiresAt != nil && user.PasswordExpiresAt.Before(now) {
		return nil, &apperr.Error{
			Status: http.StatusUnauthorized,
			Code:   "PASSWORD_EXPIRED",
			Title:  "Sandi sudah kedaluwarsa. Minta sandi baru ke admin sekolah.",
		}
	}

	if needsRehash {
		s.rehashPassword(ctx, user.ID, in.Password)
	}

	token, tokenHash, err := securetoken.SessionToken()
	if err != nil {
		return nil, fmt.Errorf("auth: generate session token: %w", err)
	}

	session, err := s.store.CreateSession(ctx, store.CreateSessionParams{
		UserID:        user.ID,
		TokenHash:     tokenHash,
		Ip:            parseIP(in.IP),
		UserAgent:     truncateUserAgent(in.UserAgent),
		IdleExpiresAt: now.Add(s.cfg.SessionIdle),
		ExpiresAt:     now.Add(s.cfg.SessionAbsolute),
	})
	if err != nil {
		return nil, fmt.Errorf("auth: create session: %w", err)
	}

	if err := s.store.SetLastLogin(ctx, user.ID); err != nil {
		s.logger.Warn("auth: set last login", "user_id", user.ID, "error", err)
	}
	if err := s.limiter.Reset(ctx, loginIdentifierKey(identifier)); err != nil {
		s.logger.Warn("auth: reset login rate limit", "error", err)
	}

	principal, err := s.loadPrincipal(ctx, session)
	if err != nil {
		return nil, err
	}

	return &LoginResult{Token: token, Principal: principal}, nil
}

func (s *Service) allowLogin(ctx context.Context, identifier, ip string) error {
	result, err := s.limiter.Allow(ctx, loginIdentifierKey(identifier), ratelimit.Rule{
		Limit:  loginIdentifierLimit,
		Window: loginWindow,
	})
	if err != nil {
		s.logger.Warn("auth: identifier rate limiter failed", "error", err)
		return apperr.Unavailable("rate limiter")
	}
	if !result.Allowed {
		return apperr.RateLimited()
	}

	if ip == "" {
		return nil
	}

	result, err = s.limiter.Allow(ctx, "login:ip:"+ip, ratelimit.Rule{
		Limit:  loginIPLimit,
		Window: loginWindow,
	})
	if err != nil {
		s.logger.Warn("auth: ip rate limiter failed", "error", err)
		return apperr.Unavailable("rate limiter")
	}
	if !result.Allowed {
		return apperr.RateLimited()
	}

	return nil
}

func (s *Service) verifyDummy(ctx context.Context, plain string) {
	if _, _, err := s.hasher.Verify(ctx, s.dummyHash, plain); err != nil {
		s.logger.Warn("auth: dummy password verification failed", "error", err)
	}
}

func (s *Service) rehashPassword(ctx context.Context, userID uuid.UUID, plain string) {
	hash, err := s.hasher.Hash(ctx, plain)
	if err != nil {
		s.logger.Warn("auth: rehash password", "user_id", userID, "error", err)
		return
	}
	if err := s.store.UpdatePasswordHash(ctx, store.UpdatePasswordHashParams{
		ID:           userID,
		PasswordHash: hash,
	}); err != nil {
		s.logger.Warn("auth: store rehashed password", "user_id", userID, "error", err)
	}
}

// Authenticate resolves a session token to a Principal, using the Valkey
// cache when possible and falling back to PostgreSQL when it is not.
func (s *Service) Authenticate(ctx context.Context, token string) (*Principal, error) {
	if token == "" {
		return nil, apperr.Unauthenticated()
	}

	now := s.now().UTC()
	tokenHash := securetoken.HashSessionToken(token)
	cacheKey := sessionCacheKey(tokenHash)

	if cached, ok := s.getCachedPrincipal(ctx, cacheKey); ok {
		if now.Before(cached.ExpiresAt) && now.Before(cached.IdleExpiresAt) {
			if now.Sub(cached.LastSeenAt) >= s.cfg.TouchInterval {
				idleExpiresAt := now.Add(s.cfg.SessionIdle)
				if err := s.store.TouchSession(ctx, store.TouchSessionParams{
					ID:            cached.SessionID,
					IdleExpiresAt: idleExpiresAt,
				}); err != nil {
					s.logger.Warn("auth: touch session", "session_id", cached.SessionID, "error", err)
				} else {
					cached.LastSeenAt = now
					cached.IdleExpiresAt = idleExpiresAt
					s.cachePrincipal(ctx, cacheKey, &cached.Principal, cached.IdleExpiresAt, cached.LastSeenAt)
				}
			}
			principal := cached.Principal
			return &principal, nil
		}
		// The cached copy is past its idle or absolute expiry. Fall through to
		// the database so the session can be revoked.
	}

	session, err := s.store.GetSessionByTokenHash(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.Unauthenticated()
	}
	if err != nil {
		return nil, fmt.Errorf("auth: get session: %w", err)
	}

	if !now.Before(session.ExpiresAt) || !now.Before(session.IdleExpiresAt) {
		if err := s.store.RevokeSession(ctx, store.RevokeSessionParams{
			ID:     session.ID,
			Reason: "expired",
		}); err != nil {
			s.logger.Warn("auth: revoke expired session", "session_id", session.ID, "error", err)
		}
		s.deleteSessionCache(ctx, tokenHash)
		return nil, apperr.Unauthenticated()
	}

	principal, err := s.loadPrincipal(ctx, session)
	if err != nil {
		return nil, err
	}

	lastSeenAt := session.LastSeenAt
	idleExpiresAt := session.IdleExpiresAt
	if now.Sub(lastSeenAt) >= s.cfg.TouchInterval {
		newIdleExpiresAt := now.Add(s.cfg.SessionIdle)
		if err := s.store.TouchSession(ctx, store.TouchSessionParams{
			ID:            session.ID,
			IdleExpiresAt: newIdleExpiresAt,
		}); err != nil {
			s.logger.Warn("auth: touch session", "session_id", session.ID, "error", err)
		} else {
			lastSeenAt = now
			idleExpiresAt = newIdleExpiresAt
		}
	}

	s.cachePrincipal(ctx, cacheKey, principal, idleExpiresAt, lastSeenAt)

	return principal, nil
}

func (s *Service) loadPrincipal(ctx context.Context, session store.Session) (*Principal, error) {
	user, err := s.store.GetUserByID(ctx, session.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		s.revokeSessionQuietly(ctx, session.ID, "deactivated")
		return nil, apperr.Unauthenticated()
	}
	if err != nil {
		return nil, fmt.Errorf("auth: get user: %w", err)
	}

	if user.Status == "transferred" || user.Status == "inactive" || user.DeactivatedAt != nil {
		s.revokeSessionQuietly(ctx, session.ID, "deactivated")
		return nil, apperr.Unauthenticated()
	}

	explicitRows, err := s.store.ListExplicitGrants(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("auth: list explicit grants: %w", err)
	}

	memberPermissions, err := s.store.ListRolePermissionsBySlug(ctx, "member")
	if err != nil {
		return nil, fmt.Errorf("auth: list member permissions: %w", err)
	}
	implicit := append([]string(nil), memberPermissions...)
	if user.AccountType == "teacher" {
		teacherPermissions, err := s.store.ListRolePermissionsBySlug(ctx, "teacher")
		if err != nil {
			return nil, fmt.Errorf("auth: list teacher permissions: %w", err)
		}
		implicit = append(implicit, teacherPermissions...)
	}

	explicit := make([]grantRow, 0, len(explicitRows))
	for _, row := range explicitRows {
		explicit = append(explicit, grantRow{
			Permission: row.PermissionKey,
			MajorID:    row.MajorID,
		})
	}

	grants := mergeGrants(explicit, implicit)
	if user.ContentBlockedAt != nil {
		grants = removeGrant(grants, "content.create")
	}

	return &Principal{
		SessionID:   session.ID,
		UserID:      user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		AccountType: user.AccountType,
		Status:      user.Status,
		NextStep:    ComputeNextStep(user.ActivatedAt, user.PasswordKind),
		IsMigrated:  user.LegacyV1ID != nil,
		Grants:      grants,
		ExpiresAt:   session.ExpiresAt,
	}, nil
}

// Logout revokes the principal's current session and drops its cache entry.
func (s *Service) Logout(ctx context.Context, p *Principal) error {
	if p == nil {
		return apperr.Unauthenticated()
	}

	tokenHash, err := s.sessionTokenHash(ctx, p.UserID, p.SessionID)
	if err != nil {
		return err
	}

	if err := s.store.RevokeSession(ctx, store.RevokeSessionParams{
		ID:     p.SessionID,
		Reason: "logout",
	}); err != nil {
		return fmt.Errorf("auth: revoke session: %w", err)
	}

	if tokenHash != nil {
		s.deleteSessionCache(ctx, tokenHash)
	}
	return nil
}

// ListSessions returns the principal's active sessions, newest activity first.
func (s *Service) ListSessions(ctx context.Context, p *Principal) ([]Device, error) {
	if p == nil {
		return nil, apperr.Unauthenticated()
	}

	rows, err := s.store.ListActiveSessions(ctx, p.UserID)
	if err != nil {
		return nil, fmt.Errorf("auth: list sessions: %w", err)
	}

	devices := make([]Device, 0, len(rows))
	for _, row := range rows {
		devices = append(devices, Device{
			ID:         row.ID,
			Current:    row.ID == p.SessionID,
			CreatedAt:  row.CreatedAt,
			LastSeenAt: row.LastSeenAt,
			UserAgent:  row.UserAgent,
			IPPrefix:   ipPrefix(row.Ip),
		})
	}
	return devices, nil
}

// RevokeSession revokes one of the principal's sessions. Sessions owned by
// another user are reported as not found so their existence does not leak.
func (s *Service) RevokeSession(ctx context.Context, p *Principal, sessionID uuid.UUID) error {
	if p == nil {
		return apperr.Unauthenticated()
	}

	tokenHash, err := s.sessionTokenHash(ctx, p.UserID, sessionID)
	if err != nil {
		return err
	}
	if tokenHash == nil {
		return apperr.NotFound()
	}

	if err := s.store.RevokeSession(ctx, store.RevokeSessionParams{
		ID:     sessionID,
		Reason: "logout",
	}); err != nil {
		return fmt.Errorf("auth: revoke session: %w", err)
	}

	s.deleteSessionCache(ctx, tokenHash)
	return nil
}

// RevokeOtherSessions revokes every active session of the principal except
// the current one.
func (s *Service) RevokeOtherSessions(ctx context.Context, p *Principal) error {
	if p == nil {
		return apperr.Unauthenticated()
	}

	tokenHashes, err := s.store.RevokeUserSessions(ctx, store.RevokeUserSessionsParams{
		UserID:   p.UserID,
		ExceptID: p.SessionID,
		Reason:   "logout_others",
	})
	if err != nil {
		return fmt.Errorf("auth: revoke other sessions: %w", err)
	}

	for _, tokenHash := range tokenHashes {
		s.deleteSessionCache(ctx, tokenHash)
	}
	return nil
}

// InvalidateUser drops every cached principal of userID. Call it after
// changing roles, account status, or content blocking.
func (s *Service) InvalidateUser(ctx context.Context, userID uuid.UUID) error {
	setKey := userSessionsKey(userID)

	keys, err := s.rdb.SMembers(ctx, setKey).Result()
	if err != nil {
		return fmt.Errorf("auth: list cached principals: %w", err)
	}

	toDelete := append(keys, setKey)
	if err := s.rdb.Del(ctx, toDelete...).Err(); err != nil {
		return fmt.Errorf("auth: delete cached principals: %w", err)
	}
	return nil
}

// sessionTokenHash finds the token hash of an active session owned by userID.
// It returns nil when the session does not exist or is no longer active.
func (s *Service) sessionTokenHash(ctx context.Context, userID, sessionID uuid.UUID) ([]byte, error) {
	rows, err := s.store.ListActiveSessions(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("auth: list sessions: %w", err)
	}
	for _, row := range rows {
		if row.ID == sessionID {
			return row.TokenHash, nil
		}
	}
	return nil, nil
}

type cachedPrincipal struct {
	Principal
	IdleExpiresAt time.Time `json:"idleExpiresAt"`
	LastSeenAt    time.Time `json:"lastSeenAt"`
}

func (s *Service) getCachedPrincipal(ctx context.Context, key string) (cachedPrincipal, bool) {
	raw, err := s.rdb.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return cachedPrincipal{}, false
	}
	if err != nil {
		s.logger.Warn("auth: read principal cache", "error", err)
		return cachedPrincipal{}, false
	}

	var cached cachedPrincipal
	if err := json.Unmarshal(raw, &cached); err != nil {
		s.logger.Warn("auth: decode principal cache", "error", err)
		return cachedPrincipal{}, false
	}
	return cached, true
}

func (s *Service) cachePrincipal(ctx context.Context, key string, p *Principal, idleExpiresAt, lastSeenAt time.Time) {
	data, err := json.Marshal(cachedPrincipal{
		Principal:     *p,
		IdleExpiresAt: idleExpiresAt,
		LastSeenAt:    lastSeenAt,
	})
	if err != nil {
		s.logger.Warn("auth: encode principal cache", "error", err)
		return
	}

	setKey := userSessionsKey(p.UserID)
	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, key, data, s.cfg.CacheTTL)
	pipe.SAdd(ctx, setKey, key)
	pipe.Expire(ctx, setKey, s.cfg.CacheTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		s.logger.Warn("auth: write principal cache", "error", err)
	}
}

func (s *Service) deleteSessionCache(ctx context.Context, tokenHash []byte) {
	if err := s.rdb.Del(ctx, sessionCacheKey(tokenHash)).Err(); err != nil {
		s.logger.Warn("auth: delete principal cache", "error", err)
	}
}

func (s *Service) revokeSessionQuietly(ctx context.Context, sessionID uuid.UUID, reason string) {
	if err := s.store.RevokeSession(ctx, store.RevokeSessionParams{
		ID:     sessionID,
		Reason: reason,
	}); err != nil {
		s.logger.Warn("auth: revoke session", "session_id", sessionID, "error", err)
	}
}

func removeGrant(grants []Grant, permission string) []Grant {
	out := grants[:0]
	for _, grant := range grants {
		if grant.Permission != permission {
			out = append(out, grant)
		}
	}
	return out
}

func loginIdentifierKey(identifier string) string {
	return "login:id:" + strings.ToLower(identifier)
}

func sessionCacheKey(tokenHash []byte) string {
	return sessionCachePrefix + hex.EncodeToString(tokenHash)
}

func userSessionsKey(userID uuid.UUID) string {
	return userSessionsPrefix + userID.String()
}

func parseIP(raw string) netip.Addr {
	addr, err := netip.ParseAddr(strings.TrimSpace(raw))
	if err != nil {
		return netip.Addr{}
	}
	return addr.Unmap()
}

func truncateUserAgent(userAgent string) string {
	runes := []rune(userAgent)
	if len(runes) <= maxUserAgentLength {
		return userAgent
	}
	return string(runes[:maxUserAgentLength])
}

// ipPrefix masks an address for display: IPv4 keeps the first three octets
// ("a.b.c.x") and IPv6 keeps the first four groups.
func ipPrefix(addr *netip.Addr) *string {
	if addr == nil || !addr.IsValid() {
		return nil
	}

	masked := addr.Unmap()
	var prefix string
	if masked.Is4() {
		b := masked.As4()
		prefix = fmt.Sprintf("%d.%d.%d.x", b[0], b[1], b[2])
	} else {
		b := masked.As16()
		prefix = fmt.Sprintf("%x:%x:%x:%x::",
			binary.BigEndian.Uint16(b[0:2]),
			binary.BigEndian.Uint16(b[2:4]),
			binary.BigEndian.Uint16(b[4:6]),
			binary.BigEndian.Uint16(b[6:8]),
		)
	}
	return &prefix
}
