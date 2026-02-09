package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"grafikarsa/internal/middleware"
	"grafikarsa/internal/repository"
	"grafikarsa/internal/service"
	"grafikarsa/internal/utils"
)

// AdminHandler handles admin endpoints
type AdminHandler struct {
	adminService     *service.AdminService
	userService      *service.UserService
	tagService       *service.TagService
	portfolioService *service.PortfolioService
}

// NewAdminHandler creates a new AdminHandler
func NewAdminHandler(
	adminService *service.AdminService,
	userService *service.UserService,
	tagService *service.TagService,
	portfolioService *service.PortfolioService,
) *AdminHandler {
	return &AdminHandler{
		adminService:     adminService,
		userService:      userService,
		tagService:       tagService,
		portfolioService: portfolioService,
	}
}

// Register registers admin routes
func (h *AdminHandler) Register(app fiber.Router, authMiddleware *middleware.AuthMiddleware) {
	admin := app.Group("/admin", authMiddleware.AdminRequired())

	// Tags
	admin.Get("/tags", h.ListTags)
	admin.Post("/tags", h.CreateTag)
	admin.Put("/tags/:id", h.UpdateTag)
	admin.Delete("/tags/:id", h.DeleteTag)

	// Majors
	admin.Get("/majors", h.ListMajors)
	admin.Post("/majors", h.CreateMajor)
	admin.Put("/majors/:id", h.UpdateMajor)
	admin.Delete("/majors/:id", h.DeleteMajor)

	// Academic Years
	admin.Get("/academic-years", h.ListAcademicYears)
	admin.Post("/academic-years", h.CreateAcademicYear)
	admin.Put("/academic-years/:id", h.UpdateAcademicYear)
	admin.Delete("/academic-years/:id", h.DeleteAcademicYear)

	// Classes
	admin.Get("/classes", h.ListClasses)
	admin.Post("/classes", h.CreateClass)
	admin.Put("/classes/:id", h.UpdateClass)
	admin.Delete("/classes/:id", h.DeleteClass)

	// Users
	admin.Get("/users", h.ListUsers)
	admin.Post("/users", h.CreateUser)
	admin.Put("/users/:id", h.UpdateUser)
	admin.Delete("/users/:id", h.DeleteUser)
	admin.Post("/users/:id/reset-password", h.ResetUserPassword)

	// Portfolio moderation
	admin.Get("/portfolios/pending", h.ListPendingPortfolios)
	admin.Post("/portfolios/:id/approve", h.ApprovePortfolio)
	admin.Post("/portfolios/:id/reject", h.RejectPortfolio)
}

// ==================== TAG ENDPOINTS ====================

// ListTags lists all tags with portfolio count
func (h *AdminHandler) ListTags(c *fiber.Ctx) error {
	tags, err := h.tagService.ListTagsWithCount(c.Context())
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil tags")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, tags, "")
}

// CreateTagRequest is the request for creating a tag
type CreateTagRequest struct {
	Name string `json:"name" validate:"required,min=2,max=50"`
}

// CreateTag creates a new tag
func (h *AdminHandler) CreateTag(c *fiber.Ctx) error {
	var req CreateTagRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	tag, err := h.tagService.CreateTag(c.Context(), req.Name)
	if err != nil {
		if err == service.ErrTagExists {
			return utils.Error(c, fiber.StatusConflict, utils.ErrCodeDuplicateEntry, "Tag sudah ada")
		}
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal membuat tag")
	}

	return utils.SuccessResponse(c, fiber.StatusCreated, tag, "Tag berhasil dibuat")
}

// UpdateTag updates a tag
func (h *AdminHandler) UpdateTag(c *fiber.Ctx) error {
	tagID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID tag tidak valid")
	}

	var req CreateTagRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	tag, err := h.tagService.UpdateTag(c.Context(), tagID, req.Name)
	if err != nil {
		switch err {
		case service.ErrTagNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeNotFound, "Tag tidak ditemukan")
		case service.ErrTagExists:
			return utils.Error(c, fiber.StatusConflict, utils.ErrCodeDuplicateEntry, "Nama tag sudah ada")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal memperbarui tag")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, tag, "Tag berhasil diperbarui")
}

// DeleteTag deletes a tag
func (h *AdminHandler) DeleteTag(c *fiber.Ctx) error {
	tagID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID tag tidak valid")
	}

	if err := h.tagService.DeleteTag(c.Context(), tagID); err != nil {
		if err == service.ErrTagNotFound {
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeNotFound, "Tag tidak ditemukan")
		}
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal menghapus tag")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, nil, "Tag berhasil dihapus")
}

// ==================== MAJOR ENDPOINTS ====================

// ListMajors lists all majors
func (h *AdminHandler) ListMajors(c *fiber.Ctx) error {
	majors, err := h.adminService.ListMajors(c.Context())
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil jurusan")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, majors, "")
}

// CreateMajorRequest is the request for creating a major
type CreateMajorRequest struct {
	Name string `json:"name" validate:"required,min=2,max=100"`
	Code string `json:"code" validate:"required,min=2,max=10"`
}

// CreateMajor creates a new major
func (h *AdminHandler) CreateMajor(c *fiber.Ctx) error {
	var req CreateMajorRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	major, err := h.adminService.CreateMajor(c.Context(), req.Name, req.Code)
	if err != nil {
		if err == service.ErrMajorCodeExists {
			return utils.Error(c, fiber.StatusConflict, utils.ErrCodeDuplicateEntry, "Kode jurusan sudah ada")
		}
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal membuat jurusan")
	}

	return utils.SuccessResponse(c, fiber.StatusCreated, major, "Jurusan berhasil dibuat")
}

// UpdateMajor updates a major
func (h *AdminHandler) UpdateMajor(c *fiber.Ctx) error {
	majorID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID jurusan tidak valid")
	}

	var req CreateMajorRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	major, err := h.adminService.UpdateMajor(c.Context(), majorID, req.Name, req.Code)
	if err != nil {
		switch err {
		case service.ErrMajorNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeNotFound, "Jurusan tidak ditemukan")
		case service.ErrMajorCodeExists:
			return utils.Error(c, fiber.StatusConflict, utils.ErrCodeDuplicateEntry, "Kode jurusan sudah ada")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal memperbarui jurusan")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, major, "Jurusan berhasil diperbarui")
}

// DeleteMajor deletes a major
func (h *AdminHandler) DeleteMajor(c *fiber.Ctx) error {
	majorID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID jurusan tidak valid")
	}

	if err := h.adminService.DeleteMajor(c.Context(), majorID); err != nil {
		switch err {
		case service.ErrMajorNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeNotFound, "Jurusan tidak ditemukan")
		case service.ErrMajorHasClasses:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeHasDependencies, "Jurusan memiliki kelas terkait")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal menghapus jurusan")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, nil, "Jurusan berhasil dihapus")
}

// ==================== ACADEMIC YEAR ENDPOINTS ====================

// ListAcademicYears lists all academic years
func (h *AdminHandler) ListAcademicYears(c *fiber.Ctx) error {
	years, err := h.adminService.ListAcademicYears(c.Context())
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil tahun ajaran")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, years, "")
}

// CreateAcademicYearRequest is the request for creating an academic year
type CreateAcademicYearRequest struct {
	YearStart      int  `json:"year_start" validate:"required,gte=2000,lte=2100"`
	IsActive       bool `json:"is_active"`
	PromotionMonth int  `json:"promotion_month" validate:"min=1,max=12"`
	PromotionDay   int  `json:"promotion_day" validate:"min=1,max=31"`
}

// CreateAcademicYear creates a new academic year
func (h *AdminHandler) CreateAcademicYear(c *fiber.Ctx) error {
	var req CreateAcademicYearRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	// Set defaults
	if req.PromotionMonth == 0 {
		req.PromotionMonth = 7
	}
	if req.PromotionDay == 0 {
		req.PromotionDay = 1
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	year, err := h.adminService.CreateAcademicYear(c.Context(), req.YearStart, req.IsActive, req.PromotionMonth, req.PromotionDay)
	if err != nil {
		if err == service.ErrAcademicYearExists {
			return utils.Error(c, fiber.StatusConflict, utils.ErrCodeDuplicateEntry, "Tahun ajaran sudah ada")
		}
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal membuat tahun ajaran")
	}

	return utils.SuccessResponse(c, fiber.StatusCreated, year, "Tahun ajaran berhasil dibuat")
}

// UpdateAcademicYear updates an academic year
func (h *AdminHandler) UpdateAcademicYear(c *fiber.Ctx) error {
	yearID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID tahun ajaran tidak valid")
	}

	var req CreateAcademicYearRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	year, err := h.adminService.UpdateAcademicYear(c.Context(), yearID, req.YearStart, req.IsActive, req.PromotionMonth, req.PromotionDay)
	if err != nil {
		switch err {
		case service.ErrAcademicYearNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeNotFound, "Tahun ajaran tidak ditemukan")
		case service.ErrAcademicYearExists:
			return utils.Error(c, fiber.StatusConflict, utils.ErrCodeDuplicateEntry, "Tahun ajaran sudah ada")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal memperbarui tahun ajaran")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, year, "Tahun ajaran berhasil diperbarui")
}

// DeleteAcademicYear deletes an academic year
func (h *AdminHandler) DeleteAcademicYear(c *fiber.Ctx) error {
	yearID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID tahun ajaran tidak valid")
	}

	if err := h.adminService.DeleteAcademicYear(c.Context(), yearID); err != nil {
		switch err {
		case service.ErrAcademicYearNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeNotFound, "Tahun ajaran tidak ditemukan")
		case service.ErrAcademicYearHasClasses:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeHasDependencies, "Tahun ajaran memiliki kelas terkait")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal menghapus tahun ajaran")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, nil, "Tahun ajaran berhasil dihapus")
}

// ==================== CLASS ENDPOINTS ====================

// ListClasses lists classes with filtering
func (h *AdminHandler) ListClasses(c *fiber.Ctx) error {
	filter := repository.ClassFilter{
		GradeLevel: c.Query("grade_level"),
		Page:       c.QueryInt("page", 1),
		Limit:      c.QueryInt("limit", 50),
	}

	if yearID := c.Query("academic_year_id"); yearID != "" {
		if id, err := uuid.Parse(yearID); err == nil {
			filter.AcademicYearID = &id
		}
	}

	if majorID := c.Query("major_id"); majorID != "" {
		if id, err := uuid.Parse(majorID); err == nil {
			filter.MajorID = &id
		}
	}

	classes, meta, err := h.adminService.ListClasses(c.Context(), filter)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil kelas")
	}

	return utils.SuccessWithMeta(c, fiber.StatusOK, classes, meta)
}

// CreateClassRequest is the request for creating a class
type CreateClassRequest struct {
	AcademicYearID uuid.UUID `json:"academic_year_id" validate:"required"`
	MajorID        uuid.UUID `json:"major_id" validate:"required"`
	GradeLevel     string    `json:"grade_level" validate:"required,grade_level"`
	GroupLetter    string    `json:"group_letter" validate:"required,group_letter"`
}

// CreateClass creates a new class
func (h *AdminHandler) CreateClass(c *fiber.Ctx) error {
	var req CreateClassRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	class, err := h.adminService.CreateClass(c.Context(), req.AcademicYearID, req.MajorID, req.GradeLevel, req.GroupLetter)
	if err != nil {
		switch err {
		case service.ErrAcademicYearNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeNotFound, "Tahun ajaran tidak ditemukan")
		case service.ErrMajorNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeNotFound, "Jurusan tidak ditemukan")
		case service.ErrClassExists:
			return utils.Error(c, fiber.StatusConflict, utils.ErrCodeDuplicateEntry, "Kelas sudah ada")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal membuat kelas")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusCreated, class, "Kelas berhasil dibuat")
}

// UpdateClassRequest is the request for updating a class
type UpdateClassRequest struct {
	GradeLevel  string `json:"grade_level" validate:"required,grade_level"`
	GroupLetter string `json:"group_letter" validate:"required,group_letter"`
}

// UpdateClass updates a class
func (h *AdminHandler) UpdateClass(c *fiber.Ctx) error {
	classID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID kelas tidak valid")
	}

	var req UpdateClassRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	class, err := h.adminService.UpdateClass(c.Context(), classID, req.GradeLevel, req.GroupLetter)
	if err != nil {
		switch err {
		case service.ErrClassNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeNotFound, "Kelas tidak ditemukan")
		case service.ErrClassExists:
			return utils.Error(c, fiber.StatusConflict, utils.ErrCodeDuplicateEntry, "Kelas sudah ada")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal memperbarui kelas")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, class, "Kelas berhasil diperbarui")
}

// DeleteClass deletes a class
func (h *AdminHandler) DeleteClass(c *fiber.Ctx) error {
	classID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID kelas tidak valid")
	}

	if err := h.adminService.DeleteClass(c.Context(), classID); err != nil {
		switch err {
		case service.ErrClassNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeNotFound, "Kelas tidak ditemukan")
		case service.ErrClassHasStudents:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeHasDependencies, "Kelas memiliki siswa terkait")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal menghapus kelas")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, nil, "Kelas berhasil dihapus")
}

// ==================== USER ADMIN ENDPOINTS ====================

// ListUsers lists users for admin
func (h *AdminHandler) ListUsers(c *fiber.Ctx) error {
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
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil pengguna")
	}

	return utils.SuccessWithMeta(c, fiber.StatusOK, users, meta)
}

// CreateUserRequest is the request for creating a user
type CreateUserRequest struct {
	Username       string     `json:"username" validate:"required,username,not_reserved_username"`
	Email          string     `json:"email" validate:"required,email"`
	Password       string     `json:"password" validate:"required,min=8"`
	Name           string     `json:"name" validate:"required,max=100"`
	Role           string     `json:"role" validate:"required,user_role"`
	Status         string     `json:"status" validate:"omitempty,user_status"`
	NISN           string     `json:"nisn" validate:"omitempty,max=20"`
	NIS            string     `json:"nis" validate:"omitempty,max=20"`
	CurrentClassID *uuid.UUID `json:"current_class_id"`
	EntryYear      *int       `json:"entry_year"`
}

// CreateUser creates a new user (admin)
func (h *AdminHandler) CreateUser(c *fiber.Ctx) error {
	var req CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if req.Status == "" {
		req.Status = "active"
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	input := service.CreateUserInput{
		Username:       req.Username,
		Email:          req.Email,
		Password:       req.Password,
		Name:           req.Name,
		Role:           utils.ToUserRole(req.Role),
		Status:         utils.ToUserStatus(req.Status),
		NISN:           req.NISN,
		NIS:            req.NIS,
		CurrentClassID: req.CurrentClassID,
		EntryYear:      req.EntryYear,
	}

	user, err := h.userService.CreateUser(c.Context(), input)
	if err != nil {
		switch err {
		case service.ErrUsernameTaken:
			return utils.Error(c, fiber.StatusConflict, utils.ErrCodeDuplicateUsername, "Username sudah digunakan")
		case service.ErrUsernameReserved:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeReservedUsername, "Username tidak dapat digunakan")
		case service.ErrEmailTaken:
			return utils.Error(c, fiber.StatusConflict, utils.ErrCodeDuplicateEmail, "Email sudah digunakan")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal membuat pengguna")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusCreated, user.ToListItem(), "Pengguna berhasil dibuat")
}

// UpdateUserRequest is the request for updating a user (admin)
type UpdateUserRequest struct {
	Name           *string    `json:"name" validate:"omitempty,max=100"`
	Username       *string    `json:"username" validate:"omitempty,username,not_reserved_username"`
	Email          *string    `json:"email" validate:"omitempty,email"`
	Role           *string    `json:"role" validate:"omitempty,user_role"`
	Status         *string    `json:"status" validate:"omitempty,user_status"`
	CurrentClassID *uuid.UUID `json:"current_class_id"`
	NISN           *string    `json:"nisn" validate:"omitempty,max=20"`
	NIS            *string    `json:"nis" validate:"omitempty,max=20"`
	EntryYear      *int       `json:"entry_year"`
	GraduationYear *int       `json:"graduation_year"`
}

// UpdateUser updates a user (admin)
func (h *AdminHandler) UpdateUser(c *fiber.Ctx) error {
	userID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID pengguna tidak valid")
	}

	var req UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	input := service.UpdateUserInput{
		Name:           req.Name,
		Username:       req.Username,
		Email:          req.Email,
		CurrentClassID: req.CurrentClassID,
		NISN:           req.NISN,
		NIS:            req.NIS,
		EntryYear:      req.EntryYear,
		GraduationYear: req.GraduationYear,
	}

	if req.Role != nil {
		role := utils.ToUserRole(*req.Role)
		input.Role = &role
	}
	if req.Status != nil {
		status := utils.ToUserStatus(*req.Status)
		input.Status = &status
	}

	user, err := h.userService.UpdateUser(c.Context(), userID, input)
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeUserNotFound, "Pengguna tidak ditemukan")
		case service.ErrUsernameTaken:
			return utils.Error(c, fiber.StatusConflict, utils.ErrCodeDuplicateUsername, "Username sudah digunakan")
		case service.ErrUsernameReserved:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeReservedUsername, "Username tidak dapat digunakan")
		case service.ErrEmailTaken:
			return utils.Error(c, fiber.StatusConflict, utils.ErrCodeDuplicateEmail, "Email sudah digunakan")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal memperbarui pengguna")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, user.ToListItem(), "Pengguna berhasil diperbarui")
}

// DeleteUser deletes a user (admin)
func (h *AdminHandler) DeleteUser(c *fiber.Ctx) error {
	userID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID pengguna tidak valid")
	}

	if err := h.userService.DeleteUser(c.Context(), userID); err != nil {
		if err == service.ErrUserNotFound {
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeUserNotFound, "Pengguna tidak ditemukan")
		}
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal menghapus pengguna")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, nil, "Pengguna berhasil dihapus")
}

// ResetPasswordRequest is the request for resetting a user's password
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// ResetUserPassword resets a user's password (admin)
func (h *AdminHandler) ResetUserPassword(c *fiber.Ctx) error {
	userID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID pengguna tidak valid")
	}

	var req ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	if err := h.userService.ResetPassword(c.Context(), userID, req.NewPassword); err != nil {
		if err == service.ErrUserNotFound {
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodeUserNotFound, "Pengguna tidak ditemukan")
		}
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mereset password")
	}

	return utils.SuccessResponse(c, fiber.StatusOK, nil, "Password berhasil direset")
}

// ==================== PORTFOLIO MODERATION ====================

// ListPendingPortfolios lists portfolios pending review
func (h *AdminHandler) ListPendingPortfolios(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	portfolios, meta, err := h.portfolioService.ListPendingReview(c.Context(), page, limit)
	if err != nil {
		return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal mengambil portofolio")
	}

	return utils.SuccessWithMeta(c, fiber.StatusOK, portfolios, meta)
}

// ApprovePortfolio approves a portfolio
func (h *AdminHandler) ApprovePortfolio(c *fiber.Ctx) error {
	adminID := middleware.GetUserID(c)

	portfolioID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID portofolio tidak valid")
	}

	if err := h.portfolioService.ApprovePortfolio(c.Context(), portfolioID, adminID); err != nil {
		switch err {
		case service.ErrPortfolioNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodePortfolioNotFound, "Portofolio tidak ditemukan")
		case service.ErrInvalidStatusTransition:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidStatusChange, "Portofolio tidak dalam status pending review")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal menyetujui portofolio")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, nil, "Portofolio berhasil disetujui")
}

// RejectPortfolioRequest is the request for rejecting a portfolio
type RejectPortfolioRequest struct {
	Note string `json:"note" validate:"required,min=10,max=1000"`
}

// RejectPortfolio rejects a portfolio
func (h *AdminHandler) RejectPortfolio(c *fiber.Ctx) error {
	adminID := middleware.GetUserID(c)

	portfolioID, err := utils.ParseUUID(c.Params("id"))
	if err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "ID portofolio tidak valid")
	}

	var req RejectPortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidInput, "Format request tidak valid")
	}

	if errs := utils.Validate(&req); len(errs) > 0 {
		return utils.ErrorWithDetails(c, fiber.StatusBadRequest, utils.ErrCodeValidationFailed, "Validasi gagal", errs)
	}

	if err := h.portfolioService.RejectPortfolio(c.Context(), portfolioID, adminID, req.Note); err != nil {
		switch err {
		case service.ErrPortfolioNotFound:
			return utils.Error(c, fiber.StatusNotFound, utils.ErrCodePortfolioNotFound, "Portofolio tidak ditemukan")
		case service.ErrInvalidStatusTransition:
			return utils.Error(c, fiber.StatusBadRequest, utils.ErrCodeInvalidStatusChange, "Portofolio tidak dalam status pending review")
		default:
			return utils.Error(c, fiber.StatusInternalServerError, utils.ErrCodeInternalError, "Gagal menolak portofolio")
		}
	}

	return utils.SuccessResponse(c, fiber.StatusOK, nil, "Portofolio berhasil ditolak")
}
