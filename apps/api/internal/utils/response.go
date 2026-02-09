package utils

import (
	"github.com/gofiber/fiber/v2"
)

// Response represents a standard API response
type Response struct {
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Meta contains pagination metadata
type Meta struct {
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
	TotalPages  int `json:"total_pages"`
	TotalCount  int `json:"total_count"`
}

// ErrorDetail represents a single validation error
type ErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody contains error details
type ErrorBody struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details []ErrorDetail `json:"details,omitempty"`
}

// SuccessResponse sends a successful response with data
func SuccessResponse(c *fiber.Ctx, status int, data interface{}, message string) error {
	return c.Status(status).JSON(Response{
		Data:    data,
		Message: message,
	})
}

// SuccessWithMeta sends a successful response with data and pagination metadata
func SuccessWithMeta(c *fiber.Ctx, status int, data interface{}, meta *Meta) error {
	return c.Status(status).JSON(Response{
		Data: data,
		Meta: meta,
	})
}

// DataResponse sends a response with only data (no message)
func DataResponse(c *fiber.Ctx, status int, data interface{}) error {
	return c.Status(status).JSON(Response{
		Data: data,
	})
}

// MessageResponse sends a response with only a message
func MessageResponse(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{
		"message": message,
	})
}

// Error sends an error response
func Error(c *fiber.Ctx, status int, code, message string) error {
	return c.Status(status).JSON(ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
		},
	})
}

// ErrorWithDetails sends an error response with validation details
func ErrorWithDetails(c *fiber.Ctx, status int, code, message string, details []ErrorDetail) error {
	return c.Status(status).JSON(ErrorResponse{
		Error: ErrorBody{
			Code:    code,
			Message: message,
			Details: details,
		},
	})
}

// Common error codes
const (
	// Auth errors
	ErrCodeInvalidCredentials = "INVALID_CREDENTIALS"
	ErrCodeAccountDisabled    = "ACCOUNT_DISABLED"
	ErrCodeTokenExpired       = "TOKEN_EXPIRED"
	ErrCodeTokenReuseDetected = "TOKEN_REUSE_DETECTED"
	ErrCodeUnauthorized       = "UNAUTHORIZED"
	ErrCodeForbidden          = "FORBIDDEN"

	// Validation errors
	ErrCodeValidationError  = "VALIDATION_ERROR"
	ErrCodeValidationFailed = "VALIDATION_FAILED"
	ErrCodeInvalidInput     = "INVALID_INPUT"

	// Resource errors
	ErrCodeNotFound             = "NOT_FOUND"
	ErrCodeUserNotFound         = "USER_NOT_FOUND"
	ErrCodePortfolioNotFound    = "PORTFOLIO_NOT_FOUND"
	ErrCodeSessionNotFound      = "SESSION_NOT_FOUND"
	ErrCodeTagNotFound          = "TAG_NOT_FOUND"
	ErrCodeMajorNotFound        = "MAJOR_NOT_FOUND"
	ErrCodeClassNotFound        = "CLASS_NOT_FOUND"
	ErrCodeAcademicYearNotFound = "ACADEMIC_YEAR_NOT_FOUND"
	ErrCodeBlockNotFound        = "BLOCK_NOT_FOUND"
	ErrCodeUploadNotFound       = "UPLOAD_NOT_FOUND"

	// Conflict errors
	ErrCodeUsernameTaken     = "USERNAME_TAKEN"
	ErrCodeEmailTaken        = "EMAIL_TAKEN"
	ErrCodeAlreadyFollowing  = "ALREADY_FOLLOWING"
	ErrCodeNotFollowing      = "NOT_FOLLOWING"
	ErrCodeAlreadyLiked      = "ALREADY_LIKED"
	ErrCodeNotLiked          = "NOT_LIKED"
	ErrCodeDuplicateCode     = "DUPLICATE_CODE"
	ErrCodeDuplicateYear     = "DUPLICATE_YEAR"
	ErrCodeDuplicateClass    = "DUPLICATE_CLASS"
	ErrCodeDuplicateUsername = "DUPLICATE_USERNAME"
	ErrCodeDuplicateEmail    = "DUPLICATE_EMAIL"
	ErrCodeDuplicateEntry    = "DUPLICATE_ENTRY"
	ErrCodeDuplicateFollow   = "DUPLICATE_FOLLOW"
	ErrCodeDuplicateLike     = "DUPLICATE_LIKE"
	ErrCodeReservedUsername  = "RESERVED_USERNAME"
	ErrCodeHasDependencies   = "HAS_DEPENDENCIES"

	// Business logic errors
	ErrCodeCannotFollowSelf        = "CANNOT_FOLLOW_SELF"
	ErrCodeInvalidPassword         = "INVALID_PASSWORD"
	ErrCodeRateLimitExceeded       = "RATE_LIMIT_EXCEEDED"
	ErrCodeInvalidStatusTransition = "INVALID_STATUS_TRANSITION"
	ErrCodeInvalidStatusChange     = "INVALID_STATUS_CHANGE"
	ErrCodeIncompletePortfolio     = "INCOMPLETE_PORTFOLIO"
	ErrCodePortfolioNotEditable    = "PORTFOLIO_NOT_EDITABLE"
	ErrCodeMajorInUse              = "MAJOR_IN_USE"
	ErrCodeAcademicYearInUse       = "ACADEMIC_YEAR_IN_USE"
	ErrCodeClassHasStudents        = "CLASS_HAS_STUDENTS"

	// Upload errors
	ErrCodeFileTooLarge       = "FILE_TOO_LARGE"
	ErrCodeInvalidContentType = "INVALID_CONTENT_TYPE"
	ErrCodeInvalidFileType    = "INVALID_FILE_TYPE"
	ErrCodeInvalidUploadType  = "INVALID_UPLOAD_TYPE"
	ErrCodeObjectNotFound     = "OBJECT_NOT_FOUND"
	ErrCodeUploadExpired      = "UPLOAD_EXPIRED"

	// Server errors
	ErrCodeInternalError = "INTERNAL_ERROR"
)

// Common error messages (in Indonesian per API spec)
var ErrMessages = map[string]string{
	ErrCodeInvalidCredentials:      "Username atau password salah",
	ErrCodeAccountDisabled:         "Akun Anda telah dinonaktifkan. Hubungi admin.",
	ErrCodeTokenExpired:            "Refresh token telah expired. Silakan login ulang.",
	ErrCodeTokenReuseDetected:      "Aktivitas mencurigakan terdeteksi. Semua sesi telah diakhiri.",
	ErrCodeUnauthorized:            "Autentikasi diperlukan",
	ErrCodeForbidden:               "Anda tidak memiliki akses",
	ErrCodeValidationError:         "Validasi gagal",
	ErrCodeUserNotFound:            "User tidak ditemukan",
	ErrCodePortfolioNotFound:       "Portfolio tidak ditemukan",
	ErrCodeSessionNotFound:         "Sesi tidak ditemukan",
	ErrCodeUsernameTaken:           "Username sudah digunakan",
	ErrCodeEmailTaken:              "Email sudah digunakan",
	ErrCodeAlreadyFollowing:        "Anda sudah follow user ini",
	ErrCodeNotFollowing:            "Anda belum follow user ini",
	ErrCodeAlreadyLiked:            "Anda sudah like portfolio ini",
	ErrCodeCannotFollowSelf:        "Tidak bisa follow diri sendiri",
	ErrCodeInvalidPassword:         "Password lama tidak sesuai",
	ErrCodeRateLimitExceeded:       "Anda sudah mencapai batas maksimal pembuatan portfolio hari ini (10/hari)",
	ErrCodeInvalidStatusTransition: "Status transisi tidak valid",
	ErrCodeIncompletePortfolio:     "Portfolio belum lengkap",
	ErrCodeFileTooLarge:            "Ukuran file melebihi batas maksimal",
	ErrCodeInvalidContentType:      "Tipe file tidak diizinkan",
	ErrCodeInternalError:           "Terjadi kesalahan pada server",
}

// GetErrorMessage returns the message for an error code
func GetErrorMessage(code string) string {
	if msg, ok := ErrMessages[code]; ok {
		return msg
	}
	return "Unknown error"
}

// NewMeta creates pagination metadata
func NewMeta(currentPage, perPage, totalCount int) *Meta {
	totalPages := totalCount / perPage
	if totalCount%perPage > 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}

	return &Meta{
		CurrentPage: currentPage,
		PerPage:     perPage,
		TotalPages:  totalPages,
		TotalCount:  totalCount,
	}
}

// Pagination contains pagination parameters
type Pagination struct {
	Page  int
	Limit int
}

// DefaultPagination returns default pagination (page 1, limit 20)
func DefaultPagination() Pagination {
	return Pagination{Page: 1, Limit: 20}
}

// Offset calculates the SQL offset from page and limit
func (p Pagination) Offset() int {
	return (p.Page - 1) * p.Limit
}

// Validate ensures pagination values are within acceptable ranges
func (p *Pagination) Validate(maxLimit int) {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 {
		p.Limit = 20
	}
	if p.Limit > maxLimit {
		p.Limit = maxLimit
	}
}
