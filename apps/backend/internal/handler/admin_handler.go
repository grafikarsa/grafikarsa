package handler

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/middleware"
	"github.com/grafikarsa/backend/internal/repository"
	"github.com/grafikarsa/backend/internal/service"
	"golang.org/x/crypto/bcrypt"
)

type AdminHandler struct {
	adminRepo     *repository.AdminRepository
	userRepo      *repository.UserRepository
	portfolioRepo *repository.PortfolioRepository
	notifService  *service.NotificationService
}

func NewAdminHandler(adminRepo *repository.AdminRepository, userRepo *repository.UserRepository, portfolioRepo *repository.PortfolioRepository, notifService *service.NotificationService) *AdminHandler {
	return &AdminHandler{
		adminRepo:     adminRepo,
		userRepo:      userRepo,
		portfolioRepo: portfolioRepo,
		notifService:  notifService,
	}
}

// Jurusan Handlers
func (h *AdminHandler) ListJurusan(c *fiber.Ctx) error {
	jurusan, err := h.adminRepo.ListJurusan()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil data jurusan"))
	}

	var result []dto.JurusanDetailDTO
	for _, j := range jurusan {
		result = append(result, dto.JurusanDetailDTO{
			ID:        j.ID,
			Nama:      j.Nama,
			Kode:      j.Kode,
			CreatedAt: j.CreatedAt,
			UpdatedAt: j.UpdatedAt,
		})
	}

	return c.JSON(dto.SuccessResponse(result, ""))
}

func (h *AdminHandler) CreateJurusan(c *fiber.Ctx) error {
	var req dto.CreateJurusanRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	if req.Nama == "" || req.Kode == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Validasi gagal",
			dto.ErrorDetail{Field: "nama", Message: "Nama wajib diisi"},
			dto.ErrorDetail{Field: "kode", Message: "Kode wajib diisi"},
		))
	}

	exists, _ := h.adminRepo.JurusanKodeExists(req.Kode, nil)
	if exists {
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse("DUPLICATE_KODE", "Kode jurusan sudah digunakan"))
	}

	jurusan := &domain.Jurusan{Nama: req.Nama, Kode: req.Kode}
	if err := h.adminRepo.CreateJurusan(jurusan); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal membuat jurusan"))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse(dto.JurusanDetailDTO{
		ID: jurusan.ID, Nama: jurusan.Nama, Kode: jurusan.Kode, CreatedAt: jurusan.CreatedAt,
	}, "Jurusan berhasil dibuat"))
}

func (h *AdminHandler) UpdateJurusan(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	jurusan, err := h.adminRepo.FindJurusanByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Jurusan tidak ditemukan"))
	}

	var req dto.UpdateJurusanRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	if req.Nama != nil {
		jurusan.Nama = *req.Nama
	}
	if req.Kode != nil {
		exists, _ := h.adminRepo.JurusanKodeExists(*req.Kode, &id)
		if exists {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse("DUPLICATE_KODE", "Kode jurusan sudah digunakan"))
		}
		jurusan.Kode = *req.Kode
	}

	if err := h.adminRepo.UpdateJurusan(jurusan); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal memperbarui jurusan"))
	}

	return c.JSON(dto.SuccessResponse(dto.JurusanDetailDTO{
		ID: jurusan.ID, Nama: jurusan.Nama, Kode: jurusan.Kode, UpdatedAt: jurusan.UpdatedAt,
	}, "Jurusan berhasil diperbarui"))
}

func (h *AdminHandler) DeleteJurusan(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	hasKelas, _ := h.adminRepo.JurusanHasKelas(id)
	if hasKelas {
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse("JURUSAN_IN_USE", "Jurusan tidak bisa dihapus karena masih digunakan oleh kelas"))
	}

	if err := h.adminRepo.DeleteJurusan(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal menghapus jurusan"))
	}

	return c.JSON(dto.SuccessResponse(nil, "Jurusan berhasil dihapus"))
}

// Tahun Ajaran Handlers
func (h *AdminHandler) ListTahunAjaran(c *fiber.Ctx) error {
	tahunAjaran, err := h.adminRepo.ListTahunAjaran()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil data tahun ajaran"))
	}

	var result []dto.TahunAjaranDTO
	for _, ta := range tahunAjaran {
		result = append(result, dto.TahunAjaranDTO{
			ID: ta.ID, TahunMulai: ta.TahunMulai, IsActive: ta.IsActive,
			PromotionMonth: ta.PromotionMonth, PromotionDay: ta.PromotionDay, CreatedAt: ta.CreatedAt,
		})
	}

	return c.JSON(dto.SuccessResponse(result, ""))
}

func (h *AdminHandler) CreateTahunAjaran(c *fiber.Ctx) error {
	var req dto.CreateTahunAjaranRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	if req.TahunMulai == 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Validasi gagal",
			dto.ErrorDetail{Field: "tahun_mulai", Message: "Tahun mulai wajib diisi"},
		))
	}

	exists, _ := h.adminRepo.TahunAjaranExists(req.TahunMulai, nil)
	if exists {
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse("DUPLICATE_TAHUN", "Tahun ajaran sudah ada"))
	}

	if req.IsActive {
		h.adminRepo.DeactivateAllTahunAjaran()
	}

	ta := &domain.TahunAjaran{
		TahunMulai: req.TahunMulai, IsActive: req.IsActive,
		PromotionMonth: req.PromotionMonth, PromotionDay: req.PromotionDay,
	}
	if ta.PromotionMonth == 0 {
		ta.PromotionMonth = 7
	}
	if ta.PromotionDay == 0 {
		ta.PromotionDay = 1
	}

	if err := h.adminRepo.CreateTahunAjaran(ta); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal membuat tahun ajaran"))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse(dto.TahunAjaranDTO{
		ID: ta.ID, TahunMulai: ta.TahunMulai, IsActive: ta.IsActive,
		PromotionMonth: ta.PromotionMonth, PromotionDay: ta.PromotionDay, CreatedAt: ta.CreatedAt,
	}, "Tahun ajaran berhasil dibuat"))
}

func (h *AdminHandler) UpdateTahunAjaran(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	ta, err := h.adminRepo.FindTahunAjaranByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Tahun ajaran tidak ditemukan"))
	}

	var req dto.UpdateTahunAjaranRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	if req.IsActive != nil && *req.IsActive {
		h.adminRepo.DeactivateAllTahunAjaran()
		ta.IsActive = true
	} else if req.IsActive != nil {
		ta.IsActive = *req.IsActive
	}
	if req.PromotionMonth != nil {
		ta.PromotionMonth = *req.PromotionMonth
	}
	if req.PromotionDay != nil {
		ta.PromotionDay = *req.PromotionDay
	}

	if err := h.adminRepo.UpdateTahunAjaran(ta); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal memperbarui tahun ajaran"))
	}

	return c.JSON(dto.SuccessResponse(dto.TahunAjaranDTO{
		ID: ta.ID, TahunMulai: ta.TahunMulai, IsActive: ta.IsActive,
		PromotionMonth: ta.PromotionMonth, PromotionDay: ta.PromotionDay, CreatedAt: ta.CreatedAt,
	}, "Tahun ajaran berhasil diperbarui"))
}

func (h *AdminHandler) DeleteTahunAjaran(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	hasKelas, _ := h.adminRepo.TahunAjaranHasKelas(id)
	if hasKelas {
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse("TAHUN_AJARAN_IN_USE", "Tahun ajaran tidak bisa dihapus karena masih digunakan oleh kelas"))
	}

	if err := h.adminRepo.DeleteTahunAjaran(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal menghapus tahun ajaran"))
	}

	return c.JSON(dto.SuccessResponse(nil, "Tahun ajaran berhasil dihapus"))
}

// Kelas Handlers
func (h *AdminHandler) ListKelas(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	var tahunAjaranID, jurusanID *uuid.UUID
	var tingkat *int
	if id := c.Query("tahun_ajaran_id"); id != "" {
		parsed, _ := uuid.Parse(id)
		tahunAjaranID = &parsed
	}
	if id := c.Query("jurusan_id"); id != "" {
		parsed, _ := uuid.Parse(id)
		jurusanID = &parsed
	}
	if t := c.Query("tingkat"); t != "" {
		parsed, _ := strconv.Atoi(t)
		tingkat = &parsed
	}

	kelas, total, err := h.adminRepo.ListKelas(tahunAjaranID, jurusanID, tingkat, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil data kelas"))
	}

	var result []dto.KelasDetailDTO
	for _, k := range kelas {
		kDTO := dto.KelasDetailDTO{
			ID: k.ID, Nama: k.Nama, Tingkat: k.Tingkat, Rombel: k.Rombel, CreatedAt: k.CreatedAt,
		}
		if k.TahunAjaran != nil {
			kDTO.TahunAjaran = &dto.TahunAjaranDTO{
				ID: k.TahunAjaran.ID, TahunMulai: k.TahunAjaran.TahunMulai, IsActive: k.TahunAjaran.IsActive,
			}
		}
		if k.Jurusan != nil {
			kDTO.Jurusan = &dto.JurusanDTO{ID: k.Jurusan.ID, Nama: k.Jurusan.Nama, Kode: k.Jurusan.Kode}
		}
		kDTO.StudentCount, _ = h.adminRepo.GetKelasStudentCount(k.ID)
		result = append(result, kDTO)
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return c.JSON(dto.SuccessWithMeta(result, &dto.Meta{CurrentPage: page, PerPage: limit, TotalPages: totalPages, TotalCount: total}))
}

func (h *AdminHandler) CreateKelas(c *fiber.Ctx) error {
	var req dto.CreateKelasRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	if req.TahunAjaranID == uuid.Nil || req.JurusanID == uuid.Nil || req.Tingkat == 0 || req.Rombel == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Validasi gagal"))
	}

	if req.Tingkat < 10 || req.Tingkat > 12 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Validasi gagal",
			dto.ErrorDetail{Field: "tingkat", Message: "Tingkat harus 10, 11, atau 12"},
		))
	}

	exists, _ := h.adminRepo.KelasExists(req.TahunAjaranID, req.JurusanID, req.Tingkat, req.Rombel, nil)
	if exists {
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse("DUPLICATE_KELAS", "Kelas dengan kombinasi tahun ajaran, jurusan, tingkat, dan rombel yang sama sudah ada"))
	}

	jurusan, err := h.adminRepo.FindJurusanByID(req.JurusanID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Jurusan tidak ditemukan"))
	}

	tingkatRomawi := map[int]string{10: "X", 11: "XI", 12: "XII"}
	nama := tingkatRomawi[req.Tingkat] + "-" + jurusan.Kode + "-" + req.Rombel

	kelas := &domain.Kelas{
		TahunAjaranID: req.TahunAjaranID, JurusanID: req.JurusanID,
		Tingkat: req.Tingkat, Rombel: req.Rombel, Nama: nama,
	}

	if err := h.adminRepo.CreateKelas(kelas); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal membuat kelas"))
	}

	kelas, _ = h.adminRepo.FindKelasByID(kelas.ID)
	result := dto.KelasDetailDTO{ID: kelas.ID, Nama: kelas.Nama, Tingkat: kelas.Tingkat, Rombel: kelas.Rombel, CreatedAt: kelas.CreatedAt}
	if kelas.TahunAjaran != nil {
		result.TahunAjaran = &dto.TahunAjaranDTO{ID: kelas.TahunAjaran.ID, TahunMulai: kelas.TahunAjaran.TahunMulai}
	}
	if kelas.Jurusan != nil {
		result.Jurusan = &dto.JurusanDTO{ID: kelas.Jurusan.ID, Nama: kelas.Jurusan.Nama, Kode: kelas.Jurusan.Kode}
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse(result, "Kelas berhasil dibuat"))
}

func (h *AdminHandler) UpdateKelas(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	kelas, err := h.adminRepo.FindKelasByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Kelas tidak ditemukan"))
	}

	var req dto.UpdateKelasRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	if req.Rombel != nil {
		exists, _ := h.adminRepo.KelasExists(kelas.TahunAjaranID, kelas.JurusanID, kelas.Tingkat, *req.Rombel, &id)
		if exists {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse("DUPLICATE_KELAS", "Kelas sudah ada"))
		}
		kelas.Rombel = *req.Rombel
		tingkatRomawi := map[int]string{10: "X", 11: "XI", 12: "XII"}
		kelas.Nama = tingkatRomawi[kelas.Tingkat] + "-" + kelas.Jurusan.Kode + "-" + kelas.Rombel
	}

	if err := h.adminRepo.UpdateKelas(kelas); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal memperbarui kelas"))
	}

	return c.JSON(dto.SuccessResponse(map[string]interface{}{
		"id": kelas.ID, "nama": kelas.Nama, "tingkat": kelas.Tingkat, "rombel": kelas.Rombel, "updated_at": kelas.UpdatedAt,
	}, "Kelas berhasil diperbarui"))
}

func (h *AdminHandler) DeleteKelas(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	hasStudents, _ := h.adminRepo.KelasHasStudents(id)
	if hasStudents {
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse("KELAS_IN_USE", "Kelas tidak bisa dihapus karena masih memiliki siswa"))
	}

	if err := h.adminRepo.DeleteKelas(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal menghapus kelas"))
	}

	return c.JSON(dto.SuccessResponse(nil, "Kelas berhasil dihapus"))
}

func (h *AdminHandler) GetKelasStudents(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	kelasID := &id
	users, total, err := h.adminRepo.ListUsers("", nil, kelasID, nil, nil, 1, 100)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil data siswa"))
	}

	var result []map[string]interface{}
	for _, u := range users {
		result = append(result, map[string]interface{}{
			"id": u.ID, "username": u.Username, "nama": u.Nama, "nisn": u.NISN, "nis": u.NIS, "avatar_url": u.AvatarURL,
		})
	}

	return c.JSON(dto.SuccessWithMeta(result, &dto.Meta{TotalCount: total}))
}

// Tags Handlers
func (h *AdminHandler) ListTags(c *fiber.Ctx) error {
	search := c.Query("search")
	tags, err := h.adminRepo.ListTags(search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil data tags"))
	}

	var result []dto.TagDetailDTO
	for _, t := range tags {
		count, _ := h.adminRepo.GetTagPortfolioCount(t.ID)
		result = append(result, dto.TagDetailDTO{ID: t.ID, Nama: t.Nama, PortfolioCount: count, CreatedAt: t.CreatedAt})
	}

	return c.JSON(dto.SuccessResponse(result, ""))
}

func (h *AdminHandler) CreateTag(c *fiber.Ctx) error {
	var req dto.CreateTagRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	if req.Nama == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Validasi gagal",
			dto.ErrorDetail{Field: "nama", Message: "Nama wajib diisi"},
		))
	}

	exists, _ := h.adminRepo.TagNameExists(req.Nama, nil)
	if exists {
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse("DUPLICATE_TAG", "Tag dengan nama tersebut sudah ada"))
	}

	tag := &domain.Tag{Nama: req.Nama}
	if err := h.adminRepo.CreateTag(tag); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal membuat tag"))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse(dto.TagDetailDTO{ID: tag.ID, Nama: tag.Nama, CreatedAt: tag.CreatedAt}, "Tag berhasil dibuat"))
}

func (h *AdminHandler) UpdateTag(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	tag, err := h.adminRepo.FindTagByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Tag tidak ditemukan"))
	}

	var req dto.UpdateTagRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	if req.Nama != nil {
		exists, _ := h.adminRepo.TagNameExists(*req.Nama, &id)
		if exists {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse("DUPLICATE_TAG", "Tag dengan nama tersebut sudah ada"))
		}
		tag.Nama = *req.Nama
	}

	if err := h.adminRepo.UpdateTag(tag); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal memperbarui tag"))
	}

	return c.JSON(dto.SuccessResponse(map[string]interface{}{"id": tag.ID, "nama": tag.Nama, "updated_at": tag.UpdatedAt}, "Tag berhasil diperbarui"))
}

func (h *AdminHandler) DeleteTag(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	if err := h.adminRepo.DeleteTag(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal menghapus tag"))
	}

	return c.JSON(dto.SuccessResponse(nil, "Tag berhasil dihapus"))
}

// Series Management Handlers
func (h *AdminHandler) ListSeries(c *fiber.Ctx) error {
	search := c.Query("search")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	series, total, err := h.adminRepo.ListSeries(search, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil data series"))
	}

	var result []dto.SeriesDTO
	for _, s := range series {
		portfolioCount, _ := h.adminRepo.GetSeriesPortfolioCount(s.ID)
		result = append(result, dto.SeriesDTO{
			ID:             s.ID,
			Nama:           s.Nama,
			Deskripsi:      s.Deskripsi,
			IsActive:       s.IsActive,
			BlockCount:     len(s.Blocks),
			PortfolioCount: portfolioCount,
			CreatedAt:      s.CreatedAt,
		})
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return c.JSON(dto.SuccessWithMeta(result, &dto.Meta{
		CurrentPage: page,
		PerPage:     limit,
		TotalPages:  totalPages,
		TotalCount:  total,
	}))
}

func (h *AdminHandler) GetSeries(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	series, err := h.adminRepo.FindSeriesByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("SERIES_NOT_FOUND", "Series tidak ditemukan"))
	}

	portfolioCount, _ := h.adminRepo.GetSeriesPortfolioCount(series.ID)

	return c.JSON(dto.SuccessResponse(dto.SeriesDetailDTO{
		ID:             series.ID,
		Nama:           series.Nama,
		Deskripsi:      series.Deskripsi,
		IsActive:       series.IsActive,
		Blocks:         dto.SeriesBlocksToDTOs(series.Blocks),
		PortfolioCount: portfolioCount,
		CreatedAt:      series.CreatedAt,
	}, ""))
}

func (h *AdminHandler) CreateSeries(c *fiber.Ctx) error {
	var req dto.CreateSeriesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	if req.Nama == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Validasi gagal",
			dto.ErrorDetail{Field: "nama", Message: "Nama series wajib diisi"},
		))
	}

	if len(req.Blocks) == 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Validasi gagal",
			dto.ErrorDetail{Field: "blocks", Message: "Series harus memiliki minimal 1 block"},
		))
	}

	// Check duplicate
	exists, _ := h.adminRepo.SeriesNameExists(req.Nama, nil)
	if exists {
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse("DUPLICATE_ERROR", "Series dengan nama tersebut sudah ada"))
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	// Build blocks
	var blocks []domain.SeriesBlock
	for i, b := range req.Blocks {
		blocks = append(blocks, domain.SeriesBlock{
			BlockType:  domain.ContentBlockType(b.BlockType),
			BlockOrder: i,
			Instruksi:  b.Instruksi,
		})
	}

	series := &domain.Series{
		Nama:      req.Nama,
		Deskripsi: req.Deskripsi,
		IsActive:  isActive,
		Blocks:    blocks,
	}

	if err := h.adminRepo.CreateSeries(series); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal membuat series"))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse(dto.SeriesDetailDTO{
		ID:             series.ID,
		Nama:           series.Nama,
		Deskripsi:      series.Deskripsi,
		IsActive:       series.IsActive,
		Blocks:         dto.SeriesBlocksToDTOs(series.Blocks),
		PortfolioCount: 0,
		CreatedAt:      series.CreatedAt,
	}, "Series berhasil dibuat"))
}

func (h *AdminHandler) UpdateSeries(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	series, err := h.adminRepo.FindSeriesByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("SERIES_NOT_FOUND", "Series tidak ditemukan"))
	}

	var req dto.UpdateSeriesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	if req.Nama != nil {
		// Check duplicate
		exists, _ := h.adminRepo.SeriesNameExists(*req.Nama, &id)
		if exists {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse("DUPLICATE_ERROR", "Series dengan nama tersebut sudah ada"))
		}
		series.Nama = *req.Nama
	}
	if req.Deskripsi != nil {
		series.Deskripsi = req.Deskripsi
	}
	if req.IsActive != nil {
		series.IsActive = *req.IsActive
	}

	if err := h.adminRepo.UpdateSeries(series); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal memperbarui series"))
	}

	// Update blocks if provided
	if req.Blocks != nil {
		if len(req.Blocks) == 0 {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Validasi gagal",
				dto.ErrorDetail{Field: "blocks", Message: "Series harus memiliki minimal 1 block"},
			))
		}
		var blocks []domain.SeriesBlock
		for i, b := range req.Blocks {
			blocks = append(blocks, domain.SeriesBlock{
				BlockType:  domain.ContentBlockType(b.BlockType),
				BlockOrder: i,
				Instruksi:  b.Instruksi,
			})
		}
		if err := h.adminRepo.UpdateSeriesBlocks(id, blocks); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal memperbarui blocks"))
		}
	}

	// Reload series with blocks
	series, _ = h.adminRepo.FindSeriesByID(id)
	portfolioCount, _ := h.adminRepo.GetSeriesPortfolioCount(series.ID)

	return c.JSON(dto.SuccessResponse(dto.SeriesDetailDTO{
		ID:             series.ID,
		Nama:           series.Nama,
		Deskripsi:      series.Deskripsi,
		IsActive:       series.IsActive,
		Blocks:         dto.SeriesBlocksToDTOs(series.Blocks),
		PortfolioCount: portfolioCount,
		CreatedAt:      series.CreatedAt,
	}, "Series berhasil diperbarui"))
}

func (h *AdminHandler) DeleteSeries(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	if err := h.adminRepo.DeleteSeries(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal menghapus series"))
	}

	return c.JSON(dto.SuccessResponse(nil, "Series berhasil dihapus"))
}

// User Management Handlers
func (h *AdminHandler) ListUsers(c *fiber.Ctx) error {
	search := c.Query("search")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	var role *string
	var kelasID, jurusanID *uuid.UUID
	var isActive *bool

	if r := c.Query("role"); r != "" {
		role = &r
	}
	if id := c.Query("kelas_id"); id != "" {
		parsed, _ := uuid.Parse(id)
		kelasID = &parsed
	}
	if id := c.Query("jurusan_id"); id != "" {
		parsed, _ := uuid.Parse(id)
		jurusanID = &parsed
	}
	if a := c.Query("is_active"); a != "" {
		active := a == "true"
		isActive = &active
	}

	users, total, err := h.adminRepo.ListUsers(search, role, kelasID, jurusanID, isActive, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil data users"))
	}

	var result []dto.AdminUserDTO
	for _, u := range users {
		uDTO := dto.AdminUserDTO{
			ID: u.ID, Username: u.Username, Email: u.Email, Nama: u.Nama, AvatarURL: u.AvatarURL,
			Role: string(u.Role), NISN: u.NISN, NIS: u.NIS, TahunMasuk: u.TahunMasuk, TahunLulus: u.TahunLulus,
			IsActive: u.IsActive, LastLoginAt: u.LastLoginAt, CreatedAt: u.CreatedAt,
		}
		if u.Kelas != nil {
			uDTO.Kelas = &dto.KelasDTO{ID: u.Kelas.ID, Nama: u.Kelas.Nama}
			if u.Kelas.Jurusan != nil {
				uDTO.Jurusan = &dto.JurusanDTO{ID: u.Kelas.Jurusan.ID, Nama: u.Kelas.Jurusan.Nama}
			}
		}
		result = append(result, uDTO)
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return c.JSON(dto.SuccessWithMeta(result, &dto.Meta{CurrentPage: page, PerPage: limit, TotalPages: totalPages, TotalCount: total}))
}

func (h *AdminHandler) GetUser(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	user, err := h.userRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("USER_NOT_FOUND", "User tidak ditemukan"))
	}

	result := dto.AdminUserDetailDTO{
		AdminUserDTO: dto.AdminUserDTO{
			ID: user.ID, Username: user.Username, Email: user.Email, Nama: user.Nama, AvatarURL: user.AvatarURL,
			Role: string(user.Role), NISN: user.NISN, NIS: user.NIS, TahunMasuk: user.TahunMasuk, TahunLulus: user.TahunLulus,
			IsActive: user.IsActive, LastLoginAt: user.LastLoginAt, CreatedAt: user.CreatedAt,
		},
		Bio: user.Bio, BannerURL: user.BannerURL, UpdatedAt: user.UpdatedAt,
	}

	if user.Kelas != nil {
		result.Kelas = &dto.KelasDTO{ID: user.Kelas.ID, Nama: user.Kelas.Nama}
		if user.Kelas.Jurusan != nil {
			result.Jurusan = &dto.JurusanDTO{ID: user.Kelas.Jurusan.ID, Nama: user.Kelas.Jurusan.Nama}
		}
	}

	classHistory, _ := h.userRepo.GetClassHistory(user.ID)
	for _, ch := range classHistory {
		kelasNama := ""
		tahunAjaran := 0
		if ch.Kelas != nil {
			kelasNama = ch.Kelas.Nama
		}
		if ch.TahunAjaran != nil {
			tahunAjaran = ch.TahunAjaran.TahunMulai
		}
		result.ClassHistory = append(result.ClassHistory, dto.ClassHistoryDTO{KelasNama: kelasNama, TahunAjaran: tahunAjaran})
	}
	for _, sl := range user.SocialLinks {
		result.SocialLinks = append(result.SocialLinks, dto.SocialLinkDTO{Platform: string(sl.Platform), URL: sl.URL})
	}

	return c.JSON(dto.SuccessResponse(result, ""))
}

func (h *AdminHandler) CreateUser(c *fiber.Ctx) error {
	var req dto.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	if req.Username == "" || req.Email == "" || req.Password == "" || req.Nama == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Validasi gagal"))
	}

	if len(req.Password) < 8 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Validasi gagal",
			dto.ErrorDetail{Field: "password", Message: "Password minimal 8 karakter"},
		))
	}

	existingUser, _ := h.userRepo.FindByUsername(req.Username)
	if existingUser != nil {
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse("DUPLICATE_USERNAME", "Username sudah digunakan"))
	}

	existingEmail, _ := h.userRepo.FindByEmail(req.Email)
	if existingEmail != nil {
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse("DUPLICATE_EMAIL", "Email sudah digunakan"))
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	role := domain.RoleStudent
	if req.Role != "" {
		role = domain.UserRole(req.Role)
	}

	user := &domain.User{
		Username: req.Username, Email: req.Email, PasswordHash: string(hashedPassword), Nama: req.Nama,
		Role: role, NISN: req.NISN, NIS: req.NIS, KelasID: req.KelasID, TahunMasuk: req.TahunMasuk, IsActive: true,
	}

	if err := h.userRepo.Create(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal membuat user"))
	}

	user, _ = h.userRepo.FindByID(user.ID)
	result := dto.AdminUserDTO{
		ID: user.ID, Username: user.Username, Email: user.Email, Nama: user.Nama,
		Role: string(user.Role), NISN: user.NISN, NIS: user.NIS, TahunMasuk: user.TahunMasuk,
		IsActive: user.IsActive, CreatedAt: user.CreatedAt,
	}
	if user.Kelas != nil {
		result.Kelas = &dto.KelasDTO{ID: user.Kelas.ID, Nama: user.Kelas.Nama}
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse(result, "User berhasil dibuat"))
}

func (h *AdminHandler) UpdateUser(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	user, err := h.userRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("USER_NOT_FOUND", "User tidak ditemukan"))
	}

	var req dto.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	// Check username uniqueness if changed
	if req.Username != nil && *req.Username != user.Username {
		exists, _ := h.userRepo.UsernameExists(*req.Username, &id)
		if exists {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse("USERNAME_TAKEN", "Username sudah digunakan"))
		}
		user.Username = *req.Username
	}

	// Check email uniqueness if changed
	if req.Email != nil && *req.Email != user.Email {
		exists, _ := h.userRepo.EmailExists(*req.Email, &id)
		if exists {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse("DUPLICATE_EMAIL", "Email sudah digunakan"))
		}
		user.Email = *req.Email
	}

	if req.Nama != nil {
		user.Nama = *req.Nama
	}
	if req.Role != nil {
		user.Role = domain.UserRole(*req.Role)
	}
	if req.NISN != nil {
		user.NISN = req.NISN
	}
	if req.NIS != nil {
		user.NIS = req.NIS
	}
	if req.KelasID != nil {
		user.KelasID = req.KelasID
	}
	if req.TahunMasuk != nil {
		user.TahunMasuk = req.TahunMasuk
	}
	if req.TahunLulus != nil {
		user.TahunLulus = req.TahunLulus
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.AvatarURL != nil {
		user.AvatarURL = req.AvatarURL
	}
	if req.BannerURL != nil {
		user.BannerURL = req.BannerURL
	}

	if err := h.userRepo.Update(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal memperbarui user"))
	}

	return c.JSON(dto.SuccessResponse(map[string]interface{}{
		"id": user.ID, "username": user.Username, "email": user.Email, "nama": user.Nama, "role": user.Role,
		"nisn": user.NISN, "nis": user.NIS, "tahun_masuk": user.TahunMasuk, "tahun_lulus": user.TahunLulus,
		"avatar_url": user.AvatarURL, "banner_url": user.BannerURL, "updated_at": user.UpdatedAt,
	}, "User berhasil diperbarui"))
}

func (h *AdminHandler) ResetUserPassword(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	user, err := h.userRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("USER_NOT_FOUND", "User tidak ditemukan"))
	}

	var req dto.ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	if len(req.NewPassword) < 8 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Validasi gagal",
			dto.ErrorDetail{Field: "new_password", Message: "Password minimal 8 karakter"},
		))
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	user.PasswordHash = string(hashedPassword)

	if err := h.userRepo.Update(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal reset password"))
	}

	return c.JSON(dto.SuccessResponse(nil, "Password user berhasil direset"))
}

func (h *AdminHandler) DeleteUser(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	if err := h.userRepo.Delete(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal menghapus user"))
	}

	return c.JSON(dto.SuccessResponse(nil, "User berhasil dihapus"))
}

func (h *AdminHandler) CheckUsername(c *fiber.Ctx) error {
	username := c.Query("username")
	excludeID := c.Query("exclude_id")

	if username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Username wajib diisi"))
	}

	var excludeUUID *uuid.UUID
	if excludeID != "" {
		parsed, err := uuid.Parse(excludeID)
		if err == nil {
			excludeUUID = &parsed
		}
	}

	exists, _ := h.userRepo.UsernameExists(username, excludeUUID)

	return c.JSON(dto.SuccessResponse(map[string]interface{}{
		"username":  username,
		"available": !exists,
	}, ""))
}

func (h *AdminHandler) CheckEmail(c *fiber.Ctx) error {
	email := c.Query("email")
	excludeID := c.Query("exclude_id")

	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Email wajib diisi"))
	}

	var excludeUUID *uuid.UUID
	if excludeID != "" {
		parsed, err := uuid.Parse(excludeID)
		if err == nil {
			excludeUUID = &parsed
		}
	}

	exists, _ := h.userRepo.EmailExists(email, excludeUUID)

	return c.JSON(dto.SuccessResponse(map[string]interface{}{
		"email":     email,
		"available": !exists,
	}, ""))
}

func (h *AdminHandler) DeactivateUser(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	user, err := h.userRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("USER_NOT_FOUND", "User tidak ditemukan"))
	}

	user.IsActive = false
	if err := h.userRepo.Update(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal menonaktifkan user"))
	}

	return c.JSON(dto.SuccessResponse(nil, "User berhasil dinonaktifkan"))
}

func (h *AdminHandler) ActivateUser(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	user, err := h.userRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("USER_NOT_FOUND", "User tidak ditemukan"))
	}

	user.IsActive = true
	if err := h.userRepo.Update(user); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengaktifkan user"))
	}

	return c.JSON(dto.SuccessResponse(nil, "User berhasil diaktifkan"))
}

// Portfolio Moderation Handlers
func (h *AdminHandler) ListPendingPortfolios(c *fiber.Ctx) error {
	search := c.Query("search")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	sort := c.Query("sort", "-created_at")

	var jurusanID *uuid.UUID
	if id := c.Query("jurusan_id"); id != "" {
		parsed, _ := uuid.Parse(id)
		jurusanID = &parsed
	}

	portfolios, total, err := h.adminRepo.ListPendingPortfolios(search, jurusanID, sort, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil data portfolio"))
	}

	var result []dto.AdminPortfolioDTO
	for _, p := range portfolios {
		pDTO := dto.AdminPortfolioDTO{
			ID: p.ID, Judul: p.Judul, Slug: p.Slug, ThumbnailURL: p.ThumbnailURL, Status: string(p.Status), CreatedAt: p.CreatedAt,
		}
		if p.User != nil {
			var kelasNama, jurusanNama *string
			if p.User.Kelas != nil {
				kelasNama = &p.User.Kelas.Nama
				if p.User.Kelas.Jurusan != nil {
					jurusanNama = &p.User.Kelas.Jurusan.Nama
				}
			}
			pDTO.User = &dto.PortfolioUserDTO{
				ID: p.User.ID, Username: p.User.Username, Nama: p.User.Nama, AvatarURL: p.User.AvatarURL,
				Role: string(p.User.Role), KelasNama: kelasNama, JurusanNama: jurusanNama,
			}
		}
		result = append(result, pDTO)
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return c.JSON(dto.SuccessWithMeta(result, &dto.Meta{CurrentPage: page, PerPage: limit, TotalPages: totalPages, TotalCount: total}))
}

func (h *AdminHandler) ListAllPortfolios(c *fiber.Ctx) error {
	search := c.Query("search")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	var status *string
	var userID, jurusanID *uuid.UUID

	if s := c.Query("status"); s != "" {
		status = &s
	}
	if id := c.Query("user_id"); id != "" {
		parsed, _ := uuid.Parse(id)
		userID = &parsed
	}
	if id := c.Query("jurusan_id"); id != "" {
		parsed, _ := uuid.Parse(id)
		jurusanID = &parsed
	}

	portfolios, total, err := h.adminRepo.ListPortfolios(search, status, userID, jurusanID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil data portfolio"))
	}

	var result []dto.AdminPortfolioDTO
	for _, p := range portfolios {
		pDTO := dto.AdminPortfolioDTO{
			ID: p.ID, Judul: p.Judul, Slug: p.Slug, ThumbnailURL: p.ThumbnailURL, Status: string(p.Status), CreatedAt: p.CreatedAt,
		}
		if p.User != nil {
			pDTO.User = &dto.PortfolioUserDTO{
				ID: p.User.ID, Username: p.User.Username, Nama: p.User.Nama, AvatarURL: p.User.AvatarURL, Role: string(p.User.Role),
			}
		}
		result = append(result, pDTO)
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return c.JSON(dto.SuccessWithMeta(result, &dto.Meta{CurrentPage: page, PerPage: limit, TotalPages: totalPages, TotalCount: total}))
}

func (h *AdminHandler) GetPortfolio(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	portfolio, err := h.portfolioRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("PORTFOLIO_NOT_FOUND", "Portfolio tidak ditemukan"))
	}

	likeCount, _ := h.portfolioRepo.GetLikeCount(portfolio.ID)

	result := dto.PortfolioDetailDTO{
		ID: portfolio.ID, Judul: portfolio.Judul, Slug: portfolio.Slug, ThumbnailURL: portfolio.ThumbnailURL,
		Status: string(portfolio.Status), AdminReviewNote: portfolio.AdminReviewNote, ReviewedAt: portfolio.ReviewedAt,
		PublishedAt: portfolio.PublishedAt, CreatedAt: portfolio.CreatedAt, UpdatedAt: portfolio.UpdatedAt, LikeCount: likeCount,
	}

	if portfolio.User != nil {
		var kelasNama, jurusanNama *string
		if portfolio.User.Kelas != nil {
			kelasNama = &portfolio.User.Kelas.Nama
			if portfolio.User.Kelas.Jurusan != nil {
				jurusanNama = &portfolio.User.Kelas.Jurusan.Nama
			}
		}
		result.User = &dto.PortfolioUserDTO{
			ID: portfolio.User.ID, Username: portfolio.User.Username, Nama: portfolio.User.Nama,
			AvatarURL: portfolio.User.AvatarURL, Role: string(portfolio.User.Role), KelasNama: kelasNama, JurusanNama: jurusanNama,
		}
	}

	for _, t := range portfolio.Tags {
		result.Tags = append(result.Tags, dto.TagDTO{ID: t.ID, Nama: t.Nama})
	}
	for _, b := range portfolio.ContentBlocks {
		result.ContentBlocks = append(result.ContentBlocks, dto.ContentBlockDTO{
			ID: b.ID, BlockType: string(b.BlockType), BlockOrder: b.BlockOrder, Payload: b.Payload,
		})
	}

	return c.JSON(dto.SuccessResponse(result, ""))
}

func (h *AdminHandler) ApprovePortfolio(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	portfolio, err := h.portfolioRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("PORTFOLIO_NOT_FOUND", "Portfolio tidak ditemukan"))
	}

	var req dto.ModeratePortfolioRequest
	c.BodyParser(&req)

	adminID := middleware.GetUserID(c)
	now := time.Now()

	portfolio.Status = domain.StatusPublished
	portfolio.AdminReviewNote = &req.Note
	portfolio.ReviewedBy = adminID
	portfolio.ReviewedAt = &now
	portfolio.PublishedAt = &now

	if err := h.portfolioRepo.Update(portfolio); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal menyetujui portfolio"))
	}

	// Send notification to portfolio owner
	if h.notifService != nil {
		_ = h.notifService.NotifyPortfolioApproved(portfolio)
	}

	return c.JSON(dto.SuccessResponse(map[string]interface{}{
		"id": portfolio.ID, "status": portfolio.Status, "admin_review_note": portfolio.AdminReviewNote,
		"reviewed_at": portfolio.ReviewedAt, "published_at": portfolio.PublishedAt,
	}, "Portfolio berhasil disetujui dan dipublish"))
}

func (h *AdminHandler) RejectPortfolio(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	portfolio, err := h.portfolioRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("PORTFOLIO_NOT_FOUND", "Portfolio tidak ditemukan"))
	}

	var req dto.ModeratePortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	if req.Note == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Validasi gagal",
			dto.ErrorDetail{Field: "note", Message: "Alasan penolakan wajib diisi"},
		))
	}

	adminID := middleware.GetUserID(c)
	now := time.Now()

	portfolio.Status = domain.StatusRejected
	portfolio.AdminReviewNote = &req.Note
	portfolio.ReviewedBy = adminID
	portfolio.ReviewedAt = &now

	if err := h.portfolioRepo.Update(portfolio); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal menolak portfolio"))
	}

	// Send notification to portfolio owner
	if h.notifService != nil {
		_ = h.notifService.NotifyPortfolioRejected(portfolio, req.Note)
	}

	return c.JSON(dto.SuccessResponse(map[string]interface{}{
		"id": portfolio.ID, "status": portfolio.Status, "admin_review_note": portfolio.AdminReviewNote, "reviewed_at": portfolio.ReviewedAt,
	}, "Portfolio ditolak"))
}

func (h *AdminHandler) UpdatePortfolio(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	portfolio, err := h.portfolioRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("PORTFOLIO_NOT_FOUND", "Portfolio tidak ditemukan"))
	}

	var req dto.AdminUpdatePortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	if req.Judul != nil {
		portfolio.Judul = *req.Judul
	}
	if req.Status != nil {
		portfolio.Status = domain.PortfolioStatus(*req.Status)
		if portfolio.Status == domain.StatusPublished && portfolio.PublishedAt == nil {
			now := time.Now()
			portfolio.PublishedAt = &now
		}
	}

	if err := h.portfolioRepo.Update(portfolio); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal memperbarui portfolio"))
	}

	if req.TagIDs != nil {
		h.portfolioRepo.UpdateTags(portfolio.ID, req.TagIDs)
	}

	return c.JSON(dto.SuccessResponse(map[string]interface{}{
		"id": portfolio.ID, "judul": portfolio.Judul, "status": portfolio.Status, "updated_at": portfolio.UpdatedAt,
	}, "Portfolio berhasil diperbarui"))
}

func (h *AdminHandler) DeletePortfolio(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	if err := h.portfolioRepo.Delete(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal menghapus portfolio"))
	}

	return c.JSON(dto.SuccessResponse(nil, "Portfolio berhasil dihapus"))
}

// Dashboard Stats
func (h *AdminHandler) GetDashboardStats(c *fiber.Ctx) error {
	userTotal, students, alumni, admins, userNewMonth, _ := h.adminRepo.GetUserStats()
	portfolioTotal, published, pending, draft, rejected, archived, portfolioNewMonth, _ := h.adminRepo.GetPortfolioStats()
	jurusanCount, _ := h.adminRepo.GetJurusanCount()
	kelasTotal, kelasActive, _ := h.adminRepo.GetKelasStats()

	// Get recent users (5)
	recentUsers, _ := h.adminRepo.GetRecentUsers(5)
	var recentUsersDTO []dto.RecentUserDTO
	for _, u := range recentUsers {
		uDTO := dto.RecentUserDTO{
			ID:        u.ID,
			Username:  u.Username,
			Nama:      u.Nama,
			AvatarURL: u.AvatarURL,
			Role:      string(u.Role),
			CreatedAt: u.CreatedAt,
		}
		if u.Kelas != nil {
			uDTO.KelasNama = &u.Kelas.Nama
		}
		recentUsersDTO = append(recentUsersDTO, uDTO)
	}

	// Get recent pending portfolios (5)
	recentPending, _ := h.adminRepo.GetRecentPendingPortfolios(5)
	var recentPendingDTO []dto.RecentPendingPortfolioDTO
	for _, p := range recentPending {
		pDTO := dto.RecentPendingPortfolioDTO{
			ID:           p.ID,
			Judul:        p.Judul,
			Slug:         p.Slug,
			ThumbnailURL: p.ThumbnailURL,
			CreatedAt:    p.CreatedAt,
		}
		if p.User != nil {
			pDTO.UserNama = p.User.Nama
			pDTO.UserUsername = p.User.Username
			pDTO.UserAvatarURL = p.User.AvatarURL
		}
		recentPendingDTO = append(recentPendingDTO, pDTO)
	}

	return c.JSON(dto.SuccessResponse(dto.DashboardStatsDTO{
		Users: dto.UserStatsDTO{
			Total: userTotal, Students: students, Alumni: alumni, Admins: admins, NewThisMonth: userNewMonth,
		},
		Portfolios: dto.PortfolioStatsDTO{
			Total: portfolioTotal, Published: published, PendingReview: pending, Draft: draft,
			Rejected: rejected, Archived: archived, NewThisMonth: portfolioNewMonth,
		},
		Jurusan:                 dto.CountDTO{Total: jurusanCount},
		Kelas:                   dto.KelasStatsDTO{Total: kelasTotal, ActiveTahunAjaran: kelasActive},
		RecentUsers:             recentUsersDTO,
		RecentPendingPortfolios: recentPendingDTO,
	}, ""))
}

// ============================================================================
// SPECIAL ROLE HANDLERS
// ============================================================================

// ListSpecialRoles returns all special roles
func (h *AdminHandler) ListSpecialRoles(c *fiber.Ctx) error {
	search := c.Query("search")
	includeInactive := c.Query("include_inactive") == "true"

	roles, err := h.adminRepo.ListSpecialRoles(search, includeInactive)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil data special roles"))
	}

	var result []dto.SpecialRoleDTO
	for _, r := range roles {
		userCount, _ := h.adminRepo.GetSpecialRoleUserCount(r.ID)
		caps := []string(r.Capabilities)
		if caps == nil {
			caps = []string{}
		}
		result = append(result, dto.SpecialRoleDTO{
			ID:           r.ID,
			Nama:         r.Nama,
			Description:  r.Description,
			Color:        r.Color,
			Capabilities: caps,
			IsActive:     r.IsActive,
			UserCount:    int(userCount),
			CreatedAt:    r.CreatedAt,
		})
	}

	return c.JSON(dto.SuccessResponse(result, ""))
}

// CreateSpecialRole creates a new special role
func (h *AdminHandler) CreateSpecialRole(c *fiber.Ctx) error {
	var req dto.CreateSpecialRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	if req.Nama == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Validasi gagal",
			dto.ErrorDetail{Field: "nama", Message: "Nama role wajib diisi"},
		))
	}

	if len(req.Capabilities) == 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Validasi gagal",
			dto.ErrorDetail{Field: "capabilities", Message: "Minimal satu capability harus dipilih"},
		))
	}

	// Validate capabilities
	for _, cap := range req.Capabilities {
		if _, ok := dto.ValidCapabilities[cap]; !ok {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Validasi gagal",
				dto.ErrorDetail{Field: "capabilities", Message: "Capability tidak valid: " + cap},
			))
		}
	}

	// Check duplicate name
	exists, _ := h.adminRepo.SpecialRoleNameExists(req.Nama, nil)
	if exists {
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse("DUPLICATE_ERROR", "Special role dengan nama tersebut sudah ada"))
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	color := "#6366f1"
	if req.Color != "" {
		color = req.Color
	}

	role := &domain.SpecialRole{
		Nama:         req.Nama,
		Description:  req.Description,
		Color:        color,
		Capabilities: domain.StringArray(req.Capabilities),
		IsActive:     isActive,
	}

	if err := h.adminRepo.CreateSpecialRole(role); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal membuat special role"))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse(dto.SpecialRoleDTO{
		ID:           role.ID,
		Nama:         role.Nama,
		Description:  role.Description,
		Color:        role.Color,
		Capabilities: []string(role.Capabilities),
		IsActive:     role.IsActive,
		CreatedAt:    role.CreatedAt,
	}, "Special role berhasil dibuat"))
}

// GetSpecialRole returns a special role with its users
func (h *AdminHandler) GetSpecialRole(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	role, err := h.adminRepo.FindSpecialRoleByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Special role tidak ditemukan"))
	}

	userRoles, _ := h.adminRepo.GetSpecialRoleUsers(role.ID)
	var users []dto.SpecialRoleUserDTO
	for _, ur := range userRoles {
		if ur.User != nil {
			var kelasNama *string
			if ur.User.Kelas != nil {
				kelasNama = &ur.User.Kelas.Nama
			}
			users = append(users, dto.SpecialRoleUserDTO{
				ID:         ur.User.ID,
				Username:   ur.User.Username,
				Nama:       ur.User.Nama,
				AvatarURL:  ur.User.AvatarURL,
				KelasNama:  kelasNama,
				AssignedAt: ur.AssignedAt,
				AssignedBy: ur.AssignedBy,
			})
		}
	}

	caps := []string(role.Capabilities)
	if caps == nil {
		caps = []string{}
	}

	return c.JSON(dto.SuccessResponse(dto.SpecialRoleDetailDTO{
		SpecialRoleDTO: dto.SpecialRoleDTO{
			ID:           role.ID,
			Nama:         role.Nama,
			Description:  role.Description,
			Color:        role.Color,
			Capabilities: caps,
			IsActive:     role.IsActive,
			UserCount:    len(users),
			CreatedAt:    role.CreatedAt,
		},
		Users: users,
	}, ""))
}

// UpdateSpecialRole updates a special role
func (h *AdminHandler) UpdateSpecialRole(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	role, err := h.adminRepo.FindSpecialRoleByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Special role tidak ditemukan"))
	}

	var req dto.UpdateSpecialRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	if req.Nama != nil {
		exists, _ := h.adminRepo.SpecialRoleNameExists(*req.Nama, &id)
		if exists {
			return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse("DUPLICATE_ERROR", "Special role dengan nama tersebut sudah ada"))
		}
		role.Nama = *req.Nama
	}

	if req.Description != nil {
		role.Description = req.Description
	}

	if req.Color != nil {
		role.Color = *req.Color
	}

	if req.Capabilities != nil {
		// Validate capabilities
		for _, cap := range req.Capabilities {
			if _, ok := dto.ValidCapabilities[cap]; !ok {
				return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Validasi gagal",
					dto.ErrorDetail{Field: "capabilities", Message: "Capability tidak valid: " + cap},
				))
			}
		}
		role.Capabilities = domain.StringArray(req.Capabilities)
	}

	if req.IsActive != nil {
		role.IsActive = *req.IsActive
	}

	if err := h.adminRepo.UpdateSpecialRole(role); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal memperbarui special role"))
	}

	return c.JSON(dto.SuccessResponse(dto.SpecialRoleDTO{
		ID:           role.ID,
		Nama:         role.Nama,
		Description:  role.Description,
		Color:        role.Color,
		Capabilities: []string(role.Capabilities),
		IsActive:     role.IsActive,
		CreatedAt:    role.CreatedAt,
	}, "Special role berhasil diperbarui"))
}

// DeleteSpecialRole deletes a special role
func (h *AdminHandler) DeleteSpecialRole(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	if err := h.adminRepo.DeleteSpecialRole(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal menghapus special role"))
	}

	return c.JSON(dto.SuccessResponse(nil, "Special role berhasil dihapus"))
}

// AssignUsersToRole assigns users to a special role
func (h *AdminHandler) AssignUsersToRole(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	_, err = h.adminRepo.FindSpecialRoleByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Special role tidak ditemukan"))
	}

	var req dto.AssignUsersRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	if len(req.UserIDs) == 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Validasi gagal",
			dto.ErrorDetail{Field: "user_ids", Message: "Minimal satu user harus dipilih"},
		))
	}

	adminID := middleware.GetUserID(c)
	if err := h.adminRepo.AssignUsersToRole(id, req.UserIDs, *adminID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal assign users ke role"))
	}

	return c.JSON(dto.SuccessResponse(nil, "Users berhasil di-assign ke role"))
}

// RemoveUserFromRole removes a user from a special role
func (h *AdminHandler) RemoveUserFromRole(c *fiber.Ctx) error {
	roleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Role ID tidak valid"))
	}

	userID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "User ID tidak valid"))
	}

	if err := h.adminRepo.RemoveUserFromRole(roleID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal menghapus user dari role"))
	}

	return c.JSON(dto.SuccessResponse(nil, "User berhasil dihapus dari role"))
}

// GetUserSpecialRoles returns special roles for a user
func (h *AdminHandler) GetUserSpecialRoles(c *fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	roles, err := h.adminRepo.GetUserSpecialRoles(userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil special roles user"))
	}

	var result []dto.SpecialRoleDTO
	for _, r := range roles {
		result = append(result, dto.SpecialRoleDTO{
			ID:           r.ID,
			Nama:         r.Nama,
			Description:  r.Description,
			Color:        r.Color,
			Capabilities: []string(r.Capabilities),
			IsActive:     r.IsActive,
			CreatedAt:    r.CreatedAt,
		})
	}

	return c.JSON(dto.SuccessResponse(result, ""))
}

// UpdateUserSpecialRoles updates special roles for a user
func (h *AdminHandler) UpdateUserSpecialRoles(c *fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	var req dto.UserSpecialRolesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Request body tidak valid"))
	}

	adminID := middleware.GetUserID(c)
	if err := h.adminRepo.UpdateUserSpecialRoles(userID, req.SpecialRoleIDs, *adminID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal memperbarui special roles user"))
	}

	return c.JSON(dto.SuccessResponse(nil, "Special roles user berhasil diperbarui"))
}

// GetCapabilities returns list of available capabilities
func (h *AdminHandler) GetCapabilities(c *fiber.Ctx) error {
	return c.JSON(dto.SuccessResponse(dto.GetCapabilitiesList(), ""))
}

// GetActiveSpecialRoles returns only active special roles (for assignment UI)
func (h *AdminHandler) GetActiveSpecialRoles(c *fiber.Ctx) error {
	roles, err := h.adminRepo.GetActiveSpecialRoles()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil data special roles"))
	}

	var result []dto.SpecialRoleDTO
	for _, r := range roles {
		caps := []string(r.Capabilities)
		if caps == nil {
			caps = []string{}
		}
		result = append(result, dto.SpecialRoleDTO{
			ID:           r.ID,
			Nama:         r.Nama,
			Description:  r.Description,
			Color:        r.Color,
			Capabilities: caps,
			IsActive:     r.IsActive,
			CreatedAt:    r.CreatedAt,
		})
	}

	return c.JSON(dto.SuccessResponse(result, ""))
}

// ============================================================================
// SERIES EXPORT HANDLERS
// ============================================================================

// GetSeriesExportPreview returns preview info before export
func (h *AdminHandler) GetSeriesExportPreview(c *fiber.Ctx) error {
	seriesID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	// Check series exists
	_, err = h.adminRepo.FindSeriesByID(seriesID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("SERIES_NOT_FOUND", "Series tidak ditemukan"))
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

	portfolioCount, userCount, err := h.adminRepo.CountPortfoliosForExport(seriesID, jurusanID, kelasID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal menghitung data export"))
	}

	// Estimate pages (roughly 2 pages per portfolio)
	estimatedPages := portfolioCount * 2

	return c.JSON(dto.SuccessResponse(dto.ExportPreviewResponse{
		PortfolioCount: portfolioCount,
		UserCount:      userCount,
		EstimatedPages: estimatedPages,
	}, ""))
}

// GetSeriesExportData returns full data for PDF export
func (h *AdminHandler) GetSeriesExportData(c *fiber.Ctx) error {
	seriesID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	// Get series with blocks
	series, err := h.adminRepo.FindSeriesByID(seriesID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("SERIES_NOT_FOUND", "Series tidak ditemukan"))
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

	// Get portfolios
	portfolios, err := h.adminRepo.GetPortfoliosForExport(seriesID, jurusanID, kelasID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil data portfolio"))
	}

	// Build response
	var portfolioDTOs []dto.PortfolioExportDTO
	for _, p := range portfolios {
		userDTO := dto.PortfolioExportUserDTO{
			ID:       p.User.ID,
			Nama:     p.User.Nama,
			Username: p.User.Username,
		}
		if p.User.AvatarURL != nil {
			userDTO.AvatarURL = p.User.AvatarURL
		}
		if p.User.NISN != nil {
			userDTO.NISN = p.User.NISN
		}
		if p.User.NIS != nil {
			userDTO.NIS = p.User.NIS
		}
		if p.User.Kelas != nil {
			userDTO.KelasNama = &p.User.Kelas.Nama
			if p.User.Kelas.Jurusan != nil {
				userDTO.JurusanNama = &p.User.Kelas.Jurusan.Nama
			}
		}

		var blocks []dto.ContentBlockDTO
		for _, b := range p.ContentBlocks {
			blocks = append(blocks, dto.ContentBlockDTO{
				ID:         b.ID,
				BlockType:  string(b.BlockType),
				BlockOrder: b.BlockOrder,
				Payload:    b.Payload,
			})
		}

		portfolioDTOs = append(portfolioDTOs, dto.PortfolioExportDTO{
			ID:            p.ID,
			Judul:         p.Judul,
			ThumbnailURL:  p.ThumbnailURL,
			CreatedAt:     p.CreatedAt,
			ContentBlocks: blocks,
			User:          userDTO,
		})
	}

	portfolioCount, userCount, _ := h.adminRepo.CountPortfoliosForExport(seriesID, jurusanID, kelasID)

	return c.JSON(dto.SuccessResponse(dto.SeriesExportResponse{
		Series: dto.SeriesDetailDTO{
			ID:        series.ID,
			Nama:      series.Nama,
			Deskripsi: series.Deskripsi,
			IsActive:  series.IsActive,
			Blocks:    dto.SeriesBlocksToDTOs(series.Blocks),
			CreatedAt: series.CreatedAt,
		},
		Portfolios: portfolioDTOs,
		Meta: dto.ExportMeta{
			TotalCount: portfolioCount,
			UserCount:  userCount,
			ExportedAt: time.Now(),
		},
	}, ""))
}
