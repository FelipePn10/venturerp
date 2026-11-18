package service

import (
	"context"
	"testing"

	mrprepo "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
	"github.com/shopspring/decimal"
)

func pct(v int64) *decimal.Decimal { d := decimal.NewFromInt(v); return &d }

func TestEscolherFonteSemCotaUsaAOrdemPreferencialRanking(t *testing.T) {
	fontes := []FonteDeCompra{
		{SupplierCode: 10, Preferred: true, Ranking: 1},
		{SupplierCode: 20, Ranking: 2},
	}
	if i := escolherFonte(fontes, ""); i != 0 {
		t.Fatalf("escolheu %d, queria o preferencial", i)
	}
}

func TestEscolherFonteRespeitaAMascara(t *testing.T) {
	fontes := []FonteDeCompra{
		{SupplierCode: 10, Mask: "AZUL", Ranking: 1},
		{SupplierCode: 20, Mask: "", Ranking: 2},
	}
	if i := escolherFonte(fontes, "VERDE"); i != 1 {
		t.Fatalf("máscara VERDE escolheu %d, queria o vínculo geral", i)
	}
	if i := escolherFonte(fontes, "AZUL"); i != 0 {
		t.Fatalf("máscara AZUL escolheu %d", i)
	}
	if i := escolherFonte([]FonteDeCompra{{SupplierCode: 10, Mask: "AZUL"}}, "VERDE"); i != -1 {
		t.Fatalf("sem fonte para a máscara deveria dar -1, deu %d", i)
	}
}

// 70/30: em dez necessidades iguais, sete vão para A e três para B, sem
// dividir nenhuma — o índice de cota do SAP.
func TestEscolherFonteRepartePelaCotaSemPicotar(t *testing.T) {
	fontes := []FonteDeCompra{
		{SupplierCode: 1, Ranking: 1, QuotaPercent: pct(70)},
		{SupplierCode: 2, Ranking: 2, QuotaPercent: pct(30)},
		{SupplierCode: 3, Ranking: 3}, // sem cota: fora da divisão
	}
	contagem := map[int64]int{}
	for n := 0; n < 10; n++ {
		i := escolherFonte(fontes, "")
		if i < 0 {
			t.Fatal("sem fonte")
		}
		fontes[i].Alocado = fontes[i].Alocado.Add(decimal.NewFromInt(100))
		contagem[fontes[i].SupplierCode]++
	}
	if contagem[1] != 7 || contagem[2] != 3 || contagem[3] != 0 {
		t.Fatalf("repartição %v, queria 7/3/0", contagem)
	}
}

func TestEscolherFonteCotaConsideraOJaPedido(t *testing.T) {
	fontes := []FonteDeCompra{
		{SupplierCode: 1, Ranking: 1, QuotaPercent: pct(50), Alocado: decimal.NewFromInt(900)},
		{SupplierCode: 2, Ranking: 2, QuotaPercent: pct(50), Alocado: decimal.NewFromInt(100)},
	}
	if i := escolherFonte(fontes, ""); i != 1 {
		t.Fatalf("escolheu %d; o fornecedor 2 está atrás na cota", i)
	}
}

type fontesFixas struct {
	mrprepo.MRPCalculationRepository
	fontes   []FonteDeCompra
	leituras int
}

func (f *fontesFixas) FontesDeCompra(context.Context, int64) ([]FonteDeCompra, error) {
	f.leituras++
	return append([]FonteDeCompra(nil), f.fontes...), nil
}

// As fontes são lidas uma vez por cálculo e o alocado de cada sugestão conta
// para a próxima: é o que reparte as necessidades pela cota dentro do cálculo.
func TestFonteDaCompraLeUmaVezEAcumulaOAlocado(t *testing.T) {
	repo := &fontesFixas{fontes: []FonteDeCompra{
		{SupplierCode: 1, Ranking: 1, QuotaPercent: pct(50), MinOrderQty: decimal.NewFromInt(100)},
		{SupplierCode: 2, Ranking: 2, QuotaPercent: pct(50), OrderMultiple: decimal.NewFromInt(12)},
	}}
	s := &MRPServiceImpl{MRPRepo: repo}
	cache := &cachedItemMRP{}
	var fornecedores []int64
	for n := 0; n < 4; n++ {
		f, err := s.fonteDaCompra(context.Background(), cache, 9, "")
		if err != nil || f == nil {
			t.Fatalf("fonte: %v %v", f, err)
		}
		q := aplicaPoliticaDeLote(decimal.NewFromInt(30), f.MinOrderQty, f.OrderMultiple)
		f.Alocado = f.Alocado.Add(q)
		fornecedores = append(fornecedores, f.SupplierCode)
	}
	if repo.leituras != 1 {
		t.Fatalf("leu as fontes %d vezes", repo.leituras)
	}
	// 1 recebe 100 (mínimo); 2 recebe 36 (múltiplo de 12) até alcançar.
	if got := fornecedores; got[0] != 1 || got[1] != 2 || got[2] != 2 || got[3] != 2 {
		t.Fatalf("sequência %v, queria [1 2 2 2]", got)
	}
}

func TestPrazoDaFonteUsaOFornecedorEscolhidoOuOPrazoDoItem(t *testing.T) {
	dez := 10
	if got := prazoDaFonte(&FonteDeCompra{LeadTimeDays: 25}, &dez); got != 25 {
		t.Fatalf("com prazo próprio vale o do fornecedor: %d", got)
	}
	// Cota escolheu um fornecedor sem prazo: vale o do item (10), não o do
	// preferencial.
	if got := prazoDaFonte(&FonteDeCompra{}, &dez); got != 10 {
		t.Fatalf("sem prazo próprio vale o prazo de compra do item: %d", got)
	}
	if got := prazoDaFonte(&FonteDeCompra{}, nil); got != 0 {
		t.Fatalf("sem prazo nenhum: %d", got)
	}
}
