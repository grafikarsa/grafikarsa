package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/grafikarsa/backend/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrCannotMessageUser    = errors.New("cannot message this user due to their privacy settings")
	ErrUserBlocked          = errors.New("you have blocked this user or they have blocked you")
	ErrNotInConversation    = errors.New("you are not a participant in this conversation")
	ErrConversationNotFound = errors.New("conversation not found")
	ErrMessageNotFound      = errors.New("message not found")
	ErrCannotDeleteMessage  = errors.New("you can only delete your own messages")
)

type DMService struct {
	dmRepo     *repository.DMRepository
	userRepo   *repository.UserRepository
	followRepo *repository.FollowRepository
}

func NewDMService(dmRepo *repository.DMRepository, userRepo *repository.UserRepository, followRepo *repository.FollowRepository) *DMService {
	return &DMService{
		dmRepo:     dmRepo,
		userRepo:   userRepo,
		followRepo: followRepo,
	}
}

// CanUserMessage checks if sender can message recipient based on privacy settings
func (s *DMService) CanUserMessage(senderID, recipientID uuid.UUID) (bool, error) {
	// Check if blocked
	blocked, err := s.dmRepo.IsBlockedEither(senderID, recipientID)
	if err != nil {
		return false, err
	}
	if blocked {
		return false, ErrUserBlocked
	}

	// Get recipient's DM settings
	settings, err := s.dmRepo.GetDMSettings(recipientID)
	if err != nil {
		return false, err
	}

	switch settings.DMPrivacy {
	case domain.DMPrivacyOpen:
		return true, nil
	case domain.DMPrivacyFollowers:
		// Check if sender follows recipient
		isFollowing, err := s.followRepo.IsFollowing(senderID, recipientID)
		if err != nil {
			return false, err
		}
		return isFollowing, nil
	case domain.DMPrivacyMutual:
		// Check if mutual follow
		isMutual, err := s.followRepo.IsMutualFollow(senderID, recipientID)
		if err != nil {
			return false, err
		}
		return isMutual, nil
	case domain.DMPrivacyClosed:
		return false, nil
	default:
		return false, nil
	}
}

// StartConversation starts a new conversation or returns existing one
func (s *DMService) StartConversation(senderID, recipientID uuid.UUID, initialMessage string) (*domain.Conversation, *domain.Message, error) {
	// Check if can message
	canMessage, err := s.CanUserMessage(senderID, recipientID)
	if err != nil {
		return nil, nil, err
	}
	if !canMessage {
		return nil, nil, ErrCannotMessageUser
	}

	// Check if conversation already exists
	existingConv, err := s.dmRepo.FindConversationByParticipants(senderID, recipientID)
	if err == nil {
		// Conversation exists
		var msg *domain.Message
		if initialMessage != "" {
			msg, err = s.SendMessage(existingConv.ID, senderID, domain.MessageTypeText, map[string]interface{}{"text": initialMessage}, nil)
			if err != nil {
				return nil, nil, err
			}
		}
		return existingConv, msg, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, nil, err
	}

	// Create new conversation
	conv := &domain.Conversation{}
	if err := s.dmRepo.CreateConversation(conv); err != nil {
		return nil, nil, err
	}

	// Add participants
	senderParticipant := &domain.ConversationParticipant{
		ConversationID: conv.ID,
		UserID:         senderID,
	}
	if err := s.dmRepo.AddParticipant(senderParticipant); err != nil {
		return nil, nil, err
	}

	recipientParticipant := &domain.ConversationParticipant{
		ConversationID: conv.ID,
		UserID:         recipientID,
	}
	if err := s.dmRepo.AddParticipant(recipientParticipant); err != nil {
		return nil, nil, err
	}

	// Send initial message if provided
	var msg *domain.Message
	if initialMessage != "" {
		msg, err = s.SendMessage(conv.ID, senderID, domain.MessageTypeText, map[string]interface{}{"text": initialMessage}, nil)
		if err != nil {
			return nil, nil, err
		}
	}

	// Reload conversation with participants
	conv, err = s.dmRepo.FindConversationByID(conv.ID)
	if err != nil {
		return nil, nil, err
	}

	return conv, msg, nil
}

// SendMessage sends a message to a conversation
func (s *DMService) SendMessage(convID, senderID uuid.UUID, msgType domain.MessageType, content map[string]interface{}, replyToID *uuid.UUID) (*domain.Message, error) {
	// Verify sender is in conversation
	isParticipant, err := s.dmRepo.IsUserInConversation(convID, senderID)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, ErrNotInConversation
	}

	// Create message
	msg := &domain.Message{
		ConversationID: convID,
		SenderID:       senderID,
		MessageType:    msgType,
		Content:        content,
		ReplyToID:      replyToID,
	}

	if err := s.dmRepo.CreateMessage(msg); err != nil {
		return nil, err
	}

	// Update conversation last message
	preview := s.getMessagePreview(msgType, content)
	if err := s.dmRepo.UpdateConversationLastMessage(convID, preview, time.Now()); err != nil {
		return nil, err
	}

	// Increment unread count for other participants
	if err := s.dmRepo.IncrementUnreadCount(convID, senderID); err != nil {
		return nil, err
	}

	// Update chat streak
	go s.updateChatStreakAsync(convID, senderID)

	// Reload message with sender
	msg, err = s.dmRepo.FindMessageByID(msg.ID)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

// getMessagePreview generates a preview text for the message
func (s *DMService) getMessagePreview(msgType domain.MessageType, content map[string]interface{}) string {
	switch msgType {
	case domain.MessageTypeText:
		if text, ok := content["text"].(string); ok {
			if len(text) > 100 {
				return text[:100] + "..."
			}
			return text
		}
	case domain.MessageTypeImage:
		return "📷 Sent an image"
	case domain.MessageTypePortfolio:
		return "📁 Shared a portfolio"
	case domain.MessageTypeSystem:
		if text, ok := content["text"].(string); ok {
			return text
		}
	}
	return ""
}

// updateChatStreakAsync updates chat streak asynchronously
func (s *DMService) updateChatStreakAsync(convID, senderID uuid.UUID) {
	conv, err := s.dmRepo.FindConversationByID(convID)
	if err != nil {
		return
	}

	// Find the other user
	var otherUserID uuid.UUID
	for _, p := range conv.Participants {
		if p.UserID != senderID {
			otherUserID = p.UserID
			break
		}
	}

	if otherUserID == uuid.Nil {
		return
	}

	streak, err := s.dmRepo.GetOrCreateChatStreak(senderID, otherUserID)
	if err != nil {
		return
	}

	today := time.Now().Truncate(24 * time.Hour)

	if streak.LastChatDate == nil {
		// First chat
		streak.CurrentStreak = 1
		streak.LongestStreak = 1
		streak.LastChatDate = &today
	} else {
		lastChat := streak.LastChatDate.Truncate(24 * time.Hour)
		diff := today.Sub(lastChat).Hours() / 24

		switch {
		case diff == 0:
			// Same day, no change
			return
		case diff == 1:
			// Consecutive day
			streak.CurrentStreak++
			if streak.CurrentStreak > streak.LongestStreak {
				streak.LongestStreak = streak.CurrentStreak
			}
			streak.LastChatDate = &today
		default:
			// Streak broken
			streak.CurrentStreak = 1
			streak.LastChatDate = &today
		}
	}

	s.dmRepo.UpdateChatStreak(streak)
}

// GetConversations gets conversations for a user
func (s *DMService) GetConversations(userID uuid.UUID, includeArchived bool, page, limit int) ([]domain.Conversation, int64, error) {
	return s.dmRepo.GetUserConversations(userID, includeArchived, page, limit)
}

// GetConversation gets a single conversation
func (s *DMService) GetConversation(convID, userID uuid.UUID) (*domain.Conversation, error) {
	// Verify user is in conversation
	isParticipant, err := s.dmRepo.IsUserInConversation(convID, userID)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, ErrNotInConversation
	}

	return s.dmRepo.FindConversationByID(convID)
}

// GetMessages gets messages for a conversation
func (s *DMService) GetMessages(convID, userID uuid.UUID, page, limit int) ([]domain.Message, int64, error) {
	// Verify user is in conversation
	isParticipant, err := s.dmRepo.IsUserInConversation(convID, userID)
	if err != nil {
		return nil, 0, err
	}
	if !isParticipant {
		return nil, 0, ErrNotInConversation
	}

	return s.dmRepo.GetMessages(convID, page, limit)
}

// DeleteMessage deletes a message
func (s *DMService) DeleteMessage(msgID, userID uuid.UUID) error {
	msg, err := s.dmRepo.FindMessageByID(msgID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrMessageNotFound
		}
		return err
	}

	// Only sender can delete
	if msg.SenderID != userID {
		return ErrCannotDeleteMessage
	}

	return s.dmRepo.DeleteMessage(msgID)
}

// MarkAsRead marks a conversation as read
func (s *DMService) MarkAsRead(convID, userID uuid.UUID) error {
	return s.dmRepo.MarkConversationAsRead(convID, userID)
}

// AddReaction adds a reaction to a message
func (s *DMService) AddReaction(msgID, userID uuid.UUID, emoji string) (*domain.MessageReaction, error) {
	msg, err := s.dmRepo.FindMessageByID(msgID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}

	// Verify user is in conversation
	isParticipant, err := s.dmRepo.IsUserInConversation(msg.ConversationID, userID)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, ErrNotInConversation
	}

	// Check if already reacted
	existing, err := s.dmRepo.GetReactionByUser(msgID, userID)
	if err == nil && existing != nil {
		// Update emoji if different
		if existing.Emoji != emoji {
			s.dmRepo.RemoveReaction(msgID, userID, existing.Emoji)
		} else {
			return existing, nil
		}
	}

	reaction := &domain.MessageReaction{
		MessageID: msgID,
		UserID:    userID,
		Emoji:     emoji,
	}

	if err := s.dmRepo.AddReaction(reaction); err != nil {
		return nil, err
	}

	return reaction, nil
}

// RemoveReaction removes a reaction from a message
func (s *DMService) RemoveReaction(msgID, userID uuid.UUID, emoji string) error {
	return s.dmRepo.RemoveReaction(msgID, userID, emoji)
}

// ArchiveConversation archives a conversation
func (s *DMService) ArchiveConversation(convID, userID uuid.UUID, archive bool) error {
	return s.dmRepo.ArchiveConversation(convID, userID, archive)
}

// MuteConversation mutes a conversation
func (s *DMService) MuteConversation(convID, userID uuid.UUID, mute bool) error {
	return s.dmRepo.MuteConversation(convID, userID, mute)
}

// GetDMSettings gets DM settings for a user
func (s *DMService) GetDMSettings(userID uuid.UUID) (*domain.DMSettings, error) {
	return s.dmRepo.GetDMSettings(userID)
}

// UpdateDMSettings updates DM settings
func (s *DMService) UpdateDMSettings(userID uuid.UUID, privacy *string, showReadReceipts, showTypingIndicator *bool) (*domain.DMSettings, error) {
	settings, err := s.dmRepo.GetDMSettings(userID)
	if err != nil {
		return nil, err
	}

	if privacy != nil {
		settings.DMPrivacy = domain.DMPrivacy(*privacy)
	}
	if showReadReceipts != nil {
		settings.ShowReadReceipts = *showReadReceipts
	}
	if showTypingIndicator != nil {
		settings.ShowTypingIndicator = *showTypingIndicator
	}

	if err := s.dmRepo.UpdateDMSettings(userID, settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// BlockUser blocks a user
func (s *DMService) BlockUser(blockerID, blockedID uuid.UUID) error {
	return s.dmRepo.BlockUser(blockerID, blockedID)
}

// UnblockUser unblocks a user
func (s *DMService) UnblockUser(blockerID, blockedID uuid.UUID) error {
	return s.dmRepo.UnblockUser(blockerID, blockedID)
}

// GetBlockedUsers gets all blocked users
func (s *DMService) GetBlockedUsers(userID uuid.UUID) ([]domain.UserBlock, error) {
	return s.dmRepo.GetBlockedUsers(userID)
}

// GetChatStreaks gets active chat streaks
func (s *DMService) GetChatStreaks(userID uuid.UUID) ([]domain.ChatStreak, error) {
	return s.dmRepo.GetUserChatStreaks(userID)
}
