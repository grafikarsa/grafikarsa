package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"grafikarsa/internal/domain"
	"grafikarsa/internal/repository"
	"grafikarsa/internal/utils"
)

// UserService handles user business logic
type UserService struct {
	userRepo   *repository.UserRepository
	followRepo *repository.FollowRepository
}

// NewUserService creates a new UserService
func NewUserService(userRepo *repository.UserRepository, followRepo *repository.FollowRepository) *UserService {
	return &UserService{
		userRepo:   userRepo,
		followRepo: followRepo,
	}
}

// GetUserByUsername retrieves a user by username with counts
func (s *UserService) GetUserByUsername(ctx context.Context, username string, currentUserID *uuid.UUID) (*domain.UserPublicProfile, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Get counts
	followers, following, portfolios, err := s.userRepo.GetUserCounts(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get counts: %w", err)
	}
	user.FollowerCount = followers
	user.FollowingCount = following
	user.PortfolioCount = portfolios

	// Check if current user follows this user
	if currentUserID != nil && *currentUserID != user.ID {
		isFollowing, _ := s.followRepo.Exists(ctx, *currentUserID, user.ID)
		user.IsFollowing = isFollowing
	}

	// Get class history
	history, _ := s.userRepo.GetClassHistory(ctx, user.ID)
	user.ClassHistory = history

	profile := user.ToPublicProfile()
	return &profile, nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Get counts
	followers, following, _, err := s.userRepo.GetUserCounts(ctx, user.ID)
	if err == nil {
		user.FollowerCount = followers
		user.FollowingCount = following
	}

	return user, nil
}

// ListUsers lists users with filtering
func (s *UserService) ListUsers(ctx context.Context, filter repository.UserFilter) ([]domain.UserListItem, *utils.Meta, error) {
	users, total, err := s.userRepo.List(ctx, filter)
	if err != nil {
		return nil, nil, fmt.Errorf("database error: %w", err)
	}

	meta := utils.NewMeta(filter.Page, filter.Limit, total)
	return users, meta, nil
}

// UpdateProfileInput contains profile update fields
type UpdateProfileInput struct {
	Name     *string
	Username *string
	Email    *string
	Bio      *string
}

// UpdateProfile updates user profile
func (s *UserService) UpdateProfile(ctx context.Context, userID uuid.UUID, input UpdateProfileInput) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Apply updates
	if input.Name != nil {
		user.Name = *input.Name
	}
	if input.Username != nil && *input.Username != user.Username {
		// Check if username is reserved
		if utils.IsReservedUsername(*input.Username) {
			return nil, ErrUsernameReserved
		}
		// Check if username is taken
		exists, _ := s.userRepo.UsernameExists(ctx, *input.Username, &userID)
		if exists {
			return nil, ErrUsernameTaken
		}
		user.Username = *input.Username
	}
	if input.Email != nil && *input.Email != user.Email {
		// Check if email is taken
		exists, _ := s.userRepo.EmailExists(ctx, *input.Email, &userID)
		if exists {
			return nil, ErrEmailTaken
		}
		user.Email = *input.Email
	}
	if input.Bio != nil {
		user.Bio = *input.Bio
	}

	// Update in database
	if err := s.userRepo.UpdateProfile(ctx, userID, user.Name, user.Username, user.Email, user.Bio); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	return user, nil
}

// ChangePassword changes user password
func (s *UserService) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	// Verify current password
	if !utils.CheckPassword(currentPassword, user.PasswordHash) {
		return ErrInvalidPassword
	}

	// Hash new password
	newHash, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	if err := s.userRepo.UpdatePassword(ctx, userID, newHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// UpdateSocialLinks updates user social links
func (s *UserService) UpdateSocialLinks(ctx context.Context, userID uuid.UUID, links domain.SocialLinks) error {
	// Validate platforms
	for platform := range links {
		if !utils.IsValidSocialPlatform(platform) {
			return fmt.Errorf("invalid social platform: %s", platform)
		}
	}

	if err := s.userRepo.UpdateSocialLinks(ctx, userID, links); err != nil {
		return fmt.Errorf("failed to update social links: %w", err)
	}

	return nil
}

// CheckUsernameAvailability checks if a username is available
func (s *UserService) CheckUsernameAvailability(ctx context.Context, username string, userID *uuid.UUID) (bool, error) {
	if utils.IsReservedUsername(username) {
		return false, nil
	}

	exists, err := s.userRepo.UsernameExists(ctx, username, userID)
	if err != nil {
		return false, fmt.Errorf("database error: %w", err)
	}

	return !exists, nil
}

// Follow follows a user
func (s *UserService) Follow(ctx context.Context, followerID uuid.UUID, followingUsername string) (bool, int, error) {
	// Get target user
	targetUser, err := s.userRepo.GetByUsername(ctx, followingUsername)
	if err != nil {
		return false, 0, fmt.Errorf("database error: %w", err)
	}
	if targetUser == nil {
		return false, 0, ErrUserNotFound
	}

	// Can't follow self
	if followerID == targetUser.ID {
		return false, 0, ErrCannotFollowSelf
	}

	// Check if already following
	exists, _ := s.followRepo.Exists(ctx, followerID, targetUser.ID)
	if exists {
		return false, 0, ErrAlreadyFollowing
	}

	// Create follow
	if err := s.followRepo.Create(ctx, followerID, targetUser.ID); err != nil {
		return false, 0, fmt.Errorf("failed to follow: %w", err)
	}

	// Get new count
	count, _ := s.followRepo.GetFollowerCount(ctx, targetUser.ID)

	return true, count, nil
}

// Unfollow unfollows a user
func (s *UserService) Unfollow(ctx context.Context, followerID uuid.UUID, followingUsername string) (bool, int, error) {
	// Get target user
	targetUser, err := s.userRepo.GetByUsername(ctx, followingUsername)
	if err != nil {
		return false, 0, fmt.Errorf("database error: %w", err)
	}
	if targetUser == nil {
		return false, 0, ErrUserNotFound
	}

	// Check if following
	exists, _ := s.followRepo.Exists(ctx, followerID, targetUser.ID)
	if !exists {
		return false, 0, ErrNotFollowing
	}

	// Delete follow
	if err := s.followRepo.Delete(ctx, followerID, targetUser.ID); err != nil {
		return false, 0, fmt.Errorf("failed to unfollow: %w", err)
	}

	// Get new count
	count, _ := s.followRepo.GetFollowerCount(ctx, targetUser.ID)

	return false, count, nil
}

// GetFollowers returns followers of a user
func (s *UserService) GetFollowers(ctx context.Context, username string, currentUserID *uuid.UUID, search string, page, limit int) ([]domain.FollowerItem, *utils.Meta, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, nil, fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return nil, nil, ErrUserNotFound
	}

	followers, total, err := s.userRepo.GetFollowers(ctx, user.ID, currentUserID, search, page, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get followers: %w", err)
	}

	meta := utils.NewMeta(page, limit, total)
	return followers, meta, nil
}

// GetFollowing returns users that a user is following
func (s *UserService) GetFollowing(ctx context.Context, username string, currentUserID *uuid.UUID, search string, page, limit int) ([]domain.FollowerItem, *utils.Meta, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, nil, fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return nil, nil, ErrUserNotFound
	}

	following, total, err := s.userRepo.GetFollowing(ctx, user.ID, currentUserID, search, page, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get following: %w", err)
	}

	meta := utils.NewMeta(page, limit, total)
	return following, meta, nil
}

// CreateUserInput contains input for creating a user (admin)
type CreateUserInput struct {
	Username       string
	Email          string
	Password       string
	Name           string
	Role           domain.UserRole
	Status         domain.UserStatus
	NISN           string
	NIS            string
	CurrentClassID *uuid.UUID
	EntryYear      *int
}

// CreateUser creates a new user (admin only)
func (s *UserService) CreateUser(ctx context.Context, input CreateUserInput) (*domain.User, error) {
	// Check reserved username
	if utils.IsReservedUsername(input.Username) {
		return nil, ErrUsernameReserved
	}

	// Check username exists
	exists, _ := s.userRepo.UsernameExists(ctx, input.Username, nil)
	if exists {
		return nil, ErrUsernameTaken
	}

	// Check email exists
	exists, _ = s.userRepo.EmailExists(ctx, input.Email, nil)
	if exists {
		return nil, ErrEmailTaken
	}

	// Hash password
	passwordHash, err := utils.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	user := &domain.User{
		ID:             uuid.New(),
		Username:       input.Username,
		Email:          input.Email,
		PasswordHash:   passwordHash,
		Name:           input.Name,
		Role:           input.Role,
		Status:         input.Status,
		NISN:           input.NISN,
		NIS:            input.NIS,
		CurrentClassID: input.CurrentClassID,
		EntryYear:      input.EntryYear,
		SocialLinks:    make(domain.SocialLinks),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// UpdateUserInput contains input for updating a user (admin)
type UpdateUserInput struct {
	Name           *string
	Username       *string
	Email          *string
	Role           *domain.UserRole
	Status         *domain.UserStatus
	CurrentClassID *uuid.UUID
	NISN           *string
	NIS            *string
	EntryYear      *int
	GraduationYear *int
}

// UpdateUser updates a user (admin only)
func (s *UserService) UpdateUser(ctx context.Context, userID uuid.UUID, input UpdateUserInput) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Apply updates
	if input.Name != nil {
		user.Name = *input.Name
	}
	if input.Username != nil && *input.Username != user.Username {
		if utils.IsReservedUsername(*input.Username) {
			return nil, ErrUsernameReserved
		}
		exists, _ := s.userRepo.UsernameExists(ctx, *input.Username, &userID)
		if exists {
			return nil, ErrUsernameTaken
		}
		user.Username = *input.Username
	}
	if input.Email != nil && *input.Email != user.Email {
		exists, _ := s.userRepo.EmailExists(ctx, *input.Email, &userID)
		if exists {
			return nil, ErrEmailTaken
		}
		user.Email = *input.Email
	}
	if input.Role != nil {
		user.Role = *input.Role
	}
	if input.Status != nil {
		user.Status = *input.Status
	}
	if input.CurrentClassID != nil {
		user.CurrentClassID = input.CurrentClassID
	}
	if input.NISN != nil {
		user.NISN = *input.NISN
	}
	if input.NIS != nil {
		user.NIS = *input.NIS
	}
	if input.EntryYear != nil {
		user.EntryYear = input.EntryYear
	}
	if input.GraduationYear != nil {
		user.GraduationYear = input.GraduationYear
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user, nil
}

// ResetPassword resets a user's password (admin only)
func (s *UserService) ResetPassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	passwordHash, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, passwordHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// DeleteUser soft deletes a user (admin only)
func (s *UserService) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	if err := s.userRepo.SoftDelete(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// User service errors
var (
	ErrUserNotFound     = fmt.Errorf("user not found")
	ErrUsernameTaken    = fmt.Errorf("username taken")
	ErrUsernameReserved = fmt.Errorf("username reserved")
	ErrEmailTaken       = fmt.Errorf("email taken")
	ErrInvalidPassword  = fmt.Errorf("invalid password")
	ErrCannotFollowSelf = fmt.Errorf("cannot follow self")
	ErrAlreadyFollowing = fmt.Errorf("already following")
	ErrNotFollowing     = fmt.Errorf("not following")
)
