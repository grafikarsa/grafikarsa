package http

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"grafikarsa/api/internal/observability"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

func baseMiddleware() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(recoverPanic)
	// Tracing OpenTelemetry (PRD §13.1): span per request. Tracing no-op
	// kalau collector tidak dikonfigurasi, jadi tidak ada overhead berarti.
	r.Use(observability.TraceMiddleware)
	r.Use(requestLogger)
	r.Use(middleware.Timeout(30 * time.Second))
	r.NotFound(notFoundHandler)
	r.MethodNotAllowed(methodNotAllowedHandler)
	return r
}

// uuidPathParams adalah nama parameter path yang isinya harus UUID.
//
// Parameter lain (username, slug, version, key) bukan UUID, jadi tidak
// boleh ikut divalidasi.
var uuidPathParams = map[string]bool{
	"id":           true,
	"blockId":      true,
	"userId":       true,
	"portfolio_id": true,
	"threadId":     true,
}

// validateUUIDParams menolak parameter path yang isinya bukan UUID.
//
// Sebelumnya mustUUID "menelan" kegagalan parse: string yang bukan UUID
// menjadi UUID nol (00000000-...), dan query tetap jalan — hasilnya 200
// dengan daftar kosong. Jadi "/portfolios/id/abc/comments" membalas
// "tidak ada komentar" padahal id-nya salah ketik. Klien lalu menampilkan
// "belum ada komentar" untuk portofolio yang sebenarnya ada.
//
// Sekarang id yang tidak valid ditolak 400 dengan pesan yang jelas.
func validateUUIDParams(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Nama parameter diambil dari pola route yang cocok
		// (mis. "/v1/portfolios/id/{id}/comments" -> "id").
		pattern := chi.RouteContext(r.Context()).RoutePattern()
		for _, name := range paramNames(pattern) {
			if !uuidPathParams[name] {
				continue
			}
			raw := chi.URLParam(r, name)
			if _, err := uuid.Parse(raw); err != nil {
				writeJSON(w, http.StatusBadRequest,
					fail("VALIDATION_ERROR", "Parameter "+name+" bukan UUID yang valid.", r))
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// notFoundHandler & methodNotAllowedHandler: error 404/405 memakai envelope
// JSON yang sama dengan handler lain.
//
// Tanpa ini, chi menjawab 404/405 dengan body text/plain — sedangkan
// kontrak API menjanjikan application/json untuk semua error. Klien yang
// mem-parse JSON akan gagal dengan pesan yang tidak jelas, dan contract test
// menandai "Undocumented Content-Type".
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotFound, fail("NOT_FOUND", "Endpoint tidak ditemukan.", r))
}

func methodNotAllowedHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusMethodNotAllowed, fail("METHOD_NOT_ALLOWED", "Metode tidak diizinkan.", r))
}

// requestLogger: satu baris JSON per request (request_id, method, path,
// status, durasi) tanpa mencatat body/PII.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Jam sistem (bukan App.Clock) sengaja dipakai di sini: yang
		// diukur adalah durasi request yang sebenarnya.
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		observability.Record(r.URL.Path, ww.Status(), time.Since(start))
		slog.Info("http",
			"id", middleware.GetReqID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"ms", time.Since(start).Milliseconds(),
		)
	})
}

var _ = http.MethodGet

// paramNames mengambil nama parameter path dari pola route
// ("/v1/portfolios/id/{id}" -> ["id"]).
func paramNames(pattern string) []string {
	var out []string
	for _, seg := range strings.Split(pattern, "/") {
		if len(seg) > 2 && strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
			out = append(out, seg[1:len(seg)-1])
		}
	}
	return out
}

// recoverPanic menangkap panic di handler.
//
// Dua jenis panic yang ditangani:
//   - invalidUUIDError: input pengguna bukan UUID -> 400 dengan pesan jelas
//     (bukan 500, dan bukan 200 dengan data kosong).
//   - panic lain: kesalahan server -> 500 + dicatat di log JSON.
//
// Recovery milik chi (middleware.Recoverer) menulis ke stderr dengan format
// bawaan, bukan JSON — sementara log aplikasi semuanya JSON. Jadi recoverer
// sendiri dipakai supaya format log tetap seragam.
func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if _, ok := rec.(invalidUUIDError); ok {
					writeJSON(w, http.StatusBadRequest,
						fail("VALIDATION_ERROR", "Identifier bukan UUID yang valid.", r))
					return
				}
				slog.Error("panic di handler",
					"method", r.Method,
					"path", r.URL.Path,
					"panic", rec,
					"stack", string(debug.Stack()),
				)
				writeJSON(w, http.StatusInternalServerError,
					fail("INTERNAL_ERROR", "Terjadi kesalahan di server.", r))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
