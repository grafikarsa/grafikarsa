package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/middleware"
	"github.com/grafikarsa/backend/internal/repository"
)

type NotificationHandler struct {
	repo       *repository.NotificationRepository
	userRepo   *repository.UserRepository
	followRepo *repository.FollowRepository
}

func NewNotificationHandler(repo *repository.NotificationRepository, userRepo *repository.UserRepository, followRepo *repository.FollowRepository) *NotificationHandler {
	return &NotificationHandler{repo: repo, userRepo: userRepo, followRepo: followRepo}
}

// List - GET /notifications
func (h *NotificationHandler) List(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse("UNAUTHORIZED", "Unauthorized"))
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	unreadOnly := c.QueryBool("unread_only", false)

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}

	notifications, total, err := h.repo.FindByUserID(*userID, unreadOnly, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("FETCH_FAILED", "Gagal mengambil notifikasi"))
	}

	unreadCount, _ := h.repo.CountUnread(*userID)

	// Collect unique actor IDs from notification data
	actorIDs := make(map[uuid.UUID]bool)
	for _, n := range notifications {
		data := map[string]interface{}(n.Data)
		// Check various actor ID fields
		for _, key := range []string{"follower_id", "liker_id", "actor_id"} {
			if idStr, exists := data[key]; exists {
				if strVal, ok := idStr.(string); ok {
					if id, err := uuid.Parse(strVal); err == nil {
						actorIDs[id] = true
					}
				}
			}
		}
	}

	// Batch fetch current user data for actors
	actorData := make(map[uuid.UUID]*dto.ActorInfo)
	for id := range actorIDs {
		if user, err := h.userRepo.FindByID(id); err == nil {
			isFollowing, _ := h.followRepo.IsFollowing(*userID, id)
			actorData[id] = &dto.ActorInfo{
				ID:          user.ID.String(),
				Username:    user.Username,
				Nama:        user.Nama,
				AvatarURL:   user.AvatarURL,
				IsFollowing: isFollowing,
			}
		}
	}

	var responses []dto.NotificationResponse
	for _, n := range notifications {
		resp := dto.NotificationResponse{
			ID:        n.ID.String(),
			Type:      string(n.Type),
			Title:     n.Title,
			Message:   n.Message,
			Data:      n.Data,
			IsRead:    n.IsRead,
			ReadAt:    n.ReadAt,
			CreatedAt: n.CreatedAt,
		}

		// Enrich data with current user info
		data := map[string]interface{}(n.Data)
		enrichedData := make(map[string]interface{})
		for k, v := range data {
			enrichedData[k] = v
		}

		// Update actor info based on notification type
		var actorID uuid.UUID
		var actorKey string
		switch n.Type {
		case domain.NotifNewFollower:
			if idStr, exists := data["follower_id"]; exists {
				if strVal, ok := idStr.(string); ok {
					actorID, _ = uuid.Parse(strVal)
					actorKey = "follower"
				}
			}
		case domain.NotifPortfolioLiked:
			if idStr, exists := data["liker_id"]; exists {
				if strVal, ok := idStr.(string); ok {
					actorID, _ = uuid.Parse(strVal)
					actorKey = "liker"
				}
			}
		case domain.NotifFeedbackUpdated:
			if idStr, exists := data["actor_id"]; exists {
				if strVal, ok := idStr.(string); ok {
					actorID, _ = uuid.Parse(strVal)
					actorKey = "actor"
				}
			}
		}

		// Enrich with current data
		if actor, exists := actorData[actorID]; exists && actorKey != "" {
			enrichedData[actorKey+"_username"] = actor.Username
			enrichedData[actorKey+"_nama"] = actor.Nama
			enrichedData[actorKey+"_avatar"] = actor.AvatarURL
			enrichedData[actorKey+"_is_following"] = actor.IsFollowing

			// Regenerate message with current username
			if n.Message != nil {
				switch n.Type {
				case domain.NotifNewFollower:
					newMsg := "@" + actor.Username + " mulai mengikuti kamu"
					resp.Message = &newMsg
				case domain.NotifPortfolioLiked:
					if judul, ok := data["portfolio_judul"].(string); ok {
						newMsg := "@" + actor.Username + " menyukai portfolio \"" + judul + "\""
						resp.Message = &newMsg
					}
				case domain.NotifFeedbackUpdated:
					if newStatus, ok := data["new_status"].(string); ok {
						statusMap := map[string]string{
							"pending":  "Pending",
							"read":     "Dibaca",
							"resolved": "Selesai",
						}
						newMsg := "@" + actor.Username + " mengubah status feedback kamu menjadi " + statusMap[newStatus]
						resp.Message = &newMsg
					}
				}
			}
		}

		resp.Data = enrichedData
		responses = append(responses, resp)
	}

	totalPages := (int(total) + limit - 1) / limit

	return c.JSON(fiber.Map{
		"success": true,
		"data":    responses,
		"meta": dto.NotificationListMeta{
			Page:        page,
			Limit:       limit,
			Total:       total,
			TotalPages:  totalPages,
			UnreadCount: unreadCount,
		},
	})
}

// Count - GET /notifications/count
func (h *NotificationHandler) Count(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse("UNAUTHORIZED", "Unauthorized"))
	}

	count, err := h.repo.CountUnread(*userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("FETCH_FAILED", "Gagal mengambil jumlah notifikasi"))
	}

	return c.JSON(dto.SuccessResponse(dto.NotificationCountResponse{UnreadCount: count}, ""))
}

// MarkAsRead - PATCH /notifications/:id/read
func (h *NotificationHandler) MarkAsRead(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse("UNAUTHORIZED", "Unauthorized"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "ID tidak valid"))
	}

	// Check ownership
	notification, err := h.repo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Notifikasi tidak ditemukan"))
	}
	if notification.UserID != *userID {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse("FORBIDDEN", "Tidak memiliki akses"))
	}

	if err := h.repo.MarkAsRead(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("UPDATE_FAILED", "Gagal mengupdate notifikasi"))
	}

	return c.JSON(dto.SuccessResponse(nil, "Notifikasi ditandai sudah dibaca"))
}

// MarkAllAsRead - POST /notifications/read-all
func (h *NotificationHandler) MarkAllAsRead(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse("UNAUTHORIZED", "Unauthorized"))
	}

	if err := h.repo.MarkAllAsRead(*userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("UPDATE_FAILED", "Gagal mengupdate notifikasi"))
	}

	return c.JSON(dto.SuccessResponse(nil, "Semua notifikasi ditandai sudah dibaca"))
}

// Delete - DELETE /notifications/:id
func (h *NotificationHandler) Delete(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse("UNAUTHORIZED", "Unauthorized"))
	}

	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "ID tidak valid"))
	}

	// Check ownership
	notification, err := h.repo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Notifikasi tidak ditemukan"))
	}
	if notification.UserID != *userID {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse("FORBIDDEN", "Tidak memiliki akses"))
	}

	if err := h.repo.Delete(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("DELETE_FAILED", "Gagal menghapus notifikasi"))
	}

	return c.JSON(dto.SuccessResponse(nil, "Notifikasi berhasil dihapus"))
}
