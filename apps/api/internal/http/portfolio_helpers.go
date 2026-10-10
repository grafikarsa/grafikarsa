package http

import (
	"encoding/json"
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
	db, _ := json.Marshal(map[string]string{"note": diff})
	_ = a.Q.InsertAuditLog(r.Context(), repo.InsertAuditLogParams{
		ActorID: actor, Action: action, Entity: entity, EntityID: entityID,
		Diff: db, Ip: toPgInet(clientIP(r)),
	})
}

func buildSnapshot(r *http.Request, a *App, p repo.GetPortfolioByIDRow) []byte {
	blocks, _ := a.Q.ListBlocks(r.Context(), p.ID)
	tags, _ := a.Q.ListPortfolioTags(r.Context(), p.ID)
	tagIDs := []string{}
	for _, t := range tags {
		tagIDs = append(tagIDs, uuidStr(t.ID))
	}
	snap := map[string]any{
		"title": p.Title, "tag_ids": tagIDs, "series_id": nullableUUID(p.SeriesID),
		"thumbnail_media_id": nullableUUID(p.ThumbnailMediaID),
		"blocks":             blocks,
	}
	b, _ := json.Marshal(snap)
	return b
}
