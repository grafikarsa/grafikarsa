package http

import (
	"context"
)

// ownsEntity memeriksa apakah entitas induk (portfolio atau thread feedback)
// milik user yang sedang mengunggah.
//
// Object key memakai path per-entity (PRD §7.1), jadi sebelum entity_id
// dipakai di path, harus dipastikan pemegangnya memang pemilik entitas itu.
// Tanpa pengecekan ini, user bisa menaruh objek media di folder portfolio
// orang lain — lalu ikut terhapus saat portfolio itu dihapus, atau serves
// sebagai media milik orang lain.
//
// Purpose yang tidak entity-scoped (avatar, banner) selalu milik user, jadi
// pemanggil yang tidak mengirim entity_id sudah otomatis aman.
func (a *App) ownsEntity(ctx context.Context, purpose, entityID, userID string) bool {
	switch purpose {
	case "thumbnail", "block_image", "document":
		p, err := a.Q.GetPortfolioByID(ctx, mustUUID(entityID))
		if err != nil {
			return false
		}
		return uuidStr(p.OwnerID) == userID
	case "feedback":
		th, err := a.Q.GetFeedbackThread(ctx, mustUUID(entityID))
		if err != nil {
			return false
		}
		// Pelapor thread boleh melampirkan lampiran ke thread-nya sendiri.
		return uuidStr(th.ReporterID) == userID
	default:
		// Purpose lain tidak punya entitas induk; entity_id diabaikan.
		return false
	}
}
