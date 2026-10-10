package http

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/h2non/filetype"
	"github.com/jackc/pgx/v5/pgtype"
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

// imagePolicy: lebar maksimum + varian yang dibuat per purpose.
type imagePolicy struct {
	maxWidth int
	variants []int
}

var purposeImages = map[string]imagePolicy{
	"avatar":      {maxWidth: 800, variants: []int{320}},
	"banner":      {maxWidth: 1500, variants: []int{800}},
	"thumbnail":   {maxWidth: 1200, variants: []int{320, 800}},
	"block_image": {maxWidth: 1600, variants: []int{320, 800}},
}

func (a *App) handleUpload(w http.ResponseWriter, r *http.Request) {
	uid := UserID(r)
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, 52<<20)
	if err := r.ParseMultipartForm(52 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, fail("FILE_TOO_LARGE", "File terlalu besar (maks 52MB).", r))
		return
	}
	if n, _ := a.Cache.IncrWithExpire(ctx, "upload:hour:"+uid, time.Hour); n > 20 {
		writeJSON(w, http.StatusTooManyRequests, fail("TOO_MANY_UPLOADS", "Maksimal 20 file per jam.", r))
		return
	}
	if n, _ := a.Cache.IncrWithExpire(ctx, "upload:day:"+uid, 24*time.Hour); n > 100 {
		writeJSON(w, http.StatusTooManyRequests, fail("TOO_MANY_UPLOADS", "Maksimal 100 file per hari.", r))
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
	} else {
		// Magic bytes tidak dikenal: JANGAN percaya header klien.
		writeJSON(w, http.StatusUnprocessableEntity, fail("INVALID_CONTENT_TYPE", "Tipe file tidak dikenali.", r))
		return
	}
	lower := strings.ToLower(string(head))
	if strings.Contains(lower, "<svg") || strings.Contains(lower, "<html") || strings.Contains(lower, "<!doctype html") {
		writeJSON(w, http.StatusUnprocessableEntity, fail("INVALID_CONTENT_TYPE", "SVG/HTML tidak diizinkan.", r))
		return
	}
	allowed := false
	for _, m := range purposeMimes[purpose] {
		if mime == m {
			allowed = true
		}
	}
	if !allowed {
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

	// Gambar: buang EXIF (re-encode) + siapkan varian.
	var variants map[int][]byte
	width, height := 0, 0
	if policy, isImage := purposeImages[purpose]; isImage {
		proc, perr := storage.ProcessImage(data, policy.maxWidth, policy.variants)
		if perr != nil {
			writeJSON(w, http.StatusUnprocessableEntity, fail("INVALID_CONTENT_TYPE", "Gambar tidak bisa diproses.", r))
			return
		}
		data = proc.Data
		mime = proc.Mime
		width, height = proc.Width, proc.Height
		variants = proc.Variants
	}

	sum := sha256.Sum256(data)
	checksum := hex.EncodeToString(sum[:])

	// Dedup: file identik milik user yang sama untuk purpose sama -> reuse.
	if existing, derr := a.Q.FindMediaByChecksum(ctx, repo.FindMediaByChecksumParams{
		OwnerID: mustUUID(uid), Purpose: purpose, ChecksumSha256: checksum,
	}); derr == nil {
		writeJSON(w, http.StatusOK, ok(map[string]any{
			"media_id": uuidStr(existing.ID), "object_key": existing.ObjectKey, "deduplicated": true,
		}, r))
		return
	}

	// Quota 200MB (hitung file baru agar tidak overshoot).
	used, _ := a.Q.SumUserMedia(ctx, mustUUID(uid))
	if toInt64(used)+int64(len(data)) > 200<<20 {
		writeJSON(w, http.StatusUnprocessableEntity, fail("QUOTA_EXCEEDED", "Kuota 200MB habis.", r))
		return
	}

	ext := extForMime(mime)
	base := fmt.Sprintf("%ss/%s/%s", purpose, uid, uuid.NewString())
	key := base + ext
	if err := a.Storage.Put(ctx, key, mime, bytes.NewReader(data)); err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal menyimpan file.", r))
		return
	}
	// Varian disimpan di key berakhiran _<lebar><ext>.
	variantKeys := map[string]string{}
	for wpx, vdata := range variants {
		vkey := fmt.Sprintf("%s_%d%s", base, wpx, ext)
		if err := a.Storage.Put(ctx, vkey, mime, bytes.NewReader(vdata)); err != nil {
			continue
		}
		variantKeys[strconv.Itoa(wpx)] = vkey
	}
	vb, _ := json.Marshal(variantKeys)

	mediaID, err := a.Q.CreateMedia(ctx, repo.CreateMediaParams{
		OwnerID:        mustUUID(uid),
		Purpose:        purpose,
		ObjectKey:      key,
		Mime:           mime,
		SizeBytes:      int64(len(data)),
		Width:          toPgInt4(width),
		Height:         toPgInt4(height),
		ChecksumSha256: checksum,
		Status:         "pending",
		Variants:       vb,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, fail("INTERNAL_ERROR", "Gagal mencatat file.", r))
		return
	}
	writeJSON(w, http.StatusCreated, ok(map[string]any{
		"media_id": uuidStr(mediaID), "object_key": key, "width": width, "height": height,
	}, r))
}

func extForMime(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "application/pdf":
		return ".pdf"
	}
	return ".bin"
}

func toPgInt4(v int) pgtype.Int4 {
	if v <= 0 {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(v), Valid: true}
}

// handleConfirmMedia: kaitkan media ke entitas (avatar/banner/thumbnail/block).
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
		LinkTo      string `json:"link_to"`
		PortfolioID string `json:"portfolio_id"`
		BlockID     string `json:"block_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	switch req.LinkTo {
	case "avatar":
		_ = a.Q.UpdateUserAvatar(r.Context(), repo.UpdateUserAvatarParams{ID: mustUUID(uid), AvatarMediaID: m.ID})
	case "banner":
		_ = a.Q.UpdateUserBanner(r.Context(), repo.UpdateUserBannerParams{ID: mustUUID(uid), BannerMediaID: m.ID})
	case "thumbnail":
		if req.PortfolioID == "" {
			writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "portfolio_id wajib untuk thumbnail.", r))
			return
		}
		p, perr := a.Q.GetPortfolioByID(r.Context(), mustUUID(req.PortfolioID))
		if perr != nil || !isOwnerOrAdmin(r, p.OwnerID) {
			writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Bukan portfolio-mu.", r))
			return
		}
		_ = a.Q.UpdatePortfolioThumbnail(r.Context(), repo.UpdatePortfolioThumbnailParams{ID: p.ID, ThumbnailMediaID: m.ID})
	case "block":
		if req.BlockID == "" {
			writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "block_id wajib untuk block.", r))
			return
		}
		b, berr := a.Q.GetBlock(r.Context(), mustUUID(req.BlockID))
		if berr != nil {
			writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Block tidak ditemukan.", r))
			return
		}
		p, perr := a.Q.GetPortfolioByID(r.Context(), b.PortfolioID)
		if perr != nil || !isOwnerOrAdmin(r, p.OwnerID) {
			writeJSON(w, http.StatusForbidden, fail("FORBIDDEN_SCOPE", "Bukan portfolio-mu.", r))
			return
		}
		_ = a.Q.SetBlockAsset(r.Context(), repo.SetBlockAssetParams{ID: b.ID, AssetMediaID: m.ID})
	default:
		writeJSON(w, http.StatusUnprocessableEntity, fail("VALIDATION_ERROR", "link_to tidak valid.", r))
		return
	}
	_ = a.Q.ConfirmMedia(r.Context(), m.ID)
	invalidatePerms(r.Context(), a.Cache, uid)
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "confirmed"}, r))
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
	// Hapus objek utama + varian dari storage (bukan hanya tandai DB).
	_ = a.Storage.Delete(r.Context(), m.ObjectKey)
	var vs map[string]string
	if err := json.Unmarshal(m.Variants, &vs); err == nil {
		for _, vkey := range vs {
			_ = a.Storage.Delete(r.Context(), vkey)
		}
	}
	_ = a.Q.ClearUserAvatarIfMedia(r.Context(), m.ID)
	_ = a.Q.MarkMediaDeleted(r.Context(), m.ID)
	auditLog(r, a, "media.delete", "media_assets", id, m.Purpose)
	writeJSON(w, http.StatusOK, ok(map[string]string{"status": "deleted"}, r))
}
