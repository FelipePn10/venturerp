package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

func tenantRequest(body string, enterpriseCode int64) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/resource", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), contextkey.UserKey, &security.AuthUser{EnterpriseID: 9, EnterpriseCode: enterpriseCode})
	return req.WithContext(ctx)
}

func TestTenantBodyGuardRewritesForeignEnterpriseRecursively(t *testing.T) {
	var got string
	h := TenantBodyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = string(body)
		w.WriteHeader(http.StatusNoContent)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, tenantRequest(`{"items":[{"enterprise_code":77}]}`, 42))
	if rec.Code != http.StatusNoContent || got != `{"items":[{"enterprise_code":42}]}` {
		t.Fatalf("status/body = %d/%q", rec.Code, got)
	}
}

func TestTenantBodyGuardPreservesMatchingPayload(t *testing.T) {
	var got string
	h := TenantBodyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = string(body)
		w.WriteHeader(http.StatusNoContent)
	}))
	rec := httptest.NewRecorder()
	payload := `{"enterprise_code":42,"name":"ok"}`
	h.ServeHTTP(rec, tenantRequest(payload, 42))
	if rec.Code != http.StatusNoContent || got != payload {
		t.Fatalf("status/body = %d/%q", rec.Code, got)
	}
}
