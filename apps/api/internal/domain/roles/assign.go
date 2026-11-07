package roles

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

// ListAssignments returns the active role assignments of one user that the
// actor may see. Scoped actors only see assignments within their scope.
func (s *Service) ListAssignments(ctx context.Context, a Actor, userID uuid.UUID) ([]Assignment, error) {
	global, majorIDs, ok := a.assignScope()
	if !ok {
		return nil, apperr.NotFound()
	}

	if _, err := s.st.GetRoleAssignmentTarget(ctx, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound()
		}
		return nil, fmt.Errorf("roles: get assignment target: %w", err)
	}

	rows, err := s.st.ListActiveUserRoleAssignments(ctx, store.ListActiveUserRoleAssignmentsParams{
		UserID:        userID,
		ScopeGlobal:   global,
		ScopeMajorIds: append([]uuid.UUID{}, majorIDs...),
	})
	if err != nil {
		return nil, fmt.Errorf("roles: list assignments: %w", err)
	}

	assignments := make([]Assignment, 0, len(rows))
	for _, row := range rows {
		assignments = append(assignments, assignmentFromRow(row))
	}
	return assignments, nil
}

// Assign grants a role to a user, globally or scoped to one major. The actor
// must hold role.assign at that scope and every permission of the role with at
// least the same scope.
func (s *Service) Assign(
	ctx context.Context,
	a Actor,
	userID uuid.UUID,
	roleSlug string,
	majorID *uuid.UUID,
	expiresAt *time.Time,
) (Assignment, error) {
	if !a.checkAssignmentScope(majorID) {
		return Assignment{}, apperr.NotFound()
	}

	expiresAt = normalizeExpiresAt(expiresAt)
	if expiresAt != nil && !expiresAt.After(s.now().UTC()) {
		return Assignment{}, apperr.Validation(apperr.Field(
			"expiresAt", "future", "Masa berlaku harus di masa depan.",
		))
	}

	roleSlug = strings.TrimSpace(roleSlug)
	if roleSlug == "" {
		return Assignment{}, apperr.NotFound()
	}

	var assignment Assignment
	err := s.st.WithTx(ctx, func(q *store.Queries) error {
		role, err := q.GetRoleBySlug(ctx, roleSlug)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.NotFound()
			}
			return fmt.Errorf("roles: get role by slug: %w", err)
		}

		if role.Slug == roleMember || role.Slug == roleTeacher {
			return apperr.Validation(apperr.Field(
				"role", "implicit", "Role ini otomatis dimiliki, tidak perlu diberikan.",
			))
		}

		permissions, err := q.ListRolePermissions(ctx, role.ID)
		if err != nil {
			return fmt.Errorf("roles: list role permissions: %w", err)
		}

		// Anti-escalation comes before the scopable check so an actor without
		// the role's permissions cannot learn the role shape.
		if !a.checkHoldsRolePermissions(permissions, majorID) {
			return apperr.NotFound()
		}

		if majorID != nil && !allScopable(permissions) {
			return apperr.Validation(apperr.Field(
				"majorId", "not_scopable", "Role ini hanya bisa diberikan tanpa scope jurusan.",
			))
		}

		if majorID != nil {
			if _, err := q.GetRoleMajor(ctx, *majorID); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return apperr.Validation(apperr.Field(
						"majorId", "not_found", "Jurusan tidak ditemukan.",
					))
				}
				return fmt.Errorf("roles: get major: %w", err)
			}
		}

		target, err := q.GetRoleAssignmentTarget(ctx, userID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.NotFound()
			}
			return fmt.Errorf("roles: get assignment target: %w", err)
		}
		if target.Status != statusActive && target.Status != statusAlumni {
			return apperr.Validation(apperr.Field(
				"user", "invalid", "Pengguna tidak aktif.",
			))
		}

		inserted, err := q.InsertUserRole(ctx, store.InsertUserRoleParams{
			UserID:     userID,
			RoleID:     role.ID,
			MajorID:    majorID,
			AssignedBy: a.Principal.UserID,
			ExpiresAt:  expiresAt,
		})
		if err != nil {
			if isUniqueViolation(err) {
				return apperr.Conflict("Role ini sudah dimiliki pengguna.")
			}
			return fmt.Errorf("roles: insert assignment: %w", err)
		}

		if err := s.recordAudit(ctx, q, a, "role.assign", targetTypeUser, userID, majorID, map[string]any{
			"assignment_id": inserted.ID,
			"role_id":       role.ID,
			"role_slug":     role.Slug,
			"expires_at":    expiresAt,
		}); err != nil {
			return err
		}

		assignment = Assignment{
			ID:         inserted.ID,
			UserID:     userID,
			RoleID:     role.ID,
			RoleSlug:   role.Slug,
			RoleName:   role.Name,
			MajorID:    majorID,
			AssignedBy: &a.Principal.UserID,
			AssignedAt: inserted.AssignedAt,
			ExpiresAt:  expiresAt,
		}
		return nil
	})
	if err != nil {
		return Assignment{}, err
	}

	if err := s.invalidate(ctx, userID); err != nil {
		return Assignment{}, err
	}
	return assignment, nil
}

// Revoke ends one active assignment. The actor must satisfy the same
// anti-escalation rule as if they were granting the role with that scope.
func (s *Service) Revoke(ctx context.Context, a Actor, assignmentID uuid.UUID) error {
	if a.Principal == nil {
		return apperr.NotFound()
	}

	var userID uuid.UUID
	err := s.st.WithTx(ctx, func(q *store.Queries) error {
		assignment, err := q.GetUserRoleAssignment(ctx, assignmentID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.NotFound()
			}
			return fmt.Errorf("roles: get assignment: %w", err)
		}

		if !a.checkAssignmentScope(assignment.MajorID) {
			return apperr.NotFound()
		}

		permissions, err := q.ListRolePermissions(ctx, assignment.RoleID)
		if err != nil {
			return fmt.Errorf("roles: list role permissions: %w", err)
		}
		if !a.checkHoldsRolePermissions(permissions, assignment.MajorID) {
			return apperr.NotFound()
		}

		revokedUserID, err := q.RevokeUserRole(ctx, store.RevokeUserRoleParams{
			RevokedBy: a.Principal.UserID,
			ID:        assignmentID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.NotFound()
			}
			return fmt.Errorf("roles: revoke assignment: %w", err)
		}

		if err := s.recordAudit(ctx, q, a, "role.revoke", targetTypeUser, revokedUserID, assignment.MajorID, map[string]any{
			"assignment_id": assignment.ID,
			"role_id":       assignment.RoleID,
			"role_slug":     assignment.RoleSlug,
		}); err != nil {
			return err
		}

		userID = revokedUserID
		return nil
	})
	if err != nil {
		return err
	}

	return s.invalidate(ctx, userID)
}

func normalizeExpiresAt(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func allScopable(permissions []store.ListRolePermissionsRow) bool {
	for _, permission := range permissions {
		if !permission.Scopable {
			return false
		}
	}
	return true
}

func assignmentFromRow(row store.ListActiveUserRoleAssignmentsRow) Assignment {
	return Assignment{
		ID:         row.ID,
		UserID:     row.UserID,
		RoleID:     row.RoleID,
		RoleSlug:   row.RoleSlug,
		RoleName:   row.RoleName,
		MajorID:    row.MajorID,
		AssignedBy: row.AssignedBy,
		AssignedAt: row.AssignedAt,
		ExpiresAt:  row.ExpiresAt,
	}
}
