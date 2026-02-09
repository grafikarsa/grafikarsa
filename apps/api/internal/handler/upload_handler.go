package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"grafikarsa/internal/domain"
	"grafikarsa/internal/middleware"
	"grafikarsa/internal/service"
	"grafikarsa/internal/utils"
)

// UploadHandler handles file upload endpoints
type UploadHandler struct {
	uploadService *service.UploadService
}

// NewUploadHandler creates a new UploadHandler
func NewUploadHandler(uploadService *service.UploadService) *UploadHandler {
	return &UploadHandler{uploadService: uploadService}
}

// Register registers upload routes
func (h *UploadHandler) Register(app fiber.Router, authMiddleware *middleware.AuthMiddleware) {
	uploads := app.Group("/uploads", authMiddleware.Required())

	uploads.Post("/presign", h.GeneratePresignedURL)
	uploads.Post("/confirm", h.ConfirmUpload)
}

// PresignRequest is the request for generating a presigned URL
type PresignRequest struct {
	UploadType  string     `json:"upload_type" validate:"required,upload_type"`
	Filename    string     `json:"filename" validate:"required,max=255"`
	ContentType string     `json:"content_type" validate:"required"`
	FileSize    int64      `json:"file_size" validate:"required,min=1"`
	PortfolioID *uuid.UUID `json:"portfolio_id,omitempty"`
}

// GeneratePresignedURL generates a presigned URL for uploading
func (h *UploadHandler) GeneratePresignedURL(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req PresignRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	input := service.PresignInput{
		UploadType:  domain.UploadType(req.UploadType),
		Filename:    req.Filename,
		ContentType: req.ContentType,
		FileSize:    req.FileSize,
		PortfolioID: req.PortfolioID,
	}

	result, err := h.uploadService.GeneratePresignedURL(c.Context(), userID, input)
	if err != nil {
		switch err {
		case service.ErrInvalidUploadType:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidUploadType, "Tipe upload tidak valid")
		case service.ErrFileTooLarge:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeFileTooLarge, "Ukuran file terlalu besar")
		case service.ErrInvalidContentType:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidFileType, "Tipe file tidak didukung")
		case service.ErrPortfolioIDRequired:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Portfolio ID diperlukan untuk tipe upload ini")
		case service.ErrPortfolioNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodePortfolioNotFound, "Portofolio tidak ditemukan")
		case service.ErrUploadForbidden:
			return utils.Error(c, fiber.StatusForbidden, utils.ErrCodeForbidden, "Anda tidak memiliki akses ke portofolio ini")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal membuat presigned URL")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, result, "Presigned URL berhasil dibuat")
}

// ConfirmUploadRequest is the request for confirming an upload
type ConfirmUploadRequest struct {
	UploadID  uuid.UUID `json:"upload_id" validate:"required"`
	ObjectKey string    `json:"object_key" validate:"required"`
}

// ConfirmUpload confirms an upload after the file has been uploaded
func (h *UploadHandler) ConfirmUpload(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req ConfirmUploadRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	result, err := h.uploadService.ConfirmUpload(c.Context(), userID, req.UploadID, req.ObjectKey)
	if err != nil {
		switch err {
		case service.ErrUploadNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeNotFound, "Upload tidak ditemukan")
		case service.ErrUploadForbidden:
			return utils.Error(c, fiber.StatusForbidden, utils.ErrCodeForbidden, "Anda tidak memiliki akses")
		case service.ErrObjectKeyMismatch:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Object key tidak cocok")
		case service.ErrUploadExpired:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeUploadExpired, "Upload sudah kedaluwarsa")
		case service.ErrObjectNotFound:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeObjectNotFound, "File tidak ditemukan di storage")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengkonfirmasi upload")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, result, "Upload berhasil dikonfirmasi")
}
