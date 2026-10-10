package http

import (
	"encoding/json"
	"net/http"
	neturl "net/url"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"grafikarsa/api/internal/repo"
)

var allowedBlockTypes = map[string]bool{
	"text": true, "image": true, "table": true, "youtube": true,
	"button": true, "embed": true, "figma": true, "canva": true,
	"ppt": true, "pdf": true, "doc": true, "link": true,
}

var allowedEmbedDomains = []string{
	"codepen.io", "codesandbox.io", "youtube-nocookie.com", "figma.com",
	"canva.com", "docs.google.com", "drive.google.com", "slides.google.com",
}

var youtubeRe = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// payloadMediaID mengambil media_id dari payload block (bila ada).
func payloadMediaID(payload map[string]any) string {
	v, _ := payload["media_id"].(string)
	return v
}

func validateBlockPayload(blockType string, payload map[string]any) (string, bool) {
	switch blockType {
	case "text":
		if s, _ := payload["markdown"].(string); len(s) >= 1 && len(s) <= 20000 {
			return "", true
		}
		return "markdown 1-20000 karakter.", false
	case "image":
		mid, _ := payload["media_id"].(string)
		url, _ := payload["url"].(string)
		// media_id (kanonis) ATAU url https (kompatibilitas hasil migrasi v1),
		// sesuai packages/contracts/blocks/image.json.
		if mid == "" && !strings.HasPrefix(url, "https://") {
			return "media_id atau url https wajib.", false
		}
		alt, _ := payload["alt"].(string)
		if len(alt) < 5 || len(alt) > 200 {
			return "alt wajib 5-200 karakter.", false
		}
		return "", true
	case "table":
		h, _ := payload["headers"].([]any)
		rows, _ := payload["rows"].([]any)
		if len(h) < 1 || len(h) > 10 || len(rows) > 50 {
			return "headers 1-10, rows maks 50.", false
		}
		return "", true
	case "youtube":
		vid, _ := payload["video_id"].(string)
		if !youtubeRe.MatchString(vid) {
			return "video_id harus 11 karakter YouTube.", false
		}
		return "", true
	case "button":
		label, _ := payload["label"].(string)
		url, _ := payload["url"].(string)
		if len(label) < 1 || len(label) > 50 || !strings.HasPrefix(url, "https://") {
			return "label 1-50 + url https.", false
		}
		return "", true
	case "pdf":
		// PDF boleh dari URL https mana pun (file, bukan iframe) atau media
		// yang diunggah; tidak terkena allowlist domain embed.
		mid, _ := payload["media_id"].(string)
		url, _ := payload["url"].(string)
		if mid == "" && !strings.HasPrefix(url, "https://") {
			return "media_id atau url https wajib.", false
		}
		return "", true
	case "embed", "figma", "canva", "ppt", "doc":
		url, _ := payload["url"].(string)
		if !strings.HasPrefix(url, "https://") {
			return "url harus https.", false
		}
		// Bandingkan HOSTNAME persis (atau subdomain), bukan substring:
		// 'https://evil.com/?x=figma.com' tidak boleh lolos.
		parsed, perr := neturl.Parse(url)
		if perr != nil || parsed.Host == "" {
			return "url tidak valid.", false
		}
		host := strings.ToLower(parsed.Hostname())
		allowed := false
		for _, d := range allowedEmbedDomains {
			if host == d || strings.HasSuffix(host, "."+d) {
				allowed = true
				break
			}
		}
		if !allowed {
			return "domain embed tidak diizinkan.", false
		}
		return "", true
	case "link":
		url, _ := payload["url"].(string)
		if !strings.HasPrefix(url, "https://") {
			return "url harus https.", false
		}
		return "", true
	}
	return "tipe tidak dikenal.", false
}

func (a *App) mustOwnPortfolio(w http.ResponseWriter, r *http.Request) (repo.GetPortfolioByIDRow, bool) {
	pid := chi.URLParam(r, "portfolio_id")
	if pid == "" {
		pid = chi.URLParam(r, "id")
	}
	p, err := a.Q.GetPortfolioByID(r.Context(), mustUUID(pid))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Portfolio tidak ditemukan.", r))
		return p, false
	}
	if !isOwnerOrAdmin(r, p.OwnerID) {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Bukan milikmu.", r))
		return p, false
	}
	if p.Status == "pending_review" {
		writeJSON(w, http.StatusUnprocessableEntity, fail("INVALID_STATUS_TRANSITION", "Portfolio sedang direview.", r))
		return p, false
	}
	return p, true
}

func (a *App) handleCreateBlock(w http.ResponseWriter, r *http.Request) {
	p, owned := a.mustOwnPortfolio(w, r)
	if !owned {
		return
	}
	var req struct {
		BlockType string         `json:"block_type"`
		Payload   map[string]any `json:"payload"`
		MediaID   *string        `json:"media_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if !allowedBlockTypes[req.BlockType] {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Tipe block tidak valid.", r))
		return
	}
	if req.Payload == nil {
		req.Payload = map[string]any{}
	}
	if msg, valid := validateBlockPayload(req.BlockType, req.Payload); !valid {
		writeJSON(w, http.StatusUnprocessableEntity, failDetails("VALIDATION_ERROR", "Payload tidak valid.", []map[string]string{{"field": "payload", "message": msg}}, r))
		return
	}
	// Media di dalam payload (image/gallery) juga harus milik sendiri.
	if mid := payloadMediaID(req.Payload); mid != "" {
		if !a.ownsMedia(r.Context(), mustUUID(mid), UserID(r)) {
			writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Media di payload bukan milikmu.", r))
			return
		}
		// Ditandai confirmed agar tidak dihapus job cleanup (media pending).
		_ = a.Q.ConfirmMedia(r.Context(), mustUUID(mid))
	}
	maxPos, _ := a.Q.MaxBlockPosition(r.Context(), p.ID)
	var assetID = nilUUID()
	if req.MediaID != nil && *req.MediaID != "" {
		assetID = mustUUID(*req.MediaID)
		if !a.ownsMedia(r.Context(), assetID, UserID(r)) {
			writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Media bukan milikmu.", r))
			return
		}
		_ = a.Q.ConfirmMedia(r.Context(), assetID)
	}
	payloadBytes := toJSONBytes(req.Payload)
	blockID, err := a.Q.CreateBlock(r.Context(), repo.CreateBlockParams{
		PortfolioID: p.ID, BlockType: req.BlockType, Position: maxPos + 1,
		Payload: payloadBytes, AssetMediaID: assetID,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal menambah block.", r))
		return
	}
	writeJSON(w, http.StatusCreated, ok(map[string]string{"id": uuidStr(blockID)}, r))
}

func (a *App) handleUpdateBlock(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "blockId")
	b, err := a.Q.GetBlock(r.Context(), mustUUID(bid))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Block tidak ditemukan.", r))
		return
	}
	p, err := a.Q.GetPortfolioByID(r.Context(), b.PortfolioID)
	if err != nil || !isOwnerOrAdmin(r, p.OwnerID) || p.Status == "pending_review" {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Tidak bisa edit block ini.", r))
		return
	}
	var req struct {
		Payload map[string]any `json:"payload"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if msg, valid := validateBlockPayload(b.BlockType, req.Payload); !valid {
		writeJSON(w, http.StatusUnprocessableEntity, failDetails("VALIDATION_ERROR", "Payload tidak valid.", []map[string]string{{"field": "payload", "message": msg}}, r))
		return
	}
	if mid := payloadMediaID(req.Payload); mid != "" {
		if !a.ownsMedia(r.Context(), mustUUID(mid), UserID(r)) {
			writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Media di payload bukan milikmu.", r))
			return
		}
		_ = a.Q.ConfirmMedia(r.Context(), mustUUID(mid))
	}
	_ = a.Q.UpdateBlockPayload(r.Context(), repo.UpdateBlockPayloadParams{ID: b.ID, Payload: toJSONBytes(req.Payload)})
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleDeleteBlock(w http.ResponseWriter, r *http.Request) {
	bid := chi.URLParam(r, "blockId")
	b, err := a.Q.GetBlock(r.Context(), mustUUID(bid))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Block tidak ditemukan.", r))
		return
	}
	p, err := a.Q.GetPortfolioByID(r.Context(), b.PortfolioID)
	if err != nil || !isOwnerOrAdmin(r, p.OwnerID) || p.Status == "pending_review" {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Tidak bisa hapus block ini.", r))
		return
	}
	if b.IsFromSeries {
		auditLog(r, a, "block.delete_series", "portfolio_blocks", bid, "")
	}
	_ = a.Q.DeleteBlock(r.Context(), b.ID)
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "deleted"}, r))
}

func (a *App) handleReorderBlocks(w http.ResponseWriter, r *http.Request) {
	p, owned := a.mustOwnPortfolio(w, r)
	if !owned {
		return
	}
	var req struct {
		Orders []struct {
			ID       string `json:"id"`
			Position int32  `json:"position"`
		} `json:"orders"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	for _, o := range req.Orders {
		b, err := a.Q.GetBlock(r.Context(), mustUUID(o.ID))
		if err != nil || uuidStr(b.PortfolioID) != uuidStr(p.ID) {
			writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Block tidak valid.", r))
			return
		}
		_ = a.Q.UpdateBlockPosition(r.Context(), repo.UpdateBlockPositionParams{ID: b.ID, Position: o.Position})
	}
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleListTags(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := pageParams(r)
	rows, err := a.Q.ListTags(r.Context(), repo.ListTagsParams{Limit: limit, Offset: offset})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat tags.", r))
		return
	}
	out := []map[string]any{}
	for _, t := range rows {
		out = append(out, map[string]any{"id": uuidStr(t.ID), "slug": t.Slug, "name": t.Name})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, int64(len(out))), r))
}

func (a *App) handleListSeries(w http.ResponseWriter, r *http.Request) {
	limit, offset, page := pageParams(r)
	rows, err := a.Q.ListSeries(r.Context(), repo.ListSeriesParams{Limit: limit, Offset: offset})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal memuat series.", r))
		return
	}
	out := []map[string]any{}
	for _, s := range rows {
		out = append(out, map[string]any{"id": uuidStr(s.ID), "slug": s.Slug, "name": s.Name})
	}
	if out == nil {
		out = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, okMeta(out, meta(page, limit, int64(len(out))), r))
}

func (a *App) handleGetSeries(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s, err := a.Q.GetSeries(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Series tidak ditemukan.", r))
		return
	}
	blocks, _ := a.Q.ListSeriesBlocks(r.Context(), s.ID)
	bout := []map[string]any{}
	for _, b := range blocks {
		bout = append(bout, map[string]any{"block_type": b.BlockType, "position": b.Position, "instruction": b.Instruction})
	}
	if bout == nil {
		bout = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, ok(map[string]any{
		"id": s.ID, "slug": s.Slug, "name": s.Name, "description": s.Description, "blocks": bout,
	}, r))
}

func toJSONBytes(m map[string]any) []byte {
	b, _ := json.Marshal(m)
	if b == nil {
		return []byte("{}")
	}
	return b
}

var _ = uuid.NewString
