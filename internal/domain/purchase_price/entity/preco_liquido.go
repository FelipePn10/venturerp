package entity

import (
	"sort"

	"github.com/shopspring/decimal"
)

// CasasDoPrecoUnitario: o preço unitário do pedido guarda 6 casas.
const CasasDoPrecoUnitario = 6

var cem = decimal.NewFromInt(100)

// PrecoLiquido aplica ao preço da tabela os descontos e acréscimos em cadeia,
// na ordem da sequência: 100 com 10 % de desconto e depois 5 % de desconto dá
// 85,50 (não 85); um acréscimo fixo de 2 soma 2 por unidade da linha da tabela.
// Arredonda nas casas do preço unitário e nunca fica negativo (desconto fixo
// maior que o preço zera, e o pedido sem preço é recusado adiante).
//
// É a regra única: linha manual do pedido, pedido gerado pela requisição e
// aprovação da sugestão do MRP usam esta conta — antes nenhum deles aplicava
// os ajustes que a VSUP0120 promete.
func PrecoLiquido(preco decimal.Decimal, ajustes []*PriceAdjustment) decimal.Decimal {
	ordem := make([]*PriceAdjustment, 0, len(ajustes))
	for _, a := range ajustes {
		if a != nil {
			ordem = append(ordem, a)
		}
	}
	sort.SliceStable(ordem, func(i, j int) bool { return ordem[i].Sequence < ordem[j].Sequence })
	p := preco
	for _, a := range ordem {
		var delta decimal.Decimal
		if a.CalculationType == "PERCENT" {
			delta = p.Mul(a.Value).Div(cem)
		} else {
			delta = a.Value
		}
		if a.Kind == "DISCOUNT" {
			p = p.Sub(delta)
		} else {
			p = p.Add(delta)
		}
	}
	if p.IsNegative() {
		return decimal.Zero
	}
	return p.Round(CasasDoPrecoUnitario)
}

// PrecoLiquido da linha da tabela.
func (it *PurchasePriceTableItem) PrecoLiquido() decimal.Decimal {
	return PrecoLiquido(it.Price, it.Adjustments)
}
