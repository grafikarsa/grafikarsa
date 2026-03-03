package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/middleware"
	"github.com/grafikarsa/backend/internal/repository"
	"github.com/grafikarsa/backend/internal/service"
)

type UserHandler struct {
	userRepo     *repository.UserRepository
	followRepo   *repository.FollowRepository
	notifService *service.NotificationService
}

func NewUserHandler(userRepo *repository.UserRepository, followRepo *repository.FollowRepository, notifService *service.NotificationService) *UserHandler {
	return &UserHandler{
		userRepo:     userRepo,
		followRepo:   followRepo,
		notifService: notifService,
	}
}

func (h *UserHandler) List(c *fiber.Ctx) error {
	search := c.Query("search")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if limit > 50 {
		limit = 50
	}

	var jurusanID, kelasID *uuid.UUID
	if id := c.Query("jurusan_id"); id != "" {
		parsed, _ := uuid.Parse(id)
		jurusanID = &parsed
	}
	if id := c.Query("kelas_id"); id != "" {
		parsed, _ := uuid.Parse(id)
		kelasID = &parsed
	}

	var role *string
	if r := c.Query("role"); r != "" {
		role = &r
	}

	users, total, err := h.userRepo.List(search, jurusanID, kelasID, role, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal mengambil data user",
		))
	}

	var userDTOs []dto.UserListDTO
	for _, u := range users {
		userDTO := dto.UserListDTO{
			ID:         u.ID,
			Username:   u.Username,
			Nama:       u.Nama,
			AvatarURL:  u.AvatarURL,
			BannerURL:  u.BannerURL,
			Role:       string(u.Role),
			TahunMasuk: u.TahunMasuk,
			TahunLulus: u.TahunLulus,
		}
		if u.Kelas != nil {
			userDTO.Kelas = &dto.KelasDTO{ID: u.Kelas.ID, Nama: u.Kelas.Nama}
			if u.Kelas.Jurusan != nil {
				userDTO.Jurusan = &dto.JurusanDTO{ID: u.Kelas.Jurusan.ID, Nama: u.Kelas.Jurusan.Nama, Kode: u.Kelas.Jurusan.Kode}
			}
		}
		userDTOs = append(userDTOs, userDTO)
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return c.JSON(dto.SuccessWithMeta(userDTOs, &dto.Meta{
		CurrentPage: page,
		PerPage:     limit,
		TotalPages:  totalPages,
		TotalCount:  total,
	}))
}

func (h *UserHandler) GetByUsername(c *fiber.Ctx) error {
	username := c.Params("username")
	currentUserID := middleware.GetUserID(c)
	isAdmin := middleware.GetUserRole(c) == "admin"

	user, err := h.userRepo.FindByUsername(username)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"USER_NOT_FOUND", "User tidak ditemukan",
		))
	}

	// Check if user is active - admin can still view inactive users
	if !user.IsActive && !isAdmin {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"USER_NOT_FOUND", "User tidak ditemukan",
		))
	}

	followerCount, _ := h.userRepo.GetFollowerCount(user.ID)
	followingCount, _ := h.userRepo.GetFollowingCount(user.ID)
	portfolioCount, _ := h.userRepo.GetPublishedPortfolioCount(user.ID)

	isFollowing := false
	if currentUserID != nil {
		isFollowing, _ = h.userRepo.IsFollowing(*currentUserID, user.ID)
	}

	classHistory, _ := h.userRepo.GetClassHistory(user.ID)
	var historyDTOs []dto.ClassHistoryDTO
	for _, ch := range classHistory {
		historyDTOs = append(historyDTOs, dto.ClassHistoryDTO{
			KelasNama:   ch.Kelas.Nama,
			TahunAjaran: ch.TahunAjaran.TahunMulai,
		})
	}

	var socialLinks []dto.SocialLinkDTO
	for _, sl := range user.SocialLinks {
		socialLinks = append(socialLinks, dto.SocialLinkDTO{
			Platform: string(sl.Platform),
			URL:      sl.URL,
		})
	}

	// Calculate tahun_lulus from tahun_masuk + 3 if not set
	var tahunLulus *int
	if user.TahunLulus != nil {
		tahunLulus = user.TahunLulus
	} else if user.TahunMasuk != nil {
		calculated := *user.TahunMasuk + 3
		tahunLulus = &calculated
	}

	// Get special roles for public display
	specialRoles, _ := h.userRepo.GetUserSpecialRoles(user.ID)
	var specialRoleDTOs []dto.UserSpecialRoleDTO
	for _, sr := range specialRoles {
		specialRoleDTOs = append(specialRoleDTOs, dto.UserSpecialRoleDTO{
			ID:    sr.ID,
			Nama:  sr.Nama,
			Color: sr.Color,
		})
	}

	userDTO := dto.UserDetailDTO{
		ID:             user.ID,
		Username:       user.Username,
		Email:          user.Email,
		Nama:           user.Nama,
		Bio:            user.Bio,
		AvatarURL:      user.AvatarURL,
		BannerURL:      user.BannerURL,
		Role:           string(user.Role),
		IsActive:       user.IsActive,
		TahunMasuk:     user.TahunMasuk,
		TahunLulus:     tahunLulus,
		ClassHistory:   historyDTOs,
		SocialLinks:    socialLinks,
		SpecialRoles:   specialRoleDTOs,
		FollowerCount:  followerCount,
		FollowingCount: followingCount,
		PortfolioCount: portfolioCount,
		IsFollowing:    isFollowing,
		CreatedAt:      user.CreatedAt,
	}

	if user.Kelas != nil {
		userDTO.Kelas = &dto.KelasDTO{ID: user.Kelas.ID, Nama: user.Kelas.Nama}
		if user.Kelas.Jurusan != nil {
			userDTO.Jurusan = &dto.JurusanDTO{ID: user.Kelas.Jurusan.ID, Nama: user.Kelas.Jurusan.Nama}
		}
	}

	return c.JSON(dto.SuccessResponse(userDTO, ""))
}

func (h *UserHandler) GetFollowers(c *fiber.Ctx) error {
	username := c.Params("username")
	search := c.Query("search")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	currentUserID := middleware.GetUserID(c)

	user, err := h.userRepo.FindByUsername(username)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"USER_NOT_FOUND", "User tidak ditemukan",
		))
	}

	follows, total, err := h.followRepo.GetFollowers(user.ID, search, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal mengambil data follower",
		))
	}

	var followerDTOs []dto.FollowerDTO
	for _, f := range follows {
		isFollowing := false
		if currentUserID != nil {
			isFollowing, _ = h.followRepo.IsFollowing(*currentUserID, f.FollowerID)
		}

		var kelasNama *string
		if f.Follower != nil && f.Follower.Kelas != nil {
			kelasNama = &f.Follower.Kelas.Nama
		}

		followerDTOs = append(followerDTOs, dto.FollowerDTO{
			ID:          f.Follower.ID,
			Username:    f.Follower.Username,
			Nama:        f.Follower.Nama,
			AvatarURL:   f.Follower.AvatarURL,
			Role:        string(f.Follower.Role),
			KelasNama:   kelasNama,
			IsFollowing: isFollowing,
			FollowedAt:  f.CreatedAt,
		})
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return c.JSON(dto.SuccessWithMeta(followerDTOs, &dto.Meta{
		CurrentPage: page,
		PerPage:     limit,
		TotalPages:  totalPages,
		TotalCount:  total,
	}))
}

func (h *UserHandler) GetFollowing(c *fiber.Ctx) error {
	username := c.Params("username")
	search := c.Query("search")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	currentUserID := middleware.GetUserID(c)

	user, err := h.userRepo.FindByUsername(username)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"USER_NOT_FOUND", "User tidak ditemukan",
		))
	}

	follows, total, err := h.followRepo.GetFollowing(user.ID, search, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal mengambil data following",
		))
	}

	var followingDTOs []dto.FollowerDTO
	for _, f := range follows {
		isFollowing := false
		if currentUserID != nil {
			isFollowing, _ = h.followRepo.IsFollowing(*currentUserID, f.FollowingID)
		}

		var kelasNama *string
		if f.Following != nil && f.Following.Kelas != nil {
			kelasNama = &f.Following.Kelas.Nama
		}

		followingDTOs = append(followingDTOs, dto.FollowerDTO{
			ID:          f.Following.ID,
			Username:    f.Following.Username,
			Nama:        f.Following.Nama,
			AvatarURL:   f.Following.AvatarURL,
			Role:        string(f.Following.Role),
			KelasNama:   kelasNama,
			IsFollowing: isFollowing,
			FollowedAt:  f.CreatedAt,
		})
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return c.JSON(dto.SuccessWithMeta(followingDTOs, &dto.Meta{
		CurrentPage: page,
		PerPage:     limit,
		TotalPages:  totalPages,
		TotalCount:  total,
	}))
}

func (h *UserHandler) Follow(c *fiber.Ctx) error {
	currentUserID := middleware.GetUserID(c)
	if currentUserID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"UNAUTHORIZED", "User tidak terautentikasi",
		))
	}

	// Admin cannot follow users
	if middleware.GetUserRole(c) == "admin" {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
			"FORBIDDEN", "Admin tidak dapat follow user",
		))
	}

	username := c.Params("username")
	targetUser, err := h.userRepo.FindByUsername(username)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"USER_NOT_FOUND", "User tidak ditemukan",
		))
	}

	if *currentUserID == targetUser.ID {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"CANNOT_FOLLOW_SELF", "Tidak bisa follow diri sendiri",
		))
	}

	isFollowing, _ := h.followRepo.IsFollowing(*currentUserID, targetUser.ID)
	if isFollowing {
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse(
			"ALREADY_FOLLOWING", "Anda sudah follow user ini",
		))
	}

	if err := h.followRepo.Follow(*currentUserID, targetUser.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal follow user",
		))
	}

	// Send notification to followed user
	if h.notifService != nil {
		follower, _ := h.userRepo.FindByID(*currentUserID)
		if follower != nil {
			_ = h.notifService.NotifyNewFollower(follower, targetUser.ID)
		}
	}

	followerCount, _ := h.followRepo.GetFollowerCount(targetUser.ID)

	return c.JSON(dto.SuccessResponse(dto.FollowResponse{
		IsFollowing:   true,
		FollowerCount: followerCount,
	}, "Berhasil follow "+username))
}

func (h *UserHandler) Unfollow(c *fiber.Ctx) error {
	currentUserID := middleware.GetUserID(c)
	if currentUserID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"UNAUTHORIZED", "User tidak terautentikasi",
		))
	}

	// Admin cannot unfollow users
	if middleware.GetUserRole(c) == "admin" {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
			"FORBIDDEN", "Admin tidak dapat unfollow user",
		))
	}

	username := c.Params("username")
	targetUser, err := h.userRepo.FindByUsername(username)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"USER_NOT_FOUND", "User tidak ditemukan",
		))
	}

	isFollowing, _ := h.followRepo.IsFollowing(*currentUserID, targetUser.ID)
	if !isFollowing {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"NOT_FOLLOWING", "Anda belum follow user ini",
		))
	}

	if err := h.followRepo.Unfollow(*currentUserID, targetUser.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal unfollow user",
		))
	}

	followerCount, _ := h.followRepo.GetFollowerCount(targetUser.ID)

	return c.JSON(dto.SuccessResponse(dto.FollowResponse{
		IsFollowing:   false,
		FollowerCount: followerCount,
	}, "Berhasil unfollow "+username))
}
