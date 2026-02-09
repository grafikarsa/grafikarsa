package domain

import (
	"time"

	"github.com/google/uuid"
)

// Major represents a school department/jurusan
type Major struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Code      string     `json:"code"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"-"`
}

// MajorInfo is a lightweight reference for embedding
type MajorInfo struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Code string    `json:"code,omitempty"`
}

// ToInfo converts Major to MajorInfo
func (m *Major) ToInfo() MajorInfo {
	return MajorInfo{
		ID:   m.ID,
		Name: m.Name,
		Code: m.Code,
	}
}

// AcademicYear represents a school year
type AcademicYear struct {
	ID             uuid.UUID  `json:"id"`
	YearStart      int        `json:"year_start"`
	IsActive       bool       `json:"is_active"`
	PromotionMonth int        `json:"promotion_month"`
	PromotionDay   int        `json:"promotion_day"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"-"`
}

// AcademicYearInfo is a lightweight reference
type AcademicYearInfo struct {
	ID        uuid.UUID `json:"id"`
	YearStart int       `json:"year_start"`
}

// ToInfo converts AcademicYear to AcademicYearInfo
func (ay *AcademicYear) ToInfo() AcademicYearInfo {
	return AcademicYearInfo{
		ID:        ay.ID,
		YearStart: ay.YearStart,
	}
}

// Class represents a school class
type Class struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	GradeLevel     string     `json:"grade_level"`
	GroupLetter    string     `json:"group_letter"`
	AcademicYearID uuid.UUID  `json:"academic_year_id"`
	MajorID        uuid.UUID  `json:"major_id"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"-"`

	// Related data
	AcademicYear *AcademicYear `json:"academic_year,omitempty"`
	Major        *Major        `json:"major,omitempty"`
	StudentCount int           `json:"student_count,omitempty"`
}

// ClassInfo is a lightweight reference
type ClassInfo struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// ToInfo converts Class to ClassInfo
func (c *Class) ToInfo() ClassInfo {
	return ClassInfo{
		ID:   c.ID,
		Name: c.Name,
	}
}

// ClassWithRelations includes related data for admin views
type ClassWithRelations struct {
	ID           uuid.UUID        `json:"id"`
	Name         string           `json:"name"`
	GradeLevel   string           `json:"grade_level"`
	GroupLetter  string           `json:"group_letter"`
	AcademicYear AcademicYearInfo `json:"academic_year"`
	Major        MajorInfo        `json:"major"`
	StudentCount int              `json:"student_count"`
	CreatedAt    time.Time        `json:"created_at"`
}

// UserClassHistory represents the assignment of a user to a class
type UserClassHistory struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	ClassID        uuid.UUID `json:"class_id"`
	AcademicYearID uuid.UUID `json:"academic_year_id"`
	AssignedAt     time.Time `json:"assigned_at"`

	// Related
	ClassName    string `json:"class_name,omitempty"`
	AcademicYear int    `json:"academic_year,omitempty"`
}
