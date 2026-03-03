package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/middleware"
	"github.com/grafikarsa/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type ProfileHandler struct {
	userRepo  *repository.UserRepository
	adminRepo *repository.AdminRepository
}

func NewProfileHandler(userRepo *repository.UserRepository, adminRepo *repository.AdminRepository) *ProfileHandler {
	return &ProfileHandler{userRepo: userRepo, adminRepo: adminRepo}
}

func (h *ProfileHandler) GetMe(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"UNAUTHORIZED", "User tidak terautentikasi",
		))
	}

	user, err := h.userRepo.FindByID(*userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"USER_NOT_FOUND", "User tidak ditemukan",
		))
	}

	followerCount, _ := h.userRepo.GetFollowerCount(user.ID)
	followingCount, _ := h.userRepo.GetFollowingCount(user.ID)

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

	profileDTO := dto.ProfileDTO{
		ID:             user.ID,
		Username:       user.Username,
		Email:          user.Email,
		Nama:           user.Nama,
		Bio:            user.Bio,
		AvatarURL:      user.AvatarURL,
		BannerURL:      user.BannerURL,
		Role:           string(user.Role),
		NISN:           user.NISN,
		NIS:            user.NIS,
		TahunMasuk:     user.TahunMasuk,
		TahunLulus:     tahunLulus,
		SocialLinks:    socialLinks,
		FollowerCount:  followerCount,
		FollowingCount: followingCount,
		CreatedAt:      user.CreatedAt,
	}

	if user.Kelas != nil {
		profileDTO.Kelas = &dto.KelasDTO{ID: user.Kelas.ID, Nama: user.Kelas.Nama}
		if user.Kelas.Jurusan != nil {
			profileDTO.Jurusan = &dto.JurusanDTO{ID: user.Kelas.Jurusan.ID, Nama: user.Kelas.Jurusan.Nama}
		}
	}

	// Get special roles and capabilities
	if h.adminRepo != nil {
		specialRoles, _ := h.adminRepo.GetUserSpecialRoles(user.ID)
		for _, sr := range specialRoles {
			profileDTO.SpecialRoles = append(profileDTO.SpecialRoles, dto.ProfileSpecialRole{
				ID:           sr.ID,
				Nama:         sr.Nama,
				Color:        sr.Color,
				Capabilities: []string(sr.Capabilities),
				IsActive:     sr.IsActive,
			})
		}

		capabilities, _ := h.adminRepo.GetUserCapabilities(user.ID)
		profileDTO.Capabilities = capabilities
	}

	return c.JSON(dto.SuccessResponse(profileDTO, ""))
}

func (h *ProfileHandler) UpdateMe(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"UNAUTHORIZED", "User tidak terautentikasi",
		))
	}

	var req dto.UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Request body tidak valid",
		))
	}

	user, err := h.userRepo.FindByID(*userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"USER_NOT_FOUND", "User tidak ditemukan",
		))
	}

	// Check username uniqueness
	if req.Username != nil && *req.Username != user.Username {
		exists, _ := h.userRepo.UsernameExists(*req.Username, userID)
		if exists {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse(
				"USERNAME_TAKEN", "Username sudah digunakan",
			))
		}
		user.Username = *req.Username
	}

	// Check email uniqueness
	if req.Email != nil && *req.Email != user.Email {
		exists, _ := h.userRepo.EmailExists(*req.Email, userID)
		if exists {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse(
				"DUPLICATE_EMAIL", "Email sudah digunakan",
			))
		}
		user.Email = *req.Email
	}

	if req.Nama != nil {
		user.Nama = *req.Nama
	}
	if req.Bio != nil {
		user.Bio = req.Bio
	}

	if err := h.userRepo.Update(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal memperbarui profil",
		))
	}

	return c.JSON(dto.SuccessResponse(map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"nama":     user.Nama,
		"bio":      user.Bio,
	}, "Profil berhasil diperbarui"))
}

func (h *ProfileHandler) UpdatePassword(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"UNAUTHORIZED", "User tidak terautentikasi",
		))
	}

	var req dto.UpdatePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Request body tidak valid",
		))
	}

	if req.NewPassword != req.NewPasswordConfirmation {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Validasi gagal",
			dto.ErrorDetail{Field: "new_password_confirmation", Message: "Konfirmasi password tidak cocok"},
		))
	}

	if len(req.NewPassword) < 8 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Validasi gagal",
			dto.ErrorDetail{Field: "new_password", Message: "Password minimal 8 karakter"},
		))
	}

	user, err := h.userRepo.FindByID(*userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"USER_NOT_FOUND", "User tidak ditemukan",
		))
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"INVALID_PASSWORD", "Password lama tidak sesuai",
		))
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal mengenkripsi password",
		))
	}

	user.PasswordHash = string(hashedPassword)
	if err := h.userRepo.Update(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal memperbarui password",
		))
	}

	return c.JSON(dto.SuccessResponse(nil, "Password berhasil diubah"))
}

func (h *ProfileHandler) UpdateSocialLinks(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"UNAUTHORIZED", "User tidak terautentikasi",
		))
	}

	var req dto.UpdateSocialLinksRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Request body tidak valid",
		))
	}

	var links []domain.UserSocialLink
	for _, sl := range req.SocialLinks {
		links = append(links, domain.UserSocialLink{
			Platform: domain.SocialPlatform(sl.Platform),
			URL:      sl.URL,
		})
	}

	if err := h.userRepo.UpdateSocialLinks(*userID, links); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal memperbarui social links",
		))
	}

	return c.JSON(dto.SuccessResponse(map[string]interface{}{
		"social_links": req.SocialLinks,
	}, "Social links berhasil diperbarui"))
}

func (h *ProfileHandler) CheckUsername(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	username := c.Query("username")

	if username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Username wajib diisi",
		))
	}

	exists, _ := h.userRepo.UsernameExists(username, userID)

	return c.JSON(dto.SuccessResponse(dto.CheckUsernameResponse{
		Username:  username,
		Available: !exists,
	}, ""))
}
