package focusnfe

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func f(v float64) *float64 { return &v }

// tags lê o XML (falha se estiver mal formado) e devolve o caminho de cada
// folha com o texto, para o teste conferir valor por valor.
func tags(t *testing.T, doc []byte) map[string][]string {
	t.Helper()
	d := xml.NewDecoder(strings.NewReader(string(doc)))
	var pilha []string
	out := map[string][]string{}
	var texto string
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("XML mal formado: %v", err)
		}
		switch e := tok.(type) {
		case xml.StartElement:
			pilha = append(pilha, e.Name.Local)
			texto = ""
		case xml.CharData:
			texto = string(e)
		case xml.EndElement:
			caminho := strings.Join(pilha, "/")
			if texto != "" {
				out[caminho] = append(out[caminho], texto)
			}
			texto = ""
			pilha = pilha[:len(pilha)-1]
		}
	}
	return out
}

func confere(t *testing.T, m map[string][]string, caminho string, quer ...string) {
	t.Helper()
	got := m[caminho]
	if strings.Join(got, "|") != strings.Join(quer, "|") {
		t.Errorf("%s = %v, quer %v", caminho, got, quer)
	}
}

// A NF-e 75 da Tecnofer (homologação, 25/11/2026): o XML da prévia tem de
// trazer os mesmos valores que a Focus gerou e a SEFAZ autorizou.
func TestXMLDaPrevia_ValoresDaNotaAutorizada(t *testing.T) {
	ie := "ISENTO"
	p := NFEPayload{
		NaturezaOperacao: "VENDA DE PRODUCAO DO ESTABELECIMENTO", DataEmissao: "2026-11-25T14:24:17-03:00",
		TipoDocumento: 1, LocalDestino: 1, FinalidadeEmissao: 1, ConsumidorFinal: 1, PresencaComprador: 9,
		Emitente: NFEEmitente{CNPJ: "52.454.668/0001-02", Nome: "TECNOFER", Logradouro: "RUA JACOB EVALDO STADLER", Numero: "83",
			Bairro: "PARQUE INDUSTRIAL III", Municipio: "Mandaguari", UF: "PR", CEP: "86975-000", RegimeTributario: 3},
		Destinatario: NFEDestinatario{CNPJCPF: "078.042.219-83", Nome: "KARIN & CIA <teste>", CodigoMunicipio: "4114203", UF: "PR", IndicadorIE: 9, IE: &ie},
		Items: []NFEItem{{
			NumeroItem: 1, CodigoProduto: "RN-01001", Descricao: "BALANÇA ASA DELTA", CodigoNCM: "8716.90.90", CFOP: "5101",
			UnidadeComercial: "UN", QuantidadeComercial: 1, ValorUnitarioComercial: 360.5, ValorBruto: 360.5,
			CodigoSituacaoTributariaICMS: "00", ModalidadeBaseCalculoICMS: 3, ValorBaseCalculoICMS: 372.22, AliquotaICMS: 19.5, ValorICMS: 72.58,
			CodigoSituacaoTributariaIPI: "50", AliquotaIPI: 3.25, ValorIPI: 11.72,
			CodigoSituacaoTributariaPIS: "01", AliquotaPIS: 1.65, ValorPIS: 5.95,
			CodigoSituacaoTributariaCOFINS: "01", AliquotaCOFINS: 7.6, ValorCOFINS: 27.40,
			IBSCBS: &NFEItemIBSCBS{CST: "000", ClassTrib: "000001", Base: 254.57, AliqIBSUF: 0.1, ValorIBSUF: 0.25, ValorIBS: 0.25, AliqCBS: 0.9, ValorCBS: 2.29},
		}},
		FormaPagamento: []NFEFormaPagamento{{FormaPagamento: "01", Valor: 372.22}},
		ValorProdutos:  360.5, ValorTotal: 372.22,
	}
	doc := XMLDaPrevia(p, DadosDaPrevia{Serie: "1", Numero: 75, IE: "9103144679", CodigoMunicipio: "4114203", NomeFantasia: "Tecnofer"})
	m := tags(t, doc)
	r := "NFe/infNFe/"
	confere(t, m, r+"ide/cUF", "41")
	confere(t, m, r+"ide/nNF", "75")
	confere(t, m, r+"ide/tpAmb", "2")
	confere(t, m, r+"emit/CNPJ", "52454668000102")
	confere(t, m, r+"emit/IE", "9103144679")
	confere(t, m, r+"dest/CPF", "07804221983")
	confere(t, m, r+"dest/xNome", "KARIN & CIA <teste>")
	confere(t, m, r+"det/prod/NCM", "87169090")
	confere(t, m, r+"det/prod/vUnCom", "360.50")
	confere(t, m, r+"det/prod/cEAN", "SEM GTIN")
	confere(t, m, r+"det/imposto/ICMS/ICMS00/vBC", "372.22")
	confere(t, m, r+"det/imposto/ICMS/ICMS00/pICMS", "19.5000")
	confere(t, m, r+"det/imposto/ICMS/ICMS00/vICMS", "72.58")
	confere(t, m, r+"det/imposto/IPI/IPITrib/vBC", "360.50")
	confere(t, m, r+"det/imposto/IPI/IPITrib/vIPI", "11.72")
	confere(t, m, r+"det/imposto/PIS/PISAliq/vBC", "360.50") // como no XML autorizado: a base que reproduz o valor gravado
	confere(t, m, r+"det/imposto/COFINS/COFINSAliq/vCOFINS", "27.40")
	confere(t, m, r+"det/imposto/IBSCBS/gIBSCBS/gCBS/vCBS", "2.29")
	confere(t, m, r+"total/ICMSTot/vBC", "372.22")
	confere(t, m, r+"total/ICMSTot/vICMS", "72.58")
	confere(t, m, r+"total/ICMSTot/vIPI", "11.72")
	confere(t, m, r+"total/ICMSTot/vPIS", "5.95")
	confere(t, m, r+"total/ICMSTot/vNF", "372.22")
	confere(t, m, r+"total/IBSCBSTot/gCBS/vCBS", "2.29")
	confere(t, m, r+"transp/modFrete", "9")
	confere(t, m, r+"pag/detPag/vPag", "372.22")
	if !strings.Contains(strings.Join(m[r+"infAdic/infCpl"], ""), "PREVIA SEM VALIDADE FISCAL") {
		t.Error("a prévia não traz o aviso nas informações complementares")
	}
	if strings.Contains(string(doc), "Signature") || strings.Contains(string(doc), "protNFe") || strings.Contains(string(doc), `Id="NFe`) {
		t.Error("a prévia não pode parecer assinada ou autorizada")
	}
}

// Cada CST/CSOSN cai no grupo do layout, com ST, diferimento e crédito do Simples.
func TestXMLDaPrevia_GruposDoICMS(t *testing.T) {
	casos := []struct {
		item  NFEItem
		grupo string
		quer  map[string]string
	}{
		{NFEItem{CodigoSituacaoTributariaICMS: "10", ValorBaseCalculoICMS: 100, AliquotaICMS: 12, ValorICMS: 12,
			BaseCalculoICMSST: f(140), AliquotaICMSST: f(18), ValorICMSST: f(13.2), PercentualMVAICMSST: f(40)},
			"ICMS10", map[string]string{"vICMS": "12.00", "vBCST": "140.00", "vICMSST": "13.20", "pMVAST": "40.0000", "modBCST": "4"}},
		{NFEItem{CodigoSituacaoTributariaICMS: "41"}, "ICMS40", map[string]string{"CST": "41"}},
		{NFEItem{CodigoSituacaoTributariaICMS: "51", ValorBaseCalculoICMS: 100, AliquotaICMS: 19.5, ValorICMS: 19.5,
			PercentualDiferimento: f(33.33), ValorICMSDiferido: f(6.5)},
			"ICMS51", map[string]string{"vICMSOp": "19.50", "vICMSDif": "6.50", "vICMS": "13.00", "pDif": "33.3300"}},
		{NFEItem{CodigoSituacaoTributariaICMS: "101", AliquotaCreditoSimples: f(2.5), ValorCreditoSimples: f(2.5)},
			"ICMSSN101", map[string]string{"CSOSN": "101", "vCredICMSSN": "2.50"}},
		{NFEItem{CodigoSituacaoTributariaICMS: "400"}, "ICMSSN102", map[string]string{"CSOSN": "400"}},
		{NFEItem{CodigoSituacaoTributariaICMS: "40", MotivoDesoneracao: "7", ValorICMSDesonerado: 18},
			"ICMS40", map[string]string{"vICMSDeson": "18.00", "motDesICMS": "7", "indDeduzDeson": "1"}},
	}
	for _, c := range casos {
		c.item.NumeroItem, c.item.CodigoSituacaoTributariaPIS, c.item.CodigoSituacaoTributariaCOFINS = 1, "07", "07"
		m := tags(t, XMLDaPrevia(NFEPayload{Items: []NFEItem{c.item}}, DadosDaPrevia{}))
		for tag, v := range c.quer {
			confere(t, m, "NFe/infNFe/det/imposto/ICMS/"+c.grupo+"/"+tag, v)
		}
		confere(t, m, "NFe/infNFe/det/imposto/PIS/PISNT/CST", "07")
	}
}

// ST e desoneração entram no total; o vNF sem valor informado soma os dois.
func TestXMLDaPrevia_TotaisComSTEDesoneracao(t *testing.T) {
	p := NFEPayload{Items: []NFEItem{
		{NumeroItem: 1, ValorBruto: 100, CodigoSituacaoTributariaICMS: "10", ValorBaseCalculoICMS: 100, AliquotaICMS: 12, ValorICMS: 12,
			BaseCalculoICMSST: f(140), AliquotaICMSST: f(18), ValorICMSST: f(13.2)},
		{NumeroItem: 2, ValorBruto: 50, CodigoSituacaoTributariaICMS: "40", MotivoDesoneracao: "7", ValorICMSDesonerado: 6},
	}}
	m := tags(t, XMLDaPrevia(p, DadosDaPrevia{}))
	confere(t, m, "NFe/infNFe/total/ICMSTot/vST", "13.20")
	confere(t, m, "NFe/infNFe/total/ICMSTot/vICMSDeson", "6.00")
	confere(t, m, "NFe/infNFe/total/ICMSTot/vNF", "157.20")
}

// Totais em decimal: muitas linhas de centavos somam exato.
func TestXMLDaPrevia_TotaisSemErroDeCentavo(t *testing.T) {
	var itens []NFEItem
	for i := 1; i <= 1000; i++ {
		itens = append(itens, NFEItem{NumeroItem: i, ValorBruto: 0.1, CodigoSituacaoTributariaICMS: "00", ValorBaseCalculoICMS: 0.1, ValorICMS: 0.01, ValorIPI: 0.03, CodigoSituacaoTributariaIPI: "50"})
	}
	m := tags(t, XMLDaPrevia(NFEPayload{Items: itens}, DadosDaPrevia{}))
	confere(t, m, "NFe/infNFe/total/ICMSTot/vProd", "100.00")
	confere(t, m, "NFe/infNFe/total/ICMSTot/vICMS", "10.00")
	confere(t, m, "NFe/infNFe/total/ICMSTot/vNF", "130.00")
}
