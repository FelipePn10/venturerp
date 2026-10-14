package mrp_calculation_uc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
)

type peggingFalso struct{ rows []entity.PeggingRecord }

func (p peggingFalso) PeggingRows(context.Context, int64) ([]entity.PeggingRecord, error) {
	return p.rows, nil
}
func (p peggingFalso) PeggingLabels(_ context.Context, refs entity.PeggingRefs) (*entity.PeggingLabels, error) {
	n := int64(5001)
	return &entity.PeggingLabels{
		Suggestions: map[int64]entity.PeggingOrderLabel{
			10: {OrderNumber: &n, OrderType: "FABRICACAO", Quantity: 20},
			11: {OrderType: "COMPRA", Quantity: 60},
		},
		FirmOrders: map[int64]entity.PeggingOrderLabel{}, SalesOrderItems: map[int64]entity.PeggingSalesLabel{},
		Items: map[int64]entity.PeggingItemLabel{1: {Description: "CARRINHO"}, 2: {Description: "TUBO"}},
	}, nil
}

func i64(v int64) *int64 { return &v }

// Sugestão de 20 carrinhos atende dois pedidos (15 e 5): os 60 tubos que ela
// pediu se repartem 45/15 na cadeia de cada pedido; a falta de um componente
// aparece no resumo do pedido.
func TestPeggingCadeiaProporcionalEFalta(t *testing.T) {
	d := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	store := peggingFalso{rows: []entity.PeggingRecord{
		{ItemCode: 1, DemandKind: entity.PegDemandSalesOrder, DemandCode: i64(71), NeedDate: d, SupplyKind: entity.PegSupplySuggestion, SupplyCode: i64(10), Quantity: 15},
		{ItemCode: 1, DemandKind: entity.PegDemandSalesOrder, DemandCode: i64(72), NeedDate: d, SupplyKind: entity.PegSupplySuggestion, SupplyCode: i64(10), Quantity: 5},
		{ItemCode: 2, DemandKind: entity.PegDemandSuggestion, DemandCode: i64(10), NeedDate: d, SupplyKind: entity.PegSupplyStock, Quantity: 20},
		{ItemCode: 2, DemandKind: entity.PegDemandSuggestion, DemandCode: i64(10), NeedDate: d, SupplyKind: entity.PegSupplySuggestion, SupplyCode: i64(11), Quantity: 30},
		{ItemCode: 2, DemandKind: entity.PegDemandSuggestion, DemandCode: i64(10), NeedDate: d, SupplyKind: entity.PegSupplyShortage, Quantity: 10},
	}}
	uc := &PeggingUseCase{Store: store, Auth: profileAuth{}}

	arvore, err := uc.Down(context.Background(), 1, "PEDIDO_VENDA", i64(71), nil, nil)
	if err != nil || len(arvore) != 1 || len(arvore[0].Children) != 3 {
		t.Fatalf("cadeia do pedido 71: %+v (%v)", arvore, err)
	}
	if arvore[0].Supply.Label != "Sugestão 5001 · fabricação" {
		t.Fatalf("rótulo: %q", arvore[0].Supply.Label)
	}
	total := 0.0
	for _, c := range arvore[0].Children {
		total += c.Quantity
	}
	if total != 45 {
		t.Fatalf("componentes do pedido 71 = %v, esperado 45 (15/20 de 60)", total)
	}

	acima, err := uc.Up(context.Background(), 1, "SUGESTAO", 11)
	if err != nil || len(acima) != 1 || len(acima[0].Children) != 2 {
		t.Fatalf("de quem depende a compra de tubo: %+v (%v)", acima, err)
	}

	raizes, err := uc.Roots(context.Background(), 1)
	if err != nil || len(raizes) != 2 || !raizes[0].ChainShortage || raizes[0].FromSuggestions != 15 {
		t.Fatalf("raízes: %+v (%v)", raizes, err)
	}
}

type peggingAtraso struct {
	rows   []entity.PeggingRecord
	labels *entity.PeggingLabels
}

func (p peggingAtraso) PeggingRows(context.Context, int64) ([]entity.PeggingRecord, error) {
	return p.rows, nil
}
func (p peggingAtraso) PeggingLabels(context.Context, entity.PeggingRefs) (*entity.PeggingLabels, error) {
	return p.labels, nil
}
func (p peggingAtraso) SalesOrderPeggingPlans(context.Context, int64) ([]SalesOrderPlan, error) {
	return []SalesOrderPlan{{PlanCode: 1, PlanName: "Plano", ItemCodes: []int64{71}}}, nil
}

// Elo atrasado: a fabricação programada termina depois da necessidade, ou a
// ordem já devia ter começado. O atraso sobe até o pedido; a ordem firme da
// exceção diz quais pedidos dependem dela; o pedido de venda vê a cadeia.
func TestPeggingEloAtrasadoEPedidosAfetados(t *testing.T) {
	d := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	fimTarde := time.Date(2026, 10, 25, 15, 0, 0, 0, time.UTC)
	diaFim := time.Date(2026, 10, 25, 0, 0, 0, 0, time.UTC)
	inicioPassado := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	n := int64(9001)
	store := peggingAtraso{
		rows: []entity.PeggingRecord{
			{ItemCode: 1, DemandKind: entity.PegDemandSalesOrder, DemandCode: i64(71), NeedDate: d, SupplyKind: entity.PegSupplySuggestion, SupplyCode: i64(10), Quantity: 5},
			{ItemCode: 2, DemandKind: entity.PegDemandSuggestion, DemandCode: i64(10), NeedDate: d, SupplyKind: entity.PegSupplySuggestion, SupplyCode: i64(11), Quantity: 5},
			{ItemCode: 3, DemandKind: entity.PegDemandSuggestion, DemandCode: i64(11), NeedDate: d, SupplyKind: entity.PegSupplyFirmOrder, SupplyCode: i64(500), Quantity: 5},
			{ItemCode: 1, DemandKind: entity.PegDemandIndependent, DemandCode: i64(3), NeedDate: d, SupplyKind: entity.PegSupplySuggestion, SupplyCode: i64(12), Quantity: 5},
		},
		labels: &entity.PeggingLabels{
			Suggestions: map[int64]entity.PeggingOrderLabel{
				10: {OrderNumber: &n, OrderType: "FABRICACAO", Quantity: 5},
				11: {OrderType: "FABRICACAO", Quantity: 5, EstimatedEnd: &fimTarde, EstimatedEndDay: &diaFim},
				12: {OrderType: "FABRICACAO", Quantity: 5, StartDate: &inicioPassado},
			},
			FirmOrders: map[int64]entity.PeggingOrderLabel{}, SalesOrderItems: map[int64]entity.PeggingSalesLabel{71: {OrderNumber: i64(5001), Customer: "Cliente"}},
			Items: map[int64]entity.PeggingItemLabel{2: {Description: "CHASSI"}},
		},
	}
	uc := &PeggingUseCase{Store: store, Auth: profileAuth{}, Now: func() time.Time { return time.Date(2026, 10, 13, 10, 0, 0, 0, time.UTC) }}

	raizes, err := uc.Roots(context.Background(), 1)
	if err != nil || len(raizes) != 2 {
		t.Fatalf("raízes: %+v (%v)", raizes, err)
	}
	for _, r := range raizes {
		if !r.ChainLate {
			t.Fatalf("raiz %s sem atraso: %+v", r.Demand.Label, r)
		}
	}
	for _, r := range raizes {
		if r.Demand.Kind != entity.PegDemandSalesOrder {
			continue
		}
		if !strings.Contains(r.LateReason, "CHASSI") || !strings.Contains(r.LateReason, "25/10/2026") {
			t.Fatalf("motivo do atraso não cita o elo e o fim: %q", r.LateReason)
		}
		// "Pronto em" é quando a cadeia fica pronta: o fim na máquina, não a necessidade.
		if r.LatestSupply == nil || !r.LatestSupply.Equal(diaFim) {
			t.Fatalf("pronto em = %v, esperado %v", r.LatestSupply, diaFim)
		}
	}

	afetados, err := uc.AffectedBySupply(context.Background(), 1, entity.PegSupplyFirmOrder, []int64{500})
	if err != nil || len(afetados[500]) != 1 || afetados[500][0] != "Pedido 5001 · Cliente" {
		t.Fatalf("pedidos afetados pela ordem 500: %+v (%v)", afetados, err)
	}

	cadeia, err := uc.SalesOrder(context.Background(), 900)
	if err != nil || len(cadeia) != 1 || len(cadeia[0].Roots) != 1 || len(cadeia[0].Trees[0]) != 1 {
		t.Fatalf("cadeia do pedido: %+v (%v)", cadeia, err)
	}
}
