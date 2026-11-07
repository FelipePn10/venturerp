//go:build integration

package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// A Tecnofer tem itens como TP-0120/50-E. A tela manda %2F e o caminho decodificado
// partia o código em dois: "item TP-0120 não encontrado" no saldo e nos fornecedores.
func TestCodigoComBarraChegaInteiroNaRotaTraduzida(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	usuario := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users(id,name,email,password) VALUES($1,'Teste código com barra',$2,'x')`, usuario, usuario.String()+"@t.local"); err != nil {
		t.Fatal(err)
	}
	codigoEmpresa := int64(1_000_000_000 + testutil.UniqueCode()%500_000_000)
	var empresa, almox int64
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name,created_by) VALUES($1,'Código com barra',$2) RETURNING id`, codigoEmpresa, usuario).Scan(&empresa); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		VALUES($1,'Código com barra',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`, fmt.Sprintf("W-%d", testutil.UniqueCode()), usuario, empresa).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM items WHERE enterprise_id=$1`, empresa)
		testutil.ApagarEmpresasPorCodigo(t, pool, codigoEmpresa)
		_, _ = pool.Exec(bg, `DELETE FROM warehouse WHERE id=$1`, almox)
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE id=$1`, usuario)
	})
	chave := testutil.UniqueCode()
	comercial := fmt.Sprintf("TP-%d/50-E", chave)
	if _, err := pool.Exec(ctx, `INSERT INTO items(warehouse_code,code,name,created_by,enterprise_id,business_code,engineering_weight,engineering_type)
		VALUES ($1,$2,'BUCHA',$3,$4,$5,'{"gross":1,"net":1,"unit":"KG"}',2)`, almox, chave, usuario, empresa, comercial); err != nil {
		t.Fatal(err)
	}

	sonda := func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(chi.URLParam(r, "itemCode"))) }
	router := chi.NewRouter()
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, &auth.UserClaims{EnterpriseID: empresa}).SignedString([]byte("integration-only-key"))
			if err != nil {
				t.Fatal(err)
			}
			r.Header.Set("Authorization", "Bearer "+token)
			user := &security.AuthUser{EnterpriseID: empresa, EnterpriseCode: codigoEmpresa, Role: "ADMIN", ID: usuario.String()}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), contextkey.UserKey, user)))
		})
	})
	router.Use(ItemPathTranslation(pool, "integration-only-key"))
	router.Group(func(r chi.Router) {
		r.Use(ItemBusinessCodeCompatibility(pool))
		r.Get("/api/item-conversions/item/{itemCode}", sonda)
		r.Route("/api/stock", func(r chi.Router) { r.Get("/balances/item/{itemCode}", sonda) })
	})

	escapado := fmt.Sprintf("TP-%d%%2F50-E", chave)
	for _, caminho := range []string{"/api/item-conversions/item/" + escapado, "/api/stock/balances/item/" + escapado} {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, caminho, nil))
		if rec.Code != http.StatusOK || rec.Body.String() != fmt.Sprint(chave) {
			t.Fatalf("%s: %d %q, esperava a chave %d", caminho, rec.Code, rec.Body.String(), chave)
		}
	}
}
