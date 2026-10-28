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

// CST 51: a SEFAZ confere vICMS = vICMSOp − vICMSDif (rejeição 353); o cBenef
// vai quando informado (rejeição 930 sem ele no PR).
func TestNFEItemCST51ComDiferimentoEBeneficio(t *testing.T) {
	dif, pct := 46.15, 38.46
	m := NFEItem{CodigoSituacaoTributariaICMS: "51", ValorBaseCalculoICMS: 1000, AliquotaICMS: 12, ValorICMS: 120,
		PercentualDiferimento: &pct, ValorICMSDiferido: &dif, CodigoBeneficioFiscal: " PR830001 "}.campos()
	if m["icms_valor_operacao"] != 120.0 {
		t.Fatalf("icms_valor_operacao = %v, quer 120 (o ICMS da operação)", m["icms_valor_operacao"])
	}
	if m["icms_valor"] != 73.85 {
		t.Fatalf("icms_valor = %v, quer 73.85 (operação − diferido)", m["icms_valor"])
	}
	if m["codigo_beneficio_fiscal"] != "PR830001" {
		t.Fatalf("codigo_beneficio_fiscal = %v", m["codigo_beneficio_fiscal"])
	}
	semDif := NFEItem{CodigoSituacaoTributariaICMS: "00", ValorICMS: 120}.campos()
	if _, ok := semDif["icms_valor_operacao"]; ok || semDif["icms_valor"] != 120.0 {
		t.Fatalf("CST 00 não leva valor da operação: %v", semDif)
	}
	if _, ok := semDif["codigo_beneficio_fiscal"]; ok {
		t.Fatal("cBenef vazio não pode ir no payload")
	}
}

// CST sem tributação leva só CST e cClassTrib; CST com redução leva o
// percentual de redução e a alíquota efetiva de cada tributo (nomes da
// referência de campos da Focus: pRedAliq e pAliqEfet).
func TestGrupoIBSCBSSemValoresEComReducao(t *testing.T) {
	sem := itemParaMapa(NFEItem{NumeroItem: 1, IBSCBS: &NFEItemIBSCBS{CST: "410", ClassTrib: "410004", SemValores: true}})
	if sem["ibs_cbs_situacao_tributaria"] != "410" || sem["ibs_cbs_classificacao_tributaria"] != "410004" {
		t.Fatalf("códigos: %v", sem)
	}
	for _, k := range []string{"ibs_cbs_base_calculo", "ibs_uf_aliquota", "cbs_valor", "cbs_aliquota"} {
		if _, ok := sem[k]; ok {
			t.Fatalf("CST 410 não leva %q", k)
		}
	}
	red := itemParaMapa(NFEItem{NumeroItem: 1, IBSCBS: &NFEItemIBSCBS{CST: "200", ClassTrib: "200030", Base: 1000,
		AliqCBS: 8.7, ValorCBS: 34.8, ComReducao: true, RedCBS: 60, EfetCBS: 3.48, RedIBSUF: 60, EfetIBSUF: 0.02, RedIBSMun: 60, EfetIBSMun: 0.02}})
	for k, want := range map[string]any{"cbs_percentual_reducao_aliquota": 60.0, "cbs_aliquota_efetiva": 3.48, "cbs_aliquota": 8.7,
		"ibs_uf_percentual_reducao_aliquota": 60.0, "ibs_uf_aliquota_efetiva": 0.02, "ibs_mun_aliquota_efetiva": 0.02, "ibs_cbs_base_calculo": 1000.0} {
		if red[k] != want {
			t.Errorf("%s = %v, want %v", k, red[k], want)
		}
	}
}

func itemParaMapa(it NFEItem) map[string]any {
	b, _ := json.Marshal(NFEPayload{Items: []NFEItem{it}})
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m["items"].([]any)[0].(map[string]any)
}

// pRedBC só nos CSTs que o aceitam; crédito do Simples só em 101/201/900;
// grupo ICMSUFDest com partilha 100% e FCP quando há.
func TestNFEItemTributacaoPorCenario(t *testing.T) {
	red, aliq, valor := 33.33, 2.56, 25.6
	m := itemParaMapa(NFEItem{CodigoSituacaoTributariaICMS: "20", PercentualReducaoBC: &red})
	if m["icms_reducao_base_calculo"] != 33.33 {
		t.Fatalf("CST 20 sem pRedBC: %v", m)
	}
	if m := itemParaMapa(NFEItem{CodigoSituacaoTributariaICMS: "00", PercentualReducaoBC: &red}); m["icms_reducao_base_calculo"] != nil {
		t.Fatalf("CST 00 não leva pRedBC: %v", m)
	}
	m = itemParaMapa(NFEItem{CodigoSituacaoTributariaICMS: "101", AliquotaCreditoSimples: &aliq, ValorCreditoSimples: &valor})
	if m["icms_aliquota_credito_simples"] != 2.56 || m["icms_valor_credito_simples"] != 25.6 || m["icms_base_calculo"] != nil {
		t.Fatalf("CSOSN 101: %v", m)
	}
	if m := itemParaMapa(NFEItem{CodigoSituacaoTributariaICMS: "102", AliquotaCreditoSimples: &aliq, ValorCreditoSimples: &valor}); m["icms_aliquota_credito_simples"] != nil {
		t.Fatalf("CSOSN 102 não leva crédito: %v", m)
	}
	m = itemParaMapa(NFEItem{CodigoSituacaoTributariaICMS: "00", UFDest: &NFEItemUFDest{Base: 1050, AliquotaInterna: 17, AliquotaInterestadual: 7, ValorDestino: 105, PercentualFCP: 2, ValorFCP: 21}})
	if m["icms_base_calculo_uf_destino"] != 1050.0 || m["icms_percentual_partilha"] != 100.0 || m["icms_valor_uf_remetente"] != 0.0 || m["fcp_valor_uf_destino"] != 21.0 || m["fcp_base_calculo_uf_destino"] != 1050.0 {
		t.Fatalf("ICMSUFDest: %v", m)
	}
	if m := itemParaMapa(NFEItem{CodigoSituacaoTributariaICMS: "00", UFDest: &NFEItemUFDest{Base: 1000, AliquotaInterna: 18, AliquotaInterestadual: 12, ValorDestino: 60}}); m["fcp_valor_uf_destino"] != nil {
		t.Fatalf("sem FCP não leva o FCP: %v", m)
	}
}

// pRedBCST só vai quando a base da ST foi reduzida.
func TestNFEItemReducaoBaseST(t *testing.T) {
	base, aliq, valor, red := 1176.0, 12.0, 21.12, 20.0
	com := itemParaMapa(NFEItem{NumeroItem: 1, CodigoSituacaoTributariaICMS: "10", BaseCalculoICMSST: &base,
		AliquotaICMSST: &aliq, ValorICMSST: &valor, PercentualReducaoBCST: &red})
	if com["icms_reducao_base_calculo_st"] != 20.0 {
		t.Fatalf("icms_reducao_base_calculo_st = %v, quer 20", com["icms_reducao_base_calculo_st"])
	}
	sem := itemParaMapa(NFEItem{NumeroItem: 1, CodigoSituacaoTributariaICMS: "10", BaseCalculoICMSST: &base,
		AliquotaICMSST: &aliq, ValorICMSST: &valor})
	if _, ok := sem["icms_reducao_base_calculo_st"]; ok {
		t.Fatal("sem redução não leva pRedBCST")
	}
}
