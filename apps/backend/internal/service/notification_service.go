package service

import (
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/grafikarsa/backend/internal/repository"
)

type NotificationService struct {
	repo *repository.NotificationRepository
}

func NewNotificationService(repo *repository.NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

// NotifyNewFollower creates notification when someone follows a user
func (s *NotificationService) NotifyNewFollower(follower *domain.User, followedUserID uuid.UUID) error {
	notification := &domain.Notification{
		UserID:  followedUserID,
		Type:    domain.NotifNewFollower,
		Title:   "Pengikut Baru",
		Message: strPtr("@" + follower.Username + " mulai mengikuti kamu"),
		Data: domain.JSONB{
			"follower_id":       follower.ID.String(),
			"follower_username": follower.Username,
			"follower_nama":     follower.Nama,
			"follower_avatar":   follower.AvatarURL,
		},
	}
	return s.repo.Create(notification)
}

// NotifyPortfolioLiked creates notification when someone likes a portfolio
func (s *NotificationService) NotifyPortfolioLiked(liker *domain.User, portfolio *domain.Portfolio) error {
	// Don't notify if user likes their own portfolio
	if liker.ID == portfolio.UserID {
		return nil
	}

	notification := &domain.Notification{
		UserID:  portfolio.UserID,
		Type:    domain.NotifPortfolioLiked,
		Title:   "Portfolio Disukai",
		Message: strPtr("@" + liker.Username + " menyukai portfolio \"" + portfolio.Judul + "\""),
		Data: domain.JSONB{
			"liker_id":        liker.ID.String(),
			"liker_username":  liker.Username,
			"liker_nama":      liker.Nama,
			"liker_avatar":    liker.AvatarURL,
			"portfolio_id":    portfolio.ID.String(),
			"portfolio_judul": portfolio.Judul,
			"portfolio_slug":  portfolio.Slug,
		},
	}
	return s.repo.Create(notification)
}

// NotifyPortfolioApproved creates notification when portfolio is approved
func (s *NotificationService) NotifyPortfolioApproved(portfolio *domain.Portfolio) error {
	notification := &domain.Notification{
		UserID:  portfolio.UserID,
		Type:    domain.NotifPortfolioApproved,
		Title:   "Portfolio Dipublikasikan",
		Message: strPtr("Portfolio \"" + portfolio.Judul + "\" telah disetujui dan dipublikasikan"),
		Data: domain.JSONB{
			"portfolio_id":    portfolio.ID.String(),
			"portfolio_judul": portfolio.Judul,
			"portfolio_slug":  portfolio.Slug,
		},
	}
	return s.repo.Create(notification)
}

// NotifyPortfolioRejected creates notification when portfolio is rejected
func (s *NotificationService) NotifyPortfolioRejected(portfolio *domain.Portfolio, note string) error {
	notification := &domain.Notification{
		UserID:  portfolio.UserID,
		Type:    domain.NotifPortfolioRejected,
		Title:   "Portfolio Perlu Diperbaiki",
		Message: strPtr("Portfolio \"" + portfolio.Judul + "\" perlu diperbaiki"),
		Data: domain.JSONB{
			"portfolio_id":    portfolio.ID.String(),
			"portfolio_judul": portfolio.Judul,
			"portfolio_slug":  portfolio.Slug,
			"admin_note":      note,
		},
	}
	return s.repo.Create(notification)
}

// NotifyFeedbackStatusUpdated creates notification when feedback status is updated
func (s *NotificationService) NotifyFeedbackStatusUpdated(feedback *domain.Feedback, actor *domain.User, actorRole string, oldStatus, newStatus domain.FeedbackStatus) error {
	if feedback.UserID == nil {
		return nil // Don't notify generic/anonymous feedback
	}

	// Format status for display
	statusMap := map[domain.FeedbackStatus]string{
		domain.FeedbackStatusPending:  "Pending",
		domain.FeedbackStatusRead:     "Dibaca",
		domain.FeedbackStatusResolved: "Selesai",
	}

	title := "Status Feedback Diperbarui"
	message := "@" + actor.Username + " mengubah status feedback kamu menjadi " + statusMap[newStatus]

	data := domain.JSONB{
		"feedback_id":    feedback.ID.String(),
		"actor_id":       actor.ID.String(),
		"actor_username": actor.Username,
		"actor_nama":     actor.Nama,
		"actor_role":     actorRole,
		"old_status":     string(oldStatus),
		"new_status":     string(newStatus),
	}

	if feedback.AdminNotes != nil && *feedback.AdminNotes != "" {
		data["admin_note"] = *feedback.AdminNotes
	}

	notification := &domain.Notification{
		UserID:  *feedback.UserID,
		Type:    domain.NotifFeedbackUpdated,
		Title:   title,
		Message: &message,
		Data:    data,
	}
	return s.repo.Create(notification)
}

// NotifyNewComment creates notification when someone comments on a portfolio
func (s *NotificationService) NotifyNewComment(portfolio *domain.Portfolio, commenter *domain.User, comment *domain.Comment) error {
	notification := &domain.Notification{
		UserID:  portfolio.UserID,
		Type:    domain.NotifNewComment,
		Title:   "Komentar Baru di Portfolio",
		Message: strPtr("@" + commenter.Username + " berkomentar di \"" + portfolio.Judul + "\""),
		Data: domain.JSONB{
			"actor_id":       commenter.ID.String(),
			"actor_username": commenter.Username,
			"actor_nama":     commenter.Nama,
			"actor_avatar":   commenter.AvatarURL,
			"portfolio_id":   portfolio.ID.String(),
			"portfolio_slug": portfolio.Slug,
			// Assuming owner username is passed or loaded. If portfolio.User is nil, this might panic.
			// Check caller to ensure Portfolio is loaded with User.
			"portfolio_owner_username": portfolio.User.Username,
			"portfolio_judul":          portfolio.Judul,
			"comment_id":               comment.ID.String(),
		},
	}
	return s.repo.Create(notification)
}

// NotifyReplyComment creates notification when someone replies to a comment
func (s *NotificationService) NotifyReplyComment(parentComment *domain.Comment, portfolio *domain.Portfolio, replier *domain.User, reply *domain.Comment) error {
	notification := &domain.Notification{
		UserID:  parentComment.UserID,
		Type:    domain.NotifReplyComment,
		Title:   "Balasan Komentar Baru",
		Message: strPtr("@" + replier.Username + " membalas komentar Anda"),
		Data: domain.JSONB{
			"actor_id":                 replier.ID.String(),
			"actor_username":           replier.Username,
			"actor_nama":               replier.Nama,
			"actor_avatar":             replier.AvatarURL,
			"portfolio_id":             portfolio.ID.String(),
			"portfolio_slug":           portfolio.Slug,
			"portfolio_owner_username": portfolio.User.Username,
			"comment_id":               reply.ID.String(),
			"parent_comment_id":        parentComment.ID.String(),
		},
	}
	return s.repo.Create(notification)
}

func strPtr(s string) *string {
	return &s
}
