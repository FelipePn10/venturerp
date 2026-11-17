package service

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	mrprepo "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
	routing "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/shopspring/decimal"
)

// roteiroFixo devolve um roteiro de uma etapa e o código comercial do item.
type roteiroFixo struct {
	mrprepo.MRPCalculationRepository
	ops []*routing.RouteOperation
}

func (r roteiroFixo) MachineRouteQty(context.Context, int64, string, decimal.Decimal) ([]*routing.RouteOperation, []*routing.NetworkEdge, error) {
	return r.ops, nil, nil
}

func (roteiroFixo) ItemBusinessCodes(_ context.Context, codes []int64) (map[int64]string, error) {
	return map[int64]string{codes[0]: "RN-01001"}, nil
}

func etapa(run float64) *routing.RouteOperation {
	centro := int64(3)
	return &routing.RouteOperation{ID: 1, Sequence: 10, OperationName: "Corte", WorkCenterName: "Corte a laser",
		EffectiveWorkCenterID: &centro, OperationOrigin: routing.OriginInternal,
		EffTime: routing.OperationTime{Run: run, RunBaseQty: 1}}
}

// Com tempo no roteiro, a produtividade só precisa existir (máquina) e ter
// rendimento válido: tempo zero nela não pode recusar o plano.
func TestTempoDoRoteiroDispensaTempoNaProdutividade(t *testing.T) {
	s := &MRPServiceImpl{MRPRepo: roteiroFixo{ops: []*routing.RouteOperation{etapa(0.5)}}}
	sem := &entity.MachineTimeInfo{ItemCode: 7, MachineCode: 70, WorkCenterID: 3, ProductionTime: 0,
		ProductionTimeUnit: "HORA", ProductionBaseQty: 1, MachineEfficiencyRate: 1, WorkingHoursPerDay: 8}
	sug := &entity.PlannedOrderSuggestion{ItemCode: 7, Quantity: decimal.NewFromInt(4)}

	steps, err := s.machineSteps(context.Background(), sug, []*entity.MachineTimeInfo{sem})
	if err != nil {
		t.Fatalf("tempo zero na produtividade recusou o plano: %v", err)
	}
	if len(steps) != 1 || math.Abs(steps[0].minutes-120) > 1e-9 { // 4 × 0,5 h
		t.Fatalf("o tempo deveria vir do roteiro (120 min): %+v", steps)
	}
}

func TestMensagensDaEtapaSemMaquinaOuSemTempo(t *testing.T) {
	sug := &entity.PlannedOrderSuggestion{ItemCode: 7, Quantity: decimal.NewFromInt(4)}

	// Sem máquina no centro, com tempo no roteiro: só falta a máquina.
	s := &MRPServiceImpl{MRPRepo: roteiroFixo{ops: []*routing.RouteOperation{etapa(0.5)}}}
	_, err := s.machineSteps(context.Background(), sug, nil)
	if err == nil || !strings.Contains(err.Error(), "cadastre o RN-01001 na produtividade de uma máquina de cada centro do roteiro (o tempo vem do roteiro)") ||
		!strings.Contains(err.Error(), "Corte a laser") {
		t.Fatalf("mensagem sem máquina: %v", err)
	}

	// Etapa sem tempo e produtividade sem tempo: o tempo falta nos dois lugares.
	s = &MRPServiceImpl{MRPRepo: roteiroFixo{ops: []*routing.RouteOperation{etapa(0)}}}
	sem := &entity.MachineTimeInfo{ItemCode: 7, MachineCode: 70, WorkCenterID: 3, ProductionTimeUnit: "HORA", ProductionBaseQty: 1}
	_, err = s.machineSteps(context.Background(), sug, []*entity.MachineTimeInfo{sem})
	if err == nil || !strings.Contains(err.Error(), "não tem tempo") || !strings.Contains(err.Error(), "RN-01001") {
		t.Fatalf("mensagem sem tempo nos dois lugares: %v", err)
	}

	// Rendimento impossível com tempo no roteiro é recusado, não vira 100 %.
	s = &MRPServiceImpl{MRPRepo: roteiroFixo{ops: []*routing.RouteOperation{etapa(0.5)}}}
	ruim := 1.5
	sem.EfficiencyRate = &ruim
	if _, err = s.machineSteps(context.Background(), sug, []*entity.MachineTimeInfo{sem}); err == nil || !strings.Contains(err.Error(), "eficiência inválida") {
		t.Fatalf("eficiência inválida passou: %v", err)
	}
}
