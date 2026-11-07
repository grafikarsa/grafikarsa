package academic

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

// AcademicYear is one school year.
type AcademicYear struct {
	ID        uuid.UUID
	StartYear int16
	Label     string
	Slug      string
	StartsOn  time.Time
	EndsOn    time.Time
	IsCurrent bool
}

// CreateYearInput is the payload for CreateYear.
type CreateYearInput struct {
	StartYear int16
	StartsOn  time.Time
	EndsOn    time.Time
}

// UpdateYearInput holds the optional dates of UpdateYear.
type UpdateYearInput struct {
	StartsOn *time.Time
	EndsOn   *time.Time
}

// ListYears returns academic years, newest first.
func (s *Service) ListYears(ctx context.Context) ([]AcademicYear, error) {
	rows, err := s.st.ListAcademicYears(ctx)
	if err != nil {
		return nil, fmt.Errorf("academic: list academic years: %w", err)
	}
	years := make([]AcademicYear, 0, len(rows))
	for _, row := range rows {
		years = append(years, newAcademicYear(
			row.ID, row.StartYear, row.StartsOn, row.EndsOn, row.IsCurrent,
		))
	}
	return years, nil
}

// CreateYear inserts a new academic year.
func (s *Service) CreateYear(ctx context.Context, a Actor, in CreateYearInput) (AcademicYear, error) {
	if !a.canGlobally(permissionManageYears) {
		return AcademicYear{}, apperr.NotFound()
	}

	startsOn := dateOnly(in.StartsOn)
	endsOn := dateOnly(in.EndsOn)
	if err := validateYear(in.StartYear, startsOn, endsOn); err != nil {
		return AcademicYear{}, err
	}

	var year AcademicYear
	err := s.st.WithTx(ctx, func(q *store.Queries) error {
		row, err := q.InsertAcademicYear(ctx, store.InsertAcademicYearParams{
			StartYear: in.StartYear,
			StartsOn:  dateFromTime(startsOn),
			EndsOn:    dateFromTime(endsOn),
		})
		if err != nil {
			if isUniqueViolation(err) {
				return apperr.Conflict("Tahun ajaran sudah ada.")
			}
			return fmt.Errorf("academic: insert academic year: %w", err)
		}

		if err := s.recordAudit(ctx, q, a, "academic.year_create", "academic_year", row.ID, nil, map[string]any{
			"start_year": row.StartYear,
			"label":      YearLabel(row.StartYear),
		}); err != nil {
			return err
		}

		year = newAcademicYear(row.ID, row.StartYear, row.StartsOn, row.EndsOn, row.IsCurrent)
		return nil
	})
	if err != nil {
		return AcademicYear{}, err
	}
	return year, nil
}

// UpdateYear changes the start and end dates of an academic year.
func (s *Service) UpdateYear(ctx context.Context, a Actor, id uuid.UUID, in UpdateYearInput) (AcademicYear, error) {
	if !a.canGlobally(permissionManageYears) {
		return AcademicYear{}, apperr.NotFound()
	}

	var year AcademicYear
	err := s.st.WithTx(ctx, func(q *store.Queries) error {
		current, err := q.GetAcademicYearByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return apperr.NotFound()
			}
			return fmt.Errorf("academic: get academic year: %w", err)
		}

		startsOn := timeFromDate(current.StartsOn)
		endsOn := timeFromDate(current.EndsOn)
		if in.StartsOn != nil {
			startsOn = dateOnly(*in.StartsOn)
		}
		if in.EndsOn != nil {
			endsOn = dateOnly(*in.EndsOn)
		}
		if err := validateYear(current.StartYear, startsOn, endsOn); err != nil {
			return err
		}

		row, err := q.UpdateAcademicYearDates(ctx, store.UpdateAcademicYearDatesParams{
			StartsOn: dateFromTime(startsOn),
			EndsOn:   dateFromTime(endsOn),
			ID:       id,
		})
		if err != nil {
			if isNotFound(err) {
				return apperr.NotFound()
			}
			return fmt.Errorf("academic: update academic year: %w", err)
		}

		year = newAcademicYear(row.ID, row.StartYear, row.StartsOn, row.EndsOn, row.IsCurrent)
		return nil
	})
	if err != nil {
		return AcademicYear{}, err
	}
	return year, nil
}

// SetCurrentYear makes id the single current academic year.
func (s *Service) SetCurrentYear(ctx context.Context, a Actor, id uuid.UUID) error {
	if !a.canGlobally(permissionManageYears) {
		return apperr.NotFound()
	}

	return s.st.WithTx(ctx, func(q *store.Queries) error {
		row, err := q.GetAcademicYearByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return apperr.NotFound()
			}
			return fmt.Errorf("academic: get academic year: %w", err)
		}

		if err := q.ClearCurrentAcademicYear(ctx); err != nil {
			return fmt.Errorf("academic: clear current academic year: %w", err)
		}
		if err := q.SetCurrentAcademicYear(ctx, id); err != nil {
			return fmt.Errorf("academic: set current academic year: %w", err)
		}
		if err := s.recordAudit(ctx, q, a, "academic.year_set_current", "academic_year", id, nil, map[string]any{
			"start_year": row.StartYear,
		}); err != nil {
			return err
		}
		return nil
	})
}

func newAcademicYear(
	id uuid.UUID,
	startYear int16,
	startsOn, endsOn pgtype.Date,
	isCurrent bool,
) AcademicYear {
	return AcademicYear{
		ID:        id,
		StartYear: startYear,
		Label:     YearLabel(startYear),
		Slug:      YearSlug(startYear),
		StartsOn:  timeFromDate(startsOn),
		EndsOn:    timeFromDate(endsOn),
		IsCurrent: isCurrent,
	}
}
