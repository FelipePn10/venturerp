package engine

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

// Tributação por cenário: o mesmo produto (R$ 1.000, NCM com IPI de 5%) sai
// com tributação diferente conforme o regime do emitente, a finalidade da
// compra, a UF de destino, a regra do cadastro e o tipo de nota. Os valores
// esperados foram calculados à mão; a configuração padrão é a da Tecnofer
// (PR, ICMS interno 12% e diferimento de 38,46% nos testes).

type cenario struct {
	destUF, destTipo, finalidade string
	cfg                          FiscalConfig
	flags                        InvoiceTypeFlags
	item                         TaxItem
}

func calcularCenario(t *testing.T, c cenario) TaxItemResult {
	t.Helper()
	if c.cfg.UFEmpresa == "" {
		c.cfg = defaultFiscalConfig()
	}
	if c.item.Ncm == "" {
		c.item = TaxItem{Ncm: "84714900", ValorUnitario: 1000, Quantidade: 1, OrigemMercadoria: "0", Parametro: c.item.Parametro, MvaPct: c.item.MvaPct}
	}
	params := TaxCalculationParams{
		Itens: []TaxItem{c.item}, EmitenteUF: c.cfg.UFEmpresa, DestinoUF: c.destUF,
		DestinoTipo: c.destTipo, Finalidade: c.finalidade, Flags: c.flags, Cfop: "5101",
	}
	res, err := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, c.cfg)
	if err != nil {
		t.Fatal(err)
	}
	return res.Itens[0]
}

func querCST(t *testing.T, label, want, got string) {
	t.Helper()
	if want != got {
		t.Fatalf("%s = %q, quer %q", label, got, want)
	}
}

func comRegime(r Regime, credito float64) FiscalConfig {
	c := defaultFiscalConfig()
	c.Regime, c.CreditoSimplesPct = r, credito
	return c
}

// ── Finalidade da compra ────────────────────────────────────────────────────

// Tecnofer vendendo no PR para quem revende: IPI fora da base, diferimento
// parcial (CST 51). É o cálculo de sempre, que não pode mudar.
func TestCenario_PR_ContribuinteRevenda_Diferimento(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "PR", destTipo: "contribuinte", finalidade: FinalidadeRevenda})
	near(t, "base ICMS", 1000, r.BaseICMS)
	near(t, "ICMS", 120, r.ValorICMS)
	near(t, "diferido", 46.15, r.ValorICMSDiferido)
	querCST(t, "CST", "51", r.CSTICMS)
	near(t, "IPI", 50, r.ValorIPI)
	near(t, "PIS", 16.50, r.ValorPIS)
	near(t, "COFINS", 76, r.ValorCOFINS)
}

// Contribuinte comprando para uso/consumo é consumidor final: IPI na base e
// sem diferimento (SEFA/PR, Consulta 11/2011).
func TestCenario_PR_ContribuinteUsoConsumo_SemDiferimentoComIPINaBase(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "PR", destTipo: "contribuinte", finalidade: FinalidadeUsoConsumo})
	near(t, "base ICMS (com IPI)", 1050, r.BaseICMS)
	near(t, "ICMS", 126, r.ValorICMS)
	near(t, "diferido", 0, r.ValorICMSDiferido)
	querCST(t, "CST", "00", r.CSTICMS)
}

// Interestadual para contribuinte de uso/consumo: IPI na base, mas o DIFAL é
// do comprador — não vai na nota do vendedor.
func TestCenario_Interestadual_ContribuinteUsoConsumo_SemDIFALNaNota(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "SP", destTipo: "contribuinte", finalidade: FinalidadeUsoConsumo})
	near(t, "base ICMS", 1050, r.BaseICMS)
	near(t, "ICMS 12%", 126, r.ValorICMS)
	near(t, "DIFAL", 0, r.ValorDIFAL)
	near(t, "base UF destino", 0, r.BaseUFDest)
}

// Interestadual para não contribuinte: grupo ICMSUFDest completo.
func TestCenario_Interestadual_NaoContribuinte_GrupoDIFAL(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "GO", destTipo: "nao_contribuinte"})
	near(t, "base ICMS", 1050, r.BaseICMS)
	near(t, "ICMS 7%", 73.50, r.ValorICMS)
	near(t, "base UF destino", 1050, r.BaseUFDest)
	near(t, "alíquota interna destino", 17, r.AliqInternaUFDestPct)
	near(t, "alíquota interestadual", 7, r.AliqInterestadualPct)
	near(t, "DIFAL", 105, r.ValorDIFAL)
	near(t, "FCP %", 2, r.FCPUFDestPct)
	near(t, "FCP", 21, r.ValorFCP)
}

// Importado (origem 1) para não contribuinte: 4% e DIFAL sobre 4%.
func TestCenario_Importado_NaoContribuinte_DIFALSobre4(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "SP", destTipo: "nao_contribuinte",
		item: TaxItem{Ncm: "84714900", ValorUnitario: 1000, Quantidade: 1, OrigemMercadoria: "1"}})
	near(t, "ICMS 4%", 42, r.ValorICMS)
	near(t, "interestadual", 4, r.AliqInterestadualPct)
	near(t, "DIFAL 18-4", 147, r.ValorDIFAL)
}

// ── Regime do emitente ──────────────────────────────────────────────────────

func TestCenario_LucroPresumido_PISCOFINSCumulativo(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "PR", destTipo: "contribuinte", cfg: comRegime(RegimePresumido, 0)})
	near(t, "PIS 0,65%", 6.50, r.ValorPIS)
	near(t, "COFINS 3%", 30, r.ValorCOFINS)
	querCST(t, "CST PIS", "01", r.CSTPIS)
	near(t, "ICMS segue igual", 120, r.ValorICMS)
	near(t, "IPI segue igual", 50, r.ValorIPI)
}

// No presumido, a exceção da tabela de NCM (monofásico) continua valendo.
func TestCenario_LucroPresumido_MonofasicoDaTabela(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "PR", destTipo: "contribuinte", cfg: comRegime(RegimePresumido, 0),
		item: TaxItem{Ncm: "27101259", ValorUnitario: 1000, Quantidade: 1, OrigemMercadoria: "0"}})
	querCST(t, "CST PIS", "04", r.CSTPIS)
	near(t, "PIS", 0, r.ValorPIS)
}

func TestCenario_LucroReal_SemNCMNaTabela_AliquotaBasica(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "PR", destTipo: "contribuinte",
		item: TaxItem{Ncm: "99999999", ValorUnitario: 1000, Quantidade: 1, OrigemMercadoria: "0"}})
	near(t, "PIS", 16.50, r.ValorPIS)
	near(t, "COFINS", 76, r.ValorCOFINS)
}

// Simples para contribuinte que revende: CSOSN 101 com crédito; sem destaque
// de ICMS, IPI e PIS/COFINS.
func TestCenario_Simples_ContribuinteRevenda_CSOSN101ComCredito(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "PR", destTipo: "contribuinte", finalidade: FinalidadeRevenda, cfg: comRegime(RegimeSimples, 2.56)})
	querCST(t, "CSOSN", "101", r.CSTICMS)
	near(t, "pCredSN", 2.56, r.AliqCreditoSimplesPct)
	near(t, "vCredICMSSN", 25.60, r.ValorCreditoSimples)
	near(t, "ICMS destacado", 0, r.ValorICMS)
	near(t, "diferido", 0, r.ValorICMSDiferido)
	near(t, "IPI", 0, r.ValorIPI)
	querCST(t, "CST IPI", "99", r.CSTIPI)
	near(t, "PIS", 0, r.ValorPIS)
	querCST(t, "CST PIS", "49", r.CSTPIS)
}

func TestCenario_Simples_ConsumidorFinal_CSOSN102SemCredito(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "PR", destTipo: "nao_contribuinte", cfg: comRegime(RegimeSimples, 2.56)})
	querCST(t, "CSOSN", "102", r.CSTICMS)
	near(t, "crédito", 0, r.ValorCreditoSimples)
}

// Simples não recolhe DIFAL na venda a não contribuinte de outra UF (ADI 5464).
func TestCenario_Simples_Interestadual_SemDIFAL(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "GO", destTipo: "nao_contribuinte", cfg: comRegime(RegimeSimples, 2.56)})
	querCST(t, "CSOSN", "102", r.CSTICMS)
	near(t, "DIFAL", 0, r.ValorDIFAL)
	near(t, "FCP", 0, r.ValorFCP)
	near(t, "base UF destino", 0, r.BaseUFDest)
}

// Simples substituto: a ST desconta o ICMS próprio calculado pela alíquota
// normal (LC 123, art. 13, §6º) — CSOSN 201 com crédito.
func TestCenario_Simples_ComSTDaRegra_CSOSN201(t *testing.T) {
	p := &ParametroICMSIPI{ID: 7, MvaPct: 40}
	r := calcularCenario(t, cenario{destUF: "PR", destTipo: "contribuinte", finalidade: FinalidadeRevenda,
		cfg: comRegime(RegimeSimples, 2.56), item: TaxItem{Parametro: p}})
	querCST(t, "CSOSN", "201", r.CSTICMS)
	near(t, "base ST", 1400, r.BaseICMSST)
	near(t, "ICMS-ST (1400×12% − 120)", 48, r.ValorICMSST)
	near(t, "ICMS próprio não destacado", 0, r.ValorICMS)
	near(t, "crédito", 25.60, r.ValorCreditoSimples)
}

func TestCenario_Simples_CSOSNDaRegraVence(t *testing.T) {
	p := &ParametroICMSIPI{ID: 3, CSOSN: "300"}
	r := calcularCenario(t, cenario{destUF: "PR", destTipo: "contribuinte", cfg: comRegime(RegimeSimples, 2.56), item: TaxItem{Parametro: p}})
	querCST(t, "CSOSN", "300", r.CSTICMS)
	near(t, "crédito só em 101/201", 0, r.ValorCreditoSimples)
}

func TestCenario_Simples_RegraIsentaViraCSOSN400(t *testing.T) {
	p := &ParametroICMSIPI{ID: 4, Contrib: LadoParametroICMSIPI{CSTICMS: "40"}}
	r := calcularCenario(t, cenario{destUF: "PR", destTipo: "contribuinte", cfg: comRegime(RegimeSimples, 2.56), item: TaxItem{Parametro: p}})
	querCST(t, "CSOSN", "400", r.CSTICMS)
}

// ── Substituição tributária pela regra ──────────────────────────────────────

// Interestadual com MVA ajustada: 40% vira 50,2439% (inter 12%, intra SP 18%).
func TestCenario_STDaRegra_Interestadual_MVAAjustada(t *testing.T) {
	p := &ParametroICMSIPI{ID: 9, MvaPct: 40, MvaAjustar: true}
	r := calcularCenario(t, cenario{destUF: "SP", destTipo: "contribuinte", finalidade: FinalidadeRevenda, item: TaxItem{Parametro: p}})
	near(t, "MVA ajustada (fração)", 0.502439, r.MVA)
	near(t, "base ST (1050 × 1,502439)", 1577.56, r.BaseICMSST)
	near(t, "ICMS-ST", 163.96, r.ValorICMSST)
	querCST(t, "CST", "10", r.CSTICMS)
}

func TestCenario_STDaRegra_SemAjuste(t *testing.T) {
	p := &ParametroICMSIPI{ID: 9, MvaPct: 40, MvaAjustar: false, Contrib: LadoParametroICMSIPI{AliqSTPct: 18}}
	r := calcularCenario(t, cenario{destUF: "SP", destTipo: "contribuinte", item: TaxItem{Parametro: p}})
	near(t, "MVA original", 0.40, r.MVA)
	near(t, "base ST", 1470, r.BaseICMSST)
	near(t, "ICMS-ST (1470×18% − 120)", 144.60, r.ValorICMSST)
}

// A ST da regra é de operações subsequentes: consumidor final não tem.
func TestCenario_STDaRegra_NaoValeParaConsumidorFinal(t *testing.T) {
	p := &ParametroICMSIPI{ID: 9, MvaPct: 40}
	for _, c := range []cenario{
		{destUF: "PR", destTipo: "nao_contribuinte", item: TaxItem{Parametro: p}},
		{destUF: "PR", destTipo: "contribuinte", finalidade: FinalidadeUsoConsumo, item: TaxItem{Parametro: p}},
	} {
		r := calcularCenario(t, c)
		near(t, "ICMS-ST "+c.destTipo+"/"+c.finalidade, 0, r.ValorICMSST)
	}
}

// ── DIFAL pela regra ────────────────────────────────────────────────────────

func TestCenario_DIFAL_RegraNaoConsidera(t *testing.T) {
	p := &ParametroICMSIPI{ID: 5, DifalTipo: "NAO_CONSIDERA"}
	r := calcularCenario(t, cenario{destUF: "GO", destTipo: "nao_contribuinte", item: TaxItem{Parametro: p}})
	near(t, "DIFAL", 0, r.ValorDIFAL)
	near(t, "ICMS da operação segue", 73.50, r.ValorICMS)
}

func TestCenario_DIFAL_ReducaoDaBase(t *testing.T) {
	p := &ParametroICMSIPI{ID: 5, DifalRedPct: 50}
	r := calcularCenario(t, cenario{destUF: "GO", destTipo: "nao_contribuinte", item: TaxItem{Parametro: p}})
	near(t, "base UF destino", 525, r.BaseUFDest)
	near(t, "DIFAL", 52.50, r.ValorDIFAL)
}

// ── Tipo de nota ─────────────────────────────────────────────────────────────

func TestCenario_TipoDeNota_SemDiferimento(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "PR", destTipo: "contribuinte", flags: InvoiceTypeFlags{SemDiferimento: true}})
	querCST(t, "CST", "00", r.CSTICMS)
	near(t, "diferido", 0, r.ValorICMSDiferido)
}

func TestCenario_TipoDeNota_IsentoImpoeCST40(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "GO", destTipo: "nao_contribuinte", flags: InvoiceTypeFlags{CSTICMSOperacao: "40"}})
	querCST(t, "CST", "40", r.CSTICMS)
	near(t, "ICMS", 0, r.ValorICMS)
	near(t, "DIFAL", 0, r.ValorDIFAL)
	near(t, "base UF destino", 0, r.BaseUFDest)
}

func TestCenario_TipoDeNota_SemPISCOFINS_CST49(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "PR", destTipo: "contribuinte", flags: InvoiceTypeFlags{SkipPISCOFINS: true}})
	querCST(t, "CST PIS", "49", r.CSTPIS)
	querCST(t, "CST COFINS", "49", r.CSTCOFINS)
	near(t, "PIS", 0, r.ValorPIS)
	near(t, "alíquota PIS", 0, r.AliquotaPIS)
}

func TestCenario_TipoDeNota_SemDIFAL(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "GO", destTipo: "nao_contribuinte", flags: InvoiceTypeFlags{SemDIFAL: true}})
	near(t, "DIFAL", 0, r.ValorDIFAL)
}

// SemST desliga a ST da regra; MVA informada à mão na linha continua valendo.
func TestCenario_TipoDeNota_SemST(t *testing.T) {
	p := &ParametroICMSIPI{ID: 9, MvaPct: 40}
	r := calcularCenario(t, cenario{destUF: "PR", destTipo: "contribuinte", flags: InvoiceTypeFlags{SemST: true}, item: TaxItem{Parametro: p}})
	near(t, "ST da regra desligada", 0, r.ValorICMSST)
	r = calcularCenario(t, cenario{destUF: "PR", destTipo: "contribuinte", flags: InvoiceTypeFlags{SemST: true}, item: TaxItem{Parametro: p, MvaPct: 0.40}})
	near(t, "ST da linha ((1000+50 IPI)×1,4×12% − 120)", 56.40, r.ValorICMSST)
}

// ── Peças ────────────────────────────────────────────────────────────────────

func TestRegimeDoCadastro(t *testing.T) {
	for in, want := range map[string]Regime{
		"1": RegimeSimples, "Simples Nacional": RegimeSimples, "2": RegimePresumido, "lucro_presumido": RegimePresumido,
		"3": RegimeReal, "lucro_real": RegimeReal, "": RegimeReal, "xpto": RegimeReal,
	} {
		if got := RegimeDoCadastro(in); got != want {
			t.Errorf("RegimeDoCadastro(%q) = %s, quer %s", in, got, want)
		}
	}
}

func TestCSOSNDoCST(t *testing.T) {
	casos := []struct {
		cst         string
		st, credito bool
		want        string
	}{
		{"00", false, true, "101"}, {"00", false, false, "102"}, {"51", false, true, "101"},
		{"10", true, true, "201"}, {"10", true, false, "202"}, {"30", true, true, "202"},
		{"40", false, true, "400"}, {"41", false, false, "400"}, {"60", false, false, "500"},
	}
	for _, c := range casos {
		if got := CSOSNDoCST(c.cst, c.st, c.credito); got != c.want {
			t.Errorf("CSOSNDoCST(%s, st=%v, crédito=%v) = %s, quer %s", c.cst, c.st, c.credito, got, c.want)
		}
	}
}

func TestMvaAjustada(t *testing.T) {
	got := MvaAjustada(decimal.NewFromFloat(0.40), decimal.NewFromFloat(0.12), decimal.NewFromFloat(0.18))
	near(t, "MVA ajustada", 0.502439, got.InexactFloat64())
	got = MvaAjustada(decimal.NewFromFloat(0.40), decimal.NewFromFloat(0.04), decimal.NewFromFloat(0.18))
	near(t, "MVA ajustada importado", 0.639024, got.InexactFloat64())
}

func TestCenario_NotasExplicamOCalculo(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "PR", destTipo: "contribuinte", finalidade: FinalidadeUsoConsumo})
	if len(r.Notas) < 2 {
		t.Fatalf("notas = %v, quer a origem da regra e o motivo de não diferir", r.Notas)
	}
}

func TestAliquotaInterestadualLegal(t *testing.T) {
	for par, quer := range map[[2]string]float64{
		{"PR", "GO"}: 0.07, {"PR", "ES"}: 0.07, {"SP", "BA"}: 0.07, {"MG", "DF"}: 0.07, {"RS", "AM"}: 0.07,
		{"PR", "SP"}: 0.12, {"ES", "BA"}: 0.12, {"BA", "SP"}: 0.12, {"GO", "PR"}: 0.12, {"CE", "PE"}: 0.12, {"rj", "mg"}: 0.12,
	} {
		if got := AliquotaInterestadualLegal(par[0], par[1]); got != quer {
			t.Errorf("%s→%s: %.2f, quer %.2f", par[0], par[1], got, quer)
		}
	}
}

// Par fora da tabela: vale a resolução, não 12% fixo.
func TestCenario_ParForaDaTabelaUsaAResolucao(t *testing.T) {
	c := defaultFiscalConfig()
	c.UFEmpresa = "SP"
	r := calcularCenario(t, cenario{destUF: "MT", destTipo: "contribuinte", cfg: c})
	near(t, "SP→MT 7%", 0.07, r.AliquotaICMS)
}

// FCP do produto (acréscimo da regra) vence o FCP geral da tabela no DIFAL.
func TestCenario_DIFAL_FCPDoProdutoNaRegra(t *testing.T) {
	p := &ParametroICMSIPI{ID: 8, FCPProdutoPct: 1}
	r := calcularCenario(t, cenario{destUF: "GO", destTipo: "nao_contribuinte", item: TaxItem{Parametro: p}})
	near(t, "FCP %", 1, r.FCPUFDestPct)
	near(t, "FCP 1% de 1050", 10.50, r.ValorFCP)
	near(t, "DIFAL segue", 105, r.ValorDIFAL)
	// UF sem FCP geral na tabela (SP): só o do produto.
	r = calcularCenario(t, cenario{destUF: "SP", destTipo: "nao_contribuinte", item: TaxItem{Parametro: &ParametroICMSIPI{ID: 9, FCPProdutoPct: 2}}})
	near(t, "FCP do produto em SP", 21, r.ValorFCP)
	r = calcularCenario(t, cenario{destUF: "SP", destTipo: "nao_contribuinte"})
	near(t, "sem FCP em SP", 0, r.ValorFCP)
}

// ST na venda interna a quem revende prevalece sobre o diferimento: CST 10,
// nada diferido, e a explicação não pode continuar dizendo que houve
// diferimento (o simulador mostrava "ICMS diferido em 38,46%" com CST 10).
func TestCenario_STDaRegra_Interna_PrevaleceSobreDiferimento(t *testing.T) {
	p := &ParametroICMSIPI{ID: 11, MvaPct: 40}
	r := calcularCenario(t, cenario{destUF: "PR", destTipo: "contribuinte", finalidade: FinalidadeRevenda, item: TaxItem{Parametro: p}})
	querCST(t, "CST", "10", r.CSTICMS)
	near(t, "diferido", 0, r.ValorICMSDiferido)
	for _, n := range r.Notas {
		if strings.HasPrefix(n, "ICMS diferido em ") {
			t.Fatalf("nota de diferimento sobrou com ST: %v", r.Notas)
		}
	}
	if !strings.Contains(strings.Join(r.Notas, " | "), "ICMS-ST, que prevalece") {
		t.Fatalf("falta explicar por que não há diferimento: %v", r.Notas)
	}
}
