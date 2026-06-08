package middleware

import (
	"bytes"
	"net/http"
	"sync"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// IdempotencyStore is an in-memory, TTL-bounded cache of completed responses
// keyed by the client-supplied Idempotency-Key (scoped per method+path+user).
// It lets clients safely retry create requests without producing duplicates.
//
// Note: this is per-instance and cleared on restart — it deduplicates retries
// within the TTL window, which is the common cause of accidental duplicates.
type IdempotencyStore struct {
	mu  sync.Mutex
	m   map[string]*idempotencyEntry
	ttl time.Duration
}

type idempotencyEntry struct {
	status    int
	body      []byte
	createdAt time.Time
	done      bool
}

func NewIdempotencyStore(ttl time.Duration) *IdempotencyStore {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &IdempotencyStore{m: make(map[string]*idempotencyEntry), ttl: ttl}
}

func (s *IdempotencyStore) evictLocked(now time.Time) {
	for k, e := range s.m {
		if now.Sub(e.createdAt) > s.ttl {
			delete(s.m, k)
		}
	}
}

type idemRecorder struct {
	http.ResponseWriter
	status int
	buf    *bytes.Buffer
}

func (rec *idemRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *idemRecorder) Write(b []byte) (int, error) {
	rec.buf.Write(b)
	return rec.ResponseWriter.Write(b)
}

// Idempotency replays the stored response when the same Idempotency-Key is seen
// again. Only mutating methods carrying the header are affected; everything else
// passes through untouched.
func Idempotency(store *IdempotencyStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("Idempotency-Key")
			mutating := r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch
			if key == "" || !mutating {
				next.ServeHTTP(w, r)
				return
			}

			userPart := ""
			if u, ok := r.Context().Value(contextkey.UserKey).(*security.AuthUser); ok {
				userPart = u.ID
			}
			fullKey := r.Method + " " + r.URL.Path + " " + userPart + " " + key

			now := time.Now()
			store.mu.Lock()
			store.evictLocked(now)
			e, ok := store.m[fullKey]
			if ok && e.done {
				status, body := e.status, e.body
				store.mu.Unlock()
				w.Header().Set("Idempotent-Replayed", "true")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				_, _ = w.Write(body)
				return
			}
			if ok && !e.done {
				store.mu.Unlock()
				http.Error(w, `{"error":"a request with this Idempotency-Key is already in progress"}`, http.StatusConflict)
				return
			}
			e = &idempotencyEntry{createdAt: now}
			store.m[fullKey] = e
			store.mu.Unlock()

			rec := &idemRecorder{ResponseWriter: w, status: http.StatusOK, buf: &bytes.Buffer{}}
			next.ServeHTTP(rec, r)

			store.mu.Lock()
			// Only memoize successful, replayable responses; let failures be retried.
			if rec.status >= 200 && rec.status < 300 {
				e.status = rec.status
				e.body = rec.buf.Bytes()
				e.done = true
			} else {
				delete(store.m, fullKey)
			}
			store.mu.Unlock()
		})
	}
}
