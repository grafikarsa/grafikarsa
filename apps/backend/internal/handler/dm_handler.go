package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/middleware"
	"github.com/grafikarsa/backend/internal/service"
)

type DMHandler struct {
	dmService *service.DMService
}

func NewDMHandler(dmService *service.DMService) *DMHandler {
	return &DMHandler{dmService: dmService}
}

// getUserID extracts user ID from context, returns uuid.Nil if not present
func getUserID(c *fiber.Ctx) uuid.UUID {
	userIDPtr := middleware.GetUserID(c)
	if userIDPtr == nil {
		return uuid.Nil
	}
	return *userIDPtr
}

// ============================================================================
// CONVERSATION ENDPOINTS
// ============================================================================

// ListConversations - GET /api/v1/conversations
// @Summary List user's conversations
// @Tags DM
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param include_archived query bool false "Include archived" default(false)
// @Success 200 {object} map[string]interface{}
// @Router /conversations [get]
func (h *DMHandler) ListConversations(c *fiber.Ctx) error {
	userID := getUserID(c)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	includeArchived := c.Query("include_archived", "false") == "true"

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}

	conversations, total, err := h.dmService.GetConversations(userID, includeArchived, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "FETCH_ERROR",
				"message": "Gagal mengambil conversations",
			},
		})
	}

	// Map to response
	items := make([]dto.ConversationResponse, len(conversations))
	for i, conv := range conversations {
		items[i] = dto.MapConversationToResponse(&conv, userID)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    items,
		"meta": fiber.Map{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_page": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// StartConversation - POST /api/v1/conversations
// @Summary Start a new conversation
// @Tags DM
// @Security BearerAuth
// @Param body body dto.StartConversationRequest true "Start conversation request"
// @Success 201 {object} map[string]interface{}
// @Router /conversations [post]
func (h *DMHandler) StartConversation(c *fiber.Ctx) error {
	userID := getUserID(c)

	var req dto.StartConversationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Request body tidak valid",
			},
		})
	}

	if req.RecipientID == uuid.Nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "recipient_id wajib diisi",
			},
		})
	}

	if req.RecipientID == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Tidak bisa mengirim pesan ke diri sendiri",
			},
		})
	}

	conv, msg, err := h.dmService.StartConversation(userID, req.RecipientID, req.Message)
	if err != nil {
		if err == service.ErrCannotMessageUser {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "PRIVACY_RESTRICTED",
					"message": "User tidak menerima DM dari kamu",
				},
			})
		}
		if err == service.ErrUserBlocked {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "USER_BLOCKED",
					"message": "Kamu atau user ini telah memblokir",
				},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "CREATE_ERROR",
				"message": "Gagal memulai conversation",
			},
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"conversation": dto.MapConversationToResponse(conv, userID),
			"message":      dto.MapMessageToResponse(msg),
		},
	})
}

// GetConversation - GET /api/v1/conversations/:id
// @Summary Get a conversation by ID
// @Tags DM
// @Security BearerAuth
// @Param id path string true "Conversation ID"
// @Success 200 {object} map[string]interface{}
// @Router /conversations/{id} [get]
func (h *DMHandler) GetConversation(c *fiber.Ctx) error {
	userID := getUserID(c)

	convID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_ID",
				"message": "ID tidak valid",
			},
		})
	}

	conv, err := h.dmService.GetConversation(convID, userID)
	if err != nil {
		if err == service.ErrNotInConversation {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "NOT_PARTICIPANT",
					"message": "Kamu bukan peserta conversation ini",
				},
			})
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": "Conversation tidak ditemukan",
			},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.MapConversationToResponse(conv, userID),
	})
}

// ArchiveConversation - POST /api/v1/conversations/:id/archive
// @Summary Archive a conversation
// @Tags DM
// @Security BearerAuth
// @Param id path string true "Conversation ID"
// @Success 200 {object} map[string]interface{}
// @Router /conversations/{id}/archive [post]
func (h *DMHandler) ArchiveConversation(c *fiber.Ctx) error {
	userID := getUserID(c)

	convID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_ID",
				"message": "ID tidak valid",
			},
		})
	}

	if err := h.dmService.ArchiveConversation(convID, userID, true); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "ARCHIVE_ERROR",
				"message": "Gagal mengarsipkan conversation",
			},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Conversation berhasil diarsipkan",
	})
}

// UnarchiveConversation - DELETE /api/v1/conversations/:id/archive
// @Summary Unarchive a conversation
// @Tags DM
// @Security BearerAuth
// @Param id path string true "Conversation ID"
// @Success 200 {object} map[string]interface{}
// @Router /conversations/{id}/archive [delete]
func (h *DMHandler) UnarchiveConversation(c *fiber.Ctx) error {
	userID := getUserID(c)

	convID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_ID",
				"message": "ID tidak valid",
			},
		})
	}

	if err := h.dmService.ArchiveConversation(convID, userID, false); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "UNARCHIVE_ERROR",
				"message": "Gagal mengembalikan conversation",
			},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Conversation berhasil dikembalikan",
	})
}

// MuteConversation - POST /api/v1/conversations/:id/mute
// @Summary Mute a conversation
// @Tags DM
// @Security BearerAuth
// @Param id path string true "Conversation ID"
// @Success 200 {object} map[string]interface{}
// @Router /conversations/{id}/mute [post]
func (h *DMHandler) MuteConversation(c *fiber.Ctx) error {
	userID := getUserID(c)

	convID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_ID",
				"message": "ID tidak valid",
			},
		})
	}

	if err := h.dmService.MuteConversation(convID, userID, true); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "MUTE_ERROR",
				"message": "Gagal mematikan notifikasi",
			},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Notifikasi conversation dimatikan",
	})
}

// UnmuteConversation - DELETE /api/v1/conversations/:id/mute
// @Summary Unmute a conversation
// @Tags DM
// @Security BearerAuth
// @Param id path string true "Conversation ID"
// @Success 200 {object} map[string]interface{}
// @Router /conversations/{id}/mute [delete]
func (h *DMHandler) UnmuteConversation(c *fiber.Ctx) error {
	userID := getUserID(c)

	convID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_ID",
				"message": "ID tidak valid",
			},
		})
	}

	if err := h.dmService.MuteConversation(convID, userID, false); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "UNMUTE_ERROR",
				"message": "Gagal mengaktifkan notifikasi",
			},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Notifikasi conversation diaktifkan",
	})
}

// MarkAsRead - POST /api/v1/conversations/:id/read
// @Summary Mark conversation as read
// @Tags DM
// @Security BearerAuth
// @Param id path string true "Conversation ID"
// @Success 200 {object} map[string]interface{}
// @Router /conversations/{id}/read [post]
func (h *DMHandler) MarkAsRead(c *fiber.Ctx) error {
	userID := getUserID(c)

	convID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_ID",
				"message": "ID tidak valid",
			},
		})
	}

	if err := h.dmService.MarkAsRead(convID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "READ_ERROR",
				"message": "Gagal menandai sebagai dibaca",
			},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Conversation ditandai sebagai dibaca",
	})
}

// ============================================================================
// MESSAGE ENDPOINTS
// ============================================================================

// GetMessages - GET /api/v1/conversations/:id/messages
// @Summary Get messages in a conversation
// @Tags DM
// @Security BearerAuth
// @Param id path string true "Conversation ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(30)
// @Success 200 {object} map[string]interface{}
// @Router /conversations/{id}/messages [get]
func (h *DMHandler) GetMessages(c *fiber.Ctx) error {
	userID := getUserID(c)

	convID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_ID",
				"message": "ID tidak valid",
			},
		})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "30"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 30
	}

	messages, total, err := h.dmService.GetMessages(convID, userID, page, limit)
	if err != nil {
		if err == service.ErrNotInConversation {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "NOT_PARTICIPANT",
					"message": "Kamu bukan peserta conversation ini",
				},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "FETCH_ERROR",
				"message": "Gagal mengambil pesan",
			},
		})
	}

	// Map to response
	items := make([]dto.MessageResponse, len(messages))
	for i, msg := range messages {
		items[i] = dto.MapMessageToResponse(&msg)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    items,
		"meta": fiber.Map{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_page": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// SendMessage - POST /api/v1/conversations/:id/messages
// @Summary Send a message
// @Tags DM
// @Security BearerAuth
// @Param id path string true "Conversation ID"
// @Param body body dto.SendMessageRequest true "Send message request"
// @Success 201 {object} map[string]interface{}
// @Router /conversations/{id}/messages [post]
func (h *DMHandler) SendMessage(c *fiber.Ctx) error {
	userID := getUserID(c)

	convID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_ID",
				"message": "ID tidak valid",
			},
		})
	}

	var req dto.SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Request body tidak valid",
			},
		})
	}

	// Validate message type
	msgType := domain.MessageType(req.MessageType)
	if msgType != domain.MessageTypeText && msgType != domain.MessageTypeImage && msgType != domain.MessageTypePortfolio {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_TYPE",
				"message": "message_type harus text, image, atau portfolio",
			},
		})
	}

	msg, err := h.dmService.SendMessage(convID, userID, msgType, req.Content, req.ReplyToID)
	if err != nil {
		if err == service.ErrNotInConversation {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "NOT_PARTICIPANT",
					"message": "Kamu bukan peserta conversation ini",
				},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "SEND_ERROR",
				"message": "Gagal mengirim pesan",
			},
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    dto.MapMessageToResponse(msg),
	})
}

// DeleteMessage - DELETE /api/v1/messages/:id
// @Summary Delete a message
// @Tags DM
// @Security BearerAuth
// @Param id path string true "Message ID"
// @Success 200 {object} map[string]interface{}
// @Router /messages/{id} [delete]
func (h *DMHandler) DeleteMessage(c *fiber.Ctx) error {
	userID := getUserID(c)

	msgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_ID",
				"message": "ID tidak valid",
			},
		})
	}

	if err := h.dmService.DeleteMessage(msgID, userID); err != nil {
		if err == service.ErrMessageNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "NOT_FOUND",
					"message": "Pesan tidak ditemukan",
				},
			})
		}
		if err == service.ErrCannotDeleteMessage {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "NOT_OWNER",
					"message": "Kamu hanya bisa menghapus pesan sendiri",
				},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "DELETE_ERROR",
				"message": "Gagal menghapus pesan",
			},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Pesan berhasil dihapus",
	})
}

// AddReaction - POST /api/v1/messages/:id/reactions
// @Summary Add a reaction to a message
// @Tags DM
// @Security BearerAuth
// @Param id path string true "Message ID"
// @Param body body dto.AddReactionRequest true "Add reaction request"
// @Success 201 {object} map[string]interface{}
// @Router /messages/{id}/reactions [post]
func (h *DMHandler) AddReaction(c *fiber.Ctx) error {
	userID := getUserID(c)

	msgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_ID",
				"message": "ID tidak valid",
			},
		})
	}

	var req dto.AddReactionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Request body tidak valid",
			},
		})
	}

	if req.Emoji == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "emoji wajib diisi",
			},
		})
	}

	reaction, err := h.dmService.AddReaction(msgID, userID, req.Emoji)
	if err != nil {
		if err == service.ErrMessageNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "NOT_FOUND",
					"message": "Pesan tidak ditemukan",
				},
			})
		}
		if err == service.ErrNotInConversation {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "NOT_PARTICIPANT",
					"message": "Kamu bukan peserta conversation ini",
				},
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "REACTION_ERROR",
				"message": "Gagal menambahkan reaction",
			},
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"id":    reaction.ID,
			"emoji": reaction.Emoji,
		},
	})
}

// RemoveReaction - DELETE /api/v1/messages/:id/reactions/:emoji
// @Summary Remove a reaction from a message
// @Tags DM
// @Security BearerAuth
// @Param id path string true "Message ID"
// @Param emoji path string true "Emoji to remove"
// @Success 200 {object} map[string]interface{}
// @Router /messages/{id}/reactions/{emoji} [delete]
func (h *DMHandler) RemoveReaction(c *fiber.Ctx) error {
	userID := getUserID(c)

	msgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_ID",
				"message": "ID tidak valid",
			},
		})
	}

	emoji := c.Params("emoji")
	if emoji == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "emoji wajib diisi",
			},
		})
	}

	if err := h.dmService.RemoveReaction(msgID, userID, emoji); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "REMOVE_ERROR",
				"message": "Gagal menghapus reaction",
			},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Reaction berhasil dihapus",
	})
}

// ============================================================================
// DM SETTINGS ENDPOINTS
// ============================================================================

// GetDMSettings - GET /api/v1/dm/settings
// @Summary Get DM settings
// @Tags DM
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /dm/settings [get]
func (h *DMHandler) GetDMSettings(c *fiber.Ctx) error {
	userID := getUserID(c)

	settings, err := h.dmService.GetDMSettings(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "FETCH_ERROR",
				"message": "Gagal mengambil pengaturan DM",
			},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.MapDMSettingsToResponse(settings),
	})
}

// UpdateDMSettings - PATCH /api/v1/dm/settings
// @Summary Update DM settings
// @Tags DM
// @Security BearerAuth
// @Param body body dto.UpdateDMSettingsRequest true "Update settings request"
// @Success 200 {object} map[string]interface{}
// @Router /dm/settings [patch]
func (h *DMHandler) UpdateDMSettings(c *fiber.Ctx) error {
	userID := getUserID(c)

	var req dto.UpdateDMSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Request body tidak valid",
			},
		})
	}

	// Validate privacy value
	if req.DMPrivacy != nil {
		validPrivacy := map[string]bool{
			"open": true, "followers": true, "mutual": true, "closed": true,
		}
		if !validPrivacy[*req.DMPrivacy] {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "INVALID_PRIVACY",
					"message": "dm_privacy harus open, followers, mutual, atau closed",
				},
			})
		}
	}

	settings, err := h.dmService.UpdateDMSettings(userID, req.DMPrivacy, req.ShowReadReceipts, req.ShowTypingIndicator)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "UPDATE_ERROR",
				"message": "Gagal memperbarui pengaturan",
			},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.MapDMSettingsToResponse(settings),
	})
}

// ============================================================================
// BLOCK ENDPOINTS
// ============================================================================

// BlockUser - POST /api/v1/dm/block/:userId
// @Summary Block a user
// @Tags DM
// @Security BearerAuth
// @Param userId path string true "User ID to block"
// @Success 200 {object} map[string]interface{}
// @Router /dm/block/{userId} [post]
func (h *DMHandler) BlockUser(c *fiber.Ctx) error {
	userID := getUserID(c)

	blockedID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_ID",
				"message": "ID tidak valid",
			},
		})
	}

	if blockedID == userID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_REQUEST",
				"message": "Tidak bisa memblokir diri sendiri",
			},
		})
	}

	if err := h.dmService.BlockUser(userID, blockedID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "BLOCK_ERROR",
				"message": "Gagal memblokir user",
			},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User berhasil diblokir",
	})
}

// UnblockUser - DELETE /api/v1/dm/block/:userId
// @Summary Unblock a user
// @Tags DM
// @Security BearerAuth
// @Param userId path string true "User ID to unblock"
// @Success 200 {object} map[string]interface{}
// @Router /dm/block/{userId} [delete]
func (h *DMHandler) UnblockUser(c *fiber.Ctx) error {
	userID := getUserID(c)

	blockedID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_ID",
				"message": "ID tidak valid",
			},
		})
	}

	if err := h.dmService.UnblockUser(userID, blockedID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "UNBLOCK_ERROR",
				"message": "Gagal membuka blokir user",
			},
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User berhasil dibuka blokirnya",
	})
}

// GetBlockedUsers - GET /api/v1/dm/blocked
// @Summary Get list of blocked users
// @Tags DM
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /dm/blocked [get]
func (h *DMHandler) GetBlockedUsers(c *fiber.Ctx) error {
	userID := getUserID(c)

	blocks, err := h.dmService.GetBlockedUsers(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "FETCH_ERROR",
				"message": "Gagal mengambil daftar blokir",
			},
		})
	}

	items := make([]dto.BlockedUserResponse, len(blocks))
	for i, b := range blocks {
		items[i] = dto.BlockedUserResponse{
			UserID:    b.BlockedID,
			BlockedAt: b.CreatedAt,
		}
		if b.Blocked != nil {
			items[i].Username = b.Blocked.Username
			items[i].Nama = b.Blocked.Nama
			items[i].AvatarURL = b.Blocked.AvatarURL
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    items,
	})
}

// ============================================================================
// STREAK ENDPOINTS
// ============================================================================

// GetChatStreaks - GET /api/v1/dm/streaks
// @Summary Get active chat streaks
// @Tags DM
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /dm/streaks [get]
func (h *DMHandler) GetChatStreaks(c *fiber.Ctx) error {
	userID := getUserID(c)

	streaks, err := h.dmService.GetChatStreaks(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "FETCH_ERROR",
				"message": "Gagal mengambil data streak",
			},
		})
	}

	items := make([]dto.ChatStreakResponse, len(streaks))
	for i, s := range streaks {
		// Determine other user
		var otherUser *domain.User
		if s.UserAID == userID {
			otherUser = s.UserB
		} else {
			otherUser = s.UserA
		}

		items[i] = dto.ChatStreakResponse{
			CurrentStreak: s.CurrentStreak,
			LongestStreak: s.LongestStreak,
			LastChatDate:  s.LastChatDate,
		}

		if otherUser != nil {
			items[i].OtherUser = dto.ParticipantUserResponse{
				ID:        otherUser.ID,
				Username:  otherUser.Username,
				Nama:      otherUser.Nama,
				AvatarURL: otherUser.AvatarURL,
				Role:      string(otherUser.Role),
			}
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    items,
	})
}
