package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"grafikarsa/internal/domain"
	"grafikarsa/internal/service"
	"grafikarsa/internal/utils"
)

// AuthMiddleware provides authentication middleware
type AuthMiddleware struct {
	authService *service.AuthService
}

// NewAuthMiddleware creates a new AuthMiddleware
func NewAuthMiddleware(authService *service.AuthService) *AuthMiddleware {
	return &AuthMiddleware{authService: authService}
}

// ContextKey for user data
const (
	ContextKeyUserID   = "user_id"
	ContextKeyUsername = "username"
	ContextKeyRole     = "user_role"
	ContextKeyClaims   = "claims"
)

// Required requires authentication
func (m *AuthMiddleware) Required() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, err := m.extractAndValidateClaims(c)
		if err != nil {
			return utils.Error(c, fiber.StatusUnauthorized, utils.ErrCodeUnauthorized, "Token tidak valid atau sudah kedaluwarsa")
		}

		// Set user info in context
		c.Locals(ContextKeyUserID, claims.UserID)
		c.Locals(ContextKeyUsername, claims.Username)
		c.Locals(ContextKeyRole, claims.Role)
		c.Locals(ContextKeyClaims, claims)

		return c.Next()
	}
}

// Optional allows unauthenticated requests but sets user info if authenticated
func (m *AuthMiddleware) Optional() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, err := m.extractAndValidateClaims(c)
		if err == nil && claims != nil {
			c.Locals(ContextKeyUserID, claims.UserID)
			c.Locals(ContextKeyUsername, claims.Username)
			c.Locals(ContextKeyRole, claims.Role)
			c.Locals(ContextKeyClaims, claims)
		}

		return c.Next()
	}
}

// AdminRequired requires admin role
func (m *AuthMiddleware) AdminRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims, err := m.extractAndValidateClaims(c)
		if err != nil {
			return utils.Error(c, fiber.StatusUnauthorized, utils.ErrCodeUnauthorized, "Token tidak valid atau sudah kedaluwarsa")
		}

		// Check admin role
		if claims.Role != string(domain.RoleAdmin) {
			return utils.Error(c, fiber.StatusForbidden, utils.ErrCodeForbidden, "Akses ditolak. Hanya admin yang dapat mengakses.")
		}

		c.Locals(ContextKeyUserID, claims.UserID)
		c.Locals(ContextKeyUsername, claims.Username)
		c.Locals(ContextKeyRole, claims.Role)
		c.Locals(ContextKeyClaims, claims)

		return c.Next()
	}
}

func (m *AuthMiddleware) extractAndValidateClaims(c *fiber.Ctx) (*utils.AccessTokenClaims, error) {
	// Get token from header
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return nil, service.ErrTokenExpired
	}

	// Check Bearer prefix
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil, service.ErrTokenExpired
	}

	tokenString := parts[1]

	// Validate token
	claims, err := m.authService.ValidateAccessToken(tokenString)
	if err != nil {
		return nil, err
	}

	return claims, nil
}

// GetUserID extracts user ID from context
func GetUserID(c *fiber.Ctx) uuid.UUID {
	if id, ok := c.Locals(ContextKeyUserID).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

// GetUserIDOptional extracts user ID pointer from context (nil if not authenticated)
func GetUserIDOptional(c *fiber.Ctx) *uuid.UUID {
	if id, ok := c.Locals(ContextKeyUserID).(uuid.UUID); ok && id != uuid.Nil {
		return &id
	}
	return nil
}

// GetUsername extracts username from context
func GetUsername(c *fiber.Ctx) string {
	if username, ok := c.Locals(ContextKeyUsername).(string); ok {
		return username
	}
	return ""
}

// GetRole extracts role from context
func GetRole(c *fiber.Ctx) string {
	if role, ok := c.Locals(ContextKeyRole).(string); ok {
		return role
	}
	return ""
}

// IsAdmin checks if the current user is an admin
func IsAdmin(c *fiber.Ctx) bool {
	return GetRole(c) == string(domain.RoleAdmin)
}
