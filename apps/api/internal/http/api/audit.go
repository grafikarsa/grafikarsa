package api

import (
	"context"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/audit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
)

// auditScopeFor maps the principal's audit.view grant to a domain scope.
// A principal without the grant yields a scope that the domain reports as 404.
func auditScopeFor(p *auth.Principal) audit.Scope {
	if p == nil {
		return audit.Scope{}
	}
	for _, grant := range p.Grants {
		if grant.Permission != audit.PermissionView {
			continue
		}
		if grant.Global {
			return audit.Scope{Granted: true, Global: true}
		}
		return audit.Scope{
			Granted:  true,
			MajorIDs: append([]uuid.UUID(nil), grant.MajorIDs...),
		}
	}
	return audit.Scope{}
}

// toAuditLogEntry maps a domain audit item to its API representation.
func toAuditLogEntry(item audit.Item) apigen.AuditLogEntry {
	return apigen.AuditLogEntry{
		Id:            item.ID,
		OccurredAt:    item.OccurredAt,
		ActorId:       toOptionalOpenAPIUUID(item.ActorID),
		ActorName:     item.ActorName,
		ActorUsername: item.ActorUsername,
		ActorIp:       item.ActorIP,
		Action:        item.Action,
		TargetType:    item.TargetType,
		TargetId:      toOptionalOpenAPIUUID(item.TargetID),
		ScopeMajorId:  toOptionalOpenAPIUUID(item.ScopeMajorID),
		Data:          item.Data,
	}
}

// toOptionalOpenAPIUUID converts an optional domain UUID to the API type.
func toOptionalOpenAPIUUID(value *uuid.UUID) *openapi_types.UUID {
	if value == nil {
		return nil
	}
	converted := openapi_types.UUID(*value)
	return &converted
}

// GetAdminAuditLogs returns the audit entries visible to the authenticated
// actor, newest first.
func (s *Server) GetAdminAuditLogs(ctx context.Context, request apigen.GetAdminAuditLogsRequestObject) (apigen.GetAdminAuditLogsResponseObject, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}

	filter := audit.Filter{
		Action:       derefString(request.Params.Action),
		ActionPrefix: derefString(request.Params.ActionPrefix),
		TargetType:   derefString(request.Params.TargetType),
		Cursor:       derefString(request.Params.Cursor),
	}
	if request.Params.ActorId != nil {
		id := uuid.UUID(*request.Params.ActorId)
		filter.ActorID = &id
	}
	if request.Params.TargetId != nil {
		id := uuid.UUID(*request.Params.TargetId)
		filter.TargetID = &id
	}
	if request.Params.MajorId != nil {
		id := uuid.UUID(*request.Params.MajorId)
		filter.ScopeMajorID = &id
	}
	filter.From = request.Params.From
	filter.To = request.Params.To
	if request.Params.Limit != nil {
		filter.Limit = *request.Params.Limit
	}

	page, err := s.audit.ListForActor(ctx, auditScopeFor(principal), filter)
	if err != nil {
		return nil, err
	}

	items := make([]apigen.AuditLogEntry, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, toAuditLogEntry(item))
	}

	var nextCursor *string
	if page.NextCursor != "" {
		nextCursor = &page.NextCursor
	}

	return apigen.GetAdminAuditLogs200JSONResponse{Items: items, NextCursor: nextCursor}, nil
}
