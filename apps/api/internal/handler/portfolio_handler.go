package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"grafikarsa/internal/domain"
	"grafikarsa/internal/middleware"
	"grafikarsa/internal/repository"
	"grafikarsa/internal/service"
	"grafikarsa/internal/utils"
)

// PortfolioHandler handles portfolio endpoints
type PortfolioHandler struct {
	portfolioService *service.PortfolioService
}

// NewPortfolioHandler creates a new PortfolioHandler
func NewPortfolioHandler(portfolioService *service.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{portfolioService: portfolioService}
}

// Register registers portfolio routes
func (h *PortfolioHandler) Register(app fiber.Router, authMiddleware *middleware.AuthMiddleware) {
	portfolios := app.Group("/portfolios")

	// Public routes
	portfolios.Get("", authMiddleware.Optional(), h.ListPortfolios)
	portfolios.Get("/:username/:slug", authMiddleware.Optional(), h.GetPortfolio)

	// Protected portfolio management
	portfolios.Post("", authMiddleware.Required(), h.CreatePortfolio)

	// Portfolio by ID (for owner/admin)
	portfolioByID := portfolios.Group("/:id", authMiddleware.Required())
	portfolioByID.Get("", h.GetPortfolioByID)
	portfolioByID.Patch("", h.UpdatePortfolio)
	portfolioByID.Post("/submit", h.SubmitPortfolio)
	portfolioByID.Post("/archive", h.ArchivePortfolio)
	portfolioByID.Post("/unarchive", h.UnarchivePortfolio)
	portfolioByID.Delete("", h.DeletePortfolio)

	// Content blocks
	portfolioByID.Get("/blocks", h.GetContentBlocks)
	portfolioByID.Post("/blocks", h.AddContentBlock)
	portfolioByID.Patch("/blocks/:blockId", h.UpdateContentBlock)
	portfolioByID.Delete("/blocks/:blockId", h.DeleteContentBlock)
	portfolioByID.Put("/blocks/reorder", h.ReorderContentBlocks)

	// Likes
	portfolioByID.Post("/like", h.LikePortfolio)
	portfolioByID.Delete("/like", h.UnlikePortfolio)

	// Feed
	app.Get("/feed", authMiddleware.Required(), h.GetFeed)

	// My portfolios
	app.Get("/me/portfolios", authMiddleware.Required(), h.GetMyPortfolios)
}

// ListPortfolios lists published portfolios
func (h *PortfolioHandler) ListPortfolios(c *fiber.Ctx) error {
	currentUserID := middleware.GetUserIDOptional(c)

	filter := repository.PortfolioFilter{
		Search: c.Query("search"),
		Sort:   c.Query("sort", "-published_at"),
		Page:   c.QueryInt("page", 1),
		Limit:  c.QueryInt("limit", 20),
	}

	if majorID := c.Query("major_id"); majorID != "" {
		if id, err := uuid.Parse(majorID); err == nil {
			filter.MajorID = &id
		}
	}

	if classID := c.Query("class_id"); classID != "" {
		if id, err := uuid.Parse(classID); err == nil {
			filter.ClassID = &id
		}
	}

	// Parse tag IDs
	if tagIDsStr := c.Query("tag_ids"); tagIDsStr != "" {
		var tagIDs []uuid.UUID
		for _, idStr := range c.Context().QueryArgs().PeekMulti("tag_ids") {
			if id, err := uuid.Parse(string(idStr)); err == nil {
				tagIDs = append(tagIDs, id)
			}
		}
		filter.TagIDs = tagIDs
	}

	portfolios, meta, err := h.portfolioService.ListPublishedPortfolios(c.Context(), filter, currentUserID)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil portofolio")
	}

	return utils.SuccessWithMeta(c, fiber.StatusOK, portfolios, meta)
}

// GetPortfolio gets a portfolio by username and slug
func (h *PortfolioHandler) GetPortfolio(c *fiber.Ctx) error {
	username := c.Params("username")
	slug := c.Params("slug")
	currentUserID := middleware.GetUserIDOptional(c)

	portfolio, err := h.portfolioService.GetPortfolio(c.Context(), username, slug, currentUserID)
	if err != nil {
		if err == service.ErrPortfolioNotFound {
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodePortfolioNotFound, "Portofolio tidak ditemukan")
		}
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil portofolio")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, portfolio, "")
}

// CreatePortfolioRequest is the request for creating a portfolio
type CreatePortfolioRequest struct {
	Title  string      `json:"title" validate:"required,min=3,max=200"`
	TagIDs []uuid.UUID `json:"tag_ids" validate:"omitempty,max=10"`
}

// CreatePortfolio creates a new portfolio
func (h *PortfolioHandler) CreatePortfolio(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req CreatePortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	input := service.CreatePortfolioInput{
		Title:  req.Title,
		TagIDs: req.TagIDs,
	}

	portfolio, err := h.portfolioService.CreatePortfolio(c.Context(), userID, input)
	if err != nil {
		if err == service.ErrRateLimitExceeded {
			return utils.Error(c, fiber.StatusTooManyRequests, utils.ErrCodeRateLimitExceeded, "Batas pembuatan portofolio harian tercapai")
		}
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal membuat portofolio")
	}

	return utils.SuccessResponse(c, fiber.StatusCreated, portfolio, "Portofolio berhasil dibuat")
}

// GetPortfolioByID gets a portfolio by ID (for owner/admin)
func (h *PortfolioHandler) GetPortfolioByID(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	isAdmin := middleware.IsAdmin(c)

	portfolioID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID portofolio tidak valid")
	}

	portfolio, err := h.portfolioService.GetPortfolioByID(c.Context(), portfolioID, userID, isAdmin)
	if err != nil {
		switch err {
		case service.ErrPortfolioNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodePortfolioNotFound, "Portofolio tidak ditemukan")
		case service.ErrPortfolioForbidden:
			return utils.Error(c, fiber.StatusForbidden, utils.ErrCodeForbidden, "Anda tidak memiliki akses ke portofolio ini")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil portofolio")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, portfolio, "")
}

// UpdatePortfolioRequest is the request for updating a portfolio
type UpdatePortfolioRequest struct {
	Title        *string     `json:"title" validate:"omitempty,min=3,max=200"`
	ThumbnailURL *string     `json:"thumbnail_url" validate:"omitempty,url"`
	TagIDs       []uuid.UUID `json:"tag_ids" validate:"omitempty,max=10"`
}

// UpdatePortfolio updates a portfolio
func (h *PortfolioHandler) UpdatePortfolio(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	isAdmin := middleware.IsAdmin(c)

	portfolioID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID portofolio tidak valid")
	}

	var req UpdatePortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	input := service.UpdatePortfolioInput{
		Title:        req.Title,
		ThumbnailURL: req.ThumbnailURL,
		TagIDs:       req.TagIDs,
	}

	portfolio, err := h.portfolioService.UpdatePortfolio(c.Context(), portfolioID, userID, isAdmin, input)
	if err != nil {
		switch err {
		case service.ErrPortfolioNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodePortfolioNotFound, "Portofolio tidak ditemukan")
		case service.ErrPortfolioForbidden:
			return utils.Error(c, fiber.StatusForbidden, utils.ErrCodeForbidden, "Anda tidak memiliki akses")
		case service.ErrPortfolioNotEditable:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodePortfolioNotEditable, "Portofolio tidak dapat diedit")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal memperbarui portofolio")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, portfolio, "Portofolio berhasil diperbarui")
}

// SubmitPortfolio submits a portfolio for review
func (h *PortfolioHandler) SubmitPortfolio(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	portfolioID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID portofolio tidak valid")
	}

	if err := h.portfolioService.SubmitPortfolio(c.Context(), portfolioID, userID); err != nil {
		switch err {
		case service.ErrPortfolioNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodePortfolioNotFound, "Portofolio tidak ditemukan")
		case service.ErrPortfolioForbidden:
			return utils.Error(c, fiber.StatusForbidden, utils.ErrCodeForbidden, "Anda tidak memiliki akses")
		case service.ErrInvalidStatusTransition:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidStatusChange, "Portofolio tidak dapat diajukan")
		case service.ErrIncompletePortfolio:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeIncompletePortfolio, "Portofolio belum lengkap")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengajukan portofolio")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"status": domain.StatusPendingReview,
	}, "Portofolio berhasil diajukan untuk review")
}

// ArchivePortfolio archives a portfolio
func (h *PortfolioHandler) ArchivePortfolio(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	isAdmin := middleware.IsAdmin(c)

	portfolioID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID portofolio tidak valid")
	}

	if err := h.portfolioService.ArchivePortfolio(c.Context(), portfolioID, userID, isAdmin); err != nil {
		switch err {
		case service.ErrPortfolioNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodePortfolioNotFound, "Portofolio tidak ditemukan")
		case service.ErrPortfolioForbidden:
			return utils.Error(c, fiber.StatusForbidden, utils.ErrCodeForbidden, "Anda tidak memiliki akses")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengarsipkan portofolio")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"status": domain.StatusArchived,
	}, "Portofolio berhasil diarsipkan")
}

// UnarchivePortfolio unarchives a portfolio
func (h *PortfolioHandler) UnarchivePortfolio(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	isAdmin := middleware.IsAdmin(c)

	portfolioID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID portofolio tidak valid")
	}

	if err := h.portfolioService.UnarchivePortfolio(c.Context(), portfolioID, userID, isAdmin); err != nil {
		switch err {
		case service.ErrPortfolioNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodePortfolioNotFound, "Portofolio tidak ditemukan")
		case service.ErrPortfolioForbidden:
			return utils.Error(c, fiber.StatusForbidden, utils.ErrCodeForbidden, "Anda tidak memiliki akses")
		case service.ErrInvalidStatusTransition:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidStatusChange, "Portofolio tidak dapat dipulihkan")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal memulihkan portofolio")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"status": domain.StatusDraft,
	}, "Portofolio berhasil dipulihkan")
}

// DeletePortfolio deletes a portfolio
func (h *PortfolioHandler) DeletePortfolio(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	isAdmin := middleware.IsAdmin(c)

	portfolioID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID portofolio tidak valid")
	}

	if err := h.portfolioService.DeletePortfolio(c.Context(), portfolioID, userID, isAdmin); err != nil {
		switch err {
		case service.ErrPortfolioNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodePortfolioNotFound, "Portofolio tidak ditemukan")
		case service.ErrPortfolioForbidden:
			return utils.Error(c, fiber.StatusForbidden, utils.ErrCodeForbidden, "Anda tidak memiliki akses")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal menghapus portofolio")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, nil, "Portofolio berhasil dihapus")
}

// GetContentBlocks gets content blocks for a portfolio
func (h *PortfolioHandler) GetContentBlocks(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	isAdmin := middleware.IsAdmin(c)

	portfolioID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID portofolio tidak valid")
	}

	portfolio, err := h.portfolioService.GetPortfolioByID(c.Context(), portfolioID, userID, isAdmin)
	if err != nil {
		if err == service.ErrPortfolioNotFound || err == service.ErrPortfolioForbidden {
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodePortfolioNotFound, "Portofolio tidak ditemukan")
		}
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil konten")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, portfolio.ContentBlocks, "")
}

// AddContentBlockRequest is the request for adding a content block
type AddContentBlockRequest struct {
	BlockType  domain.BlockType `json:"block_type" validate:"required,block_type"`
	BlockOrder int              `json:"block_order" validate:"gte=0"`
	Payload    json.RawMessage  `json:"payload" validate:"required"`
}

// AddContentBlock adds a content block
func (h *PortfolioHandler) AddContentBlock(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	isAdmin := middleware.IsAdmin(c)

	portfolioID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID portofolio tidak valid")
	}

	var req AddContentBlockRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	block, err := h.portfolioService.AddContentBlock(c.Context(), portfolioID, userID, isAdmin, req.BlockType, req.BlockOrder, req.Payload)
	if err != nil {
		switch err {
		case service.ErrPortfolioNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodePortfolioNotFound, "Portofolio tidak ditemukan")
		case service.ErrPortfolioForbidden:
			return utils.Error(c, fiber.StatusForbidden, utils.ErrCodeForbidden, "Anda tidak memiliki akses")
		case service.ErrPortfolioNotEditable:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodePortfolioNotEditable, "Portofolio tidak dapat diedit")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal menambah blok")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusCreated, block, "Blok berhasil ditambahkan")
}

// UpdateContentBlockRequest is the request for updating a content block
type UpdateContentBlockRequest struct {
	Payload json.RawMessage `json:"payload" validate:"required"`
}

// UpdateContentBlock updates a content block
func (h *PortfolioHandler) UpdateContentBlock(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	isAdmin := middleware.IsAdmin(c)

	portfolioID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID portofolio tidak valid")
	}

	blockID, err := utils.ParseUUID(c.Params("blockId"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID blok tidak valid")
	}

	var req UpdateContentBlockRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	block, err := h.portfolioService.UpdateContentBlock(c.Context(), portfolioID, blockID, userID, isAdmin, req.Payload)
	if err != nil {
		switch err {
		case service.ErrPortfolioNotFound, service.ErrBlockNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeNotFound, "Blok tidak ditemukan")
		case service.ErrPortfolioForbidden:
			return utils.Error(c, fiber.StatusForbidden, utils.ErrCodeForbidden, "Anda tidak memiliki akses")
		case service.ErrPortfolioNotEditable:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodePortfolioNotEditable, "Portofolio tidak dapat diedit")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal memperbarui blok")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, block, "Blok berhasil diperbarui")
}

// DeleteContentBlock deletes a content block
func (h *PortfolioHandler) DeleteContentBlock(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	isAdmin := middleware.IsAdmin(c)

	portfolioID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID portofolio tidak valid")
	}

	blockID, err := utils.ParseUUID(c.Params("blockId"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID blok tidak valid")
	}

	if err := h.portfolioService.DeleteContentBlock(c.Context(), portfolioID, blockID, userID, isAdmin); err != nil {
		switch err {
		case service.ErrPortfolioNotFound, service.ErrBlockNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeNotFound, "Blok tidak ditemukan")
		case service.ErrPortfolioForbidden:
			return utils.Error(c, fiber.StatusForbidden, utils.ErrCodeForbidden, "Anda tidak memiliki akses")
		case service.ErrPortfolioNotEditable:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodePortfolioNotEditable, "Portofolio tidak dapat diedit")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal menghapus blok")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, nil, "Blok berhasil dihapus")
}

// ReorderContentBlocksRequest is the request for reordering blocks
type ReorderContentBlocksRequest struct {
	Blocks []struct {
		ID    uuid.UUID `json:"id"`
		Order int       `json:"order"`
	} `json:"blocks" validate:"required,min=1"`
}

// ReorderContentBlocks reorders content blocks
func (h *PortfolioHandler) ReorderContentBlocks(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	isAdmin := middleware.IsAdmin(c)

	portfolioID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID portofolio tidak valid")
	}

	var req ReorderContentBlocksRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	orders := make([]struct {
		ID    uuid.UUID `json:"id"`
		Order int       `json:"order"`
	}, len(req.Blocks))
	copy(orders, req.Blocks)

	if err := h.portfolioService.ReorderContentBlocks(c.Context(), portfolioID, userID, isAdmin, orders); err != nil {
		switch err {
		case service.ErrPortfolioNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodePortfolioNotFound, "Portofolio tidak ditemukan")
		case service.ErrPortfolioForbidden:
			return utils.Error(c, fiber.StatusForbidden, utils.ErrCodeForbidden, "Anda tidak memiliki akses")
		case service.ErrPortfolioNotEditable:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodePortfolioNotEditable, "Portofolio tidak dapat diedit")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengurutkan blok")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, nil, "Urutan blok berhasil diperbarui")
}

// LikePortfolio likes a portfolio
func (h *PortfolioHandler) LikePortfolio(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	portfolioID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID portofolio tidak valid")
	}

	isLiked, likeCount, err := h.portfolioService.LikePortfolio(c.Context(), portfolioID, userID)
	if err != nil {
		switch err {
		case service.ErrPortfolioNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodePortfolioNotFound, "Portofolio tidak ditemukan")
		case service.ErrAlreadyLiked:
			return utils.Error(c, fiber.StatusConflict, utils.ErrCodeDuplicateLike, "Portofolio sudah disukai")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal menyukai portofolio")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"is_liked":   isLiked,
		"like_count": likeCount,
	}, "Portofolio berhasil disukai")
}

// UnlikePortfolio unlikes a portfolio
func (h *PortfolioHandler) UnlikePortfolio(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	portfolioID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID portofolio tidak valid")
	}

	isLiked, likeCount, err := h.portfolioService.UnlikePortfolio(c.Context(), portfolioID, userID)
	if err != nil {
		switch err {
		case service.ErrPortfolioNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodePortfolioNotFound, "Portofolio tidak ditemukan")
		case service.ErrNotLiked:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Portofolio belum disukai")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal batal menyukai")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"is_liked":   isLiked,
		"like_count": likeCount,
	}, "Berhasil batal menyukai")
}

// GetFeed gets the user's feed
func (h *PortfolioHandler) GetFeed(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	portfolios, meta, err := h.portfolioService.GetFeed(c.Context(), userID, page, limit)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil feed")
	}

	return utils.SuccessWithMeta(c, fiber.StatusOK, portfolios, meta)
}

// GetMyPortfolios gets the current user's portfolios
func (h *PortfolioHandler) GetMyPortfolios(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	status := c.Query("status")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	portfolios, meta, err := h.portfolioService.ListUserPortfolios(c.Context(), userID, status, page, limit)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil portofolio")
	}

	return utils.SuccessWithMeta(c, fiber.StatusOK, portfolios, meta)
}
