package academic

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

// Class is one rombel (class section) in an academic year.
type Class struct {
	ID              uuid.UUID
	AcademicYearID  uuid.UUID
	MajorID         uuid.UUID
	GroupID         uuid.UUID
	MajorCode       string
	Grade           int16
	Section         string
	Name            string
	Slug            string
	EnrollmentCount int64
}

// ClassFilter narrows ListClasses. Zero fields mean "no filter".
type ClassFilter struct {
	AcademicYearID *uuid.UUID
	MajorID        *uuid.UUID
}

// CreateClassInput is the payload for CreateClass.
type CreateClassInput struct {
	AcademicYearID uuid.UUID
	MajorID        uuid.UUID
	Grade          int16
	Section        string
}

// ListClasses returns classes ordered by grade then section.
func (s *Service) ListClasses(ctx context.Context, f ClassFilter) ([]Class, error) {
	rows, err := s.st.ListClasses(ctx, store.ListClassesParams{
		AcademicYearID: f.AcademicYearID,
		MajorID:        f.MajorID,
	})
	if err != nil {
		return nil, fmt.Errorf("academic: list classes: %w", err)
	}
	classes := make([]Class, 0, len(rows))
	for _, row := range rows {
		classes = append(classes, Class{
			ID:              row.ID,
			AcademicYearID:  row.AcademicYearID,
			MajorID:         row.MajorID,
			GroupID:         row.GroupID,
			MajorCode:       row.MajorCode,
			Grade:           row.Grade,
			Section:         row.Section,
			Name:            row.Name,
			Slug:            row.Slug,
			EnrollmentCount: row.EnrollmentCount,
		})
	}
	return classes, nil
}

// CreateClass inserts a class and its mirror group in one transaction.
func (s *Service) CreateClass(ctx context.Context, a Actor, in CreateClassInput) (Class, error) {
	if !a.canForMajor(permissionManageClasses, in.MajorID) {
		return Class{}, apperr.NotFound()
	}

	in.Section = normalizeSection(in.Section)
	if err := validateClass(in.Grade, in.Section); err != nil {
		return Class{}, err
	}

	var class Class
	err := s.st.WithTx(ctx, func(q *store.Queries) error {
		year, err := q.GetAcademicYearByID(ctx, in.AcademicYearID)
		if err != nil {
			if isNotFound(err) {
				return apperr.NotFound()
			}
			return fmt.Errorf("academic: get academic year: %w", err)
		}
		major, err := q.GetMajorByID(ctx, in.MajorID)
		if err != nil {
			if isNotFound(err) {
				return apperr.NotFound()
			}
			return fmt.Errorf("academic: get major: %w", err)
		}

		name := ClassName(in.Grade, major.ShortName, in.Section)
		slug := ClassSlug(in.Grade, major.ShortName, in.Section)

		id, err := q.InsertClass(ctx, store.InsertClassParams{
			AcademicYearID: in.AcademicYearID,
			MajorID:        in.MajorID,
			Grade:          in.Grade,
			Section:        in.Section,
			Name:           name,
			Slug:           slug,
		})
		if err != nil {
			if isUniqueViolation(err) {
				return apperr.Conflict("Kelas ini sudah ada di tahun ajaran tersebut.")
			}
			return fmt.Errorf("academic: insert class: %w", err)
		}

		status := groupStatusActive
		var historySince *time.Time
		if timeFromDate(year.EndsOn).Before(dateOnly(s.now())) {
			status = groupStatusHistory
			now := s.now().UTC()
			historySince = &now
		}
		if _, err := q.InsertClassGroup(ctx, store.InsertClassGroupParams{
			ClassID:      id,
			Status:       status,
			HistorySince: historySince,
		}); err != nil {
			return fmt.Errorf("academic: insert class group: %w", err)
		}

		majorID := in.MajorID
		if err := s.recordAudit(ctx, q, a, "academic.class_create", "class", id, &majorID, map[string]any{
			"name":    name,
			"slug":    slug,
			"grade":   in.Grade,
			"section": in.Section,
		}); err != nil {
			return err
		}

		row, err := q.GetClassByID(ctx, id)
		if err != nil {
			return fmt.Errorf("academic: get class after create: %w", err)
		}
		class = classFromRow(row)
		return nil
	})
	if err != nil {
		return Class{}, err
	}
	return class, nil
}

// UpdateClassSection moves a class to another section and recomputes its name
// and slug.
func (s *Service) UpdateClassSection(ctx context.Context, a Actor, id uuid.UUID, section string) (Class, error) {
	section = normalizeSection(section)

	var class Class
	err := s.st.WithTx(ctx, func(q *store.Queries) error {
		current, err := q.GetClassByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return apperr.NotFound()
			}
			return fmt.Errorf("academic: get class: %w", err)
		}
		if !a.canForMajor(permissionManageClasses, current.MajorID) {
			return apperr.NotFound()
		}
		if err := validateClass(current.Grade, section); err != nil {
			return err
		}

		name := ClassName(current.Grade, current.MajorShortName, section)
		slug := ClassSlug(current.Grade, current.MajorShortName, section)
		if _, err := q.UpdateClassSection(ctx, store.UpdateClassSectionParams{
			Section: section,
			Name:    name,
			Slug:    slug,
			ID:      id,
		}); err != nil {
			if isNotFound(err) {
				return apperr.NotFound()
			}
			if isUniqueViolation(err) {
				return apperr.Conflict("Kelas ini sudah ada di tahun ajaran tersebut.")
			}
			return fmt.Errorf("academic: update class section: %w", err)
		}

		if err := s.recordAudit(ctx, q, a, "academic.class_update", "class", id, &current.MajorID, map[string]any{
			"section": section,
			"name":    name,
			"slug":    slug,
		}); err != nil {
			return err
		}

		row, err := q.GetClassByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return apperr.NotFound()
			}
			return fmt.Errorf("academic: get class after update: %w", err)
		}
		class = classFromRow(row)
		return nil
	})
	if err != nil {
		return Class{}, err
	}
	return class, nil
}

// DeleteClass removes an empty class and its mirror group.
func (s *Service) DeleteClass(ctx context.Context, a Actor, id uuid.UUID) error {
	return s.st.WithTx(ctx, func(q *store.Queries) error {
		current, err := q.GetClassByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return apperr.NotFound()
			}
			return fmt.Errorf("academic: get class: %w", err)
		}
		if !a.canForMajor(permissionManageClasses, current.MajorID) {
			return apperr.NotFound()
		}

		count, err := q.CountClassEnrollments(ctx, id)
		if err != nil {
			return fmt.Errorf("academic: count class enrollments: %w", err)
		}
		if count > 0 {
			return apperr.Conflict("Kelas sudah punya siswa, tidak bisa dihapus.")
		}

		if err := q.DeleteClassGroupByClassID(ctx, id); err != nil {
			return fmt.Errorf("academic: delete class group: %w", err)
		}
		if err := q.DeleteClass(ctx, id); err != nil {
			return fmt.Errorf("academic: delete class: %w", err)
		}

		if err := s.recordAudit(ctx, q, a, "academic.class_delete", "class", id, &current.MajorID, map[string]any{
			"name": current.Name,
			"slug": current.Slug,
		}); err != nil {
			return err
		}
		return nil
	})
}

func classFromRow(row store.GetClassByIDRow) Class {
	return Class{
		ID:              row.ID,
		AcademicYearID:  row.AcademicYearID,
		MajorID:         row.MajorID,
		GroupID:         row.GroupID,
		MajorCode:       row.MajorCode,
		Grade:           row.Grade,
		Section:         row.Section,
		Name:            row.Name,
		Slug:            row.Slug,
		EnrollmentCount: row.EnrollmentCount,
	}
}
