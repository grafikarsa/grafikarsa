package handler

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/middleware"
	"github.com/grafikarsa/backend/internal/repository"
	"github.com/grafikarsa/backend/internal/service"
)

type FeedbackHandler struct {
	feedbackRepo *repository.FeedbackRepository
	userRepo     *repository.UserRepository
	notifService *service.NotificationService
}

func NewFeedbackHandler(
	feedbackRepo *repository.FeedbackRepository,
	userRepo *repository.UserRepository,
	notifService *service.NotificationService,
) *FeedbackHandler {
	return &FeedbackHandler{
		feedbackRepo: feedbackRepo,
		userRepo:     userRepo,
		notifService: notifService,
	}
}

// CreateFeedback - POST /feedback (public, auth optional)
func (h *FeedbackHandler) CreateFeedback(c *fiber.Ctx) error {
	var req dto.CreateFeedbackRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"INVALID_REQUEST", "Format request tidak valid",
		))
	}

	if req.Kategori == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Kategori wajib diisi",
		))
	}
	if len(req.Pesan) < 10 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Pesan minimal 10 karakter",
		))
	}
	if len(req.Pesan) > 2000 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Pesan maksimal 2000 karakter",
		))
	}

	userID := middleware.GetUserID(c)

	feedback := &domain.Feedback{
		UserID:   userID,
		Kategori: domain.FeedbackKategori(req.Kategori),
		Pesan:    req.Pesan,
		Status:   domain.FeedbackStatusPending,
	}

	if err := h.feedbackRepo.Create(feedback); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"CREATE_FAILED", "Gagal menyimpan feedback",
		))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse(
		h.toResponse(feedback), "Feedback berhasil dikirim",
	))
}

// AdminListFeedback - GET /admin/feedback (admin only)
func (h *FeedbackHandler) AdminListFeedback(c *fiber.Ctx) error {
	status := c.Query("status")
	kategori := c.Query("kategori")
	search := c.Query("search")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	feedbacks, total, err := h.feedbackRepo.List(status, kategori, search, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"FETCH_FAILED", "Gagal mengambil data feedback",
		))
	}

	var responses []dto.FeedbackResponse
	for _, f := range feedbacks {
		responses = append(responses, h.toResponse(&f))
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return c.JSON(dto.PaginatedResponse(
		responses,
		dto.PaginationMeta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: int(totalPages),
		},
	))
}

// AdminGetFeedback - GET /admin/feedback/:id (admin only)
func (h *FeedbackHandler) AdminGetFeedback(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "ID tidak valid"))
	}

	feedback, err := h.feedbackRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Feedback tidak ditemukan"))
	}

	return c.JSON(dto.SuccessResponse(h.toResponse(feedback), ""))
}

// AdminUpdateFeedback - PATCH /admin/feedback/:id (admin only)
func (h *FeedbackHandler) AdminUpdateFeedback(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "ID tidak valid"))
	}

	feedback, err := h.feedbackRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Feedback tidak ditemukan"))
	}

	var req dto.UpdateFeedbackRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_REQUEST", "Format request tidak valid"))
	}

	adminID := middleware.GetUserID(c)

	// Track changes for notification
	oldStatus := feedback.Status
	statusChanged := false

	if req.Status != nil {
		if feedback.Status != domain.FeedbackStatus(*req.Status) {
			statusChanged = true
			feedback.Status = domain.FeedbackStatus(*req.Status)
			if *req.Status == dto.FeedbackStatusResolved {
				now := time.Now()
				feedback.ResolvedAt = &now
				feedback.ResolvedBy = adminID
			}
		}
	}

	if req.AdminNotes != nil {
		feedback.AdminNotes = req.AdminNotes
	}

	if err := h.feedbackRepo.Update(feedback); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("UPDATE_FAILED", "Gagal mengupdate feedback"))
	}

	// Send notification if status changed and feedback belongs to a user
	if statusChanged && feedback.UserID != nil {
		go func() {
			// Fetch admin details to get role name
			adminUser, err := h.userRepo.FindByID(*adminID)
			if err != nil {
				log.Printf("[NotificationDebug] Failed to find admin user %s: %v", adminID, err)
				return
			}

			var roleName string
			if adminUser.Role == domain.RoleAdmin {
				roleName = "Admin"
			} else {
				// Check special roles
				specialRoles, err := h.userRepo.GetUserSpecialRoles(*adminID)
				if err == nil && len(specialRoles) > 0 {
					roleName = specialRoles[0].Nama // Use the first special role
				} else {
					roleName = "Moderator" // Fallback
				}
			}

			log.Printf("[NotificationDebug] Triggering notification. FeedbackID: %s, Actor: %s (%s), OldStatus: %s, NewStatus: %s",
				feedback.ID, adminUser.Username, roleName, oldStatus, feedback.Status)

			if err := h.notifService.NotifyFeedbackStatusUpdated(feedback, adminUser, roleName, oldStatus, feedback.Status); err != nil {
				log.Printf("[NotificationDebug] Failed to send notification: %v", err)
			} else {
				log.Printf("[NotificationDebug] Notification sent successfully")
			}
		}()
	} else {
		log.Printf("[NotificationDebug] Skipped notification. StatusChanged: %v, UserID: %v", statusChanged, feedback.UserID)
	}

	return c.JSON(dto.SuccessResponse(h.toResponse(feedback), "Feedback berhasil diupdate"))
}

// AdminDeleteFeedback - DELETE /admin/feedback/:id (admin only)
func (h *FeedbackHandler) AdminDeleteFeedback(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "ID tidak valid"))
	}

	if _, err := h.feedbackRepo.FindByID(id); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Feedback tidak ditemukan"))
	}

	if err := h.feedbackRepo.Delete(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("DELETE_FAILED", "Gagal menghapus feedback"))
	}

	return c.JSON(dto.SuccessResponse(nil, "Feedback berhasil dihapus"))
}

// AdminGetFeedbackStats - GET /admin/feedback/stats (admin only)
func (h *FeedbackHandler) AdminGetFeedbackStats(c *fiber.Ctx) error {
	total, pending, read, resolved, err := h.feedbackRepo.GetStats()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("FETCH_FAILED", "Gagal mengambil statistik"))
	}

	return c.JSON(dto.SuccessResponse(dto.FeedbackStats{
		Total:    total,
		Pending:  pending,
		Read:     read,
		Resolved: resolved,
	}, ""))
}

func (h *FeedbackHandler) toResponse(f *domain.Feedback) dto.FeedbackResponse {
	resp := dto.FeedbackResponse{
		ID:         f.ID.String(),
		Kategori:   dto.FeedbackKategori(f.Kategori),
		Pesan:      f.Pesan,
		Status:     dto.FeedbackStatus(f.Status),
		AdminNotes: f.AdminNotes,
		ResolvedAt: f.ResolvedAt,
		CreatedAt:  f.CreatedAt,
		UpdatedAt:  f.UpdatedAt,
	}

	if f.UserID != nil {
		s := f.UserID.String()
		resp.UserID = &s
	}
	if f.ResolvedBy != nil {
		s := f.ResolvedBy.String()
		resp.ResolvedBy = &s
	}
	if f.User != nil {
		resp.User = &dto.UserBriefResponse{
			ID:        f.User.ID.String(),
			Username:  f.User.Username,
			Nama:      f.User.Nama,
			AvatarURL: f.User.AvatarURL,
		}
	}

	return resp
}
