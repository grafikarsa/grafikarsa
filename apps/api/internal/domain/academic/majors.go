package academic

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

// Major is a study program (jurusan) with its mirror group.
type Major struct {
	ID        uuid.UUID
	Code      string
	Name      string
	ShortName string
	IsActive  bool
	SortOrder int16
	GroupID   uuid.UUID
}

// CreateMajorInput is the payload for CreateMajor.
type CreateMajorInput struct {
	Code      string
	Name      string
	ShortName string
	SortOrder int16
}

// UpdateMajorInput holds the optional fields of UpdateMajor.
type UpdateMajorInput struct {
	Name      *string
	ShortName *string
	SortOrder *int16
	IsActive  *bool
}

// ListMajors returns all majors ordered by sort order then name.
func (s *Service) ListMajors(ctx context.Context) ([]Major, error) {
	rows, err := s.st.ListMajors(ctx)
	if err != nil {
		return nil, fmt.Errorf("academic: list majors: %w", err)
	}
	majors := make([]Major, 0, len(rows))
	for _, row := range rows {
		majors = append(majors, Major{
			ID:        row.ID,
			Code:      row.Code,
			Name:      row.Name,
			ShortName: row.ShortName,
			IsActive:  row.IsActive,
			SortOrder: row.SortOrder,
			GroupID:   row.GroupID,
		})
	}
	return majors, nil
}

// CreateMajor inserts a major and its mirror group in one transaction.
func (s *Service) CreateMajor(ctx context.Context, a Actor, in CreateMajorInput) (Major, error) {
	if !a.canGlobally(permissionManageMajors) {
		return Major{}, apperr.NotFound()
	}

	in.Code = normalizeCode(in.Code)
	in.Name = normalizeName(in.Name)
	in.ShortName = normalizeShortName(in.ShortName)
	if err := validateCreateMajor(in.Code, in.Name, in.ShortName); err != nil {
		return Major{}, err
	}

	var major Major
	err := s.st.WithTx(ctx, func(q *store.Queries) error {
		id, err := q.InsertMajor(ctx, store.InsertMajorParams{
			Code:      in.Code,
			Name:      in.Name,
			ShortName: in.ShortName,
			SortOrder: in.SortOrder,
		})
		if err != nil {
			if isUniqueViolation(err) {
				return apperr.Conflict("Kode jurusan sudah dipakai.")
			}
			return fmt.Errorf("academic: insert major: %w", err)
		}

		groupID, err := q.InsertMajorGroup(ctx, id)
		if err != nil {
			return fmt.Errorf("academic: insert major group: %w", err)
		}

		if err := s.recordAudit(ctx, q, a, "academic.major_create", "major", id, &id, map[string]any{
			"code":       in.Code,
			"name":       in.Name,
			"short_name": in.ShortName,
			"sort_order": in.SortOrder,
		}); err != nil {
			return err
		}

		major = Major{
			ID:        id,
			Code:      in.Code,
			Name:      in.Name,
			ShortName: in.ShortName,
			IsActive:  true,
			SortOrder: in.SortOrder,
			GroupID:   groupID,
		}
		return nil
	})
	if err != nil {
		return Major{}, err
	}
	return major, nil
}

// UpdateMajor updates a major. Deactivating a major also moves its group to
// history; reactivating it makes the group active again.
func (s *Service) UpdateMajor(ctx context.Context, a Actor, id uuid.UUID, in UpdateMajorInput) (Major, error) {
	if !a.canGlobally(permissionManageMajors) {
		return Major{}, apperr.NotFound()
	}

	var name, shortName *string
	if in.Name != nil {
		value := normalizeName(*in.Name)
		name = &value
	}
	if in.ShortName != nil {
		value := normalizeShortName(*in.ShortName)
		shortName = &value
	}
	if err := validateUpdateMajor(name, shortName); err != nil {
		return Major{}, err
	}

	var major Major
	err := s.st.WithTx(ctx, func(q *store.Queries) error {
		_, err := q.UpdateMajor(ctx, store.UpdateMajorParams{
			Name:      name,
			ShortName: shortName,
			SortOrder: in.SortOrder,
			IsActive:  in.IsActive,
			ID:        id,
		})
		if err != nil {
			if isNotFound(err) {
				return apperr.NotFound()
			}
			return fmt.Errorf("academic: update major: %w", err)
		}

		if in.IsActive != nil {
			status := groupStatusActive
			var historySince *time.Time
			if !*in.IsActive {
				status = groupStatusHistory
				now := s.now().UTC()
				historySince = &now
			}
			if err := q.SetMajorGroupStatus(ctx, store.SetMajorGroupStatusParams{
				Status:       status,
				HistorySince: historySince,
				MajorID:      id,
			}); err != nil {
				return fmt.Errorf("academic: update major group status: %w", err)
			}
		}

		row, err := q.GetMajorByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return apperr.NotFound()
			}
			return fmt.Errorf("academic: get major after update: %w", err)
		}

		data := map[string]any{}
		if name != nil {
			data["name"] = *name
		}
		if shortName != nil {
			data["short_name"] = *shortName
		}
		if in.SortOrder != nil {
			data["sort_order"] = *in.SortOrder
		}
		if in.IsActive != nil {
			data["is_active"] = *in.IsActive
		}
		if err := s.recordAudit(ctx, q, a, "academic.major_update", "major", id, &id, data); err != nil {
			return err
		}

		major = Major{
			ID:        row.ID,
			Code:      row.Code,
			Name:      row.Name,
			ShortName: row.ShortName,
			IsActive:  row.IsActive,
			SortOrder: row.SortOrder,
			GroupID:   row.GroupID,
		}
		return nil
	})
	if err != nil {
		return Major{}, err
	}
	return major, nil
}
