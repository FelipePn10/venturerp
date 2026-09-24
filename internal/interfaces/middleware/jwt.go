package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	userrepo "github.com/FelipePn10/panossoerp/internal/domain/user/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	applogger "github.com/FelipePn10/panossoerp/internal/infrastructure/logger"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AuthorizationValidator interface {
	CurrentAuthorization(context.Context, string, int64) (userrepo.Authorization, error)
}

func JWT(secret string, log *applogger.Logger, validators ...AuthorizationValidator) func(http.Handler) http.Handler {
	return JWTForEnvironment(secret, "production", log, validators...)
}

func JWTForEnvironment(secret, environment string, log *applogger.Logger, validators ...AuthorizationValidator) func(http.Handler) http.Handler {
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
					return []byte(secret), nil
				},
				jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
				jwt.WithIssuer("panosso-erp"),
				jwt.WithExpirationRequired(),
				jwt.WithIssuedAt(),
			)

			// Subject is the canonical user identifier. Older tokens encoded the
			// same value through the embedded registered claim only.
			if claims.UserID == "" {
				claims.UserID = claims.Subject
			}
			role := strings.ToUpper(strings.TrimSpace(claims.Role))
			tokenEnvironment := strings.ToLower(strings.TrimSpace(claims.Environment))
			if tokenEnvironment == "" {
				tokenEnvironment = "production"
			}
			_, userIDErr := uuid.Parse(claims.UserID)
			if err != nil || !token.Valid || userIDErr != nil || claims.Subject != claims.UserID ||
				claims.EnterpriseID <= 0 || tokenEnvironment != environment || !perfilConhecido(role) {
				// Use the per-request logger so the warning carries request_id.
				applogger.FromContext(r.Context()).Warn(
					"invalid token attempt",
					"error", err,
					"ip", realIP(r),
				)
				http.Error(w, `{"error": "Invalid token"}`, http.StatusUnauthorized)
				return
			}

			enterpriseCode := int64(0)
			if len(validators) > 0 {
				current, validationErr := validators[0].CurrentAuthorization(r.Context(), claims.UserID, claims.EnterpriseID)
				currentRole := strings.ToUpper(strings.TrimSpace(current.Role))
				if validationErr != nil || current.EnterpriseID != claims.EnterpriseID ||
					current.AuthVersion != claims.AuthVersion || currentRole != role {
					applogger.FromContext(r.Context()).Warn("revoked token attempt", "user_id", claims.UserID, "ip", realIP(r))
					http.Error(w, `{"error": "Invalid token"}`, http.StatusUnauthorized)
					return
				}
				role = currentRole
				enterpriseCode = current.EnterpriseCode
			}

			user := &security.AuthUser{
				ID:             claims.UserID,
				Role:           role,
				EnterpriseID:   claims.EnterpriseID,
				EnterpriseCode: enterpriseCode,
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
				respondJSONErro(w, http.StatusUnauthorized, "NAO_AUTENTICADO",
					"sessão não encontrada ou expirada; entre novamente")
				return
			}

			if _, allowed := roleSet[user.Role]; !allowed {
				respondJSONErro(w, http.StatusForbidden, "ACESSO_NEGADO", fmt.Sprintf(
					"seu perfil (%s) não tem permissão para esta operação; ela exige %s. "+
						"O perfil vale por empresa — peça ao administrador para revisar o seu acesso nesta empresa.",
					perfilLegivel(user.Role), listaDePerfis(roles)))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// respondJSONErro devolve o mesmo envelope que o resto do sistema usa, para que
// a tela consiga exibir a mensagem em vez de um texto solto.
func respondJSONErro(w http.ResponseWriter, status int, code, mensagem string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "error": mensagem})
}

// perfilConhecido recusa token com papel que a aplicação não conhece — papel
// desconhecido não recebe escopo nenhum, e deixá-lo entrar só adiaria a recusa
// para dentro das rotas, com 403 em vez de 401.
//
// A lista vem de `rolePermissions`: adicionar um perfil ao mapa de permissões
// passa a valer aqui automaticamente. Antes eram duas listas independentes, e
// foi por isso que OPERATOR nasceu com escopo próprio e mesmo assim tomava 401
// antes de chegar a qualquer rota.
func perfilConhecido(role string) bool {
	_, existe := rolePermissions[role]
	return existe
}

func perfilLegivel(role string) string {
	if strings.TrimSpace(role) == "" {
		return "sem perfil definido"
	}
	return role
}

func listaDePerfis(roles []string) string {
	switch len(roles) {
	case 0:
		return "outro perfil"
	case 1:
		return "o perfil " + roles[0]
	default:
		return "um destes perfis: " + strings.Join(roles, ", ")
	}
}
