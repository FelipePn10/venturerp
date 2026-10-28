package fiscal_uc

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/engine"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

func sp(s string) *string { return &s }

func niveis(ps []response.PreviaPendenciaNFe) map[string]string {
	m := map[string]string{}
	for _, p := range ps {
		m[p.Campo] = p.Nivel
	}
	return m
}

func TestConferirTributacao(t *testing.T) {
	regra := int64(1)
	normal := &entity.FiscalConfig{UFEmpresa: "PR", RegimeTributario: "3"}
	exigente := &entity.FiscalConfig{UFEmpresa: "PR", RegimeTributario: "3", ExigirRegraICMSIPI: true}
	simples := &entity.FiscalConfig{UFEmpresa: "PR", RegimeTributario: "1"}
	nota := func(uf string, ie *string) *entity.FiscalExit {
		return &entity.FiscalExit{Finalidade: 1, UFDestinatario: &uf, IEDestinatario: ie}
	}
	linha := func(cst string, comRegra bool) *entity.FiscalExitItem {
		it := &entity.FiscalExitItem{Sequence: 1, CstICMS: &cst}
		if comRegra {
			it.RegraICMSIPIID = &regra
		}
		return it
	}

	if n := niveis(conferirTributacao(nota("PR", sp("1")), []*entity.FiscalExitItem{linha("00", false)}, normal)); n["itens.regra_icms_ipi"] != NivelAtencao {
		t.Fatalf("sem exigência: %v", n)
	}
	if n := niveis(conferirTributacao(nota("PR", sp("1")), []*entity.FiscalExitItem{linha("00", false)}, exigente)); n["itens.regra_icms_ipi"] != NivelImpede {
		t.Fatalf("com exigência: %v", n)
	}
	if n := niveis(conferirTributacao(nota("PR", sp("1")), []*entity.FiscalExitItem{linha("00", true)}, exigente)); n["itens.regra_icms_ipi"] != "" {
		t.Fatalf("com regra: %v", n)
	}
	// Devolução e beneficiamento têm tributação própria.
	dev := nota("PR", sp("1"))
	dev.Finalidade = 4
	benef := nota("PR", sp("1"))
	benef.SourceType = sp(OrigemBeneficiamento)
	for nome, e := range map[string]*entity.FiscalExit{"devolução": dev, "beneficiamento": benef} {
		if n := niveis(conferirTributacao(e, []*entity.FiscalExitItem{linha("50", false)}, exigente)); n["itens.regra_icms_ipi"] != "" {
			t.Fatalf("%s não exige regra: %v", nome, n)
		}
	}
	// Regime trocado depois de gerar a nota.
	if n := niveis(conferirTributacao(nota("PR", sp("1")), []*entity.FiscalExitItem{linha("00", true)}, simples)); n["itens[1].cst_icms"] != NivelImpede {
		t.Fatalf("CST no Simples: %v", n)
	}
	if n := niveis(conferirTributacao(nota("PR", sp("1")), []*entity.FiscalExitItem{linha("102", true)}, normal)); n["itens[1].cst_icms"] != NivelImpede {
		t.Fatalf("CSOSN no regime normal: %v", n)
	}
	comST := linha("102", true)
	comST.ValorICMSST = 10
	if n := niveis(conferirTributacao(nota("PR", sp("1")), []*entity.FiscalExitItem{comST}, simples)); n["itens[1].cst_icms"] != NivelImpede {
		t.Fatalf("CSOSN 102 com ST: %v", n)
	}
	if n := niveis(conferirTributacao(nota("PR", sp("1")), []*entity.FiscalExitItem{linha("101", true)}, simples)); n["itens[1].credito_simples"] != NivelImpede {
		t.Fatalf("CSOSN 101 sem pCredSN: %v", n)
	}
	// DIFAL: não contribuinte (indIEDest 9) de outra UF sem o grupo impede;
	// contribuinte isento (2) e linha isenta não.
	if n := niveis(conferirTributacao(nota("GO", nil), []*entity.FiscalExitItem{linha("00", true)}, normal)); n["itens[1].difal"] != NivelImpede {
		t.Fatalf("sem grupo DIFAL: %v", n)
	}
	if n := niveis(conferirTributacao(nota("GO", sp("ISENTO")), []*entity.FiscalExitItem{linha("00", true)}, normal)); n["itens[1].difal"] != "" {
		t.Fatalf("contribuinte isento: %v", n)
	}
	if n := niveis(conferirTributacao(nota("GO", nil), []*entity.FiscalExitItem{linha("40", true)}, normal)); n["itens[1].difal"] != "" {
		t.Fatalf("linha isenta: %v", n)
	}
	comGrupo := linha("00", true)
	comGrupo.ICMSUFDestBase, comGrupo.ICMSUFDestAliqInterna, comGrupo.FCPUFDestPct = 1050, 17, 2
	if n := niveis(conferirTributacao(nota("GO", nil), []*entity.FiscalExitItem{comGrupo}, normal)); n["itens[1].difal"] != "" || n["itens[1].difal_aliquotas"] != NivelAtencao {
		t.Fatalf("com grupo DIFAL: %v", n)
	}
	if err := recusarPendenciasTributarias(nota("PR", sp("1")), []*entity.FiscalExitItem{linha("00", false)}, exigente); err == nil {
		t.Fatal("autorização tem de recusar a linha sem regra")
	}
}

func TestIndicadoresDestinatario(t *testing.T) {
	uso := "USO_CONSUMO"
	for nome, c := range map[string]struct {
		e          *entity.FiscalExit
		final, ind int
	}{
		"contribuinte":         {&entity.FiscalExit{IEDestinatario: sp("901")}, 0, 1},
		"contribuinte isento":  {&entity.FiscalExit{IEDestinatario: sp("isento")}, 1, 2},
		"não contribuinte":     {&entity.FiscalExit{}, 1, 9},
		"contribuinte uso/con": {&entity.FiscalExit{IEDestinatario: sp("901"), FinalidadeCompra: &uso}, 1, 1},
	} {
		if f, i := indicadoresDestinatario(c.e); f != c.final || i != c.ind {
			t.Errorf("%s: indFinal %d indIEDest %d, quer %d %d", nome, f, i, c.final, c.ind)
		}
	}
}

func TestPercentualXMLSemResiduoDeFloat(t *testing.T) {
	for in, quer := range map[float64]float64{0.07: 7, 0.195: 19.5, 0.0165: 1.65, 0.076: 7.6, 0.502439: 50.2439, 0: 0} {
		if got := percentualXML(in); got != quer {
			t.Errorf("percentualXML(%v) = %v, quer %v", in, got, quer)
		}
	}
	if pctTexto(19.5) != "19,50" {
		t.Fatalf("pctTexto: %s", pctTexto(19.5))
	}
}

func TestConferirCFOPDoDestino(t *testing.T) {
	cfg := &entity.FiscalConfig{UFEmpresa: "PR", RegimeTributario: "3"}
	regra := int64(1)
	linha := func(cfop string, st float64) *entity.FiscalExitItem {
		cst := "00"
		return &entity.FiscalExitItem{Sequence: 1, Cfop: cfop, CstICMS: &cst, RegraICMSIPIID: &regra, ValorICMSST: st, ICMSUFDestBase: 1}
	}
	nota := func(uf string) *entity.FiscalExit {
		ie := "1"
		return &entity.FiscalExit{Finalidade: 1, UFDestinatario: &uf, IEDestinatario: &ie}
	}
	casos := []struct {
		uf, cfop string
		st       float64
		cfopNiv  string
		stNiv    string
	}{
		{"PR", "5102", 0, "", ""}, {"SP", "6102", 0, "", ""}, {"EX", "7101", 0, "", ""},
		{"SP", "5102", 0, NivelImpede, ""}, {"PR", "6102", 0, NivelImpede, ""},
		{"SP", "6102", 10, "", NivelAtencao}, {"SP", "6401", 10, "", ""},
	}
	for _, c := range casos {
		n := niveis(conferirTributacao(nota(c.uf), []*entity.FiscalExitItem{linha(c.cfop, c.st)}, cfg))
		if n["itens[1].cfop"] != c.cfopNiv || n["itens[1].cfop_st"] != c.stNiv {
			t.Errorf("%s %s ST %.0f: %v", c.uf, c.cfop, c.st, n)
		}
	}
}

func TestTipoDestinatarioProdutorRural(t *testing.T) {
	f, j := "F", "J"
	for nome, c := range map[string]struct {
		ie, tp *string
		quer   string
	}{
		"PJ com IE": {sp("901"), &j, "contribuinte"}, "produtor rural (PF com IE)": {sp("901"), &f, "contribuinte"},
		"PF sem IE": {nil, &f, "pessoa_fisica"}, "PJ sem IE": {nil, &j, "nao_contribuinte"}, "isento": {sp("ISENTO"), &j, "nao_contribuinte"},
	} {
		if got := tipoDestinatario(c.ie, c.tp); got != c.quer {
			t.Errorf("%s: %s, quer %s", nome, got, c.quer)
		}
	}
}

func TestInformacoesComplementares(t *testing.T) {
	if informacoesComplementares(nil, false, decimal.Zero, 0, nil) != nil {
		t.Fatal("sem texto, sem Simples e sem base legal: nada")
	}
	got := *informacoesComplementares(sp(" Pedido 123 "), true, decimal.RequireFromString("25.6"), 2.56, []string{"Isento conforme art. 10"})
	for _, quer := range []string{"Pedido 123", "OPTANTE PELO SIMPLES NACIONAL", "NAO GERA DIREITO A CREDITO FISCAL DE IPI",
		"CREDITO DE ICMS NO VALOR DE R$ 25,60, CORRESPONDENTE A ALIQUOTA DE 2,56%", "Isento conforme art. 10"} {
		if !strings.Contains(got, quer) {
			t.Errorf("infCpl sem %q: %s", quer, got)
		}
	}
	if s := *informacoesComplementares(nil, true, decimal.Zero, 0, nil); strings.Contains(s, "CREDITO DE ICMS") {
		t.Fatalf("sem crédito não cita crédito: %s", s)
	}
}

func TestDispositivosAplicados(t *testing.T) {
	p := &engine.ParametroICMSIPI{Dispositivos: engine.DispositivosLegais{ICMSContrib: "Isenção art. 1", ICMSNaoContrib: "Isenção NC art. 2",
		ReducaoContrib: "Redução art. 3", Diferimento: "Diferimento art. 28"}}
	if got := p.DispositivosAplicados("contribuinte", engine.TaxItemResult{CSTICMS: "40"}); len(got) != 1 || got[0] != "Isenção art. 1" {
		t.Fatalf("isenção contribuinte: %v", got)
	}
	if got := p.DispositivosAplicados("nao_contribuinte", engine.TaxItemResult{CSTICMS: "40"}); len(got) != 1 || got[0] != "Isenção NC art. 2" {
		t.Fatalf("isenção não contribuinte: %v", got)
	}
	if got := p.DispositivosAplicados("contribuinte", engine.TaxItemResult{CSTICMS: "00"}); len(got) != 0 {
		t.Fatalf("tributada integral não cita benefício: %v", got)
	}
	if got := p.DispositivosAplicados("contribuinte", engine.TaxItemResult{CSTICMS: "51", ValorICMSDiferido: 75}); len(got) != 1 || got[0] != "Diferimento art. 28" {
		t.Fatalf("diferimento: %v", got)
	}
	if got := p.DispositivosAplicados("contribuinte", engine.TaxItemResult{CSTICMS: "20", ReducaoBaseICMSPct: 33}); len(got) != 2 {
		t.Fatalf("redução (CST 20 + redução): %v", got)
	}
}
