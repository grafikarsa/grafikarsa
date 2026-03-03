package service

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/repository"
)

type CommentService struct {
	commentRepo         *repository.CommentRepository
	userRepo            *repository.UserRepository
	portfolioRepo       *repository.PortfolioRepository
	notificationService *NotificationService
}

func NewCommentService(
	commentRepo *repository.CommentRepository,
	userRepo *repository.UserRepository,
	portfolioRepo *repository.PortfolioRepository,
	notificationRepo *repository.NotificationRepository, // Inject NotificationService instead if possible, but for now we follow pattern
) *CommentService {
	// Assuming NotificationService has a constructor that we can use or we should reuse the existing one.
	// In main.go, we will inject the actual NotificationService.
	// But here I'm constructing it or accepting it?
	// Let's change the constructor to accept NotificationService directly to avoid creating new one.
	return &CommentService{
		commentRepo:   commentRepo,
		userRepo:      userRepo,
		portfolioRepo: portfolioRepo,
	}
}

// SetNotificationService - safer way to inject circular prod dependency if needed
func (s *CommentService) SetNotificationService(ns *NotificationService) {
	s.notificationService = ns
}

func (s *CommentService) Create(userID uuid.UUID, req dto.CreateCommentRequest) (*domain.Comment, error) {
	portfolio, err := s.portfolioRepo.FindByID(req.PortfolioID)
	if err != nil {
		return nil, fmt.Errorf("portfolio not found")
	}

	comment := &domain.Comment{
		PortfolioID: req.PortfolioID,
		UserID:      userID,
		ParentID:    req.ParentID,
		Content:     req.Content,
	}

	if err := s.commentRepo.Create(comment); err != nil {
		return nil, err
	}

	// Trigger Notification
	go func() {
		// Get Portfolio Owner details for username
		portfolioOwner, err := s.userRepo.FindByID(portfolio.UserID)
		if err == nil {
			portfolio.User = portfolioOwner // Attach for NotificationService usage
		}

		// If replying to a comment
		if req.ParentID != nil {
			parentComment, err := s.commentRepo.FindByID(*req.ParentID)
			if err == nil && parentComment.UserID != userID {
				// We need current user (replier) details
				replier, _ := s.userRepo.FindByID(userID)
				if replier != nil {
					s.notificationService.NotifyReplyComment(parentComment, portfolio, replier, comment)
				}
			}
		}

		// Notify Portfolio Owner
		if portfolio.UserID != userID && (req.ParentID == nil || (req.ParentID != nil)) {
			shouldNotifyOwner := true
			if req.ParentID != nil {
				parentComment, _ := s.commentRepo.FindByID(*req.ParentID)
				if parentComment != nil && parentComment.UserID == portfolio.UserID {
					shouldNotifyOwner = false // Already notified as reply
				}
			}

			if shouldNotifyOwner {
				commenter, _ := s.userRepo.FindByID(userID)
				if commenter != nil {
					s.notificationService.NotifyNewComment(portfolio, commenter, comment)
				}
			}
		}
	}()

	return comment, nil
}

func (s *CommentService) GetByPortfolioID(portfolioID uuid.UUID) ([]dto.CommentResponse, error) {
	comments, err := s.commentRepo.GetByPortfolioID(portfolioID)
	if err != nil {
		return nil, err
	}

	// Build Tree
	return s.buildCommentTree(comments), nil
}

func (s *CommentService) buildCommentTree(comments []domain.Comment) []dto.CommentResponse {
	// Map ID -> *CommentResponse
	commentMap := make(map[uuid.UUID]*dto.CommentResponse)
	var rootComments []dto.CommentResponse

	// First pass: Create Response objects and populate map
	for _, c := range comments {
		resp := &dto.CommentResponse{
			ID:        c.ID,
			Content:   c.Content,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
			User: dto.UserBriefDTO{
				ID:        c.User.ID,
				Username:  c.User.Username,
				Nama:      c.User.Nama,
				AvatarURL: c.User.AvatarURL,
				Role:      string(c.User.Role),
			},
			Children: []*dto.CommentResponse{},
		}
		commentMap[c.ID] = resp
	}

	// Second pass: Link children using pointers
	for _, c := range comments {
		if c.ParentID != nil {
			if parent, exists := commentMap[*c.ParentID]; exists {
				// Append the POINTER to the parent's children
				parent.Children = append(parent.Children, commentMap[c.ID])
			}
		}
	}

	// Final pass: Collect roots (values)
	for _, c := range comments {
		if c.ParentID == nil {
			if root, exists := commentMap[c.ID]; exists {
				rootComments = append(rootComments, *root)
			}
		}
	}

	return rootComments
}

func (s *CommentService) Delete(userID uuid.UUID, commentID uuid.UUID, isAdmin bool) error {
	comment, err := s.commentRepo.FindByID(commentID)
	if err != nil {
		return err
	}

	if comment.UserID != userID && !isAdmin {
		// Check if portfolio owner
		portfolio, err := s.portfolioRepo.FindByID(comment.PortfolioID)
		if err != nil || portfolio.UserID != userID {
			return fmt.Errorf("unauthorized")
		}
	}

	return s.commentRepo.Delete(commentID)
}
