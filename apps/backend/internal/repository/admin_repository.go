package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"gorm.io/gorm"
)

type AdminRepository struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// Jurusan
func (r *AdminRepository) CreateJurusan(jurusan *domain.Jurusan) error {
	return r.db.Create(jurusan).Error
}

func (r *AdminRepository) FindJurusanByID(id uuid.UUID) (*domain.Jurusan, error) {
	var jurusan domain.Jurusan
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&jurusan).Error
	return &jurusan, err
}

func (r *AdminRepository) ListJurusan() ([]domain.Jurusan, error) {
	var jurusan []domain.Jurusan
	err := r.db.Where("deleted_at IS NULL").Order("nama ASC").Find(&jurusan).Error
	return jurusan, err
}

func (r *AdminRepository) UpdateJurusan(jurusan *domain.Jurusan) error {
	return r.db.Save(jurusan).Error
}

func (r *AdminRepository) DeleteJurusan(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&domain.Jurusan{}).Error
}

func (r *AdminRepository) JurusanKodeExists(kode string, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.Model(&domain.Jurusan{}).Where("kode = ? AND deleted_at IS NULL", kode)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *AdminRepository) JurusanHasKelas(id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&domain.Kelas{}).Where("jurusan_id = ? AND deleted_at IS NULL", id).Count(&count).Error
	return count > 0, err
}

// Tahun Ajaran
func (r *AdminRepository) CreateTahunAjaran(ta *domain.TahunAjaran) error {
	return r.db.Create(ta).Error
}

func (r *AdminRepository) FindTahunAjaranByID(id uuid.UUID) (*domain.TahunAjaran, error) {
	var ta domain.TahunAjaran
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&ta).Error
	return &ta, err
}

func (r *AdminRepository) ListTahunAjaran() ([]domain.TahunAjaran, error) {
	var ta []domain.TahunAjaran
	err := r.db.Where("deleted_at IS NULL").Order("tahun_mulai DESC").Find(&ta).Error
	return ta, err
}

func (r *AdminRepository) UpdateTahunAjaran(ta *domain.TahunAjaran) error {
	return r.db.Save(ta).Error
}

func (r *AdminRepository) DeleteTahunAjaran(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&domain.TahunAjaran{}).Error
}

func (r *AdminRepository) TahunAjaranExists(tahunMulai int, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.Model(&domain.TahunAjaran{}).Where("tahun_mulai = ? AND deleted_at IS NULL", tahunMulai)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *AdminRepository) TahunAjaranHasKelas(id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&domain.Kelas{}).Where("tahun_ajaran_id = ? AND deleted_at IS NULL", id).Count(&count).Error
	return count > 0, err
}

func (r *AdminRepository) DeactivateAllTahunAjaran() error {
	return r.db.Model(&domain.TahunAjaran{}).Where("deleted_at IS NULL").Update("is_active", false).Error
}

// Kelas
func (r *AdminRepository) CreateKelas(kelas *domain.Kelas) error {
	return r.db.Create(kelas).Error
}

func (r *AdminRepository) FindKelasByID(id uuid.UUID) (*domain.Kelas, error) {
	var kelas domain.Kelas
	err := r.db.Preload("TahunAjaran").Preload("Jurusan").
		Where("id = ? AND deleted_at IS NULL", id).First(&kelas).Error
	return &kelas, err
}

func (r *AdminRepository) ListKelas(tahunAjaranID, jurusanID *uuid.UUID, tingkat *int, page, limit int) ([]domain.Kelas, int64, error) {
	var kelas []domain.Kelas
	var total int64

	query := r.db.Model(&domain.Kelas{}).Where("deleted_at IS NULL")

	if tahunAjaranID != nil {
		query = query.Where("tahun_ajaran_id = ?", *tahunAjaranID)
	}
	if jurusanID != nil {
		query = query.Where("jurusan_id = ?", *jurusanID)
	}
	if tingkat != nil {
		query = query.Where("tingkat = ?", *tingkat)
	}

	query.Count(&total)

	offset := (page - 1) * limit
	err := query.Preload("TahunAjaran").Preload("Jurusan").
		Offset(offset).Limit(limit).
		Order("nama ASC").
		Find(&kelas).Error

	return kelas, total, err
}

func (r *AdminRepository) ListKelasPublic(jurusanID *uuid.UUID, tingkat *int) ([]domain.Kelas, error) {
	var kelas []domain.Kelas

	query := r.db.Model(&domain.Kelas{}).
		Joins("JOIN tahun_ajaran ON kelas.tahun_ajaran_id = tahun_ajaran.id").
		Where("kelas.deleted_at IS NULL AND tahun_ajaran.is_active = true")

	if jurusanID != nil {
		query = query.Where("kelas.jurusan_id = ?", *jurusanID)
	}
	if tingkat != nil {
		query = query.Where("kelas.tingkat = ?", *tingkat)
	}

	err := query.Preload("Jurusan").Order("kelas.nama ASC").Find(&kelas).Error
	return kelas, err
}

func (r *AdminRepository) UpdateKelas(kelas *domain.Kelas) error {
	return r.db.Save(kelas).Error
}

func (r *AdminRepository) DeleteKelas(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&domain.Kelas{}).Error
}

func (r *AdminRepository) KelasExists(tahunAjaranID, jurusanID uuid.UUID, tingkat int, rombel string, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.Model(&domain.Kelas{}).
		Where("tahun_ajaran_id = ? AND jurusan_id = ? AND tingkat = ? AND rombel = ? AND deleted_at IS NULL",
			tahunAjaranID, jurusanID, tingkat, rombel)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *AdminRepository) KelasHasStudents(id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&domain.User{}).Where("kelas_id = ? AND deleted_at IS NULL", id).Count(&count).Error
	return count > 0, err
}

func (r *AdminRepository) GetKelasStudentCount(id uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&domain.User{}).Where("kelas_id = ? AND deleted_at IS NULL", id).Count(&count).Error
	return count, err
}

// Tags
func (r *AdminRepository) CreateTag(tag *domain.Tag) error {
	return r.db.Create(tag).Error
}

func (r *AdminRepository) FindTagByID(id uuid.UUID) (*domain.Tag, error) {
	var tag domain.Tag
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&tag).Error
	return &tag, err
}

func (r *AdminRepository) ListTags(search string) ([]domain.Tag, error) {
	var tags []domain.Tag
	query := r.db.Where("deleted_at IS NULL")
	if search != "" {
		query = query.Where("nama ILIKE ?", "%"+search+"%")
	}
	err := query.Order("nama ASC").Find(&tags).Error
	return tags, err
}

func (r *AdminRepository) UpdateTag(tag *domain.Tag) error {
	return r.db.Save(tag).Error
}

func (r *AdminRepository) DeleteTag(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&domain.Tag{}).Error
}

func (r *AdminRepository) TagNameExists(nama string, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.Model(&domain.Tag{}).Where("nama = ? AND deleted_at IS NULL", nama)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *AdminRepository) GetTagPortfolioCount(id uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&domain.PortfolioTag{}).Where("tag_id = ?", id).Count(&count).Error
	return count, err
}

// Series CRUD
func (r *AdminRepository) ListSeries(search string, page, limit int) ([]domain.Series, int64, error) {
	var series []domain.Series
	var total int64

	query := r.db.Model(&domain.Series{}).Where("deleted_at IS NULL")

	if search != "" {
		query = query.Where("nama ILIKE ?", "%"+search+"%")
	}

	query.Count(&total)

	offset := (page - 1) * limit
	err := query.Preload("Blocks", func(db *gorm.DB) *gorm.DB {
		return db.Order("block_order ASC")
	}).Order("nama ASC").Offset(offset).Limit(limit).Find(&series).Error

	return series, total, err
}

func (r *AdminRepository) CreateSeries(series *domain.Series) error {
	return r.db.Create(series).Error
}

func (r *AdminRepository) FindSeriesByID(id uuid.UUID) (*domain.Series, error) {
	var series domain.Series
	err := r.db.Preload("Blocks", func(db *gorm.DB) *gorm.DB {
		return db.Order("block_order ASC")
	}).Where("id = ? AND deleted_at IS NULL", id).First(&series).Error
	return &series, err
}

func (r *AdminRepository) UpdateSeries(series *domain.Series) error {
	return r.db.Save(series).Error
}

func (r *AdminRepository) DeleteSeries(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&domain.Series{}).Error
}

func (r *AdminRepository) ListActiveSeries() ([]domain.Series, error) {
	var series []domain.Series
	err := r.db.Preload("Blocks", func(db *gorm.DB) *gorm.DB {
		return db.Order("block_order ASC")
	}).Where("deleted_at IS NULL AND is_active = true").Order("nama ASC").Find(&series).Error
	return series, err
}

func (r *AdminRepository) SeriesNameExists(nama string, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.Model(&domain.Series{}).Where("nama = ? AND deleted_at IS NULL", nama)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *AdminRepository) GetSeriesPortfolioCount(id uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&domain.Portfolio{}).Where("series_id = ? AND deleted_at IS NULL", id).Count(&count).Error
	return count, err
}

// UpdateSeriesBlocks replaces all blocks for a series
func (r *AdminRepository) UpdateSeriesBlocks(seriesID uuid.UUID, blocks []domain.SeriesBlock) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete existing blocks
		if err := tx.Where("series_id = ?", seriesID).Delete(&domain.SeriesBlock{}).Error; err != nil {
			return err
		}
		// Insert new blocks
		if len(blocks) > 0 {
			for i := range blocks {
				blocks[i].SeriesID = seriesID
				blocks[i].BlockOrder = i
			}
			if err := tx.Create(&blocks).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Admin Users
func (r *AdminRepository) ListUsers(search string, role *string, kelasID, jurusanID *uuid.UUID, isActive *bool, page, limit int) ([]domain.User, int64, error) {
	var users []domain.User
	var total int64

	query := r.db.Model(&domain.User{}).Where("deleted_at IS NULL")

	if search != "" {
		query = query.Where("nama ILIKE ? OR username ILIKE ? OR email ILIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	if role != nil {
		query = query.Where("role = ?", *role)
	}
	if kelasID != nil {
		query = query.Where("kelas_id = ?", *kelasID)
	} else if jurusanID != nil {
		query = query.Joins("JOIN kelas ON users.kelas_id = kelas.id").
			Where("kelas.jurusan_id = ?", *jurusanID)
	}
	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	query.Count(&total)

	offset := (page - 1) * limit
	err := query.Preload("Kelas.Jurusan").
		Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&users).Error

	return users, total, err
}

// Admin Portfolios
func (r *AdminRepository) ListPortfolios(search string, status *string, userID, jurusanID *uuid.UUID, page, limit int) ([]domain.Portfolio, int64, error) {
	var portfolios []domain.Portfolio
	var total int64

	query := r.db.Model(&domain.Portfolio{}).Where("portfolios.deleted_at IS NULL")

	if search != "" {
		query = query.Joins("JOIN users ON portfolios.user_id = users.id").
			Where("portfolios.judul ILIKE ? OR users.nama ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if status != nil {
		query = query.Where("portfolios.status = ?", *status)
	}
	if userID != nil {
		query = query.Where("portfolios.user_id = ?", *userID)
	}
	if jurusanID != nil {
		query = query.Joins("JOIN users u ON portfolios.user_id = u.id").
			Joins("JOIN kelas k ON u.kelas_id = k.id").
			Where("k.jurusan_id = ?", *jurusanID)
	}

	query.Count(&total)

	offset := (page - 1) * limit
	err := query.Preload("User.Kelas.Jurusan").
		Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&portfolios).Error

	return portfolios, total, err
}

func (r *AdminRepository) ListPendingPortfolios(search string, jurusanID *uuid.UUID, sort string, page, limit int) ([]domain.Portfolio, int64, error) {
	status := "pending_review"
	return r.ListPortfolios(search, &status, nil, jurusanID, page, limit)
}

// Dashboard Stats
func (r *AdminRepository) GetUserStats() (total, students, alumni, admins, newThisMonth int64, err error) {
	r.db.Model(&domain.User{}).Where("deleted_at IS NULL").Count(&total)
	r.db.Model(&domain.User{}).Where("deleted_at IS NULL AND role = 'student'").Count(&students)
	r.db.Model(&domain.User{}).Where("deleted_at IS NULL AND role = 'alumni'").Count(&alumni)
	r.db.Model(&domain.User{}).Where("deleted_at IS NULL AND role = 'admin'").Count(&admins)

	startOfMonth := time.Now().AddDate(0, 0, -time.Now().Day()+1)
	r.db.Model(&domain.User{}).Where("deleted_at IS NULL AND created_at >= ?", startOfMonth).Count(&newThisMonth)
	return
}

func (r *AdminRepository) GetPortfolioStats() (total, published, pending, draft, rejected, archived, newThisMonth int64, err error) {
	r.db.Model(&domain.Portfolio{}).Where("deleted_at IS NULL").Count(&total)
	r.db.Model(&domain.Portfolio{}).Where("deleted_at IS NULL AND status = 'published'").Count(&published)
	r.db.Model(&domain.Portfolio{}).Where("deleted_at IS NULL AND status = 'pending_review'").Count(&pending)
	r.db.Model(&domain.Portfolio{}).Where("deleted_at IS NULL AND status = 'draft'").Count(&draft)
	r.db.Model(&domain.Portfolio{}).Where("deleted_at IS NULL AND status = 'rejected'").Count(&rejected)
	r.db.Model(&domain.Portfolio{}).Where("deleted_at IS NULL AND status = 'archived'").Count(&archived)

	startOfMonth := time.Now().AddDate(0, 0, -time.Now().Day()+1)
	r.db.Model(&domain.Portfolio{}).Where("deleted_at IS NULL AND created_at >= ?", startOfMonth).Count(&newThisMonth)
	return
}

func (r *AdminRepository) GetJurusanCount() (int64, error) {
	var count int64
	err := r.db.Model(&domain.Jurusan{}).Where("deleted_at IS NULL").Count(&count).Error
	return count, err
}

func (r *AdminRepository) GetKelasStats() (total, activeTahunAjaran int64, err error) {
	r.db.Model(&domain.Kelas{}).Where("deleted_at IS NULL").Count(&total)
	r.db.Model(&domain.Kelas{}).
		Joins("JOIN tahun_ajaran ON kelas.tahun_ajaran_id = tahun_ajaran.id").
		Where("kelas.deleted_at IS NULL AND tahun_ajaran.is_active = true").
		Count(&activeTahunAjaran)
	return
}

// GetRecentUsers returns the most recent users
func (r *AdminRepository) GetRecentUsers(limit int) ([]domain.User, error) {
	var users []domain.User
	err := r.db.Preload("Kelas.Jurusan").
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Limit(limit).
		Find(&users).Error
	return users, err
}

// GetRecentPendingPortfolios returns the most recent portfolios pending review
func (r *AdminRepository) GetRecentPendingPortfolios(limit int) ([]domain.Portfolio, error) {
	var portfolios []domain.Portfolio
	err := r.db.Preload("User.Kelas.Jurusan").
		Where("deleted_at IS NULL AND status = 'pending_review'").
		Order("created_at DESC").
		Limit(limit).
		Find(&portfolios).Error
	return portfolios, err
}

// ============================================================================
// SPECIAL ROLE METHODS
// ============================================================================

// ListSpecialRoles returns all special roles with user count
func (r *AdminRepository) ListSpecialRoles(search string, includeInactive bool) ([]domain.SpecialRole, error) {
	var roles []domain.SpecialRole
	query := r.db.Where("deleted_at IS NULL")

	if search != "" {
		query = query.Where("nama ILIKE ?", "%"+search+"%")
	}
	if !includeInactive {
		query = query.Where("is_active = true")
	}

	err := query.Order("nama ASC").Find(&roles).Error
	return roles, err
}

// CreateSpecialRole creates a new special role
func (r *AdminRepository) CreateSpecialRole(role *domain.SpecialRole) error {
	return r.db.Create(role).Error
}

// FindSpecialRoleByID finds a special role by ID
func (r *AdminRepository) FindSpecialRoleByID(id uuid.UUID) (*domain.SpecialRole, error) {
	var role domain.SpecialRole
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&role).Error
	return &role, err
}

// UpdateSpecialRole updates a special role
func (r *AdminRepository) UpdateSpecialRole(role *domain.SpecialRole) error {
	return r.db.Save(role).Error
}

// DeleteSpecialRole soft deletes a special role
func (r *AdminRepository) DeleteSpecialRole(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&domain.SpecialRole{}).Error
}

// SpecialRoleNameExists checks if a special role name already exists
func (r *AdminRepository) SpecialRoleNameExists(nama string, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.Model(&domain.SpecialRole{}).Where("nama = ? AND deleted_at IS NULL", nama)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

// GetSpecialRoleUserCount returns the number of users assigned to a special role
func (r *AdminRepository) GetSpecialRoleUserCount(roleID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&domain.UserSpecialRole{}).Where("special_role_id = ?", roleID).Count(&count).Error
	return count, err
}

// GetSpecialRoleUsers returns users assigned to a special role
func (r *AdminRepository) GetSpecialRoleUsers(roleID uuid.UUID) ([]domain.UserSpecialRole, error) {
	var userRoles []domain.UserSpecialRole
	err := r.db.Preload("User.Kelas").
		Where("special_role_id = ?", roleID).
		Order("assigned_at DESC").
		Find(&userRoles).Error
	return userRoles, err
}

// AssignUsersToRole assigns multiple users to a special role
func (r *AdminRepository) AssignUsersToRole(roleID uuid.UUID, userIDs []uuid.UUID, assignedBy uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, userID := range userIDs {
			// Check if already assigned
			var count int64
			tx.Model(&domain.UserSpecialRole{}).
				Where("user_id = ? AND special_role_id = ?", userID, roleID).
				Count(&count)
			if count > 0 {
				continue // Skip if already assigned
			}

			userRole := &domain.UserSpecialRole{
				UserID:        userID,
				SpecialRoleID: roleID,
				AssignedBy:    &assignedBy,
				AssignedAt:    time.Now(),
			}
			if err := tx.Create(userRole).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// RemoveUserFromRole removes a user from a special role
func (r *AdminRepository) RemoveUserFromRole(roleID, userID uuid.UUID) error {
	return r.db.Where("special_role_id = ? AND user_id = ?", roleID, userID).
		Delete(&domain.UserSpecialRole{}).Error
}

// GetUserSpecialRoles returns all special roles for a user
func (r *AdminRepository) GetUserSpecialRoles(userID uuid.UUID) ([]domain.SpecialRole, error) {
	var roles []domain.SpecialRole
	err := r.db.Joins("JOIN user_special_roles ON special_roles.id = user_special_roles.special_role_id").
		Where("user_special_roles.user_id = ? AND special_roles.deleted_at IS NULL", userID).
		Find(&roles).Error
	return roles, err
}

// UpdateUserSpecialRoles replaces all special roles for a user
func (r *AdminRepository) UpdateUserSpecialRoles(userID uuid.UUID, roleIDs []uuid.UUID, assignedBy uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Remove all existing roles
		if err := tx.Where("user_id = ?", userID).Delete(&domain.UserSpecialRole{}).Error; err != nil {
			return err
		}

		// Add new roles
		for _, roleID := range roleIDs {
			userRole := &domain.UserSpecialRole{
				UserID:        userID,
				SpecialRoleID: roleID,
				AssignedBy:    &assignedBy,
				AssignedAt:    time.Now(),
			}
			if err := tx.Create(userRole).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetUserCapabilities returns merged capabilities from all user's special roles
func (r *AdminRepository) GetUserCapabilities(userID uuid.UUID) ([]string, error) {
	var roles []domain.SpecialRole
	err := r.db.Joins("JOIN user_special_roles ON special_roles.id = user_special_roles.special_role_id").
		Where("user_special_roles.user_id = ? AND special_roles.deleted_at IS NULL AND special_roles.is_active = true", userID).
		Find(&roles).Error
	if err != nil {
		return nil, err
	}

	// Merge capabilities (unique)
	capMap := make(map[string]bool)
	for _, role := range roles {
		for _, cap := range role.Capabilities {
			capMap[cap] = true
		}
	}

	capabilities := make([]string, 0, len(capMap))
	for cap := range capMap {
		capabilities = append(capabilities, cap)
	}
	return capabilities, nil
}

// HasCapability checks if a user has a specific capability
func (r *AdminRepository) HasCapability(userID uuid.UUID, capability string) (bool, error) {
	capabilities, err := r.GetUserCapabilities(userID)
	if err != nil {
		return false, err
	}
	for _, cap := range capabilities {
		if cap == capability {
			return true, nil
		}
	}
	return false, nil
}

// GetActiveSpecialRoles returns only active special roles (for assignment UI)
func (r *AdminRepository) GetActiveSpecialRoles() ([]domain.SpecialRole, error) {
	var roles []domain.SpecialRole
	err := r.db.Where("deleted_at IS NULL AND is_active = true").Order("nama ASC").Find(&roles).Error
	return roles, err
}

// ============================================================================
// SERIES EXPORT METHODS
// ============================================================================

// GetPortfoliosForExport returns published portfolios for a series with full user data
func (r *AdminRepository) GetPortfoliosForExport(seriesID uuid.UUID, jurusanID, kelasID *uuid.UUID) ([]domain.Portfolio, error) {
	var portfolios []domain.Portfolio

	query := r.db.Model(&domain.Portfolio{}).
		Joins("JOIN users ON portfolios.user_id = users.id").
		Where("portfolios.series_id = ? AND portfolios.status = 'published' AND portfolios.deleted_at IS NULL AND users.deleted_at IS NULL", seriesID)

	// Filter by kelas or jurusan
	if kelasID != nil {
		query = query.Where("users.kelas_id = ?", *kelasID)
	} else if jurusanID != nil {
		query = query.Joins("JOIN kelas ON users.kelas_id = kelas.id").
			Where("kelas.jurusan_id = ?", *jurusanID)
	}

	err := query.
		Preload("User.Kelas.Jurusan").
		Preload("ContentBlocks", func(db *gorm.DB) *gorm.DB {
			return db.Order("block_order ASC")
		}).
		Order("users.nama ASC, portfolios.created_at DESC").
		Find(&portfolios).Error

	return portfolios, err
}

// CountPortfoliosForExport returns count of portfolios for export preview
func (r *AdminRepository) CountPortfoliosForExport(seriesID uuid.UUID, jurusanID, kelasID *uuid.UUID) (portfolioCount int64, userCount int64, err error) {
	query := r.db.Model(&domain.Portfolio{}).
		Joins("JOIN users ON portfolios.user_id = users.id").
		Where("portfolios.series_id = ? AND portfolios.status = 'published' AND portfolios.deleted_at IS NULL AND users.deleted_at IS NULL", seriesID)

	if kelasID != nil {
		query = query.Where("users.kelas_id = ?", *kelasID)
	} else if jurusanID != nil {
		query = query.Joins("JOIN kelas ON users.kelas_id = kelas.id").
			Where("kelas.jurusan_id = ?", *jurusanID)
	}

	query.Count(&portfolioCount)

	// Count unique users
	userQuery := r.db.Model(&domain.Portfolio{}).
		Select("COUNT(DISTINCT portfolios.user_id)").
		Joins("JOIN users ON portfolios.user_id = users.id").
		Where("portfolios.series_id = ? AND portfolios.status = 'published' AND portfolios.deleted_at IS NULL AND users.deleted_at IS NULL", seriesID)

	if kelasID != nil {
		userQuery = userQuery.Where("users.kelas_id = ?", *kelasID)
	} else if jurusanID != nil {
		userQuery = userQuery.Joins("JOIN kelas ON users.kelas_id = kelas.id").
			Where("kelas.jurusan_id = ?", *jurusanID)
	}

	userQuery.Scan(&userCount)

	return portfolioCount, userCount, nil
}

// ============================================================================
// STUDENT IMPORT METHODS
// ============================================================================

// GetActiveTahunAjaran returns the currently active tahun ajaran
func (r *AdminRepository) GetActiveTahunAjaran() (*domain.TahunAjaran, error) {
	var ta domain.TahunAjaran
	err := r.db.Where("is_active = true AND deleted_at IS NULL").First(&ta).Error
	return &ta, err
}

// FindJurusanByKode finds a jurusan by its kode
func (r *AdminRepository) FindJurusanByKode(kode string) (*domain.Jurusan, error) {
	var jurusan domain.Jurusan
	err := r.db.Where("LOWER(kode) = LOWER(?) AND deleted_at IS NULL", kode).First(&jurusan).Error
	return &jurusan, err
}

// FindKelasByTingkatJurusanRombel finds a kelas by tingkat, jurusan, and rombel
func (r *AdminRepository) FindKelasByTingkatJurusanRombel(tahunAjaranID, jurusanID uuid.UUID, tingkat int, rombel string) (*domain.Kelas, error) {
	var kelas domain.Kelas
	err := r.db.Where("tahun_ajaran_id = ? AND jurusan_id = ? AND tingkat = ? AND UPPER(rombel) = UPPER(?) AND deleted_at IS NULL",
		tahunAjaranID, jurusanID, tingkat, rombel).First(&kelas).Error
	return &kelas, err
}

// FindExistingNIS returns list of NIS that already exist in database
func (r *AdminRepository) FindExistingNIS(nisList []string) ([]string, error) {
	var existingNIS []string
	err := r.db.Model(&domain.User{}).
		Where("nis IN ? AND deleted_at IS NULL", nisList).
		Pluck("nis", &existingNIS).Error
	return existingNIS, err
}

// FindExistingUsernames returns list of usernames that already exist in database
func (r *AdminRepository) FindExistingUsernames(usernames []string) ([]string, error) {
	var existing []string
	err := r.db.Model(&domain.User{}).
		Where("username IN ? AND deleted_at IS NULL", usernames).
		Pluck("username", &existing).Error
	return existing, err
}

// BulkCreateStudents creates multiple students in a transaction
func (r *AdminRepository) BulkCreateStudents(students []domain.User) (int, error) {
	created := 0
	err := r.db.Transaction(func(tx *gorm.DB) error {
		for _, student := range students {
			if err := tx.Create(&student).Error; err != nil {
				return err
			}
			created++
		}
		return nil
	})
	return created, err
}

// BulkCreateKelas creates multiple kelas in a transaction
func (r *AdminRepository) BulkCreateKelas(kelasList []domain.Kelas) ([]domain.Kelas, error) {
	var created []domain.Kelas
	err := r.db.Transaction(func(tx *gorm.DB) error {
		for _, kelas := range kelasList {
			if err := tx.Create(&kelas).Error; err != nil {
				return err
			}
			created = append(created, kelas)
		}
		return nil
	})
	return created, err
}
