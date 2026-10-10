package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func (a *App) handleServeMedia(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m, err := a.Q.GetMediaByID(r.Context(), mustUUID(id))
	if err != nil || m.Status == "deleted" {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Media tidak ditemukan.", r))
		return
	}
	publicPurpose := m.Purpose == "avatar" || m.Purpose == "banner" || m.Purpose == "changelog_cover"
	if !publicPurpose {
		used, _ := a.Q.MediaUsedInPublishedPortfolio(r.Context(), m.ID)
		ownerView := UserID(r) != "" && (uuidStr(m.OwnerID) == UserID(r) || UserRole(r) == "admin" || UserPerms(r)["portfolios.moderate"])
		if !used && !ownerView {
			if UserID(r) == "" {
				writeJSON(w, http.StatusUnauthorized, fail("UNAUTHORIZED", "Sesi berakhir.", r))
				return
			}
			writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Bukan milikmu.", r))
			return
		}
	}
	// Varian: ?w=320|800 memakai file yang sudah dibuat saat upload.
	key := m.ObjectKey
	if wq := r.URL.Query().Get("w"); wq != "" {
		var vs map[string]string
		if err := json.Unmarshal(m.Variants, &vs); err == nil {
			if vk, ok := vs[wq]; ok {
				key = vk
			}
		}
	}
	body, err := a.Storage.Get(r.Context(), key)
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "File tidak ditemukan.", r))
		return
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal membaca file.", r))
		return
	}
	w.Header().Set("Content-Type", m.Mime)
	w.Header().Set("ETag", `"`+m.ChecksumSha256+`"`)
	if publicPurpose {
		// Konten publik boleh di-cache lama; konten privat tidak.
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "private, max-age=300")
	}
	http.ServeContent(w, r, key, time.Now(), bytes.NewReader(data))
}
