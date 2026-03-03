package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/auth"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/middleware"
	"github.com/grafikarsa/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userRepo *repository.UserRepository
	authRepo *repository.AuthRepository
	jwt      *auth.JWTService
}

func NewAuthHandler(userRepo *repository.UserRepository, authRepo *repository.AuthRepository, jwt *auth.JWTService) *AuthHandler {
	return &AuthHandler{
		userRepo: userRepo,
		authRepo: authRepo,
		jwt:      jwt,
	}
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Request body tidak valid",
		))
	}

	user, err := h.userRepo.FindByUsernameOrEmail(req.Username)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"INVALID_CREDENTIALS", "Username atau password salah",
		))
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"INVALID_CREDENTIALS", "Username atau password salah",
		))
	}

	if !user.IsActive {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
			"ACCOUNT_DISABLED", "Akun Anda telah dinonaktifkan. Hubungi admin.",
		))
	}

	// Generate tokens
	accessToken, _, err := h.jwt.GenerateAccessToken(user.ID, string(user.Role))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal membuat token",
		))
	}

	refreshToken, tokenHash, expiresAt := h.jwt.GenerateRefreshToken()
	familyID := uuid.New()

	// Get device info
	deviceInfo := domain.JSONB{
		"user_agent":  c.Get("User-Agent"),
		"device_type": "unknown",
	}
	ipAddress := c.IP()

	// Store refresh token
	rt := &domain.RefreshToken{
		UserID:     user.ID,
		TokenHash:  tokenHash,
		FamilyID:   familyID,
		DeviceInfo: deviceInfo,
		IPAddress:  &ipAddress,
		ExpiresAt:  expiresAt,
	}
	if err := h.authRepo.CreateRefreshToken(rt); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal menyimpan token",
		))
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	h.userRepo.Update(user)

	// Set refresh token cookie
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/api/v1/auth",
		Expires:  expiresAt,
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Strict",
	})

	return c.JSON(dto.SuccessResponse(dto.LoginResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(h.jwt.GetAccessExpiry().Seconds()),
		User: dto.UserBriefDTO{
			ID:        user.ID,
			Username:  user.Username,
			Nama:      user.Nama,
			Role:      string(user.Role),
			AvatarURL: user.AvatarURL,
		},
	}, ""))
}

func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"TOKEN_EXPIRED", "Refresh token tidak ada. Silakan login ulang.",
		))
	}

	tokenHash := auth.HashToken(refreshToken)
	storedToken, err := h.authRepo.FindRefreshTokenByHash(tokenHash)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"TOKEN_EXPIRED", "Refresh token tidak valid. Silakan login ulang.",
		))
	}

	// Check if token is revoked (potential reuse attack)
	if storedToken.IsRevoked {
		// Revoke entire token family
		h.authRepo.RevokeTokenFamily(storedToken.FamilyID, "token_reuse_detected")
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"TOKEN_REUSE_DETECTED", "Aktivitas mencurigakan terdeteksi. Semua sesi telah diakhiri. Silakan login ulang.",
		))
	}

	// Check expiration
	if time.Now().After(storedToken.ExpiresAt) {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"TOKEN_EXPIRED", "Refresh token telah expired. Silakan login ulang.",
		))
	}

	// Get user
	user, err := h.userRepo.FindByID(storedToken.UserID)
	if err != nil || !user.IsActive {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"UNAUTHORIZED", "User tidak ditemukan atau tidak aktif",
		))
	}

	// Revoke old token
	h.authRepo.RevokeRefreshToken(storedToken.ID, "rotated")

	// Generate new tokens
	accessToken, _, err := h.jwt.GenerateAccessToken(user.ID, string(user.Role))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal membuat token",
		))
	}

	newRefreshToken, newTokenHash, expiresAt := h.jwt.GenerateRefreshToken()

	// Store new refresh token (same family)
	rt := &domain.RefreshToken{
		UserID:     user.ID,
		TokenHash:  newTokenHash,
		FamilyID:   storedToken.FamilyID,
		DeviceInfo: storedToken.DeviceInfo,
		IPAddress:  storedToken.IPAddress,
		ExpiresAt:  expiresAt,
	}
	if err := h.authRepo.CreateRefreshToken(rt); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal menyimpan token",
		))
	}

	// Set new refresh token cookie
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		Path:     "/api/v1/auth",
		Expires:  expiresAt,
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Strict",
	})

	return c.JSON(dto.SuccessResponse(dto.RefreshResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(h.jwt.GetAccessExpiry().Seconds()),
	}, ""))
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	// Revoke refresh token
	refreshToken := c.Cookies("refresh_token")
	if refreshToken != "" {
		tokenHash := auth.HashToken(refreshToken)
		storedToken, err := h.authRepo.FindRefreshTokenByHash(tokenHash)
		if err == nil {
			h.authRepo.RevokeRefreshToken(storedToken.ID, "logout")
		}
	}

	// Clear cookie
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Strict",
	})

	return c.JSON(dto.SuccessResponse(nil, "Berhasil logout"))
}

func (h *AuthHandler) LogoutAll(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"UNAUTHORIZED", "User tidak terautentikasi",
		))
	}

	count, err := h.authRepo.RevokeAllUserTokens(*userID, "logout_all")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal logout dari semua perangkat",
		))
	}

	// Clear cookie
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/v1/auth",
		Expires:  time.Now().Add(-time.Hour),
		HTTPOnly: true,
		Secure:   c.Protocol() == "https",
		SameSite: "Strict",
	})

	return c.JSON(dto.SuccessResponse(dto.LogoutAllResponse{
		SessionsTerminated: int(count),
	}, "Berhasil logout dari semua perangkat"))
}

func (h *AuthHandler) GetSessions(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"UNAUTHORIZED", "User tidak terautentikasi",
		))
	}

	sessions, err := h.authRepo.GetUserSessions(*userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal mengambil data sesi",
		))
	}

	// Get current session
	currentTokenHash := ""
	if refreshToken := c.Cookies("refresh_token"); refreshToken != "" {
		currentTokenHash = auth.HashToken(refreshToken)
	}

	var sessionDTOs []dto.SessionDTO
	for _, s := range sessions {
		var lastUsed *string
		if s.LastUsedAt != nil {
			t := s.LastUsedAt.Format(time.RFC3339)
			lastUsed = &t
		}

		sessionDTOs = append(sessionDTOs, dto.SessionDTO{
			ID:         s.ID,
			DeviceInfo: s.DeviceInfo,
			IPAddress:  s.IPAddress,
			CreatedAt:  s.CreatedAt.Format(time.RFC3339),
			LastUsedAt: lastUsed,
			IsCurrent:  s.TokenHash == currentTokenHash,
		})
	}

	return c.JSON(dto.SuccessResponse(sessionDTOs, ""))
}

func (h *AuthHandler) DeleteSession(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"UNAUTHORIZED", "User tidak terautentikasi",
		))
	}

	sessionID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "ID sesi tidak valid",
		))
	}

	// Verify session ownership
	session, err := h.authRepo.FindRefreshTokenByID(sessionID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"SESSION_NOT_FOUND", "Sesi tidak ditemukan",
		))
	}

	if session.UserID != *userID {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
			"FORBIDDEN", "Anda tidak memiliki akses untuk menghapus sesi ini",
		))
	}

	if err := h.authRepo.RevokeRefreshToken(sessionID, "manual_revoke"); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal menghapus sesi",
		))
	}

	return c.JSON(dto.SuccessResponse(nil, "Sesi berhasil dihapus"))
}
