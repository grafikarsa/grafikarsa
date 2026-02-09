package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"grafikarsa/internal/domain"
	"grafikarsa/internal/repository"
	"grafikarsa/internal/utils"
)

// AdminService handles admin-specific business logic
type AdminService struct {
	majorRepo *repository.MajorRepository
	yearRepo  *repository.AcademicYearRepository
	classRepo *repository.ClassRepository
	userRepo  *repository.UserRepository
}

// NewAdminService creates a new AdminService
func NewAdminService(
	majorRepo *repository.MajorRepository,
	yearRepo *repository.AcademicYearRepository,
	classRepo *repository.ClassRepository,
	userRepo *repository.UserRepository,
) *AdminService {
	return &AdminService{
		majorRepo: majorRepo,
		yearRepo:  yearRepo,
		classRepo: classRepo,
		userRepo:  userRepo,
	}
}

// ==================== MAJOR OPERATIONS ====================

// ListMajors lists all majors
func (s *AdminService) ListMajors(ctx context.Context) ([]domain.Major, error) {
	majors, err := s.majorRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list majors: %w", err)
	}
	return majors, nil
}

// GetMajor gets a major by ID
func (s *AdminService) GetMajor(ctx context.Context, id uuid.UUID) (*domain.Major, error) {
	major, err := s.majorRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if major == nil {
		return nil, ErrMajorNotFound
	}
	return major, nil
}

// CreateMajor creates a new major
func (s *AdminService) CreateMajor(ctx context.Context, name, code string) (*domain.Major, error) {
	// Check if code exists
	exists, _ := s.majorRepo.CodeExists(ctx, code, nil)
	if exists {
		return nil, ErrMajorCodeExists
	}

	now := time.Now()
	major := &domain.Major{
		ID:        uuid.New(),
		Name:      name,
		Code:      code,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.majorRepo.Create(ctx, major); err != nil {
		return nil, fmt.Errorf("failed to create major: %w", err)
	}

	return major, nil
}

// UpdateMajor updates a major
func (s *AdminService) UpdateMajor(ctx context.Context, id uuid.UUID, name, code string) (*domain.Major, error) {
	major, err := s.majorRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if major == nil {
		return nil, ErrMajorNotFound
	}

	// Check if code exists (excluding current)
	exists, _ := s.majorRepo.CodeExists(ctx, code, &id)
	if exists {
		return nil, ErrMajorCodeExists
	}

	major.Name = name
	major.Code = code

	if err := s.majorRepo.Update(ctx, major); err != nil {
		return nil, fmt.Errorf("failed to update major: %w", err)
	}

	return major, nil
}

// DeleteMajor deletes a major
func (s *AdminService) DeleteMajor(ctx context.Context, id uuid.UUID) error {
	major, err := s.majorRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if major == nil {
		return ErrMajorNotFound
	}

	// Check if has classes
	hasClasses, _ := s.majorRepo.HasClasses(ctx, id)
	if hasClasses {
		return ErrMajorHasClasses
	}

	if err := s.majorRepo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete major: %w", err)
	}

	return nil
}

// ==================== ACADEMIC YEAR OPERATIONS ====================

// ListAcademicYears lists all academic years
func (s *AdminService) ListAcademicYears(ctx context.Context) ([]domain.AcademicYear, error) {
	years, err := s.yearRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list academic years: %w", err)
	}
	return years, nil
}

// GetAcademicYear gets an academic year by ID
func (s *AdminService) GetAcademicYear(ctx context.Context, id uuid.UUID) (*domain.AcademicYear, error) {
	year, err := s.yearRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if year == nil {
		return nil, ErrAcademicYearNotFound
	}
	return year, nil
}

// GetActiveAcademicYear gets the active academic year
func (s *AdminService) GetActiveAcademicYear(ctx context.Context) (*domain.AcademicYear, error) {
	year, err := s.yearRepo.GetActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	return year, nil
}

// CreateAcademicYear creates a new academic year
func (s *AdminService) CreateAcademicYear(ctx context.Context, yearStart int, isActive bool, promotionMonth, promotionDay int) (*domain.AcademicYear, error) {
	// Check if year exists
	exists, _ := s.yearRepo.YearExists(ctx, yearStart, nil)
	if exists {
		return nil, ErrAcademicYearExists
	}

	// If setting as active, deactivate others
	if isActive {
		_ = s.yearRepo.DeactivateAll(ctx)
	}

	now := time.Now()
	year := &domain.AcademicYear{
		ID:             uuid.New(),
		YearStart:      yearStart,
		IsActive:       isActive,
		PromotionMonth: promotionMonth,
		PromotionDay:   promotionDay,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.yearRepo.Create(ctx, year); err != nil {
		return nil, fmt.Errorf("failed to create academic year: %w", err)
	}

	return year, nil
}

// UpdateAcademicYear updates an academic year
func (s *AdminService) UpdateAcademicYear(ctx context.Context, id uuid.UUID, yearStart int, isActive bool, promotionMonth, promotionDay int) (*domain.AcademicYear, error) {
	year, err := s.yearRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if year == nil {
		return nil, ErrAcademicYearNotFound
	}

	// Check if year exists (excluding current)
	exists, _ := s.yearRepo.YearExists(ctx, yearStart, &id)
	if exists {
		return nil, ErrAcademicYearExists
	}

	// If setting as active, deactivate others
	if isActive && !year.IsActive {
		_ = s.yearRepo.DeactivateAll(ctx)
	}

	year.YearStart = yearStart
	year.IsActive = isActive
	year.PromotionMonth = promotionMonth
	year.PromotionDay = promotionDay

	if err := s.yearRepo.Update(ctx, year); err != nil {
		return nil, fmt.Errorf("failed to update academic year: %w", err)
	}

	return year, nil
}

// DeleteAcademicYear deletes an academic year
func (s *AdminService) DeleteAcademicYear(ctx context.Context, id uuid.UUID) error {
	year, err := s.yearRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if year == nil {
		return ErrAcademicYearNotFound
	}

	// Check if has classes
	hasClasses, _ := s.yearRepo.HasClasses(ctx, id)
	if hasClasses {
		return ErrAcademicYearHasClasses
	}

	if err := s.yearRepo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete academic year: %w", err)
	}

	return nil
}

// ==================== CLASS OPERATIONS ====================

// ListClasses lists classes with filtering
func (s *AdminService) ListClasses(ctx context.Context, filter repository.ClassFilter) ([]domain.ClassWithRelations, *utils.Meta, error) {
	classes, total, err := s.classRepo.List(ctx, filter)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list classes: %w", err)
	}

	meta := utils.NewMeta(filter.Page, filter.Limit, total)
	return classes, meta, nil
}

// ListPublicClasses lists classes for public view (active year only)
func (s *AdminService) ListPublicClasses(ctx context.Context) ([]domain.ClassInfo, error) {
	classes, err := s.classRepo.GetAllForPublic(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list classes: %w", err)
	}
	return classes, nil
}

// GetClass gets a class by ID
func (s *AdminService) GetClass(ctx context.Context, id uuid.UUID) (*domain.Class, error) {
	class, err := s.classRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if class == nil {
		return nil, ErrClassNotFound
	}
	return class, nil
}

// CreateClass creates a new class
func (s *AdminService) CreateClass(ctx context.Context, academicYearID, majorID uuid.UUID, gradeLevel, groupLetter string) (*domain.Class, error) {
	// Check academic year exists
	year, _ := s.yearRepo.GetByID(ctx, academicYearID)
	if year == nil {
		return nil, ErrAcademicYearNotFound
	}

	// Check major exists
	major, _ := s.majorRepo.GetByID(ctx, majorID)
	if major == nil {
		return nil, ErrMajorNotFound
	}

	// Check if class combination exists
	exists, _ := s.classRepo.Exists(ctx, academicYearID, majorID, gradeLevel, groupLetter, nil)
	if exists {
		return nil, ErrClassExists
	}

	// Generate class name
	className := repository.GenerateClassName(gradeLevel, major.Code, groupLetter)

	now := time.Now()
	class := &domain.Class{
		ID:             uuid.New(),
		Name:           className,
		GradeLevel:     gradeLevel,
		GroupLetter:    groupLetter,
		AcademicYearID: academicYearID,
		MajorID:        majorID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.classRepo.Create(ctx, class); err != nil {
		return nil, fmt.Errorf("failed to create class: %w", err)
	}

	// Set related data
	class.AcademicYear = year
	class.Major = major

	return class, nil
}

// UpdateClass updates a class
func (s *AdminService) UpdateClass(ctx context.Context, id uuid.UUID, gradeLevel, groupLetter string) (*domain.Class, error) {
	class, err := s.classRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if class == nil {
		return nil, ErrClassNotFound
	}

	// Check if class combination exists (excluding current)
	exists, _ := s.classRepo.Exists(ctx, class.AcademicYearID, class.MajorID, gradeLevel, groupLetter, &id)
	if exists {
		return nil, ErrClassExists
	}

	// Generate new class name
	class.GradeLevel = gradeLevel
	class.GroupLetter = groupLetter
	class.Name = repository.GenerateClassName(gradeLevel, class.Major.Code, groupLetter)

	if err := s.classRepo.Update(ctx, class); err != nil {
		return nil, fmt.Errorf("failed to update class: %w", err)
	}

	return class, nil
}

// DeleteClass deletes a class
func (s *AdminService) DeleteClass(ctx context.Context, id uuid.UUID) error {
	class, err := s.classRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("database error: %w", err)
	}
	if class == nil {
		return ErrClassNotFound
	}

	// Check if has students
	hasStudents, _ := s.classRepo.HasStudents(ctx, id)
	if hasStudents {
		return ErrClassHasStudents
	}

	if err := s.classRepo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete class: %w", err)
	}

	return nil
}

// Admin service errors
var (
	ErrMajorNotFound          = fmt.Errorf("major not found")
	ErrMajorCodeExists        = fmt.Errorf("major code already exists")
	ErrMajorHasClasses        = fmt.Errorf("major has classes")
	ErrAcademicYearNotFound   = fmt.Errorf("academic year not found")
	ErrAcademicYearExists     = fmt.Errorf("academic year already exists")
	ErrAcademicYearHasClasses = fmt.Errorf("academic year has classes")
	ErrClassNotFound          = fmt.Errorf("class not found")
	ErrClassExists            = fmt.Errorf("class already exists")
	ErrClassHasStudents       = fmt.Errorf("class has students")
)
