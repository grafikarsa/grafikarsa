package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"github.com/grafikarsa/backend/internal/dto"
	"github.com/grafikarsa/backend/internal/middleware"
	"github.com/grafikarsa/backend/internal/repository"
)

type AssessmentHandler struct {
	assessmentRepo *repository.AssessmentRepository
	portfolioRepo  *repository.PortfolioRepository
}

func NewAssessmentHandler(assessmentRepo *repository.AssessmentRepository, portfolioRepo *repository.PortfolioRepository) *AssessmentHandler {
	return &AssessmentHandler{
		assessmentRepo: assessmentRepo,
		portfolioRepo:  portfolioRepo,
	}
}

// ============================================================================
// ASSESSMENT STATS HANDLER
// ============================================================================

// GetAssessmentStats - GET /admin/assessments/stats
func (h *AssessmentHandler) GetAssessmentStats(c *fiber.Ctx) error {
	stats, err := h.assessmentRepo.GetAssessmentStats()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"FETCH_FAILED", "Gagal mengambil statistik penilaian",
		))
	}

	return c.JSON(dto.SuccessResponse(stats, ""))
}

// ============================================================================
// ASSESSMENT METRICS HANDLERS
// ============================================================================

// ListMetrics - GET /admin/assessment-metrics
func (h *AssessmentHandler) ListMetrics(c *fiber.Ctx) error {
	activeOnly := c.QueryBool("active_only", false)

	metrics, err := h.assessmentRepo.ListMetrics(activeOnly)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"FETCH_FAILED", "Gagal mengambil data metrik",
		))
	}

	var responses []dto.MetricResponse
	for _, m := range metrics {
		responses = append(responses, h.toMetricResponse(&m))
	}

	return c.JSON(dto.SuccessResponse(responses, ""))
}

// CreateMetric - POST /admin/assessment-metrics
func (h *AssessmentHandler) CreateMetric(c *fiber.Ctx) error {
	var req dto.CreateMetricRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"INVALID_REQUEST", "Format request tidak valid",
		))
	}

	if req.Nama == "" || len(req.Nama) < 2 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse(
			"VALIDATION_ERROR", "Nama metrik minimal 2 karakter",
		))
	}

	metric := &domain.AssessmentMetric{
		Nama:      req.Nama,
		Deskripsi: req.Deskripsi,
		IsActive:  true,
	}

	if err := h.assessmentRepo.CreateMetric(metric); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"CREATE_FAILED", "Gagal membuat metrik",
		))
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse(
		h.toMetricResponse(metric), "Metrik berhasil dibuat",
	))
}

// UpdateMetric - PATCH /admin/assessment-metrics/:id
func (h *AssessmentHandler) UpdateMetric(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "ID tidak valid"))
	}

	metric, err := h.assessmentRepo.FindMetricByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Metrik tidak ditemukan"))
	}

	var req dto.UpdateMetricRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_REQUEST", "Format request tidak valid"))
	}

	if req.Nama != nil {
		metric.Nama = *req.Nama
	}
	if req.Deskripsi != nil {
		metric.Deskripsi = req.Deskripsi
	}
	if req.IsActive != nil {
		metric.IsActive = *req.IsActive
	}

	if err := h.assessmentRepo.UpdateMetric(metric); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("UPDATE_FAILED", "Gagal mengupdate metrik"))
	}

	return c.JSON(dto.SuccessResponse(h.toMetricResponse(metric), "Metrik berhasil diupdate"))
}

// DeleteMetric - DELETE /admin/assessment-metrics/:id
func (h *AssessmentHandler) DeleteMetric(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "ID tidak valid"))
	}

	if _, err := h.assessmentRepo.FindMetricByID(id); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Metrik tidak ditemukan"))
	}

	if err := h.assessmentRepo.DeleteMetric(id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("DELETE_FAILED", "Gagal menghapus metrik"))
	}

	return c.JSON(dto.SuccessResponse(nil, "Metrik berhasil dihapus"))
}

// ReorderMetrics - PUT /admin/assessment-metrics/reorder
func (h *AssessmentHandler) ReorderMetrics(c *fiber.Ctx) error {
	var req dto.ReorderMetricsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_REQUEST", "Format request tidak valid"))
	}

	if len(req.Orders) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Orders tidak boleh kosong"))
	}

	orders := make([]struct {
		ID     uuid.UUID
		Urutan int
	}, len(req.Orders))

	for i, o := range req.Orders {
		id, err := uuid.Parse(o.ID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "ID tidak valid: "+o.ID))
		}
		orders[i] = struct {
			ID     uuid.UUID
			Urutan int
		}{ID: id, Urutan: o.Urutan}
	}

	if err := h.assessmentRepo.ReorderMetrics(orders); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("REORDER_FAILED", "Gagal mengubah urutan metrik"))
	}

	return c.JSON(dto.SuccessResponse(nil, "Urutan metrik berhasil diubah"))
}

// ============================================================================
// PORTFOLIO ASSESSMENT HANDLERS
// ============================================================================

// ListPortfoliosForAssessment - GET /admin/assessments
func (h *AssessmentHandler) ListPortfoliosForAssessment(c *fiber.Ctx) error {
	filter := c.Query("filter", "all") // pending, assessed, all
	search := c.Query("search")
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	portfolios, total, err := h.assessmentRepo.ListPublishedPortfolios(filter, search, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse(
			"FETCH_FAILED", "Gagal mengambil data portfolio",
		))
	}

	var responses []dto.PortfolioForAssessment
	for _, p := range portfolios {
		resp := dto.PortfolioForAssessment{
			ID:           p.Portfolio.ID.String(),
			Judul:        p.Portfolio.Judul,
			Slug:         p.Portfolio.Slug,
			ThumbnailURL: p.Portfolio.ThumbnailURL,
			PublishedAt:  p.Portfolio.PublishedAt,
		}
		if p.Portfolio.User != nil {
			resp.User = &dto.UserBriefResponse{
				ID:        p.Portfolio.User.ID.String(),
				Username:  p.Portfolio.User.Username,
				Nama:      p.Portfolio.User.Nama,
				AvatarURL: p.Portfolio.User.AvatarURL,
			}
		}
		if p.Assessment != nil {
			resp.Assessment = &dto.AssessmentBrief{
				ID:         p.Assessment.ID.String(),
				TotalScore: p.Assessment.TotalScore,
				AssessedAt: p.Assessment.CreatedAt,
			}
			if p.Assessment.Assessor != nil {
				resp.Assessment.Assessor = &dto.UserBriefResponse{
					ID:        p.Assessment.Assessor.ID.String(),
					Username:  p.Assessment.Assessor.Username,
					Nama:      p.Assessment.Assessor.Nama,
					AvatarURL: p.Assessment.Assessor.AvatarURL,
				}
			}
		}
		responses = append(responses, resp)
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

// GetAssessment - GET /admin/assessments/:portfolio_id
func (h *AssessmentHandler) GetAssessment(c *fiber.Ctx) error {
	portfolioID, err := uuid.Parse(c.Params("portfolio_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "Portfolio ID tidak valid"))
	}

	// Check portfolio exists and is published
	portfolio, err := h.portfolioRepo.FindByID(portfolioID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Portfolio tidak ditemukan"))
	}
	if portfolio.Status != domain.StatusPublished {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_STATUS", "Portfolio belum dipublish"))
	}

	assessment, err := h.assessmentRepo.FindAssessmentByPortfolioID(portfolioID)
	if err != nil {
		// Return empty assessment with portfolio info
		return c.JSON(dto.SuccessResponse(fiber.Map{
			"portfolio": dto.PortfolioBrief{
				ID:           portfolio.ID.String(),
				Judul:        portfolio.Judul,
				Slug:         portfolio.Slug,
				ThumbnailURL: portfolio.ThumbnailURL,
			},
			"assessment": nil,
		}, ""))
	}

	return c.JSON(dto.SuccessResponse(fiber.Map{
		"portfolio": dto.PortfolioBrief{
			ID:           portfolio.ID.String(),
			Judul:        portfolio.Judul,
			Slug:         portfolio.Slug,
			ThumbnailURL: portfolio.ThumbnailURL,
		},
		"assessment": h.toAssessmentResponse(assessment),
	}, ""))
}

// CreateOrUpdateAssessment - POST /admin/assessments/:portfolio_id
func (h *AssessmentHandler) CreateOrUpdateAssessment(c *fiber.Ctx) error {
	portfolioID, err := uuid.Parse(c.Params("portfolio_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "Portfolio ID tidak valid"))
	}

	// Check portfolio exists and is published
	portfolio, err := h.portfolioRepo.FindByID(portfolioID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Portfolio tidak ditemukan"))
	}
	if portfolio.Status != domain.StatusPublished {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_STATUS", "Portfolio belum dipublish"))
	}

	var req dto.CreateAssessmentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_REQUEST", "Format request tidak valid"))
	}

	if len(req.Scores) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Minimal satu nilai harus diisi"))
	}

	// Validate scores
	for _, s := range req.Scores {
		if s.Score < 1 || s.Score > 10 {
			return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("VALIDATION_ERROR", "Nilai harus antara 1-10"))
		}
	}

	adminID := middleware.GetUserID(c)

	// Check if assessment already exists
	existingAssessment, _ := h.assessmentRepo.FindAssessmentByPortfolioID(portfolioID)

	if existingAssessment != nil {
		// Update existing assessment
		existingAssessment.FinalComment = req.FinalComment
		existingAssessment.AssessedBy = *adminID

		if err := h.assessmentRepo.UpdateAssessment(existingAssessment); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("UPDATE_FAILED", "Gagal mengupdate penilaian"))
		}

		// Replace scores
		scores := make([]domain.PortfolioAssessmentScore, len(req.Scores))
		for i, s := range req.Scores {
			metricID, _ := uuid.Parse(s.MetricID)
			scores[i] = domain.PortfolioAssessmentScore{
				AssessmentID: existingAssessment.ID,
				MetricID:     metricID,
				Score:        s.Score,
				Comment:      s.Comment,
			}
		}

		if err := h.assessmentRepo.ReplaceScores(existingAssessment.ID, scores); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("UPDATE_FAILED", "Gagal mengupdate nilai"))
		}

		// Reload assessment
		assessment, _ := h.assessmentRepo.FindAssessmentByPortfolioID(portfolioID)
		return c.JSON(dto.SuccessResponse(h.toAssessmentResponse(assessment), "Penilaian berhasil diupdate"))
	}

	// Create new assessment
	assessment := &domain.PortfolioAssessment{
		PortfolioID:  portfolioID,
		AssessedBy:   *adminID,
		FinalComment: req.FinalComment,
	}

	if err := h.assessmentRepo.CreateAssessment(assessment); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("CREATE_FAILED", "Gagal membuat penilaian"))
	}

	// Create scores
	scores := make([]domain.PortfolioAssessmentScore, len(req.Scores))
	for i, s := range req.Scores {
		metricID, _ := uuid.Parse(s.MetricID)
		scores[i] = domain.PortfolioAssessmentScore{
			AssessmentID: assessment.ID,
			MetricID:     metricID,
			Score:        s.Score,
			Comment:      s.Comment,
		}
	}

	if err := h.assessmentRepo.CreateScores(scores); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("CREATE_FAILED", "Gagal menyimpan nilai"))
	}

	// Reload assessment with scores
	assessment, _ = h.assessmentRepo.FindAssessmentByPortfolioID(portfolioID)

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResponse(h.toAssessmentResponse(assessment), "Penilaian berhasil disimpan"))
}

// DeleteAssessment - DELETE /admin/assessments/:portfolio_id
func (h *AssessmentHandler) DeleteAssessment(c *fiber.Ctx) error {
	portfolioID, err := uuid.Parse(c.Params("portfolio_id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResponse("INVALID_ID", "Portfolio ID tidak valid"))
	}

	assessment, err := h.assessmentRepo.FindAssessmentByPortfolioID(portfolioID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResponse("NOT_FOUND", "Penilaian tidak ditemukan"))
	}

	if err := h.assessmentRepo.DeleteAssessment(assessment.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResponse("DELETE_FAILED", "Gagal menghapus penilaian"))
	}

	return c.JSON(dto.SuccessResponse(nil, "Penilaian berhasil dihapus"))
}

// ============================================================================
// HELPER METHODS
// ============================================================================

func (h *AssessmentHandler) toMetricResponse(m *domain.AssessmentMetric) dto.MetricResponse {
	return dto.MetricResponse{
		ID:        m.ID.String(),
		Nama:      m.Nama,
		Deskripsi: m.Deskripsi,
		Urutan:    m.Urutan,
		IsActive:  m.IsActive,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func (h *AssessmentHandler) toAssessmentResponse(a *domain.PortfolioAssessment) dto.AssessmentResponse {
	resp := dto.AssessmentResponse{
		ID:           a.ID.String(),
		PortfolioID:  a.PortfolioID.String(),
		AssessedBy:   a.AssessedBy.String(),
		FinalComment: a.FinalComment,
		TotalScore:   a.TotalScore,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}

	if a.Assessor != nil {
		resp.Assessor = &dto.UserBriefResponse{
			ID:        a.Assessor.ID.String(),
			Username:  a.Assessor.Username,
			Nama:      a.Assessor.Nama,
			AvatarURL: a.Assessor.AvatarURL,
		}
	}

	if a.Portfolio != nil {
		resp.Portfolio = &dto.PortfolioBrief{
			ID:           a.Portfolio.ID.String(),
			Judul:        a.Portfolio.Judul,
			Slug:         a.Portfolio.Slug,
			ThumbnailURL: a.Portfolio.ThumbnailURL,
		}
	}

	for _, s := range a.Scores {
		scoreResp := dto.ScoreResponse{
			ID:        s.ID.String(),
			MetricID:  s.MetricID.String(),
			Score:     s.Score,
			Comment:   s.Comment,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		}
		if s.Metric != nil {
			scoreResp.Metric = &dto.MetricResponse{
				ID:        s.Metric.ID.String(),
				Nama:      s.Metric.Nama,
				Deskripsi: s.Metric.Deskripsi,
				Urutan:    s.Metric.Urutan,
				IsActive:  s.Metric.IsActive,
				CreatedAt: s.Metric.CreatedAt,
				UpdatedAt: s.Metric.UpdatedAt,
			}
		}
		resp.Scores = append(resp.Scores, scoreResp)
	}

	return resp
}
