package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
)

// ============================================================================
// REQUEST DTOs
// ============================================================================

// StartConversationRequest - Request to start a new conversation
type StartConversationRequest struct {
	RecipientID uuid.UUID `json:"recipient_id" validate:"required"`
	Message     string    `json:"message" validate:"max=5000"`
}

// SendMessageRequest - Request to send a message
type SendMessageRequest struct {
	MessageType string                 `json:"message_type" validate:"required,oneof=text image portfolio"`
	Content     map[string]interface{} `json:"content" validate:"required"`
	ReplyToID   *uuid.UUID             `json:"reply_to_id,omitempty"`
}

// UpdateDMSettingsRequest - Request to update DM settings
type UpdateDMSettingsRequest struct {
	DMPrivacy           *string `json:"dm_privacy,omitempty" validate:"omitempty,oneof=open followers mutual closed"`
	ShowReadReceipts    *bool   `json:"show_read_receipts,omitempty"`
	ShowTypingIndicator *bool   `json:"show_typing_indicator,omitempty"`
}

// AddReactionRequest - Request to add a reaction
type AddReactionRequest struct {
	Emoji string `json:"emoji" validate:"required,max=10"`
}

// ============================================================================
// RESPONSE DTOs
// ============================================================================

// ConversationResponse - Response for a conversation
type ConversationResponse struct {
	ID                 uuid.UUID                `json:"id"`
	CreatedAt          time.Time                `json:"created_at"`
	LastMessageAt      *time.Time               `json:"last_message_at,omitempty"`
	LastMessagePreview *string                  `json:"last_message_preview,omitempty"`
	Participants       []ParticipantResponse    `json:"participants"`
	UnreadCount        int                      `json:"unread_count"`
	IsMuted            bool                     `json:"is_muted"`
	IsArchived         bool                     `json:"is_archived"`
	OtherUser          *ParticipantUserResponse `json:"other_user,omitempty"`
}

// ParticipantResponse - Response for a participant
type ParticipantResponse struct {
	UserID      uuid.UUID                `json:"user_id"`
	JoinedAt    time.Time                `json:"joined_at"`
	LastReadAt  *time.Time               `json:"last_read_at,omitempty"`
	UnreadCount int                      `json:"unread_count"`
	User        *ParticipantUserResponse `json:"user,omitempty"`
}

// ParticipantUserResponse - Minimal user info for participant
type ParticipantUserResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Nama      string    `json:"nama"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	Role      string    `json:"role"`
	IsOnline  bool      `json:"is_online"`
}

// MessageResponse - Response for a message
type MessageResponse struct {
	ID             uuid.UUID                `json:"id"`
	ConversationID uuid.UUID                `json:"conversation_id"`
	SenderID       uuid.UUID                `json:"sender_id"`
	Sender         *ParticipantUserResponse `json:"sender,omitempty"`
	MessageType    string                   `json:"message_type"`
	Content        map[string]interface{}   `json:"content"`
	ReplyTo        *MessageResponse         `json:"reply_to,omitempty"`
	Reactions      []ReactionResponse       `json:"reactions"`
	CreatedAt      time.Time                `json:"created_at"`
	IsDeleted      bool                     `json:"is_deleted"`
}

// ReactionResponse - Response for a reaction
type ReactionResponse struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Emoji    string    `json:"emoji"`
}

// DMSettingsResponse - Response for DM settings
type DMSettingsResponse struct {
	DMPrivacy           string `json:"dm_privacy"`
	ShowReadReceipts    bool   `json:"show_read_receipts"`
	ShowTypingIndicator bool   `json:"show_typing_indicator"`
}

// ChatStreakResponse - Response for a chat streak
type ChatStreakResponse struct {
	OtherUser     ParticipantUserResponse `json:"other_user"`
	CurrentStreak int                     `json:"current_streak"`
	LongestStreak int                     `json:"longest_streak"`
	LastChatDate  *time.Time              `json:"last_chat_date,omitempty"`
}

// BlockedUserResponse - Response for a blocked user
type BlockedUserResponse struct {
	UserID    uuid.UUID `json:"user_id"`
	Username  string    `json:"username"`
	Nama      string    `json:"nama"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	BlockedAt time.Time `json:"blocked_at"`
}

// ============================================================================
// WEBSOCKET EVENT DTOs
// ============================================================================

// WSEvent - Base WebSocket event
type WSEvent struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

// WSMessageNew - New message event
type WSMessageNew struct {
	ConversationID uuid.UUID       `json:"conversation_id"`
	Message        MessageResponse `json:"message"`
}

// WSMessageDeleted - Message deleted event
type WSMessageDeleted struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	MessageID      uuid.UUID `json:"message_id"`
}

// WSMessageReaction - Message reaction event
type WSMessageReaction struct {
	ConversationID uuid.UUID        `json:"conversation_id"`
	MessageID      uuid.UUID        `json:"message_id"`
	Reaction       ReactionResponse `json:"reaction"`
	Action         string           `json:"action"` // "add" or "remove"
}

// WSTyping - Typing indicator event
type WSTyping struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	UserID         uuid.UUID `json:"user_id"`
	Username       string    `json:"username"`
	IsTyping       bool      `json:"is_typing"`
}

// WSPresence - Online presence event
type WSPresence struct {
	UserID   uuid.UUID `json:"user_id"`
	IsOnline bool      `json:"is_online"`
}

// WSReadReceipt - Read receipt event
type WSReadReceipt struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	UserID         uuid.UUID `json:"user_id"`
	ReadAt         time.Time `json:"read_at"`
}

// ============================================================================
// MAPPER FUNCTIONS
// ============================================================================

// MapConversationToResponse maps a conversation to response
func MapConversationToResponse(conv *domain.Conversation, currentUserID uuid.UUID) ConversationResponse {
	resp := ConversationResponse{
		ID:                 conv.ID,
		CreatedAt:          conv.CreatedAt,
		LastMessageAt:      conv.LastMessageAt,
		LastMessagePreview: conv.LastMessagePreview,
		Participants:       make([]ParticipantResponse, 0),
	}

	for _, p := range conv.Participants {
		pr := ParticipantResponse{
			UserID:      p.UserID,
			JoinedAt:    p.JoinedAt,
			LastReadAt:  p.LastReadAt,
			UnreadCount: p.UnreadCount,
		}

		if p.User != nil {
			pr.User = &ParticipantUserResponse{
				ID:        p.User.ID,
				Username:  p.User.Username,
				Nama:      p.User.Nama,
				AvatarURL: p.User.AvatarURL,
				Role:      string(p.User.Role),
			}

			// Set other user for 1-on-1 conversations
			if p.UserID != currentUserID {
				resp.OtherUser = pr.User
			}
		}

		// Get current user's unread count and settings
		if p.UserID == currentUserID {
			resp.UnreadCount = p.UnreadCount
			resp.IsMuted = p.IsMuted
			resp.IsArchived = p.IsArchived
		}

		resp.Participants = append(resp.Participants, pr)
	}

	return resp
}

// MapMessageToResponse maps a message to response
func MapMessageToResponse(msg *domain.Message) MessageResponse {
	resp := MessageResponse{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		SenderID:       msg.SenderID,
		MessageType:    string(msg.MessageType),
		Content:        msg.Content,
		Reactions:      make([]ReactionResponse, 0),
		CreatedAt:      msg.CreatedAt,
		IsDeleted:      msg.DeletedAt != nil,
	}

	if msg.Sender != nil {
		resp.Sender = &ParticipantUserResponse{
			ID:        msg.Sender.ID,
			Username:  msg.Sender.Username,
			Nama:      msg.Sender.Nama,
			AvatarURL: msg.Sender.AvatarURL,
			Role:      string(msg.Sender.Role),
		}
	}

	if msg.ReplyTo != nil && msg.ReplyTo.DeletedAt == nil {
		replyResp := MapMessageToResponse(msg.ReplyTo)
		resp.ReplyTo = &replyResp
	}

	for _, r := range msg.Reactions {
		rr := ReactionResponse{
			UserID: r.UserID,
			Emoji:  r.Emoji,
		}
		if r.User != nil {
			rr.Username = r.User.Username
		}
		resp.Reactions = append(resp.Reactions, rr)
	}

	return resp
}

// MapDMSettingsToResponse maps DM settings to response
func MapDMSettingsToResponse(settings *domain.DMSettings) DMSettingsResponse {
	return DMSettingsResponse{
		DMPrivacy:           string(settings.DMPrivacy),
		ShowReadReceipts:    settings.ShowReadReceipts,
		ShowTypingIndicator: settings.ShowTypingIndicator,
	}
}
