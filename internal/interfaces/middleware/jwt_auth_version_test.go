package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	applogger "github.com/FelipePn10/panossoerp/internal/infrastructure/logger"
)

type fixedAuthVersion struct{ version int64 }

func (v fixedAuthVersion) AuthVersion(context.Context, string) (int64, error) { return v.version, nil }

func TestJWTRejectsRevokedAuthVersion(t *testing.T) {
	const secret = "test-secret"
	token, err := auth.GenerateToken("00000000-0000-0000-0000-000000000001", "USER", 1, 1, secret)
	if err != nil {
		t.Fatal(err)
	}
	handler := JWT(secret, applogger.New("error"), fixedAuthVersion{version: 2})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}
}

func TestJWTAcceptsCurrentAuthVersion(t *testing.T) {
	const secret = "test-secret"
	token, err := auth.GenerateToken("00000000-0000-0000-0000-000000000001", "USER", 1, 2, secret)
	if err != nil {
		t.Fatal(err)
	}
	handler := JWT(secret, applogger.New("error"), fixedAuthVersion{version: 2})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/items", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", response.Code)
	}
}
