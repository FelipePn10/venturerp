package impressao_uc

import (
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

var cem = decimal.NewFromInt(100)

// TotaisPedido soma o pedido como a nota vai somar: produtos, IPI, ST, frete e
// seguro, menos o desconto da capa.
type TotaisPedido struct {
	Produtos, IPI, ST, ICMS, Total decimal.Decimal
	PesoLiquido, PesoBruto         decimal.Decimal
	Quantidade                     decimal.Decimal
}

func (d PedidoVenda) Totais() TotaisPedido {
	var t TotaisPedido
	for _, it := range d.Itens {
		t.Produtos = t.Produtos.Add(it.Total)
		t.IPI = t.IPI.Add(it.IPI)
		t.ST = t.ST.Add(it.ST)
		t.ICMS = t.ICMS.Add(it.Total.Mul(it.ICMSPct).Div(cem))
		t.PesoLiquido = t.PesoLiquido.Add(it.PesoLiquido.Mul(it.Quantidade))
		t.PesoBruto = t.PesoBruto.Add(it.PesoBruto.Mul(it.Quantidade))
		t.Quantidade = t.Quantidade.Add(it.Quantidade)
	}
	t.Total = t.Produtos.Add(t.IPI).Add(t.ST).Add(d.Frete).Add(d.Seguro).Sub(d.Desconto)
	return t
}

// parcelasDoPedido aplica a condição de pagamento (dias e percentual de cada
// parcela, vindos da consulta) sobre o total do pedido a partir da emissão.
// A última parcela absorve o arredondamento, como no faturamento.
func parcelasDoPedido(d *PedidoVenda) []Parcela {
	if len(d.Parcelas) == 0 {
		return nil
	}
	ps := append([]Parcela(nil), d.Parcelas...)
	sort.SliceStable(ps, func(i, j int) bool { return ps[i].Numero < ps[j].Numero })
	total := d.Totais().Total
	somaPct := decimal.Zero
	for _, p := range ps {
		somaPct = somaPct.Add(p.Percentual)
	}
	if !somaPct.IsPositive() {
		somaPct = decimal.NewFromInt(int64(len(ps)))
		for i := range ps {
			ps[i].Percentual = decimal.NewFromInt(1)
		}
	}
	acumulado := decimal.Zero
	for i := range ps {
		ps[i].Vencimento = d.Emissao.AddDate(0, 0, ps[i].Dias)
		if i == len(ps)-1 {
			ps[i].Valor = total.Sub(acumulado)
		} else {
			ps[i].Valor = total.Mul(ps[i].Percentual).Div(somaPct).Round(2)
			acumulado = acumulado.Add(ps[i].Valor)
		}
		ps[i].Percentual = ps[i].Percentual.Mul(cem).Div(somaPct).Round(2)
	}
	return ps
}

// Atrasado diz se o título passou do vencimento sem baixa na data de
// referência.
func (t Titulo) Atrasado(ref time.Time) bool {
	return t.Pagamento == nil && t.Saldo().IsPositive() && dia(t.Vencimento).Before(dia(ref))
}

// Saldo é o que falta pagar ou receber.
func (t Titulo) Saldo() decimal.Decimal {
	s := t.Valor.Sub(t.Desconto).Add(t.Juros).Add(t.Multa).Sub(t.Pago)
	if s.IsNegative() {
		return decimal.Zero
	}
	return s
}

// DiasParaVencer é negativo quando vencido.
func (t Titulo) DiasParaVencer(ref time.Time) int {
	return int(dia(t.Vencimento).Sub(dia(ref)).Hours() / 24)
}

func dia(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC) }
