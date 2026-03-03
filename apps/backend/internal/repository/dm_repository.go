package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"gorm.io/gorm"
)

type DMRepository struct {
	db *gorm.DB
}

func NewDMRepository(db *gorm.DB) *DMRepository {
	return &DMRepository{db: db}
}

// ============================================================================
// CONVERSATION METHODS
// ============================================================================

// CreateConversation creates a new conversation
func (r *DMRepository) CreateConversation(conv *domain.Conversation) error {
	return r.db.Create(conv).Error
}

// FindConversationByID finds a conversation by ID with participants
func (r *DMRepository) FindConversationByID(id uuid.UUID) (*domain.Conversation, error) {
	var conv domain.Conversation
	err := r.db.Preload("Participants").Preload("Participants.User").
		Where("id = ?", id).First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// FindConversationByParticipants finds an existing conversation between two users
func (r *DMRepository) FindConversationByParticipants(userA, userB uuid.UUID) (*domain.Conversation, error) {
	var conv domain.Conversation

	// Subquery to find conversation with both participants
	subquery := r.db.Model(&domain.ConversationParticipant{}).
		Select("conversation_id").
		Where("user_id IN ?", []uuid.UUID{userA, userB}).
		Group("conversation_id").
		Having("COUNT(DISTINCT user_id) = 2")

	err := r.db.Preload("Participants").Preload("Participants.User").
		Where("id IN (?)", subquery).First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// GetUserConversations gets all conversations for a user with metadata
func (r *DMRepository) GetUserConversations(userID uuid.UUID, includeArchived bool, page, limit int) ([]domain.Conversation, int64, error) {
	var conversations []domain.Conversation
	var total int64

	// Get participant records for this user
	subquery := r.db.Model(&domain.ConversationParticipant{}).
		Select("conversation_id").
		Where("user_id = ?", userID)

	if !includeArchived {
		subquery = subquery.Where("is_archived = ?", false)
	}

	query := r.db.Model(&domain.Conversation{}).Where("id IN (?)", subquery)

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results with preloads
	offset := (page - 1) * limit
	err := r.db.Preload("Participants").Preload("Participants.User").
		Where("id IN (?)", subquery).
		Order("COALESCE(last_message_at, created_at) DESC").
		Offset(offset).Limit(limit).
		Find(&conversations).Error

	if err != nil {
		return nil, 0, err
	}

	return conversations, total, nil
}

// UpdateConversationLastMessage updates the last message info
func (r *DMRepository) UpdateConversationLastMessage(convID uuid.UUID, preview string, timestamp time.Time) error {
	return r.db.Model(&domain.Conversation{}).
		Where("id = ?", convID).
		Updates(map[string]interface{}{
			"last_message_at":      timestamp,
			"last_message_preview": preview,
			"updated_at":           time.Now(),
		}).Error
}

// ============================================================================
// PARTICIPANT METHODS
// ============================================================================

// AddParticipant adds a user to a conversation
func (r *DMRepository) AddParticipant(participant *domain.ConversationParticipant) error {
	return r.db.Create(participant).Error
}

// GetParticipant gets a participant record
func (r *DMRepository) GetParticipant(convID, userID uuid.UUID) (*domain.ConversationParticipant, error) {
	var participant domain.ConversationParticipant
	err := r.db.Where("conversation_id = ? AND user_id = ?", convID, userID).First(&participant).Error
	if err != nil {
		return nil, err
	}
	return &participant, nil
}

// UpdateParticipantUnreadCount updates the unread count for a participant
func (r *DMRepository) UpdateParticipantUnreadCount(convID, userID uuid.UUID, count int) error {
	return r.db.Model(&domain.ConversationParticipant{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Update("unread_count", count).Error
}

// IncrementUnreadCount increments unread count for participants except sender
func (r *DMRepository) IncrementUnreadCount(convID, senderID uuid.UUID) error {
	return r.db.Model(&domain.ConversationParticipant{}).
		Where("conversation_id = ? AND user_id != ?", convID, senderID).
		UpdateColumn("unread_count", gorm.Expr("unread_count + 1")).Error
}

// MarkConversationAsRead marks all messages as read for a user
func (r *DMRepository) MarkConversationAsRead(convID, userID uuid.UUID) error {
	return r.db.Model(&domain.ConversationParticipant{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Updates(map[string]interface{}{
			"unread_count": 0,
			"last_read_at": time.Now(),
		}).Error
}

// ArchiveConversation archives/unarchives a conversation for a user
func (r *DMRepository) ArchiveConversation(convID, userID uuid.UUID, archive bool) error {
	return r.db.Model(&domain.ConversationParticipant{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Update("is_archived", archive).Error
}

// MuteConversation mutes/unmutes a conversation for a user
func (r *DMRepository) MuteConversation(convID, userID uuid.UUID, mute bool) error {
	return r.db.Model(&domain.ConversationParticipant{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Update("is_muted", mute).Error
}

// IsUserInConversation checks if a user is part of a conversation
func (r *DMRepository) IsUserInConversation(convID, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&domain.ConversationParticipant{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Count(&count).Error
	return count > 0, err
}

// ============================================================================
// MESSAGE METHODS
// ============================================================================

// CreateMessage creates a new message
func (r *DMRepository) CreateMessage(msg *domain.Message) error {
	return r.db.Create(msg).Error
}

// FindMessageByID finds a message by ID
func (r *DMRepository) FindMessageByID(id uuid.UUID) (*domain.Message, error) {
	var msg domain.Message
	err := r.db.Preload("Sender").Preload("Reactions").Preload("Reactions.User").
		Where("id = ? AND deleted_at IS NULL", id).First(&msg).Error
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetMessages gets messages for a conversation with pagination
func (r *DMRepository) GetMessages(convID uuid.UUID, page, limit int) ([]domain.Message, int64, error) {
	var messages []domain.Message
	var total int64

	query := r.db.Model(&domain.Message{}).
		Where("conversation_id = ? AND deleted_at IS NULL", convID)

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results (newest first)
	offset := (page - 1) * limit
	err := r.db.Preload("Sender").Preload("Reactions").Preload("Reactions.User").Preload("ReplyTo").
		Where("conversation_id = ? AND deleted_at IS NULL", convID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&messages).Error

	if err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

// DeleteMessage soft deletes a message
func (r *DMRepository) DeleteMessage(id uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&domain.Message{}).
		Where("id = ?", id).
		Update("deleted_at", &now).Error
}

// ============================================================================
// REACTION METHODS
// ============================================================================

// AddReaction adds a reaction to a message
func (r *DMRepository) AddReaction(reaction *domain.MessageReaction) error {
	return r.db.Create(reaction).Error
}

// RemoveReaction removes a reaction from a message
func (r *DMRepository) RemoveReaction(messageID, userID uuid.UUID, emoji string) error {
	return r.db.Where("message_id = ? AND user_id = ? AND emoji = ?", messageID, userID, emoji).
		Delete(&domain.MessageReaction{}).Error
}

// GetReactionByUser gets a user's reaction on a message
func (r *DMRepository) GetReactionByUser(messageID, userID uuid.UUID) (*domain.MessageReaction, error) {
	var reaction domain.MessageReaction
	err := r.db.Where("message_id = ? AND user_id = ?", messageID, userID).First(&reaction).Error
	if err != nil {
		return nil, err
	}
	return &reaction, nil
}

// ============================================================================
// DM SETTINGS METHODS
// ============================================================================

// GetDMSettings gets DM settings for a user, creates default if not exists
func (r *DMRepository) GetDMSettings(userID uuid.UUID) (*domain.DMSettings, error) {
	var settings domain.DMSettings
	err := r.db.Where("user_id = ?", userID).First(&settings).Error
	if err == gorm.ErrRecordNotFound {
		// Create default settings
		settings = domain.DMSettings{
			UserID:              userID,
			DMPrivacy:           domain.DMPrivacyFollowers,
			ShowReadReceipts:    true,
			ShowTypingIndicator: true,
			UpdatedAt:           time.Now(),
		}
		if err := r.db.Create(&settings).Error; err != nil {
			return nil, err
		}
		return &settings, nil
	}
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

// UpdateDMSettings updates DM settings for a user
func (r *DMRepository) UpdateDMSettings(userID uuid.UUID, settings *domain.DMSettings) error {
	return r.db.Model(&domain.DMSettings{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"dm_privacy":            settings.DMPrivacy,
			"show_read_receipts":    settings.ShowReadReceipts,
			"show_typing_indicator": settings.ShowTypingIndicator,
			"updated_at":            time.Now(),
		}).Error
}

// ============================================================================
// BLOCK METHODS
// ============================================================================

// BlockUser blocks a user
func (r *DMRepository) BlockUser(blockerID, blockedID uuid.UUID) error {
	block := domain.UserBlock{
		BlockerID: blockerID,
		BlockedID: blockedID,
	}
	return r.db.Create(&block).Error
}

// UnblockUser unblocks a user
func (r *DMRepository) UnblockUser(blockerID, blockedID uuid.UUID) error {
	return r.db.Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).
		Delete(&domain.UserBlock{}).Error
}

// IsBlocked checks if userA blocked userB
func (r *DMRepository) IsBlocked(blockerID, blockedID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&domain.UserBlock{}).
		Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).
		Count(&count).Error
	return count > 0, err
}

// IsBlockedEither checks if either user has blocked the other
func (r *DMRepository) IsBlockedEither(userA, userB uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&domain.UserBlock{}).
		Where("(blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)",
			userA, userB, userB, userA).
		Count(&count).Error
	return count > 0, err
}

// GetBlockedUsers gets all users blocked by a user
func (r *DMRepository) GetBlockedUsers(userID uuid.UUID) ([]domain.UserBlock, error) {
	var blocks []domain.UserBlock
	err := r.db.Preload("Blocked").Where("blocker_id = ?", userID).Find(&blocks).Error
	return blocks, err
}

// ============================================================================
// CHAT STREAK METHODS
// ============================================================================

// GetOrCreateChatStreak gets or creates a chat streak between two users
func (r *DMRepository) GetOrCreateChatStreak(userA, userB uuid.UUID) (*domain.ChatStreak, error) {
	// Ensure consistent ordering
	if userA.String() > userB.String() {
		userA, userB = userB, userA
	}

	var streak domain.ChatStreak
	err := r.db.Where("user_a_id = ? AND user_b_id = ?", userA, userB).First(&streak).Error
	if err == gorm.ErrRecordNotFound {
		streak = domain.ChatStreak{
			UserAID:       userA,
			UserBID:       userB,
			CurrentStreak: 0,
			LongestStreak: 0,
		}
		if err := r.db.Create(&streak).Error; err != nil {
			return nil, err
		}
		return &streak, nil
	}
	if err != nil {
		return nil, err
	}
	return &streak, nil
}

// UpdateChatStreak updates a chat streak
func (r *DMRepository) UpdateChatStreak(streak *domain.ChatStreak) error {
	return r.db.Save(streak).Error
}

// GetUserChatStreaks gets all active streaks for a user
func (r *DMRepository) GetUserChatStreaks(userID uuid.UUID) ([]domain.ChatStreak, error) {
	var streaks []domain.ChatStreak
	err := r.db.Preload("UserA").Preload("UserB").
		Where("(user_a_id = ? OR user_b_id = ?) AND current_streak > 0", userID, userID).
		Find(&streaks).Error
	return streaks, err
}
