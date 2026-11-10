package service

import (
	"context"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	mrpports "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/ports"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	routingrepo "github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
	structentity "github.com/FelipePn10/panossoerp/internal/domain/structure/entity"
	"github.com/shopspring/decimal"
)

// Armário: 10 corte (16 h) → 20 furação (8 h) → 30 montagem (4 h). A montagem
// começa em 24 h de 28 h — 6/7 do processo. Numa janela de 14 dias do início à
// entrega, a ferragem é necessária 12 dias depois do início (6/7 × 14).
type roteiroDoArmario struct {
	routingrepo.RoutingRepository
}

func (roteiroDoArmario) GetRouteForItem(context.Context, int64, string) (*routingentity.ManufacturingRoute, error) {
	return &routingentity.ManufacturingRoute{ID: 9}, nil
}

func (roteiroDoArmario) GetRouteOperations(context.Context, int64) ([]*routingentity.RouteOperation, error) {
	return []*routingentity.RouteOperation{
		{ID: 1, Sequence: 10, EffTime: routingentity.OperationTime{Run: 16, RunBaseQty: 1000}},
		{ID: 2, Sequence: 20, EffTime: routingentity.OperationTime{Run: 8, RunBaseQty: 1000}},
		{ID: 3, Sequence: 30, EffTime: routingentity.OperationTime{Run: 4, RunBaseQty: 1000}},
	}, nil
}

func (roteiroDoArmario) GetNetworkEdges(context.Context, int64) ([]*routingentity.NetworkEdge, error) {
	return nil, nil
}

func TestComponenteDaMontagemENecessarioQuandoAMontagemComeca(t *testing.T) {
	s := &MRPServiceImpl{RoutingRepo: roteiroDoArmario{}}
	inicio := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	entrega := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	sug := &entity.PlannedOrderSuggestion{StartDate: &inicio, NeedDate: entrega, Quantity: decimal.NewFromInt(1)}
	cache := map[string]map[int16]float64{}

	montagem := int16(30)
	got, err := s.necessidadeNaEtapa(context.Background(), cache, 50, "", sug, &montagem)
	if err != nil {
		t.Fatal(err)
	}
	if quer := inicio.AddDate(0, 0, 12); !got.Equal(quer) {
		t.Fatalf("ferragem da montagem: quer %s, veio %s", quer.Format("02/01"), got.Format("02/01"))
	}

	// Sem etapa: início da ordem, como sempre foi.
	if got, _ := s.necessidadeNaEtapa(context.Background(), cache, 50, "", sug, nil); !got.Equal(inicio) {
		t.Fatalf("sem etapa a necessidade é o início da ordem; veio %s", got.Format("02/01"))
	}
	// Etapa que não existe no roteiro: início da ordem (não some, não atrasa).
	inexistente := int16(99)
	if got, _ := s.necessidadeNaEtapa(context.Background(), cache, 50, "", sug, &inexistente); !got.Equal(inicio) {
		t.Fatalf("etapa inexistente cai no início da ordem; veio %s", got.Format("02/01"))
	}
	// Nunca depois da entrega do pai.
	apertada := &entity.PlannedOrderSuggestion{StartDate: &inicio, NeedDate: inicio.AddDate(0, 0, 1), Quantity: decimal.NewFromInt(1)}
	if got, _ := s.necessidadeNaEtapa(context.Background(), cache, 50, "", apertada, &montagem); got.After(apertada.NeedDate) {
		t.Fatalf("a necessidade do componente não passa da entrega do pai; veio %s", got.Format("02/01"))
	}
}

// A explosão da estrutura leva a etapa do componente até a necessidade do filho.
// (Foi o elo que faltou: a etapa estava gravada e o MRP a perdia na explosão.)
func TestExplosaoLevaAEtapaDoComponente(t *testing.T) {
	montagem := int16(20)
	bom := map[int64][]*structentity.ItemStructure{
		1: {
			{ParentCode: 1, ChildCode: 2, Quantity: decimal.NewFromInt(2), IsActive: true, RouteOperationSequence: &montagem},
			{ParentCode: 1, ChildCode: 3, Quantity: decimal.NewFromInt(1), IsActive: true},
		},
	}
	filhos := explodeFromBOMWithVars(bom, 1, "", decimal.NewFromInt(60), 1, 1, nil, time.Time{})
	porItem := map[int64]*entity.MRPInput{}
	for _, f := range filhos {
		porItem[f.ItemCode] = f
	}
	if porItem[2] == nil || porItem[2].ParentStepSequence == nil || *porItem[2].ParentStepSequence != 20 {
		t.Fatalf("o componente alocado à montagem tem de chegar com a etapa 20: %+v", porItem[2])
	}
	if porItem[3] == nil || porItem[3].ParentStepSequence != nil {
		t.Fatalf("componente sem etapa continua sem etapa: %+v", porItem[3])
	}
}

type empenhoFixo struct{ lista []mrpports.OrderAllocation }

func (empenhoFixo) ListFirmSupplyForItems(context.Context, []int64) (map[int64][]mrpports.SupplyEntry, error) {
	return nil, nil
}
func (e empenhoFixo) ListOpenOrderAllocations(context.Context) ([]mrpports.OrderAllocation, error) {
	return e.lista, nil
}

// O empenho da OF aberta vira demanda do componente; o que entra na montagem
// é necessário quando a montagem começa (6/7 do prazo da OF), não no início.
func TestEmpenhoDaOFAbertaEntraNaDataDaEtapa(t *testing.T) {
	inicio := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	montagem := int16(30)
	s := &MRPServiceImpl{RoutingRepo: roteiroDoArmario{}, SupplyPort: empenhoFixo{lista: []mrpports.OrderAllocation{
		{ProductionOrderID: 7, ParentItemCode: 50, ItemCode: 2, Quantity: decimal.NewFromInt(4), OrderStart: inicio, OrderEnd: inicio.AddDate(0, 0, 14), Step: &montagem, ParentQuantity: decimal.NewFromInt(1)},
		{ProductionOrderID: 7, ParentItemCode: 50, ItemCode: 3, Quantity: decimal.NewFromInt(1), OrderStart: inicio, OrderEnd: inicio.AddDate(0, 0, 14)},
		{ProductionOrderID: 7, ParentItemCode: 50, ItemCode: 4, Quantity: decimal.Zero, OrderStart: inicio, OrderEnd: inicio},
	}}}
	got, err := s.empenhoDasOrdensAbertas(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("empenho zerado não é demanda; vieram %d", len(got))
	}
	if !got[0].NeedDate.Equal(inicio.AddDate(0, 0, 12)) || got[0].DemandType != DemandTypeOrderAllocation || *got[0].SourceCode != 7 {
		t.Fatalf("componente da montagem: %+v", got[0])
	}
	if !got[1].NeedDate.Equal(inicio) {
		t.Fatalf("componente sem etapa entra no início da OF; veio %s", got[1].NeedDate)
	}
}

// Planejada firme sem OF: o MRP já a conta como suprimento do armário, então a
// ferragem dela é demanda — na data da montagem.
func TestPlanejadaFirmeSemOFExplodeOsComponentes(t *testing.T) {
	inicio := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	montagem := int16(30)
	s := &MRPServiceImpl{RoutingRepo: roteiroDoArmario{}}
	bom := map[int64][]*structentity.ItemStructure{
		50: {{ParentCode: 50, ChildCode: 7, Quantity: decimal.NewFromInt(4), IsActive: true, RouteOperationSequence: &montagem}},
	}
	got := s.demandaDasPlanejadasSemOF(context.Background(), []mrpports.FirmPlannedWithoutOrder{
		{Code: 11, ItemCode: 50, Quantity: decimal.NewFromInt(3), Start: inicio, Need: inicio.AddDate(0, 0, 14)},
		{Code: 12, ItemCode: 50, Quantity: decimal.Zero, Start: inicio, Need: inicio},
	}, bom, 1, newMaskVarCache(nil))
	if len(got) != 1 {
		t.Fatalf("quer 1 necessidade (a planejada zerada não explode); vieram %d", len(got))
	}
	g := got[0]
	if g.ItemCode != 7 || !g.Quantity.Equal(decimal.NewFromInt(12)) || g.DemandType != DemandTypeFirmPlanned || *g.SourceCode != 11 {
		t.Fatalf("componente da planejada: %+v", g)
	}
	if !g.NeedDate.Equal(inicio.AddDate(0, 0, 12)) {
		t.Fatalf("a ferragem entra na montagem (12º dia); veio %s", g.NeedDate.Format("02/01"))
	}
}
