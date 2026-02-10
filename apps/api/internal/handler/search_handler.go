package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"grafikarsa/internal/middleware"
	"grafikarsa/internal/repository"
	"grafikarsa/internal/service"
	"grafikarsa/internal/utils"
)

// SearchHandler handles search endpoints
type SearchHandler struct {
	userService      *service.UserService
	portfolioService *service.PortfolioService
}

// NewSearchHandler creates a new SearchHandler
func NewSearchHandler(userService *service.UserService, portfolioService *service.PortfolioService) *SearchHandler {
	return &SearchHandler{
		userService:      userService,
		portfolioService: portfolioService,
	}
}

// Register registers search routes
func (h *SearchHandler) Register(app fiber.Router, authMiddleware *middleware.AuthMiddleware) {
	search := app.Group("/search")

	// Search endpoints
	search.Get("/users", authMiddleware.Optional(), h.SearchUsers)
	search.Get("/portfolios", authMiddleware.Optional(), h.SearchPortfolios)
}

// SearchUsers searches for users
func (h *SearchHandler) SearchUsers(c *fiber.Ctx) error {
	filter := repository.UserFilter{
		Search: c.Query("q"),
		Role:   c.Query("role"),
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

	users, meta, err := h.userService.ListUsers(c.Context(), filter)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mencari user")
	}

	return utils.SuccessWithMeta(c, fiber.StatusOK, users, meta)
}

// SearchPortfolios searches for portfolios
func (h *SearchHandler) SearchPortfolios(c *fiber.Ctx) error {
	currentUserID := middleware.GetUserIDOptional(c)

	filter := repository.PortfolioFilter{
		Search: c.Query("q"),
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
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mencari portfolio")
	}

	return utils.SuccessWithMeta(c, fiber.StatusOK, portfolios, meta)
}
