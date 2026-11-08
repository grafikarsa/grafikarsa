package profiles

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

var monthPattern = regexp.MustCompile(`^[0-9]{4}-(0[1-9]|1[0-2])$`)

var experienceKinds = map[string]struct{}{
	"education": {}, "internship": {}, "job": {}, "freelance": {}, "organization": {},
	"committee": {}, "competition": {}, "certification": {}, "volunteer": {},
}

var employmentTypes = map[string]struct{}{
	"full_time": {}, "part_time": {}, "contract": {}, "internship": {}, "freelance": {},
}

type validExperience struct {
	Kind           string
	Title          string
	Organization   string
	Location       *string
	EmploymentType *string
	Start          pgtype.Date
	End            pgtype.Date
	IsCurrent      bool
	Description    *string
	CredentialURL  *string
	Skills         []string
}

// ListExperiences returns the public experience list of username.
func (s *Service) ListExperiences(ctx context.Context, username string) ([]Experience, error) {
	target, err := s.resolveVisible(ctx, username)
	if err != nil {
		return nil, err
	}
	return s.listExperiences(ctx, s.st.Queries, target.ID)
}

// ListMyExperiences returns the authenticated user's experience list.
func (s *Service) ListMyExperiences(ctx context.Context, p *auth.Principal) ([]Experience, error) {
	if p == nil {
		return nil, apperr.Unauthenticated()
	}
	return s.listExperiences(ctx, s.st.Queries, p.UserID)
}

func (s *Service) listExperiences(ctx context.Context, q *store.Queries, userID uuid.UUID) ([]Experience, error) {
	rows, err := q.ListExperiencesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("profiles: list experiences: %w", err)
	}
	if len(rows) == 0 {
		return []Experience{}, nil
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	skills, err := q.ListExperienceSkills(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("profiles: list experience skills: %w", err)
	}
	grouped := groupExperienceSkills(skills)

	items := make([]Experience, 0, len(rows))
	for _, row := range rows {
		items = append(items, Experience{
			ID:             row.ID,
			Kind:           row.Kind,
			Title:          row.Title,
			Organization:   row.Organization,
			Location:       row.Location,
			EmploymentType: row.EmploymentType,
			StartMonth:     formatMonth(row.StartMonth),
			EndMonth:       formatOptionalMonth(row.EndMonth),
			IsCurrent:      row.IsCurrent,
			Description:    row.Description,
			CredentialURL:  row.CredentialUrl,
			IsSystem:       row.IsSystem,
			Skills:         grouped[row.ID],
		})
	}
	return items, nil
}

// CreateExperience adds one self-declared experience.
func (s *Service) CreateExperience(ctx context.Context, p *auth.Principal, in ExperienceInput) (*Experience, error) {
	if p == nil {
		return nil, apperr.Unauthenticated()
	}
	valid, err := validateExperience(in)
	if err != nil {
		return nil, err
	}

	var created uuid.UUID
	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		slugs, err := validateExperienceSkills(ctx, q, valid.Skills)
		if err != nil {
			return err
		}
		id, err := q.InsertExperience(ctx, store.InsertExperienceParams{
			UserID:         p.UserID,
			Kind:           valid.Kind,
			Title:          valid.Title,
			Organization:   valid.Organization,
			Location:       valid.Location,
			EmploymentType: valid.EmploymentType,
			StartMonth:     valid.Start,
			EndMonth:       valid.End,
			IsCurrent:      valid.IsCurrent,
			Description:    valid.Description,
			CredentialUrl:  valid.CredentialURL,
		})
		if err != nil {
			return fmt.Errorf("profiles: insert experience: %w", err)
		}
		created = id
		if err := insertExperienceSkills(ctx, q, id, slugs); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.experienceByID(ctx, p.UserID, created)
}

// UpdateExperience replaces one of the authenticated user's experiences.
func (s *Service) UpdateExperience(ctx context.Context, p *auth.Principal, id uuid.UUID, in ExperienceInput) (*Experience, error) {
	if p == nil {
		return nil, apperr.Unauthenticated()
	}
	valid, err := validateExperience(in)
	if err != nil {
		return nil, err
	}

	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		existing, err := q.GetExperienceByOwner(ctx, store.GetExperienceByOwnerParams{
			ID:     id,
			UserID: p.UserID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.NotFound()
			}
			return fmt.Errorf("profiles: load experience: %w", err)
		}
		if existing.IsSystem {
			return apperr.Conflict("Entri ini dikelola sekolah.")
		}

		slugs, err := validateExperienceSkills(ctx, q, valid.Skills)
		if err != nil {
			return err
		}
		if err := q.UpdateExperience(ctx, store.UpdateExperienceParams{
			Kind:           valid.Kind,
			Title:          valid.Title,
			Organization:   valid.Organization,
			Location:       valid.Location,
			EmploymentType: valid.EmploymentType,
			StartMonth:     valid.Start,
			EndMonth:       valid.End,
			IsCurrent:      valid.IsCurrent,
			Description:    valid.Description,
			CredentialUrl:  valid.CredentialURL,
			ID:             id,
			UserID:         p.UserID,
		}); err != nil {
			return fmt.Errorf("profiles: update experience: %w", err)
		}
		if err := q.DeleteExperienceSkills(ctx, id); err != nil {
			return fmt.Errorf("profiles: delete experience skills: %w", err)
		}
		return insertExperienceSkills(ctx, q, id, slugs)
	})
	if err != nil {
		return nil, err
	}

	return s.experienceByID(ctx, p.UserID, id)
}

// DeleteExperience removes one of the authenticated user's experiences.
func (s *Service) DeleteExperience(ctx context.Context, p *auth.Principal, id uuid.UUID) error {
	if p == nil {
		return apperr.Unauthenticated()
	}

	return s.st.WithTx(ctx, func(q *store.Queries) error {
		existing, err := q.GetExperienceByOwner(ctx, store.GetExperienceByOwnerParams{
			ID:     id,
			UserID: p.UserID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.NotFound()
			}
			return fmt.Errorf("profiles: load experience: %w", err)
		}
		if existing.IsSystem {
			return apperr.Conflict("Entri ini dikelola sekolah.")
		}
		if err := q.DeleteExperience(ctx, store.DeleteExperienceParams{
			ID:     id,
			UserID: p.UserID,
		}); err != nil {
			return fmt.Errorf("profiles: delete experience: %w", err)
		}
		return nil
	})
}

func (s *Service) experienceByID(ctx context.Context, userID, id uuid.UUID) (*Experience, error) {
	row, err := s.st.GetExperienceByOwner(ctx, store.GetExperienceByOwnerParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound()
		}
		return nil, fmt.Errorf("profiles: load experience: %w", err)
	}
	skills, err := s.st.ListExperienceSkills(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, fmt.Errorf("profiles: list experience skills: %w", err)
	}
	grouped := groupExperienceSkills(skills)
	return &Experience{
		ID:             row.ID,
		Kind:           row.Kind,
		Title:          row.Title,
		Organization:   row.Organization,
		Location:       row.Location,
		EmploymentType: row.EmploymentType,
		StartMonth:     formatMonth(row.StartMonth),
		EndMonth:       formatOptionalMonth(row.EndMonth),
		IsCurrent:      row.IsCurrent,
		Description:    row.Description,
		CredentialURL:  row.CredentialUrl,
		IsSystem:       row.IsSystem,
		Skills:         grouped[row.ID],
	}, nil
}

func groupExperienceSkills(rows []store.ListExperienceSkillsRow) map[uuid.UUID][]ExperienceSkill {
	grouped := make(map[uuid.UUID][]ExperienceSkill)
	for _, row := range rows {
		grouped[row.ExperienceID] = append(grouped[row.ExperienceID], ExperienceSkill{
			Slug: row.Slug,
			Name: row.Name,
		})
	}
	return grouped
}

func insertExperienceSkills(ctx context.Context, q *store.Queries, experienceID uuid.UUID, slugs []string) error {
	for _, slug := range slugs {
		if err := q.InsertExperienceSkill(ctx, store.InsertExperienceSkillParams{
			ExperienceID: experienceID,
			Slug:         slug,
		}); err != nil {
			return fmt.Errorf("profiles: insert experience skill: %w", err)
		}
	}
	return nil
}

func validateExperience(in ExperienceInput) (validExperience, error) {
	kind := strings.TrimSpace(in.Kind)
	if _, ok := experienceKinds[kind]; !ok {
		return validExperience{}, apperr.Validation(apperr.Field(
			"kind", "invalid", "Jenis experience tidak dikenal.",
		))
	}

	title := strings.TrimSpace(in.Title)
	if length := utf8.RuneCountInString(title); length < 2 || length > maxExperienceTitleLength {
		return validExperience{}, apperr.Validation(apperr.Field(
			"title", "length", "Judul 2–120 karakter.",
		))
	}

	organization := strings.TrimSpace(in.Organization)
	if length := utf8.RuneCountInString(organization); length < 2 || length > maxExperienceOrganizationLen {
		return validExperience{}, apperr.Validation(apperr.Field(
			"organization", "length", "Instansi 2–120 karakter.",
		))
	}

	location, err := optionalText(in.Location, maxExperienceLocationLength, "location",
		"Lokasi maksimal 120 karakter.")
	if err != nil {
		return validExperience{}, err
	}

	var employmentType *string
	if in.EmploymentType != nil {
		value := strings.TrimSpace(*in.EmploymentType)
		if value != "" {
			if _, ok := employmentTypes[value]; !ok {
				return validExperience{}, apperr.Validation(apperr.Field(
					"employmentType", "invalid", "Tipe kerja tidak dikenal.",
				))
			}
			employmentType = &value
		}
	}

	start, err := parseMonth(in.StartMonth, "startMonth")
	if err != nil {
		return validExperience{}, err
	}

	var end pgtype.Date
	var endMonth *string
	if in.EndMonth != nil && strings.TrimSpace(*in.EndMonth) != "" {
		parsed, err := parseMonth(*in.EndMonth, "endMonth")
		if err != nil {
			return validExperience{}, err
		}
		if parsed.Time.Before(start.Time) {
			return validExperience{}, apperr.Validation(apperr.Field(
				"endMonth", "order", "Bulan selesai tidak boleh sebelum bulan mulai.",
			))
		}
		end = parsed
		formatted := parsed.Time.Format("2006-01")
		endMonth = &formatted
	}

	switch {
	case in.IsCurrent && endMonth != nil:
		return validExperience{}, apperr.Validation(apperr.Field(
			"endMonth", "not_allowed", "Bulan selesai tidak boleh diisi saat masih berjalan.",
		))
	case !in.IsCurrent && endMonth == nil:
		return validExperience{}, apperr.Validation(apperr.Field(
			"endMonth", "required", "Bulan selesai wajib diisi.",
		))
	}

	description, err := optionalText(in.Description, maxExperienceDescriptionLength, "description",
		"Deskripsi maksimal 2000 karakter.")
	if err != nil {
		return validExperience{}, err
	}

	credential, err := optionalText(in.CredentialURL, maxCredentialURLLength, "credentialUrl",
		"Tautan kredensial maksimal 500 karakter.")
	if err != nil {
		return validExperience{}, err
	}
	if credential != nil && !strings.HasPrefix(*credential, "https://") {
		return validExperience{}, apperr.Validation(apperr.Field(
			"credentialUrl", "invalid", "Tautan kredensial harus diawali https://.",
		))
	}

	return validExperience{
		Kind:           kind,
		Title:          title,
		Organization:   organization,
		Location:       location,
		EmploymentType: employmentType,
		Start:          start,
		End:            end,
		IsCurrent:      in.IsCurrent,
		Description:    description,
		CredentialURL:  credential,
		Skills:         in.Skills,
	}, nil
}

func validateExperienceSkills(ctx context.Context, q *store.Queries, raw []string) ([]string, error) {
	if len(raw) > maxExperienceSkills {
		return nil, apperr.Validation(apperr.Field(
			"skills", "limit", "Maksimal 10 skill.",
		))
	}
	slugs := make([]string, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for i, value := range raw {
		slug := strings.ToLower(strings.TrimSpace(value))
		if slug == "" {
			return nil, apperr.Validation(apperr.Field(
				fmt.Sprintf("skills[%d]", i), "invalid", "Slug skill tidak valid.",
			))
		}
		if _, duplicate := seen[slug]; duplicate {
			return nil, apperr.Validation(apperr.Field(
				fmt.Sprintf("skills[%d]", i), "duplicate", "Skill ganda tidak diperbolehkan.",
			))
		}
		seen[slug] = struct{}{}
		slugs[i] = slug
	}
	if len(slugs) == 0 {
		return slugs, nil
	}

	found, err := q.ListSkillsBySlugs(ctx, slugs)
	if err != nil {
		return nil, fmt.Errorf("profiles: load skills: %w", err)
	}
	if len(found) != len(slugs) {
		existing := make(map[string]struct{}, len(found))
		for _, skill := range found {
			existing[skill.Slug] = struct{}{}
		}
		for i, slug := range slugs {
			if _, ok := existing[slug]; !ok {
				return nil, apperr.Validation(apperr.Field(
					fmt.Sprintf("skills[%d]", i), "unknown", "Skill tidak ditemukan.",
				))
			}
		}
	}
	return slugs, nil
}

// parseMonth converts a YYYY-MM string to the first day of that month (UTC).
func parseMonth(value, field string) (pgtype.Date, error) {
	trimmed := strings.TrimSpace(value)
	if !monthPattern.MatchString(trimmed) {
		return pgtype.Date{}, apperr.Validation(apperr.Field(
			field, "format", "Format bulan harus YYYY-MM.",
		))
	}
	parsed, err := time.Parse("2006-01", trimmed)
	if err != nil {
		return pgtype.Date{}, apperr.Validation(apperr.Field(
			field, "format", "Format bulan harus YYYY-MM.",
		))
	}
	return pgtype.Date{Time: parsed, Valid: true}, nil
}

// formatMonth trims a database date text (YYYY-MM-DD) to YYYY-MM.
func formatMonth(value string) string {
	if len(value) >= 7 {
		return value[:7]
	}
	return value
}

// formatOptionalMonth maps the empty string used for a NULL end month to nil.
func formatOptionalMonth(value string) *string {
	if value == "" {
		return nil
	}
	formatted := formatMonth(value)
	return &formatted
}
