package repository

import (
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *domain.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) FindByID(id uuid.UUID) (*domain.User, error) {
	var user domain.User
	err := r.db.Preload("Kelas.Jurusan").Preload("SocialLinks").
		Where("id = ? AND deleted_at IS NULL", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByUsername(username string) (*domain.User, error) {
	var user domain.User
	err := r.db.Preload("Kelas.Jurusan").Preload("SocialLinks").
		Where("username = ? AND deleted_at IS NULL", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByEmail(email string) (*domain.User, error) {
	var user domain.User
	err := r.db.Where("email = ? AND deleted_at IS NULL", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByUsernameOrEmail(identifier string) (*domain.User, error) {
	var user domain.User
	err := r.db.Where("(username = ? OR email = ?) AND deleted_at IS NULL", identifier, identifier).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(user *domain.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&domain.User{}).Error
}

func (r *UserRepository) UsernameExists(username string, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.Model(&domain.User{}).Where("username = ? AND deleted_at IS NULL", username)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *UserRepository) EmailExists(email string, excludeID *uuid.UUID) (bool, error) {
	var count int64
	query := r.db.Model(&domain.User{}).Where("email = ? AND deleted_at IS NULL", email)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *UserRepository) List(search string, jurusanID, kelasID *uuid.UUID, role *string, page, limit int) ([]domain.User, int64, error) {
	var users []domain.User
	var total int64

	// Base query for counting - exclude admin role
	countQuery := r.db.Model(&domain.User{}).Where("users.deleted_at IS NULL AND users.role != 'admin'")

	// Base query for fetching - exclude admin role
	fetchQuery := r.db.Model(&domain.User{}).Where("users.deleted_at IS NULL AND users.role != 'admin'")

	if search != "" {
		searchCondition := "users.nama ILIKE ? OR users.username ILIKE ? OR users.bio ILIKE ?"
		countQuery = countQuery.Where(searchCondition, "%"+search+"%", "%"+search+"%", "%"+search+"%")
		fetchQuery = fetchQuery.Where(searchCondition, "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	if role != nil && *role != "admin" {
		countQuery = countQuery.Where("users.role = ?", *role)
		fetchQuery = fetchQuery.Where("users.role = ?", *role)
	}

	if kelasID != nil {
		countQuery = countQuery.Where("users.kelas_id = ?", *kelasID)
		fetchQuery = fetchQuery.Where("users.kelas_id = ?", *kelasID)
	} else if jurusanID != nil {
		// Use subquery to find users whose kelas belongs to the jurusan
		countQuery = countQuery.Where("users.kelas_id IN (SELECT id FROM kelas WHERE jurusan_id = ? AND deleted_at IS NULL)", *jurusanID)
		fetchQuery = fetchQuery.Where("users.kelas_id IN (SELECT id FROM kelas WHERE jurusan_id = ? AND deleted_at IS NULL)", *jurusanID)
	}

	// Count total
	countQuery.Count(&total)

	// Fetch with pagination
	offset := (page - 1) * limit
	err := fetchQuery.Preload("Kelas.Jurusan").
		Offset(offset).Limit(limit).
		Order("users.nama ASC").
		Find(&users).Error

	return users, total, err
}

func (r *UserRepository) GetFollowerCount(userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&domain.Follow{}).Where("following_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *UserRepository) GetFollowingCount(userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&domain.Follow{}).Where("follower_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *UserRepository) GetPublishedPortfolioCount(userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&domain.Portfolio{}).
		Where("user_id = ? AND status = 'published' AND deleted_at IS NULL", userID).
		Count(&count).Error
	return count, err
}

func (r *UserRepository) IsFollowing(followerID, followingID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&domain.Follow{}).
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Count(&count).Error
	return count > 0, err
}

func (r *UserRepository) GetClassHistory(userID uuid.UUID) ([]domain.StudentClassHistory, error) {
	var history []domain.StudentClassHistory
	err := r.db.Preload("Kelas").Preload("TahunAjaran").
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&history).Error
	return history, err
}

func (r *UserRepository) UpdateSocialLinks(userID uuid.UUID, links []domain.UserSocialLink) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete existing links
		if err := tx.Where("user_id = ?", userID).Delete(&domain.UserSocialLink{}).Error; err != nil {
			return err
		}
		// Insert new links
		if len(links) > 0 {
			for i := range links {
				links[i].UserID = userID
			}
			if err := tx.Create(&links).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetUserSpecialRoles returns active special roles for a user (for public profile)
func (r *UserRepository) GetUserSpecialRoles(userID uuid.UUID) ([]domain.SpecialRole, error) {
	var roles []domain.SpecialRole
	err := r.db.Joins("JOIN user_special_roles ON special_roles.id = user_special_roles.special_role_id").
		Where("user_special_roles.user_id = ? AND special_roles.deleted_at IS NULL AND special_roles.is_active = true", userID).
		Find(&roles).Error
	return roles, err
}

// TopStudentResult represents a top student with calculated score
type TopStudentResult struct {
	ID                 uuid.UUID `json:"id"`
	Username           string    `json:"username"`
	Nama               string    `json:"nama"`
	AvatarURL          *string   `json:"avatar_url"`
	BannerURL          *string   `json:"banner_url"`
	KelasNama          *string   `json:"kelas_nama"`
	JurusanNama        *string   `json:"jurusan_nama"`
	PortfolioCount     int       `json:"portfolio_count"`
	TotalLikes         int       `json:"total_likes"`
	AvgAssessmentScore float64   `json:"avg_assessment_score"`
	FollowerCount      int       `json:"follower_count"`
	Score              float64   `json:"score"`
}

// GetTopStudents returns top students based on portfolio count, likes, assessment scores, and followers
func (r *UserRepository) GetTopStudents(limit int) ([]TopStudentResult, error) {
	var results []TopStudentResult

	query := `
		WITH student_stats AS (
			SELECT 
				u.id,
				u.username,
				u.nama,
				u.avatar_url,
				u.banner_url,
				k.nama as kelas_nama,
				j.nama as jurusan_nama,
				COUNT(DISTINCT p.id) as portfolio_count,
				COALESCE((
					SELECT COUNT(*) FROM portfolio_likes pl 
					JOIN portfolios pp ON pl.portfolio_id = pp.id 
					WHERE pp.user_id = u.id AND pp.status = 'published' AND pp.deleted_at IS NULL
				), 0) as total_likes,
				COALESCE((
					SELECT AVG(pa.total_score) FROM portfolio_assessments pa 
					JOIN portfolios pp ON pa.portfolio_id = pp.id 
					WHERE pp.user_id = u.id AND pp.status = 'published' AND pp.deleted_at IS NULL AND pa.total_score IS NOT NULL
				), 0) as avg_assessment_score,
				(SELECT COUNT(*) FROM follows f WHERE f.following_id = u.id) as follower_count
			FROM users u
			LEFT JOIN kelas k ON u.kelas_id = k.id AND k.deleted_at IS NULL
			LEFT JOIN jurusan j ON k.jurusan_id = j.id AND j.deleted_at IS NULL
			LEFT JOIN portfolios p ON p.user_id = u.id AND p.status = 'published' AND p.deleted_at IS NULL
			WHERE u.role = 'student'
				AND u.is_active = true
				AND u.deleted_at IS NULL
			GROUP BY u.id, u.username, u.nama, u.avatar_url, u.banner_url, k.nama, j.nama
			HAVING COUNT(DISTINCT p.id) > 0
		),
		max_values AS (
			SELECT 
				GREATEST(MAX(portfolio_count), 1) as max_portfolio,
				GREATEST(MAX(total_likes), 1) as max_likes,
				GREATEST(MAX(follower_count), 1) as max_followers
			FROM student_stats
		)
		SELECT 
			s.id,
			s.username,
			s.nama,
			s.avatar_url,
			s.banner_url,
			s.kelas_nama,
			s.jurusan_nama,
			s.portfolio_count,
			s.total_likes,
			s.avg_assessment_score,
			s.follower_count,
			(
				(s.portfolio_count::float / m.max_portfolio) * 0.30 +
				(s.total_likes::float / m.max_likes) * 0.25 +
				(s.avg_assessment_score / 10) * 0.30 +
				(s.follower_count::float / m.max_followers) * 0.15
			) * 100 as score
		FROM student_stats s, max_values m
		ORDER BY score DESC
		LIMIT ?
	`

	err := r.db.Raw(query, limit).Scan(&results).Error
	return results, err
}

// TopProjectResult represents a top project with calculated score
type TopProjectResult struct {
	ID              uuid.UUID `json:"id"`
	Judul           string    `json:"judul"`
	Slug            string    `json:"slug"`
	ThumbnailURL    *string   `json:"thumbnail_url"`
	PublishedAt     *string   `json:"published_at"`
	UserID          uuid.UUID `json:"user_id"`
	Username        string    `json:"username"`
	UserNama        string    `json:"user_nama"`
	UserAvatar      *string   `json:"user_avatar"`
	AssessmentScore float64   `json:"assessment_score"`
	LikeCount       int       `json:"like_count"`
	Score           float64   `json:"score"`
}

// GetTopProjects returns top projects based on assessment scores, likes, and recency
func (r *UserRepository) GetTopProjects(limit int) ([]TopProjectResult, error) {
	var results []TopProjectResult

	query := `
		WITH project_stats AS (
			SELECT 
				p.id,
				p.judul,
				p.slug,
				p.thumbnail_url,
				p.published_at,
				u.id as user_id,
				u.username,
				u.nama as user_nama,
				u.avatar_url as user_avatar,
				COALESCE(pa.total_score, 0) as assessment_score,
				(SELECT COUNT(*) FROM portfolio_likes pl WHERE pl.portfolio_id = p.id) as like_count,
				CASE
					WHEN p.published_at > NOW() - INTERVAL '7 days' THEN 1.0
					WHEN p.published_at > NOW() - INTERVAL '30 days' THEN 0.8
					WHEN p.published_at > NOW() - INTERVAL '90 days' THEN 0.5
					ELSE 0.2
				END as recency_factor
			FROM portfolios p
			JOIN users u ON p.user_id = u.id
			LEFT JOIN portfolio_assessments pa ON pa.portfolio_id = p.id
			WHERE p.status = 'published'
				AND p.deleted_at IS NULL
				AND u.role = 'student'
				AND u.is_active = true
				AND u.deleted_at IS NULL
		),
		max_values AS (
			SELECT GREATEST(MAX(like_count), 1) as max_likes FROM project_stats
		)
		SELECT 
			ps.id,
			ps.judul,
			ps.slug,
			ps.thumbnail_url,
			ps.published_at,
			ps.user_id,
			ps.username,
			ps.user_nama,
			ps.user_avatar,
			ps.assessment_score,
			ps.like_count,
			(
				(ps.assessment_score / 10) * 0.50 +
				(ps.like_count::float / m.max_likes) * 0.30 +
				ps.recency_factor * 0.20
			) * 100 as score
		FROM project_stats ps, max_values m
		ORDER BY score DESC
		LIMIT ?
	`

	err := r.db.Raw(query, limit).Scan(&results).Error
	return results, err
}
