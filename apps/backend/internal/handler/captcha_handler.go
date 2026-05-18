package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/service"
)

type CaptchaHandler struct {
	captchaService *service.CaptchaService
}

func NewCaptchaHandler(captchaService *service.CaptchaService) *CaptchaHandler {
	return &CaptchaHandler{captchaService: captchaService}
}

func (h *CaptchaHandler) Generate(c *fiber.Ctx) error {
	ctx := c.Context()

	id, question, err := h.captchaService.Generate(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal membuat CAPTCHA",
		))
	}

	return c.JSON(dto.SuccessResponse(dto.CaptchaResponse{
		ID:       id,
		Question: question,
	}, ""))
}
