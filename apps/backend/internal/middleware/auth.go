package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/auth"
	"github.com/grafikarsa/backend/internal/dto"
	"gorm.io/gorm"
)

type AuthMiddleware struct {
	jwtService *auth.JWTService
	db         *gorm.DB
}

func NewAuthMiddleware(jwtService *auth.JWTService, db *gorm.DB) *AuthMiddleware {
	return &AuthMiddleware{
		jwtService: jwtService,
		db:         db,
	}
}

// Required authentication
func (m *AuthMiddleware) Required() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
				"UNAUTHORIZED",
				"Token tidak ada",
			))
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
				"UNAUTHORIZED",
				"Format token tidak valid",
			))
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := m.jwtService.ValidateAccessToken(tokenString)
		if err != nil {
			if strings.Contains(err.Error(), "expired") {
				return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
					"TOKEN_EXPIRED",
					"Token sudah expired",
				))
			}
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
				"INVALID_TOKEN",
				"Token tidak valid",
			))
		}

		// Check if token is blacklisted
		var count int64
		m.db.Table("token_blacklist").Where("jti = ?", claims.JTI).Count(&count)
		if count > 0 {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
				"TOKEN_REVOKED",
				"Token telah di-revoke",
			))
		}

		userID, _ := uuid.Parse(claims.Sub)
		c.Locals("userID", userID)
		c.Locals("userRole", claims.Role)
		c.Locals("jti", claims.JTI)

		return c.Next()
	}
}

// Optional authentication
func (m *AuthMiddleware) Optional() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Next()
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Next()
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := m.jwtService.ValidateAccessToken(tokenString)
		if err != nil {
			return c.Next()
		}

		// Check if token is blacklisted
		var count int64
		m.db.Table("token_blacklist").Where("jti = ?", claims.JTI).Count(&count)
		if count > 0 {
			return c.Next()
		}

		userID, _ := uuid.Parse(claims.Sub)
		c.Locals("userID", userID)
		c.Locals("userRole", claims.Role)
		c.Locals("jti", claims.JTI)

		return c.Next()
	}
}

// Admin only
func (m *AuthMiddleware) AdminOnly() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := c.Locals("userRole")
		if role == nil || role.(string) != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
				"FORBIDDEN",
				"Akses ditolak. Hanya admin yang diizinkan.",
			))
		}
		return c.Next()
	}
}

// Get current user ID from context
func GetUserID(c *fiber.Ctx) *uuid.UUID {
	userID := c.Locals("userID")
	if userID == nil {
		return nil
	}
	id := userID.(uuid.UUID)
	return &id
}

// Get current user role from context
func GetUserRole(c *fiber.Ctx) string {
	role := c.Locals("userRole")
	if role == nil {
		return ""
	}
	return role.(string)
}

// GetJWTService returns the JWT service for token validation
func (m *AuthMiddleware) GetJWTService() *auth.JWTService {
	return m.jwtService
}
