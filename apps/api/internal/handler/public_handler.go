package handler

import (
	"github.com/gofiber/fiber/v2"

	"grafikarsa/internal/middleware"
	"grafikarsa/internal/service"
	"grafikarsa/internal/utils"
)

// PublicHandler handles public endpoints
type PublicHandler struct {
	tagService   *service.TagService
	adminService *service.AdminService
}

// NewPublicHandler creates a new PublicHandler
func NewPublicHandler(tagService *service.TagService, adminService *service.AdminService) *PublicHandler {
	return &PublicHandler{
		tagService:   tagService,
		adminService: adminService,
	}
}

// Register registers public routes
func (h *PublicHandler) Register(app fiber.Router, authMiddleware *middleware.AuthMiddleware) {
	// Tags - public list
	app.Get("/tags", h.ListTags)

	// Classes - public list for registration/filtering
	app.Get("/classes", h.ListClasses)

	// Majors - public list
	app.Get("/majors", h.ListMajors)

	// Active academic year
	app.Get("/active-year", h.GetActiveAcademicYear)

	// Health check
	app.Get("/health", h.HealthCheck)
}

// ListTags lists all tags (public)
func (h *PublicHandler) ListTags(c *fiber.Ctx) error {
	search := c.Query("search")

	tags, err := h.tagService.ListTags(c.Context(), search)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil tags")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, tags, "")
}

// ListClasses lists classes for public use (active year only)
func (h *PublicHandler) ListClasses(c *fiber.Ctx) error {
	classes, err := h.adminService.ListPublicClasses(c.Context())
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil kelas")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, classes, "")
}

// ListMajors lists all majors (public)
func (h *PublicHandler) ListMajors(c *fiber.Ctx) error {
	majors, err := h.adminService.ListMajors(c.Context())
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil jurusan")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, majors, "")
}

// GetActiveAcademicYear gets the active academic year
func (h *PublicHandler) GetActiveAcademicYear(c *fiber.Ctx) error {
	year, err := h.adminService.GetActiveAcademicYear(c.Context())
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil tahun ajaran")
	}

	if year == nil {
		return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeNotFound, "Tahun ajaran aktif tidak ditemukan")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, year, "")
}

// HealthCheck returns the health status of the API
func (h *PublicHandler) HealthCheck(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "healthy",
		"api":    "grafikarsa-api",
	})
}
