package fiscal_uc

import (
	"strings"
	"testing"
)

func TestEnderecoFornecedor_FaltandoEmOrdemESobreposicao(t *testing.T) {
	e := enderecoFornecedor{Logradouro: "RUA A", Numero: "1", Municipio: "CURITIBA"}
	if got := strings.Join(e.faltando(), ", "); got != "bairro, código do município (IBGE), CEP" {
		t.Fatalf("faltando = %q", got)
	}
	bairro, cep, cmun, vazio := "CENTRO", "80.000-000", "41 06902", "  "
	e.sobrepor(DevolucaoCompraDTO{DestBairro: &bairro, DestCEP: &cep, DestCodigoMunicipio: &cmun, DestLogradouro: &vazio})
	if len(e.faltando()) != 0 || e.CEP != "80000000" || e.CodigoMunicipio != "4106902" || e.Logradouro != "RUA A" {
		t.Fatalf("sobreposição: %+v", e)
	}
}
