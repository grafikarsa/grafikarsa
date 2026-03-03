package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/middleware"
	"github.com/grafikarsa/backend/internal/repository"
	"github.com/grafikarsa/backend/internal/service"
)

type FeedHandler struct {
	feedRepo     *repository.FeedRepository
	feedService  *service.FeedService
	interestRepo *repository.InterestRepository
	userRepo     *repository.UserRepository
}

func NewFeedHandler(
	feedRepo *repository.FeedRepository,
	feedService *service.FeedService,
	interestRepo *repository.InterestRepository,
	userRepo *repository.UserRepository,
) *FeedHandler {
	return &FeedHandler{
		feedRepo:     feedRepo,
		feedService:  feedService,
		interestRepo: interestRepo,
		userRepo:     userRepo,
	}
}

// GetFeed handles GET /api/v1/feed
// Query params: algorithm (smart|recent|following), page, limit
func (h *FeedHandler) GetFeed(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	algorithm := c.Query("algorithm", "smart")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	// Validate algorithm
	if algorithm != "smart" && algorithm != "recent" && algorithm != "following" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"INVALID_ALGORITHM", "Algorithm harus salah satu dari: smart, recent, following",
		))
	}

	// Following and smart require authentication
	if (algorithm == "following" || algorithm == "smart") && userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"UNAUTHORIZED", "User tidak terautentikasi",
		))
	}

	var feedItems []dto.FeedItemDTO
	var total int64
	var err error

	switch algorithm {
	case "recent":
		feedItems, total, err = h.getRecentFeed(userID, page, limit)
	case "following":
		feedItems, total, err = h.getFollowingFeed(*userID, page, limit)
	case "smart":
		feedItems, total, err = h.getSmartFeed(*userID, page, limit)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal mengambil feed",
		))
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return c.JSON(dto.SuccessWithMeta(feedItems, &dto.Meta{
		CurrentPage: page,
		PerPage:     limit,
		TotalPages:  totalPages,
		TotalCount:  total,
	}))
}

// getSmartFeed returns feed ranked by smart algorithm
func (h *FeedHandler) getSmartFeed(userID uuid.UUID, page, limit int) ([]dto.FeedItemDTO, int64, error) {
	// Get user info for relevance calculation
	user, _ := h.userRepo.FindByID(userID)
	var userJurusanID, userKelasID *uuid.UUID
	if user != nil && user.Kelas != nil {
		userKelasID = user.KelasID
		userJurusanID = &user.Kelas.JurusanID
	}

	// Get user interest profile
	userInterest, _ := h.interestRepo.GetUserInterest(userID)

	// Get max engagement stats for normalization
	maxLikes, maxViews, _ := h.feedRepo.GetMaxEngagementStats()

	// Get portfolios for ranking (fetch more than needed for better ranking)
	batchSize := limit * 5 // Fetch 5x to have enough for ranking
	if batchSize < 100 {
		batchSize = 100
	}
	portfolios, err := h.feedRepo.GetPortfoliosForSmartFeed(userID, batchSize)
	if err != nil {
		return nil, 0, err
	}

	// Calculate rankings and get paginated results
	rankedItems, total := h.feedService.GetSmartFeed(
		userID,
		userInterest,
		userJurusanID,
		userKelasID,
		portfolios,
		maxLikes,
		maxViews,
		page,
		limit,
	)

	// Convert to DTOs
	var feedItems []dto.FeedItemDTO
	for _, item := range rankedItems {
		feedItems = append(feedItems, h.toFeedItemDTO(item.Portfolio, item.LikeCount, item.ViewCount, item.IsLiked, item.RankingScore))
	}

	return feedItems, total, nil
}

// getRecentFeed returns feed sorted by published_at
func (h *FeedHandler) getRecentFeed(userID *uuid.UUID, page, limit int) ([]dto.FeedItemDTO, int64, error) {
	portfolios, total, err := h.feedRepo.GetRecentFeed(page, limit)
	if err != nil {
		return nil, 0, err
	}

	var feedItems []dto.FeedItemDTO
	for _, p := range portfolios {
		isLiked := false
		if userID != nil {
			isLiked, _ = h.feedRepo.IsLikedByUser(*userID, p.ID)
		}
		feedItems = append(feedItems, h.toFeedItemDTO(&p.Portfolio, p.LikeCount, p.ViewCount, isLiked, 0))
	}

	return feedItems, total, nil
}

// getFollowingFeed returns feed from followed users
func (h *FeedHandler) getFollowingFeed(userID uuid.UUID, page, limit int) ([]dto.FeedItemDTO, int64, error) {
	portfolios, total, err := h.feedRepo.GetFollowingFeed(userID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	var feedItems []dto.FeedItemDTO
	for _, p := range portfolios {
		isLiked, _ := h.feedRepo.IsLikedByUser(userID, p.ID)
		feedItems = append(feedItems, h.toFeedItemDTO(&p.Portfolio, p.LikeCount, p.ViewCount, isLiked, 0))
	}

	return feedItems, total, nil
}

// toFeedItemDTO converts portfolio to FeedItemDTO
func (h *FeedHandler) toFeedItemDTO(p *domain.Portfolio, likeCount, viewCount int64, isLiked bool, rankingScore float64) dto.FeedItemDTO {
	item := dto.FeedItemDTO{
		ID:           p.ID,
		Judul:        p.Judul,
		Slug:         p.Slug,
		ThumbnailURL: p.ThumbnailURL,
		PublishedAt:  p.PublishedAt,
		CreatedAt:    p.CreatedAt,
		LikeCount:    likeCount,
		ViewCount:    viewCount,
		IsLiked:      isLiked,
		RankingScore: rankingScore,
	}

	// Extract preview text from first text block
	item.PreviewText = h.extractPreviewText(p)

	// User info
	if p.User != nil {
		var kelasNama *string
		if p.User.Kelas != nil {
			kelasNama = &p.User.Kelas.Nama
		}
		item.User = &dto.FeedUserDTO{
			ID:        p.User.ID,
			Username:  p.User.Username,
			Nama:      p.User.Nama,
			AvatarURL: p.User.AvatarURL,
			Role:      string(p.User.Role),
			KelasNama: kelasNama,
		}
	}

	// Tags
	for _, t := range p.Tags {
		item.Tags = append(item.Tags, dto.TagDTO{ID: t.ID, Nama: t.Nama})
	}

	return item
}

// extractPreviewText extracts preview text from first text block (max 280 chars)
func (h *FeedHandler) extractPreviewText(p *domain.Portfolio) *string {
	if p == nil || len(p.ContentBlocks) == 0 {
		return nil
	}

	for _, block := range p.ContentBlocks {
		if block.BlockType == domain.BlockText {
			if content, ok := block.Payload["content"].(string); ok && content != "" {
				// Strip HTML tags (simple approach)
				text := stripHTMLTags(content)
				// Truncate to 280 chars
				if len(text) > 280 {
					text = text[:277] + "..."
				}
				return &text
			}
		}
	}

	return nil
}

// stripHTMLTags removes HTML tags from string (simple implementation)
func stripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false

	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}

	return strings.TrimSpace(result.String())
}

// GetFeedPreferences handles GET /api/v1/feed/preferences
func (h *FeedHandler) GetFeedPreferences(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"UNAUTHORIZED", "User tidak terautentikasi",
		))
	}

	algorithm, err := h.interestRepo.GetFeedPreference(*userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal mengambil preferensi feed",
		))
	}

	return c.JSON(dto.SuccessResponse(dto.FeedPreferenceDTO{
		Algorithm: string(algorithm),
	}, ""))
}

// UpdateFeedPreferences handles PUT /api/v1/feed/preferences
func (h *FeedHandler) UpdateFeedPreferences(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"UNAUTHORIZED", "User tidak terautentikasi",
		))
	}

	var req dto.UpdateFeedPreferenceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Request body tidak valid",
		))
	}

	// Validate algorithm
	if req.Algorithm != "smart" && req.Algorithm != "recent" && req.Algorithm != "following" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"INVALID_ALGORITHM", "Algorithm harus salah satu dari: smart, recent, following",
		))
	}

	err := h.interestRepo.SaveFeedPreference(*userID, domain.FeedAlgorithm(req.Algorithm))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal menyimpan preferensi feed",
		))
	}

	return c.JSON(dto.SuccessResponse(dto.FeedPreferenceDTO{
		Algorithm: req.Algorithm,
	}, "Preferensi feed berhasil disimpan"))
}
