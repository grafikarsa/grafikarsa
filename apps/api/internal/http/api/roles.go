package api

import (
	"context"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/roles"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/authctx"
)

// rolesActor builds the service actor for an authenticated role admin request.
func (s *Server) rolesActor(ctx context.Context) (roles.Actor, error) {
	p, err := requirePrincipal(ctx)
	if err != nil {
		return roles.Actor{}, err
	}
	return roles.Actor{Principal: p, IP: authctx.MetaFrom(ctx).IP}, nil
}

// toAdminPermission maps a catalog entry to its API representation.
func toAdminPermission(p roles.Permission) apigen.AdminPermission {
	return apigen.AdminPermission{
		Key:         p.Key,
		Description: p.Description,
		Scopable:    p.Scopable,
	}
}

// toAdminRole maps a role definition to its API representation.
func toAdminRole(r roles.Role) apigen.AdminRole {
	permissions := make([]string, len(r.Permissions))
	copy(permissions, r.Permissions)

	return apigen.AdminRole{
		Id:          r.ID,
		Slug:        r.Slug,
		Name:        r.Name,
		Description: r.Description,
		IsSystem:    r.IsSystem,
		Permissions: permissions,
		HolderCount: r.HolderCount,
	}
}

// toAdminRoleAssignment maps a role assignment to its API representation.
func toAdminRoleAssignment(a roles.Assignment) apigen.AdminRoleAssignment {
	assignedAt := a.AssignedAt

	return apigen.AdminRoleAssignment{
		Id:         a.ID,
		RoleSlug:   a.RoleSlug,
		RoleName:   a.RoleName,
		MajorId:    a.MajorID,
		ExpiresAt:  a.ExpiresAt,
		AssignedAt: &assignedAt,
	}
}

// GetAdminPermissions returns the permission catalog.
func (s *Server) GetAdminPermissions(ctx context.Context, _ apigen.GetAdminPermissionsRequestObject) (apigen.GetAdminPermissionsResponseObject, error) {
	actor, err := s.rolesActor(ctx)
	if err != nil {
		return nil, err
	}

	permissions, err := s.roles.ListPermissions(ctx, actor)
	if err != nil {
		return nil, err
	}

	items := make([]apigen.AdminPermission, 0, len(permissions))
	for _, permission := range permissions {
		items = append(items, toAdminPermission(permission))
	}

	return apigen.GetAdminPermissions200JSONResponse{Items: items}, nil
}

// GetAdminRoles returns every role definition.
func (s *Server) GetAdminRoles(ctx context.Context, _ apigen.GetAdminRolesRequestObject) (apigen.GetAdminRolesResponseObject, error) {
	actor, err := s.rolesActor(ctx)
	if err != nil {
		return nil, err
	}

	roleList, err := s.roles.ListRoles(ctx, actor)
	if err != nil {
		return nil, err
	}

	items := make([]apigen.AdminRole, 0, len(roleList))
	for _, role := range roleList {
		items = append(items, toAdminRole(role))
	}

	return apigen.GetAdminRoles200JSONResponse{Items: items}, nil
}

// PostAdminRole creates a custom role.
func (s *Server) PostAdminRole(ctx context.Context, request apigen.PostAdminRoleRequestObject) (apigen.PostAdminRoleResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	actor, err := s.rolesActor(ctx)
	if err != nil {
		return nil, err
	}

	created, err := s.roles.CreateRole(
		ctx,
		actor,
		request.Body.Slug,
		request.Body.Name,
		request.Body.Description,
		request.Body.Permissions,
	)
	if err != nil {
		return nil, err
	}

	return apigen.PostAdminRole201JSONResponse(toAdminRole(created)), nil
}

// PatchAdminRole changes the name and description of a role.
func (s *Server) PatchAdminRole(ctx context.Context, request apigen.PatchAdminRoleRequestObject) (apigen.PatchAdminRoleResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	actor, err := s.rolesActor(ctx)
	if err != nil {
		return nil, err
	}

	updated, err := s.roles.UpdateRole(
		ctx,
		actor,
		uuid.UUID(request.Id),
		request.Body.Name,
		request.Body.Description,
	)
	if err != nil {
		return nil, err
	}

	return apigen.PatchAdminRole200JSONResponse(toAdminRole(updated)), nil
}

// PutAdminRolePermissions replaces the permission set of a custom role.
func (s *Server) PutAdminRolePermissions(ctx context.Context, request apigen.PutAdminRolePermissionsRequestObject) (apigen.PutAdminRolePermissionsResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	actor, err := s.rolesActor(ctx)
	if err != nil {
		return nil, err
	}

	updated, err := s.roles.SetRolePermissions(ctx, actor, uuid.UUID(request.Id), request.Body.Permissions)
	if err != nil {
		return nil, err
	}

	return apigen.PutAdminRolePermissions200JSONResponse(toAdminRole(updated)), nil
}

// DeleteAdminRole removes a custom role that has no active holders.
func (s *Server) DeleteAdminRole(ctx context.Context, request apigen.DeleteAdminRoleRequestObject) (apigen.DeleteAdminRoleResponseObject, error) {
	actor, err := s.rolesActor(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.roles.DeleteRole(ctx, actor, uuid.UUID(request.Id)); err != nil {
		return nil, err
	}

	return apigen.DeleteAdminRole204Response{}, nil
}

// GetAdminUserRoles lists the active role assignments of one user.
func (s *Server) GetAdminUserRoles(ctx context.Context, request apigen.GetAdminUserRolesRequestObject) (apigen.GetAdminUserRolesResponseObject, error) {
	actor, err := s.rolesActor(ctx)
	if err != nil {
		return nil, err
	}

	assignments, err := s.roles.ListAssignments(ctx, actor, uuid.UUID(request.Id))
	if err != nil {
		return nil, err
	}

	items := make([]apigen.AdminRoleAssignment, 0, len(assignments))
	for _, assignment := range assignments {
		items = append(items, toAdminRoleAssignment(assignment))
	}

	return apigen.GetAdminUserRoles200JSONResponse{Items: items}, nil
}

// PostAdminUserRole grants a role to one user, globally or scoped to a major.
func (s *Server) PostAdminUserRole(ctx context.Context, request apigen.PostAdminUserRoleRequestObject) (apigen.PostAdminUserRoleResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	actor, err := s.rolesActor(ctx)
	if err != nil {
		return nil, err
	}

	var majorID *uuid.UUID
	if request.Body.MajorId != nil {
		id := uuid.UUID(*request.Body.MajorId)
		majorID = &id
	}

	assignment, err := s.roles.Assign(
		ctx,
		actor,
		uuid.UUID(request.Id),
		request.Body.RoleSlug,
		majorID,
		request.Body.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}

	return apigen.PostAdminUserRole201JSONResponse(toAdminRoleAssignment(assignment)), nil
}

// DeleteAdminUserRole revokes one role assignment.
func (s *Server) DeleteAdminUserRole(ctx context.Context, request apigen.DeleteAdminUserRoleRequestObject) (apigen.DeleteAdminUserRoleResponseObject, error) {
	actor, err := s.rolesActor(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.roles.Revoke(ctx, actor, uuid.UUID(request.AssignmentId)); err != nil {
		return nil, err
	}

	return apigen.DeleteAdminUserRole204Response{}, nil
}
