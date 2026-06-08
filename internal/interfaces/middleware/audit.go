package middleware

import (
	"net/http"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/audit"
	applogger "github.com/FelipePn10/panossoerp/internal/infrastructure/logger"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// Audit records every state-changing request (POST/PUT/PATCH/DELETE) to the
// audit sink, capturing the authenticated actor, the action (method + route),
// the target path, and the outcome. Read requests are ignored to keep the trail
// focused on changes. A nil sink disables auditing (pass-through).
//
// Place this AFTER the JWT middleware so the actor (user_id/role) is available
// in the request context.
func Audit(sink audit.Sink) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if sink == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isMutating(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = "unmatched"
			}
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}

			var userID, role string
			if u, ok := r.Context().Value(contextkey.UserKey).(*security.AuthUser); ok && u != nil {
				userID, role = u.ID, u.Role
			}

			sink.Record(audit.Event{
				OccurredAt: start,
				RequestID:  applogger.CorrelationIDFromContext(r.Context()),
				UserID:     userID,
				UserRole:   role,
				Method:     r.Method,
				Route:      route,
				Path:       r.URL.Path,
				Query:      r.URL.RawQuery,
				Status:     status,
				IP:         realIP(r),
				UserAgent:  r.UserAgent(),
				LatencyMS:  time.Since(start).Milliseconds(),
			})
		})
	}
}

func isMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}
