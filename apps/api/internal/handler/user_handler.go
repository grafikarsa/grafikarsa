package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"grafikarsa/internal/domain"
	"grafikarsa/internal/middleware"
	"grafikarsa/internal/repository"
	"grafikarsa/internal/service"
	"grafikarsa/internal/utils"
)

// UserHandler handles user endpoints
type UserHandler struct {
	userService *service.UserService
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// Register registers user routes
func (h *UserHandler) Register(app fiber.Router, authMiddleware *middleware.AuthMiddleware) {
	users := app.Group("/users")

	// Public routes
	users.Get("", authMiddleware.Optional(), h.ListUsers)
	users.Get("/check-username", h.CheckUsername)
	users.Get("/:username", authMiddleware.Optional(), h.GetUserProfile)
	users.Get("/:username/followers", authMiddleware.Optional(), h.GetFollowers)
	users.Get("/:username/following", authMiddleware.Optional(), h.GetFollowing)

	// Protected routes
	users.Post("/:username/follow", authMiddleware.Required(), h.Follow)
	users.Delete("/:username/follow", authMiddleware.Required(), h.Unfollow)

	// Me routes
	me := app.Group("/me", authMiddleware.Required())
	me.Get("", h.GetMe)
	me.Patch("", h.UpdateMe)
	me.Patch("/password", h.ChangePassword)
	me.Get("/check-username", h.CheckUsername)
	me.Put("/social-links", h.UpdateSocialLinks)
}

// ListUsers lists users with filtering
func (h *UserHandler) ListUsers(c *fiber.Ctx) error {
	filter := repository.UserFilter{
		Search: c.Query("search"),
		Role:   c.Query("role"),
		Status: c.Query("status"),
		Page:   c.QueryInt("page", 1),
		Limit:  c.QueryInt("limit", 20),
	}

	if majorID := c.Query("major_id"); majorID != "" {
		if id, err := uuid.Parse(majorID); err == nil {
			filter.MajorID = &id
		}
	}

	if classID := c.Query("class_id"); classID != "" {
		if id, err := uuid.Parse(classID); err == nil {
			filter.ClassID = &id
		}
	}

	users, meta, err := h.userService.ListUsers(c.Context(), filter)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil daftar pengguna")
	}

	return utils.SuccessWithMeta(c, fiber.StatusOK, users, meta)
}

// GetUserProfile gets a user's public profile
func (h *UserHandler) GetUserProfile(c *fiber.Ctx) error {
	username := c.Params("username")
	currentUserID := middleware.GetUserIDOptional(c)

	profile, err := h.userService.GetUserByUsername(c.Context(), username, currentUserID)
	if err != nil {
		if err == service.ErrUserNotFound {
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeUserNotFound, "Pengguna tidak ditemukan")
		}
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil profil")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, profile, "")
}

// GetMe returns the current user's profile
func (h *UserHandler) GetMe(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	user, err := h.userService.GetUserByID(c.Context(), userID)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil profil")
	}

	// Return full user data for /me endpoint
	return utils.SuccessResponse(c, fiber.StatusOK, user.ToMeProfile(), "")
}

// UpdateMeRequest is the request body for updating profile
type UpdateMeRequest struct {
	Name     *string `json:"name" validate:"omitempty,max=100"`
	Username *string `json:"username" validate:"omitempty,username,not_reserved_username"`
	Email    *string `json:"email" validate:"omitempty,email"`
	Bio      *string `json:"bio" validate:"omitempty,max=500"`
}

// UpdateMe updates the current user's profile
func (h *UserHandler) UpdateMe(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req UpdateMeRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	input := service.UpdateProfileInput{
		Name:     req.Name,
		Username: req.Username,
		Email:    req.Email,
		Bio:      req.Bio,
	}

	user, err := h.userService.UpdateProfile(c.Context(), userID, input)
	if err != nil {
		switch err {
		case service.ErrUsernameTaken:
			return utils.Error(c, fiber.StatusConflict, utils.ErrCodeDuplicateUsername, "Username sudah digunakan")
		case service.ErrUsernameReserved:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeReservedUsername, "Username tidak dapat digunakan")
		case service.ErrEmailTaken:
			return utils.Error(c, fiber.StatusConflict, utils.ErrCodeDuplicateEmail, "Email sudah digunakan")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal memperbarui profil")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, user.ToMeProfile(), "Profil berhasil diperbarui")
}

// ChangePasswordRequest is the request body for changing password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

// ChangePassword changes the user's password
func (h *UserHandler) ChangePassword(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req ChangePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	err := h.userService.ChangePassword(c.Context(), userID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		if err == service.ErrInvalidPassword {
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidPassword, "Password saat ini salah")
		}
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengubah password")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, nil, "Password berhasil diubah")
}

// UpdateSocialLinksRequest is the request body for updating social links
type UpdateSocialLinksRequest struct {
	SocialLinks domain.SocialLinks `json:"social_links" validate:"dive,keys,social_platform,endkeys,url"`
}

// UpdateSocialLinks updates the user's social links
func (h *UserHandler) UpdateSocialLinks(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req UpdateSocialLinksRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	err := h.userService.UpdateSocialLinks(c.Context(), userID, req.SocialLinks)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal memperbarui social links")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"social_links": req.SocialLinks,
	}, "Social links berhasil diperbarui")
}

// CheckUsername checks if a username is available
func (h *UserHandler) CheckUsername(c *fiber.Ctx) error {
	username := c.Query("username")
	if username == "" {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Username diperlukan")
	}

	// Validate format
	if !utils.IsValidUsername(username) {
		return utils.SuccessResponse(c, fiber.StatusOK, fiber.Map{
			"available": false,
			"reason":    "Format username tidak valid",
		}, "")
	}

	// Check reserved
	if utils.IsReservedUsername(username) {
		return utils.SuccessResponse(c, fiber.StatusOK, fiber.Map{
			"available": false,
			"reason":    "Username tidak dapat digunakan",
		}, "")
	}

	available, err := h.userService.CheckUsernameAvailability(c.Context(), username, nil)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal memeriksa username")
	}

	response := fiber.Map{"available": available}
	if !available {
		response["reason"] = "Username sudah digunakan"
	}

	return utils.SuccessResponse(c, fiber.StatusOK, response, "")
}

// Follow follows a user
func (h *UserHandler) Follow(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	username := c.Params("username")

	isFollowing, followerCount, err := h.userService.Follow(c.Context(), userID, username)
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeUserNotFound, "Pengguna tidak ditemukan")
		case service.ErrCannotFollowSelf:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Tidak dapat mengikuti diri sendiri")
		case service.ErrAlreadyFollowing:
			return utils.Error(c, fiber.StatusConflict, utils.ErrCodeDuplicateFollow, "Sudah mengikuti pengguna ini")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengikuti pengguna")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"is_following":   isFollowing,
		"follower_count": followerCount,
	}, "Berhasil mengikuti")
}

// Unfollow unfollows a user
func (h *UserHandler) Unfollow(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	username := c.Params("username")

	isFollowing, followerCount, err := h.userService.Unfollow(c.Context(), userID, username)
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeUserNotFound, "Pengguna tidak ditemukan")
		case service.ErrNotFollowing:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Tidak mengikuti pengguna ini")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal berhenti mengikuti")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"is_following":   isFollowing,
		"follower_count": followerCount,
	}, "Berhasil berhenti mengikuti")
}

// GetFollowers gets a user's followers
func (h *UserHandler) GetFollowers(c *fiber.Ctx) error {
	username := c.Params("username")
	currentUserID := middleware.GetUserIDOptional(c)
	search := c.Query("search")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	followers, meta, err := h.userService.GetFollowers(c.Context(), username, currentUserID, search, page, limit)
	if err != nil {
		if err == service.ErrUserNotFound {
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeUserNotFound, "Pengguna tidak ditemukan")
		}
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil followers")
	}

	return utils.SuccessWithMeta(c, fiber.StatusOK, followers, meta)
}

// GetFollowing gets users that a user is following
func (h *UserHandler) GetFollowing(c *fiber.Ctx) error {
	username := c.Params("username")
	currentUserID := middleware.GetUserIDOptional(c)
	search := c.Query("search")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	following, meta, err := h.userService.GetFollowing(c.Context(), username, currentUserID, search, page, limit)
	if err != nil {
		if err == service.ErrUserNotFound {
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeUserNotFound, "Pengguna tidak ditemukan")
		}
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil following")
	}

	return utils.SuccessWithMeta(c, fiber.StatusOK, following, meta)
}
