package service

import (
	"context"
	"strings"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	routing "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	"github.com/shopspring/decimal"
)

// A mensagem da etapa sem centro citava "operação 15 do item 91000116551": o
// id da etapa e a chave do item, que o usuário não acha em tela nenhuma.
func TestEtapaSemCentroCitaSequenciaECodigoDeNegocio(t *testing.T) {
	op := etapa(0.5)
	op.ID = 15
	op.EffectiveWorkCenterID = nil
	s := &MRPServiceImpl{MRPRepo: roteiroFixo{ops: []*routing.RouteOperation{op}}}
	sug := &entity.PlannedOrderSuggestion{ItemCode: 91000116551, Quantity: decimal.NewFromInt(4)}
	_, err := s.machineSteps(context.Background(), sug, nil)
	if err == nil || !strings.Contains(err.Error(), "etapa 10 (Corte) do roteiro do RN-01001 sem centro de trabalho") ||
		strings.Contains(err.Error(), "91000116551") {
		t.Fatalf("mensagem: %v", err)
	}
}

// Sem roteiro e com produtividade sem tempo: a mensagem cita o código de
// negócio (machineMinutes só conhece a chave interna).
func TestSemRoteiroProdutividadeInvalidaCitaCodigoDeNegocio(t *testing.T) {
	s := &MRPServiceImpl{MRPRepo: roteiroFixo{}}
	ruim := &entity.MachineTimeInfo{ItemCode: 91000116551, MachineCode: 70, WorkCenterID: 3, ProductionTimeUnit: "HORA"}
	sug := &entity.PlannedOrderSuggestion{ItemCode: 91000116551, Quantity: decimal.NewFromInt(4)}
	_, err := s.machineSteps(context.Background(), sug, []*entity.MachineTimeInfo{ruim})
	if err == nil || !strings.Contains(err.Error(), "o RN-01001 não tem roteiro") || strings.Contains(err.Error(), "91000116551") {
		t.Fatalf("mensagem: %v", err)
	}
}
