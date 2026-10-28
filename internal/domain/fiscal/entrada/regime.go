package entrada

import (
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/engine"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

// CreditosDoRegime tira da linha os créditos que o regime da empresa não
// permite aproveitar e diz quais foram tirados:
//   - Simples Nacional: nenhum crédito de ICMS, IPI, PIS e COFINS (LC 123,
//     art. 23) — os tributos destacados viram custo;
//   - Lucro Presumido: PIS e COFINS são cumulativos, sem crédito (Lei
//     10.637/02 e 10.833/03 valem só para o não cumulativo); ICMS e IPI seguem.
//
// O tipo de operação de entrada continua decidindo o resto (uso e consumo sem
// crédito, por exemplo); o regime só restringe.
func CreditosDoRegime(regime engine.Regime, it *entity.FiscalEntryItem) []string {
	var tirados []string
	tirar := func(flag *bool, nome string) {
		if *flag {
			*flag = false
			tirados = append(tirados, nome)
		}
	}
	switch regime {
	case engine.RegimeSimples:
		tirar(&it.GeraCreditoICMS, "ICMS")
		tirar(&it.GeraCreditoIPI, "IPI")
		tirar(&it.GeraCreditoPIS, "PIS")
		tirar(&it.GeraCreditoCOFINS, "COFINS")
	case engine.RegimePresumido:
		tirar(&it.GeraCreditoPIS, "PIS")
		tirar(&it.GeraCreditoCOFINS, "COFINS")
	}
	return tirados
}

// cfopUsoConsumoInterestadual: compra de outra UF para uso e consumo (2556,
// 2407 com ST) ou ativo imobilizado (2551, 2406 com ST).
var cfopUsoConsumoInterestadual = map[string]bool{"2556": true, "2407": true, "2551": true, "2406": true}

// CFOPDeUsoConsumoInterestadual diz se a entrada paga DIFAL do comprador.
func CFOPDeUsoConsumoInterestadual(cfopEntrada string) bool {
	return cfopUsoConsumoInterestadual[soNumeros(cfopEntrada)]
}

// DifalUsoConsumo é o diferencial de alíquota que a empresa recolhe à própria
// UF na compra interestadual para uso, consumo ou ativo, pela base dupla da LC
// 190/2022 (o ICMS do destino "por dentro"):
//
//	base  = (valor da operação − ICMS da origem) / (1 − alíquota interna)
//	DIFAL = base × alíquota interna − ICMS da origem   (mínimo zero)
//
// Frações (0,195 = 19,5%). Devolve a base e o DIFAL, em centavos.
func DifalUsoConsumo(valorOperacao, icmsOrigem, aliqInterna decimal.Decimal) (base, difal decimal.Decimal) {
	um := decimal.NewFromInt(1)
	if !aliqInterna.IsPositive() || !um.Sub(aliqInterna).IsPositive() || !valorOperacao.IsPositive() {
		return decimal.Zero, decimal.Zero
	}
	base = valorOperacao.Sub(icmsOrigem).Div(um.Sub(aliqInterna))
	difal = base.Mul(aliqInterna).Sub(icmsOrigem)
	if difal.IsNegative() {
		difal = decimal.Zero
	}
	return base.Round(2), difal.Round(2)
}
