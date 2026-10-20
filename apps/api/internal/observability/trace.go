package observability

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/go-chi/chi/v5"
)

// Tracer setup (PRD §13.1, §17).
//
// Fase 1 tidak mewajibkan collector: "Loki/Prom opsional di fase 2, fase 1
// cukup log file + docker logs". Karena itu tracing di sini:
//   - aktif hanya kalau OTEL_EXPORTER_OTLP_ENDPOINT di-set;
//   - kalau tidak, semuanya no-op dan aplikasi jalan seperti biasa (nol
//     dependensi runtime, nol overhead berarti);
//   - trace id tetap dikembalikan di header X-Trace-Id supaya log lokal
//     bisa dikorelasikan dengan trace di collector.
//
// Tidak ada sampling agresif: default "parentbased_always_on" supaya trace
// lengkap untuk request yang lambat. Env OTEL_TRACES_SAMPLER bisa mengubah.

const tracerName = "grafikarsa/api"

// Tracer mengembalikan tracer aplikasi.
func Tracer() trace.Tracer { return otel.Tracer(tracerName) }

// TraceIDFromContext mengambil trace id dari context.
func TraceIDFromContext(ctx context.Context) string {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.HasTraceID() {
		return ""
	}
	return sc.TraceID().String()
}

// InitTracing menyiapkan tracing OpenTelemetry.
//
// Mengembalikan fungsi shutdown (selalu aman dipanggil, apa pun hasilnya).
// Kalau OTEL_EXPORTER_OTLP_ENDPOINT kosong, tracing dimatikan sepenuhnya
// (no-op provider) — aplikasi tetap jalan normal.
func InitTracing(ctx context.Context, serviceName, endpoint string, sampleRatio float64) (func(context.Context) error, error) {
	noop := func(context.Context) error { return nil }
	if strings.TrimSpace(endpoint) == "" {
		slog.Info("tracing dimatikan (OTEL_EXPORTER_OTLP_ENDPOINT kosong)")
		return noop, nil
	}

	exp, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(strings.TrimPrefix(strings.TrimPrefix(endpoint, "http://"), "https://")),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithTimeout(10*time.Second),
	)
	if err != nil {
		return noop, fmt.Errorf("observability: buat exporter OTLP: %w", err)
	}

	res := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exp),
		sdktrace.WithSampler(sampler(sampleRatio)),
	)

	otel.SetTracerProvider(res)
	// W3C Trace Context: header `traceparent` diteruskan dari FE/collector
	// supaya satu request bisa dilacak lintas service.
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	slog.Info("tracing aktif", "service", serviceName, "endpoint", endpoint, "sample_ratio", sampleRatio)

	return func(shutdownCtx context.Context) error {
		ctx, cancel := context.WithTimeout(shutdownCtx, 5*time.Second)
		defer cancel()
		return res.Shutdown(ctx)
	}, nil
}

// sampler mengembalikan sampler sesuai rasio.
//
// Rasio 0 = tidak ada span yang direkam (hanya tidak berguna kalau
// sampling dimatikan total; default-nya 1.0).
func sampler(ratio float64) sdktrace.Sampler {
	switch {
	case ratio <= 0:
		return sdktrace.NeverSample()
	case ratio >= 1:
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	default:
		return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
	}
}

// TracingConfig membaca konfigurasi tracing dari env.
func TracingConfig() (serviceName, endpoint string, sampleRatio float64) {
	serviceName = getEnv("OTEL_SERVICE_NAME", "grafikarsa-api")
	endpoint = getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	sampleRatio = 1
	if v := getEnv("OTEL_TRACES_SAMPLER_RATIO", ""); v != "" {
		if r, err := strconv.ParseFloat(v, 64); err == nil && r >= 0 && r <= 1 {
			sampleRatio = r
		}
	}
	return serviceName, endpoint, sampleRatio
}

func getEnv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// TraceMiddleware membungkus handler dengan span per request.
//
// Span berisi: nama route (bukan path mentah, supaya kardinalitas tetap
// rendah), method, status, dan durasi. Trace id dikembalikan di header
// X-Trace-Id supaya bisa dicari di log lokal maupun di collector.
func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := Tracer().Start(r.Context(), spanName(r))
		defer span.End()

		// Teruskan context trace ke handler.
		r = r.WithContext(ctx)

		// Header trace id harus ditulis SEBELUM handler dipanggil: begitu
		// response mulai ditulis, header yang ditambah belakangan tidak
		// terkirim sama sekali.
		if id := TraceIDFromContext(ctx); id != "" {
			w.Header().Set("X-Trace-Id", id)
		}

		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(ww, r)

		span.SetAttributes(
			attribute.String("http.request.method", r.Method),
			attribute.Int("http.response.status_code", ww.status),
			attribute.String("url.path", r.URL.Path),
		)
	})
}

// spanName memberi nama span yang stabil: "METHOD /route" dengan route
// pola chi, bukan path mentah.
//
// Path mentah membuat satu span per URL — ribuan span untuk satu endpoint
// (mis. /portfolios/{id}) dan collector jadi mahal. Chi menyimpan pola
// route di context, jadi itu yang dipakai.
func spanName(r *http.Request) string {
	if rc := chiRouteContext(r); rc != "" {
		return r.Method + " " + rc
	}
	return r.Method
}

// statusWriter remembers the response status code.
type statusWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (s *statusWriter) WriteHeader(code int) {
	if !s.written {
		s.status = code
		s.written = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	s.written = true
	return s.ResponseWriter.Write(b)
}

// Flush meng delegation ke ResponseWriter bila ada (Writes streaming).
func (s *statusWriter) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// chiRouteContext mengembalikan pola route chi dari context (mis.
// "/portfolios/{id}"), bukan path yang sudah di-substitusi.
//
// Kalau context-nya bukan chi (mis. dipanggil di luar router), hasilnya
// kosong dan pemanggil memakai path mentah.
func chiRouteContext(r *http.Request) string {
	if rctx := chi.NewRouteContext(); rctx != nil {
		if rc := r.Context().Value(chi.RouteCtxKey); rc != nil {
			if routeCtx, ok := rc.(*chi.Context); ok {
				routeCtx.RoutePatterns = append(routeCtx.RoutePatterns, "")
				if p := routeCtx.RoutePattern(); p != "" {
					return p
				}
			}
		}
	}
	return ""
}
