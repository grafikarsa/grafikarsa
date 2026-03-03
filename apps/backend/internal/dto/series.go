package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
)

// SeriesDTO untuk response list
type SeriesDTO struct {
	ID             uuid.UUID `json:"id"`
	Nama           string    `json:"nama"`
	Deskripsi      *string   `json:"deskripsi,omitempty"`
	IsActive       bool      `json:"is_active"`
	BlockCount     int       `json:"block_count"`
	PortfolioCount int64     `json:"portfolio_count"`
	CreatedAt      time.Time `json:"created_at"`
}

// SeriesDetailDTO untuk response detail dengan blocks
type SeriesDetailDTO struct {
	ID             uuid.UUID        `json:"id"`
	Nama           string           `json:"nama"`
	Deskripsi      *string          `json:"deskripsi,omitempty"`
	IsActive       bool             `json:"is_active"`
	Blocks         []SeriesBlockDTO `json:"blocks"`
	PortfolioCount int64            `json:"portfolio_count"`
	CreatedAt      time.Time        `json:"created_at"`
}

// SeriesBlockDTO untuk response block template
type SeriesBlockDTO struct {
	ID         uuid.UUID `json:"id"`
	BlockType  string    `json:"block_type"`
	BlockOrder int       `json:"block_order"`
	Instruksi  string    `json:"instruksi"`
}

// SeriesBriefDTO untuk dropdown/select (minimal info)
type SeriesBriefDTO struct {
	ID         uuid.UUID        `json:"id"`
	Nama       string           `json:"nama"`
	Deskripsi  *string          `json:"deskripsi,omitempty"`
	BlockCount int              `json:"block_count"`
	Blocks     []SeriesBlockDTO `json:"blocks,omitempty"`
}

// CreateSeriesRequest untuk admin create
type CreateSeriesRequest struct {
	Nama      string                     `json:"nama" validate:"required,max=100"`
	Deskripsi *string                    `json:"deskripsi,omitempty"`
	IsActive  *bool                      `json:"is_active,omitempty"`
	Blocks    []CreateSeriesBlockRequest `json:"blocks" validate:"required,min=1"`
}

// CreateSeriesBlockRequest untuk block dalam create series
type CreateSeriesBlockRequest struct {
	BlockType string `json:"block_type" validate:"required"`
	Instruksi string `json:"instruksi" validate:"required"`
}

// UpdateSeriesRequest untuk admin update
type UpdateSeriesRequest struct {
	Nama      *string                    `json:"nama,omitempty" validate:"omitempty,max=100"`
	Deskripsi *string                    `json:"deskripsi,omitempty"`
	IsActive  *bool                      `json:"is_active,omitempty"`
	Blocks    []CreateSeriesBlockRequest `json:"blocks,omitempty"`
}

// Helper function to convert domain.SeriesBlock to DTO
func SeriesBlockToDTO(block domain.SeriesBlock) SeriesBlockDTO {
	return SeriesBlockDTO{
		ID:         block.ID,
		BlockType:  string(block.BlockType),
		BlockOrder: block.BlockOrder,
		Instruksi:  block.Instruksi,
	}
}

// Helper function to convert slice of domain.SeriesBlock to DTOs
func SeriesBlocksToDTOs(blocks []domain.SeriesBlock) []SeriesBlockDTO {
	result := make([]SeriesBlockDTO, len(blocks))
	for i, b := range blocks {
		result[i] = SeriesBlockToDTO(b)
	}
	return result
}

// ============================================================================
// EXPORT DTOs
// ============================================================================

// PortfolioExportUserDTO - user data for export
type PortfolioExportUserDTO struct {
	ID          uuid.UUID `json:"id"`
	Nama        string    `json:"nama"`
	Username    string    `json:"username"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	NISN        *string   `json:"nisn,omitempty"`
	NIS         *string   `json:"nis,omitempty"`
	KelasNama   *string   `json:"kelas_nama,omitempty"`
	JurusanNama *string   `json:"jurusan_nama,omitempty"`
}

// PortfolioExportDTO - portfolio data for export
type PortfolioExportDTO struct {
	ID            uuid.UUID              `json:"id"`
	Judul         string                 `json:"judul"`
	ThumbnailURL  *string                `json:"thumbnail_url,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	ContentBlocks []ContentBlockDTO      `json:"content_blocks"`
	User          PortfolioExportUserDTO `json:"user"`
}

// SeriesExportResponse - response for export endpoint
type SeriesExportResponse struct {
	Series     SeriesDetailDTO      `json:"series"`
	Portfolios []PortfolioExportDTO `json:"portfolios"`
	Meta       ExportMeta           `json:"meta"`
}

// ExportMeta - metadata for export
type ExportMeta struct {
	TotalCount int64     `json:"total_count"`
	UserCount  int64     `json:"user_count"`
	ExportedAt time.Time `json:"exported_at"`
}

// ExportPreviewResponse - preview before export
type ExportPreviewResponse struct {
	PortfolioCount int64 `json:"portfolio_count"`
	UserCount      int64 `json:"user_count"`
	EstimatedPages int64 `json:"estimated_pages"`
}
