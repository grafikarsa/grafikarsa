package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/middleware"
	"github.com/grafikarsa/backend/internal/repository"
)

type ContentBlockHandler struct {
	portfolioRepo *repository.PortfolioRepository
}

func NewContentBlockHandler(portfolioRepo *repository.PortfolioRepository) *ContentBlockHandler {
	return &ContentBlockHandler{portfolioRepo: portfolioRepo}
}

func (h *ContentBlockHandler) Create(c *fiber.Ctx) error {
	portfolioID, err := uuid.Parse(c.Params("portfolio_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Portfolio ID tidak valid",
		))
	}

	currentUserID := middleware.GetUserID(c)
	portfolio, err := h.portfolioRepo.FindByID(portfolioID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"PORTFOLIO_NOT_FOUND", "Portfolio tidak ditemukan",
		))
	}

	isOwner := currentUserID != nil && *currentUserID == portfolio.UserID
	isAdmin := middleware.GetUserRole(c) == "admin"
	if !isOwner && !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
			"FORBIDDEN", "Anda tidak memiliki akses",
		))
	}

	var req dto.CreateContentBlockRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Request body tidak valid",
		))
	}

	// Get max order
	maxOrder, _ := h.portfolioRepo.GetMaxBlockOrder(portfolioID)
	blockOrder := maxOrder + 1
	if req.BlockOrder > 0 {
		blockOrder = req.BlockOrder
	}

	block := &domain.ContentBlock{
		PortfolioID: portfolioID,
		BlockType:   domain.ContentBlockType(req.BlockType),
		BlockOrder:  blockOrder,
		Payload:     req.Payload,
	}

	if err := h.portfolioRepo.CreateContentBlock(block); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal membuat content block",
		))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse(dto.ContentBlockDTO{
		ID:         block.ID,
		BlockType:  string(block.BlockType),
		BlockOrder: block.BlockOrder,
		Payload:    block.Payload,
		CreatedAt:  block.CreatedAt,
	}, "Content block berhasil ditambahkan"))
}

func (h *ContentBlockHandler) Update(c *fiber.Ctx) error {
	portfolioID, err := uuid.Parse(c.Params("portfolio_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Portfolio ID tidak valid",
		))
	}

	blockID, err := uuid.Parse(c.Params("block_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Block ID tidak valid",
		))
	}

	currentUserID := middleware.GetUserID(c)
	portfolio, err := h.portfolioRepo.FindByID(portfolioID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"PORTFOLIO_NOT_FOUND", "Portfolio tidak ditemukan",
		))
	}

	isOwner := currentUserID != nil && *currentUserID == portfolio.UserID
	isAdmin := middleware.GetUserRole(c) == "admin"
	if !isOwner && !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
			"FORBIDDEN", "Anda tidak memiliki akses",
		))
	}

	block, err := h.portfolioRepo.FindContentBlockByID(blockID)
	if err != nil || block.PortfolioID != portfolioID {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"NOT_FOUND", "Content block tidak ditemukan",
		))
	}

	var req dto.UpdateContentBlockRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Request body tidak valid",
		))
	}

	if req.Payload != nil {
		block.Payload = req.Payload
	}

	if err := h.portfolioRepo.UpdateContentBlock(block); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal memperbarui content block",
		))
	}

	return c.JSON(dto.SuccessResponse(dto.ContentBlockDTO{
		ID:         block.ID,
		BlockType:  string(block.BlockType),
		BlockOrder: block.BlockOrder,
		Payload:    block.Payload,
		UpdatedAt:  block.UpdatedAt,
	}, "Content block berhasil diperbarui"))
}

func (h *ContentBlockHandler) Delete(c *fiber.Ctx) error {
	portfolioID, err := uuid.Parse(c.Params("portfolio_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Portfolio ID tidak valid",
		))
	}

	blockID, err := uuid.Parse(c.Params("block_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Block ID tidak valid",
		))
	}

	currentUserID := middleware.GetUserID(c)
	portfolio, err := h.portfolioRepo.FindByID(portfolioID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"PORTFOLIO_NOT_FOUND", "Portfolio tidak ditemukan",
		))
	}

	isOwner := currentUserID != nil && *currentUserID == portfolio.UserID
	isAdmin := middleware.GetUserRole(c) == "admin"
	if !isOwner && !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
			"FORBIDDEN", "Anda tidak memiliki akses",
		))
	}

	block, err := h.portfolioRepo.FindContentBlockByID(blockID)
	if err != nil || block.PortfolioID != portfolioID {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"NOT_FOUND", "Content block tidak ditemukan",
		))
	}

	if err := h.portfolioRepo.DeleteContentBlock(blockID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal menghapus content block",
		))
	}

	return c.JSON(dto.SuccessResponse(nil, "Content block berhasil dihapus"))
}

func (h *ContentBlockHandler) Reorder(c *fiber.Ctx) error {
	portfolioID, err := uuid.Parse(c.Params("portfolio_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Portfolio ID tidak valid",
		))
	}

	currentUserID := middleware.GetUserID(c)
	portfolio, err := h.portfolioRepo.FindByID(portfolioID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"PORTFOLIO_NOT_FOUND", "Portfolio tidak ditemukan",
		))
	}

	isOwner := currentUserID != nil && *currentUserID == portfolio.UserID
	isAdmin := middleware.GetUserRole(c) == "admin"
	if !isOwner && !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
			"FORBIDDEN", "Anda tidak memiliki akses",
		))
	}

	var req dto.ReorderBlocksRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Request body tidak valid",
		))
	}

	orders := make(map[uuid.UUID]int)
	for _, item := range req.BlockOrders {
		orders[item.ID] = item.Order
	}

	if err := h.portfolioRepo.ReorderContentBlocks(portfolioID, orders); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal mengubah urutan content blocks",
		))
	}

	return c.JSON(dto.SuccessResponse(nil, "Urutan content blocks berhasil diperbarui"))
}
