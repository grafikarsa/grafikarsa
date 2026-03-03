package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/middleware"
	"github.com/grafikarsa/backend/internal/repository"
	"github.com/grafikarsa/backend/internal/service"
)

type ChangelogHandler struct {
	changelogRepo       *repository.ChangelogRepository
	notificationService *service.NotificationService
	userRepo            *repository.UserRepository
}

func NewChangelogHandler(
	changelogRepo *repository.ChangelogRepository,
	notificationService *service.NotificationService,
	userRepo *repository.UserRepository,
) *ChangelogHandler {
	return &ChangelogHandler{
		changelogRepo:       changelogRepo,
		notificationService: notificationService,
		userRepo:            userRepo,
	}
}

// ============================================================================
// PUBLIC ENDPOINTS
// ============================================================================

// List - GET /changelogs (public, paginated)
func (h *ChangelogHandler) List(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	changelogs, total, err := h.changelogRepo.ListPublished(page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"FETCH_FAILED", "Gagal mengambil data changelog",
		))
	}

	var responses []dto.ChangelogResponse
	for _, cl := range changelogs {
		responses = append(responses, h.toFullResponse(&cl))
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return c.JSON(dto.PaginatedResponse(
		responses,
		dto.PaginationMeta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: int(totalPages),
		},
	))
}

// GetByID - GET /changelogs/:id (public)
func (h *ChangelogHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "ID tidak valid"))
	}

	changelog, err := h.changelogRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Changelog tidak ditemukan"))
	}

	// Only show published changelogs to public
	if !changelog.IsPublished {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Changelog tidak ditemukan"))
	}

	return c.JSON(dto.SuccessResponse(h.toFullResponse(changelog), ""))
}

// GetLatest - GET /changelogs/latest (public)
func (h *ChangelogHandler) GetLatest(c *fiber.Ctx) error {
	changelog, err := h.changelogRepo.GetLatest()
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Belum ada changelog"))
	}

	return c.JSON(dto.SuccessResponse(h.toFullResponse(changelog), ""))
}

// GetUnreadCount - GET /changelogs/unread-count (auth required)
func (h *ChangelogHandler) GetUnreadCount(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse("UNAUTHORIZED", "Login diperlukan"))
	}

	count, err := h.changelogRepo.GetUnreadCount(*userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"FETCH_FAILED", "Gagal mengambil jumlah unread",
		))
	}

	return c.JSON(dto.SuccessResponse(dto.ChangelogUnreadCountResponse{Count: count}, ""))
}

// MarkAsRead - POST /changelogs/:id/mark-read (auth required)
func (h *ChangelogHandler) MarkAsRead(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse("UNAUTHORIZED", "Login diperlukan"))
	}

	changelogID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "ID tidak valid"))
	}

	if err := h.changelogRepo.MarkAsRead(*userID, changelogID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"MARK_READ_FAILED", "Gagal menandai sebagai dibaca",
		))
	}

	return c.JSON(dto.SuccessResponse(nil, "Berhasil ditandai sebagai dibaca"))
}

// MarkAllAsRead - POST /changelogs/mark-all-read (auth required)
func (h *ChangelogHandler) MarkAllAsRead(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse("UNAUTHORIZED", "Login diperlukan"))
	}

	if err := h.changelogRepo.MarkAllAsRead(*userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"MARK_READ_FAILED", "Gagal menandai semua sebagai dibaca",
		))
	}

	return c.JSON(dto.SuccessResponse(nil, "Semua changelog ditandai sebagai dibaca"))
}

// ============================================================================
// ADMIN ENDPOINTS
// ============================================================================

// AdminList - GET /admin/changelogs (admin)
func (h *ChangelogHandler) AdminList(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	changelogs, total, err := h.changelogRepo.ListAll(page, limit, search)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"FETCH_FAILED", "Gagal mengambil data changelog",
		))
	}

	var responses []dto.ChangelogListResponse
	for _, cl := range changelogs {
		responses = append(responses, h.toListResponse(&cl))
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return c.JSON(dto.PaginatedResponse(
		responses,
		dto.PaginationMeta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: int(totalPages),
		},
	))
}

// AdminGetByID - GET /admin/changelogs/:id (admin)
func (h *ChangelogHandler) AdminGetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "ID tidak valid"))
	}

	changelog, err := h.changelogRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Changelog tidak ditemukan"))
	}

	return c.JSON(dto.SuccessResponse(h.toFullResponse(changelog), ""))
}

// Create - POST /admin/changelogs (admin)
func (h *ChangelogHandler) Create(c *fiber.Ctx) error {
	var req dto.CreateChangelogRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"INVALID_REQUEST", "Format request tidak valid",
		))
	}

	// Validation
	if req.Version == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Version wajib diisi"))
	}
	if req.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Title wajib diisi"))
	}
	if len(req.Sections) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Minimal 1 section wajib diisi"))
	}
	if len(req.Contributors) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Minimal 1 contributor wajib diisi"))
	}

	adminID := middleware.GetUserID(c)

	// Parse release date
	releaseDate := time.Now()
	if req.ReleaseDate != nil && *req.ReleaseDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.ReleaseDate)
		if err == nil {
			releaseDate = parsed
		}
	}

	// Create changelog
	changelog := &domain.Changelog{
		Version:     req.Version,
		Title:       req.Title,
		Description: req.Description,
		ReleaseDate: releaseDate,
		IsPublished: false,
		CreatedBy:   *adminID,
	}

	if err := h.changelogRepo.Create(changelog); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"CREATE_FAILED", "Gagal membuat changelog",
		))
	}

	// Create sections and blocks
	for i, sectionReq := range req.Sections {
		section := &domain.ChangelogSection{
			ChangelogID:  changelog.ID,
			Category:     domain.ChangelogCategory(sectionReq.Category),
			SectionOrder: i,
		}
		if err := h.changelogRepo.CreateSection(section); err != nil {
			continue
		}

		for j, blockReq := range sectionReq.Blocks {
			block := &domain.ChangelogSectionBlock{
				SectionID:  section.ID,
				BlockType:  domain.ContentBlockType(blockReq.BlockType),
				BlockOrder: j,
				Payload:    domain.JSONB(blockReq.Payload),
			}
			h.changelogRepo.CreateSectionBlock(block)
		}
	}

	// Create contributors
	for i, contribReq := range req.Contributors {
		userID, err := uuid.Parse(contribReq.UserID)
		if err != nil {
			continue
		}
		contributor := &domain.ChangelogContributor{
			ChangelogID:      changelog.ID,
			UserID:           userID,
			Contribution:     contribReq.Contribution,
			ContributorOrder: i,
		}
		h.changelogRepo.CreateContributor(contributor)
	}

	// Reload to get full data
	changelog, _ = h.changelogRepo.FindByID(changelog.ID)

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse(
		h.toFullResponse(changelog), "Changelog berhasil dibuat",
	))
}

// Update - PATCH /admin/changelogs/:id (admin)
func (h *ChangelogHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "ID tidak valid"))
	}

	changelog, err := h.changelogRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Changelog tidak ditemukan"))
	}

	var req dto.UpdateChangelogRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"INVALID_REQUEST", "Format request tidak valid",
		))
	}

	// Update basic fields
	if req.Version != nil {
		changelog.Version = *req.Version
	}
	if req.Title != nil {
		changelog.Title = *req.Title
	}
	if req.Description != nil {
		changelog.Description = req.Description
	}
	if req.ReleaseDate != nil && *req.ReleaseDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.ReleaseDate)
		if err == nil {
			changelog.ReleaseDate = parsed
		}
	}

	if err := h.changelogRepo.Update(changelog); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"UPDATE_FAILED", "Gagal mengupdate changelog",
		))
	}

	// Update sections if provided
	if req.Sections != nil {
		// Delete existing sections
		h.changelogRepo.DeleteSections(changelog.ID)

		// Create new sections
		for i, sectionReq := range req.Sections {
			section := &domain.ChangelogSection{
				ChangelogID:  changelog.ID,
				Category:     domain.ChangelogCategory(sectionReq.Category),
				SectionOrder: i,
			}
			if err := h.changelogRepo.CreateSection(section); err != nil {
				continue
			}

			for j, blockReq := range sectionReq.Blocks {
				block := &domain.ChangelogSectionBlock{
					SectionID:  section.ID,
					BlockType:  domain.ContentBlockType(blockReq.BlockType),
					BlockOrder: j,
					Payload:    domain.JSONB(blockReq.Payload),
				}
				h.changelogRepo.CreateSectionBlock(block)
			}
		}
	}

	// Update contributors if provided
	if req.Contributors != nil {
		// Delete existing contributors
		h.changelogRepo.DeleteContributors(changelog.ID)

		// Create new contributors
		for i, contribReq := range req.Contributors {
			userID, err := uuid.Parse(contribReq.UserID)
			if err != nil {
				continue
			}
			contributor := &domain.ChangelogContributor{
				ChangelogID:      changelog.ID,
				UserID:           userID,
				Contribution:     contribReq.Contribution,
				ContributorOrder: i,
			}
			h.changelogRepo.CreateContributor(contributor)
		}
	}

	// Reload to get full data
	changelog, _ = h.changelogRepo.FindByID(changelog.ID)

	return c.JSON(dto.SuccessResponse(h.toFullResponse(changelog), "Changelog berhasil diupdate"))
}

// Delete - DELETE /admin/changelogs/:id (admin)
func (h *ChangelogHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "ID tidak valid"))
	}

	if _, err := h.changelogRepo.FindByID(id); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Changelog tidak ditemukan"))
	}

	if err := h.changelogRepo.Delete(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"DELETE_FAILED", "Gagal menghapus changelog",
		))
	}

	return c.JSON(dto.SuccessResponse(nil, "Changelog berhasil dihapus"))
}

// Publish - POST /admin/changelogs/:id/publish (admin)
func (h *ChangelogHandler) Publish(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "ID tidak valid"))
	}

	changelog, err := h.changelogRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Changelog tidak ditemukan"))
	}

	if changelog.IsPublished {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("ALREADY_PUBLISHED", "Changelog sudah dipublish"))
	}

	if err := h.changelogRepo.Publish(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"PUBLISH_FAILED", "Gagal mempublish changelog",
		))
	}

	// Send notification to all users (async in background)
	go h.sendChangelogNotification(changelog)

	return c.JSON(dto.SuccessResponse(nil, "Changelog berhasil dipublish"))
}

// Unpublish - POST /admin/changelogs/:id/unpublish (admin)
func (h *ChangelogHandler) Unpublish(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "ID tidak valid"))
	}

	changelog, err := h.changelogRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Changelog tidak ditemukan"))
	}

	if !changelog.IsPublished {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("NOT_PUBLISHED", "Changelog belum dipublish"))
	}

	if err := h.changelogRepo.Unpublish(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"UNPUBLISH_FAILED", "Gagal meng-unpublish changelog",
		))
	}

	return c.JSON(dto.SuccessResponse(nil, "Changelog berhasil di-unpublish"))
}

// ============================================================================
// HELPER METHODS
// ============================================================================

func (h *ChangelogHandler) sendChangelogNotification(changelog *domain.Changelog) {
	// This would send notifications to all active users
	// For now, we'll skip the actual implementation as it requires
	// fetching all users and creating notifications in batch
	// TODO: Implement batch notification sending
}

func (h *ChangelogHandler) toFullResponse(cl *domain.Changelog) dto.ChangelogResponse {
	resp := dto.ChangelogResponse{
		ID:           cl.ID.String(),
		Version:      cl.Version,
		Title:        cl.Title,
		Description:  cl.Description,
		ReleaseDate:  cl.ReleaseDate.Format("2006-01-02"),
		IsPublished:  cl.IsPublished,
		Sections:     make([]dto.ChangelogSectionResponse, 0),
		Contributors: make([]dto.ChangelogContributorResponse, 0),
		CreatedAt:    cl.CreatedAt,
		UpdatedAt:    cl.UpdatedAt,
	}

	if cl.Creator != nil {
		resp.CreatedBy = &dto.UserBriefResponse{
			ID:        cl.Creator.ID.String(),
			Username:  cl.Creator.Username,
			Nama:      cl.Creator.Nama,
			AvatarURL: cl.Creator.AvatarURL,
		}
	}

	for _, section := range cl.Sections {
		sectionResp := dto.ChangelogSectionResponse{
			ID:       section.ID.String(),
			Category: dto.ChangelogCategory(section.Category),
			Blocks:   make([]dto.ChangelogBlockResponse, 0),
		}

		for _, block := range section.Blocks {
			sectionResp.Blocks = append(sectionResp.Blocks, dto.ChangelogBlockResponse{
				ID:        block.ID.String(),
				BlockType: string(block.BlockType),
				Payload:   map[string]interface{}(block.Payload),
			})
		}

		resp.Sections = append(resp.Sections, sectionResp)
	}

	for _, contrib := range cl.Contributors {
		contribResp := dto.ChangelogContributorResponse{
			ID:           contrib.ID.String(),
			Contribution: contrib.Contribution,
		}

		if contrib.User != nil {
			contribResp.User = &dto.UserBriefResponse{
				ID:        contrib.User.ID.String(),
				Username:  contrib.User.Username,
				Nama:      contrib.User.Nama,
				AvatarURL: contrib.User.AvatarURL,
			}
		}

		resp.Contributors = append(resp.Contributors, contribResp)
	}

	return resp
}

func (h *ChangelogHandler) toListResponse(cl *domain.Changelog) dto.ChangelogListResponse {
	resp := dto.ChangelogListResponse{
		ID:          cl.ID.String(),
		Version:     cl.Version,
		Title:       cl.Title,
		Description: cl.Description,
		ReleaseDate: cl.ReleaseDate.Format("2006-01-02"),
		IsPublished: cl.IsPublished,
		Categories:  make([]string, 0),
		CreatedAt:   cl.CreatedAt,
	}

	// Extract unique categories
	categoryMap := make(map[string]bool)
	for _, section := range cl.Sections {
		categoryMap[string(section.Category)] = true
	}
	for cat := range categoryMap {
		resp.Categories = append(resp.Categories, cat)
	}

	return resp
}
