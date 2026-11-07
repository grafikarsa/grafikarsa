package users

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
)

const (
	accountTypeStudent = "student"
	accountTypeTeacher = "teacher"
	accountTypeStaff   = "staff"

	defaultListLimit = 50
	maxListLimit     = 100
	maxQueryLength   = 100

	minDisplayNameLength      = 1
	maxDisplayNameLength      = 100
	minStaffNumberLength      = 3
	maxStaffNumberLength      = 30
	minBlockReasonLength      = 3
	maxBlockReasonLength      = 300
	schoolOrganization        = "SMKN 4 Malang"
	initialPasswordTTLDays    = 30
	temporaryPasswordTTLHours = 72
)

var nisnPattern = regexp.MustCompile(`^[0-9]{10}$`)

// Actor is the authenticated principal performing an action, plus the request
// IP recorded in the audit log.
type Actor struct {
	Principal *auth.Principal
	IP        string
}

// UserSummary is the list representation of a user.
type UserSummary struct {
	ID             uuid.UUID
	AccountType    string
	Status         string
	DisplayName    string
	Username       *string
	NISN           *string
	StaffNumber    *string
	Activated      bool
	ContentBlocked bool
	CurrentClass   *ClassRef
	LastLoginAt    *time.Time
	CreatedAt      time.Time
}

// ClassRef identifies a class together with its major and academic year.
type ClassRef struct {
	ID                uuid.UUID
	Name              string
	MajorCode         string
	AcademicYearLabel string
}

// UserDetail is the admin detail representation of a user.
type UserDetail struct {
	UserSummary
	EntryYear         *int16
	GraduationYear    *int16
	TeacherMajorIDs   []uuid.UUID
	Roles             []RoleAssignment
	ActiveSessions    int
	PasswordKind      string
	PasswordExpiresAt *time.Time
	DeactivatedAt     *time.Time
	Enrollments       []EnrollmentRef
}

// RoleAssignment is one active role assignment of a user.
type RoleAssignment struct {
	ID        uuid.UUID
	RoleSlug  string
	RoleName  string
	MajorID   *uuid.UUID
	ExpiresAt *time.Time
}

// EnrollmentRef is one class enrollment in the user's history.
type EnrollmentRef struct {
	ClassID           uuid.UUID
	ClassName         string
	MajorCode         string
	AcademicYearLabel string
	Outcome           *string
}

// ListFilter narrows List. Zero values mean "no filter".
type ListFilter struct {
	AccountType string
	Status      string
	MajorID     *uuid.UUID
	ClassID     *uuid.UUID
	Query       string
	Cursor      string
	Limit       int
}

// ListPage is one page of users.
type ListPage struct {
	Items      []UserSummary
	NextCursor string
}

// CreateInput is the payload for Create.
type CreateInput struct {
	AccountType     string
	DisplayName     string
	NISN            *string
	StaffNumber     *string
	ClassID         *uuid.UUID
	EntryYear       *int16
	TeacherMajorIDs []uuid.UUID
}

// Created is the result of Create. InitialPassword is returned only once.
type Created struct {
	User            UserDetail
	InitialPassword string
}

// UpdateInput is the payload for Update. Nil fields are left unchanged.
type UpdateInput struct {
	DisplayName     *string
	NISN            *string
	StaffNumber     *string
	AccountType     *string
	TeacherMajorIDs *[]uuid.UUID
}

type permissionScope struct {
	global   bool
	majorIDs []uuid.UUID
}

// permissionScope returns the actor's scope for permission. The second result
// is false when the actor holds no grant for it.
func (a Actor) permissionScope(permission string) (permissionScope, bool) {
	if a.Principal == nil {
		return permissionScope{}, false
	}
	for _, grant := range a.Principal.Grants {
		if grant.Permission != permission {
			continue
		}
		if grant.Global {
			return permissionScope{global: true}, true
		}
		return permissionScope{majorIDs: append([]uuid.UUID(nil), grant.MajorIDs...)}, true
	}
	return permissionScope{}, false
}

func (s permissionScope) allows(majorIDs []uuid.UUID) bool {
	if s.global {
		return true
	}
	for _, allowed := range s.majorIDs {
		for _, majorID := range majorIDs {
			if allowed == majorID {
				return true
			}
		}
	}
	return false
}

func (in *CreateInput) normalize() {
	in.AccountType = strings.TrimSpace(in.AccountType)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.NISN = trimOptional(in.NISN)
	in.StaffNumber = trimOptional(in.StaffNumber)
	in.TeacherMajorIDs = dedupeUUIDs(in.TeacherMajorIDs)
}

func validateCreate(in CreateInput) error {
	fields := validateProfile(in.AccountType, in.DisplayName, in.NISN, in.StaffNumber)

	if in.AccountType == accountTypeStudent && in.ClassID == nil {
		fields = append(fields, apperr.Field(
			"class_id", "required", "Kelas wajib dipilih untuk siswa.",
		))
	}
	if in.AccountType != accountTypeTeacher && len(in.TeacherMajorIDs) > 0 {
		fields = append(fields, apperr.Field(
			"teacher_major_ids", "invalid", "Jurusan yang diampu hanya untuk guru.",
		))
	}
	if in.EntryYear != nil && (*in.EntryYear < 1950 || *in.EntryYear > 2100) {
		fields = append(fields, apperr.Field(
			"entry_year", "range", "Angkatan harus antara 1950 dan 2100.",
		))
	}
	return validationOrNil(fields)
}

func validateProfile(accountType, displayName string, nisn, staffNumber *string) []apperr.FieldError {
	var fields []apperr.FieldError

	if !validAccountType(accountType) {
		fields = append(fields, apperr.Field(
			"account_type", "invalid", "Tipe akun tidak valid.",
		))
	}

	length := utf8.RuneCountInString(displayName)
	if length < minDisplayNameLength || length > maxDisplayNameLength {
		fields = append(fields, apperr.Field(
			"display_name", "length", "Nama tampilan harus 1–100 karakter.",
		))
	}

	if nisn == nil {
		if accountType == accountTypeStudent {
			fields = append(fields, apperr.Field(
				"nisn", "required", "NISN wajib diisi untuk siswa.",
			))
		}
	} else if !nisnPattern.MatchString(*nisn) {
		fields = append(fields, apperr.Field(
			"nisn", "format", "NISN harus 10 digit angka.",
		))
	}

	if staffNumber != nil {
		n := utf8.RuneCountInString(*staffNumber)
		if n < minStaffNumberLength || n > maxStaffNumberLength {
			fields = append(fields, apperr.Field(
				"staff_number", "length", "NIP/ID staf harus 3–30 karakter.",
			))
		}
	}

	return fields
}

func validateBlockReason(reason string) error {
	n := utf8.RuneCountInString(reason)
	if n < minBlockReasonLength || n > maxBlockReasonLength {
		return apperr.Validation(apperr.Field(
			"reason", "length", "Alasan harus 3–300 karakter.",
		))
	}
	return nil
}

func validationOrNil(fields []apperr.FieldError) error {
	if len(fields) == 0 {
		return nil
	}
	return apperr.Validation(fields...)
}

func validAccountType(value string) bool {
	switch value {
	case accountTypeStudent, accountTypeTeacher, accountTypeStaff:
		return true
	default:
		return false
	}
}

func trimOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func dedupeUUIDs(values []uuid.UUID) []uuid.UUID {
	if len(values) == 0 {
		return []uuid.UUID{}
	}
	seen := make(map[uuid.UUID]struct{}, len(values))
	out := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func sameUUIDSet(a, b []uuid.UUID) bool {
	if len(a) != len(b) {
		return false
	}
	counts := make(map[uuid.UUID]int, len(a))
	for _, value := range a {
		counts[value]++
	}
	for _, value := range b {
		counts[value]--
		if counts[value] < 0 {
			return false
		}
	}
	return true
}
