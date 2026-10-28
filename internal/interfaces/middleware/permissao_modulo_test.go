package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/FelipePn10/panossoerp/internal/application/security"
	"github.com/FelipePn10/panossoerp/internal/domain/acesso"
	contextkey "github.com/FelipePn10/panossoerp/internal/interfaces/http/context"
)

func TestClassificarRota(t *testing.T) {
	casos := []struct {
		metodo, caminho string
		modulo          acesso.Modulo
		acao            acesso.Acao
		referencia      bool
	}{
		{"GET", "/api/sales-order/list", acesso.Comercial, acesso.Ler, false},
		{"POST", "/api/sales-order/12/release", acesso.Comercial, acesso.Aprovar, false},
		{"POST", "/api/fiscal/exits/9/authorize", acesso.Fiscal, acesso.Autorizar, false},
		{"POST", "/api/fiscal/exits/create", acesso.Fiscal, acesso.IncluirAlterar, false},
		{"DELETE", "/api/fiscal/reforma/cfop/5910", acesso.Fiscal, acesso.Excluir, false},
		{"GET", "/api/fiscal/support/cfops/", acesso.Fiscal, acesso.Ler, true},
		{"POST", "/api/procurement/receiving-inspection-orders/3/results", acesso.Qualidade, acesso.IncluirAlterar, false},
		{"POST", "/api/procurement/records", acesso.Suprimentos, acesso.IncluirAlterar, false},
		{"GET", "/api/stock-remnants", acesso.Estoque, acesso.Ler, false},
		{"GET", "/api/items/search", acesso.Engenharia, acesso.Ler, true},
		{"PUT", "/api/items/10", acesso.Engenharia, acesso.IncluirAlterar, false},
		{"POST", "/api/planning/run-pipeline", acesso.Planejamento, acesso.Executar, false},
		{"PUT", "/api/planning/auto-run", acesso.Planejamento, acesso.Executar, false},
		{"POST", "/api/purchase-order/7/approve", acesso.Suprimentos, acesso.Aprovar, false},
		{"POST", "/api/quality/amostragem/estados/liberar", acesso.Qualidade, acesso.Aprovar, false},
		{"POST", "/api/quality/amostragem/procedimentos", acesso.Qualidade, acesso.IncluirAlterar, false},
		{"GET", "/api/quality/amostragem/calcular", acesso.Qualidade, acesso.Ler, false},
		{"POST", "/api/quality/lotes/9/decisao", acesso.Qualidade, acesso.Aprovar, false},
		{"POST", "/api/quality/lotes/9/cancelar", acesso.Qualidade, acesso.Excluir, false},
		{"POST", "/api/quality/lotes/9/resultados", acesso.Qualidade, acesso.IncluirAlterar, false},
		{"POST", "/api/quality/capas/3/encerrar", acesso.Qualidade, acesso.Aprovar, false},
		{"POST", "/api/quality/capas/3/avancar", acesso.Qualidade, acesso.IncluirAlterar, false},
	}
	for _, c := range casos {
		got := ClassificarRota(c.metodo, c.caminho)
		if !got.Conhecida || got.Modulo != c.modulo || got.Acao != c.acao || got.LeituraReferencia != c.referencia {
			t.Errorf("%s %s: veio %+v", c.metodo, c.caminho, got)
		}
	}
	if !ClassificarRota("POST", "/api/password-change-requests/").Livre || !ClassificarRota("GET", "/api/acesso/minhas-permissoes").Livre {
		t.Error("pedido de troca de senha e minhas permissões são livres")
	}
	if ClassificarRota("GET", "/api/nao-existe").Conhecida {
		t.Error("rota desconhecida não pode ser classificada")
	}
}

type fonteFake struct {
	a       acesso.AcessoDoUsuario
	leitura int
}

func (f *fonteFake) AcessoDoUsuario(context.Context, uuid.UUID, int64) (acesso.AcessoDoUsuario, error) {
	f.leitura++
	return f.a, nil
}

func chamar(t *testing.T, c *ControleDeAcesso, papel, metodo, caminho string) *httptest.ResponseRecorder {
	t.Helper()
	h := c.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }))
	req := httptest.NewRequest(metodo, caminho, nil)
	user := &security.AuthUser{ID: uuid.NewString(), Role: papel, EnterpriseID: 7}
	req = req.WithContext(context.WithValue(req.Context(), contextkey.UserKey, user))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestControleDeAcessoPorPerfil(t *testing.T) {
	vendas := acesso.Permissoes{acesso.Comercial: {acesso.Ler: true, acesso.IncluirAlterar: true}}
	fonte := &fonteFake{a: acesso.AcessoDoUsuario{Restrito: true, Perfis: []string{"Vendas"}, Permissoes: vendas}}
	c := NewControleDeAcesso(fonte)

	if rec := chamar(t, c, "USER", "POST", "/api/sales-order/create"); rec.Code != http.StatusOK {
		t.Fatalf("vendas inclui pedido: %d", rec.Code)
	}
	if rec := chamar(t, c, "USER", "GET", "/api/items/search"); rec.Code != http.StatusOK {
		t.Fatalf("pesquisa de item é leitura de referência: %d", rec.Code)
	}
	rec := chamar(t, c, "USER", "POST", "/api/sales-order/1/release")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("vendas sem APROVAR não libera pedido: %d", rec.Code)
	}
	var corpo map[string]string
	_ = json.Unmarshal(rec.Body.Bytes(), &corpo)
	if corpo["code"] != "ACESSO_NEGADO" || !strings.Contains(corpo["error"], "aprovar") || !strings.Contains(corpo["error"], "Vendas") {
		t.Fatalf("mensagem do bloqueio: %v", corpo)
	}
	if rec := chamar(t, c, "USER", "PUT", "/api/items/10"); rec.Code != http.StatusForbidden {
		t.Fatalf("vendas não altera item: %d", rec.Code)
	}
	if rec := chamar(t, c, "ADMIN", "POST", "/api/fiscal/exits/9/authorize"); rec.Code != http.StatusOK {
		t.Fatalf("ADMIN não é restringido: %d", rec.Code)
	}
	if rec := chamar(t, c, "USER", "POST", "/api/password-change-requests"); rec.Code != http.StatusOK {
		t.Fatalf("troca de senha é livre: %d", rec.Code)
	}

	semPerfil := NewControleDeAcesso(&fonteFake{a: acesso.AcessoDoUsuario{}})
	if rec := chamar(t, semPerfil, "USER", "POST", "/api/fiscal/exits/9/authorize"); rec.Code != http.StatusOK {
		t.Fatalf("usuário sem perfil segue com o papel: %d", rec.Code)
	}
}

func TestControleDeAcessoGuardaEInvalida(t *testing.T) {
	fonte := &fonteFake{a: acesso.AcessoDoUsuario{}}
	c := NewControleDeAcesso(fonte)
	user := &security.AuthUser{ID: uuid.NewString(), Role: "USER", EnterpriseID: 3}
	for i := 0; i < 3; i++ {
		if _, err := c.AcessoDe(context.Background(), user); err != nil {
			t.Fatal(err)
		}
	}
	if fonte.leitura != 1 {
		t.Fatalf("deveria ler uma vez e guardar: %d", fonte.leitura)
	}
	c.Invalidar()
	_, _ = c.AcessoDe(context.Background(), user)
	if fonte.leitura != 2 {
		t.Fatalf("depois de invalidar deveria ler de novo: %d", fonte.leitura)
	}
}
