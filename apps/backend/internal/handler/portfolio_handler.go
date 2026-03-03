package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/middleware"
	"github.com/grafikarsa/backend/internal/repository"
	"github.com/grafikarsa/backend/internal/service"
)

type PortfolioHandler struct {
	portfolioRepo *repository.PortfolioRepository
	userRepo      *repository.UserRepository
	viewRepo      *repository.ViewRepository
	interestRepo  *repository.InterestRepository
	notifService  *service.NotificationService
}

func NewPortfolioHandler(portfolioRepo *repository.PortfolioRepository, userRepo *repository.UserRepository, viewRepo *repository.ViewRepository, interestRepo *repository.InterestRepository, notifService *service.NotificationService) *PortfolioHandler {
	return &PortfolioHandler{
		portfolioRepo: portfolioRepo,
		userRepo:      userRepo,
		viewRepo:      viewRepo,
		interestRepo:  interestRepo,
		notifService:  notifService,
	}
}

func (h *PortfolioHandler) List(c *fiber.Ctx) error {
	search := c.Query("search")
	sort := c.Query("sort", "-published_at")
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

	var jurusanID, kelasID, userID *uuid.UUID
	if id := c.Query("jurusan_id"); id != "" {
		parsed, _ := uuid.Parse(id)
		jurusanID = &parsed
	}
	if id := c.Query("kelas_id"); id != "" {
		parsed, _ := uuid.Parse(id)
		kelasID = &parsed
	}
	if id := c.Query("user_id"); id != "" {
		parsed, _ := uuid.Parse(id)
		userID = &parsed
	}

	portfolios, total, err := h.portfolioRepo.ListPublished(search, tagIDs, jurusanID, kelasID, userID, sort, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal mengambil data portfolio",
		))
	}

	var portfolioDTOs []dto.PortfolioListDTO
	for _, p := range portfolios {
		likeCount, _ := h.portfolioRepo.GetLikeCount(p.ID)
		pDTO := dto.PortfolioListDTO{
			ID:           p.ID,
			Judul:        p.Judul,
			Slug:         p.Slug,
			ThumbnailURL: p.ThumbnailURL,
			PublishedAt:  p.PublishedAt,
			CreatedAt:    p.CreatedAt,
			LikeCount:    likeCount,
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

		portfolioDTOs = append(portfolioDTOs, pDTO)
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return c.JSON(dto.SuccessWithMeta(portfolioDTOs, &dto.Meta{
		CurrentPage: page,
		PerPage:     limit,
		TotalPages:  totalPages,
		TotalCount:  total,
	}))
}

func (h *PortfolioHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	username := c.Query("username")
	currentUserID := middleware.GetUserID(c)

	var portfolio *domain.Portfolio
	var err error

	if username != "" {
		user, userErr := h.userRepo.FindByUsername(username)
		if userErr != nil {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
				"PORTFOLIO_NOT_FOUND", "Portfolio tidak ditemukan",
			))
		}
		portfolio, err = h.portfolioRepo.FindBySlugAndUserID(slug, user.ID)
	} else {
		// Find by slug only (first match)
		portfolios, _, _ := h.portfolioRepo.ListPublished("", nil, nil, nil, nil, "", 1, 100)
		for _, p := range portfolios {
			if p.Slug == slug {
				portfolio, err = h.portfolioRepo.FindByID(p.ID)
				break
			}
		}
	}

	if err != nil || portfolio == nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"PORTFOLIO_NOT_FOUND", "Portfolio tidak ditemukan",
		))
	}

	// Check visibility
	if portfolio.Status != domain.StatusPublished {
		isOwner := currentUserID != nil && *currentUserID == portfolio.UserID
		isAdmin := middleware.GetUserRole(c) == "admin"
		if !isOwner && !isAdmin {
			return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
				"PORTFOLIO_NOT_FOUND", "Portfolio tidak ditemukan",
			))
		}
	}

	// Record view for published portfolios
	if portfolio.Status == domain.StatusPublished && h.viewRepo != nil {
		sessionID := c.Get("X-Session-ID")
		var sessionPtr *string
		if sessionID != "" {
			sessionPtr = &sessionID
		}
		// Record view asynchronously to not block response
		go h.viewRepo.RecordView(portfolio.ID, currentUserID, sessionPtr)
	}

	return c.JSON(dto.SuccessResponse(h.toPortfolioDetailDTO(portfolio, currentUserID), ""))
}

func (h *PortfolioHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "ID tidak valid",
		))
	}

	currentUserID := middleware.GetUserID(c)
	portfolio, err := h.portfolioRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"PORTFOLIO_NOT_FOUND", "Portfolio tidak ditemukan",
		))
	}

	// Check access
	isOwner := currentUserID != nil && *currentUserID == portfolio.UserID
	isAdmin := middleware.GetUserRole(c) == "admin"
	if !isOwner && !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
			"FORBIDDEN", "Anda tidak memiliki akses ke portfolio ini",
		))
	}

	return c.JSON(dto.SuccessResponse(h.toPortfolioDetailDTO(portfolio, currentUserID), ""))
}

func (h *PortfolioHandler) GetMyPortfolios(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"UNAUTHORIZED", "User tidak terautentikasi",
		))
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	var status *string
	if s := c.Query("status"); s != "" {
		status = &s
	}

	portfolios, total, err := h.portfolioRepo.ListByUser(*userID, status, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal mengambil data portfolio",
		))
	}

	var portfolioDTOs []dto.MyPortfolioDTO
	for _, p := range portfolios {
		likeCount, _ := h.portfolioRepo.GetLikeCount(p.ID)
		portfolioDTOs = append(portfolioDTOs, dto.MyPortfolioDTO{
			ID:              p.ID,
			Judul:           p.Judul,
			Slug:            p.Slug,
			ThumbnailURL:    p.ThumbnailURL,
			Status:          string(p.Status),
			AdminReviewNote: p.AdminReviewNote,
			ReviewedAt:      p.ReviewedAt,
			PublishedAt:     p.PublishedAt,
			CreatedAt:       p.CreatedAt,
			UpdatedAt:       p.UpdatedAt,
			LikeCount:       likeCount,
		})
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return c.JSON(dto.SuccessWithMeta(portfolioDTOs, &dto.Meta{
		CurrentPage: page,
		PerPage:     limit,
		TotalPages:  totalPages,
		TotalCount:  total,
	}))
}

func (h *PortfolioHandler) Create(c *fiber.Ctx) error {
	currentUserID := middleware.GetUserID(c)
	if currentUserID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"UNAUTHORIZED", "User tidak terautentikasi",
		))
	}

	var req dto.CreatePortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Request body tidak valid",
		))
	}

	if req.Judul == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Validasi gagal",
			dto.ErrorDetail{Field: "judul", Message: "Judul wajib diisi"},
		))
	}

	// Determine owner and status based on who's creating
	isAdmin := middleware.GetUserRole(c) == "admin"
	ownerID := *currentUserID
	status := domain.StatusDraft

	// Admin can assign portfolio to another user and it's auto-published
	if isAdmin && req.UserID != nil {
		ownerID = *req.UserID
		status = domain.StatusPublished
	}

	portfolio := &domain.Portfolio{
		UserID: ownerID,
		Judul:  req.Judul,
		Status: status,
	}

	if err := h.portfolioRepo.Create(portfolio); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal membuat portfolio",
		))
	}

	if len(req.TagIDs) > 0 {
		h.portfolioRepo.UpdateTags(portfolio.ID, req.TagIDs)
	}

	if req.SeriesID != nil {
		h.portfolioRepo.UpdateSeriesID(portfolio.ID, req.SeriesID)
	}

	portfolio, _ = h.portfolioRepo.FindByID(portfolio.ID)

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse(h.toPortfolioDetailDTO(portfolio, currentUserID), "Portfolio berhasil dibuat"))
}

func (h *PortfolioHandler) Update(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "ID tidak valid",
		))
	}

	currentUserID := middleware.GetUserID(c)
	portfolio, err := h.portfolioRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"PORTFOLIO_NOT_FOUND", "Portfolio tidak ditemukan",
		))
	}

	isOwner := currentUserID != nil && *currentUserID == portfolio.UserID
	isAdmin := middleware.GetUserRole(c) == "admin"
	if !isOwner && !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
			"FORBIDDEN", "Anda tidak memiliki akses untuk mengedit portfolio ini",
		))
	}

	var req dto.UpdatePortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Request body tidak valid",
		))
	}

	if req.Judul != nil {
		portfolio.Judul = *req.Judul
	}
	if req.ThumbnailURL != nil {
		portfolio.ThumbnailURL = req.ThumbnailURL
	}

	if err := h.portfolioRepo.Update(portfolio); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal memperbarui portfolio",
		))
	}

	if req.TagIDs != nil {
		h.portfolioRepo.UpdateTags(portfolio.ID, req.TagIDs)
	}

	if req.SeriesID != nil {
		h.portfolioRepo.UpdateSeriesID(portfolio.ID, req.SeriesID)
	}

	return c.JSON(dto.SuccessResponse(map[string]interface{}{
		"id":         portfolio.ID,
		"judul":      portfolio.Judul,
		"slug":       portfolio.Slug,
		"status":     portfolio.Status,
		"updated_at": portfolio.UpdatedAt,
	}, "Portfolio berhasil diperbarui"))
}

func (h *PortfolioHandler) Delete(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "ID tidak valid",
		))
	}

	currentUserID := middleware.GetUserID(c)
	portfolio, err := h.portfolioRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"PORTFOLIO_NOT_FOUND", "Portfolio tidak ditemukan",
		))
	}

	isOwner := currentUserID != nil && *currentUserID == portfolio.UserID
	isAdmin := middleware.GetUserRole(c) == "admin"
	if !isOwner && !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
			"FORBIDDEN", "Anda tidak memiliki akses untuk menghapus portfolio ini",
		))
	}

	if err := h.portfolioRepo.Delete(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal menghapus portfolio",
		))
	}

	return c.JSON(dto.SuccessResponse(nil, "Portfolio berhasil dihapus"))
}

func (h *PortfolioHandler) Submit(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "ID tidak valid",
		))
	}

	currentUserID := middleware.GetUserID(c)
	portfolio, err := h.portfolioRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"PORTFOLIO_NOT_FOUND", "Portfolio tidak ditemukan",
		))
	}

	if currentUserID == nil || *currentUserID != portfolio.UserID {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
			"FORBIDDEN", "Anda tidak memiliki akses",
		))
	}

	if portfolio.Status != domain.StatusDraft && portfolio.Status != domain.StatusRejected {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"INVALID_STATUS_TRANSITION", "Portfolio hanya bisa disubmit dari status draft atau rejected",
		))
	}

	// Validate completeness
	var details []dto.ErrorDetail
	if portfolio.ThumbnailURL == nil {
		details = append(details, dto.ErrorDetail{Field: "thumbnail", Message: "Thumbnail wajib diisi sebelum submit"})
	}
	if len(portfolio.ContentBlocks) == 0 {
		details = append(details, dto.ErrorDetail{Field: "content_blocks", Message: "Portfolio harus memiliki minimal 1 content block"})
	}
	if len(details) > 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(dto.ErrorResponse(
			"INCOMPLETE_PORTFOLIO", "Portfolio belum lengkap", details...,
		))
	}

	portfolio.Status = domain.StatusPendingReview
	if err := h.portfolioRepo.Update(portfolio); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal submit portfolio",
		))
	}

	return c.JSON(dto.SuccessResponse(dto.PortfolioStatusResponse{
		ID:     portfolio.ID,
		Status: string(portfolio.Status),
	}, "Portfolio berhasil diajukan untuk review"))
}

func (h *PortfolioHandler) Archive(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "ID tidak valid",
		))
	}

	currentUserID := middleware.GetUserID(c)
	portfolio, err := h.portfolioRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"PORTFOLIO_NOT_FOUND", "Portfolio tidak ditemukan",
		))
	}

	isOwner := currentUserID != nil && *currentUserID == portfolio.UserID
	isAdmin := middleware.GetUserRole(c) == "admin"
	if !isOwner && !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
			"FORBIDDEN", "Anda tidak memiliki akses",
		))
	}

	portfolio.Status = domain.StatusArchived
	if err := h.portfolioRepo.Update(portfolio); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal mengarsipkan portfolio",
		))
	}

	return c.JSON(dto.SuccessResponse(dto.PortfolioStatusResponse{
		ID:     portfolio.ID,
		Status: string(portfolio.Status),
	}, "Portfolio berhasil diarsipkan"))
}

func (h *PortfolioHandler) Unarchive(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "ID tidak valid",
		))
	}

	currentUserID := middleware.GetUserID(c)
	portfolio, err := h.portfolioRepo.FindByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse(
			"PORTFOLIO_NOT_FOUND", "Portfolio tidak ditemukan",
		))
	}

	isOwner := currentUserID != nil && *currentUserID == portfolio.UserID
	isAdmin := middleware.GetUserRole(c) == "admin"
	if !isOwner && !isAdmin {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
			"FORBIDDEN", "Anda tidak memiliki akses",
		))
	}

	portfolio.Status = domain.StatusDraft
	if err := h.portfolioRepo.Update(portfolio); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal mengembalikan portfolio",
		))
	}

	return c.JSON(dto.SuccessResponse(dto.PortfolioStatusResponse{
		ID:     portfolio.ID,
		Status: string(portfolio.Status),
	}, "Portfolio berhasil dikembalikan"))
}

func (h *PortfolioHandler) Like(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "ID tidak valid",
		))
	}

	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"UNAUTHORIZED", "User tidak terautentikasi",
		))
	}

	// Admin cannot like portfolios
	if middleware.GetUserRole(c) == "admin" {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
			"FORBIDDEN", "Admin tidak dapat like portfolio",
		))
	}

	isLiked, _ := h.portfolioRepo.IsLiked(*userID, id)
	if isLiked {
		return c.Status(fiber.StatusConflict).JSON(dto.ErrorResponse(
			"ALREADY_LIKED", "Anda sudah like portfolio ini",
		))
	}

	if err := h.portfolioRepo.Like(*userID, id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal like portfolio",
		))
	}

	// Get portfolio for interest tracking and notification
	portfolio, _ := h.portfolioRepo.FindByID(id)

	// Update user interest profile (async)
	if h.interestRepo != nil && portfolio != nil {
		go func() {
			// Extract tag IDs from portfolio
			var tagIDs []uuid.UUID
			for _, tag := range portfolio.Tags {
				tagIDs = append(tagIDs, tag.ID)
			}
			if len(tagIDs) > 0 {
				h.interestRepo.UpdateTagInterest(*userID, tagIDs)
			}

			// Update jurusan interest if portfolio owner has kelas
			if portfolio.User != nil && portfolio.User.Kelas != nil {
				h.interestRepo.UpdateJurusanInterest(*userID, portfolio.User.Kelas.JurusanID)
			}

			// Increment total likes
			h.interestRepo.IncrementTotalLikes(*userID)
		}()
	}

	// Send notification to portfolio owner
	if h.notifService != nil && portfolio != nil {
		liker, _ := h.userRepo.FindByID(*userID)
		if liker != nil {
			_ = h.notifService.NotifyPortfolioLiked(liker, portfolio)
		}
	}

	likeCount, _ := h.portfolioRepo.GetLikeCount(id)

	return c.JSON(dto.SuccessResponse(dto.LikeResponse{
		IsLiked:   true,
		LikeCount: likeCount,
	}, "Portfolio berhasil di-like"))
}

func (h *PortfolioHandler) Unlike(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "ID tidak valid",
		))
	}

	userID := middleware.GetUserID(c)
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(dto.ErrorResponse(
			"UNAUTHORIZED", "User tidak terautentikasi",
		))
	}

	// Admin cannot unlike portfolios
	if middleware.GetUserRole(c) == "admin" {
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResponse(
			"FORBIDDEN", "Admin tidak dapat unlike portfolio",
		))
	}

	// Get portfolio for interest decrement before unlike
	portfolio, _ := h.portfolioRepo.FindByID(id)

	if err := h.portfolioRepo.Unlike(*userID, id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"INTERNAL_ERROR", "Gagal unlike portfolio",
		))
	}

	// Decrement user interest profile (async)
	if h.interestRepo != nil && portfolio != nil {
		go func() {
			// Extract tag IDs from portfolio
			var tagIDs []uuid.UUID
			for _, tag := range portfolio.Tags {
				tagIDs = append(tagIDs, tag.ID)
			}
			if len(tagIDs) > 0 {
				h.interestRepo.DecrementTagInterest(*userID, tagIDs)
			}

			// Decrement jurusan interest if portfolio owner has kelas
			if portfolio.User != nil && portfolio.User.Kelas != nil {
				h.interestRepo.DecrementJurusanInterest(*userID, portfolio.User.Kelas.JurusanID)
			}

			// Decrement total likes
			h.interestRepo.DecrementTotalLikes(*userID)
		}()
	}

	likeCount, _ := h.portfolioRepo.GetLikeCount(id)

	return c.JSON(dto.SuccessResponse(dto.LikeResponse{
		IsLiked:   false,
		LikeCount: likeCount,
	}, "Like berhasil dihapus"))
}

func (h *PortfolioHandler) toPortfolioDetailDTO(p *domain.Portfolio, currentUserID *uuid.UUID) dto.PortfolioDetailDTO {
	likeCount, _ := h.portfolioRepo.GetLikeCount(p.ID)
	isLiked := false
	if currentUserID != nil {
		isLiked, _ = h.portfolioRepo.IsLiked(*currentUserID, p.ID)
	}

	pDTO := dto.PortfolioDetailDTO{
		ID:              p.ID,
		Judul:           p.Judul,
		Slug:            p.Slug,
		ThumbnailURL:    p.ThumbnailURL,
		Status:          string(p.Status),
		AdminReviewNote: p.AdminReviewNote,
		ReviewedAt:      p.ReviewedAt,
		PublishedAt:     p.PublishedAt,
		CreatedAt:       p.CreatedAt,
		UpdatedAt:       p.UpdatedAt,
		LikeCount:       likeCount,
		IsLiked:         isLiked,
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

	if p.Series != nil {
		pDTO.Series = &dto.PortfolioSeriesDTO{ID: p.Series.ID, Nama: p.Series.Nama}
	}

	for _, b := range p.ContentBlocks {
		pDTO.ContentBlocks = append(pDTO.ContentBlocks, dto.ContentBlockDTO{
			ID:         b.ID,
			BlockType:  string(b.BlockType),
			BlockOrder: b.BlockOrder,
			Payload:    b.Payload,
			CreatedAt:  b.CreatedAt,
			UpdatedAt:  b.UpdatedAt,
		})
	}

	return pDTO
}
