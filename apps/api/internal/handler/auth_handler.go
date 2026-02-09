package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"grafikarsa/internal/domain"
	"grafikarsa/internal/middleware"
	"grafikarsa/internal/service"
	"grafikarsa/internal/utils"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register registers auth routes
func (h *AuthHandler) Register(app fiber.Router, authMiddleware *middleware.AuthMiddleware) {
	auth := app.Group("/auth")

	auth.Post("/login", h.Login)
	auth.Post("/refresh", h.RefreshToken)
	auth.Post("/logout", h.Logout)
	auth.Post("/logout-all", authMiddleware.Required(), h.LogoutAll)

	// Protected routes
	auth.Get("/sessions", authMiddleware.Required(), h.GetSessions)
	auth.Post("/sessions/logout-all", authMiddleware.Required(), h.LogoutAll)
	auth.Delete("/sessions/:id", authMiddleware.Required(), h.RevokeSession)
}

// LoginRequest is the login request body
type LoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required,min=8"`
}

// Login handles user login
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	// Validate
	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	// Get client info
	loginInfo := domain.LoginInfo{
		Username:  req.Username,
		Password:  req.Password,
		UserAgent: c.Get("User-Agent"),
		IPAddress: c.IP(),
	}

	result, err := h.authService.Login(c.Context(), loginInfo)
	if err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			return utils.Error(c, fiber.StatusUnauthorized, utils.ErrCodeInvalidCredentials, "Username atau password salah")
		case service.ErrAccountDisabled:
			return utils.Error(c, fiber.StatusForbidden, utils.ErrCodeAccountDisabled, "Akun Anda dinonaktifkan. Hubungi admin.")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Terjadi kesalahan saat login")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"expires_in":    result.ExpiresIn,
		"user": fiber.Map{
			"id":       result.User.ID,
			"username": result.User.Username,
			"name":     result.User.Name,
			"role":     result.User.Role,
		},
	}, "Login berhasil")
}

// RefreshTokenRequest is the refresh token request body
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req RefreshTokenRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if req.RefreshToken == "" {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Refresh token diperlukan")
	}

	result, err := h.authService.RefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		switch err {
		case service.ErrTokenExpired:
			return utils.Error(c, fiber.StatusUnauthorized, utils.ErrCodeTokenExpired, "Token sudah kedaluwarsa. Silakan login ulang.")
		case service.ErrTokenReuseDetected:
			return utils.Error(c, fiber.StatusUnauthorized, utils.ErrCodeTokenReuseDetected, "Penggunaan token yang mencurigakan terdeteksi. Semua sesi telah diakhiri.")
		case service.ErrAccountDisabled:
			return utils.Error(c, fiber.StatusForbidden, utils.ErrCodeAccountDisabled, "Akun Anda dinonaktifkan.")
		default:
			return utils.Error(c, fiber.StatusUnauthorized, utils.ErrCodeTokenExpired, "Token tidak valid")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"expires_in":    result.ExpiresIn,
	}, "Token diperbarui")
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	// Get refresh token from request body or Authorization header
	var refreshToken string

	var req RefreshTokenRequest
	if err := c.BodyParser(&req); err == nil && req.RefreshToken != "" {
		refreshToken = req.RefreshToken
	}

	if refreshToken == "" {
		// Try to get from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 {
				refreshToken = parts[1]
			}
		}
	}

	if err := h.authService.Logout(c.Context(), refreshToken); err != nil {
		// Log error but don't fail - user still logs out
	}

	return utils.SuccessResponse(c, fiber.StatusOK, nil, "Logout berhasil")
}

// GetSessions returns all active sessions for the user
func (h *AuthHandler) GetSessions(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	// Get refresh token from Authorization header for current session detection
	refreshToken := ""
	authHeader := c.Get("X-Refresh-Token")
	if authHeader != "" {
		refreshToken = authHeader
	}

	sessions, err := h.authService.GetActiveSessions(c.Context(), userID, refreshToken)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil sesi")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"sessions": sessions,
	}, "")
}

// LogoutAll logs out from all sessions
func (h *AuthHandler) LogoutAll(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	count, err := h.authService.LogoutAll(c.Context(), userID)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengakhiri sesi")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, fiber.Map{
		"revoked_sessions": count,
	}, "Semua sesi berhasil diakhiri")
}

// RevokeSession revokes a specific session
func (h *AuthHandler) RevokeSession(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	sessionID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID sesi tidak valid")
	}

	if err := h.authService.RevokeSession(c.Context(), userID, sessionID); err != nil {
		switch err {
		case service.ErrSessionNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeNotFound, "Sesi tidak ditemukan")
		case service.ErrForbidden:
			return utils.Error(c, fiber.StatusForbidden, utils.ErrCodeForbidden, "Tidak dapat mengakhiri sesi milik pengguna lain")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengakhiri sesi")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, nil, "Sesi berhasil diakhiri")
}
