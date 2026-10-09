package http

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/h2non/filetype"
	"grafikarsa/api/internal/repo"
	"grafikarsa/api/internal/storage"
)

var purposeLimits = map[string]int64{
	"avatar":          2 << 20,
	"banner":          10 << 20,
	"thumbnail":       5 << 20,
	"block_image":     10 << 20,
	"document":        20 << 20,
	"feedback":        5 << 20,
	"changelog_cover": 5 << 20,
}

var purposeMimes = map[string][]string{
	"avatar":          {"image/jpeg", "image/png", "image/webp"},
	"banner":          {"image/jpeg", "image/png", "image/webp"},
	"thumbnail":       {"image/jpeg", "image/png", "image/webp"},
	"block_image":     {"image/jpeg", "image/png", "image/webp", "image/gif"},
	"document":        {"application/pdf"},
	"feedback":        {"image/jpeg", "image/png", "image/webp", "application/pdf"},
	"changelog_cover": {"image/jpeg", "image/png", "image/webp"},
}

func (a *App) handleUpload(w http.ResponseWriter, r *http.Request) {
	uid := UserID(r)
	r.Body = http.MaxBytesReader(w, r.Body, 52<<20)
	if err := r.ParseMultipartForm(52 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, fail("FILE_TOO_LARGE", "File terlalu besar (maks 52MB).", r))
		return
	}
	purpose := r.FormValue("purpose")
	limit, found := purposeLimits[purpose]
	if !found {
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "Purpose tidak valid.", r))
		return
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, fail("VALIDATION_ERROR", "Field file wajib ada.", r))
		return
	}
	defer f.Close()
	if hdr.Size > limit {
		writeJSON(w, http.StatusUnprocessableEntity, fail("FILE_TOO_LARGE", fmt.Sprintf("Maksimal %dMB.", limit>>20), r))
		return
	}
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	kind, _ := filetype.Match(head)
	mime := hdr.Header.Get("Content-Type")
	if kind != filetype.Unknown {
		mime = kind.MIME.Value
	}
	// filetype tidak kenal SVG teks — deteksi manual via magic bytes.
	lower := strings.ToLower(string(head))
	if strings.Contains(lower, "<svg") || strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype html") {
		mime = "image/svg+xml"
	}
	allowed := false
	for _, m := range purposeMimes[purpose] {
		if mime == m {
			allowed = true
		}
	}
	if !allowed || strings.Contains(mime, "svg") || strings.Contains(mime, "html") {
		writeJSON(w, http.StatusUnprocessableEntity, fail("INVALID_CONTENT_TYPE", "Tipe file tidak diizinkan: "+mime, r))
		return
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal membaca file.", r))
		return
	}
	data, err := io.ReadAll(f)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal membaca file.", r))
		return
	}
	sum := sha256.Sum256(data)
	checksum := hex.EncodeToString(sum[:])
	ext := ".bin"
	switch mime {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	case "image/gif":
		ext = ".gif"
	case "application/pdf":
		ext = ".pdf"
	}
	key := fmt.Sprintf("%ss/%s/%s%s", purpose, uid, uuid.NewString()[:8], ext)
	if purpose == "avatar" || purpose == "banner" {
		key = fmt.Sprintf("%ss/%s/%s%s", purpose, uid, uuid.NewString(), ext)
	}
	if err := a.Storage.Put(r.Context(), key, mime, bytes.NewReader(data)); err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal menyimpan file.", r))
		return
	}
	mediaID, err := a.Q.CreateMedia(r.Context(), repo.CreateMediaParams{
		OwnerID:        mustUUID(uid),
		Purpose:        purpose,
		ObjectKey:      key,
		Mime:           mime,
		SizeBytes:      int64(len(data)),
		ChecksumSha256: checksum,
		Status:         "confirmed",
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal mencatat file.", r))
		return
	}
	writeJSON(w, http.StatusCreated, ok(map[string]any{"media_id": uuidStr(mediaID), "object_key": key}, r))
}

func (a *App) handleConfirmMedia(w http.ResponseWriter, r *http.Request) {
	uid := UserID(r)
	id := chi.URLParam(r, "id")
	m, err := a.Q.GetMediaByID(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Media tidak ditemukan.", r))
		return
	}
	if uuidStr(m.OwnerID) != uid && UserRole(r) != "admin" {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Bukan milikmu.", r))
		return
	}
	var req struct {
		LinkTo string `json:"link_to"`
	}
	_ = req
	switch m.Purpose {
	case "avatar":
		_ = a.Q.UpdateUserAvatar(r.Context(), repo.UpdateUserAvatarParams{ID: mustUUID(uid), AvatarMediaID: m.ID})
	case "banner":
		_ = a.Q.UpdateUserBanner(r.Context(), repo.UpdateUserBannerParams{ID: mustUUID(uid), BannerMediaID: m.ID})
	}
	_ = a.Q.ConfirmMedia(r.Context(), m.ID)
	invalidatePerms(r.Context(), a.Cache, uid)
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
}

func (a *App) handleDeleteMedia(w http.ResponseWriter, r *http.Request) {
	uid := UserID(r)
	id := chi.URLParam(r, "id")
	m, err := a.Q.GetMediaByID(r.Context(), mustUUID(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Media tidak ditemukan.", r))
		return
	}
	if uuidStr(m.OwnerID) != uid && !UserPerms(r)["media.delete_any"] {
		writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Bukan milikmu.", r))
		return
	}
	_ = a.Q.MarkMediaDeleted(r.Context(), m.ID)
	go func() {
		time.Sleep(100 * time.Millisecond)
		_ = storage.Noop()
	}()
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "deleted"}, r))
}

func (a *App) handleServeMedia(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m, err := a.Q.GetMediaByID(r.Context(), mustUUID(id))
	if err != nil || m.Status != "confirmed" {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Media tidak ditemukan.", r))
		return
	}
	if m.Purpose != "avatar" && m.Purpose != "banner" {
		if UserID(r) == "" {
			writeJSON(w, http.StatusUnauthorized, fail("UNAUTHORIZED", "Sesi berakhir.", r))
			return
		}
		if uuidStr(m.OwnerID) != UserID(r) && UserRole(r) != "admin" {
			writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Bukan milikmu.", r))
			return
		}
	}
	body, err := a.Storage.Get(r.Context(), m.ObjectKey)
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
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeContent(w, r, m.ObjectKey, time.Now(), bytes.NewReader(data))
}
