package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/middleware"
	"github.com/grafikarsa/backend/internal/repository"
)

type SearchHandler struct {
	userRepo      *repository.UserRepository
	portfolioRepo *repository.PortfolioRepository
}

func NewSearchHandler(userRepo *repository.UserRepository, portfolioRepo *repository.PortfolioRepository) *SearchHandler {
	return &SearchHandler{
		userRepo:      userRepo,
		portfolioRepo: portfolioRepo,
	}
}

func (h *SearchHandler) SearchUsers(c *fiber.Ctx) error {
	query := c.Query("q")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	var jurusanID, kelasID *uuid.UUID
	var role *string

	if id := c.Query("jurusan_id"); id != "" {
		parsed, _ := uuid.Parse(id)
		jurusanID = &parsed
	}
	if id := c.Query("kelas_id"); id != "" {
		parsed, _ := uuid.Parse(id)
		kelasID = &parsed
	}
	if r := c.Query("role"); r != "" {
		role = &r
	}

	users, total, err := h.userRepo.List(query, jurusanID, kelasID, role, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mencari users"))
	}

	var result []map[string]interface{}
	for _, u := range users {
		item := map[string]interface{}{
			"id":         u.ID,
			"username":   u.Username,
			"nama":       u.Nama,
			"avatar_url": u.AvatarURL,
			"bio":        u.Bio,
			"role":       u.Role,
		}
		if u.Kelas != nil {
			item["kelas_nama"] = u.Kelas.Nama
			if u.Kelas.Jurusan != nil {
				item["jurusan_nama"] = u.Kelas.Jurusan.Nama
			}
		}
		result = append(result, item)
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

func (h *SearchHandler) SearchPortfolios(c *fiber.Ctx) error {
	query := c.Query("q")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	var tagIDs []uuid.UUID
	if tags := c.Query("tag_ids"); tags != "" {
		for _, id := range strings.Split(tags, ",") {
			if parsed, err := uuid.Parse(id); err == nil {
				tagIDs = append(tagIDs, parsed)
			}
		}
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

	currentUserID := middleware.GetUserID(c)

	portfolios, total, err := h.portfolioRepo.ListPublished(query, tagIDs, jurusanID, kelasID, nil, "-published_at", page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("INTERNAL_ERROR", "Gagal mencari portfolios"))
	}

	var result []dto.PortfolioListDTO
	for _, p := range portfolios {
		likeCount, _ := h.portfolioRepo.GetLikeCount(p.ID)
		isLiked := false
		if currentUserID != nil {
			isLiked, _ = h.portfolioRepo.IsLiked(*currentUserID, p.ID)
		}

		pDTO := dto.PortfolioListDTO{
			ID:           p.ID,
			Judul:        p.Judul,
			Slug:         p.Slug,
			ThumbnailURL: p.ThumbnailURL,
			PublishedAt:  p.PublishedAt,
			LikeCount:    likeCount,
			IsLiked:      isLiked,
		}

		if p.User != nil {
			var kelasNama *string
			if p.User.Kelas != nil {
				kelasNama = &p.User.Kelas.Nama
			}
			pDTO.User = &dto.PortfolioUserDTO{
				ID:        p.User.ID,
				Username:  p.User.Username,
				Nama:      p.User.Nama,
				AvatarURL: p.User.AvatarURL,
				Role:      string(p.User.Role),
				KelasNama: kelasNama,
			}
		}

		for _, t := range p.Tags {
			pDTO.Tags = append(pDTO.Tags, dto.TagDTO{ID: t.ID, Nama: t.Nama})
		}

		result = append(result, pDTO)
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
