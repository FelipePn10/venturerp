package entity

import "github.com/shopspring/decimal"

// Fórmulas de perda da estrutura. O parâmetro de planejamento
// FormulaPerdasEstrutura escolhe qual vale; o padrão do sistema é a 2.
const (
	FormulaPerdaMultiplica = 1 // base × (1 + p/100)
	FormulaPerdaDivide     = 2 // base ÷ (1 − p/100)  ← padrão
	FormulaPerdaIgnora     = 3 // base
	FormulaPerdaPadrao     = FormulaPerdaDivide
)

// QuantidadeComPerda aplica a fórmula de perdas sobre a quantidade bruta.
//
// Fonte única da conta. Antes cada rotina tinha a sua: o MRP respeitava o
// parâmetro (padrão 2, divisão) e a criação da ordem, o encerramento, o
// apontamento e o custo fixavam a fórmula 1 (multiplicação) no código. Para uma
// perda de 5% sobre 2250, o MRP comprava 2368,42 e a ordem consumia 2362,50 —
// a fábrica ficava permanentemente com uma sobra que ninguém explicava, e o
// custo do produto saía calculado sobre uma terceira quantidade.
//
// A divisão é a leitura correta de "perde-se p% do que foi requisitado": para
// obter `base` peças boas é preciso requisitar base/(1−p).
func QuantidadeComPerda(base, perda float64, formula int) float64 {
	if perda <= 0 {
		return base
	}
	switch formula {
	case FormulaPerdaMultiplica:
		return base * (1 + perda/100)
	case FormulaPerdaIgnora:
		return base
	default: // FormulaPerdaDivide
		if denominador := 1 - perda/100; denominador > 0 {
			return base / denominador
		}
		return base
	}
}

// QuantidadeComPerdaDecimal é QuantidadeComPerda em decimal, para quem soma
// dinheiro sobre a quantidade (o custo-padrão). A conta não arredonda — quem grava arredonda na
// escala da coluna.
func QuantidadeComPerdaDecimal(base, perda decimal.Decimal, formula int) decimal.Decimal {
	if !perda.IsPositive() {
		return base
	}
	p := perda.Div(decimal.NewFromInt(100))
	um := decimal.NewFromInt(1)
	switch formula {
	case FormulaPerdaMultiplica:
		return base.Mul(um.Add(p))
	case FormulaPerdaIgnora:
		return base
	default: // FormulaPerdaDivide
		if d := um.Sub(p); d.IsPositive() {
			return base.Div(d)
		}
		return base
	}
}

// EscalaDaRequisicao é o número de casas em que o componente é comprado e
// recebido: o pedido de compra, a ordem planejada e a nota fiscal gravam
// NUMERIC(15,4).
const EscalaDaRequisicao = 4

// QuantidadeDoComponente é quanto do componente uma ordem do pai requisita:
// base (a quantidade do pai, ou 1 para quantidade fixa) × quantidade por
// unidade, com a perda, arredondada para cima na escala da compra.
//
// Fonte única para o MRP e para a OF. Antes a OF guardava 6 casas e a compra 4:
// o MRP pedia 52.593,317345 kg, o pedido de compra gravava 52.593,3173 e as OFs
// paravam por "saldo insuficiente" de 0,00004 kg. Arredondar para cima (nunca
// para baixo) mantém a ordem coberta pelo que foi comprado.
func QuantidadeDoComponente(base, porUnidade, perda decimal.Decimal, formula int) decimal.Decimal {
	q := QuantidadeComPerdaDecimal(base.Mul(porUnidade.Round(6)), perda, formula)
	return q.Round(6).RoundCeil(EscalaDaRequisicao)
}
