package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	userrepo "github.com/FelipePn10/panossoerp/internal/domain/user/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	applogger "github.com/FelipePn10/panossoerp/internal/infrastructure/logger"
	"github.com/golang-jwt/jwt/v5"
)

type fixedAuthVersion struct {
	version int64
	role    string
	err     error
}

func TestJWTRejectsChangedRoleOrRemovedMembership(t *testing.T) {
	const secret = "test-secret"
	token, err := auth.GenerateToken("00000000-0000-0000-0000-000000000001", "ADMIN", 1, 2, secret)
	if err != nil {
		t.Fatal(err)
	}
	for name, validator := range map[string]fixedAuthVersion{
		"role changed":       {version: 2, role: "USER"},
		"membership removed": {err: errors.New("no rows")},
	} {
		t.Run(name, func(t *testing.T) {
			handler := JWT(secret, applogger.New("error"), validator)(okHandler())
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
		})
	}
}

func (v fixedAuthVersion) CurrentAuthorization(_ context.Context, _ string, enterpriseID int64) (userrepo.Authorization, error) {
	role := v.role
	if role == "" {
		role = "USER"
	}
	return userrepo.Authorization{EnterpriseID: enterpriseID, Role: role, AuthVersion: v.version}, v.err
}

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

func TestJWTRejectsUnexpectedHMACAlgorithm(t *testing.T) {
	const secret = "test-secret"
	claims := auth.UserClaims{
		UserID: "00000000-0000-0000-0000-000000000001", Role: "USER", EnterpriseID: 1, AuthVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "panosso-erp", Subject: "00000000-0000-0000-0000-000000000001",
			IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS384, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	handler := JWT(secret, applogger.New("error"))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for HS384, got %d", response.Code)
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
