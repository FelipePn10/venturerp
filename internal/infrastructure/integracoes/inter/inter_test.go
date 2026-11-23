package inter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
)

// Servidor falso no formato da Cobrança v3: emissão assíncrona, depois a
// consulta aninhada (cobranca / boleto / pix).
func TestEmissaoEConsulta(t *testing.T) {
	consultas := 0
	var emitido map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth/v2/token":
			_ = r.ParseForm()
			if r.Form.Get("grant_type") != "client_credentials" || !strings.Contains(r.Form.Get("scope"), "boleto-cobranca.write") {
				w.WriteHeader(400)
				return
			}
			_, _ = io.WriteString(w, `{"access_token":"tok","expires_in":3600}`)
		case r.Method == http.MethodPost && r.URL.Path == "/cobranca/v3/cobrancas":
			if r.Header.Get("Authorization") != "Bearer tok" || r.Header.Get("x-conta-corrente") != "123" {
				w.WriteHeader(401)
				return
			}
			_ = json.NewDecoder(r.Body).Decode(&emitido)
			_, _ = io.WriteString(w, `{"codigoSolicitacao":"abc-1"}`)
		case r.Method == http.MethodGet && r.URL.Path == "/cobranca/v3/cobrancas/abc-1":
			consultas++
			if consultas == 1 {
				_, _ = io.WriteString(w, `{"cobranca":{"codigoSolicitacao":"abc-1","seuNumero":"CR7","situacao":"EM_PROCESSAMENTO"}}`)
				return
			}
			_, _ = io.WriteString(w, `{"cobranca":{"codigoSolicitacao":"abc-1","seuNumero":"CR7","situacao":"RECEBIDO","dataSituacao":"2026-11-21","valorTotalRecebido":"361.90","origemRecebimento":"PIX"},
				"boleto":{"nossoNumero":"999","codigoBarras":"1234","linhaDigitavel":"0001 2"},"pix":{"txid":"tx1","pixCopiaECola":"000201PIX"}}`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	c := newClient(Config{ClientID: "id", ClientSecret: "s", ContaCorrente: "123"}, srv.URL, srv.Client())
	c.espera = time.Millisecond
	g, err := c.GerarPix(context.Background(), ports.CobrancaPix{SeuNumero: "CR7", Valor: decimal.RequireFromString("360.5"), Vencimento: time.Now().AddDate(0, 0, 10),
		Pagador: ports.Pagador{Nome: "Cliente", Documento: "11.444.777/0001-61", Logradouro: "Rua A", Cidade: "Maringá", UF: "pr", CEP: "87000-000"}})
	if err != nil {
		t.Fatal(err)
	}
	if g.Pendente || g.LinhaDigitavel != "0001 2" || g.CopiaECola != "000201PIX" || g.IDExterno != "abc-1" {
		t.Fatalf("gerado: %+v", g)
	}
	if v, ok := emitido["valorNominal"].(float64); !ok || v != 360.5 {
		t.Fatalf("valorNominal tem de ir como número: %#v", emitido["valorNominal"])
	}
	pg := emitido["pagador"].(map[string]any)
	if pg["cpfCnpj"] != "11444777000161" || pg["tipoPessoa"] != "JURIDICA" || pg["uf"] != "PR" || pg["cep"] != "87000000" {
		t.Fatalf("pagador: %+v", pg)
	}
	p, err := c.ConsultarPagamento(context.Background(), "abc-1")
	if err != nil || !p.Pago || !p.Valor.Equal(decimal.RequireFromString("361.9")) || p.SeuNumero != "CR7" || p.Forma != "PIX" {
		t.Fatalf("consulta: %+v err=%v", p, err)
	}
	if _, err := c.GerarPix(context.Background(), ports.CobrancaPix{Valor: decimal.NewFromInt(10)}); err == nil {
		t.Fatal("sem endereço do pagador deveria ser recusado antes de chamar o banco")
	}
}

// Cancelamento, cadastro do webhook e consulta de cobrança já emitida; o valor
// recebido chega como número (antes só era lido como texto).
func TestCancelarWebhookEConsulta(t *testing.T) {
	var cancelado, webhook map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/oauth/v2/token":
			_, _ = io.WriteString(w, `{"access_token":"tok","expires_in":3600}`)
		case r.Method == http.MethodPost && r.URL.Path == "/cobranca/v3/cobrancas/abc-9/cancelar":
			_ = json.NewDecoder(r.Body).Decode(&cancelado)
			w.WriteHeader(http.StatusAccepted)
		case r.Method == http.MethodPut && r.URL.Path == "/cobranca/v3/cobrancas/webhook":
			_ = json.NewDecoder(r.Body).Decode(&webhook)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/cobranca/v3/cobrancas/abc-9":
			_, _ = io.WriteString(w, `{"cobranca":{"codigoSolicitacao":"abc-9","seuNumero":"CR9","situacao":"RECEBIDO","dataSituacao":"2026-11-21","valorTotalRecebido":120.5,"origemRecebimento":"BOLETO"},
				"boleto":{"nossoNumero":"999","codigoBarras":"0779","linhaDigitavel":"07790.0"},"pix":{"txid":"t9","pixCopiaECola":"000201"}}`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	c := newClient(Config{ClientID: "id", ClientSecret: "s"}, srv.URL, srv.Client())
	ctx := context.Background()
	if err := c.CancelarCobranca(ctx, "abc-9", ""); err != nil || cancelado["motivoCancelamento"] != "Cancelado no ERP" {
		t.Fatalf("cancelamento: %v %v", err, cancelado)
	}
	if err := c.RegistrarWebhook(ctx, "http://inseguro"); err == nil {
		t.Fatal("webhook sem HTTPS precisa ser recusado")
	}
	if err := c.RegistrarWebhook(ctx, "https://api.exemplo/api/integracoes/inter/webhook/seg"); err != nil || webhook["webhookUrl"] == "" {
		t.Fatalf("webhook: %v %v", err, webhook)
	}
	g, err := c.ConsultarCobranca(ctx, "abc-9")
	if err != nil || g.Pendente || g.LinhaDigitavel != "07790.0" || g.CopiaECola != "000201" || g.Situacao != "RECEBIDO" {
		t.Fatalf("consulta: %+v err=%v", g, err)
	}
	p, err := c.ConsultarPagamento(ctx, "abc-9")
	if err != nil || !p.Pago || p.Valor.String() != "120.5" {
		t.Fatalf("pagamento com valor numérico: %+v err=%v", p, err)
	}
}
