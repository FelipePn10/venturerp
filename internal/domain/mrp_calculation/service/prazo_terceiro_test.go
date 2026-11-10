package service

import (
	"context"
	"testing"

	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	routingrepo "github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
	"github.com/shopspring/decimal"
)

// A bucha: 12 h de trabalho nosso (2 dias úteis) e 9 dias corridos na
// cementação (7 dias úteis). Antes o MRP planejava 2 dias e a OP da balança
// começava sem a bucha.
func TestPrazoDoRoteiroSomaOsDiasNoTerceiro(t *testing.T) {
	if got := prazoEmDiasUteis(12, 9); got != 9 {
		t.Fatalf("12 h + 9 dias corridos no terceiro = 2 + 7 dias úteis; veio %d", got)
	}
	if got := prazoEmDiasUteis(12, 0); got != 2 {
		t.Fatalf("sem terceiro, só as horas; veio %d", got)
	}
	if got := prazoEmDiasUteis(0, 0); got != 0 {
		t.Fatalf("roteiro sem tempo tem prazo zero; veio %d", got)
	}
}

// Roteiro de uma etapa de 0,2 h por peça e uma zincagem de 5 dias corridos.
type roteiroPorPeca struct{ routingrepo.RoutingRepository }

func (roteiroPorPeca) GetRouteForItem(context.Context, int64, string) (*routingentity.ManufacturingRoute, error) {
	return &routingentity.ManufacturingRoute{ID: 1}, nil
}
func (roteiroPorPeca) GetRouteOperations(context.Context, int64) ([]*routingentity.RouteOperation, error) {
	return []*routingentity.RouteOperation{
		{ID: 1, Sequence: 10, EffTime: routingentity.OperationTime{Run: 0.2, RunBaseQty: 1}},
		{ID: 2, Sequence: 20, EffectiveSubcontractDays: 5},
	}, nil
}
func (roteiroPorPeca) GetNetworkEdges(context.Context, int64) ([]*routingentity.NetworkEdge, error) {
	return nil, nil
}

// 60 armários não levam o prazo de um: 60 × 0,2 h = 12 h (2 dias úteis) + 5
// dias corridos no terceiro (4 úteis) = 6; uma peça leva 1 + 4 = 5.
func TestPrazoDoRoteiroEscalaComOLote(t *testing.T) {
	s := &MRPServiceImpl{RoutingRepo: roteiroPorPeca{}}
	um, err := s.prazoDoRoteiroParaQuantidade(context.Background(), 1, "", decimal.NewFromInt(1))
	if err != nil {
		t.Fatal(err)
	}
	lote, _ := s.prazoDoRoteiroParaQuantidade(context.Background(), 1, "", decimal.NewFromInt(60))
	if um != 5 || lote != 6 {
		t.Fatalf("1 peça = 5 dias e 60 peças = 6 dias; veio %d e %d", um, lote)
	}
}
