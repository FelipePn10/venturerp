package engine

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func calcularZF(t *testing.T, destUF string, suframa bool, cfg FiscalConfig) TaxItemResult {
	t.Helper()
	if cfg.UFEmpresa == "" {
		cfg = defaultFiscalConfig()
	}
	item := TaxItem{Ncm: "84714900", ValorUnitario: decimal.NewFromInt(1000), Quantidade: decimal.NewFromInt(1), OrigemMercadoria: "0",
		Parametro: &ParametroICMSIPI{ID: 9, ZonaFranca: true, CSTIPI: "52", EnquadramentoIPI: "324"}}
	res, err := CalcularImpostos(TaxCalculationParams{Itens: []TaxItem{item}, EmitenteUF: cfg.UFEmpresa, DestinoUF: destUF,
		DestinoTipo: "contribuinte", Finalidade: FinalidadeRevenda, Cfop: "6109", SuframaDestino: suframa},
		defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Totais.ValorICMSDesonerado.Equal(res.Itens[0].ValorICMSDesonerado) {
		t.Fatalf("total desonerado %s ≠ item %s", res.Totais.ValorICMSDesonerado, res.Itens[0].ValorICMSDesonerado)
	}
	return res.Itens[0]
}

// Venda do PR para Manaus com SUFRAMA: ICMS de 7% desonerado (CST 40, motivo
// 7), PIS/COFINS a alíquota zero e o IPI pela regra (CST 52, cEnq 324).
func TestZonaFranca_ComSuframaDesonera(t *testing.T) {
	r := calcularZF(t, "AM", true, FiscalConfig{})
	if r.CSTICMS != "40" || !r.ValorICMS.IsZero() || !r.BaseICMS.IsZero() || r.MotivoDesoneracao != "7" {
		t.Fatalf("ICMS: CST %s valor %s base %s motivo %q", r.CSTICMS, r.ValorICMS, r.BaseICMS, r.MotivoDesoneracao)
	}
	if !r.ValorICMSDesonerado.Equal(decimal.NewFromInt(70)) {
		t.Fatalf("desonerado %s, quer 70 (7%% de 1.000)", r.ValorICMSDesonerado)
	}
	if r.CSTPIS != "06" || r.CSTCOFINS != "06" || !r.ValorPIS.IsZero() || !r.ValorCOFINS.IsZero() {
		t.Fatalf("PIS/COFINS: %s %s %s %s", r.CSTPIS, r.CSTCOFINS, r.ValorPIS, r.ValorCOFINS)
	}
	if r.CSTIPI != "52" || r.EnquadramentoIPI != "324" {
		t.Fatalf("IPI: CST %s cEnq %s", r.CSTIPI, r.EnquadramentoIPI)
	}
}

// Sem SUFRAMA válida, operação interna ou Simples: sem benefício, tributação
// normal e a linha explica por quê.
func TestZonaFranca_SemCondicaoTributaNormal(t *testing.T) {
	casos := []struct {
		nome, uf string
		suframa  bool
		cfg      FiscalConfig
		motivo   string
	}{
		{"sem SUFRAMA", "AM", false, FiscalConfig{}, "SUFRAMA"},
		{"operação interna", "PR", true, FiscalConfig{}, "interna"},
		{"Simples Nacional", "AM", true, FiscalConfig{UFEmpresa: "PR", IcmsInternoAliquota: decimal.NewFromFloat(0.12), Regime: RegimeSimples}, "Simples"},
	}
	for _, c := range casos {
		r := calcularZF(t, c.uf, c.suframa, c.cfg)
		if r.MotivoDesoneracao != "" || r.ValorICMSDesonerado.IsPositive() {
			t.Errorf("%s: não devia desonerar (%s)", c.nome, r.ValorICMSDesonerado)
		}
		if !strings.Contains(strings.Join(r.Notas, " | "), c.motivo) {
			t.Errorf("%s: a linha devia explicar (%q): %v", c.nome, c.motivo, r.Notas)
		}
	}
	if r := calcularZF(t, "AM", false, FiscalConfig{}); r.CSTICMS != "00" || !r.ValorICMS.Equal(decimal.NewFromInt(70)) {
		t.Fatalf("sem SUFRAMA devia tributar: CST %s ICMS %s", r.CSTICMS, r.ValorICMS)
	}
}

// O Convênio ICM 65/88 isenta a mercadoria para comercialização ou
// industrialização: não contribuinte e compra para uso e consumo ficam fora.
func TestZonaFranca_ConsumidorFinalNaoTemBeneficio(t *testing.T) {
	item := TaxItem{Ncm: "84714900", ValorUnitario: decimal.NewFromInt(1000), Quantidade: decimal.NewFromInt(1), OrigemMercadoria: "0",
		Parametro: &ParametroICMSIPI{ID: 9, ZonaFranca: true}}
	for _, c := range []struct{ tipo, finalidade string }{{"nao_contribuinte", ""}, {"contribuinte", FinalidadeUsoConsumo}} {
		res, err := CalcularImpostos(TaxCalculationParams{Itens: []TaxItem{item}, EmitenteUF: "PR", DestinoUF: "AM", DestinoTipo: c.tipo,
			Finalidade: c.finalidade, Cfop: "6108", SuframaDestino: true}, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())
		if err != nil {
			t.Fatal(err)
		}
		if r := res.Itens[0]; r.MotivoDesoneracao != "" || !strings.Contains(strings.Join(r.Notas, " "), "comercialização ou industrialização") {
			t.Errorf("%s/%s: não devia desonerar: %+v", c.tipo, c.finalidade, r.Notas)
		}
	}
}

func TestCFOPZonaFranca(t *testing.T) {
	for de, para := range map[string]string{"6101": "6109", "6107": "6109", "6102": "6110", "6108": "6110", "6109": "6109", "5101": "", "6949": ""} {
		if got := CFOPZonaFranca(de); got != para {
			t.Errorf("%s → %q, quer %q", de, got, para)
		}
	}
}

// Regra de Zona Franca com MVA (manual ou da regra): a venda isenta não tem ST —
// antes a base própria zerada fazia ST sobre o IPI.
func TestZonaFranca_SemST(t *testing.T) {
	cfg := defaultFiscalConfig()
	item := TaxItem{Ncm: "84714900", ValorUnitario: decimal.NewFromInt(1000), Quantidade: decimal.NewFromInt(1), OrigemMercadoria: "0",
		MvaPct:    decimal.RequireFromString("0.40"),
		Parametro: &ParametroICMSIPI{ID: 9, ZonaFranca: true, MvaPct: decimal.NewFromInt(40)}}
	res, err := CalcularImpostos(TaxCalculationParams{Itens: []TaxItem{item}, EmitenteUF: cfg.UFEmpresa, DestinoUF: "AM",
		DestinoTipo: "contribuinte", Finalidade: FinalidadeRevenda, Cfop: "6109", SuframaDestino: true},
		defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	r := res.Itens[0]
	if !r.ValorICMSDesonerado.IsPositive() || !r.ValorICMSST.IsZero() || !r.BaseICMSST.IsZero() {
		t.Fatalf("desonerado %s, ST %s base ST %s", r.ValorICMSDesonerado, r.ValorICMSST, r.BaseICMSST)
	}
}
