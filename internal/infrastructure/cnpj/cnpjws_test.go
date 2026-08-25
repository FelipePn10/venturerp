package cnpj

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const cnpjwsFixture = `{
  "razao_social": "TECNOFER FABRICACAO E MONTAGEM DE ESTRUTURAS METALICAS LTDA",
  "porte": {"descricao": "Micro Empresa"},
  "natureza_juridica": {"descricao": "Sociedade Empresária Limitada"},
  "simples": {"simples": "Não", "mei": "Não"},
  "estabelecimento": {
    "cnpj": "52454668000102",
    "nome_fantasia": "TECNOFER",
    "situacao_cadastral": "Ativa",
    "data_inicio_atividade": "2023-10-06",
    "tipo_logradouro": "RUA",
    "logradouro": "JACOB EVALDO STADLER",
    "numero": "83",
    "complemento": null,
    "bairro": "PARQUE INDUSTRIAL III",
    "cep": "86975000",
    "ddd1": "44",
    "telefone1": "98904502",
    "email": "comercial@tecnofer.com.br",
    "cidade": {"nome": "Mandaguari"},
    "estado": {"sigla": "PR"},
    "atividade_principal": {"id": "2511000", "descricao": "Fabricação de estruturas metálicas"},
    "atividades_secundarias": [{"id": "2512800", "descricao": "Fabricação de esquadrias de metal"}],
    "inscricoes_estaduais": [
      {"inscricao_estadual": "9103144679", "ativo": true, "estado": {"sigla": "PR"}},
      {"inscricao_estadual": "0123456789", "ativo": false, "estado": {"sigla": "SP"}}
    ]
  }
}`

func TestCnpjwsProviderParsesStateRegistrations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cnpj/52454668000102" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(cnpjwsFixture))
	}))
	defer srv.Close()

	p := &cnpjwsProvider{base: srv.URL, http: &http.Client{Timeout: time.Second}}
	company, err := p.Lookup(context.Background(), "52.454.668/0001-02")
	if err != nil {
		t.Fatalf("Lookup() error: %v", err)
	}

	if company.LegalName != "TECNOFER FABRICACAO E MONTAGEM DE ESTRUTURAS METALICAS LTDA" {
		t.Fatalf("LegalName = %q", company.LegalName)
	}
	if company.TradeName != "TECNOFER" || company.RegistrationStatus != "ATIVA" {
		t.Fatalf("unexpected identification: %+v", company)
	}
	if company.Size != "Micro Empresa" || company.LegalNature != "Sociedade Empresária Limitada" {
		t.Fatalf("unexpected enrichment: %+v", company)
	}
	if company.Address.City != "Mandaguari" || company.Address.UF != "PR" || company.Address.Street != "RUA JACOB EVALDO STADLER" {
		t.Fatalf("unexpected address: %+v", company.Address)
	}
	if company.Email != "comercial@tecnofer.com.br" || company.Phone != "4498904502" {
		t.Fatalf("unexpected contact: email=%q phone=%q", company.Email, company.Phone)
	}
	if company.MainActivity.Code != "2511000" {
		t.Fatalf("main activity = %+v", company.MainActivity)
	}
	if len(company.SecondaryActivities) != 1 || company.SecondaryActivities[0].Code != "2512800" {
		t.Fatalf("secondary activities = %+v", company.SecondaryActivities)
	}

	if len(company.StateRegistrations) != 2 {
		t.Fatalf("state registrations = %+v", company.StateRegistrations)
	}
	// Primary IE is the enabled registration matching the company's own UF (PR).
	if company.PrimaryStateRegistration() != "9103144679" {
		t.Fatalf("PrimaryStateRegistration() = %q", company.PrimaryStateRegistration())
	}
	if company.StateRegistrations[0].UF != "PR" || !company.StateRegistrations[0].Enabled {
		t.Fatalf("first registration = %+v", company.StateRegistrations[0])
	}
	if company.StateRegistrations[1].UF != "SP" || company.StateRegistrations[1].Enabled {
		t.Fatalf("second registration = %+v", company.StateRegistrations[1])
	}
}
