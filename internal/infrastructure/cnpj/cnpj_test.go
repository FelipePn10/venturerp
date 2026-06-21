package cnpj

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/cnpj/service"
)

const validCNPJ = "19131243000197" // Receita Federal sample (valid check digits)

func TestBrasilAPIProviderParses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"cnpj":"19131243000197",
			"razao_social":"OPEN KNOWLEDGE BRASIL",
			"nome_fantasia":"REDE PELO CONHECIMENTO LIVRE",
			"descricao_situacao_cadastral":"ativa",
			"cep":"01310100","logradouro":"AV PAULISTA","numero":"37","bairro":"BELA VISTA",
			"municipio":"SAO PAULO","uf":"SP",
			"cnae_fiscal":9430800,"cnae_fiscal_descricao":"Atividades associativas",
			"opcao_pelo_simples":false
		}`))
	}))
	defer srv.Close()

	p := New(Config{Provider: "brasilapi", BrasilAPIURL: srv.URL, Timeout: 2 * time.Second})
	c, err := p.Lookup(context.Background(), "19.131.243/0001-97")
	if err != nil {
		t.Fatal(err)
	}
	if c.LegalName != "OPEN KNOWLEDGE BRASIL" {
		t.Errorf("legal name = %q", c.LegalName)
	}
	if c.Address.City != "SAO PAULO" || c.Address.UF != "SP" {
		t.Errorf("bad address: %+v", c.Address)
	}
	if c.MainActivity.Code != "9430800" {
		t.Errorf("cnae = %q", c.MainActivity.Code)
	}
	if c.Source != "brasilapi" {
		t.Errorf("source = %q", c.Source)
	}
}

func TestCNPJaProviderReturnsStateRegistration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"taxId":"19131243000197",
			"alias":"REDE",
			"company":{"name":"OPEN KNOWLEDGE BRASIL","size":{"acronym":"EPP"}},
			"status":{"text":"Ativa"},
			"address":{"zip":"01310100","street":"AV PAULISTA","number":"37","city":"SAO PAULO","state":"SP"},
			"registrations":[{"state":"SP","number":"111222333444","enabled":true}]
		}`))
	}))
	defer srv.Close()

	p := New(Config{Provider: "cnpja", CNPJaURL: srv.URL, Timeout: 2 * time.Second})
	c, err := p.Lookup(context.Background(), validCNPJ)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.PrimaryStateRegistration(); got != "111222333444" {
		t.Errorf("IE = %q", got)
	}
}

// TestAutoFallsBackToBrasilAPI verifies the chain degrades to BrasilAPI when the
// IE-capable primary is unavailable (e.g. rate-limited).
func TestAutoFallsBackToBrasilAPI(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer primary.Close()
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"cnpj":"19131243000197","razao_social":"FALLBACK LTDA","uf":"PR","municipio":"MARINGA"}`))
	}))
	defer fallback.Close()

	p := New(Config{Provider: "auto", CNPJaURL: primary.URL, BrasilAPIURL: fallback.URL, Timeout: 2 * time.Second})
	c, err := p.Lookup(context.Background(), validCNPJ)
	if err != nil {
		t.Fatal(err)
	}
	if c.LegalName != "FALLBACK LTDA" || c.Source != "brasilapi" {
		t.Errorf("expected brasilapi fallback, got %+v", c)
	}
}

func TestNotFoundPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	p := New(Config{Provider: "auto", CNPJaURL: srv.URL, BrasilAPIURL: srv.URL, Timeout: 2 * time.Second})
	if _, err := p.Lookup(context.Background(), validCNPJ); err != service.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
