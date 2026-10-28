package entity

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
)

// Os totais da linha saem em decimal, na escala da coluna (4 casas): o mesmo
// valor que o float gravava depois do arredondamento do banco, sem o resíduo.
func TestRecalcularTotaisDaLinha(t *testing.T) {
	d := decimal.RequireFromString
	casos := []struct {
		qtd, preco                   string
		desconto, ipi, st            float64
		bruto, ipiValor, comIPI, st2 string
	}{
		{"3", "10.05", 0, 0, 0, "30.15", "0", "30.15", "0"}, // float: 30.150000000000002
		{"0.1", "0.2", 0, 0, 0, "0.02", "0", "0.02", "0"},
		{"12.7", "48.9", 7.5, 5, 0, "574.4528", "28.7226", "603.1754", "0"}, // 48,9 × 0,925 × 12,7 = 574,45275
		{"4", "125", 0, 0, 18, "500", "0", "500", "90"},
	}
	for _, c := range casos {
		it := &SalesOrderItem{RequestedQty: d(c.qtd), UnitPrice: d(c.preco), DiscountPct: c.desconto, IPIPct: c.ipi, STPct: c.st}
		it.RecalcularTotais()
		if !it.TotalGross.Equal(d(c.bruto)) || !it.TotalIPI.Equal(d(c.ipiValor)) || !it.TotalNetWithIPI.Equal(d(c.comIPI)) || !it.TotalST.Equal(d(c.st2)) {
			t.Fatalf("%+v: bruto %s ipi %s c/IPI %s st %s", c, it.TotalGross, it.TotalIPI, it.TotalNetWithIPI, it.TotalST)
		}
		// Mesmo resultado da fórmula antiga em float depois das 4 casas do banco.
		q, p := d(c.qtd).InexactFloat64(), d(c.preco).InexactFloat64()
		antigo := math.Round(p*(1-c.desconto/100)*q*10000) / 10000
		if math.Abs(antigo-it.TotalGross.InexactFloat64()) > 0.00005 {
			t.Fatalf("%+v: decimal %s difere do float arredondado %v", c, it.TotalGross, antigo)
		}
	}
	it := &SalesOrderItem{RequestedQty: d("10"), AttendedQty: d("3.5"), CancelledQty: d("1.25"), UnitPrice: d("1")}
	it.RecalcularTotais()
	if !it.Balance.Equal(d("5.25")) {
		t.Fatalf("saldo %s", it.Balance)
	}
}
