package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
)

func TestWriteError_MapsAppErrToProblem(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/x", nil)
	err := apperr.Validation(apperr.Field("username", "taken", "Username sudah dipakai."))

	WriteError(rec, req, slog.New(slog.NewTextHandler(io.Discard, nil)), err)

	if rec.Code != 422 {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
	var body struct {
		Code   string `json:"code"`
		Errors []struct {
			Field string `json:"field"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != "VALIDATION_FAILED" || len(body.Errors) != 1 || body.Errors[0].Field != "username" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestWriteError_UnknownErrorIs500WithoutDetail(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/x", nil)

	WriteError(rec, req, slog.New(slog.NewTextHandler(io.Discard, nil)), errors.New("secret db detail"))

	if rec.Code != 500 {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := rec.Body.String(); strings.Contains(got, "secret db detail") {
		t.Fatalf("internal detail leaked: %s", got)
	}
}
