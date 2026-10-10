package observability

import (
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics minimal tanpa dependensi eksternal: counter per route+status,
// histogram kasar durasi, dan runtime stats. Format: Prometheus text.
type Metrics struct {
	mu        sync.RWMutex
	counters  map[string]*atomic.Int64
	durations map[string]*atomic.Int64 // total ms
	buckets   map[string]*atomic.Int64 // le/100/500/1000/5000 ms
}

func NewMetrics() *Metrics {
	return &Metrics{
		counters:  map[string]*atomic.Int64{},
		durations: map[string]*atomic.Int64{},
		buckets:   map[string]*atomic.Int64{},
	}
}

var metrics = NewMetrics()

func counter(key string, m map[string]*atomic.Int64, mu *sync.RWMutex) *atomic.Int64 {
	mu.RLock()
	c, ok := m[key]
	mu.RUnlock()
	if ok {
		return c
	}
	mu.Lock()
	defer mu.Unlock()
	if c, ok := m[key]; ok {
		return c
	}
	c = &atomic.Int64{}
	m[key] = c
	return c
}

// Record dipanggil dari middleware request logger.
func Record(route string, status int, dur time.Duration) {
	key := route + "|" + statusClass(status)
	counter(key, metrics.counters, &metrics.mu).Add(1)
	durKey := "duration_ms|" + route
	counter(durKey, metrics.durations, &metrics.mu).Add(dur.Milliseconds())
	for _, le := range []int64{100, 500, 1000, 5000} {
		if dur.Milliseconds() <= le {
			counter(fmt.Sprintf("bucket|%s|%d", route, le), metrics.buckets, &metrics.mu).Add(1)
		}
	}
}

func statusClass(status int) string {
	switch {
	case status >= 500:
		return "5xx"
	case status >= 400:
		return "4xx"
	case status >= 300:
		return "3xx"
	default:
		return "2xx"
	}
}

func (m *Metrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		var b strings.Builder
		b.WriteString("# HELP grafikarsa_requests_total Total request per route dan kelas status.\n")
		b.WriteString("# TYPE grafikarsa_requests_total counter\n")
		m.mu.RLock()
		keys := make([]string, 0, len(m.counters))
		for k := range m.counters {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			parts := strings.SplitN(k, "|", 2)
			fmt.Fprintf(&b, "grafikarsa_requests_total{route=%q,status=%q} %d\n", parts[0], parts[1], m.counters[k].Load())
		}
		b.WriteString("# TYPE grafikarsa_request_duration_ms_total counter\n")
		dkeys := make([]string, 0, len(m.durations))
		for k := range m.durations {
			dkeys = append(dkeys, k)
		}
		sort.Strings(dkeys)
		for _, k := range dkeys {
			fmt.Fprintf(&b, "grafikarsa_request_duration_ms_total{route=%q} %d\n", strings.TrimPrefix(k, "duration_ms|"), m.durations[k].Load())
		}
		m.mu.RUnlock()
		fmt.Fprintf(&b, "# TYPE grafikarsa_goroutines gauge\ngrafikarsa_goroutines %d\n", runtime.NumGoroutine())
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		fmt.Fprintf(&b, "# TYPE grafikarsa_memory_bytes gauge\ngrafikarsa_memory_bytes %d\n", ms.Alloc)
		_, _ = w.Write([]byte(b.String()))
	}
}

// Handler global (dipakai router untuk /metrics).
func MetricsHandler() http.HandlerFunc { return metrics.Handler() }
