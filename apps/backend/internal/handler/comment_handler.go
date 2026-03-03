package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/service"
)

type CommentHandler struct {
	service *service.CommentService
}

func NewCommentHandler(service *service.CommentService) *CommentHandler {
	return &CommentHandler{service: service}
}

func (h *CommentHandler) Create(c *fiber.Ctx) error {
	userID, err := GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse("UNAUTHORIZED", "Unauthorized"))
	}

	var req dto.CreateCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_BODY", "Invalid request body"))
	}

	// If portfolio_id is in param, override it (or optional)
	portfolioIDParam := c.Params("portfolio_id")
	if portfolioIDParam != "" {
		if pid, err := uuid.Parse(portfolioIDParam); err == nil {
			req.PortfolioID = pid
		}
	}

	if req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_CONTENT", "Content is required"))
	}

	comment, err := h.service.Create(userID, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.JSON(dto.SuccessResponse(comment, "Comment created successfully"))
}

func (h *CommentHandler) GetByPortfolioID(c *fiber.Ctx) error {
	idStr := c.Params("id") // Portfolio ID
	portfolioID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "Invalid portfolio ID"))
	}

	comments, err := h.service.GetByPortfolioID(portfolioID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.JSON(dto.SuccessResponse(comments, "Comments retrieved successfully"))
}

func (h *CommentHandler) Delete(c *fiber.Ctx) error {
	userID, err := GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse("UNAUTHORIZED", "Unauthorized"))
	}

	idStr := c.Params("id")
	commentID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "Invalid comment ID"))
	}

	// Check if admin
	userRole := GetUserRole(c)
	isAdmin := userRole == string(domain.RoleAdmin)

	if err := h.service.Delete(userID, commentID, isAdmin); err != nil {
		if err.Error() == "unauthorized" {
			return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse("FORBIDDEN", "You are not allowed to delete this comment"))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", err.Error()))
	}

	return c.JSON(dto.SuccessResponse(nil, "Comment deleted successfully"))
}

// Helpers (assuming they exist in other handlers or utils, duplicating if not exported)
// In a real scenario, these should be imported from middleware or util package
// But handler package usually has them or uses c.Locals
func GetUserID(c *fiber.Ctx) (uuid.UUID, error) {
	// Middleware sets "userID" as uuid.UUID
	userID := c.Locals("userID")
	if userID == nil {
		return uuid.Nil, fiber.ErrUnauthorized
	}

	// Type assertion
	id, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, fiber.ErrUnauthorized
	}
	return id, nil
}

func GetUserRole(c *fiber.Ctx) string {
	// Middleware sets "userRole"
	role, ok := c.Locals("userRole").(string)
	if !ok {
		return ""
	}
	return role
}
