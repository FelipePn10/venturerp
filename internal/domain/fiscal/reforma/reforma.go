// Package reforma calcula o grupo IBS/CBS da NF-e na transição da reforma
// tributária (LC 214/2025).
//
// Em 2026 o grupo é obrigatório na NF-e do regime normal com as alíquotas de
// teste — CBS 0,9% e IBS estadual 0,1% (municipal 0%) —, sem recolhimento: o
// valor é compensado com PIS/COFINS. A partir de 2027 as alíquotas mudam e
// esta tabela tem de ser atualizada antes da primeira nota do ano.
package reforma

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// Aliquotas do IBS (estadual e municipal) e da CBS, em %.
type Aliquotas struct {
	IBSUF, IBSMun, CBS decimal.Decimal
}

// AliquotasDoAno devolve as alíquotas vigentes no ano da emissão.
func AliquotasDoAno(ano int) (Aliquotas, error) {
	switch ano {
	case 2026:
		return Aliquotas{IBSUF: decimal.RequireFromString("0.1"), IBSMun: decimal.Zero, CBS: decimal.RequireFromString("0.9")}, nil
	}
	return Aliquotas{}, fmt.Errorf("alíquotas de IBS/CBS de %d não cadastradas: atualize o pacote fiscal/reforma", ano)
}

// Grupo é o IBS/CBS calculado de um item.
type Grupo struct {
	CST, ClassTrib          string
	Base                    decimal.Decimal
	Aliq                    Aliquotas
	IBSUF, IBSMun, IBS, CBS decimal.Decimal
}

// CSTPadrao e ClassTribPadrao: tributação integral (CST 000, cClassTrib 000001),
// o caso geral da venda e da devolução de mercadoria.
const (
	CSTPadrao       = "000"
	ClassTribPadrao = "000001"
)

// Calcular monta o grupo do item. A base é o valor da operação (item + frete +
// seguro + outras − desconto) sem o ICMS, o PIS e a COFINS, que não a integram
// na transição.
func Calcular(ano int, valorOperacao, icms, pis, cofins decimal.Decimal) (Grupo, error) {
	a, err := AliquotasDoAno(ano)
	if err != nil {
		return Grupo{}, err
	}
	base := valorOperacao.Sub(icms).Sub(pis).Sub(cofins)
	if base.IsNegative() {
		base = decimal.Zero
	}
	base = base.Round(2)
	cem := decimal.NewFromInt(100)
	g := Grupo{CST: CSTPadrao, ClassTrib: ClassTribPadrao, Base: base, Aliq: a,
		IBSUF: base.Mul(a.IBSUF).Div(cem).Round(2), IBSMun: base.Mul(a.IBSMun).Div(cem).Round(2),
		CBS: base.Mul(a.CBS).Div(cem).Round(2)}
	g.IBS = g.IBSUF.Add(g.IBSMun)
	return g, nil
}
