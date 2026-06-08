package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/audit"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/go-chi/chi/v5"
)

// fakeSink captures recorded events for assertions.
type fakeSink struct {
	mu     sync.Mutex
	events []audit.Event
}

func (f *fakeSink) Record(e audit.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, e)
}

func (f *fakeSink) all() []audit.Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]audit.Event(nil), f.events...)
}

// withUser injects an authenticated actor like the JWT middleware would.
func withUser(next http.Handler, id, role string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), contextkey.UserKey, &security.AuthUser{ID: id, Role: role})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TestAudit_RecordsMutationWithActor(t *testing.T) {
	sink := &fakeSink{}

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler { return withUser(next, "u-42", "ADMIN") })
	r.Use(Audit(sink))
	r.Post("/api/sales-order/{code}/status", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/sales-order/SO-1/status", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	events := sink.all()
	if len(events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(events))
	}
	e := events[0]
	if e.UserID != "u-42" || e.UserRole != "ADMIN" {
		t.Fatalf("actor not captured: %+v", e)
	}
	if e.Method != http.MethodPost || e.Route != "/api/sales-order/{code}/status" {
		t.Fatalf("action not captured: method=%s route=%s", e.Method, e.Route)
	}
	if e.Path != "/api/sales-order/SO-1/status" || e.Status != http.StatusOK {
		t.Fatalf("target/outcome wrong: path=%s status=%d", e.Path, e.Status)
	}
}

func TestAudit_SkipsReads(t *testing.T) {
	sink := &fakeSink{}

	r := chi.NewRouter()
	r.Use(Audit(sink))
	r.Get("/api/items", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/items", nil))

	if got := len(sink.all()); got != 0 {
		t.Fatalf("GET should not be audited, got %d events", got)
	}
}

func TestAudit_NilSinkIsPassthrough(t *testing.T) {
	called := false
	h := Audit(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/x", nil))

	if !called {
		t.Fatal("handler must still run when sink is nil")
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
}
