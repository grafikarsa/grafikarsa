package repository

import (
	"testing"

	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupInterestTestDB creates an in-memory SQLite database for testing
func setupInterestTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate the schema
	err = db.AutoMigrate(&domain.UserInterest{}, &domain.UserFeedPreference{})
	require.NoError(t, err)

	return db
}

// Property 10: User interest updates on like
// For any like action on a portfolio, the user's interest profile should have incremented counters
func TestProperty10_UserInterestUpdatesOnLike(t *testing.T) {
	db := setupInterestTestDB(t)
	repo := NewInterestRepository(db)

	userID := uuid.New()
	tagID1 := uuid.New()
	tagID2 := uuid.New()

	// Initial state - no interest profile
	interest, err := repo.GetUserInterest(userID)
	require.NoError(t, err)
	assert.Nil(t, interest, "Initial interest should be nil")

	// Update tag interest (simulating a like)
	err = repo.UpdateTagInterest(userID, []uuid.UUID{tagID1, tagID2})
	require.NoError(t, err)

	// Verify interest profile was created and tags were incremented
	interest, err = repo.GetUserInterest(userID)
	require.NoError(t, err)
	require.NotNil(t, interest, "Interest profile should be created")

	// Check tag counts
	tag1Count, ok := interest.LikedTags[tagID1.String()]
	assert.True(t, ok, "Tag1 should exist in liked_tags")
	assert.Equal(t, float64(1), tag1Count, "Tag1 count should be 1")

	tag2Count, ok := interest.LikedTags[tagID2.String()]
	assert.True(t, ok, "Tag2 should exist in liked_tags")
	assert.Equal(t, float64(1), tag2Count, "Tag2 count should be 1")
}

// Property 10 (continued): Multiple likes increment counters correctly
func TestProperty10_MultipleLikesIncrementCounters(t *testing.T) {
	db := setupInterestTestDB(t)
	repo := NewInterestRepository(db)

	userID := uuid.New()
	tagID := uuid.New()

	// Like 3 portfolios with the same tag
	for i := 0; i < 3; i++ {
		err := repo.UpdateTagInterest(userID, []uuid.UUID{tagID})
		require.NoError(t, err)
	}

	// Verify count is 3
	interest, err := repo.GetUserInterest(userID)
	require.NoError(t, err)
	require.NotNil(t, interest)

	tagCount, ok := interest.LikedTags[tagID.String()]
	assert.True(t, ok, "Tag should exist")
	assert.Equal(t, float64(3), tagCount, "Tag count should be 3 after 3 likes")
}

// Test jurusan interest updates
func TestUpdateJurusanInterest(t *testing.T) {
	db := setupInterestTestDB(t)
	repo := NewInterestRepository(db)

	userID := uuid.New()
	jurusanID := uuid.New()

	// Update jurusan interest
	err := repo.UpdateJurusanInterest(userID, jurusanID)
	require.NoError(t, err)

	// Verify
	interest, err := repo.GetUserInterest(userID)
	require.NoError(t, err)
	require.NotNil(t, interest)

	jurusanCount, ok := interest.LikedJurusan[jurusanID.String()]
	assert.True(t, ok, "Jurusan should exist")
	assert.Equal(t, float64(1), jurusanCount, "Jurusan count should be 1")
}

// Test decrement on unlike
func TestDecrementTagInterest(t *testing.T) {
	db := setupInterestTestDB(t)
	repo := NewInterestRepository(db)

	userID := uuid.New()
	tagID := uuid.New()

	// Like twice
	repo.UpdateTagInterest(userID, []uuid.UUID{tagID})
	repo.UpdateTagInterest(userID, []uuid.UUID{tagID})

	// Unlike once
	err := repo.DecrementTagInterest(userID, []uuid.UUID{tagID})
	require.NoError(t, err)

	// Verify count is 1
	interest, err := repo.GetUserInterest(userID)
	require.NoError(t, err)

	tagCount := interest.LikedTags[tagID.String()]
	assert.Equal(t, float64(1), tagCount, "Tag count should be 1 after decrement")
}

// Test GetTopTagInterests
func TestGetTopTagInterests(t *testing.T) {
	db := setupInterestTestDB(t)
	repo := NewInterestRepository(db)

	userID := uuid.New()
	tagID1 := uuid.New()
	tagID2 := uuid.New()
	tagID3 := uuid.New()

	// Create varying interest levels
	// Tag1: 5 likes
	for i := 0; i < 5; i++ {
		repo.UpdateTagInterest(userID, []uuid.UUID{tagID1})
	}
	// Tag2: 3 likes
	for i := 0; i < 3; i++ {
		repo.UpdateTagInterest(userID, []uuid.UUID{tagID2})
	}
	// Tag3: 1 like
	repo.UpdateTagInterest(userID, []uuid.UUID{tagID3})

	// Get top 2
	topTags, err := repo.GetTopTagInterests(userID, 2)
	require.NoError(t, err)
	require.Len(t, topTags, 2)

	// Tag1 should be first (most likes)
	assert.Equal(t, tagID1, topTags[0], "Tag1 should be first (5 likes)")
	// Tag2 should be second
	assert.Equal(t, tagID2, topTags[1], "Tag2 should be second (3 likes)")
}

// Test feed preference
func TestFeedPreference(t *testing.T) {
	db := setupInterestTestDB(t)
	repo := NewInterestRepository(db)

	userID := uuid.New()

	// Default should be smart
	pref, err := repo.GetFeedPreference(userID)
	require.NoError(t, err)
	assert.Equal(t, domain.FeedAlgorithmSmart, pref, "Default should be smart")

	// Save preference
	err = repo.SaveFeedPreference(userID, domain.FeedAlgorithmRecent)
	require.NoError(t, err)

	// Get preference
	pref, err = repo.GetFeedPreference(userID)
	require.NoError(t, err)
	assert.Equal(t, domain.FeedAlgorithmRecent, pref, "Should return saved preference")

	// Update preference
	err = repo.SaveFeedPreference(userID, domain.FeedAlgorithmFollowing)
	require.NoError(t, err)

	pref, err = repo.GetFeedPreference(userID)
	require.NoError(t, err)
	assert.Equal(t, domain.FeedAlgorithmFollowing, pref, "Should return updated preference")
}

// Test empty interest profile
func TestEmptyInterestProfile(t *testing.T) {
	db := setupInterestTestDB(t)
	repo := NewInterestRepository(db)

	userID := uuid.New()

	// Get or create
	interest, err := repo.GetOrCreateUserInterest(userID)
	require.NoError(t, err)
	require.NotNil(t, interest)

	assert.Equal(t, userID, interest.UserID)
	assert.Equal(t, 0, interest.TotalLikes)
	assert.NotNil(t, interest.LikedTags)
	assert.NotNil(t, interest.LikedJurusan)
}
