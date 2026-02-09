package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"grafikarsa/internal/domain"
	"grafikarsa/internal/utils"
)

// MajorRepository handles major/jurusan data access
type MajorRepository struct {
	db *pgxpool.Pool
}

// NewMajorRepository creates a new MajorRepository
func NewMajorRepository(db *pgxpool.Pool) *MajorRepository {
	return &MajorRepository{db: db}
}

// GetAll retrieves all majors
func (r *MajorRepository) GetAll(ctx context.Context) ([]domain.Major, error) {
	query := `
		SELECT id, name, code, created_at, updated_at
		FROM majors
		WHERE deleted_at IS NULL
		ORDER BY name ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var majors []domain.Major
	for rows.Next() {
		var m domain.Major
		if err := rows.Scan(&m.ID, &m.Name, &m.Code, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		majors = append(majors, m)
	}

	return majors, rows.Err()
}

// GetByID retrieves a major by ID
func (r *MajorRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Major, error) {
	query := `
		SELECT id, name, code, created_at, updated_at
		FROM majors
		WHERE id = $1 AND deleted_at IS NULL
	`

	var m domain.Major
	err := r.db.QueryRow(ctx, query, id).Scan(&m.ID, &m.Name, &m.Code, &m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &m, nil
}

// Create creates a new major
func (r *MajorRepository) Create(ctx context.Context, major *domain.Major) error {
	query := `
		INSERT INTO majors (id, name, code, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(ctx, query, major.ID, major.Name, major.Code, major.CreatedAt, major.UpdatedAt)
	return err
}

// Update updates a major
func (r *MajorRepository) Update(ctx context.Context, major *domain.Major) error {
	query := `
		UPDATE majors
		SET name = $2, code = $3, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.db.Exec(ctx, query, major.ID, major.Name, major.Code)
	return err
}

// SoftDelete soft deletes a major
func (r *MajorRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE majors SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// CodeExists checks if a major code exists
func (r *MajorRepository) CodeExists(ctx context.Context, code string, excludeID *uuid.UUID) (bool, error) {
	var query string
	var args []interface{}

	if excludeID != nil {
		query = `SELECT EXISTS(SELECT 1 FROM majors WHERE LOWER(code) = LOWER($1) AND id != $2 AND deleted_at IS NULL)`
		args = []interface{}{code, *excludeID}
	} else {
		query = `SELECT EXISTS(SELECT 1 FROM majors WHERE LOWER(code) = LOWER($1) AND deleted_at IS NULL)`
		args = []interface{}{code}
	}

	var exists bool
	err := r.db.QueryRow(ctx, query, args...).Scan(&exists)
	return exists, err
}

// HasClasses checks if a major has associated classes
func (r *MajorRepository) HasClasses(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM classes WHERE major_id = $1 AND deleted_at IS NULL)`
	var exists bool
	err := r.db.QueryRow(ctx, query, id).Scan(&exists)
	return exists, err
}

// AcademicYearRepository handles academic year data access
type AcademicYearRepository struct {
	db *pgxpool.Pool
}

// NewAcademicYearRepository creates a new AcademicYearRepository
func NewAcademicYearRepository(db *pgxpool.Pool) *AcademicYearRepository {
	return &AcademicYearRepository{db: db}
}

// GetAll retrieves all academic years
func (r *AcademicYearRepository) GetAll(ctx context.Context) ([]domain.AcademicYear, error) {
	query := `
		SELECT id, year_start, is_active, promotion_month, promotion_day, created_at, updated_at
		FROM academic_years
		WHERE deleted_at IS NULL
		ORDER BY year_start DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var years []domain.AcademicYear
	for rows.Next() {
		var ay domain.AcademicYear
		if err := rows.Scan(&ay.ID, &ay.YearStart, &ay.IsActive, &ay.PromotionMonth, &ay.PromotionDay, &ay.CreatedAt, &ay.UpdatedAt); err != nil {
			return nil, err
		}
		years = append(years, ay)
	}

	return years, rows.Err()
}

// GetByID retrieves an academic year by ID
func (r *AcademicYearRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.AcademicYear, error) {
	query := `
		SELECT id, year_start, is_active, promotion_month, promotion_day, created_at, updated_at
		FROM academic_years
		WHERE id = $1 AND deleted_at IS NULL
	`

	var ay domain.AcademicYear
	err := r.db.QueryRow(ctx, query, id).Scan(&ay.ID, &ay.YearStart, &ay.IsActive, &ay.PromotionMonth, &ay.PromotionDay, &ay.CreatedAt, &ay.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &ay, nil
}

// GetActive retrieves the active academic year
func (r *AcademicYearRepository) GetActive(ctx context.Context) (*domain.AcademicYear, error) {
	query := `
		SELECT id, year_start, is_active, promotion_month, promotion_day, created_at, updated_at
		FROM academic_years
		WHERE is_active = true AND deleted_at IS NULL
		LIMIT 1
	`

	var ay domain.AcademicYear
	err := r.db.QueryRow(ctx, query).Scan(&ay.ID, &ay.YearStart, &ay.IsActive, &ay.PromotionMonth, &ay.PromotionDay, &ay.CreatedAt, &ay.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &ay, nil
}

// Create creates a new academic year
func (r *AcademicYearRepository) Create(ctx context.Context, ay *domain.AcademicYear) error {
	query := `
		INSERT INTO academic_years (id, year_start, is_active, promotion_month, promotion_day, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(ctx, query, ay.ID, ay.YearStart, ay.IsActive, ay.PromotionMonth, ay.PromotionDay, ay.CreatedAt, ay.UpdatedAt)
	return err
}

// Update updates an academic year
func (r *AcademicYearRepository) Update(ctx context.Context, ay *domain.AcademicYear) error {
	query := `
		UPDATE academic_years
		SET year_start = $2, is_active = $3, promotion_month = $4, promotion_day = $5, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.db.Exec(ctx, query, ay.ID, ay.YearStart, ay.IsActive, ay.PromotionMonth, ay.PromotionDay)
	return err
}

// DeactivateAll deactivates all academic years
func (r *AcademicYearRepository) DeactivateAll(ctx context.Context) error {
	query := `UPDATE academic_years SET is_active = false WHERE is_active = true`
	_, err := r.db.Exec(ctx, query)
	return err
}

// SoftDelete soft deletes an academic year
func (r *AcademicYearRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE academic_years SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// YearExists checks if an academic year exists
func (r *AcademicYearRepository) YearExists(ctx context.Context, yearStart int, excludeID *uuid.UUID) (bool, error) {
	var query string
	var args []interface{}

	if excludeID != nil {
		query = `SELECT EXISTS(SELECT 1 FROM academic_years WHERE year_start = $1 AND id != $2 AND deleted_at IS NULL)`
		args = []interface{}{yearStart, *excludeID}
	} else {
		query = `SELECT EXISTS(SELECT 1 FROM academic_years WHERE year_start = $1 AND deleted_at IS NULL)`
		args = []interface{}{yearStart}
	}

	var exists bool
	err := r.db.QueryRow(ctx, query, args...).Scan(&exists)
	return exists, err
}

// HasClasses checks if an academic year has associated classes
func (r *AcademicYearRepository) HasClasses(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM classes WHERE academic_year_id = $1 AND deleted_at IS NULL)`
	var exists bool
	err := r.db.QueryRow(ctx, query, id).Scan(&exists)
	return exists, err
}

// ClassRepository handles class data access
type ClassRepository struct {
	db *pgxpool.Pool
}

// NewClassRepository creates a new ClassRepository
func NewClassRepository(db *pgxpool.Pool) *ClassRepository {
	return &ClassRepository{db: db}
}

// ClassFilter contains filter options for class listing
type ClassFilter struct {
	AcademicYearID *uuid.UUID
	MajorID        *uuid.UUID
	GradeLevel     string
	Page           int
	Limit          int
}

// List retrieves classes with filtering
func (r *ClassRepository) List(ctx context.Context, filter ClassFilter) ([]domain.ClassWithRelations, int, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, "c.deleted_at IS NULL")

	if filter.AcademicYearID != nil {
		conditions = append(conditions, fmt.Sprintf("c.academic_year_id = $%d", argNum))
		args = append(args, *filter.AcademicYearID)
		argNum++
	}

	if filter.MajorID != nil {
		conditions = append(conditions, fmt.Sprintf("c.major_id = $%d", argNum))
		args = append(args, *filter.MajorID)
		argNum++
	}

	if filter.GradeLevel != "" {
		conditions = append(conditions, fmt.Sprintf("c.grade_level = $%d", argNum))
		args = append(args, filter.GradeLevel)
		argNum++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM classes c WHERE %s`, whereClause)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	pagination := utils.Pagination{Page: filter.Page, Limit: filter.Limit}
	pagination.Validate(100)

	query := fmt.Sprintf(`
		SELECT c.id, c.name, c.grade_level, c.group_letter,
		       ay.id, ay.year_start,
		       m.id, m.name, m.code,
		       (SELECT COUNT(*) FROM users u WHERE u.current_class_id = c.id AND u.deleted_at IS NULL) as student_count,
		       c.created_at
		FROM classes c
		INNER JOIN academic_years ay ON c.academic_year_id = ay.id
		INNER JOIN majors m ON c.major_id = m.id
		WHERE %s
		ORDER BY ay.year_start DESC, c.name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	args = append(args, pagination.Limit, pagination.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var classes []domain.ClassWithRelations
	for rows.Next() {
		var c domain.ClassWithRelations
		if err := rows.Scan(
			&c.ID, &c.Name, &c.GradeLevel, &c.GroupLetter,
			&c.AcademicYear.ID, &c.AcademicYear.YearStart,
			&c.Major.ID, &c.Major.Name, &c.Major.Code,
			&c.StudentCount, &c.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		classes = append(classes, c)
	}

	return classes, total, rows.Err()
}

// GetByID retrieves a class by ID
func (r *ClassRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Class, error) {
	query := `
		SELECT c.id, c.name, c.grade_level, c.group_letter, c.academic_year_id, c.major_id, c.created_at, c.updated_at,
		       ay.id, ay.year_start,
		       m.id, m.name, m.code
		FROM classes c
		INNER JOIN academic_years ay ON c.academic_year_id = ay.id
		INNER JOIN majors m ON c.major_id = m.id
		WHERE c.id = $1 AND c.deleted_at IS NULL
	`

	var c domain.Class
	var ay domain.AcademicYear
	var m domain.Major

	err := r.db.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.Name, &c.GradeLevel, &c.GroupLetter, &c.AcademicYearID, &c.MajorID, &c.CreatedAt, &c.UpdatedAt,
		&ay.ID, &ay.YearStart,
		&m.ID, &m.Name, &m.Code,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	c.AcademicYear = &ay
	c.Major = &m

	return &c, nil
}

// Create creates a new class
func (r *ClassRepository) Create(ctx context.Context, class *domain.Class) error {
	query := `
		INSERT INTO classes (id, name, grade_level, group_letter, academic_year_id, major_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query,
		class.ID, class.Name, class.GradeLevel, class.GroupLetter,
		class.AcademicYearID, class.MajorID, class.CreatedAt, class.UpdatedAt,
	)
	return err
}

// Update updates a class
func (r *ClassRepository) Update(ctx context.Context, class *domain.Class) error {
	query := `
		UPDATE classes
		SET name = $2, grade_level = $3, group_letter = $4, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.db.Exec(ctx, query, class.ID, class.Name, class.GradeLevel, class.GroupLetter)
	return err
}

// SoftDelete soft deletes a class
func (r *ClassRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE classes SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// Exists checks if a class combination exists
func (r *ClassRepository) Exists(ctx context.Context, academicYearID, majorID uuid.UUID, gradeLevel, groupLetter string, excludeID *uuid.UUID) (bool, error) {
	var query string
	var args []interface{}

	if excludeID != nil {
		query = `SELECT EXISTS(SELECT 1 FROM classes WHERE academic_year_id = $1 AND major_id = $2 AND grade_level = $3 AND group_letter = $4 AND id != $5 AND deleted_at IS NULL)`
		args = []interface{}{academicYearID, majorID, gradeLevel, groupLetter, *excludeID}
	} else {
		query = `SELECT EXISTS(SELECT 1 FROM classes WHERE academic_year_id = $1 AND major_id = $2 AND grade_level = $3 AND group_letter = $4 AND deleted_at IS NULL)`
		args = []interface{}{academicYearID, majorID, gradeLevel, groupLetter}
	}

	var exists bool
	err := r.db.QueryRow(ctx, query, args...).Scan(&exists)
	return exists, err
}

// HasStudents checks if a class has students
func (r *ClassRepository) HasStudents(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE current_class_id = $1 AND deleted_at IS NULL)`
	var exists bool
	err := r.db.QueryRow(ctx, query, id).Scan(&exists)
	return exists, err
}

// GenerateClassName generates the class name based on grade, major code, and group
func GenerateClassName(gradeLevel, majorCode, groupLetter string) string {
	gradeMap := map[string]string{
		"10": "X",
		"11": "XI",
		"12": "XII",
	}
	romanGrade := gradeMap[gradeLevel]
	if romanGrade == "" {
		romanGrade = gradeLevel
	}
	return fmt.Sprintf("%s-%s-%s", romanGrade, strings.ToUpper(majorCode), groupLetter)
}

// GetAllForPublic retrieves all classes for public view (active academic year only)
func (r *ClassRepository) GetAllForPublic(ctx context.Context) ([]domain.ClassInfo, error) {
	query := `
		SELECT c.id, c.name
		FROM classes c
		INNER JOIN academic_years ay ON c.academic_year_id = ay.id
		WHERE ay.is_active = true AND c.deleted_at IS NULL
		ORDER BY c.name ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var classes []domain.ClassInfo
	for rows.Next() {
		var c domain.ClassInfo
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		classes = append(classes, c)
	}

	return classes, rows.Err()
}
