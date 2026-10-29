package service

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	inddemandentity "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
	inddemandrepo "github.com/FelipePn10/panossoerp/internal/domain/independent_demand/repository"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	mrprepo "github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/repository"
	planentity "github.com/FelipePn10/panossoerp/internal/domain/production_plan/entity"
	planrepo "github.com/FelipePn10/panossoerp/internal/domain/production_plan/repository"
)

type vendasQuebradas struct{}

func (vendasQuebradas) Vendas(context.Context, []int64) ([]inddemandentity.Venda, error) {
	return nil, errors.New("conexão perdida")
}

// calculoAteOConsumo responde só o que o cálculo lê antes do consumo pela venda
// e guarda como ele terminou.
type calculoAteOConsumo struct {
	mrprepo.MRPCalculationRepository
	status string
	erros  map[string]interface{}
}

func (*calculoAteOConsumo) StartCalculation(context.Context, int64) (*entity.MRPCalculationLog, error) {
	return &entity.MRPCalculationLog{Code: 1}, nil
}
func (*calculoAteOConsumo) DeleteSuggestionsByPlan(context.Context, int64) error    { return nil }
func (*calculoAteOConsumo) DeleteProfilesByPlan(context.Context, int64) error       { return nil }
func (*calculoAteOConsumo) DeleteProfileDetailsByPlan(context.Context, int64) error { return nil }
func (*calculoAteOConsumo) DeleteExceptionsByPlan(context.Context, int64) error     { return nil }
func (*calculoAteOConsumo) LoadTypedPlanningParams(context.Context) (*entity.TypedPlanningParams, error) {
	return entity.DefaultTypedPlanningParams(), nil
}
func (*calculoAteOConsumo) ListOpenSalesOrderDemands(context.Context, int64, *int64) ([]*entity.MRPInput, error) {
	return []*entity.MRPInput{{ItemCode: 7, Quantity: decimal.NewFromInt(12), NeedDate: today()}}, nil
}
func (c *calculoAteOConsumo) FinishCalculation(_ context.Context, _ int64, status string, erros map[string]interface{}, _, _ int) (*entity.MRPCalculationLog, error) {
	c.status, c.erros = status, erros
	return &entity.MRPCalculationLog{Code: 1, Status: status}, nil
}

type planoTodos struct {
	planrepo.ProductionPlanRepository
}

func (planoTodos) GetByCode(context.Context, int64) (*planentity.ProductionPlan, error) {
	return &planentity.ProductionPlan{Code: 5, IndependentDemands: planentity.IndependentDemandsAll}, nil
}

type umaDemandaDeEstoque struct {
	inddemandrepo.IndependentDemandRepository
}

func (umaDemandaDeEstoque) List(context.Context) ([]*inddemandentity.IndependentDemand, error) {
	return []*inddemandentity.IndependentDemand{{CodeDemand: 1, ItemCode: 7, Quantity: decimal.NewFromInt(10), DemandDate: today(),
		Motivo: inddemandentity.MotivoEstoque, Situacao: inddemandentity.SituacaoAberta}}, nil
}

// Sem as vendas a demanda de estoque e a previsão somariam com o pedido (a
// dupla contagem): o cálculo para em ERRO em vez de gravar um plano inflado.
func TestCalculoParaQuandoNaoLeAsVendasDoConsumo(t *testing.T) {
	repo := &calculoAteOConsumo{}
	s := &MRPServiceImpl{MRPRepo: repo, PlanRepo: planoTodos{}, DemandRepo: umaDemandaDeEstoque{}, Vendas: vendasQuebradas{}}
	if _, err := s.Calculate(context.Background(), 5, 1, false); err != nil {
		t.Fatal(err)
	}
	if repo.status != "ERROR" || repo.erros["consumo_pela_venda"] == nil {
		t.Fatalf("status %q, erros %v", repo.status, repo.erros)
	}
}
