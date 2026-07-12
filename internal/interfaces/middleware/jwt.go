package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	applogger "github.com/FelipePn10/panossoerp/internal/infrastructure/logger"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/golang-jwt/jwt/v5"
)

func JWT(secret string, log *applogger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error": "Authorization header missing"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, `{"error": "Invalid auth header format"}`, http.StatusUnauthorized)
				return
			}

			claims := &auth.UserClaims{}
			token, err := jwt.ParseWithClaims(
				parts[1],
				claims,
				func(t *jwt.Token) (interface{}, error) {
					if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, jwt.ErrSignatureInvalid
					}
					return []byte(secret), nil
				},
			)

			if err != nil || !token.Valid {
				// Use the per-request logger so the warning carries request_id.
				applogger.FromContext(r.Context()).Warn(
					"invalid token attempt",
					"error", err,
					"ip", realIP(r),
				)
				http.Error(w, `{"error": "Invalid token"}`, http.StatusUnauthorized)
				return
			}

			user := &security.AuthUser{
				ID:           claims.UserID,
				Role:         claims.Role,
				EnterpriseID: claims.EnterpriseID,
			}

			// Store user in context and propagate user_id to the logger.
			ctx := context.WithValue(r.Context(), contextkey.UserKey, user)
			ctx = applogger.WithUserID(ctx, claims.UserID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	roleSet := make(map[string]struct{})
	for _, r := range roles {
		roleSet[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := r.Context().Value(contextkey.UserKey).(*security.AuthUser)
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			if _, allowed := roleSet[user.Role]; !allowed {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
