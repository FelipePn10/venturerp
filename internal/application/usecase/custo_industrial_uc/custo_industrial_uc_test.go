package custo_industrial_uc

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

type repoFixo struct {
	linhas []LinhaDoPedidoCusto
	ofs    []VariacaoDaOF
}

func (r repoFixo) Variacoes(context.Context, time.Time, time.Time) ([]VariacaoDaOF, error) {
	return r.ofs, nil
}
func (r repoFixo) LinhasDoPedido(context.Context, int64) ([]LinhaDoPedidoCusto, error) {
	return r.linhas, nil
}
func (repoFixo) ItensParaApurar(context.Context) ([]int64, error) { return nil, nil }

func TestMargemDoPedido(t *testing.T) {
	d := decimal.RequireFromString
	// Exemplo da rodada de testes: venda R$ 360,50 × custo R$ 230,22 (1 peça, faturada).
	uc := &UseCase{Repo: repoFixo{linhas: []LinhaDoPedidoCusto{
		{Linha: 1, ItemCode: 601, Quantidade: d("1"), Receita: d("360.50"), CustoPadraoUnit: d("230.22"), TemPadrao: true, FaturadoQtd: d("1"), CustoReal: d("240")},
	}}}
	m, err := uc.Margem(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if !m.MargemPadrao.Equal(d("130.28")) || !m.PctPadrao.Equal(d("36.14")) {
		t.Fatalf("margem padrão %s (%s%%), quer 130,28 (36,14%%)", m.MargemPadrao, m.PctPadrao)
	}
	if !m.MargemReal.Equal(d("120.5")) || !m.PctReal.Equal(d("33.43")) {
		t.Fatalf("margem real %s (%s%%), quer 120,50 (33,43%%)", m.MargemReal, m.PctReal)
	}
}

func TestVariacoesSomaODesfavoravel(t *testing.T) {
	d := decimal.RequireFromString
	uc := &UseCase{Repo: repoFixo{ofs: []VariacaoDaOF{{Padrao: d("100"), Real: d("120"), Total: d("20")}, {Padrao: d("50"), Real: d("45"), Total: d("-5")}}}}
	v, err := uc.VariacoesDoPeriodo(context.Background(), time.Now().AddDate(0, -1, 0), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !v.Total.Equal(d("15")) || !v.Desfavoravel.Equal(d("20")) || !v.Percentual.Equal(d("10")) || !v.Ordens[0].Percentual.Equal(d("20")) {
		t.Fatalf("%+v", v)
	}
}
