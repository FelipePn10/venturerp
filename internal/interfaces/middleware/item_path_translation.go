package middleware

import (
	"net/http"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ItemPathTranslation converte o código público do item que aparece no CAMINHO
// da URL para a chave interna, ANTES de o chi escolher a rota.
//
// Por que separado do ItemBusinessCodeCompatibility: aquele middleware é
// registrado dentro de um `r.Group`, e middleware de grupo roda DEPOIS do
// roteamento. Quando as rotas ficam atrás de um `r.Route("/api/stock", …)` — um
// mount com curinga, e há 85 deles — o chi já guardou o resto do caminho
// ("movements/item/MP-CH-3MM") num parâmetro `*` interno e é dele que o
// sub-roteador tira a sub-rota. Reescrever `r.URL.Path` naquele ponto não muda
// mais nada: o handler recebia o código público cru. Com código alfanumérico
// isso virava 400 "código do item inválido"; com código numérico, uma consulta
// pelo código errado que devolvia lista vazia — saldo e ATP zerados em silêncio.
// O log despistava a investigação, porque o logger de requisição é de raiz e
// imprime `r.URL.Path` já reescrito.
//
// Middleware de raiz (`r.Use` no mux principal) roda antes do roteamento, então
// aqui a reescrita de `r.URL.Path` é a única coisa necessária e vale para todas
// as rotas montadas.
//
// A empresa vem das claims assinadas do token. Não repetimos a checagem de
// revogação do JWTForEnvironment (que vai ao banco): ela decide se a requisição
// segue, não em qual empresa procurar o código — e o JWT de verdade continua
// rodando depois e rejeitando o que for inválido. Sem token válido, não
// traduzimos nada e deixamos a requisição seguir para tomar o 401 lá na frente.
func ItemPathTranslation(pool *pgxpool.Pool, secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			enterpriseID, ok := empresaDoTokenAssinado(r, secret)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			if !nativeItemBusinessCodePath(r) {
				if err := translateKnownItemURLPath(r, pool, enterpriseID); err != nil {
					http.Error(w, err.Error(), http.StatusUnprocessableEntity)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// empresaDoTokenAssinado lê a empresa das claims depois de conferir a
// assinatura. Devolve false para qualquer token ausente, malformado ou inválido.
func empresaDoTokenAssinado(r *http.Request, secret string) (int64, bool) {
	parts := strings.Split(r.Header.Get("Authorization"), " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return 0, false
	}
	claims := &auth.UserClaims{}
	token, err := jwt.ParseWithClaims(parts[1], claims, func(*jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid || claims.EnterpriseID <= 0 {
		return 0, false
	}
	return claims.EnterpriseID, true
}
