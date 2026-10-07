package focusnfe

import (
	"encoding/json"
	"testing"
)

// O formato da API v2 da Focus é plano: o payload aninhado (emitente{},
// destinatario{}, forma_pagamento) era recusado com "CNPJ do emitente não
// autorizado" — conferido contra a homologação em 07/10/2026.
func TestNFEPayloadFormatoFocus(t *testing.T) {
	ie := "123.456.789"
	p := NFEPayload{
		NaturezaOperacao: "Venda", DataEmissao: "2026-10-07T10:00:00-03:00", TipoDocumento: 1, LocalDestino: 1, FinalidadeEmissao: 1,
		Emitente:     NFEEmitente{CNPJ: "52.454.668/0001-02", Nome: "TECNOFER"},
		Destinatario: NFEDestinatario{CNPJCPF: "111.222.333-44", Nome: "FULANO", IndicadorIE: 1, IE: &ie, CEP: "80.000-000", Municipio: "CURITIBA", UF: "PR"},
		Items: []NFEItem{{NumeroItem: 1, CodigoProduto: "10", Descricao: "X", CFOP: "5102", CodigoNCM: "73181500", UnidadeComercial: "UN",
			QuantidadeComercial: 2, ValorUnitarioComercial: 5, ValorBruto: 10, CodigoSituacaoTributariaICMS: "00", ModalidadeBaseCalculoICMS: 3,
			ValorBaseCalculoICMS: 10, AliquotaICMS: 18, ValorICMS: 1.8, CodigoSituacaoTributariaPIS: "01", AliquotaPIS: 1.65, ValorPIS: 0.17,
			CodigoSituacaoTributariaCOFINS: "01", AliquotaCOFINS: 7.6, ValorCOFINS: 0.76, CodigoSituacaoTributariaIPI: "53", ValorFrete: 3,
			IBSCBS: &NFEItemIBSCBS{CST: "000", ClassTrib: "000001", Base: 7.27, AliqIBSUF: 0.1, ValorIBSUF: 0.01, AliqCBS: 0.9, ValorCBS: 0.07, ValorIBS: 0.01}},
			{NumeroItem: 2, CodigoSituacaoTributariaICMS: "40", CodigoSituacaoTributariaPIS: "07", CodigoSituacaoTributariaCOFINS: "07"}},
		FormaPagamento:     []NFEFormaPagamento{{FormaPagamento: "90", Valor: 0}},
		NotasReferenciadas: []NFERef{{ChaveNFe: "41261012345678000190550010000123451000123459"}},
		ValorFrete:         3, ValorTotal: 13, ValorProdutos: 10, ModalidadeFrete: 0,
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	for k, want := range map[string]any{"cnpj_emitente": "52454668000102", "cpf_destinatario": "11122233344", "nome_destinatario": "FULANO",
		"inscricao_estadual_destinatario": "123456789", "indicador_inscricao_estadual_destinatario": 1.0, "cep_destinatario": "80000000",
		"valor_frete": 3.0, "valor_total": 13.0, "modalidade_frete": 0.0} {
		if m[k] != want {
			t.Errorf("%s = %v, want %v", k, m[k], want)
		}
	}
	for _, k := range []string{"emitente", "destinatario", "forma_pagamento", "cnpj_destinatario"} {
		if _, ok := m[k]; ok {
			t.Errorf("campo %q não existe na API da Focus", k)
		}
	}
	fp := m["formas_pagamento"].([]any)[0].(map[string]any)
	if fp["forma_pagamento"] != "90" || fp["valor_pagamento"] != 0.0 {
		t.Errorf("formas_pagamento = %v", fp)
	}
	if ref := m["notas_referenciadas"].([]any)[0].(map[string]any); ref["chave_nfe"] == "" {
		t.Errorf("notas_referenciadas = %v", ref)
	}
	it := m["items"].([]any)[0].(map[string]any)
	for k, want := range map[string]any{"icms_situacao_tributaria": "00", "icms_aliquota": 18.0, "icms_origem": 0.0, "unidade_tributavel": "UN",
		"quantidade_tributavel": 2.0, "valor_frete": 3.0, "pis_base_calculo": 10.0, "ipi_situacao_tributaria": "53", "ipi_codigo_enquadramento_legal": "999",
		"ibs_cbs_situacao_tributaria": "000", "ibs_cbs_classificacao_tributaria": "000001", "cbs_aliquota": 0.9, "ibs_uf_valor": 0.01} {
		if it[k] != want {
			t.Errorf("item %s = %v, want %v", k, it[k], want)
		}
	}
	for _, k := range []string{"codigo_situacao_tributaria_icms", "origem_mercadoria", "ipi_valor"} {
		if _, ok := it[k]; ok {
			t.Errorf("item com campo %q (não é da Focus ou não se aplica)", k)
		}
	}
	// CST 40 (isento) não leva base/alíquota/valor de ICMS; sem PIS não leva base.
	it2 := m["items"].([]any)[1].(map[string]any)
	for _, k := range []string{"icms_base_calculo", "icms_valor", "pis_base_calculo", "ibs_cbs_situacao_tributaria"} {
		if _, ok := it2[k]; ok {
			t.Errorf("item 2 não deveria ter %q", k)
		}
	}
}

func TestNFEResponseChaveEStatus(t *testing.T) {
	var r NFEResponse
	_ = json.Unmarshal([]byte(`{"status":"autorizado","chave_nfe":"NFe41261052454668000102550010000000191556398072","numero":"19","serie":"1","caminho_danfe":"/a.pdf","caminho_xml_nota_fiscal":"/a.xml","status_sefaz":"100"}`), &r)
	if !r.Autorizada() || r.Chave() != "41261052454668000102550010000000191556398072" || r.Numero != "19" || r.PathDANFE != "/a.pdf" || r.PathXML != "/a.xml" {
		t.Fatalf("resposta = %+v", r)
	}
}

func TestNFEPayloadReferenciaNumNivelSo(t *testing.T) {
	p := NFEPayload{FinalidadeEmissao: 4, NotasReferenciadas: []NFERef{{ChaveNFe: "41261052454668000102550010000000191556398072"}},
		Items: []NFEItem{{NumeroItem: 1, DFeRefChave: "41261052454668000102550010000000191556398072", DFeRefItem: 2}}}
	b, _ := json.Marshal(p)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if _, ok := m["notas_referenciadas"]; ok {
		t.Error("com referência por item, a da nota não vai (rejeição 1010)")
	}
	it := m["items"].([]any)[0].(map[string]any)
	if it["chave_acesso_dfe_referenciado"] != "41261052454668000102550010000000191556398072" || it["numero_item_dfe_referenciado"] != "2" {
		t.Errorf("item = %v", it)
	}
	p.Items[0].DFeRefChave = ""
	b, _ = json.Marshal(p)
	_ = json.Unmarshal(b, &m)
	if _, ok := m["notas_referenciadas"]; !ok {
		t.Error("sem referência por item, a da nota vai")
	}
}
