package http

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"grafikarsa/api/internal/repo"
)

func isOwnerOrAdmin(r *http.Request, ownerID pgtype.UUID) bool {
	uid := UserID(r)
	return uid != "" && (uid == uuidStr(ownerID) || UserRole(r) == "admin")
}

func nilUUID() pgtype.UUID { return pgtype.UUID{Valid: false} }

func nullableUUID(v pgtype.UUID) any {
	if !v.Valid {
		return nil
	}
	return uuidStr(v)
}

func auditLog(r *http.Request, a *App, action, entity, entityID, diff string) {
	var actor pgtype.UUID
	if uid := UserID(r); uid != "" {
		actor = mustUUID(uid)
	}
	_ = a.Q.InsertAuditLog(r.Context(), repo.InsertAuditLogParams{
		ActorID: actor, Action: action, Entity: entity, EntityID: entityID,
		Diff: []byte(diff), Ip: toPgInet(clientIP(r)),
	})
}
