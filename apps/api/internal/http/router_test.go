package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	apiphttp "grafikarsa/api/internal/http"
)

func TestHealth(t *testing.T) {
	r := apiphttp.NewRouter(apiphttp.Deps{})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
