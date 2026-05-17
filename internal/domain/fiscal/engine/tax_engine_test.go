package engine

import (
	"math"
	"testing"
)

// near checks two float64 values are within epsilon of each other.
func near(t *testing.T, label string, want, got float64) {
	t.Helper()
	const eps = 0.01
	if math.Abs(want-got) > eps {
		t.Errorf("%s: want %.2f, got %.2f", label, want, got)
	}
}

func defaultFiscalConfig() FiscalConfig {
	return FiscalConfig{
		UFEmpresa:                 "PR",
		IcmsInternoAliquota:       0.12,
		IcmsDiferimentoPercentual: 38.46,
	}
}

func defaultNcmTable() map[string]*NcmTaxConfig {
	return map[string]*NcmTaxConfig{
		"84714900": {
			AliqIPI:    0.05,
			AliqPis:    0.0165,
			AliqCofins: 0.076,
			CstPis:     "01",
			CstCofins:  "01",
			CstIPI:     "50",
		},
		"22021000": {
			AliqIPI:    0.10,
			AliqPis:    0.0165,
			AliqCofins: 0.076,
			CstPis:     "01",
			CstCofins:  "01",
			CstIPI:     "50",
		},
		// NCM monofásico PIS/COFINS zero
		"27101259": {
			AliqIPI:    0,
			AliqPis:    0,
			AliqCofins: 0,
			CstPis:     "04",
			CstCofins:  "04",
			CstIPI:     "53",
		},
	}
}

func defaultInterstateTable() map[string]float64 {
	return map[string]float64{
		"PRSP": 0.12,
		"PRSC": 0.12,
		"PRRS": 0.12,
		"PRGO": 0.07,
		"PRBA": 0.07,
		"PRCE": 0.07,
		"PRPA": 0.07,
	}
}

func defaultInternalTable() map[string]ICMSInternalConfig {
	return map[string]ICMSInternalConfig{
		"SP": {ICMS: 0.18, FCP: 0},
		"SC": {ICMS: 0.17, FCP: 0},
		"GO": {ICMS: 0.17, FCP: 0.02},
		"BA": {ICMS: 0.19, FCP: 0.02},
		"PA": {ICMS: 0.17, FCP: 0.02},
		"CE": {ICMS: 0.18, FCP: 0.02},
		"PR": {ICMS: 0.12, FCP: 0},
	}
}

// ---------------------------------------------------------------------------
// Scenario 1: Intra-state (PR→PR), contributor — ICMS 12%, diferimento 38.46%
// ---------------------------------------------------------------------------

func TestScenario_IntraState_Contributor_Diferimento(t *testing.T) {
	params := TaxCalculationParams{
		Itens: []TaxItem{
			{
				Ncm:           "84714900",
				ValorUnitario: 1000.00,
				Quantidade:    1,
			},
		},
		EmitenteUF:  "PR",
		DestinoUF:   "PR",
		DestinoTipo: "contribuinte",
		Cfop:        "5101",
	}

	res, err := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())
	if err != nil {
		t.Fatal(err)
	}

	if res.Cenario != "INTERNA_CONTRIBUINTE" {
		t.Errorf("cenario: want INTERNA_CONTRIBUINTE, got %s", res.Cenario)
	}

	item := res.Itens[0]

	// Base ICMS for contributor does NOT include IPI
	near(t, "BaseICMS", 1000.00, item.BaseICMS)
	near(t, "AliquotaICMS", 0.12, item.AliquotaICMS)
	near(t, "ValorICMS", 120.00, item.ValorICMS)

	// Diferimento 38.46% of 120 = 46.15
	near(t, "ValorICMSDiferido", 46.15, item.ValorICMSDiferido)

	if item.CSTICMS != "51" {
		t.Errorf("CSTICMS: want 51, got %s", item.CSTICMS)
	}

	// IPI: 5% of 1000
	near(t, "ValorIPI", 50.00, item.ValorIPI)

	// PIS/COFINS: 1.65% and 7.6% of 1000
	near(t, "ValorPIS", 16.50, item.ValorPIS)
	near(t, "ValorCOFINS", 76.00, item.ValorCOFINS)

	// DIFAL should be zero (intra-state)
	near(t, "ValorDIFAL", 0, item.ValorDIFAL)
}

// ---------------------------------------------------------------------------
// Scenario 2: Intra-state (PR→PR), non-contributor — base includes IPI
// ---------------------------------------------------------------------------

func TestScenario_IntraState_NonContributor_IPIInBase(t *testing.T) {
	params := TaxCalculationParams{
		Itens: []TaxItem{
			{
				Ncm:           "84714900",
				ValorUnitario: 1000.00,
				Quantidade:    1,
			},
		},
		EmitenteUF:  "PR",
		DestinoUF:   "PR",
		DestinoTipo: "nao_contribuinte",
		Cfop:        "5102",
	}

	res, err := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())
	if err != nil {
		t.Fatal(err)
	}

	item := res.Itens[0]

	// Base ICMS for non-contributor includes IPI: 1000 + 50 = 1050
	near(t, "BaseICMS", 1050.00, item.BaseICMS)
	near(t, "ValorICMS", 126.00, item.ValorICMS) // 12% of 1050

	if item.CSTICMS != "00" {
		t.Errorf("CSTICMS: want 00, got %s", item.CSTICMS)
	}

	// No diferimento for non-contributor
	near(t, "ValorICMSDiferido", 0, item.ValorICMSDiferido)
}

// ---------------------------------------------------------------------------
// Scenario 3: Interstate (PR→SP), non-contributor — DIFAL applies
// ---------------------------------------------------------------------------

func TestScenario_Interstate_NonContributor_DIFAL(t *testing.T) {
	params := TaxCalculationParams{
		Itens: []TaxItem{
			{
				Ncm:           "84714900",
				ValorUnitario: 1000.00,
				Quantidade:    1,
			},
		},
		EmitenteUF:  "PR",
		DestinoUF:   "SP",
		DestinoTipo: "nao_contribuinte",
		Cfop:        "6108",
	}

	res, err := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())
	if err != nil {
		t.Fatal(err)
	}

	item := res.Itens[0]

	// Interstate to non-contributor: base includes IPI (1050), aliquota 12%
	near(t, "AliquotaICMS", 0.12, item.AliquotaICMS)
	near(t, "BaseICMS", 1050.00, item.BaseICMS)
	near(t, "ValorICMS", 126.00, item.ValorICMS)

	// DIFAL: (18% - 12%) of 1050 = 6% of 1050 = 63.00
	near(t, "ValorDIFAL", 63.00, item.ValorDIFAL)

	// SP has no FCP
	near(t, "ValorFCP", 0, item.ValorFCP)
}

// ---------------------------------------------------------------------------
// Scenario 4: Interstate (PR→GO), non-contributor — DIFAL + FCP 2%
// ---------------------------------------------------------------------------

func TestScenario_Interstate_NonContributor_DIFAL_With_FCP(t *testing.T) {
	params := TaxCalculationParams{
		Itens: []TaxItem{
			{
				Ncm:           "84714900",
				ValorUnitario: 1000.00,
				Quantidade:    1,
			},
		},
		EmitenteUF:  "PR",
		DestinoUF:   "GO",
		DestinoTipo: "nao_contribuinte",
		Cfop:        "6108",
	}

	res, err := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())
	if err != nil {
		t.Fatal(err)
	}

	item := res.Itens[0]

	// PR→GO interstate: 7% aliquota
	near(t, "AliquotaICMS", 0.07, item.AliquotaICMS)

	// DIFAL: (17% - 7%) = 10% of base (1050, includes IPI for non-contributor)
	near(t, "ValorDIFAL", 105.00, item.ValorDIFAL)

	// FCP: 2% of 1050 = 21.00
	near(t, "ValorFCP", 21.00, item.ValorFCP)
}

// ---------------------------------------------------------------------------
// Scenario 5: Interstate (PR→SP), contributor — no DIFAL
// ---------------------------------------------------------------------------

func TestScenario_Interstate_Contributor_NoDIFAL(t *testing.T) {
	params := TaxCalculationParams{
		Itens: []TaxItem{
			{
				Ncm:           "84714900",
				ValorUnitario: 1000.00,
				Quantidade:    1,
			},
		},
		EmitenteUF:  "PR",
		DestinoUF:   "SP",
		DestinoTipo: "contribuinte",
		Cfop:        "6101",
	}

	res, err := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())
	if err != nil {
		t.Fatal(err)
	}

	item := res.Itens[0]

	// Interstate to contributor: base does NOT include IPI
	near(t, "BaseICMS", 1000.00, item.BaseICMS)
	near(t, "ValorICMS", 120.00, item.ValorICMS) // 12% interstate PR→SP

	// No DIFAL for contributor
	near(t, "ValorDIFAL", 0, item.ValorDIFAL)
	near(t, "ValorFCP", 0, item.ValorFCP)
}

// ---------------------------------------------------------------------------
// Scenario 6: Resolução SF 13/2012 — imported goods origin 3 → 4% interstate
// ---------------------------------------------------------------------------

func TestScenario_ImportedGoods_SF13_FourPercent(t *testing.T) {
	params := TaxCalculationParams{
		Itens: []TaxItem{
			{
				Ncm:              "84714900",
				ValorUnitario:    1000.00,
				Quantidade:       1,
				OrigemMercadoria: "3", // importado
			},
		},
		EmitenteUF:  "PR",
		DestinoUF:   "SP",
		DestinoTipo: "contribuinte",
		Cfop:        "6101",
	}

	res, err := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())
	if err != nil {
		t.Fatal(err)
	}

	item := res.Itens[0]

	// Res. SF 13/2012: imported goods → 4% regardless of destination UF
	near(t, "AliquotaICMS", 0.04, item.AliquotaICMS)
	near(t, "ValorICMS", 40.00, item.ValorICMS)
}

// Also test origins 4, 5, 8.
func TestScenario_ImportedGoods_Origins_4_5_8(t *testing.T) {
	for _, origin := range []string{"4", "5", "8"} {
		params := TaxCalculationParams{
			Itens: []TaxItem{
				{Ncm: "84714900", ValorUnitario: 1000, Quantidade: 1, OrigemMercadoria: origin},
			},
			EmitenteUF: "PR", DestinoUF: "BA", DestinoTipo: "contribuinte", Cfop: "6101",
		}
		res, err := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())
		if err != nil {
			t.Fatal(err)
		}
		near(t, "AliquotaICMS origin "+origin, 0.04, res.Itens[0].AliquotaICMS)
	}
}

// ---------------------------------------------------------------------------
// Scenario 7: IPI zero for NCM without IPI (e.g. basic food)
// ---------------------------------------------------------------------------

func TestScenario_IPI_Zero_When_No_NCM(t *testing.T) {
	params := TaxCalculationParams{
		Itens: []TaxItem{
			{
				Ncm:           "99999999", // not in NCM table
				ValorUnitario: 500.00,
				Quantidade:    2,
			},
		},
		EmitenteUF: "PR", DestinoUF: "PR", DestinoTipo: "contribuinte", Cfop: "5101",
	}

	res, _ := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())
	item := res.Itens[0]

	near(t, "ValorIPI_zero", 0, item.ValorIPI)
	if item.CSTIPI != "50" {
		t.Errorf("CSTIPI default: want 50, got %s", item.CSTIPI)
	}
}

// ---------------------------------------------------------------------------
// Scenario 8: PIS/COFINS monofásico CST 04 → aliquota zero
// ---------------------------------------------------------------------------

func TestScenario_PIS_COFINS_Monofasico_Zero(t *testing.T) {
	params := TaxCalculationParams{
		Itens: []TaxItem{
			{Ncm: "27101259", ValorUnitario: 1000, Quantidade: 1},
		},
		EmitenteUF: "PR", DestinoUF: "PR", DestinoTipo: "contribuinte", Cfop: "5101",
	}

	res, _ := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())
	item := res.Itens[0]

	near(t, "ValorPIS_zero", 0, item.ValorPIS)
	near(t, "ValorCOFINS_zero", 0, item.ValorCOFINS)
	if item.CSTPIS != "04" {
		t.Errorf("CSTPIS: want 04, got %s", item.CSTPIS)
	}
	if item.CSTCOFINS != "04" {
		t.Errorf("CSTCOFINS: want 04, got %s", item.CSTCOFINS)
	}
}

// ---------------------------------------------------------------------------
// Scenario 9: NCM with custom PIS/COFINS rates (higher aliquota for beverage)
// ---------------------------------------------------------------------------

func TestScenario_NCM_Custom_PIS_COFINS(t *testing.T) {
	params := TaxCalculationParams{
		Itens: []TaxItem{
			{Ncm: "22021000", ValorUnitario: 100, Quantidade: 10},
		},
		EmitenteUF: "PR", DestinoUF: "PR", DestinoTipo: "contribuinte", Cfop: "5101",
	}

	res, _ := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())
	item := res.Itens[0]

	// base 1000, PIS 1.65% = 16.50, COFINS 7.6% = 76
	near(t, "ValorPIS", 16.50, item.ValorPIS)
	near(t, "ValorCOFINS", 76.00, item.ValorCOFINS)

	// IPI 10%: 100
	near(t, "ValorIPI", 100.00, item.ValorIPI)
}

// ---------------------------------------------------------------------------
// Scenario 10: Freight apportionment across two items
// ---------------------------------------------------------------------------

func TestScenario_Freight_Apportionment(t *testing.T) {
	// Item A = 400, Item B = 600, total = 1000
	// Freight = 100 → A gets 40, B gets 60
	params := TaxCalculationParams{
		Itens: []TaxItem{
			{Ncm: "99999999", ValorUnitario: 400, Quantidade: 1, ValorFrete: 100},
			{Ncm: "99999999", ValorUnitario: 600, Quantidade: 1, ValorFrete: 100},
		},
		EmitenteUF: "PR", DestinoUF: "SP", DestinoTipo: "contribuinte", Cfop: "6101",
	}

	res, _ := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())

	// Item A base ICMS = 400 + 40 (freight share) = 440
	near(t, "ItemA_BaseICMS", 440.00, res.Itens[0].BaseICMS)
	// Item B base ICMS = 600 + 60 = 660
	near(t, "ItemB_BaseICMS", 660.00, res.Itens[1].BaseICMS)
}

// ---------------------------------------------------------------------------
// Scenario 11: Discount apportionment
// ---------------------------------------------------------------------------

func TestScenario_Discount_Apportionment(t *testing.T) {
	// Discount 100 across two equal items (each 500)
	params := TaxCalculationParams{
		Itens: []TaxItem{
			{Ncm: "99999999", ValorUnitario: 500, Quantidade: 1, ValorDesconto: 100},
			{Ncm: "99999999", ValorUnitario: 500, Quantidade: 1, ValorDesconto: 100},
		},
		EmitenteUF: "PR", DestinoUF: "PR", DestinoTipo: "contribuinte", Cfop: "5101",
	}

	res, _ := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())

	// Each item: 500 - 50 discount share = 450
	near(t, "Item0_BaseICMS", 450.00, res.Itens[0].BaseICMS)
	near(t, "Item1_BaseICMS", 450.00, res.Itens[1].BaseICMS)
}

// ---------------------------------------------------------------------------
// Scenario 12: Totals aggregation across multiple items
// ---------------------------------------------------------------------------

func TestScenario_Totals_Aggregation(t *testing.T) {
	params := TaxCalculationParams{
		Itens: []TaxItem{
			{Ncm: "84714900", ValorUnitario: 1000, Quantidade: 1},
			{Ncm: "84714900", ValorUnitario: 2000, Quantidade: 1},
		},
		EmitenteUF: "PR", DestinoUF: "PR", DestinoTipo: "contribuinte", Cfop: "5101",
	}

	res, _ := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())

	// BaseICMS total: 1000 + 2000 = 3000
	near(t, "TotalBaseICMS", 3000.00, res.Totais.BaseICMS)
	// ValorICMS total: 120 + 240 = 360
	near(t, "TotalValorICMS", 360.00, res.Totais.ValorICMS)
	// ValorIPI total: 50 + 100 = 150
	near(t, "TotalValorIPI", 150.00, res.Totais.ValorIPI)
	// ValorPIS total: 16.50 + 33 = 49.50
	near(t, "TotalValorPIS", 49.50, res.Totais.ValorPIS)
	// ValorCOFINS total: 76 + 152 = 228
	near(t, "TotalValorCOFINS", 228.00, res.Totais.ValorCOFINS)
	// Diferimento: 38.46% of 120 = 46.15, 38.46% of 240 = 92.30, sum = 138.45
	near(t, "TotalValorDiferido", 138.45, res.Totais.ValorDiferido)
}

// ---------------------------------------------------------------------------
// Scenario 13: Pessoa física (interstate) — treated as non-contributor for DIFAL
// ---------------------------------------------------------------------------

func TestScenario_PessoaFisica_DIFAL(t *testing.T) {
	params := TaxCalculationParams{
		Itens: []TaxItem{
			{Ncm: "84714900", ValorUnitario: 1000, Quantidade: 1},
		},
		EmitenteUF: "PR", DestinoUF: "BA", DestinoTipo: "pessoa_fisica", Cfop: "6108",
	}

	res, _ := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())
	item := res.Itens[0]

	// PR→BA 7%, BA internal 19%, DIFAL = 12% of 1050 (includes IPI)
	near(t, "ValorDIFAL", 126.00, item.ValorDIFAL)
	// FCP: 2% of 1050 = 21.00
	near(t, "ValorFCP", 21.00, item.ValorFCP)
}

// ---------------------------------------------------------------------------
// Scenario 14: Zero IPI NCM (NCM in table with AliqIPI=0) — no IPI in total
// ---------------------------------------------------------------------------

func TestScenario_IPIZero_NoIPIInBase(t *testing.T) {
	params := TaxCalculationParams{
		Itens: []TaxItem{
			{Ncm: "27101259", ValorUnitario: 1000, Quantidade: 1},
		},
		EmitenteUF: "PR", DestinoUF: "PR", DestinoTipo: "nao_contribuinte", Cfop: "5102",
	}

	res, _ := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())
	item := res.Itens[0]

	near(t, "ValorIPI", 0, item.ValorIPI)
	// Base includes IPI (which is zero): 1000 + 0 = 1000
	near(t, "BaseICMS_nonContributor_IPIzero", 1000.00, item.BaseICMS)
}

// ---------------------------------------------------------------------------
// Scenario 15: Quantity > 1 calculation correctness
// ---------------------------------------------------------------------------

func TestScenario_MultipleQuantity(t *testing.T) {
	params := TaxCalculationParams{
		Itens: []TaxItem{
			{Ncm: "84714900", ValorUnitario: 200, Quantidade: 5},
		},
		EmitenteUF: "PR", DestinoUF: "SP", DestinoTipo: "contribuinte", Cfop: "6101",
	}

	res, _ := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())
	item := res.Itens[0]

	// 200 * 5 = 1000 base
	near(t, "BaseICMS", 1000.00, item.BaseICMS)
	near(t, "ValorICMS", 120.00, item.ValorICMS) // 12% of 1000

	// IPI: 5% of 1000 = 50
	near(t, "ValorIPI", 50.00, item.ValorIPI)
}

// ---------------------------------------------------------------------------
// Scenario 16: PR→SC (same region Sul) — 12% interstate
// ---------------------------------------------------------------------------

func TestScenario_Interstate_SameRegion_12pct(t *testing.T) {
	params := TaxCalculationParams{
		Itens: []TaxItem{
			{Ncm: "84714900", ValorUnitario: 1000, Quantidade: 1},
		},
		EmitenteUF: "PR", DestinoUF: "SC", DestinoTipo: "contribuinte", Cfop: "6101",
	}

	res, _ := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())

	near(t, "AliquotaICMS_SC", 0.12, res.Itens[0].AliquotaICMS)
}

// ---------------------------------------------------------------------------
// Scenario 17: Default 12% when UF pair not in table
// ---------------------------------------------------------------------------

func TestScenario_Interstate_DefaultRate(t *testing.T) {
	params := TaxCalculationParams{
		Itens: []TaxItem{
			{Ncm: "84714900", ValorUnitario: 1000, Quantidade: 1},
		},
		EmitenteUF: "PR", DestinoUF: "AM", // not mapped
		DestinoTipo: "contribuinte", Cfop: "6101",
	}

	res, _ := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())

	// Default fallback 12%
	near(t, "AliquotaICMS_default", 0.12, res.Itens[0].AliquotaICMS)
}

// ---------------------------------------------------------------------------
// Scenario 18: Cenário string is set correctly for each case
// ---------------------------------------------------------------------------

func TestScenario_CenarioString(t *testing.T) {
	item := TaxItem{Ncm: "84714900", ValorUnitario: 100, Quantidade: 1}

	cases := []struct {
		emitenteUF  string
		destinoUF   string
		destinoTipo string
		want        string
	}{
		{"PR", "PR", "contribuinte", "INTERNA_CONTRIBUINTE"},
		{"PR", "PR", "nao_contribuinte", "INTERNA_NAO_CONTRIBUINTE"},
		{"PR", "SP", "contribuinte", "INTERESTADUAL"},
		{"PR", "SP", "nao_contribuinte", "INTERESTADUAL"},
	}

	for _, tc := range cases {
		params := TaxCalculationParams{
			Itens:       []TaxItem{item},
			EmitenteUF:  tc.emitenteUF,
			DestinoUF:   tc.destinoUF,
			DestinoTipo: tc.destinoTipo,
		}
		res, _ := CalcularImpostos(params, defaultNcmTable(), defaultInterstateTable(), defaultInternalTable(), TaxScenarioConfig{}, defaultFiscalConfig())
		if res.Cenario != tc.want {
			t.Errorf("cenario %s→%s %s: want %s, got %s", tc.emitenteUF, tc.destinoUF, tc.destinoTipo, tc.want, res.Cenario)
		}
	}
}
