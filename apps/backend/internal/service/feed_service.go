package service

import (
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/grafikarsa/backend/internal/repository"
)

// FeedService handles feed generation and ranking
type FeedService struct {
	portfolioRepo *repository.PortfolioRepository
	followRepo    *repository.FollowRepository
	viewRepo      *repository.ViewRepository
	interestRepo  *repository.InterestRepository
	weights       domain.RankingWeights
}

// NewFeedService creates a new feed service with default weights
func NewFeedService(
	portfolioRepo *repository.PortfolioRepository,
	followRepo *repository.FollowRepository,
	viewRepo *repository.ViewRepository,
	interestRepo *repository.InterestRepository,
) *FeedService {
	return &FeedService{
		portfolioRepo: portfolioRepo,
		followRepo:    followRepo,
		viewRepo:      viewRepo,
		interestRepo:  interestRepo,
		weights:       domain.DefaultRankingWeights(),
	}
}

// ============================================================================
// SIGNAL CALCULATORS
// ============================================================================

// CalculateFollowingScore calculates the following signal score
// Returns 1.0 for mutual follow, 0.8 for one-way follow, 0.0 for non-followed
func (s *FeedService) CalculateFollowingScore(userID, authorID uuid.UUID) float64 {
	// Check if user follows author
	isFollowing, err := s.followRepo.IsFollowing(userID, authorID)
	if err != nil || !isFollowing {
		return 0.0
	}

	// Check if it's mutual (author also follows user)
	isMutual, err := s.followRepo.IsFollowing(authorID, userID)
	if err != nil {
		return 0.8 // One-way follow
	}

	if isMutual {
		return 1.0 // Mutual follow gets full score
	}
	return 0.8 // One-way follow
}

// CalculateRecencyScore calculates the recency signal score based on publish time
// Returns 1.0 for <24h, 0.8 for <7d, 0.5 for <30d, 0.2 for older
func CalculateRecencyScore(publishedAt *time.Time) float64 {
	if publishedAt == nil {
		return 0.2 // Unpublished or no date
	}

	age := time.Since(*publishedAt)

	switch {
	case age <= 24*time.Hour:
		return 1.0
	case age <= 7*24*time.Hour:
		return 0.8
	case age <= 30*24*time.Hour:
		return 0.5
	default:
		return 0.2
	}
}

// CalculateEngagementScore calculates the engagement signal score
// Normalizes likes and views against max values, weights likes at 60%, views at 40%
func CalculateEngagementScore(likeCount, viewCount, maxLikes, maxViews int64) float64 {
	if maxLikes <= 0 {
		maxLikes = 1
	}
	if maxViews <= 0 {
		maxViews = 1
	}

	likeScore := float64(likeCount) / float64(maxLikes)
	viewScore := float64(viewCount) / float64(maxViews)

	// Clamp to 0-1 range
	likeScore = math.Min(1.0, math.Max(0.0, likeScore))
	viewScore = math.Min(1.0, math.Max(0.0, viewScore))

	// Weight: likes 60%, views 40%
	return likeScore*0.6 + viewScore*0.4
}

// CalculateRelevanceScore calculates the relevance signal score
// Based on tag matches (50%), jurusan match (30%), kelas match (20%)
func (s *FeedService) CalculateRelevanceScore(
	portfolioTagIDs []uuid.UUID,
	portfolioAuthorJurusanID *uuid.UUID,
	portfolioAuthorKelasID *uuid.UUID,
	userInterest *domain.UserInterest,
	userJurusanID *uuid.UUID,
	userKelasID *uuid.UUID,
) float64 {
	// If no user interest data, return neutral score
	if userInterest == nil {
		return 0.5
	}

	// Calculate tag match score (50% weight)
	tagScore := calculateTagMatchScore(portfolioTagIDs, userInterest.LikedTags)

	// Calculate jurusan match score (30% weight)
	jurusanScore := calculateJurusanMatchScore(portfolioAuthorJurusanID, userInterest.LikedJurusan, userJurusanID)

	// Calculate kelas match score (20% weight)
	kelasScore := calculateKelasMatchScore(portfolioAuthorKelasID, userKelasID)

	return tagScore*0.5 + jurusanScore*0.3 + kelasScore*0.2
}

// calculateTagMatchScore calculates how well portfolio tags match user interests
func calculateTagMatchScore(portfolioTagIDs []uuid.UUID, likedTags domain.JSONB) float64 {
	if len(portfolioTagIDs) == 0 || likedTags == nil || len(likedTags) == 0 {
		return 0.0
	}

	matchCount := 0
	totalInterest := 0.0

	for _, tagID := range portfolioTagIDs {
		tagKey := tagID.String()
		if val, ok := likedTags[tagKey]; ok {
			if count, ok := val.(float64); ok && count > 0 {
				matchCount++
				totalInterest += count
			}
		}
	}

	if matchCount == 0 {
		return 0.0
	}

	// Score based on match ratio and interest strength
	matchRatio := float64(matchCount) / float64(len(portfolioTagIDs))

	// Normalize interest (log scale to prevent extreme values)
	interestScore := math.Min(1.0, math.Log1p(totalInterest)/5.0)

	return (matchRatio + interestScore) / 2.0
}

// calculateJurusanMatchScore calculates jurusan relevance
func calculateJurusanMatchScore(authorJurusanID *uuid.UUID, likedJurusan domain.JSONB, userJurusanID *uuid.UUID) float64 {
	if authorJurusanID == nil {
		return 0.0
	}

	score := 0.0

	// Check if user has liked portfolios from this jurusan
	if likedJurusan != nil {
		jurusanKey := authorJurusanID.String()
		if val, ok := likedJurusan[jurusanKey]; ok {
			if count, ok := val.(float64); ok && count > 0 {
				score += math.Min(0.5, math.Log1p(count)/5.0)
			}
		}
	}

	// Bonus if same jurusan as user
	if userJurusanID != nil && *authorJurusanID == *userJurusanID {
		score += 0.5
	}

	return math.Min(1.0, score)
}

// calculateKelasMatchScore calculates kelas relevance
func calculateKelasMatchScore(authorKelasID *uuid.UUID, userKelasID *uuid.UUID) float64 {
	if authorKelasID == nil || userKelasID == nil {
		return 0.0
	}

	// Same kelas gets full score
	if *authorKelasID == *userKelasID {
		return 1.0
	}

	return 0.0
}

// CalculateQualityScore calculates the quality signal score
// Based on completeness (thumbnail + content) and assessment score if available
func CalculateQualityScore(hasThumbnail bool, contentBlockCount int, assessmentScore *float64) float64 {
	// Completeness score (50% of quality)
	completeness := 0.0
	if hasThumbnail {
		completeness += 0.5
	}
	if contentBlockCount > 0 {
		completeness += 0.5
	}

	// If assessment score available, use it (70% weight) + completeness (30% weight)
	if assessmentScore != nil && *assessmentScore > 0 {
		normalizedAssessment := *assessmentScore / 10.0 // Assuming 1-10 scale
		return normalizedAssessment*0.7 + completeness*0.3
	}

	// No assessment, use completeness only
	return completeness
}

// ============================================================================
// RANKING CALCULATOR
// ============================================================================

// CalculateRankingScore calculates the final ranking score from all signals
func (s *FeedService) CalculateRankingScore(signals domain.SignalScores) float64 {
	return signals.Calculate(s.weights)
}

// CalculateAllSignals calculates all signal scores for a portfolio
func (s *FeedService) CalculateAllSignals(
	userID uuid.UUID,
	portfolio *domain.Portfolio,
	userInterest *domain.UserInterest,
	userJurusanID *uuid.UUID,
	userKelasID *uuid.UUID,
	maxLikes, maxViews int64,
	assessmentScore *float64,
) domain.SignalScores {
	// Following signal
	followingScore := s.CalculateFollowingScore(userID, portfolio.UserID)

	// Recency signal
	recencyScore := CalculateRecencyScore(portfolio.PublishedAt)

	// Engagement signal
	likeCount, _ := s.portfolioRepo.GetLikeCount(portfolio.ID)
	viewCount, _ := s.viewRepo.GetViewCount(portfolio.ID)
	engagementScore := CalculateEngagementScore(likeCount, viewCount, maxLikes, maxViews)

	// Relevance signal
	var portfolioTagIDs []uuid.UUID
	for _, tag := range portfolio.Tags {
		portfolioTagIDs = append(portfolioTagIDs, tag.ID)
	}

	var authorJurusanID, authorKelasID *uuid.UUID
	if portfolio.User != nil {
		authorKelasID = portfolio.User.KelasID
		if portfolio.User.Kelas != nil {
			authorJurusanID = &portfolio.User.Kelas.JurusanID
		}
	}

	relevanceScore := s.CalculateRelevanceScore(
		portfolioTagIDs,
		authorJurusanID,
		authorKelasID,
		userInterest,
		userJurusanID,
		userKelasID,
	)

	// Quality signal
	hasThumbnail := portfolio.ThumbnailURL != nil && *portfolio.ThumbnailURL != ""
	qualityScore := CalculateQualityScore(hasThumbnail, len(portfolio.ContentBlocks), assessmentScore)

	return domain.SignalScores{
		Following:  followingScore,
		Recency:    recencyScore,
		Engagement: engagementScore,
		Relevance:  relevanceScore,
		Quality:    qualityScore,
	}
}

// GetWeights returns the current ranking weights
func (s *FeedService) GetWeights() domain.RankingWeights {
	return s.weights
}

// SetWeights allows customizing ranking weights
func (s *FeedService) SetWeights(weights domain.RankingWeights) {
	s.weights = weights
}

// ============================================================================
// FEED GENERATION
// ============================================================================

// RankedFeedItem represents a portfolio with its ranking score
type RankedFeedItem struct {
	Portfolio    *domain.Portfolio
	LikeCount    int64
	ViewCount    int64
	IsLiked      bool
	RankingScore float64
	SignalScores domain.SignalScores
}

// GetSmartFeed returns portfolios ranked by the smart algorithm
func (s *FeedService) GetSmartFeed(
	userID uuid.UUID,
	userInterest *domain.UserInterest,
	userJurusanID *uuid.UUID,
	userKelasID *uuid.UUID,
	portfolios []repository.FeedPortfolio,
	maxLikes, maxViews int64,
	page, limit int,
) ([]RankedFeedItem, int64) {
	// Calculate ranking scores for all portfolios
	var rankedItems []RankedFeedItem

	for i := range portfolios {
		p := &portfolios[i]

		// Calculate all signals
		signals := s.CalculateAllSignalsFromFeedPortfolio(
			userID,
			p,
			userInterest,
			userJurusanID,
			userKelasID,
			maxLikes,
			maxViews,
		)

		// Calculate final ranking score
		rankingScore := s.CalculateRankingScore(signals)

		// Check if liked by user
		isLiked, _ := s.portfolioRepo.IsLiked(userID, p.ID)

		rankedItems = append(rankedItems, RankedFeedItem{
			Portfolio:    &p.Portfolio,
			LikeCount:    p.LikeCount,
			ViewCount:    p.ViewCount,
			IsLiked:      isLiked,
			RankingScore: rankingScore,
			SignalScores: signals,
		})
	}

	// Sort by ranking score descending
	sortByRankingScore(rankedItems)

	// Apply pagination
	total := int64(len(rankedItems))
	start := (page - 1) * limit
	end := start + limit

	if start >= len(rankedItems) {
		return []RankedFeedItem{}, total
	}
	if end > len(rankedItems) {
		end = len(rankedItems)
	}

	return rankedItems[start:end], total
}

// CalculateAllSignalsFromFeedPortfolio calculates signals from FeedPortfolio struct
func (s *FeedService) CalculateAllSignalsFromFeedPortfolio(
	userID uuid.UUID,
	p *repository.FeedPortfolio,
	userInterest *domain.UserInterest,
	userJurusanID *uuid.UUID,
	userKelasID *uuid.UUID,
	maxLikes, maxViews int64,
) domain.SignalScores {
	// Following signal
	followingScore := s.CalculateFollowingScore(userID, p.UserID)

	// Recency signal
	recencyScore := CalculateRecencyScore(p.PublishedAt)

	// Engagement signal (use pre-calculated counts)
	engagementScore := CalculateEngagementScore(p.LikeCount, p.ViewCount, maxLikes, maxViews)

	// Relevance signal
	var portfolioTagIDs []uuid.UUID
	for _, tag := range p.Tags {
		portfolioTagIDs = append(portfolioTagIDs, tag.ID)
	}

	var authorJurusanID, authorKelasID *uuid.UUID
	if p.User != nil {
		authorKelasID = p.User.KelasID
		if p.User.Kelas != nil {
			authorJurusanID = &p.User.Kelas.JurusanID
		}
	}

	relevanceScore := s.CalculateRelevanceScore(
		portfolioTagIDs,
		authorJurusanID,
		authorKelasID,
		userInterest,
		userJurusanID,
		userKelasID,
	)

	// Quality signal
	hasThumbnail := p.ThumbnailURL != nil && *p.ThumbnailURL != ""
	qualityScore := CalculateQualityScore(hasThumbnail, len(p.ContentBlocks), p.AssessmentScore)

	return domain.SignalScores{
		Following:  followingScore,
		Recency:    recencyScore,
		Engagement: engagementScore,
		Relevance:  relevanceScore,
		Quality:    qualityScore,
	}
}

// sortByRankingScore sorts items by ranking score in descending order
func sortByRankingScore(items []RankedFeedItem) {
	// Simple bubble sort for now (can be optimized with sort.Slice)
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].RankingScore > items[i].RankingScore {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}
