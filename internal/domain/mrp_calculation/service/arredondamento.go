package service

import "github.com/shopspring/decimal"

// O motor do MRP ainda trabalha em float64, mas todo arredondamento para cima
// (lote múltiplo, ciclos de máquina, cargas de consumível) passa por aqui em
// decimal. Em float, 0,3 ÷ 0,1 dá 2,9999999999999996 e 0,69 ÷ 0,23 dá
// 3,0000000000000004 — o Ceil devolveria 0,30000000000000004 ou um ciclo a
// mais. decimal.NewFromFloat usa a menor representação que volta ao mesmo
// float, então 0,1 entra como 0,1 exato.

// quantidadeExata elimina o resíduo binário de uma quantidade, na escala das
// colunas NUMERIC(18,6) onde ela será gravada.
func quantidadeExata(v float64) float64 {
	return decimal.NewFromFloat(v).Round(6).InexactFloat64()
}

// divisoesParaCima é ⌈a ÷ b⌉ calculado em decimal. b deve ser positivo.
func divisoesParaCima(a, b float64) float64 {
	da, db := decimal.NewFromFloat(a).Round(6), decimal.NewFromFloat(b).Round(6)
	return da.Div(db).Ceil().InexactFloat64()
}

// multiploParaCima arredonda a para o menor múltiplo de m que não fica abaixo
// de a, em decimal.
func multiploParaCima(a, m float64) float64 {
	da, dm := decimal.NewFromFloat(a).Round(6), decimal.NewFromFloat(m).Round(6)
	return da.Div(dm).Ceil().Mul(dm).InexactFloat64()
}

// dq: quantidade lida em float (cadastros ainda em float, como a estrutura e a
// previsão) para o decimal do MRP, na escala NUMERIC(18,6) das colunas.
func dq(v float64) decimal.Decimal { return decimal.NewFromFloat(v).Round(6) }

// quantidadeComPerda é a perda da estrutura (structentity.QuantidadeComPerda)
// em decimal: 1 multiplica (1 + %/100), 3 ignora, 2 (padrão) divide (1 − %/100).
// Arredonda na escala das colunas: a divisão pode ser dízima.
func quantidadeComPerda(base decimal.Decimal, perda float64, formula int) decimal.Decimal {
	if perda <= 0 {
		return base
	}
	p := dq(perda).Div(decimal.NewFromInt(100))
	um := decimal.NewFromInt(1)
	switch formula {
	case 1:
		return base.Mul(um.Add(p)).Round(6)
	case 3:
		return base
	default:
		if d := um.Sub(p); d.IsPositive() {
			return base.Div(d).Round(6)
		}
		return base
	}
}
