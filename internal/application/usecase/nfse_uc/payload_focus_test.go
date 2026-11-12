package nfse_uc

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/nfse/entity"
)

func texto(s string) *string { return &s }

// A Focus recebe a alíquota em percentual e o serviço e o endereço do tomador
// aninhados. Antes ia "aliquota": 0.05 (0,05%) solto na raiz.
func TestPayloadDaNFSeNoFormatoDaFocus(t *testing.T) {
	n := &entity.NFSe{
		DataEmissao: time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC), NaturezaOperacao: 1,
		TomadorCnpjCpf: texto("11222333000181"), TomadorRazaoSocial: texto("Cliente Exemplo"),
		TomadorLogradouro: texto("Rua A"), TomadorNumero: texto("10"), TomadorCodigoMunicipio: texto("4115200"), TomadorUF: texto("PR"),
		ItemListaServico: "1401", Discriminacao: "Manutenção", CodigoMunicipio: "4115200",
		ValorServicos: decimal.NewFromInt(1000), AliquotaISS: decimal.RequireFromString("0.03"), ValorISS: decimal.NewFromInt(30),
	}
	raw, err := json.Marshal(payloadDaNFSe(n, &fiscalentity.FiscalConfig{CnpjEmpresa: "99888777000166", CodigoMunicipio: "4115200", InscricaoMunicipal: texto("123456")}))
	if err != nil {
		t.Fatal(err)
	}
	var corpo map[string]any
	if err := json.Unmarshal(raw, &corpo); err != nil {
		t.Fatal(err)
	}
	servico, ok := corpo["servico"].(map[string]any)
	if !ok {
		t.Fatalf("sem bloco servico: %s", raw)
	}
	if servico["aliquota"] != 3.0 || servico["valor_servicos"] != 1000.0 || servico["item_lista_servico"] != "1401" {
		t.Fatalf("servico = %v", servico)
	}
	if prestador := corpo["prestador"].(map[string]any); prestador["inscricao_municipal"] != "123456" {
		t.Fatalf("prestador sem inscrição municipal: %v", prestador)
	}
	tomador := corpo["tomador"].(map[string]any)
	endereco, ok := tomador["endereco"].(map[string]any)
	if !ok || endereco["logradouro"] != "Rua A" || endereco["uf"] != "PR" || tomador["cnpj"] != "11222333000181" {
		t.Fatalf("tomador = %v", tomador)
	}
	for _, solto := range []string{"aliquota", "discriminacao", "valor_servicos", "item_lista_servico"} {
		if _, existe := corpo[solto]; existe {
			t.Fatalf("%q solto na raiz do payload: %s", solto, raw)
		}
	}
}

func TestEnderecoVazioNaoVaiNoPayload(t *testing.T) {
	if e := enderecoDoTomador(&entity.NFSe{}); e != nil {
		t.Fatalf("endereço vazio = %+v", e)
	}
}

func TestAliquotaDoISSEmFracaoAte5PorCento(t *testing.T) {
	d := decimal.RequireFromString
	casos := []struct {
		aliquota, deducoes string
		recusa             string
	}{
		{"0.05", "0", ""},
		{"0.02", "100", ""},
		{"0", "0", ""},
		{"5", "0", "fração"},    // digitou o percentual
		{"0.06", "0", "fração"}, // acima do teto da LC 116
		{"-0.01", "0", "fração"},
		{"0.03", "1500", "deduções"},
		{"0.03", "-1", "deduções"},
	}
	for _, c := range casos {
		err := conferirAliquotaEDeducoes(d(c.aliquota), d("1000"), d(c.deducoes))
		switch {
		case c.recusa == "" && err != nil:
			t.Errorf("aliquota %s deduções %s: recusou: %v", c.aliquota, c.deducoes, err)
		case c.recusa != "" && (err == nil || !strings.Contains(err.Error(), c.recusa)):
			t.Errorf("aliquota %s deduções %s: erro = %v, esperado conter %q", c.aliquota, c.deducoes, err, c.recusa)
		}
	}
}
