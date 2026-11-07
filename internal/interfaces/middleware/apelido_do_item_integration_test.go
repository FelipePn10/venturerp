//go:build integration

package middleware

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	itemrepository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Quem digita o código antigo ou o do fornecedor num campo de item não leva o
// item errado nem um "não encontrado" seco: a mensagem diz qual é o item.
func TestCodigoAnteriorEDoFornecedorApontamOItem(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	usuario := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users(id,name,email,password) VALUES($1,'Teste apelido',$2,'x')`, usuario, usuario.String()+"@t.local"); err != nil {
		t.Fatal(err)
	}
	codigoEmpresa := int64(1_000_000_000 + testutil.UniqueCode()%500_000_000)
	outraEmpresa := codigoEmpresa + 1
	var empresa, outra, almox int64
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name,created_by) VALUES($1,'Apelido',$2) RETURNING id`, codigoEmpresa, usuario).Scan(&empresa); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name,created_by) VALUES($1,'Apelido outra',$2) RETURNING id`, outraEmpresa, usuario).Scan(&outra); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		VALUES($1,'Apelido',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`, fmt.Sprintf("W-%d", testutil.UniqueCode()), usuario, empresa).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM item_preferred_suppliers WHERE enterprise_id=$1`, empresa)
		_, _ = pool.Exec(bg, `DELETE FROM items WHERE enterprise_id=$1`, empresa)
		testutil.ApagarEmpresasPorCodigo(t, pool, codigoEmpresa, outraEmpresa)
		_, _ = pool.Exec(bg, `DELETE FROM warehouse WHERE id=$1`, almox)
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE id=$1`, usuario)
	})
	chave := testutil.UniqueCode()
	comercial, anterior, doFornecedor := fmt.Sprintf("TEA%d-3", chave), fmt.Sprintf("ANT-%d", chave), fmt.Sprintf("FORN-%d", chave)
	if _, err := pool.Exec(ctx, `INSERT INTO items(warehouse_code,code,name,created_by,enterprise_id,business_code,previous_code,engineering_weight,engineering_type)
		VALUES ($1,$2,'BUCHA',$3,$4,$5,$6,'{"gross":1,"net":1,"unit":"KG"}',1)`, almox, chave, usuario, empresa, comercial, anterior); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO item_preferred_suppliers(item_code,supplier_code,ranking,supplier_item_code,created_by,enterprise_id)
		VALUES ($1,999999,1,$2,$3,$4)`, chave, doFornecedor, usuario, empresa); err != nil {
		t.Fatal(err)
	}

	sonda := func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("chegou")) }
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
		r.Route("/api/stock", func(r chi.Router) { r.Get("/balances/item/{itemCode}", sonda) })
		r.Post("/api/sales-order/items/create", sonda)
	})

	casos := []struct {
		nome, metodo, caminho, corpo, espera string
	}{
		{"código anterior no caminho", http.MethodGet, "/api/stock/balances/item/" + anterior, "", "é o código anterior do item " + comercial},
		{"código do fornecedor no corpo", http.MethodPost, "/api/sales-order/items/create", `{"item_code":"` + doFornecedor + `"}`, "é o código do fornecedor 999999 para o item " + comercial},
	}
	for _, c := range casos {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(c.metodo, c.caminho, strings.NewReader(c.corpo))
		if c.corpo != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), c.espera) {
			t.Fatalf("%s: %d %s; esperava 422 com %q", c.nome, rec.Code, rec.Body.String(), c.espera)
		}
	}

	// O código atual continua passando, e outra empresa não enxerga os apelidos.
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/stock/balances/item/"+comercial, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("código atual: %d %s", rec.Code, rec.Body.String())
	}
	if apelidos, err := itemrepository.ApelidosDoCodigo(ctx, pool, outra, anterior); err != nil || len(apelidos) != 0 {
		t.Fatalf("outra empresa viu o apelido: %v %v", apelidos, err)
	}
}
