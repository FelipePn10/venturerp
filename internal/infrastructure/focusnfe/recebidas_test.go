package focusnfe

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// A listagem da distribuição DF-e pede a partir da versão informada, lê a
// maior versão do cabeçalho X-Max-Version e nunca devolve versão menor que a
// pedida (um cabeçalho ausente não pode fazer a sincronização recomeçar).
func TestListarNFesRecebidas(t *testing.T) {
	t.Parallel()
	var pediu string
	c := NewClient("tok", "homologacao")
	c.httpCli = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		pediu = r.URL.RequestURI()
		if u, _, ok := r.BasicAuth(); !ok || u != "tok" {
			t.Errorf("token não enviado na autenticação básica")
		}
		h := http.Header{"Content-Type": []string{"application/json"}}
		if r.URL.Query().Get("versao") == "10" {
			h.Set("X-Max-Version", "42")
		}
		return &http.Response{StatusCode: 200, Header: h, Body: io.NopCloser(bytes.NewBufferString(
			`[{"chave_nfe":"41261012345678000190550010000123451000123459","nome_emitente":"ACO","documento_emitente":"12345678000190","valor_total":"100000.00","data_emissao":"2026-10-01T09:30:00-03:00","situacao":"autorizada","manifestacao_destinatario":null,"nfe_completa":false,"versao":41}]`))}, nil
	})}
	lista, max, err := c.ListarNFesRecebidas(context.Background(), "12.345.678/0001-90", 10)
	if err != nil {
		t.Fatal(err)
	}
	if pediu != "/v2/nfes_recebidas?cnpj=12345678000190&versao=10" {
		t.Fatalf("URL pedida: %s", pediu)
	}
	if len(lista) != 1 || max != 42 || lista[0].ValorTotal.String() != "100000.00" {
		t.Fatalf("lista=%+v max=%d", lista, max)
	}
	// Sem cabeçalho: vale a maior versão dos documentos, nunca menos que a pedida.
	_, max, err = c.ListarNFesRecebidas(context.Background(), "12345678000190", 99)
	if err != nil {
		t.Fatal(err)
	}
	if max != 99 {
		t.Fatalf("versão andou para trás: %d", max)
	}
}

func TestListarNFesRecebidasErroDaAPI(t *testing.T) {
	t.Parallel()
	c := NewClient("tok", "homologacao")
	c.httpCli = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 403, Header: http.Header{}, Body: io.NopCloser(bytes.NewBufferString(`{"codigo":"permissao_negada","mensagem":"CNPJ não habilitado"}`))}, nil
	})}
	if _, _, err := c.ListarNFesRecebidas(context.Background(), "12345678000190", 0); err == nil || !bytes.Contains([]byte(err.Error()), []byte("CNPJ não habilitado")) {
		t.Fatalf("erro da API deveria trazer a mensagem: %v", err)
	}
}

// A manifestação vai para /nfes_recebidas/{chave}/manifesto com tipo e,
// quando houver, a justificativa.
func TestManifestarDestinatario(t *testing.T) {
	t.Parallel()
	var metodo, caminho string
	var corpo map[string]string
	c := NewClient("tok", "homologacao")
	c.httpCli = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		metodo, caminho = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&corpo)
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewBufferString(`{"status_sefaz":"135","mensagem_sefaz":"Evento registrado"}`))}, nil
	})}
	if _, err := c.ManifestarDestinatario(context.Background(), ManifestacaoPayload{ChaveNFe: "4126 1012 3456 7800 0190 5500 1000 0123 4510 0012 3459", Tipo: "desconhecimento", Justificativa: "operação não reconhecida"}); err != nil {
		t.Fatal(err)
	}
	if metodo != http.MethodPost || caminho != "/v2/nfes_recebidas/41261012345678000190550010000123451000123459/manifesto" {
		t.Fatalf("%s %s", metodo, caminho)
	}
	if corpo["tipo"] != "desconhecimento" || corpo["justificativa"] != "operação não reconhecida" {
		t.Fatalf("corpo: %+v", corpo)
	}
}
