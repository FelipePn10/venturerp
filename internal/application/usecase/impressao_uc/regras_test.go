package impressao_uc

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestParcelasDoPedidoFechamNoTotalComIPI(t *testing.T) {
	d := &PedidoVenda{
		Emissao: time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC),
		Itens: []ItemPedidoVenda{{
			Quantidade: decimal.NewFromInt(20), Total: decimal.RequireFromString("7210"), IPI: decimal.RequireFromString("234.33"),
		}},
		Parcelas: []Parcela{{Numero: 2, Dias: 60}, {Numero: 1, Dias: 30}, {Numero: 3, Dias: 90}},
	}
	ps := parcelasDoPedido(d)
	if len(ps) != 3 {
		t.Fatalf("parcelas = %d", len(ps))
	}
	if ps[0].Numero != 1 || !ps[0].Vencimento.Equal(time.Date(2026, 12, 11, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("primeira parcela = %+v", ps[0])
	}
	soma := decimal.Zero
	for _, p := range ps {
		soma = soma.Add(p.Valor)
	}
	if !soma.Equal(decimal.RequireFromString("7444.33")) {
		t.Errorf("soma das parcelas = %s, quer 7444.33", soma)
	}
	if !ps[0].Valor.Equal(decimal.RequireFromString("2481.44")) || !ps[2].Valor.Equal(decimal.RequireFromString("2481.45")) {
		t.Errorf("valores = %s %s %s", ps[0].Valor, ps[1].Valor, ps[2].Valor)
	}
}

func TestTituloAtrasadoESaldo(t *testing.T) {
	ref := time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)
	tt := Titulo{Valor: decimal.NewFromInt(100), Pago: decimal.NewFromInt(40), Vencimento: time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)}
	if !tt.Saldo().Equal(decimal.NewFromInt(60)) || !tt.Atrasado(ref) || tt.DiasParaVencer(ref) != -1 {
		t.Errorf("saldo %s atrasado %v dias %d", tt.Saldo(), tt.Atrasado(ref), tt.DiasParaVencer(ref))
	}
}

// O cabeçalho do PDF descreve os filtros como a tela os mostra: valor em reais
// e plano de contas pelo código e nome, nunca pelo número interno.
func TestDescricaoDoFiltroDosTitulos(t *testing.T) {
	min, max := decimal.RequireFromString("1500"), decimal.RequireFromString("1234567.8")
	plano := int64(42)
	got := descreverFiltro(FiltroTitulos{Documento: "NF-10", PlanoContasID: &plano, ValorMinimo: &min, ValorMaximo: &max, SomenteVencidos: true},
		"2.1.01 Matéria-prima", "")
	quer := "documento contendo NF-10 · plano de contas 2.1.01 Matéria-prima · valor de 1.500,00 a 1.234.567,80 · somente vencidos"
	if got != quer {
		t.Fatalf("descrição:\n got %q\nquer %q", got, quer)
	}
	if got := reais(&max); got != "1.234.567,80" {
		t.Fatal(got)
	}
	neg := decimal.RequireFromString("-999.5")
	if got := reais(&neg); got != "-999,50" {
		t.Fatal(got)
	}
	centro := int64(7)
	if got := descreverFiltro(FiltroTitulos{CentroCustoID: &centro}, "", ""); got != "centro de custo não cadastrado" {
		t.Fatal(got)
	}
}
