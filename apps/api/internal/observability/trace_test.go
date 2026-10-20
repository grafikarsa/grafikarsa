package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestInitTracingTanpaEndpointMemastikan tracing mati dengan aman kalau
// collector tidak dikonfigurasi.
//
// Fase 1 tidak mewajibkan collector ("Loki/Prom opsional di fase 2"), jadi
// aplikasi HARUS tetap jalan normal tanpa env OTEL. Kalau ini error,
// seluruh API tidak akan start.
func TestInitTracingTanpaEndpoint(t *testing.T) {
	shutdown, err := InitTracing(context.Background(), "test", "", 1.0)
	if err != nil {
		t.Fatalf("InitTracing tanpa endpoint harus berhasil, err: %v", err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("shutdown harus aman, err: %v", err)
	}
}

// TestTraceMiddlewareSelaluJalan memastikan middleware tidak pernah
// memblokir request, apa pun konfigurasi tracing.
func TestTraceMiddlewareSelaluJalan(t *testing.T) {
	h := TraceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("ok"))
	}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/health", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, ingin %d", rec.Code, http.StatusTeapot)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body = %q, ingin ok", rec.Body.String())
	}
}

// TestTraceMiddlewareMenyimpanStatus memastikan status code terbaca benar
// (dipakai sebagai atribut span).
func TestTraceMiddlewareMenyimpanStatus(t *testing.T) {
	h := TraceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, ingin 404", rec.Code)
	}
}

// TestTraceMiddlewareMenulisBody memastikan response body tetap sampai ke
// klien (middleware tidak menelan bytes).
func TestTraceMiddlewareMenulisBody(t *testing.T) {
	h := TraceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("halo"))
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
	if rec.Body.String() != "halo" {
		t.Errorf("body = %q, ingin halo", rec.Body.String())
	}
}

// TestTracingConfig memastikan env dibaca dengan benar.
func TestTracingConfig(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "svc-uji")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318")
	t.Setenv("OTEL_TRACES_SAMPLER_RATIO", "0.25")
	name, endpoint, ratio := TracingConfig()
	if name != "svc-uji" {
		t.Errorf("service name = %q, ingin svc-uji", name)
	}
	if endpoint != "http://collector:4318" {
		t.Errorf("endpoint = %q, ingin http://collector:4318", endpoint)
	}
	if ratio != 0.25 {
		t.Errorf("sample ratio = %v, ingin 0.25", ratio)
	}
}

// TestTracingConfigDefault memastikan nilai default masuk akal.
func TestTracingConfigDefault(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("OTEL_TRACES_SAMPLER_RATIO", "")
	name, endpoint, ratio := TracingConfig()
	if name == "" {
		t.Error("service name default tidak boleh kosong")
	}
	if endpoint != "" {
		t.Errorf("endpoint default harus kosong (tracing mati), dapat %q", endpoint)
	}
	if ratio != 1 {
		t.Errorf("sample ratio default = %v, ingin 1", ratio)
	}
}
