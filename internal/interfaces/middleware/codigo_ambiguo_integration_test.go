//go:build integration

package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/itemresolution"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/routing_uc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/auth"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	itemrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/item"
	routingrepo "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/routing"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/testutil"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
	"github.com/FelipePn10/panossoerp/internal/interfaces/http/handler"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type authDoRoteiro struct {
	ports.AuthService
	usuario uuid.UUID
}

func (a authDoRoteiro) UserID(context.Context) (uuid.UUID, error) { return a.usuario, nil }

// A base da Tecnofer em miniatura: o item fabricado tem código comercial
// alfanumérico e chave interna K; uma chapa tem como código comercial
// exatamente o texto de K. Toda leitura do item fabricado que passa pelo
// middleware (corpo, query string ou caminho) tem de chegar no item fabricado.
//
// Defeito de produção (05/11/2026, VENT0202): o roteiro era criado pelo corpo
// (registrado) e listado pela query string (sem registro) — a lista lia a chapa
// e vinha vazia, e cada nova tentativa da alternativa 1 levava 409.
func TestCodigoComercialNumericoIgualAChaveInternaNaoTrocaOItem(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	usuario := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO users(id,name,email,password) VALUES($1,'Teste código ambíguo',$2,'x')`, usuario, usuario.String()+"@test.local"); err != nil {
		t.Fatal(err)
	}
	codigoEmpresa := int64(1_000_000_000 + testutil.UniqueCode()%500_000_000)
	var empresa int64
	if err := pool.QueryRow(ctx, `INSERT INTO enterprise(code,name,created_by) VALUES($1,'Código ambíguo',$2) RETURNING id`, codigoEmpresa, usuario).Scan(&empresa); err != nil {
		t.Fatal(err)
	}
	var almox int64
	if err := pool.QueryRow(ctx, `INSERT INTO warehouse(code,description,created_by,location,type,disposition,reservations_allowed,enterprise_id)
		VALUES($1,'Código ambíguo',$2,'INTERNO','NORMAL',TRUE,TRUE,$3) RETURNING id`, fmt.Sprintf("W-%d", testutil.UniqueCode()), usuario, empresa).Scan(&almox); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM manufacturing_routes WHERE enterprise_id=$1`, empresa)
		_, _ = pool.Exec(bg, `DELETE FROM items WHERE enterprise_id=$1`, empresa)
		testutil.ApagarEmpresasPorCodigo(t, pool, codigoEmpresa)
		_, _ = pool.Exec(bg, `DELETE FROM warehouse WHERE id=$1`, almox)
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE id=$1`, usuario)
	})

	fabricado, chapa, servico := testutil.UniqueCode(), testutil.UniqueCode(), testutil.UniqueCode()
	comercialFabricado := fmt.Sprintf("RN-%d", fabricado)
	comercialChapa := strconv.FormatInt(fabricado, 10) // o comercial da chapa É a chave do fabricado
	comercialServico := fmt.Sprintf("SV-%d", servico)
	if _, err := pool.Exec(ctx, `INSERT INTO items(warehouse_code,code,name,created_by,enterprise_id,business_code,engineering_weight,engineering_type)
		VALUES ($1,$2,'BALANÇA (fabricado)',$5,$6,$7,'{"gross":1,"net":1,"unit":"KG"}',0),
		       ($1,$3,'CHAPA 3MM',$5,$6,$8,'{"gross":1,"net":1,"unit":"KG"}',2),
		       ($1,$4,'SERVIÇO',$5,$6,$9,'{"gross":1,"net":1,"unit":"KG"}',2)`,
		almox, fabricado, chapa, servico, usuario, empresa, comercialFabricado, comercialChapa, comercialServico); err != nil {
		t.Fatal(err)
	}

	queries := sqlc.New(pool)
	itens := itemrepo.NewRepositoryItemSQLC(queries)
	rRepo := routingrepo.New(queries)
	roteiros := handler.NewRoutingHandler(
		routing_uc.NewOperationUseCase(rRepo, itens),
		routing_uc.NewRouteUseCase(rRepo, itens, authDoRoteiro{usuario: usuario}),
		routing_uc.NewLeadTimeUseCase(rRepo), nil)

	// sonda: devolve o código comercial do item que o caso de uso enxergou.
	sonda := func(valor func(*http.Request) string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			item, err := itemresolution.Resolve(r.Context(), itens, request.TextCode(valor(r)))
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnprocessableEntity)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"resolvido": string(item.BusinessCode)})
		}
	}
	doCorpo := func(campo string) func(*http.Request) string {
		return func(r *http.Request) string {
			var corpo map[string]any
			_ = json.NewDecoder(r.Body).Decode(&corpo)
			switch v := corpo[campo].(type) {
			case string:
				return v
			case float64:
				return strconv.FormatInt(int64(v), 10)
			}
			return ""
		}
	}

	// Mesma montagem da API: tradução do caminho na raiz, compatibilidade no grupo.
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
		r.Route("/api/routing", func(r chi.Router) {
			r.Route("/routes", func(r chi.Router) {
				r.Post("/", roteiros.CreateRoute)
				r.Get("/", roteiros.ListRoutesByItem)
			})
		})
		r.Route("/api/item-conversions", func(r chi.Router) {
			r.Get("/item/{itemCode}", sonda(func(r *http.Request) string { return chi.URLParam(r, "itemCode") }))
		})
		r.Get("/api/sonda", sonda(func(r *http.Request) string { return r.URL.Query().Get("item_code") }))
		r.Post("/api/sonda", sonda(doCorpo("item_code")))
		r.Post("/api/items/structure", sonda(doCorpo("parent_code")))
	})

	chamar := func(t *testing.T, metodo, caminho, corpo string) (int, string) {
		t.Helper()
		var req *http.Request
		if corpo != "" {
			req = httptest.NewRequest(metodo, caminho, strings.NewReader(corpo))
			req.Header.Set("Content-Type", "application/json")
		} else {
			req = httptest.NewRequest(metodo, caminho, nil)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code, rec.Body.String()
	}
	esperaResolvido := func(t *testing.T, metodo, caminho, corpo, quer string) {
		t.Helper()
		status, body := chamar(t, metodo, caminho, corpo)
		if status != http.StatusOK || !strings.Contains(body, `"resolvido":"`+quer+`"`) {
			t.Fatalf("%s %s %s: esperava %s, veio status=%d body=%s", metodo, caminho, corpo, quer, status, body)
		}
	}

	t.Run("VENT0202: o roteiro criado aparece na lista do item", func(t *testing.T) {
		status, body := chamar(t, http.MethodPost, "/api/routing/routes",
			fmt.Sprintf(`{"item_code":%q,"alternative":1,"description":"Processo padrão","is_standard":true}`, comercialFabricado))
		if status != http.StatusCreated {
			t.Fatalf("criar roteiro: status=%d body=%s", status, body)
		}
		status, body = chamar(t, http.MethodGet, "/api/routing/routes?incluir_inativos=true&item_code="+comercialFabricado, "")
		if status != http.StatusOK {
			t.Fatalf("listar: status=%d body=%s", status, body)
		}
		var lista []map[string]any
		if err := json.Unmarshal([]byte(body), &lista); err != nil {
			t.Fatalf("lista inválida: %v (%s)", err, body)
		}
		if len(lista) != 1 {
			t.Fatalf("a lista do item fabricado deveria trazer o roteiro criado; veio %d: %s", len(lista), body)
		}
		if lista[0]["item_code"] != comercialFabricado {
			t.Fatalf("o roteiro listado deveria ser do %s; veio %v", comercialFabricado, lista[0]["item_code"])
		}
		// A chapa não tem roteiro: a lista dela continua vazia.
		_, body = chamar(t, http.MethodGet, "/api/routing/routes?item_code="+comercialChapa, "")
		if strings.TrimSpace(body) != "[]" {
			t.Fatalf("a chapa não tem roteiro; veio %s", body)
		}
		// E a segunda tentativa da alternativa 1 continua recusada — agora com o
		// roteiro visível na tela, não escondido.
		status, _ = chamar(t, http.MethodPost, "/api/routing/routes",
			fmt.Sprintf(`{"item_code":%q,"alternative":1}`, comercialFabricado))
		if status != http.StatusConflict {
			t.Fatalf("alternativa repetida deveria dar 409; veio %d", status)
		}
	})

	t.Run("query string", func(t *testing.T) {
		esperaResolvido(t, http.MethodGet, "/api/sonda?item_code="+comercialFabricado, "", comercialFabricado)
		esperaResolvido(t, http.MethodGet, "/api/sonda?item_code="+comercialChapa, "", comercialChapa)
	})
	t.Run("caminho da URL", func(t *testing.T) {
		esperaResolvido(t, http.MethodGet, "/api/item-conversions/item/"+comercialFabricado, "", comercialFabricado)
		esperaResolvido(t, http.MethodGet, "/api/item-conversions/item/"+comercialChapa, "", comercialChapa)
	})
	t.Run("corpo", func(t *testing.T) {
		esperaResolvido(t, http.MethodPost, "/api/sonda", fmt.Sprintf(`{"item_code":%q}`, comercialFabricado), comercialFabricado)
		esperaResolvido(t, http.MethodPost, "/api/sonda", fmt.Sprintf(`{"item_code":%q}`, comercialChapa), comercialChapa)
		// Contrato legado: número JSON já é a chave interna.
		esperaResolvido(t, http.MethodPost, "/api/sonda", fmt.Sprintf(`{"item_code":%d}`, fabricado), comercialFabricado)
	})
	t.Run("campo deixado sem tradução continua comercial", func(t *testing.T) {
		// parent_code da estrutura não é traduzido pelo middleware; outro campo
		// do mesmo corpo é. A marca antiga, que valia para a requisição
		// inteira, fazia o "K" comercial (a chapa) virar a chave K (o fabricado).
		esperaResolvido(t, http.MethodPost, "/api/items/structure",
			fmt.Sprintf(`{"parent_code":%q,"service_item_code":%q}`, comercialChapa, comercialServico), comercialChapa)
	})
}
