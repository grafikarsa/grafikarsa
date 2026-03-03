package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/repository"
)

type CapabilityMiddleware struct {
	adminRepo *repository.AdminRepository
}

func NewCapabilityMiddleware(adminRepo *repository.AdminRepository) *CapabilityMiddleware {
	return &CapabilityMiddleware{adminRepo: adminRepo}
}

// RequireCapability checks if user has admin role OR the specified capability
func (m *CapabilityMiddleware) RequireCapability(capability string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get user role from context (set by auth middleware)
		role := c.Locals("userRole")
		if role == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
				"UNAUTHORIZED",
				"User tidak terautentikasi",
			))
		}

		// Admin has all capabilities
		if role.(string) == "admin" {
			return c.Next()
		}

		// Get user ID
		userIDLocal := c.Locals("userID")
		if userIDLocal == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
				"UNAUTHORIZED",
				"User tidak terautentikasi",
			))
		}
		userID := userIDLocal.(uuid.UUID)

		// Check if user has the required capability
		hasCapability, err := m.adminRepo.HasCapability(userID, capability)
		if err != nil || !hasCapability {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
				"FORBIDDEN",
				"Anda tidak memiliki akses untuk fitur ini",
			))
		}

		return c.Next()
	}
}

// RequireAnyCapability checks if user has admin role OR any of the specified capabilities
func (m *CapabilityMiddleware) RequireAnyCapability(capabilities ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get user role from context
		role := c.Locals("userRole")
		if role == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
				"UNAUTHORIZED",
				"User tidak terautentikasi",
			))
		}

		// Admin has all capabilities
		if role.(string) == "admin" {
			return c.Next()
		}

		// Get user ID
		userIDLocal := c.Locals("userID")
		if userIDLocal == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
				"UNAUTHORIZED",
				"User tidak terautentikasi",
			))
		}
		userID := userIDLocal.(uuid.UUID)

		// Check if user has any of the required capabilities
		userCapabilities, err := m.adminRepo.GetUserCapabilities(userID)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
				"FORBIDDEN",
				"Anda tidak memiliki akses untuk fitur ini",
			))
		}

		// Check if any capability matches
		capSet := make(map[string]bool)
		for _, cap := range userCapabilities {
			capSet[cap] = true
		}

		for _, required := range capabilities {
			if capSet[required] {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
			"FORBIDDEN",
			"Anda tidak memiliki akses untuk fitur ini",
		))
	}
}

// AdminOrCapability allows admin OR users with specific capability
// This replaces AdminOnly() for routes that should be accessible by special roles
func (m *CapabilityMiddleware) AdminOrCapability(capability string) fiber.Handler {
	return m.RequireCapability(capability)
}
