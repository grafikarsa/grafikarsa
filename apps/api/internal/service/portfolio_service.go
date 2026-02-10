package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"grafikarsa/internal/domain"
	"grafikarsa/internal/repository"
	"grafikarsa/internal/utils"
)

// PortfolioService handles portfolio business logic
type PortfolioService struct {
	portfolioRepo *repository.PortfolioRepository
	tagRepo       *repository.TagRepository
	userRepo      *repository.UserRepository
}

// NewPortfolioService creates a new PortfolioService
func NewPortfolioService(
	portfolioRepo *repository.PortfolioRepository,
	tagRepo *repository.TagRepository,
	userRepo *repository.UserRepository,
) *PortfolioService {
	return &PortfolioService{
		portfolioRepo: portfolioRepo,
		tagRepo:       tagRepo,
		userRepo:      userRepo,
	}
}

// CreatePortfolioInput contains input for creating a portfolio
type CreatePortfolioInput struct {
	Title  string
	TagIDs []uuid.UUID
}

// CreatePortfolio creates a new portfolio with initial version
func (s *PortfolioService) CreatePortfolio(ctx context.Context, userID uuid.UUID, input CreatePortfolioInput) (*domain.PortfolioListItem, error) {
	// Check rate limit (10 per day)
	count, err := s.portfolioRepo.CountUserPortfoliosToday(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}
	if count >= 10 {
		return nil, ErrRateLimitExceeded
	}

	// Generate slug
	baseSlug := utils.GenerateSlug(input.Title)
	slug := baseSlug

	// Check for uniqueness and append counter if needed
	counter := 1
	for {
		exists, _ := s.portfolioRepo.SlugExists(ctx, userID, slug, nil)
		if !exists {
			break
		}
		counter++
		slug = utils.UniqueSlug(baseSlug, fmt.Sprintf("%d", counter))
	}

	now := time.Now()
	portfolioID := uuid.New()
	versionID := uuid.New()

	portfolio := &domain.Portfolio{
		ID:        portfolioID,
		UserID:    userID,
		Slug:      slug,
		CreatedAt: now,
		UpdatedAt: now,
	}

	version := &domain.PortfolioVersion{
		ID:          versionID,
		PortfolioID: portfolioID,
		Title:       input.Title,
		Status:      domain.StatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Create portfolio and version
	if err := s.portfolioRepo.Create(ctx, portfolio, version); err != nil {
		return nil, fmt.Errorf("failed to create portfolio: %w", err)
	}

	// Set tags
	if len(input.TagIDs) > 0 {
		if err := s.portfolioRepo.SetPortfolioTags(ctx, portfolioID, input.TagIDs); err != nil {
			// Non-fatal, continue
		}
	}

	// Get tags for response
	tags, _ := s.portfolioRepo.GetPortfolioTags(ctx, portfolioID)

	return &domain.PortfolioListItem{
		ID:        portfolioID,
		Title:     input.Title,
		Slug:      slug,
		Status:    domain.StatusDraft,
		Tags:      tags,
		CreatedAt: now,
	}, nil
}

// GetPortfolio retrieves a portfolio by username and slug (public view)
func (s *PortfolioService) GetPortfolio(ctx context.Context, username, slug string, currentUserID *uuid.UUID) (*domain.PortfolioDetail, error) {
	portfolio, version, err := s.portfolioRepo.GetByUserAndSlug(ctx, username, slug)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if portfolio == nil || version == nil {
		return nil, ErrPortfolioNotFound
	}

	// Only show published portfolios to public
	if !version.Status.IsPublic() {
		return nil, ErrPortfolioNotFound
	}

	return s.buildPortfolioDetail(ctx, portfolio, version, currentUserID)
}

// GetPortfolioByID retrieves a portfolio by ID (for owner/admin editing)
func (s *PortfolioService) GetPortfolioByID(ctx context.Context, portfolioID uuid.UUID, requestingUserID uuid.UUID, isAdmin bool) (*domain.PortfolioDetail, error) {
	portfolio, version, err := s.portfolioRepo.GetByID(ctx, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if portfolio == nil || version == nil {
		return nil, ErrPortfolioNotFound
	}

	// Check ownership
	if !isAdmin && portfolio.UserID != requestingUserID {
		return nil, ErrPortfolioForbidden
	}

	return s.buildPortfolioDetail(ctx, portfolio, version, &requestingUserID)
}

func (s *PortfolioService) buildPortfolioDetail(ctx context.Context, portfolio *domain.Portfolio, version *domain.PortfolioVersion, currentUserID *uuid.UUID) (*domain.PortfolioDetail, error) {
	// Get user
	user, _ := s.portfolioRepo.GetPortfolioUser(ctx, portfolio.ID)

	// Get tags
	tags, _ := s.portfolioRepo.GetPortfolioTags(ctx, portfolio.ID)

	// Get content blocks
	blocks, _ := s.portfolioRepo.GetContentBlocks(ctx, version.ID)

	// Check if liked
	var isLiked bool
	if currentUserID != nil {
		isLiked, _ = s.portfolioRepo.HasLiked(ctx, *currentUserID, portfolio.ID)
	}

	return &domain.PortfolioDetail{
		ID:              portfolio.ID,
		Title:           version.Title,
		Slug:            portfolio.Slug,
		ThumbnailURL:    version.ThumbnailURL,
		Status:          version.Status,
		PublishedAt:     version.PublishedAt,
		CreatedAt:       portfolio.CreatedAt,
		UpdatedAt:       portfolio.UpdatedAt,
		LikeCount:       portfolio.LikeCount,
		IsLiked:         isLiked,
		User:            user,
		Tags:            tags,
		ContentBlocks:   blocks,
		AdminReviewNote: version.AdminReviewNote,
	}, nil
}

// ListPublishedPortfolios lists published portfolios
func (s *PortfolioService) ListPublishedPortfolios(ctx context.Context, filter repository.PortfolioFilter, currentUserID *uuid.UUID) ([]domain.PortfolioListItem, *utils.Meta, error) {
	portfolios, total, err := s.portfolioRepo.ListPublished(ctx, filter, currentUserID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list portfolios: %w", err)
	}

	meta := utils.NewMeta(filter.Page, filter.Limit, total)
	return portfolios, meta, nil
}

// ListUserPortfolios lists all portfolios for a user
func (s *PortfolioService) ListUserPortfolios(ctx context.Context, userID uuid.UUID, status string, page, limit int) ([]domain.PortfolioListItem, *utils.Meta, error) {
	portfolios, total, err := s.portfolioRepo.ListByUser(ctx, userID, status, page, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list portfolios: %w", err)
	}

	meta := utils.NewMeta(page, limit, total)
	return portfolios, meta, nil
}

// UpdatePortfolioInput contains update fields
type UpdatePortfolioInput struct {
	Title        *string
	ThumbnailURL *string
	TagIDs       []uuid.UUID
}

// UpdatePortfolio updates a portfolio
func (s *PortfolioService) UpdatePortfolio(ctx context.Context, portfolioID, userID uuid.UUID, isAdmin bool, input UpdatePortfolioInput) (*domain.PortfolioListItem, error) {
	portfolio, version, err := s.portfolioRepo.GetByID(ctx, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if portfolio == nil || version == nil {
		return nil, ErrPortfolioNotFound
	}

	// Check ownership
	if !isAdmin && portfolio.UserID != userID {
		return nil, ErrPortfolioForbidden
	}

	// Check if editable
	if !version.Status.CanEdit() {
		return nil, ErrPortfolioNotEditable
	}

	// Update version fields
	if input.Title != nil {
		version.Title = *input.Title

		// Update slug
		baseSlug := utils.GenerateSlug(*input.Title)
		slug := baseSlug
		counter := 1
		for {
			exists, _ := s.portfolioRepo.SlugExists(ctx, portfolio.UserID, slug, &portfolioID)
			if !exists {
				break
			}
			counter++
			slug = utils.UniqueSlug(baseSlug, fmt.Sprintf("%d", counter))
		}
		_ = s.portfolioRepo.UpdateSlug(ctx, portfolioID, slug)
	}

	if input.ThumbnailURL != nil {
		version.ThumbnailURL = *input.ThumbnailURL
	}

	if err := s.portfolioRepo.UpdateVersion(ctx, version); err != nil {
		return nil, fmt.Errorf("failed to update portfolio: %w", err)
	}

	// Update tags
	if input.TagIDs != nil {
		if err := s.portfolioRepo.SetPortfolioTags(ctx, portfolioID, input.TagIDs); err != nil {
			// Non-fatal
		}
	}

	return &domain.PortfolioListItem{
		ID:        portfolioID,
		Title:     version.Title,
		Slug:      portfolio.Slug,
		Status:    version.Status,
		UpdatedAt: time.Now(),
	}, nil
}

// SubmitPortfolio submits a portfolio for review
func (s *PortfolioService) SubmitPortfolio(ctx context.Context, portfolioID, userID uuid.UUID) error {
	portfolio, version, err := s.portfolioRepo.GetByID(ctx, portfolioID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if portfolio == nil || version == nil {
		return ErrPortfolioNotFound
	}

	// Check ownership
	if portfolio.UserID != userID {
		return ErrPortfolioForbidden
	}

	// Check if can submit
	if !version.Status.CanTransitionTo(domain.StatusPendingReview) {
		return ErrInvalidStatusTransition
	}

	// Check if portfolio is complete
	if version.ThumbnailURL == "" {
		return ErrIncompletePortfolio
	}

	blockCount, _ := s.portfolioRepo.CountContentBlocks(ctx, version.ID)
	if blockCount == 0 {
		return ErrIncompletePortfolio
	}

	// Update status
	if err := s.portfolioRepo.UpdateVersionStatus(ctx, version.ID, domain.StatusPendingReview); err != nil {
		return fmt.Errorf("failed to submit: %w", err)
	}

	return nil
}

// ArchivePortfolio archives a portfolio
func (s *PortfolioService) ArchivePortfolio(ctx context.Context, portfolioID, userID uuid.UUID, isAdmin bool) error {
	portfolio, version, err := s.portfolioRepo.GetByID(ctx, portfolioID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if portfolio == nil || version == nil {
		return ErrPortfolioNotFound
	}

	// Check ownership
	if !isAdmin && portfolio.UserID != userID {
		return ErrPortfolioForbidden
	}

	if err := s.portfolioRepo.UpdateVersionStatus(ctx, version.ID, domain.StatusArchived); err != nil {
		return fmt.Errorf("failed to archive: %w", err)
	}

	return nil
}

// UnarchivePortfolio restores an archived portfolio to draft
func (s *PortfolioService) UnarchivePortfolio(ctx context.Context, portfolioID, userID uuid.UUID, isAdmin bool) error {
	portfolio, version, err := s.portfolioRepo.GetByID(ctx, portfolioID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if portfolio == nil || version == nil {
		return ErrPortfolioNotFound
	}

	if !isAdmin && portfolio.UserID != userID {
		return ErrPortfolioForbidden
	}

	if version.Status != domain.StatusArchived {
		return ErrInvalidStatusTransition
	}

	if err := s.portfolioRepo.UpdateVersionStatus(ctx, version.ID, domain.StatusDraft); err != nil {
		return fmt.Errorf("failed to unarchive: %w", err)
	}

	return nil
}

// DeletePortfolio soft deletes a portfolio
func (s *PortfolioService) DeletePortfolio(ctx context.Context, portfolioID, userID uuid.UUID, isAdmin bool) error {
	portfolio, _, err := s.portfolioRepo.GetByID(ctx, portfolioID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if portfolio == nil {
		return ErrPortfolioNotFound
	}

	if !isAdmin && portfolio.UserID != userID {
		return ErrPortfolioForbidden
	}

	if err := s.portfolioRepo.SoftDelete(ctx, portfolioID); err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}

	return nil
}

// AddContentBlock adds a content block to a portfolio
func (s *PortfolioService) AddContentBlock(ctx context.Context, portfolioID, userID uuid.UUID, isAdmin bool, blockType domain.BlockType, blockOrder int, payload json.RawMessage) (*domain.ContentBlock, error) {
	portfolio, version, err := s.portfolioRepo.GetByID(ctx, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if portfolio == nil || version == nil {
		return nil, ErrPortfolioNotFound
	}

	if !isAdmin && portfolio.UserID != userID {
		return nil, ErrPortfolioForbidden
	}

	if !version.Status.CanEdit() {
		return nil, ErrPortfolioNotEditable
	}

	now := time.Now()
	block := &domain.ContentBlock{
		ID:                 uuid.New(),
		PortfolioVersionID: version.ID,
		BlockType:          blockType,
		BlockOrder:         blockOrder,
		Payload:            payload,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.portfolioRepo.CreateContentBlock(ctx, block); err != nil {
		return nil, fmt.Errorf("failed to create block: %w", err)
	}

	return block, nil
}

// UpdateContentBlock updates a content block
func (s *PortfolioService) UpdateContentBlock(ctx context.Context, portfolioID, blockID, userID uuid.UUID, isAdmin bool, payload json.RawMessage) (*domain.ContentBlock, error) {
	portfolio, version, err := s.portfolioRepo.GetByID(ctx, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if portfolio == nil || version == nil {
		return nil, ErrPortfolioNotFound
	}

	if !isAdmin && portfolio.UserID != userID {
		return nil, ErrPortfolioForbidden
	}

	if !version.Status.CanEdit() {
		return nil, ErrPortfolioNotEditable
	}

	block, err := s.portfolioRepo.GetContentBlockByID(ctx, blockID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if block == nil || block.PortfolioVersionID != version.ID {
		return nil, ErrBlockNotFound
	}

	if err := s.portfolioRepo.UpdateContentBlock(ctx, blockID, payload); err != nil {
		return nil, fmt.Errorf("failed to update block: %w", err)
	}

	block.Payload = payload
	block.UpdatedAt = time.Now()

	return block, nil
}

// ReorderContentBlocks reorders content blocks
func (s *PortfolioService) ReorderContentBlocks(ctx context.Context, portfolioID, userID uuid.UUID, isAdmin bool, orders []struct {
	ID    uuid.UUID `json:"id"`
	Order int       `json:"order"`
}) error {
	portfolio, version, err := s.portfolioRepo.GetByID(ctx, portfolioID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if portfolio == nil || version == nil {
		return ErrPortfolioNotFound
	}

	if !isAdmin && portfolio.UserID != userID {
		return ErrPortfolioForbidden
	}

	if !version.Status.CanEdit() {
		return ErrPortfolioNotEditable
	}

	repoOrders := make([]struct {
		ID    uuid.UUID
		Order int
	}, len(orders))
	for i, o := range orders {
		repoOrders[i] = struct {
			ID    uuid.UUID
			Order int
		}{ID: o.ID, Order: o.Order}
	}

	if err := s.portfolioRepo.ReorderContentBlocks(ctx, repoOrders); err != nil {
		return fmt.Errorf("failed to reorder: %w", err)
	}

	return nil
}

// DeleteContentBlock deletes a content block
func (s *PortfolioService) DeleteContentBlock(ctx context.Context, portfolioID, blockID, userID uuid.UUID, isAdmin bool) error {
	portfolio, version, err := s.portfolioRepo.GetByID(ctx, portfolioID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if portfolio == nil || version == nil {
		return ErrPortfolioNotFound
	}

	if !isAdmin && portfolio.UserID != userID {
		return ErrPortfolioForbidden
	}

	if !version.Status.CanEdit() {
		return ErrPortfolioNotEditable
	}

	block, _ := s.portfolioRepo.GetContentBlockByID(ctx, blockID)
	if block == nil || block.PortfolioVersionID != version.ID {
		return ErrBlockNotFound
	}

	if err := s.portfolioRepo.DeleteContentBlock(ctx, blockID); err != nil {
		return fmt.Errorf("failed to delete block: %w", err)
	}

	return nil
}

// LikePortfolio likes a portfolio
func (s *PortfolioService) LikePortfolio(ctx context.Context, portfolioID, userID uuid.UUID) (bool, int, error) {
	portfolio, version, err := s.portfolioRepo.GetByID(ctx, portfolioID)
	if err != nil {
		return false, 0, fmt.Errorf("database error: %w", err)
	}
	if portfolio == nil || version == nil || !version.Status.IsPublic() {
		return false, 0, ErrPortfolioNotFound
	}

	// Check if already liked
	liked, _ := s.portfolioRepo.HasLiked(ctx, userID, portfolioID)
	if liked {
		return false, 0, ErrAlreadyLiked
	}

	// Create like
	if err := s.portfolioRepo.CreateLike(ctx, userID, portfolioID); err != nil {
		return false, 0, fmt.Errorf("failed to like: %w", err)
	}

	// Increment count
	_ = s.portfolioRepo.IncrementLikeCount(ctx, portfolioID)

	return true, portfolio.LikeCount + 1, nil
}

// UnlikePortfolio unlikes a portfolio
func (s *PortfolioService) UnlikePortfolio(ctx context.Context, portfolioID, userID uuid.UUID) (bool, int, error) {
	portfolio, version, err := s.portfolioRepo.GetByID(ctx, portfolioID)
	if err != nil {
		return false, 0, fmt.Errorf("database error: %w", err)
	}
	if portfolio == nil || version == nil || !version.Status.IsPublic() {
		return false, 0, ErrPortfolioNotFound
	}

	// Check if liked
	liked, _ := s.portfolioRepo.HasLiked(ctx, userID, portfolioID)
	if !liked {
		return false, 0, ErrNotLiked
	}

	// Delete like
	if err := s.portfolioRepo.DeleteLike(ctx, userID, portfolioID); err != nil {
		return false, 0, fmt.Errorf("failed to unlike: %w", err)
	}

	// Decrement count
	_ = s.portfolioRepo.DecrementLikeCount(ctx, portfolioID)

	newCount := portfolio.LikeCount - 1
	if newCount < 0 {
		newCount = 0
	}

	return false, newCount, nil
}

// GetFeed returns portfolios from followed users
func (s *PortfolioService) GetFeed(ctx context.Context, userID uuid.UUID, page, limit int) ([]domain.PortfolioListItem, *utils.Meta, error) {
	portfolios, total, err := s.portfolioRepo.GetFeed(ctx, userID, page, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get feed: %w", err)
	}

	meta := utils.NewMeta(page, limit, total)
	return portfolios, meta, nil
}

// ApprovePortfolio approves a portfolio (admin)
func (s *PortfolioService) ApprovePortfolio(ctx context.Context, portfolioID, adminID uuid.UUID) error {
	_, version, err := s.portfolioRepo.GetByID(ctx, portfolioID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if version == nil {
		return ErrPortfolioNotFound
	}

	if version.Status != domain.StatusPendingReview {
		return ErrInvalidStatusTransition
	}

	if err := s.portfolioRepo.SetVersionPublished(ctx, version.ID, adminID); err != nil {
		return fmt.Errorf("failed to approve: %w", err)
	}

	return nil
}

// RejectPortfolio rejects a portfolio (admin)
func (s *PortfolioService) RejectPortfolio(ctx context.Context, portfolioID, adminID uuid.UUID, note string) error {
	_, version, err := s.portfolioRepo.GetByID(ctx, portfolioID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if version == nil {
		return ErrPortfolioNotFound
	}

	if version.Status != domain.StatusPendingReview {
		return ErrInvalidStatusTransition
	}

	if err := s.portfolioRepo.SetVersionRejected(ctx, version.ID, adminID, note); err != nil {
		return fmt.Errorf("failed to reject: %w", err)
	}

	return nil
}

// ListPendingReview lists portfolios pending review (admin)
func (s *PortfolioService) ListPendingReview(ctx context.Context, page, limit int) ([]domain.PortfolioListItem, *utils.Meta, error) {
	portfolios, total, err := s.portfolioRepo.ListPendingReview(ctx, page, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list pending: %w", err)
	}

	meta := utils.NewMeta(page, limit, total)
	return portfolios, meta, nil
}

// ListAllPortfoliosAdmin lists all portfolios for admin (all statuses)
func (s *PortfolioService) ListAllPortfoliosAdmin(ctx context.Context, filter repository.PortfolioFilter) ([]domain.PortfolioListItem, *utils.Meta, error) {
	portfolios, total, err := s.portfolioRepo.ListAllForAdmin(ctx, filter)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list portfolios: %w", err)
	}

	meta := utils.NewMeta(filter.Page, filter.Limit, total)
	return portfolios, meta, nil
}

// Portfolio service errors
var (
	ErrPortfolioNotFound       = fmt.Errorf("portfolio not found")
	ErrPortfolioForbidden      = fmt.Errorf("portfolio access forbidden")
	ErrPortfolioNotEditable    = fmt.Errorf("portfolio not editable")
	ErrRateLimitExceeded       = fmt.Errorf("rate limit exceeded")
	ErrInvalidStatusTransition = fmt.Errorf("invalid status transition")
	ErrIncompletePortfolio     = fmt.Errorf("incomplete portfolio")
	ErrBlockNotFound           = fmt.Errorf("block not found")
	ErrAlreadyLiked            = fmt.Errorf("already liked")
	ErrNotLiked                = fmt.Errorf("not liked")
)
