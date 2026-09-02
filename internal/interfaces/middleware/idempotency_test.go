package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestIdempotencyScopesKeyByCanonicalQuery(t *testing.T) {
	store := NewIdempotencyStore(time.Hour)
	var calls atomic.Int32
	handler := Idempotency(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(r.URL.Query().Get("page")))
	}))

	for _, target := range []string{"/resource?page=1&sort=code", "/resource?sort=code&page=1", "/resource?page=2&sort=code"} {
		req := httptest.NewRequest(http.MethodPost, target, nil)
		req.Header.Set("Idempotency-Key", "same-key")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status for %s = %d", target, rec.Code)
		}
	}

	if got := calls.Load(); got != 2 {
		t.Fatalf("handler calls = %d, want 2 (canonical replay plus distinct query)", got)
	}
}

func TestIdempotencyLimitsRequestBody(t *testing.T) {
	store := NewIdempotencyStore(time.Hour)
	handler := Idempotency(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "corpo muito grande", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/resource", strings.NewReader(strings.Repeat("x", maxIdempotencyRequestBody+1)))
	req.Header.Set("Idempotency-Key", "large-body")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}
