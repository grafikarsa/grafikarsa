// Package users implements user administration: listing and searching users,
// creating accounts, editing profile data, resetting passwords, deactivating
// accounts, and blocking content creation. HTTP handlers live elsewhere; this
// package only exposes the domain service.
package users

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/audit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/settings"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/password"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	permissionView         = "user.view"
	permissionCreate       = "user.create"
	permissionUpdate       = "user.update"
	permissionReset        = "user.reset_password"
	permissionDeactivate   = "user.deactivate"
	permissionBlockContent = "user.block_content"

	targetTypeUser = "user"

	initialPasswordTTLKey   = "auth.initial_password_ttl_days"
	temporaryPasswordTTLKey = "auth.temporary_password_ttl_hours"
)

// SessionInvalidator drops cached principals for a user after their account
// changes. *auth.Service implements it in production.
type SessionInvalidator interface {
	InvalidateUser(ctx context.Context, userID uuid.UUID) error
}

// Service implements user administration operations.
type Service struct {
	st      *store.Store
	authSvc SessionInvalidator
	hasher  *password.Hasher
	now     func() time.Time
}

// NewService builds a users Service.
func NewService(st *store.Store, authSvc SessionInvalidator, hasher *password.Hasher) *Service {
	return &Service{
		st:      st,
		authSvc: authSvc,
		hasher:  hasher,
		now:     time.Now,
	}
}

// List returns users visible to the actor, newest first.
func (s *Service) List(ctx context.Context, a Actor, f ListFilter) (ListPage, error) {
	scope, ok := a.permissionScope(permissionView)
	if !ok {
		return ListPage{}, apperr.NotFound()
	}

	limit := f.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	query := strings.TrimSpace(f.Query)
	var queryParam *string
	if query != "" {
		if utf8.RuneCountInString(query) > maxQueryLength {
			return ListPage{}, apperr.Validation(apperr.Field(
				"q", "length", "Kata kunci maksimal 100 karakter.",
			))
		}
		queryParam = &query
	}

	params := store.ListUsersParams{
		AccountType:   optionalNonEmpty(f.AccountType),
		Status:        optionalNonEmpty(f.Status),
		MajorID:       f.MajorID,
		ClassID:       f.ClassID,
		Query:         queryParam,
		ScopeGlobal:   scope.global,
		ScopeMajorIds: append([]uuid.UUID{}, scope.majorIDs...),
		// Fetch one extra row to detect whether another page exists.
		LimitCount: int32(limit + 1),
	}

	if f.Cursor != "" {
		cursorTime, cursorID, err := decodeCursor(f.Cursor)
		if err != nil {
			return ListPage{}, apperr.BadRequest("cursor tidak valid")
		}
		params.CursorTime = &cursorTime
		params.CursorID = &cursorID
	}

	rows, err := s.st.ListUsers(ctx, params)
	if err != nil {
		return ListPage{}, fmt.Errorf("users: list users: %w", err)
	}

	page := ListPage{Items: make([]UserSummary, 0, min(len(rows), limit))}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		page.NextCursor = encodeCursor(last.CreatedAt, last.ID)
	}
	for _, row := range rows {
		page.Items = append(page.Items, summaryFromListRow(row))
	}
	return page, nil
}

// Get returns the admin detail of one user.
func (s *Service) Get(ctx context.Context, a Actor, id uuid.UUID) (UserDetail, error) {
	scope, ok := a.permissionScope(permissionView)
	if !ok {
		return UserDetail{}, apperr.NotFound()
	}

	target, err := s.loadTarget(ctx, s.st.Queries, id)
	if err != nil {
		return UserDetail{}, err
	}
	if !scope.allows(target.majorIDs()) {
		return UserDetail{}, apperr.NotFound()
	}
	return s.detail(ctx, s.st.Queries, target)
}

type targetUser struct {
	row             store.GetUserAdminDetailRow
	teacherMajorIDs []uuid.UUID
}

// majorIDs returns the majors the target belongs to according to the scope
// rule: a student's latest enrollment major, or every major a teacher
// teaches. Staff belong to no major.
func (t targetUser) majorIDs() []uuid.UUID {
	switch t.row.AccountType {
	case accountTypeStudent:
		if t.row.ClassMajorID != nil {
			return []uuid.UUID{*t.row.ClassMajorID}
		}
	case accountTypeTeacher:
		return t.teacherMajorIDs
	}
	return nil
}

func (t targetUser) auditScopeMajorID() *uuid.UUID {
	majorIDs := t.majorIDs()
	if len(majorIDs) == 1 {
		majorID := majorIDs[0]
		return &majorID
	}
	return nil
}

func (s *Service) loadTarget(ctx context.Context, q *store.Queries, id uuid.UUID) (targetUser, error) {
	row, err := q.GetUserAdminDetail(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return targetUser{}, apperr.NotFound()
		}
		return targetUser{}, fmt.Errorf("users: get user %s: %w", id, err)
	}

	teacherMajorIDs, err := q.ListTeacherMajorIDs(ctx, id)
	if err != nil {
		return targetUser{}, fmt.Errorf("users: list teacher majors: %w", err)
	}

	return targetUser{row: row, teacherMajorIDs: teacherMajorIDs}, nil
}

func (s *Service) detail(ctx context.Context, q *store.Queries, target targetUser) (UserDetail, error) {
	roles, err := q.ListUserRoleAssignments(ctx, target.row.ID)
	if err != nil {
		return UserDetail{}, fmt.Errorf("users: list roles: %w", err)
	}
	enrollments, err := q.ListUserEnrollments(ctx, target.row.ID)
	if err != nil {
		return UserDetail{}, fmt.Errorf("users: list enrollments: %w", err)
	}
	activeSessions, err := q.CountUserActiveSessions(ctx, target.row.ID)
	if err != nil {
		return UserDetail{}, fmt.Errorf("users: count active sessions: %w", err)
	}

	detail := UserDetail{
		UserSummary:       summaryFromDetailRow(target.row),
		EntryYear:         target.row.EntryYear,
		GraduationYear:    target.row.GraduationYear,
		TeacherMajorIDs:   append([]uuid.UUID{}, target.teacherMajorIDs...),
		Roles:             make([]RoleAssignment, 0, len(roles)),
		ActiveSessions:    int(activeSessions),
		PasswordKind:      target.row.PasswordKind,
		PasswordExpiresAt: target.row.PasswordExpiresAt,
		DeactivatedAt:     target.row.DeactivatedAt,
		Enrollments:       make([]EnrollmentRef, 0, len(enrollments)),
	}
	for _, role := range roles {
		detail.Roles = append(detail.Roles, RoleAssignment{
			ID:        role.ID,
			RoleSlug:  role.RoleSlug,
			RoleName:  role.RoleName,
			MajorID:   role.MajorID,
			ExpiresAt: role.ExpiresAt,
		})
	}
	for _, enrollment := range enrollments {
		detail.Enrollments = append(detail.Enrollments, EnrollmentRef{
			ClassID:           enrollment.ClassID,
			ClassName:         enrollment.ClassName,
			MajorCode:         enrollment.MajorCode,
			AcademicYearLabel: academicYearLabel(enrollment.StartYear),
			Outcome:           enrollment.Outcome,
		})
	}
	return detail, nil
}

func (s *Service) detailByID(ctx context.Context, id uuid.UUID) (UserDetail, error) {
	target, err := s.loadTarget(ctx, s.st.Queries, id)
	if err != nil {
		return UserDetail{}, err
	}
	return s.detail(ctx, s.st.Queries, target)
}

func (s *Service) recordAudit(
	ctx context.Context,
	q *store.Queries,
	a Actor,
	action string,
	targetID uuid.UUID,
	scopeMajorID *uuid.UUID,
	data map[string]any,
) error {
	if a.Principal == nil {
		return apperr.NotFound()
	}
	actorID := a.Principal.UserID
	if err := audit.Record(ctx, q, audit.Entry{
		ActorID:      &actorID,
		ActorIP:      a.IP,
		Action:       action,
		TargetType:   targetTypeUser,
		TargetID:     &targetID,
		ScopeMajorID: scopeMajorID,
		Data:         data,
	}); err != nil {
		return fmt.Errorf("users: record audit %s: %w", action, err)
	}
	return nil
}

func (s *Service) invalidate(ctx context.Context, userID uuid.UUID) error {
	if s.authSvc == nil {
		return nil
	}
	if err := s.authSvc.InvalidateUser(ctx, userID); err != nil {
		return fmt.Errorf("users: invalidate user %s: %w", userID, err)
	}
	return nil
}

func (s *Service) generatePassword(ctx context.Context) (plain, hash string, err error) {
	plain, err = securePassword()
	if err != nil {
		return "", "", err
	}
	hash, err = s.hasher.Hash(ctx, plain)
	if err != nil {
		return "", "", fmt.Errorf("users: hash password: %w", err)
	}
	return plain, hash, nil
}

func (s *Service) settingInt(ctx context.Context, key string, def int) int {
	return settings.Int(ctx, s.st.Queries, key, def)
}

func summaryFromListRow(row store.ListUsersRow) UserSummary {
	return UserSummary{
		ID:             row.ID,
		AccountType:    row.AccountType,
		Status:         row.Status,
		DisplayName:    row.DisplayName,
		Username:       row.Username,
		NISN:           row.Nisn,
		StaffNumber:    row.StaffNumber,
		Activated:      row.Activated,
		ContentBlocked: row.ContentBlocked,
		CurrentClass:   classRef(row.ClassID, row.ClassName, row.ClassMajorCode, row.ClassStartYear),
		LastLoginAt:    row.LastLoginAt,
		CreatedAt:      row.CreatedAt,
	}
}

func summaryFromDetailRow(row store.GetUserAdminDetailRow) UserSummary {
	return UserSummary{
		ID:             row.ID,
		AccountType:    row.AccountType,
		Status:         row.Status,
		DisplayName:    row.DisplayName,
		Username:       row.Username,
		NISN:           row.Nisn,
		StaffNumber:    row.StaffNumber,
		Activated:      row.Activated,
		ContentBlocked: row.ContentBlocked,
		CurrentClass:   classRef(row.ClassID, row.ClassName, row.ClassMajorCode, row.ClassStartYear),
		LastLoginAt:    row.LastLoginAt,
		CreatedAt:      row.CreatedAt,
	}
}

func classRef(id *uuid.UUID, name, majorCode *string, startYear *int16) *ClassRef {
	if id == nil || name == nil || majorCode == nil || startYear == nil {
		return nil
	}
	return &ClassRef{
		ID:                *id,
		Name:              *name,
		MajorCode:         *majorCode,
		AcademicYearLabel: academicYearLabel(*startYear),
	}
}

func academicYearLabel(startYear int16) string {
	return fmt.Sprintf("%d/%d", startYear, startYear+1)
}

func optionalNonEmpty(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func userConflict(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return nil
	}
	switch pgErr.ConstraintName {
	case "users_nisn_key":
		return apperr.Conflict("NISN sudah terdaftar.")
	case "users_staff_number_key":
		return apperr.Conflict("NIP/ID staf sudah terdaftar.")
	default:
		return apperr.Conflict("Data pengguna sudah terdaftar.")
	}
}

type cursorPayload struct {
	Time string `json:"t"`
	ID   string `json:"i"`
}

func encodeCursor(createdAt time.Time, id uuid.UUID) string {
	raw, err := json.Marshal(cursorPayload{
		Time: createdAt.UTC().Format(time.RFC3339Nano),
		ID:   id.String(),
	})
	if err != nil {
		// Marshalling a struct of strings cannot fail.
		panic("users: encode cursor: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeCursor(value string) (time.Time, uuid.UUID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	var payload cursorPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return time.Time{}, uuid.Nil, err
	}
	createdAt, err := time.Parse(time.RFC3339Nano, payload.Time)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	id, err := uuid.Parse(payload.ID)
	if err != nil || id == uuid.Nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("users: invalid cursor id")
	}
	return createdAt, id, nil
}

func firstDayOfJuly(year int) pgtype.Date {
	return pgtype.Date{
		Time:  time.Date(year, time.July, 1, 0, 0, 0, 0, time.UTC),
		Valid: true,
	}
}

// Ensure the production auth service satisfies the interface.
var _ SessionInvalidator = (*auth.Service)(nil)
