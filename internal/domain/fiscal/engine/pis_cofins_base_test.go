package engine

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

// Exemplo da Tecnofer (NF nº 1, 11/11/2026): venda interestadual de 6.950,00
// com ICMS de 7%. O ICMS destacado (486,50) sai da base do PIS/COFINS
// (STF, Tema 69): 6.463,50 × 1,65% = 106,65 e × 7,6% = 491,23. Antes o ERP
// calculava sobre 6.950,00 (114,68 e 528,20).
func TestPISCOFINS_ExcluiICMSDestacado_ExemploTecnofer(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "BA", destTipo: "contribuinte", finalidade: FinalidadeRevenda,
		item: TaxItem{Ncm: "99999999", ValorUnitario: decimal.NewFromFloat(6950), Quantidade: decimal.NewFromInt(1), OrigemMercadoria: "0"}})
	near(t, "ICMS 7%", 486.50, r.ValorICMS)
	near(t, "base PIS", 6463.50, r.BasePIS)
	near(t, "base COFINS", 6463.50, r.BaseCOFINS)
	near(t, "PIS 1,65%", 106.65, r.ValorPIS)
	near(t, "COFINS 7,6%", 491.23, r.ValorCOFINS)
	if !temNota(r, "Tema 69") {
		t.Errorf("a linha deveria explicar a exclusão do ICMS; notas: %v", r.Notas)
	}
}

// Só o ICMS próprio destacado sai: ICMS-ST não é do vendedor e não entra na conta.
func TestPISCOFINS_STNaoEntraNaExclusao(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "PR", destTipo: "contribuinte", finalidade: FinalidadeRevenda,
		item: TaxItem{Ncm: "99999999", ValorUnitario: decimal.NewFromFloat(1000), Quantidade: decimal.NewFromInt(1), OrigemMercadoria: "0",
			MvaPct: decimal.NewFromFloat(0.40)}})
	if !r.ValorICMSST.IsPositive() {
		t.Fatalf("cenário sem ST: %+v", r)
	}
	// CST 10 (ST prevalece sobre o diferimento): ICMS próprio 120 destacado inteiro.
	near(t, "base PIS", 880, r.BasePIS)
	near(t, "PIS", 14.52, r.ValorPIS)
}

// Operação sem ICMS destacado (CST isento imposto pelo tipo de nota) segue com a
// base cheia.
func TestPISCOFINS_SemICMSDestacado_BaseCheia(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "PR", destTipo: "contribuinte", finalidade: FinalidadeRevenda,
		flags: InvoiceTypeFlags{CSTICMSOperacao: "40"}})
	near(t, "ICMS", 0, r.ValorICMS)
	near(t, "base PIS", 1000, r.BasePIS)
	near(t, "PIS", 16.50, r.ValorPIS)
	near(t, "COFINS", 76, r.ValorCOFINS)
	if temNota(r, "Tema 69") {
		t.Errorf("sem ICMS destacado não há exclusão a explicar; notas: %v", r.Notas)
	}
}

// O tipo de nota que não calcula PIS/COFINS continua zerado (CST 49).
func TestPISCOFINS_TipoDeNotaSemPISCOFINS_SegueZerado(t *testing.T) {
	r := calcularCenario(t, cenario{destUF: "BA", destTipo: "contribuinte", flags: InvoiceTypeFlags{SkipPISCOFINS: true}})
	near(t, "PIS", 0, r.ValorPIS)
	near(t, "COFINS", 0, r.ValorCOFINS)
	querCST(t, "CST PIS", "49", r.CSTPIS)
	if temNota(r, "Tema 69") {
		t.Errorf("sem PIS/COFINS não há exclusão a explicar; notas: %v", r.Notas)
	}
}

func temNota(r TaxItemResult, trecho string) bool {
	for _, n := range r.Notas {
		if strings.Contains(n, trecho) {
			return true
		}
	}
	return false
}
