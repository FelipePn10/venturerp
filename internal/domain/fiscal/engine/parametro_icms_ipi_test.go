package engine

import (
	"strings"
	"testing"
)

// Cenários da regra do cadastro de parâmetros de ICMS/IPI (VFIS0320) aplicada
// pelo motor. Base de todos: R$ 1.000 de um NCM com IPI 5%, empresa no PR com
// ICMS interno de 12% e diferimento de 38,46% na configuração.

func calcularUm(t *testing.T, it TaxItem, destUF, destTipo string) TaxItemResult {
	t.Helper()
	params := TaxCalculationParams{Itens: []TaxItem{it}, EmitenteUF: "PR", DestinoUF: destUF, DestinoTipo: destTipo, Cfop: "5101"}
	res, err := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())
	if err != nil {
		t.Fatal(err)
	}
	return res.Itens[0]
}

func itemDeMil(p *ParametroICMSIPI) TaxItem {
	return TaxItem{Ncm: "84714900", ValorUnitario: 1000, Quantidade: 1, OrigemMercadoria: "0", Parametro: p}
}

func TestParametro_SemRegraMantemARegraGeral(t *testing.T) {
	r := calcularUm(t, itemDeMil(nil), "PR", "contribuinte")
	near(t, "aliq", 0.12, r.AliquotaICMS)
	near(t, "diferido", 46.15, r.ValorICMSDiferido)
	near(t, "pDif da configuração", 38.46, r.DiferimentoICMSPct)
	if r.CSTICMS != "51" {
		t.Fatalf("CST = %s, quer 51 (diferimento da configuração)", r.CSTICMS)
	}
}

// O mesmo produto sai com uma alíquota para quem revende/industrializa
// (contribuinte) e outra para consumidor final (não contribuinte).
func TestParametro_AliquotaPorTipoDeDestinatario(t *testing.T) {
	p := &ParametroICMSIPI{ID: 7,
		Contrib:    LadoParametroICMSIPI{AliqICMSPct: 18, CSTICMS: "00"},
		NaoContrib: LadoParametroICMSIPI{AliqICMSPct: 19.5},
	}
	c := calcularUm(t, itemDeMil(p), "PR", "contribuinte")
	near(t, "aliq contrib", 0.18, c.AliquotaICMS)
	near(t, "ICMS contrib", 180, c.ValorICMS)
	if c.CSTICMS != "00" || c.ValorICMSDiferido != 0 {
		t.Fatalf("contribuinte: CST %s diferido %.2f, quer 00 sem diferimento (a regra disse 00)", c.CSTICMS, c.ValorICMSDiferido)
	}
	n := calcularUm(t, itemDeMil(p), "PR", "nao_contribuinte")
	near(t, "aliq consumidor", 0.195, n.AliquotaICMS)
	near(t, "base consumidor (com IPI)", 1050, n.BaseICMS)
	near(t, "ICMS consumidor", 204.75, n.ValorICMS)
}

func TestParametro_ReducaoDeBaseViraCST20(t *testing.T) {
	p := &ParametroICMSIPI{Contrib: LadoParametroICMSIPI{RedICMSPct: 33.33, AlvoRedICMS: "BASE"}}
	r := calcularUm(t, itemDeMil(p), "PR", "contribuinte")
	near(t, "base reduzida", 666.70, r.BaseICMS)
	near(t, "ICMS", 80.00, r.ValorICMS)
	// O diferimento da configuração continua (CST 51 prevalece sobre 20).
	if r.CSTICMS != "51" {
		t.Fatalf("CST = %s, quer 51", r.CSTICMS)
	}
	semDif := calcularUmCfg(t, itemDeMil(p), FiscalConfig{UFEmpresa: "PR", IcmsInternoAliquota: 0.12})
	if semDif.CSTICMS != "20" || semDif.ReducaoBaseICMSPct != 33.33 {
		t.Fatalf("sem diferimento: CST = %s pRedBC %.2f, quer 20 e 33,33", semDif.CSTICMS, semDif.ReducaoBaseICMSPct)
	}
}

func calcularUmCfg(t *testing.T, it TaxItem, cfg FiscalConfig) TaxItemResult {
	t.Helper()
	params := TaxCalculationParams{Itens: []TaxItem{it}, EmitenteUF: "PR", DestinoUF: "PR", DestinoTipo: "contribuinte", Cfop: "5101"}
	res, err := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return res.Itens[0]
}

func TestParametro_ReducaoNoPercentual(t *testing.T) {
	// Redução na alíquota não é redução de base: sem pRedBC e sem forçar CST 20.
	p := &ParametroICMSIPI{Contrib: LadoParametroICMSIPI{AliqICMSPct: 12, RedICMSPct: 50, AlvoRedICMS: "PERCENTUAL"}}
	r := calcularUmCfg(t, itemDeMil(p), FiscalConfig{UFEmpresa: "PR", IcmsInternoAliquota: 0.12})
	near(t, "aliq reduzida", 0.06, r.AliquotaICMS)
	near(t, "base cheia", 1000, r.BaseICMS)
	near(t, "ICMS", 60, r.ValorICMS)
	if r.CSTICMS != "00" || r.ReducaoBaseICMSPct != 0 {
		t.Fatalf("CST %s pRedBC %.2f, quer 00 e 0", r.CSTICMS, r.ReducaoBaseICMSPct)
	}
}

func TestParametro_DiferimentoDaRegra(t *testing.T) {
	p := &ParametroICMSIPI{DiferimentoPct: 100}
	r := calcularUm(t, itemDeMil(p), "PR", "contribuinte")
	near(t, "diferido integral", 120, r.ValorICMSDiferido)
	near(t, "pDif da regra", 100, r.DiferimentoICMSPct)
	if r.CSTICMS != "51" {
		t.Fatalf("CST = %s, quer 51", r.CSTICMS)
	}
}

func TestParametro_IsencaoZeraICMSEDIFAL(t *testing.T) {
	p := &ParametroICMSIPI{Contrib: LadoParametroICMSIPI{CSTICMS: "40"}, NaoContrib: LadoParametroICMSIPI{CSTICMS: "41"}}
	r := calcularUm(t, itemDeMil(p), "PR", "contribuinte")
	if r.CSTICMS != "40" || r.ValorICMS != 0 || r.BaseICMS != 0 || r.ValorICMSDiferido != 0 {
		t.Fatalf("isenta: CST %s base %.2f ICMS %.2f diferido %.2f", r.CSTICMS, r.BaseICMS, r.ValorICMS, r.ValorICMSDiferido)
	}
	inter := calcularUm(t, itemDeMil(p), "BA", "nao_contribuinte")
	if inter.CSTICMS != "41" || inter.ValorICMS != 0 || inter.ValorDIFAL != 0 || inter.ValorFCP != 0 {
		t.Fatalf("não tributada interestadual: CST %s ICMS %.2f DIFAL %.2f FCP %.2f", inter.CSTICMS, inter.ValorICMS, inter.ValorDIFAL, inter.ValorFCP)
	}
	// Isenta com substituição tributária vira CST 30.
	st := itemDeMil(p)
	st.MvaPct = 0.4
	if r := calcularUm(t, st, "PR", "contribuinte"); r.CSTICMS != "30" {
		t.Fatalf("isenta com ST: CST %s, quer 30", r.CSTICMS)
	}
}

func TestParametro_Interestadual(t *testing.T) {
	// A regra da UF de destino traz a alíquota da operação; importada segue a
	// Resolução SF 13/2012 salvo alíquota própria na regra.
	p := &ParametroICMSIPI{Contrib: LadoParametroICMSIPI{AliqICMSPct: 12}}
	r := calcularUm(t, itemDeMil(p), "BA", "contribuinte")
	near(t, "aliq interestadual da regra", 0.12, r.AliquotaICMS)
	imp := itemDeMil(p)
	imp.OrigemMercadoria = "1"
	near(t, "importada", 0.04, calcularUm(t, imp, "BA", "contribuinte").AliquotaICMS)
}

func TestParametro_IPI(t *testing.T) {
	isento := &ParametroICMSIPI{CSTIPI: "52"}
	r := calcularUm(t, itemDeMil(isento), "PR", "contribuinte")
	if r.CSTIPI != "52" || r.ValorIPI != 0 || r.AliquotaIPI != 0 {
		t.Fatalf("IPI isento: CST %s aliq %.4f valor %.2f", r.CSTIPI, r.AliquotaIPI, r.ValorIPI)
	}
	reduzido := &ParametroICMSIPI{Contrib: LadoParametroICMSIPI{RedIPIPct: 50, AlvoRedIPI: "BASE"}}
	r = calcularUm(t, itemDeMil(reduzido), "PR", "contribuinte")
	near(t, "base IPI reduzida", 500, r.BaseIPI)
	near(t, "IPI", 25, r.ValorIPI)
}

func TestParametro_BeneficioFiscalPorLado(t *testing.T) {
	p := &ParametroICMSIPI{CodBeneficio: "PR800001", Contrib: LadoParametroICMSIPI{CodBeneficio: "PR810001"}}
	if got := p.BeneficioFiscal("contribuinte"); got != "PR810001" {
		t.Fatalf("contribuinte: %s", got)
	}
	if got := p.BeneficioFiscal("nao_contribuinte"); got != "PR800001" {
		t.Fatalf("não contribuinte: %s", got)
	}
	var nada *ParametroICMSIPI
	if nada.BeneficioFiscal("contribuinte") != "" {
		t.Fatal("sem regra não há cBenef")
	}
}

// A explicação da linha não pode falar em diferimento quando a regra impôs
// outro CST e o diferido foi zerado (aparecia "ICMS diferido em 38,46%" numa
// linha CST 00 do simulador).
func TestParametro_CSTImpostoTiraANotaDeDiferimento(t *testing.T) {
	p := &ParametroICMSIPI{Contrib: LadoParametroICMSIPI{AliqICMSPct: 12, CSTICMS: "00"}}
	r := calcularUm(t, itemDeMil(p), "PR", "contribuinte")
	if r.CSTICMS != "00" || r.ValorICMSDiferido != 0 {
		t.Fatalf("CST %s diferido %.2f", r.CSTICMS, r.ValorICMSDiferido)
	}
	for _, n := range r.Notas {
		if strings.HasPrefix(n, "ICMS diferido em ") {
			t.Fatalf("nota de diferimento numa linha sem diferimento: %v", r.Notas)
		}
	}
	achou := false
	for _, n := range r.Notas {
		achou = achou || strings.Contains(n, "CST 00 foi imposto")
	}
	if !achou {
		t.Fatalf("a linha deve explicar por que não há diferimento: %v", r.Notas)
	}
}
