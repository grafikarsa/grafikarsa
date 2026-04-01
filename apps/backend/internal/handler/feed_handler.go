package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
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
	redisClient  *redis.Client
}

func NewFeedHandler(
	feedRepo *repository.FeedRepository,
	feedService *service.FeedService,
	interestRepo *repository.InterestRepository,
	userRepo *repository.UserRepository,
	redisClient *redis.Client,
) *FeedHandler {
	return &FeedHandler{
		feedRepo:     feedRepo,
		feedService:  feedService,
		interestRepo: interestRepo,
		userRepo:     userRepo,
		redisClient:  redisClient,
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

// getSmartFeed returns feed ranked by smart algorithm with Redis caching
func (h *FeedHandler) getSmartFeed(userID uuid.UUID, page, limit int) ([]dto.FeedItemDTO, int64, error) {
	ctx := context.Background()

	// Try cache first (only for first page to keep it simple)
	if page == 1 && h.redisClient != nil {
		cacheKey := fmt.Sprintf("feed:smart:%s:limit:%d", userID.String(), limit)
		cached, err := h.redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			var cachedData struct {
				Items []dto.FeedItemDTO `json:"items"`
				Total int64             `json:"total"`
			}
			if json.Unmarshal([]byte(cached), &cachedData) == nil {
				return cachedData.Items, cachedData.Total, nil
			}
		}
	}

	// Get user info for relevance calculation
	user, _ := h.userRepo.FindByID(userID)
	var userJurusanID, userKelasID *uuid.UUID
	if user != nil && user.Kelas != nil {
		userKelasID = user.KelasID
		userJurusanID = &user.Kelas.JurusanID
	}

	// Get user interest profile
	userInterest, _ := h.interestRepo.GetUserInterest(userID)

	// Get max engagement stats from cache or database
	maxLikes, maxViews := h.getMaxEngagementStats(ctx)

	// Get portfolios for ranking (optimized batch size)
	batchSize := limit * 2 // Fetch 2x for better ranking quality
	if batchSize < 30 {
		batchSize = 30 // Reasonable minimum
	}
	if batchSize > 100 {
		batchSize = 100 // Cap to prevent over-fetching
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

	// Batch query for likes
	portfolioIDs := make([]uuid.UUID, len(rankedItems))
	for i, item := range rankedItems {
		portfolioIDs[i] = item.Portfolio.ID
	}

	likedMap, _ := h.feedRepo.GetLikedPortfolioIDs(userID, portfolioIDs)

	// Convert to DTOs
	var feedItems []dto.FeedItemDTO
	for _, item := range rankedItems {
		isLiked := likedMap[item.Portfolio.ID]
		feedItems = append(feedItems, h.toFeedItemDTO(item.Portfolio, item.LikeCount, item.ViewCount, isLiked, item.RankingScore))
	}

	// Cache result for 2 minutes (only first page)
	if page == 1 && h.redisClient != nil {
		cacheKey := fmt.Sprintf("feed:smart:%s:limit:%d", userID.String(), limit)
		cacheData := struct {
			Items []dto.FeedItemDTO `json:"items"`
			Total int64             `json:"total"`
		}{
			Items: feedItems,
			Total: total,
		}
		if jsonData, err := json.Marshal(cacheData); err == nil {
			h.redisClient.Set(ctx, cacheKey, jsonData, 2*time.Minute)
		}
	}

	return feedItems, total, nil
}

// getMaxEngagementStats gets max stats from cache or database
func (h *FeedHandler) getMaxEngagementStats(ctx context.Context) (int64, int64) {
	var maxLikes, maxViews int64 = 1, 1

	if h.redisClient != nil {
		// Try to get from cache
		if val, err := h.redisClient.Get(ctx, "feed:max_likes").Int64(); err == nil && val > 0 {
			maxLikes = val
		}
		if val, err := h.redisClient.Get(ctx, "feed:max_views").Int64(); err == nil && val > 0 {
			maxViews = val
		}

		// If both found in cache, return
		if maxLikes > 1 && maxViews > 1 {
			return maxLikes, maxViews
		}
	}

	// Fallback to database
	dbMaxLikes, dbMaxViews, err := h.feedRepo.GetMaxEngagementStats()
	if err == nil {
		maxLikes = dbMaxLikes
		maxViews = dbMaxViews

		// Cache for 5 minutes
		if h.redisClient != nil {
			h.redisClient.Set(ctx, "feed:max_likes", maxLikes, 5*time.Minute)
			h.redisClient.Set(ctx, "feed:max_views", maxViews, 5*time.Minute)
		}
	}

	return maxLikes, maxViews
}

// getRecentFeed returns feed sorted by published_at with batch like queries
func (h *FeedHandler) getRecentFeed(userID *uuid.UUID, page, limit int) ([]dto.FeedItemDTO, int64, error) {
	portfolios, total, err := h.feedRepo.GetRecentFeed(page, limit)
	if err != nil {
		return nil, 0, err
	}

	var feedItems []dto.FeedItemDTO

	// If user is authenticated, batch query for likes
	if userID != nil {
		portfolioIDs := make([]uuid.UUID, len(portfolios))
		for i, p := range portfolios {
			portfolioIDs[i] = p.ID
		}

		likedMap, _ := h.feedRepo.GetLikedPortfolioIDs(*userID, portfolioIDs)

		for _, p := range portfolios {
			isLiked := likedMap[p.ID]
			feedItems = append(feedItems, h.toFeedItemDTO(&p.Portfolio, p.LikeCount, p.ViewCount, isLiked, 0))
		}
	} else {
		// Guest user - no likes
		for _, p := range portfolios {
			feedItems = append(feedItems, h.toFeedItemDTO(&p.Portfolio, p.LikeCount, p.ViewCount, false, 0))
		}
	}

	return feedItems, total, nil
}

// getFollowingFeed returns feed from followed users with batch like queries
func (h *FeedHandler) getFollowingFeed(userID uuid.UUID, page, limit int) ([]dto.FeedItemDTO, int64, error) {
	portfolios, total, err := h.feedRepo.GetFollowingFeed(userID, page, limit)
	if err != nil {
		return nil, 0, err
	}

	// Batch query for likes
	portfolioIDs := make([]uuid.UUID, len(portfolios))
	for i, p := range portfolios {
		portfolioIDs[i] = p.ID
	}

	likedMap, _ := h.feedRepo.GetLikedPortfolioIDs(userID, portfolioIDs)

	var feedItems []dto.FeedItemDTO
	for _, p := range portfolios {
		isLiked := likedMap[p.ID]
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
