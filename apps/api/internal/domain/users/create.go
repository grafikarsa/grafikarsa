package users

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/securetoken"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

// Create inserts one account, its initial academic links, and the audit entry
// in a single transaction. The plain initial password is returned once.
func (s *Service) Create(ctx context.Context, a Actor, in CreateInput) (Created, error) {
	scope, ok := a.permissionScope(permissionCreate)
	if !ok {
		return Created{}, apperr.NotFound()
	}

	in.normalize()
	if err := validateCreate(in); err != nil {
		return Created{}, err
	}
	if in.AccountType != accountTypeStudent && !scope.global {
		return Created{}, apperr.NotFound()
	}

	plain, hash, err := s.generatePassword(ctx)
	if err != nil {
		return Created{}, err
	}

	ttlDays := s.settingInt(ctx, initialPasswordTTLKey, initialPasswordTTLDays)
	expiresAt := s.now().UTC().Add(time.Duration(ttlDays) * 24 * time.Hour)

	var createdID uuid.UUID
	err = s.st.WithTx(ctx, func(q *store.Queries) error {
		var (
			class        store.GetClassByIDRow
			entryYear    *int16
			scopeMajorID *uuid.UUID
		)

		if in.AccountType == accountTypeStudent {
			row, err := q.GetClassByID(ctx, *in.ClassID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return apperr.NotFound()
				}
				return fmt.Errorf("users: get class: %w", err)
			}
			if !scope.allows([]uuid.UUID{row.MajorID}) {
				return apperr.NotFound()
			}

			year, err := q.GetAcademicYearByID(ctx, row.AcademicYearID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return apperr.NotFound()
				}
				return fmt.Errorf("users: get academic year: %w", err)
			}

			computedEntryYear := int16(year.StartYear) - (row.Grade - 10)
			if in.EntryYear != nil {
				computedEntryYear = *in.EntryYear
			}
			entryYear = &computedEntryYear
			class = row
			scopeMajorID = &row.MajorID
		}

		if in.AccountType == accountTypeTeacher {
			if err := s.ensureMajorsExist(ctx, q, in.TeacherMajorIDs); err != nil {
				return err
			}
		}

		id, err := q.InsertUser(ctx, store.InsertUserParams{
			AccountType:       in.AccountType,
			Nisn:              in.NISN,
			StaffNumber:       in.StaffNumber,
			DisplayName:       in.DisplayName,
			PasswordHash:      hash,
			PasswordKind:      "initial",
			PasswordExpiresAt: expiresAt,
			EntryYear:         entryYear,
		})
		if err != nil {
			if conflict := userConflict(err); conflict != nil {
				return conflict
			}
			return fmt.Errorf("users: insert user: %w", err)
		}

		switch in.AccountType {
		case accountTypeStudent:
			if err := q.InsertEnrollment(ctx, store.InsertEnrollmentParams{
				UserID:         id,
				ClassID:        class.ID,
				AcademicYearID: class.AcademicYearID,
				Source:         "admin",
			}); err != nil {
				return fmt.Errorf("users: insert enrollment: %w", err)
			}

			major, err := q.GetMajorByID(ctx, class.MajorID)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return apperr.NotFound()
				}
				return fmt.Errorf("users: get major: %w", err)
			}
			if err := q.InsertSystemExperience(ctx, store.InsertSystemExperienceParams{
				UserID:       id,
				Title:        "Siswa " + major.Name,
				Organization: schoolOrganization,
				StartMonth:   firstDayOfJuly(int(*entryYear)),
			}); err != nil {
				return fmt.Errorf("users: insert system experience: %w", err)
			}
		case accountTypeTeacher:
			if err := s.replaceTeacherMajors(ctx, q, id, in.TeacherMajorIDs); err != nil {
				return err
			}
		}

		if err := s.recordAudit(ctx, q, a, "user.create", id, scopeMajorID, map[string]any{
			"account_type": in.AccountType,
			"class_id":     in.ClassID,
		}); err != nil {
			return err
		}

		createdID = id
		return nil
	})
	if err != nil {
		return Created{}, err
	}

	detail, err := s.detailByID(ctx, createdID)
	if err != nil {
		return Created{}, err
	}
	return Created{User: detail, InitialPassword: plain}, nil
}

// Update changes profile fields and teacher assignments. Denials are reported
// as not found so the existence of other majors' users does not leak.
func (s *Service) Update(ctx context.Context, a Actor, id uuid.UUID, in UpdateInput) (UserDetail, error) {
	scope, ok := a.permissionScope(permissionUpdate)
	if !ok {
		return UserDetail{}, apperr.NotFound()
	}

	err := s.st.WithTx(ctx, func(q *store.Queries) error {
		target, err := s.loadTarget(ctx, q, id)
		if err != nil {
			return err
		}
		if !scope.allows(target.majorIDs()) {
			return apperr.NotFound()
		}

		current := target.row
		globalRequired := current.AccountType == accountTypeTeacher ||
			current.AccountType == accountTypeStaff

		finalType := current.AccountType
		if in.AccountType != nil {
			value := strings.TrimSpace(*in.AccountType)
			if !validAccountType(value) {
				return apperr.Validation(apperr.Field(
					"account_type", "invalid", "Tipe akun tidak valid.",
				))
			}
			if value != current.AccountType {
				globalRequired = true
				finalType = value
			}
		}
		if in.TeacherMajorIDs != nil {
			globalRequired = true
		}
		if globalRequired && !scope.global {
			return apperr.NotFound()
		}

		displayName := current.DisplayName
		if in.DisplayName != nil {
			displayName = strings.TrimSpace(*in.DisplayName)
		}
		nisn := current.Nisn
		if in.NISN != nil {
			nisn = trimOptional(in.NISN)
		}
		staffNumber := current.StaffNumber
		if in.StaffNumber != nil {
			staffNumber = trimOptional(in.StaffNumber)
		}

		if err := validationOrNil(validateProfile(finalType, displayName, nisn, staffNumber)); err != nil {
			return err
		}

		var finalTeacherMajorIDs []uuid.UUID
		teacherMajorsEdited := false
		if in.TeacherMajorIDs != nil {
			finalTeacherMajorIDs = dedupeUUIDs(*in.TeacherMajorIDs)
			teacherMajorsEdited = true
		} else {
			finalTeacherMajorIDs = target.teacherMajorIDs
		}
		if finalType != accountTypeTeacher && len(finalTeacherMajorIDs) > 0 {
			if teacherMajorsEdited {
				return apperr.Validation(apperr.Field(
					"teacher_major_ids", "invalid", "Jurusan yang diampu hanya untuk guru.",
				))
			}
			// Changing a teacher to another type removes stale assignments.
			teacherMajorsEdited = true
			finalTeacherMajorIDs = []uuid.UUID{}
		}
		if finalType == accountTypeTeacher && teacherMajorsEdited {
			if err := s.ensureMajorsExist(ctx, q, finalTeacherMajorIDs); err != nil {
				return err
			}
		}

		fields := changedFields(current, finalType, displayName, nisn, staffNumber)
		if teacherMajorsEdited && !sameUUIDSet(finalTeacherMajorIDs, target.teacherMajorIDs) {
			fields = append(fields, "teacher_major_ids")
		}
		if len(fields) == 0 {
			return nil
		}

		if err := q.UpdateUserProfile(ctx, store.UpdateUserProfileParams{
			AccountType: finalType,
			DisplayName: displayName,
			Nisn:        nisn,
			StaffNumber: staffNumber,
			ID:          id,
		}); err != nil {
			if conflict := userConflict(err); conflict != nil {
				return conflict
			}
			return fmt.Errorf("users: update profile: %w", err)
		}

		if teacherMajorsEdited {
			if err := s.replaceTeacherMajors(ctx, q, id, finalTeacherMajorIDs); err != nil {
				return err
			}
		}

		return s.recordAudit(ctx, q, a, "user.update", id, target.auditScopeMajorID(), map[string]any{
			"fields": fields,
		})
	})
	if err != nil {
		return UserDetail{}, err
	}

	if err := s.invalidate(ctx, id); err != nil {
		return UserDetail{}, err
	}
	return s.detailByID(ctx, id)
}

func (s *Service) ensureMajorsExist(ctx context.Context, q *store.Queries, majorIDs []uuid.UUID) error {
	for _, majorID := range majorIDs {
		if _, err := q.GetMajorByID(ctx, majorID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.Validation(apperr.Field(
					"teacher_major_ids", "not_found", "Jurusan tidak ditemukan.",
				))
			}
			return fmt.Errorf("users: get major: %w", err)
		}
	}
	return nil
}

func (s *Service) replaceTeacherMajors(ctx context.Context, q *store.Queries, userID uuid.UUID, majorIDs []uuid.UUID) error {
	if err := q.DeleteTeacherMajors(ctx, userID); err != nil {
		return fmt.Errorf("users: delete teacher majors: %w", err)
	}
	for _, majorID := range majorIDs {
		if err := q.InsertTeacherMajor(ctx, store.InsertTeacherMajorParams{
			UserID:  userID,
			MajorID: majorID,
		}); err != nil {
			return fmt.Errorf("users: insert teacher major: %w", err)
		}
	}
	return nil
}

func changedFields(
	current store.GetUserAdminDetailRow,
	finalType, displayName string,
	nisn, staffNumber *string,
) []string {
	fields := make([]string, 0, 4)
	if finalType != current.AccountType {
		fields = append(fields, "account_type")
	}
	if displayName != current.DisplayName {
		fields = append(fields, "display_name")
	}
	if !sameOptional(nisn, current.Nisn) {
		fields = append(fields, "nisn")
	}
	if !sameOptional(staffNumber, current.StaffNumber) {
		fields = append(fields, "staff_number")
	}
	return fields
}

func sameOptional(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func securePassword() (string, error) {
	password, err := securetoken.ReadablePassword()
	if err != nil {
		return "", fmt.Errorf("users: generate password: %w", err)
	}
	return password, nil
}
