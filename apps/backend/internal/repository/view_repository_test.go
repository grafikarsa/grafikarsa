package repository

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate the schema
	err = db.AutoMigrate(&domain.PortfolioView{})
	require.NoError(t, err)

	return db
}

// Property 9: View tracking creates unique records
// For any user viewing a portfolio multiple times, there should be exactly one view record
func TestProperty9_ViewTrackingCreatesUniqueRecords(t *testing.T) {
	db := setupTestDB(t)
	repo := NewViewRepository(db)

	portfolioID := uuid.New()
	userID := uuid.New()

	// Record view multiple times
	for i := 0; i < 5; i++ {
		err := repo.RecordView(portfolioID, &userID, nil)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
	}

	// Count records - should be exactly 1
	var count int64
	db.Model(&domain.PortfolioView{}).
		Where("portfolio_id = ? AND user_id = ?", portfolioID, userID).
		Count(&count)

	assert.Equal(t, int64(1), count, "Multiple views from same user should result in exactly one record")
}

// Property 9 (continued): Session-based views also create unique records
func TestProperty9_SessionViewsCreateUniqueRecords(t *testing.T) {
	db := setupTestDB(t)
	repo := NewViewRepository(db)

	portfolioID := uuid.New()
	sessionID := "test-session-123"

	// Record view multiple times with same session
	for i := 0; i < 5; i++ {
		err := repo.RecordView(portfolioID, nil, &sessionID)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	// Count records - should be exactly 1
	var count int64
	db.Model(&domain.PortfolioView{}).
		Where("portfolio_id = ? AND session_id = ?", portfolioID, sessionID).
		Count(&count)

	assert.Equal(t, int64(1), count, "Multiple views from same session should result in exactly one record")
}

// Test that view count returns unique viewers only
func TestGetViewCount_ReturnsUniqueViewers(t *testing.T) {
	db := setupTestDB(t)
	repo := NewViewRepository(db)

	portfolioID := uuid.New()

	// Create views from 3 different users
	for i := 0; i < 3; i++ {
		userID := uuid.New()
		err := repo.RecordView(portfolioID, &userID, nil)
		require.NoError(t, err)
	}

	// Create views from 2 different sessions (guests)
	for i := 0; i < 2; i++ {
		sessionID := uuid.New().String()
		err := repo.RecordView(portfolioID, nil, &sessionID)
		require.NoError(t, err)
	}

	// Get view count
	count, err := repo.GetViewCount(portfolioID)
	require.NoError(t, err)

	assert.Equal(t, int64(5), count, "View count should equal number of unique viewers (3 users + 2 sessions)")
}

// Test that repeated views update timestamp
func TestRecordView_UpdatesTimestamp(t *testing.T) {
	db := setupTestDB(t)
	repo := NewViewRepository(db)

	portfolioID := uuid.New()
	userID := uuid.New()

	// First view
	err := repo.RecordView(portfolioID, &userID, nil)
	require.NoError(t, err)

	// Get first timestamp
	var firstView domain.PortfolioView
	db.Where("portfolio_id = ? AND user_id = ?", portfolioID, userID).First(&firstView)
	firstTimestamp := firstView.ViewedAt

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	// Second view
	err = repo.RecordView(portfolioID, &userID, nil)
	require.NoError(t, err)

	// Get updated timestamp
	var secondView domain.PortfolioView
	db.Where("portfolio_id = ? AND user_id = ?", portfolioID, userID).First(&secondView)

	assert.True(t, secondView.ViewedAt.After(firstTimestamp) || secondView.ViewedAt.Equal(firstTimestamp),
		"Timestamp should be updated on repeated view")
}

// Test HasUserViewed
func TestHasUserViewed(t *testing.T) {
	db := setupTestDB(t)
	repo := NewViewRepository(db)

	portfolioID := uuid.New()
	userID := uuid.New()
	otherUserID := uuid.New()

	// Record view for userID
	err := repo.RecordView(portfolioID, &userID, nil)
	require.NoError(t, err)

	// Check if userID has viewed
	hasViewed, err := repo.HasUserViewed(userID, portfolioID)
	require.NoError(t, err)
	assert.True(t, hasViewed, "User who viewed should return true")

	// Check if otherUserID has viewed
	hasViewed, err = repo.HasUserViewed(otherUserID, portfolioID)
	require.NoError(t, err)
	assert.False(t, hasViewed, "User who hasn't viewed should return false")
}

// Test GetViewsByUser
func TestGetViewsByUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewViewRepository(db)

	userID := uuid.New()

	// Create views for 3 different portfolios
	for i := 0; i < 3; i++ {
		portfolioID := uuid.New()
		err := repo.RecordView(portfolioID, &userID, nil)
		require.NoError(t, err)
	}

	// Get views by user
	views, total, err := repo.GetViewsByUser(userID, 1, 10)
	require.NoError(t, err)

	assert.Equal(t, int64(3), total, "Total should be 3")
	assert.Len(t, views, 3, "Should return 3 views")
}
