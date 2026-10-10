package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// bodyReaderAt membungkus ReadCloser menjadi io.ReaderAt sehingga
// http.ServeContent bisa memakai Range tanpa menahan seluruh file di memori.
func bodyReaderAt(rc io.ReadCloser) io.ReaderAt {
	if ra, ok := rc.(io.ReaderAt); ok {
		return ra
	}
	return &streamReaderAt{rc: rc}
}

type streamReaderAt struct {
	rc   io.ReadCloser
	buf  *bytes.Buffer
	done bool
}

func (s *streamReaderAt) ReadAt(p []byte, off int64) (int, error) {
	if s.done {
		return 0, io.EOF
	}
	// Objek storage (S3 GetObject) tidak mendukung random access: baca
	// berurutan lalu simpan ke buffer sementara sampai offset terpenuhi.
	if s.buf == nil {
		s.buf = &bytes.Buffer{}
	}
	for int64(s.buf.Len()) < off+int64(len(p)) {
		n, err := io.CopyN(s.buf, s.rc, 64*1024)
		if err != nil {
			s.done = true
			break
		}
		if n == 0 {
			s.done = true
			break
		}
	}
	data := s.buf.Bytes()
	if off >= int64(len(data)) {
		return 0, io.EOF
	}
	n := copy(p, data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

var _ io.ReaderAt = (*streamReaderAt)(nil)

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
	// Streaming: jangan membaca seluruh file ke memori (media bisa puluhan MB).
	size, err := a.Storage.Size(r.Context(), key)
	if err != nil || size <= 0 {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "File tidak ditemukan.", r))
		return
	}
	reader := io.NewSectionReader(bodyReaderAt(body), 0, size)
	w.Header().Set("Content-Type", m.Mime)
	w.Header().Set("ETag", `"`+m.ChecksumSha256+`"`)
	if publicPurpose {
		// Konten publik boleh di-cache lama; konten privat tidak.
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "private, max-age=300")
	}
	http.ServeContent(w, r, key, time.Now(), reader)
}

// handleServeMediaByKey melayani file via object key ( dipakai redirect 301
// URL lama /storage/* -> /api/media/by-key/* selama 6 bulan pasca go-live,
// PRD §16.1 ). Key dibatasi pola aman agar tidak bisa path traversal.
func (a *App) handleServeMediaByKey(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "*")
	key = strings.TrimPrefix(key, "/")
	if key == "" || strings.Contains(key, "..") || strings.HasPrefix(key, "/") {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Media tidak ditemukan.", r))
		return
	}
	m, err := a.Q.GetMediaByKey(r.Context(), key)
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
	body, err := a.Storage.Get(r.Context(), m.ObjectKey)
	if err != nil {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "File tidak ditemukan.", r))
		return
	}
	defer body.Close()
	size, serr := a.Storage.Size(r.Context(), m.ObjectKey)
	if serr != nil || size <= 0 {
		writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "File tidak ditemukan.", r))
		return
	}
	reader := io.NewSectionReader(bodyReaderAt(body), 0, size)
	w.Header().Set("Content-Type", m.Mime)
	w.Header().Set("ETag", `"`+m.ChecksumSha256+`"`)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeContent(w, r, m.ObjectKey, time.Now(), reader)
}

// handleLegacyStorageRedirect menerima URL lama /storage/* (PRD §16.1):
// bila old_url dikenal di legacy_urls -> 301 ke /api/media/by-key/<object_key>,
// bila tidak dikenal -> 410 Gone (bukan 404, supaya crawler berhenti).
func (a *App) handleLegacyStorageRedirect(w http.ResponseWriter, r *http.Request) {
	rest := chi.URLParam(r, "*")
	oldURL := "https://grafikarsa.com/storage/" + strings.TrimPrefix(rest, "/")
	leg, err := a.Q.GetLegacyURL(r.Context(), oldURL)
	if err != nil {
		// Coba juga dengan host apa adanya (tanpa asumsi domain).
		leg, err = a.Q.GetLegacyURL(r.Context(), r.URL.Path)
		if err != nil {
			w.Header().Set("Cache-Control", "public, max-age=86400")
			writeJSON(w, http.StatusGone, fail("GONE", "File lama sudah tidak tersedia.", r))
			return
		}
	}
	_ = a.Q.TouchLegacyURL(r.Context(), leg.ID)
	if leg.MediaID.Valid {
		var objectKey string
		if leg.ObjectKey.Valid {
			objectKey = leg.ObjectKey.String
		} else {
			m, merr := a.Q.GetMediaByID(r.Context(), leg.MediaID)
			if merr != nil {
				writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Media tidak ditemukan.", r))
				return
			}
			objectKey = m.ObjectKey
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.Redirect(w, r, "/v1/media/by-key/"+objectKey, http.StatusMovedPermanently)
		return
	}
	if leg.ObjectKey.Valid {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.Redirect(w, r, "/v1/media/by-key/"+leg.ObjectKey.String, http.StatusMovedPermanently)
		return
	}
	writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Media tidak ditemukan.", r))
}
