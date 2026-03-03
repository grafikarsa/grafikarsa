package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/repository"
)

type TagHandler struct {
	adminRepo *repository.AdminRepository
}

func NewTagHandler(adminRepo *repository.AdminRepository) *TagHandler {
	return &TagHandler{adminRepo: adminRepo}
}

func (h *TagHandler) List(c *fiber.Ctx) error {
	search := c.Query("search")
	tags, err := h.adminRepo.ListTags(search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil data tags"))
	}

	var result []dto.TagDTO
	for _, t := range tags {
		result = append(result, dto.TagDTO{ID: t.ID, Nama: t.Nama})
	}

	return c.JSON(dto.SuccessResponse(result, ""))
}
