package roles

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

// CreateRole inserts a custom role and its permissions. Every permission must
// exist and be held globally by the actor.
func (s *Service) CreateRole(
	ctx context.Context,
	a Actor,
	slug, name string,
	description *string,
	permissions []string,
) (Role, error) {
	if !a.canManageRoles() {
		return Role{}, apperr.NotFound()
	}

	slug = normalizeName(slug)
	name = normalizeName(name)
	description = normalizeDescription(description)
	permissionKeys := normalizePermissions(permissions)

	if err := validateRoleInput(slug, name); err != nil {
		return Role{}, err
	}

	var created Role
	err := s.st.WithTx(ctx, func(q *store.Queries) error {
		if err := s.ensurePermissionsExist(ctx, q, permissionKeys); err != nil {
			return err
		}
		for _, permission := range permissionKeys {
			if !a.holdsGlobally(permission) {
				return apperr.NotFound()
			}
		}

		id, err := q.InsertRole(ctx, store.InsertRoleParams{
			Slug:        slug,
			Name:        name,
			Description: description,
			CreatedBy:   a.Principal.UserID,
		})
		if err != nil {
			if isUniqueViolation(err) {
				return apperr.Conflict("Slug role sudah digunakan.")
			}
			return fmt.Errorf("roles: insert role: %w", err)
		}

		for _, permission := range permissionKeys {
			if err := q.InsertRolePermission(ctx, store.InsertRolePermissionParams{
				RoleID:        id,
				PermissionKey: permission,
			}); err != nil {
				return fmt.Errorf("roles: insert role permission: %w", err)
			}
		}

		if err := s.recordAudit(ctx, q, a, "role.create", targetTypeRole, id, nil, map[string]any{
			"slug":        slug,
			"permissions": permissionKeys,
		}); err != nil {
			return err
		}

		created, err = s.loadRole(ctx, q, id)
		return err
	})
	if err != nil {
		return Role{}, err
	}
	return created, nil
}

// UpdateRole edits a role's name and description. System roles may be renamed;
// their slug and permissions stay untouched.
func (s *Service) UpdateRole(ctx context.Context, a Actor, id uuid.UUID, name *string, description *string) (Role, error) {
	if !a.canManageRoles() {
		return Role{}, apperr.NotFound()
	}

	var normalizedName *string
	if name != nil {
		value := normalizeName(*name)
		if err := validateRoleName(value); err != nil {
			return Role{}, err
		}
		normalizedName = &value
	}

	var updated Role
	err := s.st.WithTx(ctx, func(q *store.Queries) error {
		current, err := q.GetRoleByID(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.NotFound()
			}
			return fmt.Errorf("roles: get role: %w", err)
		}

		fields := make([]string, 0, 2)
		if normalizedName != nil && *normalizedName != current.Name {
			fields = append(fields, "name")
		}
		normalizedDescription := normalizeDescription(description)
		if description != nil && !sameOptional(normalizedDescription, current.Description) {
			fields = append(fields, "description")
		}

		if len(fields) > 0 {
			if _, err := q.UpdateRole(ctx, store.UpdateRoleParams{
				Name:           normalizedName,
				Description:    normalizedDescription,
				DescriptionSet: description != nil,
				ID:             id,
			}); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return apperr.NotFound()
				}
				return fmt.Errorf("roles: update role: %w", err)
			}

			if err := s.recordAudit(ctx, q, a, "role.update", targetTypeRole, id, nil, map[string]any{
				"fields": fields,
			}); err != nil {
				return err
			}
		}

		updated, err = s.loadRole(ctx, q, id)
		return err
	})
	if err != nil {
		return Role{}, err
	}
	return updated, nil
}

// SetRolePermissions replaces a custom role's permission set. Only newly added
// permissions must be held globally by the actor; system roles are immutable.
func (s *Service) SetRolePermissions(ctx context.Context, a Actor, id uuid.UUID, permissions []string) (Role, error) {
	if !a.canManageRoles() {
		return Role{}, apperr.NotFound()
	}

	permissionKeys := normalizePermissions(permissions)

	var (
		updated Role
		holders []uuid.UUID
	)
	err := s.st.WithTx(ctx, func(q *store.Queries) error {
		current, err := q.GetRoleByID(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.NotFound()
			}
			return fmt.Errorf("roles: get role: %w", err)
		}
		if current.IsSystem {
			return apperr.Conflict("Role sistem tidak bisa diubah.")
		}

		if err := s.ensurePermissionsExist(ctx, q, permissionKeys); err != nil {
			return err
		}

		currentPermissions, err := q.ListRolePermissions(ctx, id)
		if err != nil {
			return fmt.Errorf("roles: list role permissions: %w", err)
		}
		currentSet := make(map[string]struct{}, len(currentPermissions))
		for _, permission := range currentPermissions {
			currentSet[permission.Key] = struct{}{}
		}

		added := make([]string, 0, len(permissionKeys))
		for _, permission := range permissionKeys {
			if _, exists := currentSet[permission]; exists {
				continue
			}
			if !a.holdsGlobally(permission) {
				return apperr.NotFound()
			}
			added = append(added, permission)
		}

		if err := q.DeleteRolePermissions(ctx, id); err != nil {
			return fmt.Errorf("roles: delete role permissions: %w", err)
		}
		for _, permission := range permissionKeys {
			if err := q.InsertRolePermission(ctx, store.InsertRolePermissionParams{
				RoleID:        id,
				PermissionKey: permission,
			}); err != nil {
				return fmt.Errorf("roles: insert role permission: %w", err)
			}
		}

		if err := s.recordAudit(ctx, q, a, "role.permissions_update", targetTypeRole, id, nil, map[string]any{
			"permissions": permissionKeys,
			"added":       added,
		}); err != nil {
			return err
		}

		holders, err = q.ListActiveRoleHolderIDs(ctx, id)
		if err != nil {
			return fmt.Errorf("roles: list role holders: %w", err)
		}

		updated, err = s.loadRole(ctx, q, id)
		return err
	})
	if err != nil {
		return Role{}, err
	}

	for _, userID := range holders {
		if err := s.invalidate(ctx, userID); err != nil {
			return Role{}, err
		}
	}
	return updated, nil
}

// DeleteRole removes a custom role that has no active holders. System roles are
// never deletable.
func (s *Service) DeleteRole(ctx context.Context, a Actor, id uuid.UUID) error {
	if !a.canManageRoles() {
		return apperr.NotFound()
	}

	return s.st.WithTx(ctx, func(q *store.Queries) error {
		role, err := q.GetRoleByID(ctx, id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperr.NotFound()
			}
			return fmt.Errorf("roles: get role: %w", err)
		}
		if role.IsSystem {
			return apperr.Conflict("Role sistem tidak bisa diubah.")
		}

		holders, err := q.CountActiveRoleHolders(ctx, id)
		if err != nil {
			return fmt.Errorf("roles: count role holders: %w", err)
		}
		if holders > 0 {
			return apperr.Conflict("Role ini masih dimiliki pengguna.")
		}

		// Revoked and expired assignments still reference the role through a
		// foreign key; the audit log keeps the history once the role is gone.
		if err := q.DeleteInactiveRoleAssignments(ctx, id); err != nil {
			return fmt.Errorf("roles: delete inactive assignments: %w", err)
		}

		if err := q.DeleteRole(ctx, id); err != nil {
			if isForeignKeyViolation(err) {
				return apperr.Conflict("Role ini masih dimiliki pengguna.")
			}
			return fmt.Errorf("roles: delete role: %w", err)
		}

		return s.recordAudit(ctx, q, a, "role.delete", targetTypeRole, id, nil, map[string]any{
			"slug": role.Slug,
		})
	})
}

// ensurePermissionsExist rejects unknown permission keys.
func (s *Service) ensurePermissionsExist(ctx context.Context, q *store.Queries, permissionKeys []string) error {
	if len(permissionKeys) == 0 {
		return nil
	}

	catalog, err := q.ListPermissions(ctx)
	if err != nil {
		return fmt.Errorf("roles: list permissions: %w", err)
	}
	known := make(map[string]struct{}, len(catalog))
	for _, permission := range catalog {
		known[permission.Key] = struct{}{}
	}

	for _, permission := range permissionKeys {
		if _, exists := known[permission]; !exists {
			return apperr.Validation(apperr.Field(
				"permissions", "not_found", "Izin tidak ditemukan.",
			))
		}
	}
	return nil
}

func sameOptional(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
