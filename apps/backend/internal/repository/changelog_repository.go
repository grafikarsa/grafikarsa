package repository

import (
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"gorm.io/gorm"
)

type ChangelogRepository struct {
	db *gorm.DB
}

func NewChangelogRepository(db *gorm.DB) *ChangelogRepository {
	return &ChangelogRepository{db: db}
}

// ============================================================================
// CHANGELOG CRUD
// ============================================================================

func (r *ChangelogRepository) Create(changelog *domain.Changelog) error {
	return r.db.Create(changelog).Error
}

func (r *ChangelogRepository) FindByID(id uuid.UUID) (*domain.Changelog, error) {
	var changelog domain.Changelog
	err := r.db.
		Preload("Creator").
		Preload("Sections", func(db *gorm.DB) *gorm.DB {
			return db.Order("section_order ASC")
		}).
		Preload("Sections.Blocks", func(db *gorm.DB) *gorm.DB {
			return db.Order("block_order ASC")
		}).
		Preload("Contributors", func(db *gorm.DB) *gorm.DB {
			return db.Order("contributor_order ASC")
		}).
		Preload("Contributors.User").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&changelog).Error
	if err != nil {
		return nil, err
	}
	return &changelog, nil
}

func (r *ChangelogRepository) Update(changelog *domain.Changelog) error {
	return r.db.Save(changelog).Error
}

func (r *ChangelogRepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&domain.Changelog{}).Error
}

// ============================================================================
// LIST METHODS
// ============================================================================

// ListPublished - list published changelogs for public view
func (r *ChangelogRepository) ListPublished(page, limit int) ([]domain.Changelog, int64, error) {
	var changelogs []domain.Changelog
	var total int64

	query := r.db.Model(&domain.Changelog{}).
		Where("is_published = ? AND deleted_at IS NULL", true)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.
		Preload("Creator").
		Preload("Sections", func(db *gorm.DB) *gorm.DB {
			return db.Order("section_order ASC")
		}).
		Preload("Sections.Blocks", func(db *gorm.DB) *gorm.DB {
			return db.Order("block_order ASC")
		}).
		Preload("Contributors", func(db *gorm.DB) *gorm.DB {
			return db.Order("contributor_order ASC")
		}).
		Preload("Contributors.User").
		Order("release_date DESC, created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&changelogs).Error

	if err != nil {
		return nil, 0, err
	}

	return changelogs, total, nil
}

// ListAll - list all changelogs for admin (including drafts)
func (r *ChangelogRepository) ListAll(page, limit int, search string) ([]domain.Changelog, int64, error) {
	var changelogs []domain.Changelog
	var total int64

	query := r.db.Model(&domain.Changelog{}).Where("deleted_at IS NULL")

	if search != "" {
		query = query.Where("title ILIKE ? OR version ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err := query.
		Preload("Sections").
		Order("release_date DESC, created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&changelogs).Error

	if err != nil {
		return nil, 0, err
	}

	return changelogs, total, nil
}

// GetLatest - get latest published changelog
func (r *ChangelogRepository) GetLatest() (*domain.Changelog, error) {
	var changelog domain.Changelog
	err := r.db.
		Preload("Creator").
		Preload("Sections", func(db *gorm.DB) *gorm.DB {
			return db.Order("section_order ASC")
		}).
		Preload("Sections.Blocks", func(db *gorm.DB) *gorm.DB {
			return db.Order("block_order ASC")
		}).
		Preload("Contributors", func(db *gorm.DB) *gorm.DB {
			return db.Order("contributor_order ASC")
		}).
		Preload("Contributors.User").
		Where("is_published = ? AND deleted_at IS NULL", true).
		Order("release_date DESC, created_at DESC").
		First(&changelog).Error
	if err != nil {
		return nil, err
	}
	return &changelog, nil
}

// ============================================================================
// PUBLISH/UNPUBLISH
// ============================================================================

func (r *ChangelogRepository) Publish(id uuid.UUID) error {
	return r.db.Model(&domain.Changelog{}).
		Where("id = ?", id).
		Update("is_published", true).Error
}

func (r *ChangelogRepository) Unpublish(id uuid.UUID) error {
	return r.db.Model(&domain.Changelog{}).
		Where("id = ?", id).
		Update("is_published", false).Error
}

// ============================================================================
// SECTIONS MANAGEMENT
// ============================================================================

func (r *ChangelogRepository) DeleteSections(changelogID uuid.UUID) error {
	// Delete blocks first (cascade should handle this, but being explicit)
	var sectionIDs []uuid.UUID
	r.db.Model(&domain.ChangelogSection{}).
		Where("changelog_id = ?", changelogID).
		Pluck("id", &sectionIDs)

	if len(sectionIDs) > 0 {
		r.db.Where("section_id IN ?", sectionIDs).Delete(&domain.ChangelogSectionBlock{})
	}

	return r.db.Where("changelog_id = ?", changelogID).Delete(&domain.ChangelogSection{}).Error
}

func (r *ChangelogRepository) CreateSection(section *domain.ChangelogSection) error {
	return r.db.Create(section).Error
}

func (r *ChangelogRepository) CreateSectionBlock(block *domain.ChangelogSectionBlock) error {
	return r.db.Create(block).Error
}

// ============================================================================
// CONTRIBUTORS MANAGEMENT
// ============================================================================

func (r *ChangelogRepository) DeleteContributors(changelogID uuid.UUID) error {
	return r.db.Where("changelog_id = ?", changelogID).Delete(&domain.ChangelogContributor{}).Error
}

func (r *ChangelogRepository) CreateContributor(contributor *domain.ChangelogContributor) error {
	return r.db.Create(contributor).Error
}

// ============================================================================
// READ TRACKING
// ============================================================================

func (r *ChangelogRepository) MarkAsRead(userID, changelogID uuid.UUID) error {
	// Use ON CONFLICT to handle duplicate
	return r.db.Exec(`
		INSERT INTO changelog_reads (id, user_id, changelog_id, read_at)
		VALUES (uuid_generate_v4(), ?, ?, NOW())
		ON CONFLICT (user_id, changelog_id) DO NOTHING
	`, userID, changelogID).Error
}

func (r *ChangelogRepository) MarkAllAsRead(userID uuid.UUID) error {
	// Get all published changelog IDs that user hasn't read
	return r.db.Exec(`
		INSERT INTO changelog_reads (id, user_id, changelog_id, read_at)
		SELECT uuid_generate_v4(), ?, c.id, NOW()
		FROM changelogs c
		WHERE c.is_published = true 
		AND c.deleted_at IS NULL
		AND NOT EXISTS (
			SELECT 1 FROM changelog_reads cr 
			WHERE cr.user_id = ? AND cr.changelog_id = c.id
		)
	`, userID, userID).Error
}

func (r *ChangelogRepository) GetUnreadCount(userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Raw(`
		SELECT COUNT(*) FROM changelogs c
		WHERE c.is_published = true 
		AND c.deleted_at IS NULL
		AND NOT EXISTS (
			SELECT 1 FROM changelog_reads cr 
			WHERE cr.user_id = ? AND cr.changelog_id = c.id
		)
	`, userID).Scan(&count).Error
	return count, err
}

// GetReadChangelogIDs - get list of changelog IDs that user has read
func (r *ChangelogRepository) GetReadChangelogIDs(userID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.Model(&domain.ChangelogRead{}).
		Where("user_id = ?", userID).
		Pluck("changelog_id", &ids).Error
	return ids, err
}
