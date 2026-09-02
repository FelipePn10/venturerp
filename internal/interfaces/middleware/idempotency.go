package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// IdempotencyStore persists completed responses in PostgreSQL when a pool is
// configured, with an in-memory fallback used by isolated unit tests.
// It lets clients safely retry create requests without producing duplicates.
type IdempotencyStore struct {
	mu   sync.Mutex
	m    map[string]*idempotencyEntry
	ttl  time.Duration
	pool *pgxpool.Pool
}

type idempotencyEntry struct {
	status    int
	body      []byte
	createdAt time.Time
	done      bool
}

const maxIdempotencyRequestBody = 10 << 20

func NewIdempotencyStore(ttl time.Duration, pool ...*pgxpool.Pool) *IdempotencyStore {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	s := &IdempotencyStore{m: make(map[string]*idempotencyEntry), ttl: ttl}
	if len(pool) > 0 {
		s.pool = pool[0]
	}
	return s
}

func requestFingerprint(r *http.Request) (string, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}

func (s *IdempotencyStore) claimPersistent(r *http.Request, scope, fingerprint string) (bool, int, []byte, error) {
	if s.pool == nil {
		return true, 0, nil, nil
	}
	_, _ = s.pool.Exec(r.Context(), `DELETE FROM http_idempotency_records WHERE expires_at<NOW()`)
	tag, err := s.pool.Exec(r.Context(), `INSERT INTO http_idempotency_records(scope_key,request_fingerprint,expires_at) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, scope, fingerprint, time.Now().Add(s.ttl))
	if err != nil {
		return false, 0, nil, err
	}
	if tag.RowsAffected() == 1 {
		return true, 0, nil, nil
	}
	var storedFingerprint string
	var status *int
	var body []byte
	var completed bool
	err = s.pool.QueryRow(r.Context(), `SELECT request_fingerprint,status_code,response_body,completed FROM http_idempotency_records WHERE scope_key=$1`, scope).Scan(&storedFingerprint, &status, &body, &completed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, 0, nil, err
	}
	if err != nil {
		return false, 0, nil, err
	}
	if storedFingerprint != fingerprint {
		return false, http.StatusConflict, []byte(`{"error":"Idempotency-Key reutilizada com corpo diferente","code":"IDEMPOTENCY_KEY_REUSED"}`), nil
	}
	if !completed {
		return false, http.StatusConflict, []byte(`{"error":"requisição com esta Idempotency-Key já está em andamento","code":"IDEMPOTENCY_IN_PROGRESS"}`), nil
	}
	if status == nil {
		return false, http.StatusConflict, nil, nil
	}
	return false, *status, body, nil
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

func RequireIdempotencyKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(r.Header.Get("Idempotency-Key")) == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`{"error":"o cabeçalho Idempotency-Key é obrigatório","code":"IDEMPOTENCY_KEY_REQUIRED"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
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
			mutating := r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch || r.Method == http.MethodDelete
			if key == "" || !mutating {
				next.ServeHTTP(w, r)
				return
			}

			userPart := ""
			if u, ok := r.Context().Value(contextkey.UserKey).(*security.AuthUser); ok {
				userPart = u.ID
			}
			enterprisePart := ""
			if u, ok := r.Context().Value(contextkey.UserKey).(*security.AuthUser); ok {
				enterprisePart = strconv.FormatInt(u.EnterpriseID, 10)
			}
			canonicalQuery := r.URL.Query().Encode()
			fullKey := r.Method + " " + r.URL.Path + "?" + canonicalQuery + " " + enterprisePart + " " + userPart + " " + key
			r.Body = http.MaxBytesReader(w, r.Body, maxIdempotencyRequestBody)
			if store.pool != nil {
				fingerprint, err := requestFingerprint(r)
				if err != nil {
					var maxBytesErr *http.MaxBytesError
					if errors.As(err, &maxBytesErr) {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusRequestEntityTooLarge)
						_, _ = w.Write([]byte(`{"error":"corpo da requisição excede o limite de 10 MiB","code":"REQUEST_BODY_TOO_LARGE"}`))
						return
					}
					http.Error(w, `{"error":"não foi possível ler a requisição"}`, http.StatusBadRequest)
					return
				}
				claimed, status, body, err := store.claimPersistent(r, fullKey, fingerprint)
				if err != nil {
					http.Error(w, `{"error":"falha no controle de idempotência"}`, http.StatusServiceUnavailable)
					return
				}
				if !claimed {
					w.Header().Set("Content-Type", "application/json")
					if status >= 200 && status < 300 {
						w.Header().Set("Idempotent-Replayed", "true")
					}
					w.WriteHeader(status)
					_, _ = w.Write(body)
					return
				}
				rec := &idemRecorder{ResponseWriter: w, status: http.StatusOK, buf: &bytes.Buffer{}}
				next.ServeHTTP(rec, r)
				if rec.status >= 200 && rec.status < 300 {
					_, _ = store.pool.Exec(r.Context(), `UPDATE http_idempotency_records SET status_code=$2,response_body=$3,completed=TRUE WHERE scope_key=$1`, fullKey, rec.status, rec.buf.Bytes())
				} else {
					_, _ = store.pool.Exec(r.Context(), `DELETE FROM http_idempotency_records WHERE scope_key=$1`, fullKey)
				}
				return
			}

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
