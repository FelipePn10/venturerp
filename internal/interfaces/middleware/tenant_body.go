package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

// TenantBodyGuard makes the revalidated JWT tenant authoritative for every
// enterprise_code in an authenticated JSON payload. Rewriting instead of
// rejecting keeps already-installed clients compatible while preventing them
// from selecting another tenant.
func TenantBodyGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil || r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		user, ok := r.Context().Value(contextkey.UserKey).(*security.AuthUser)
		if !ok || user.EnterpriseCode <= 0 {
			http.Error(w, `{"error":"invalid tenant context"}`, http.StatusUnauthorized)
			return
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		if len(bytes.TrimSpace(raw)) == 0 || !strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "json") {
			r.Body = io.NopCloser(bytes.NewReader(raw))
			next.ServeHTTP(w, r)
			return
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		var payload any
		if err := decoder.Decode(&payload); err != nil {
			r.Body = io.NopCloser(bytes.NewReader(raw))
			next.ServeHTTP(w, r)
			return
		}
		if rewriteTenantCodes(payload, user.EnterpriseCode) {
			raw, err = json.Marshal(payload)
			if err != nil {
				http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
				return
			}
			r.ContentLength = int64(len(raw))
		}
		r.Body = io.NopCloser(bytes.NewReader(raw))
		next.ServeHTTP(w, r)
	})
}

func rewriteTenantCodes(value any, expected int64) bool {
	changed := false
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if strings.EqualFold(key, "enterprise_code") {
				if tenantCode(child) != expected {
					typed[key] = json.Number(strconv.FormatInt(expected, 10))
					changed = true
				}
				continue
			}
			if rewriteTenantCodes(child, expected) {
				changed = true
			}
		}
	case []any:
		for _, child := range typed {
			if rewriteTenantCodes(child, expected) {
				changed = true
			}
		}
	}
	return changed
}

func tenantCode(value any) int64 {
	switch typed := value.(type) {
	case json.Number:
		code, _ := typed.Int64()
		return code
	case string:
		code, _ := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return code
	default:
		return 0
	}
}
