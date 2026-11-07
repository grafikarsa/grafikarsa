// Package problem writes RFC 9457 application/problem+json responses.
package problem

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/reqctx"
)

// FieldError describes a validation failure on a single request field.
type FieldError struct {
	Field   string `json:"field"`
	Ref     string `json:"ref,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Problem is the response body defined by RFC 9457.
type Problem struct {
	Type      string       `json:"type"`
	Title     string       `json:"title"`
	Status    int          `json:"status"`
	Code      string       `json:"code"`
	Detail    string       `json:"detail,omitempty"`
	Errors    []FieldError `json:"errors,omitempty"`
	RequestID string       `json:"requestId,omitempty"`
}

// Write sends p as an application/problem+json response. It fills the request
// ID from the request context and derives the type URL from the error code.
func Write(w http.ResponseWriter, r *http.Request, p Problem) {
	if p.RequestID == "" {
		p.RequestID = reqctx.RequestID(r.Context())
	}
	if p.Type == "" {
		p.Type = "https://grafikarsa.com/errors/" + strings.ReplaceAll(strings.ToLower(p.Code), "_", "-")
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	_ = json.NewEncoder(w).Encode(p)
}

// Internal returns the generic 500 response.
func Internal() Problem {
	return Problem{
		Status: http.StatusInternalServerError,
		Code:   "INTERNAL",
		Title:  "Terjadi kesalahan",
	}
}

// NotFound returns the generic 404 response.
func NotFound() Problem {
	return Problem{
		Status: http.StatusNotFound,
		Code:   "NOT_FOUND",
		Title:  "Tidak ditemukan",
	}
}

// MethodNotAllowed returns the generic 405 response.
func MethodNotAllowed() Problem {
	return Problem{
		Status: http.StatusMethodNotAllowed,
		Code:   "METHOD_NOT_ALLOWED",
		Title:  "Metode tidak diizinkan",
	}
}

// Unavailable returns the generic 503 response with a human-readable detail.
func Unavailable(detail string) Problem {
	return Problem{
		Status: http.StatusServiceUnavailable,
		Code:   "UNAVAILABLE",
		Title:  "Layanan belum siap",
		Detail: detail,
	}
}
