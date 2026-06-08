package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func TestCORS_ReflectsAllowedOrigin(t *testing.T) {
	h := CORS([]string{"http://localhost:5173"}, false)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("allowed origin not reflected, got %q", got)
	}
	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("credentials header missing")
	}
}

func TestCORS_RejectsUnknownOrigin(t *testing.T) {
	h := CORS([]string{"http://localhost:5173"}, false)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://evil.example")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected ACAO header for unknown origin: %q", got)
	}
}

func TestCORS_PreflightShortCircuits(t *testing.T) {
	called := false
	h := CORS(nil, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://anything")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if called {
		t.Fatal("preflight should not reach the next handler")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", rec.Code)
	}
}

func TestRateLimiter_BlocksAfterBurst(t *testing.T) {
	// rps=0 refill so the bucket never recovers within the test window.
	rl := NewRateLimiter(0.0001, 2)
	h := rl.Middleware(okHandler())

	codes := make([]int, 0, 4)
	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		codes = append(codes, rec.Code)
	}

	// burst=2 → first two pass, the rest are throttled.
	if codes[0] != http.StatusOK || codes[1] != http.StatusOK {
		t.Fatalf("first two requests should pass, got %v", codes)
	}
	if codes[2] != http.StatusTooManyRequests || codes[3] != http.StatusTooManyRequests {
		t.Fatalf("requests beyond burst should be 429, got %v", codes)
	}
}

func TestRateLimiter_DisabledWhenRPSZero(t *testing.T) {
	rl := NewRateLimiter(0, 0)
	h := rl.Middleware(okHandler())

	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.2:1"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("disabled limiter must pass everything, got %d at i=%d", rec.Code, i)
		}
	}
}

func TestRateLimiter_PerIPIsolation(t *testing.T) {
	rl := NewRateLimiter(0.0001, 1)
	h := rl.Middleware(okHandler())

	hit := func(ip string) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip + ":1"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	if hit("1.1.1.1") != http.StatusOK {
		t.Fatal("first request for IP A should pass")
	}
	// A is now exhausted, but B has its own bucket.
	if hit("2.2.2.2") != http.StatusOK {
		t.Fatal("first request for IP B should pass independently")
	}
	if hit("1.1.1.1") != http.StatusTooManyRequests {
		t.Fatal("second request for IP A should be throttled")
	}
}

func TestMetrics_RecordsAndExposes(t *testing.T) {
	m := NewMetrics()

	// Route through a chi router so the route pattern is populated.
	r := chi.NewRouter()
	r.Use(m.Middleware)
	r.Get("/api/items/{code}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/items/ABC", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	rec := httptest.NewRecorder()
	m.Handler()(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()

	if !strings.Contains(body, `http_requests_total{method="GET",route="/api/items/{code}",status="200"} 1`) {
		t.Fatalf("counter series missing or wrong:\n%s", body)
	}
	if !strings.Contains(body, `http_request_duration_seconds_count{method="GET",route="/api/items/{code}"} 1`) {
		t.Fatalf("histogram count series missing:\n%s", body)
	}
	if !strings.Contains(body, "app_uptime_seconds") {
		t.Fatalf("uptime gauge missing:\n%s", body)
	}
}

func TestMaxBodyBytes_CapsBody(t *testing.T) {
	var readErr error
	h := MaxBodyBytes(8)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 64)
		_, readErr = r.Body.Read(buf)
		for readErr == nil {
			_, readErr = r.Body.Read(buf)
		}
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("0123456789ABCDEF"))
	h.ServeHTTP(httptest.NewRecorder(), req)

	if readErr == nil || !strings.Contains(readErr.Error(), "too large") {
		t.Fatalf("expected body-too-large error, got %v", readErr)
	}
}
