//go:build integration

package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appsecurity "github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	middleware "github.com/FelipePn10/panossoerp/internal/interfaces/middleware"
	"github.com/google/uuid"
)

func TestPersistentIdempotencyReplaysAndRejectsDifferentBody(t *testing.T) {
	pool := testutil.Pool(t)
	key := "idem-" + uuid.NewString()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM http_idempotency_records WHERE scope_key LIKE $1`, "%"+key)
	})
	store := middleware.NewIdempotencyStore(time.Hour, pool)
	userID := uuid.NewString()
	calls := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"message":"persistido"}`))
	})
	handler := middleware.Idempotency(store)(next)
	request := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/planning/run-pipeline", strings.NewReader(body))
		req.Header.Set("Idempotency-Key", key)
		user := &appsecurity.AuthUser{ID: userID, EnterpriseID: 77}
		req = req.WithContext(context.WithValue(req.Context(), contextkey.UserKey, user))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, req)
		return recorder
	}
	first := request(`{"plan_code":10}`)
	second := request(`{"plan_code":10}`)
	if first.Code != http.StatusCreated || second.Code != http.StatusCreated || second.Header().Get("Idempotent-Replayed") != "true" || calls != 1 {
		t.Fatalf("replay inválido: first=%d second=%d replay=%q calls=%d", first.Code, second.Code, second.Header().Get("Idempotent-Replayed"), calls)
	}
	different := request(`{"plan_code":11}`)
	if different.Code != http.StatusConflict || !strings.Contains(different.Body.String(), "IDEMPOTENCY_KEY_REUSED") || calls != 1 {
		t.Fatalf("reuso divergente não bloqueado: status=%d body=%s calls=%d", different.Code, different.Body.String(), calls)
	}
	var completed bool
	if err := pool.QueryRow(context.Background(), `SELECT completed FROM http_idempotency_records WHERE scope_key LIKE $1`, "%"+key).Scan(&completed); err != nil || !completed {
		t.Fatalf("registro persistente ausente: completed=%v err=%v", completed, err)
	}
}
