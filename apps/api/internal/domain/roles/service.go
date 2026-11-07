// Package roles implements role and permission administration: the permission
// catalog, custom role definitions, and scoped role assignments with
// anti-privilege-escalation checks. HTTP handlers live elsewhere; this package
// only exposes the domain service.
package roles

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/audit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	permissionManage = "role.manage"
	permissionAssign = "role.assign"

	targetTypeRole = "role"
	targetTypeUser = "user"

	roleMember  = "member"
	roleTeacher = "teacher"

	statusActive  = "active"
	statusAlumni  = "alumni"
	minNameLength = 2
	maxNameLength = 60
)

var slugPattern = regexp.MustCompile(`^[a-z0-9_]{2,50}$`)

// SessionInvalidator drops cached principals for a user after their roles
// change. *auth.Service implements it in production.
type SessionInvalidator interface {
	InvalidateUser(ctx context.Context, userID uuid.UUID) error
}

// Service implements role and permission administration.
type Service struct {
	st  *store.Store
	inv SessionInvalidator
	now func() time.Time
}

// NewService builds a roles Service.
func NewService(st *store.Store, inv SessionInvalidator) *Service {
	return &Service{st: st, inv: inv, now: time.Now}
}

// Actor is the authenticated principal performing an action, plus the request
// IP recorded in the audit log.
type Actor struct {
	Principal *auth.Principal
	IP        string
}

// Permission is one entry of the permission catalog.
type Permission struct {
	Key         string
	Description string
	Scopable    bool
}

// Role is a role definition with its permission keys and active holder count.
type Role struct {
	ID          uuid.UUID
	Slug        string
	Name        string
	Description *string
	IsSystem    bool
	Permissions []string
	HolderCount int64
}

// Assignment is one active role assignment of a user.
type Assignment struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	RoleID     uuid.UUID
	RoleSlug   string
	RoleName   string
	MajorID    *uuid.UUID
	AssignedBy *uuid.UUID
	AssignedAt time.Time
	ExpiresAt  *time.Time
}

// ListPermissions returns the permission catalog. Visible to actors that can
// manage roles or assign them (in any scope).
func (s *Service) ListPermissions(ctx context.Context, a Actor) ([]Permission, error) {
	if !a.canViewRoles() {
		return nil, apperr.NotFound()
	}

	rows, err := s.st.ListPermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("roles: list permissions: %w", err)
	}

	permissions := make([]Permission, 0, len(rows))
	for _, row := range rows {
		permissions = append(permissions, Permission{
			Key:         row.Key,
			Description: row.Description,
			Scopable:    row.Scopable,
		})
	}
	return permissions, nil
}

// ListRoles returns every role definition. Visible to actors that can manage
// roles or assign them (in any scope).
func (s *Service) ListRoles(ctx context.Context, a Actor) ([]Role, error) {
	if !a.canViewRoles() {
		return nil, apperr.NotFound()
	}

	rows, err := s.st.ListRoles(ctx)
	if err != nil {
		return nil, fmt.Errorf("roles: list roles: %w", err)
	}

	byRole, err := s.rolePermissions(ctx, s.st.Queries)
	if err != nil {
		return nil, err
	}

	roles := make([]Role, 0, len(rows))
	for _, row := range rows {
		permissions := byRole[row.ID]
		if permissions == nil {
			permissions = []string{}
		}
		roles = append(roles, Role{
			ID:          row.ID,
			Slug:        row.Slug,
			Name:        row.Name,
			Description: row.Description,
			IsSystem:    row.IsSystem,
			Permissions: permissions,
			HolderCount: row.HolderCount,
		})
	}
	return roles, nil
}

// rolePermissions loads every role's permission keys, grouped by role id.
func (s *Service) rolePermissions(ctx context.Context, q *store.Queries) (map[uuid.UUID][]string, error) {
	rows, err := q.ListAllRolePermissions(ctx)
	if err != nil {
		return nil, fmt.Errorf("roles: list role permissions: %w", err)
	}

	byRole := make(map[uuid.UUID][]string)
	for _, row := range rows {
		byRole[row.RoleID] = append(byRole[row.RoleID], row.PermissionKey)
	}
	return byRole, nil
}

func (s *Service) loadRole(ctx context.Context, q *store.Queries, id uuid.UUID) (Role, error) {
	row, err := q.GetRoleByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Role{}, apperr.NotFound()
		}
		return Role{}, fmt.Errorf("roles: get role %s: %w", id, err)
	}

	permissions, err := q.ListRolePermissions(ctx, id)
	if err != nil {
		return Role{}, fmt.Errorf("roles: list role permissions: %w", err)
	}

	keys := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		keys = append(keys, permission.Key)
	}

	return Role{
		ID:          row.ID,
		Slug:        row.Slug,
		Name:        row.Name,
		Description: row.Description,
		IsSystem:    row.IsSystem,
		Permissions: keys,
		HolderCount: row.HolderCount,
	}, nil
}

func (s *Service) recordAudit(
	ctx context.Context,
	q *store.Queries,
	a Actor,
	action, targetType string,
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
		TargetType:   targetType,
		TargetID:     &targetID,
		ScopeMajorID: scopeMajorID,
		Data:         data,
	}); err != nil {
		return fmt.Errorf("roles: record audit %s: %w", action, err)
	}
	return nil
}

func (s *Service) invalidate(ctx context.Context, userID uuid.UUID) error {
	if s.inv == nil {
		return nil
	}
	if err := s.inv.InvalidateUser(ctx, userID); err != nil {
		return fmt.Errorf("roles: invalidate user %s: %w", userID, err)
	}
	return nil
}

func (a Actor) canViewRoles() bool {
	if a.Principal == nil {
		return false
	}
	return a.Principal.CanGlobally(permissionManage) || a.Principal.Can(permissionAssign)
}

func (a Actor) canManageRoles() bool {
	return a.Principal != nil && a.Principal.CanGlobally(permissionManage)
}

func (a Actor) canAssignGlobally() bool {
	return a.Principal != nil && a.Principal.CanGlobally(permissionAssign)
}

func (a Actor) canAssignForMajor(majorID uuid.UUID) bool {
	return a.Principal != nil && a.Principal.CanForMajor(permissionAssign, majorID)
}

func (a Actor) holdsGlobally(permission string) bool {
	return a.Principal != nil && a.Principal.CanGlobally(permission)
}

func (a Actor) holdsForMajor(permission string, majorID uuid.UUID) bool {
	return a.Principal != nil && a.Principal.CanForMajor(permission, majorID)
}

// assignScope returns the actor's role.assign scope: global or a set of major
// ids. The second result is false when the actor holds no grant.
func (a Actor) assignScope() (global bool, majorIDs []uuid.UUID, ok bool) {
	if a.Principal == nil {
		return false, nil, false
	}
	for _, grant := range a.Principal.Grants {
		if grant.Permission != permissionAssign {
			continue
		}
		if grant.Global {
			return true, nil, true
		}
		return false, append([]uuid.UUID(nil), grant.MajorIDs...), true
	}
	return false, nil, false
}

// checkAssignmentScope enforces the role.assign half of the anti-escalation
// rule for a target scope.
func (a Actor) checkAssignmentScope(majorID *uuid.UUID) bool {
	if majorID == nil {
		return a.canAssignGlobally()
	}
	return a.canAssignForMajor(*majorID)
}

// checkHoldsRolePermissions enforces the per-permission half of the
// anti-escalation rule: the actor must hold every permission of the role with
// at least the target scope.
func (a Actor) checkHoldsRolePermissions(permissions []store.ListRolePermissionsRow, majorID *uuid.UUID) bool {
	for _, permission := range permissions {
		if majorID == nil {
			if !a.holdsGlobally(permission.Key) {
				return false
			}
			continue
		}
		if !a.holdsForMajor(permission.Key, *majorID) {
			return false
		}
	}
	return true
}

func normalizeName(value string) string {
	return strings.TrimSpace(value)
}

func normalizeDescription(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// normalizePermissions trims, drops empty entries, and dedupes permission keys.
func normalizePermissions(keys []string) []string {
	if len(keys) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(keys))
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func validateRoleInput(slug, name string) error {
	var fields []apperr.FieldError

	if !slugPattern.MatchString(slug) {
		fields = append(fields, apperr.Field(
			"slug", "format", "Slug role harus 2–50 karakter, berisi huruf kecil, angka, atau garis bawah.",
		))
	}

	length := utf8.RuneCountInString(name)
	if length < minNameLength || length > maxNameLength {
		fields = append(fields, apperr.Field(
			"name", "length", "Nama role harus 2–60 karakter.",
		))
	}

	return validationOrNil(fields)
}

func validateRoleName(name string) error {
	length := utf8.RuneCountInString(name)
	if length < minNameLength || length > maxNameLength {
		return apperr.Validation(apperr.Field(
			"name", "length", "Nama role harus 2–60 karakter.",
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

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

// Ensure the production auth service satisfies the interface.
var _ SessionInvalidator = (*auth.Service)(nil)
