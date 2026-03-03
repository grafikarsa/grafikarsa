package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/repository"
)

type PublicHandler struct {
	adminRepo *repository.AdminRepository
	userRepo  *repository.UserRepository
}

func NewPublicHandler(adminRepo *repository.AdminRepository, userRepo *repository.UserRepository) *PublicHandler {
	return &PublicHandler{adminRepo: adminRepo, userRepo: userRepo}
}

func (h *PublicHandler) ListJurusan(c *fiber.Ctx) error {
	jurusan, err := h.adminRepo.ListJurusan()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil data jurusan"))
	}

	var result []dto.JurusanDTO
	for _, j := range jurusan {
		result = append(result, dto.JurusanDTO{ID: j.ID, Nama: j.Nama, Kode: j.Kode})
	}

	return c.JSON(dto.SuccessResponse(result, ""))
}

func (h *PublicHandler) ListKelas(c *fiber.Ctx) error {
	var jurusanID *uuid.UUID
	var tingkat *int

	if id := c.Query("jurusan_id"); id != "" {
		parsed, _ := uuid.Parse(id)
		jurusanID = &parsed
	}
	if t := c.Query("tingkat"); t != "" {
		parsed, _ := strconv.Atoi(t)
		tingkat = &parsed
	}

	kelas, err := h.adminRepo.ListKelasPublic(jurusanID, tingkat)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil data kelas"))
	}

	var result []map[string]interface{}
	for _, k := range kelas {
		item := map[string]interface{}{
			"id":      k.ID,
			"nama":    k.Nama,
			"tingkat": k.Tingkat,
		}
		if k.Jurusan != nil {
			item["jurusan"] = dto.JurusanDTO{ID: k.Jurusan.ID, Nama: k.Jurusan.Nama}
		}
		result = append(result, item)
	}

	return c.JSON(dto.SuccessResponse(result, ""))
}

func (h *PublicHandler) ListSeries(c *fiber.Ctx) error {
	series, err := h.adminRepo.ListActiveSeries()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil data series"))
	}

	var result []dto.SeriesBriefDTO
	for _, s := range series {
		result = append(result, dto.SeriesBriefDTO{
			ID:         s.ID,
			Nama:       s.Nama,
			Deskripsi:  s.Deskripsi,
			BlockCount: len(s.Blocks),
			Blocks:     dto.SeriesBlocksToDTOs(s.Blocks),
		})
	}

	return c.JSON(dto.SuccessResponse(result, ""))
}

func (h *PublicHandler) GetSeries(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "ID tidak valid"))
	}

	series, err := h.adminRepo.FindSeriesByID(id)
	if err != nil || !series.IsActive {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("SERIES_NOT_FOUND", "Series tidak ditemukan"))
	}

	return c.JSON(dto.SuccessResponse(dto.SeriesBriefDTO{
		ID:         series.ID,
		Nama:       series.Nama,
		Deskripsi:  series.Deskripsi,
		BlockCount: len(series.Blocks),
		Blocks:     dto.SeriesBlocksToDTOs(series.Blocks),
	}, ""))
}

func (h *PublicHandler) GetTopStudents(c *fiber.Ctx) error {
	students, err := h.userRepo.GetTopStudents(3)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil data top students"))
	}

	return c.JSON(dto.SuccessResponse(students, ""))
}

func (h *PublicHandler) GetTopProjects(c *fiber.Ctx) error {
	projects, err := h.userRepo.GetTopProjects(3)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mengambil data top projects"))
	}

	return c.JSON(dto.SuccessResponse(projects, ""))
}
