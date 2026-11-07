package problem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/reqctx"
)

func TestWrite(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	req = req.WithContext(reqctx.WithRequestID(req.Context(), "req-123"))

	rec := httptest.NewRecorder()
	Write(rec, req, Problem{
		Status: http.StatusNotFound,
		Code:   "NOT_FOUND",
		Title:  "Tidak ditemukan",
		Detail: "resource tidak ada",
		Errors: []FieldError{{
			Field:   "id",
			Code:    "INVALID",
			Message: "format tidak valid",
		}},
	})

	res := rec.Result()
	defer res.Body.Close()

	if got := res.Header.Get("Content-Type"); got != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", got)
	}
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusNotFound)
	}

	var body Problem
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if body.Type != "https://grafikarsa.com/errors/not-found" {
		t.Errorf("Type = %q, want derived not-found URL", body.Type)
	}
	if body.Status != http.StatusNotFound {
		t.Errorf("Status = %d, want %d", body.Status, http.StatusNotFound)
	}
	if body.Code != "NOT_FOUND" {
		t.Errorf("Code = %q, want NOT_FOUND", body.Code)
	}
	if body.Title != "Tidak ditemukan" {
		t.Errorf("Title = %q", body.Title)
	}
	if body.Detail != "resource tidak ada" {
		t.Errorf("Detail = %q", body.Detail)
	}
	if body.RequestID != "req-123" {
		t.Errorf("RequestID = %q, want req-123", body.RequestID)
	}
	if len(body.Errors) != 1 || body.Errors[0].Field != "id" {
		t.Errorf("Errors = %+v, want one id field error", body.Errors)
	}
}

func TestConstructors(t *testing.T) {
	tests := []struct {
		name       string
		problem    Problem
		wantStatus int
		wantCode   string
		wantTitle  string
	}{
		{
			name:       "internal",
			problem:    Internal(),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL",
			wantTitle:  "Terjadi kesalahan",
		},
		{
			name:       "not found",
			problem:    NotFound(),
			wantStatus: http.StatusNotFound,
			wantCode:   "NOT_FOUND",
			wantTitle:  "Tidak ditemukan",
		},
		{
			name:       "method not allowed",
			problem:    MethodNotAllowed(),
			wantStatus: http.StatusMethodNotAllowed,
			wantCode:   "METHOD_NOT_ALLOWED",
			wantTitle:  "Metode tidak diizinkan",
		},
		{
			name:       "unavailable",
			problem:    Unavailable("database belum siap"),
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   "UNAVAILABLE",
			wantTitle:  "Layanan belum siap",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.problem.Status != tt.wantStatus {
				t.Errorf("Status = %d, want %d", tt.problem.Status, tt.wantStatus)
			}
			if tt.problem.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", tt.problem.Code, tt.wantCode)
			}
			if tt.problem.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", tt.problem.Title, tt.wantTitle)
			}
		})
	}

	if got := Unavailable("database belum siap").Detail; got != "database belum siap" {
		t.Errorf("Unavailable detail = %q", got)
	}
}

func TestWriteDerivesTypeFromCode(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)

	Write(rec, req, MethodNotAllowed())

	var body Problem
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	if want := "https://grafikarsa.com/errors/method-not-allowed"; body.Type != want {
		t.Errorf("Type = %q, want %q", body.Type, want)
	}
}
