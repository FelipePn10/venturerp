package service

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	inddemandentity "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
)

type vendasFixas []inddemandentity.Venda

func (v vendasFixas) Vendas(context.Context, []int64) ([]inddemandentity.Venda, error) { return v, nil }

func TestConsumoPelaVendaNoMRP(t *testing.T) {
	hoje := today()
	params := entity.DefaultTypedPlanningParams()
	// Demanda de estoque de 10 e previsão de 20 do item 7, ambas de hoje;
	// pedidos de 12 (hoje) consomem a demanda inteira e 2 da previsão.
	demands := []*inddemandentity.IndependentDemand{
		{CodeDemand: 1, ItemCode: 7, Quantity: decimal.NewFromInt(10), DemandDate: hoje, Motivo: inddemandentity.MotivoEstoque, Situacao: inddemandentity.SituacaoAberta},
		{CodeDemand: 2, ItemCode: 7, Quantity: decimal.NewFromInt(3), DemandDate: hoje, Motivo: inddemandentity.MotivoFeira, Situacao: inddemandentity.SituacaoAberta, Atendida: decimal.NewFromInt(1)},
	}
	previsao := []*entity.MRPInput{{ItemCode: 7, Quantity: decimal.NewFromInt(20), NeedDate: hoje, DemandType: "FORECAST"}}
	s := &MRPServiceImpl{Vendas: vendasFixas{{Chave: inddemandentity.ChaveItem(7, ""), Data: hoje, Quantidade: decimal.NewFromInt(12)}}}
	d, p, err := s.consumirPelaVenda(context.Background(), params, demands, previsao, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 1 || d[0].CodeDemand != 2 || !d[0].Quantity.Equal(decimal.NewFromInt(2)) {
		t.Fatalf("a de estoque foi toda consumida; a de feira planeja o saldo não atendido: %+v", d)
	}
	if len(p) != 1 || !p[0].Quantity.Equal(decimal.NewFromInt(18)) {
		t.Fatalf("previsão consumida em 2: %+v", p)
	}
	if !demands[0].Quantity.Equal(decimal.NewFromInt(10)) {
		t.Fatal("o consumo não altera a demanda carregada")
	}
}

func TestSemSemanasPassadas(t *testing.T) {
	hoje := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC) // quarta
	prev := []*entity.MRPInput{
		{ItemCode: 1, NeedDate: time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)}, // semana passada
		{ItemCode: 2, NeedDate: time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)}, // semana atual
		{ItemCode: 3, NeedDate: time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)},
	}
	got := semSemanasPassadas(prev, hoje)
	if len(got) != 2 || got[0].ItemCode != 2 || got[1].ItemCode != 3 {
		t.Fatalf("só a semana passada sai: %+v", got)
	}
}

// O cenário entra antes do consumo: a demanda alterada no cenário é consumida
// pela quantidade nova, e o pedido alterado consome pela quantidade nova.
func TestCenarioAntesDoConsumo(t *testing.T) {
	hoje := today()
	params := entity.DefaultTypedPlanningParams()
	linha := int64(55)
	demanda := []*inddemandentity.IndependentDemand{
		{CodeDemand: 1, ItemCode: 7, Quantity: decimal.NewFromInt(10), DemandDate: hoje, Motivo: inddemandentity.MotivoEstoque, Situacao: inddemandentity.SituacaoAberta},
	}
	// Pedido lido do banco: 4 abertos (de 6 vendidos: 2 já atendidos).
	pedidos := []*entity.MRPInput{{ItemCode: 7, Quantity: decimal.NewFromInt(4), NeedDate: hoje, DemandType: "SALES_ORDER", SourceCode: &linha}}
	um := int64(1)
	q30, q9 := decimal.NewFromInt(30), decimal.NewFromInt(9)
	ajustes := &entity.ScenarioAdjustments{Demands: []entity.ScenarioDemandChange{
		{RefKind: entity.ScenarioDemandIndependent, RefCode: &um, Quantity: &q30},
		{RefKind: entity.ScenarioDemandSalesOrder, RefCode: &linha, Quantity: &q9},
	}}
	demanda, _ = applyScenarioDemands(ajustes, demanda, pedidos, 1)
	s := &MRPServiceImpl{Vendas: vendasFixas{{Chave: inddemandentity.ChaveItem(7, ""), Data: hoje, Quantidade: decimal.NewFromInt(6), Linha: &linha}}}
	d, _, err := s.consumirPelaVenda(context.Background(), params, demanda, nil, ajustes, pedidos)
	if err != nil {
		t.Fatal(err)
	}
	// Venda no cenário: 2 atendidos + 9 abertos = 11; demanda 30 − 11 = 19.
	if len(d) != 1 || !d[0].Quantity.Equal(decimal.NewFromInt(19)) {
		t.Fatalf("demanda do cenário (30) consumida pela venda do cenário (11): %+v", d)
	}
}

func TestAjustarVendasDoCenarioSemAjusteNaoMuda(t *testing.T) {
	linha := int64(3)
	v := []inddemandentity.Venda{{Chave: "7|", Quantidade: decimal.NewFromInt(5), Linha: &linha}}
	if got := ajustarVendasDoCenario(nil, nil, v); !got[0].Quantidade.Equal(decimal.NewFromInt(5)) {
		t.Fatalf("sem cenário a venda fica: %+v", got)
	}
}
