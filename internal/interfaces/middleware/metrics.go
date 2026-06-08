package middleware

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// defaultBuckets are the cumulative latency buckets (in seconds) used by the
// request-duration histogram. They follow the Prometheus client defaults so the
// resulting series are familiar to anyone building dashboards/alerts.
var defaultBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// histogram is a minimal, cumulative-bucket histogram for one label set.
type histogram struct {
	counts []uint64 // counts[i] == observations with value <= buckets[i] (cumulative)
	sum    float64
	count  uint64
}

// Metrics is a dependency-free, Prometheus-text-format collector for HTTP
// traffic. It keeps the vendor tree lean (no client_golang) while exposing the
// RED signals — Rate, Errors, Duration — that an on-prem deployment needs:
//
//	http_requests_total{method,route,status}      counter
//	http_request_duration_seconds{method,route}   histogram
//	http_requests_in_flight                       gauge
//	app_uptime_seconds                            gauge
//
// Series are keyed by the chi *route pattern* (e.g. "/api/sales-order/{code}")
// rather than the raw path, so cardinality stays bounded.
type Metrics struct {
	mu       sync.Mutex
	buckets  []float64
	reqTotal map[string]uint64     // key: method\x00route\x00status
	hist     map[string]*histogram // key: method\x00route
	inFlight int64
	start    time.Time
}

// NewMetrics returns a ready-to-use collector.
func NewMetrics() *Metrics {
	return &Metrics{
		buckets:  defaultBuckets,
		reqTotal: make(map[string]uint64),
		hist:     make(map[string]*histogram),
		start:    time.Now(),
	}
}

// Middleware records one observation per request. It is panic-safe: the
// in-flight gauge is decremented and the observation recorded via defer even if
// a downstream handler panics (the outer Recoverer turns it into a 500).
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)

		m.mu.Lock()
		m.inFlight++
		m.mu.Unlock()

		defer func() {
			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = "unmatched"
			}
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			m.observe(r.Method, route, status, time.Since(start).Seconds())
		}()

		next.ServeHTTP(ww, r)
	})
}

func (m *Metrics) observe(method, route string, status int, secs float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.inFlight--
	m.reqTotal[method+"\x00"+route+"\x00"+strconv.Itoa(status)]++

	hk := method + "\x00" + route
	h := m.hist[hk]
	if h == nil {
		h = &histogram{counts: make([]uint64, len(m.buckets))}
		m.hist[hk] = h
	}
	h.sum += secs
	h.count++
	for i, b := range m.buckets {
		if secs <= b {
			h.counts[i]++
		}
	}
}

// Handler exposes the collected metrics in the Prometheus text exposition
// format (v0.0.4). Label values are emitted with %q, which escapes the quote,
// backslash and newline characters Prometheus requires to be escaped.
func (m *Metrics) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		defer m.mu.Unlock()

		var b strings.Builder

		b.WriteString("# HELP http_requests_total Total number of HTTP requests processed.\n")
		b.WriteString("# TYPE http_requests_total counter\n")
		for _, k := range sortedKeys(m.reqTotal) {
			p := strings.Split(k, "\x00")
			fmt.Fprintf(&b, "http_requests_total{method=%q,route=%q,status=%q} %d\n", p[0], p[1], p[2], m.reqTotal[k])
		}

		b.WriteString("# HELP http_request_duration_seconds HTTP request latency in seconds.\n")
		b.WriteString("# TYPE http_request_duration_seconds histogram\n")
		hkeys := make([]string, 0, len(m.hist))
		for k := range m.hist {
			hkeys = append(hkeys, k)
		}
		sort.Strings(hkeys)
		for _, k := range hkeys {
			p := strings.Split(k, "\x00")
			h := m.hist[k]
			for i, bk := range m.buckets {
				fmt.Fprintf(&b, "http_request_duration_seconds_bucket{method=%q,route=%q,le=%q} %d\n",
					p[0], p[1], strconv.FormatFloat(bk, 'g', -1, 64), h.counts[i])
			}
			fmt.Fprintf(&b, "http_request_duration_seconds_bucket{method=%q,route=%q,le=\"+Inf\"} %d\n", p[0], p[1], h.count)
			fmt.Fprintf(&b, "http_request_duration_seconds_sum{method=%q,route=%q} %g\n", p[0], p[1], h.sum)
			fmt.Fprintf(&b, "http_request_duration_seconds_count{method=%q,route=%q} %d\n", p[0], p[1], h.count)
		}

		b.WriteString("# HELP http_requests_in_flight Number of HTTP requests currently being served.\n")
		b.WriteString("# TYPE http_requests_in_flight gauge\n")
		fmt.Fprintf(&b, "http_requests_in_flight %d\n", m.inFlight)

		b.WriteString("# HELP app_uptime_seconds Seconds since the process started.\n")
		b.WriteString("# TYPE app_uptime_seconds gauge\n")
		fmt.Fprintf(&b, "app_uptime_seconds %g\n", time.Since(m.start).Seconds())

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = w.Write([]byte(b.String()))
	}
}

func sortedKeys(m map[string]uint64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
