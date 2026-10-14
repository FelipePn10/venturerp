package service

import (
	"context"
	"testing"
	"time"

	inddemandentity "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/ports"
)

func ptrI(v int64) *int64 { return &v }

// O pegging segue a ordem do netting: estoque, firme que chega até a data,
// sugestão; o que sobra é falta. A sobra de lote da sugestão cobre a
// necessidade seguinte.
func TestPeggingNaOrdemDoNetting(t *testing.T) {
	b := newExceptionBasis(
		map[int64]*entity.StockSnapshot{1: {Quantity: 10}},
		map[int64][]ports.SupplyEntry{1: {firme(500, 8, diaOut(20))}},
	)
	b.addRequirement(1, diaOut(5), 15, entity.PegDemandSalesOrder, ptrI(71))
	b.addRequirement(1, diaOut(20), 12, entity.PegDemandSuggestion, ptrI(900))
	b.addRequirement(1, diaOut(25), 6, entity.PegDemandIndependent, ptrI(3))
	// Sugestão do dia 5 com lote mínimo 10 (precisava 5): sobra 5 para depois.
	b.addSuggestion(&entity.PlannedOrderSuggestion{Code: 77, ItemCode: 1, Quantity: 10, NeedDate: diaOut(5), OrderType: "COMPRA"})
	b.addSuggestion(&entity.PlannedOrderSuggestion{Code: 78, ItemCode: 1, Quantity: 3, NeedDate: diaOut(25), OrderType: "COMPRA"})
	// Ordem de serviço não é suprimento.
	b.addSuggestion(&entity.PlannedOrderSuggestion{Code: 79, ItemCode: 1, Quantity: 99, NeedDate: diaOut(5), OrderType: "SERVICO"})

	type linha struct {
		demanda string
		cod     int64
		supply  string
		scod    int64
		qtd     float64
	}
	var got []linha
	for _, r := range b.pegging() {
		l := linha{demanda: r.DemandKind, supply: r.SupplyKind, qtd: r.Quantity}
		if r.DemandCode != nil {
			l.cod = *r.DemandCode
		}
		if r.SupplyCode != nil {
			l.scod = *r.SupplyCode
		}
		got = append(got, l)
	}
	want := []linha{
		{entity.PegDemandSalesOrder, 71, entity.PegSupplyStock, 0, 10},
		{entity.PegDemandSalesOrder, 71, entity.PegSupplySuggestion, 77, 5},
		{entity.PegDemandSuggestion, 900, entity.PegSupplyFirmOrder, 500, 8},
		{entity.PegDemandSuggestion, 900, entity.PegSupplySuggestion, 77, 4},
		{entity.PegDemandIndependent, 3, entity.PegSupplySuggestion, 77, 1},
		{entity.PegDemandIndependent, 3, entity.PegSupplySuggestion, 78, 3},
		{entity.PegDemandIndependent, 3, entity.PegSupplyShortage, 0, 2},
	}
	if len(got) != len(want) {
		t.Fatalf("pegging = %+v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("linha %d = %+v, esperado %+v (tudo: %+v)", i, got[i], want[i], got)
		}
	}
}

// Ordem firme que chega depois da necessidade não a cobre (o MRP sugere
// outra); ela fica para a necessidade seguinte.
func TestPeggingFirmeAtrasadaNaoCobre(t *testing.T) {
	b := newExceptionBasis(nil, map[int64][]ports.SupplyEntry{2: {firme(600, 5, diaOut(15))}})
	b.addRequirement(2, diaOut(10), 5, entity.PegDemandForecast, nil)
	b.addRequirement(2, diaOut(20), 5, entity.PegDemandForecast, nil)
	b.addSuggestion(&entity.PlannedOrderSuggestion{Code: 1, ItemCode: 2, Quantity: 5, NeedDate: diaOut(10), OrderType: "FABRICACAO"})
	r := b.pegging()
	if len(r) != 2 || r[0].SupplyKind != entity.PegSupplySuggestion || r[1].SupplyKind != entity.PegSupplyFirmOrder || !r[1].NeedDate.Equal(diaOut(20)) {
		t.Fatalf("pegging = %+v", r)
	}
}

func TestOrigemDaNecessidade(t *testing.T) {
	casos := []struct {
		in   entity.MRPInput
		kind string
	}{
		{entity.MRPInput{DemandType: "SALES_ORDER", SourceCode: ptrI(5)}, entity.PegDemandSalesOrder},
		{entity.MRPInput{DemandType: "INDEPENDENT_DEMAND", SourceCode: ptrI(5)}, entity.PegDemandIndependent},
		{entity.MRPInput{DemandType: "FORECAST"}, entity.PegDemandForecast},
		{entity.MRPInput{PegKind: entity.PegDemandSuggestion, PegCode: ptrI(9)}, entity.PegDemandSuggestion},
		{entity.MRPInput{}, entity.PegDemandOther},
	}
	for _, c := range casos {
		if k, _ := pegOrigin(&c.in); k != c.kind {
			t.Fatalf("%+v → %s, esperado %s", c.in, k, c.kind)
		}
	}
}

// Ajustes do cenário: altera e retira demanda e pedido, cria demanda nova,
// soma estoque e troca o prazo — sem alterar os registros lidos do banco.
func TestAjustesDoCenario(t *testing.T) {
	q := func(v float64) *float64 { return &v }
	d := func(v time.Time) *time.Time { return &v }
	adj := &entity.ScenarioAdjustments{
		Demands: []entity.ScenarioDemandChange{
			{ID: 1, RefKind: entity.ScenarioDemandIndependent, RefCode: ptrI(10), Quantity: q(50), NeedDate: d(diaOut(9))},
			{ID: 2, RefKind: entity.ScenarioDemandIndependent, RefCode: ptrI(11), Quantity: q(0)},
			{ID: 3, RefKind: entity.ScenarioDemandSalesOrder, RefCode: ptrI(70), Quantity: q(40)},
			{ID: 4, RefKind: entity.ScenarioDemandNew, ItemCode: 8, Quantity: q(12), NeedDate: d(diaOut(30))},
		},
		StockDelta:   map[int64]float64{5: 30, 6: -100},
		LeadTimeDays: map[int64]int{5: 12},
	}
	original := &inddemandentity.IndependentDemand{CodeDemand: 10, ItemCode: 1, Quantity: 5, DemandDate: diaOut(1)}
	demands := []*inddemandentity.IndependentDemand{original, {CodeDemand: 11, ItemCode: 2, Quantity: 7}, {CodeDemand: 12, ItemCode: 3, Quantity: 1}}
	pedido := &entity.MRPInput{ItemCode: 4, Quantity: 9, DemandType: "SALES_ORDER", SourceCode: ptrI(70)}
	inputs := []*entity.MRPInput{pedido, {ItemCode: 4, Quantity: 1, DemandType: "SALES_ORDER", SourceCode: ptrI(71)}}

	gotD, gotI := applyScenarioDemands(adj, demands, inputs, 99)
	if len(gotD) != 2 || gotD[0].Quantity != 50 || !gotD[0].DemandDate.Equal(diaOut(9)) || gotD[1].CodeDemand != 12 {
		t.Fatalf("demandas: %+v", gotD)
	}
	if original.Quantity != 5 || pedido.Quantity != 9 {
		t.Fatal("o ajuste alterou o registro lido do banco")
	}
	if len(gotI) != 3 || gotI[0].Quantity != 40 || gotI[1].Quantity != 1 || gotI[2].ItemCode != 8 || gotI[2].PegKind != entity.PegDemandSimulated || *gotI[2].PegCode != 4 || gotI[2].PlanCode != 99 {
		t.Fatalf("entradas: %+v %+v", gotI, gotI[2])
	}

	snaps := map[int64]*entity.StockSnapshot{6: {ItemCode: 6, Quantity: 40}}
	applyScenarioStock(adj, snaps)
	if snaps[5].Quantity != 30 || snaps[6].Quantity != 0 {
		t.Fatalf("estoque: %+v %+v", snaps[5], snaps[6])
	}
	regras := map[int64][]*entity.ConfiguredItemRule{5: {{FieldName: "lead_time", RuleValue: "3"}}}
	applyScenarioLeadTimes(adj, regras)
	s := &MRPServiceImpl{}
	if got, err := s.getLeadTimeDays(context.Background(), regras, 5); err != nil || got != 12 {
		t.Fatalf("prazo do cenário = %d, esperado 12", got)
	}
}
