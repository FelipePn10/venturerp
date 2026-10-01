package cost_uc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	routingentity "github.com/FelipePn10/panossoerp/internal/domain/routing/entity"
	routingrepo "github.com/FelipePn10/panossoerp/internal/domain/routing/repository"
	scentity "github.com/FelipePn10/panossoerp/internal/domain/standard_cost/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/standard_cost/repository"
	thirdpartyentity "github.com/FelipePn10/panossoerp/internal/domain/third_party_service"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestSelectPrimaryCostSubstitutes(t *testing.T) {
	children := []domainrepo.BOMChild{
		{ChildCode: 20, Quantity: 2, SubstituteGroup: 1, SubstitutePriority: 2},
		{ChildCode: 10, Quantity: 3, SubstituteGroup: 1, SubstitutePriority: 1},
		{ChildCode: 30, Quantity: 5},
	}

	selected := selectPrimaryCostSubstitutes(children)

	got := map[int64]float64{}
	for _, child := range selected {
		got[child.ChildCode] = child.Quantity
	}
	if _, ok := got[20]; ok {
		t.Error("substituto secundário não deve entrar no custo padrão")
	}
	if got[10] != 3 {
		t.Errorf("substituto primário = %v, want 3", got[10])
	}
	if got[30] != 5 {
		t.Errorf("componente standalone = %v, want 5", got[30])
	}
	if len(selected) != 2 {
		t.Errorf("selected = %d, want 2", len(selected))
	}
}

type costRoutingStub struct{ routingrepo.RoutingRepository }

func (costRoutingStub) GetRouteForItem(context.Context, int64, string) (*routingentity.ManufacturingRoute, error) {
	return &routingentity.ManufacturingRoute{ID: 1}, nil
}
func (costRoutingStub) GetRouteOperations(context.Context, int64) ([]*routingentity.RouteOperation, error) {
	return []*routingentity.RouteOperation{{OperationID: 7, OperationOrigin: routingentity.OriginThirdPart, Situation: routingentity.RouteOpApproved}}, nil
}

type failingThirdPartyCost struct{}

func (failingThirdPartyCost) StandardCostPerUnit(context.Context, int64, string, int64, time.Time) (decimal.Decimal, error) {
	return decimal.Zero, errors.New("contract price not found")
}

func TestRollUpRejectsExternalOperationWithoutResolvedContractPrice(t *testing.T) {
	repo := &fakeCostRepo{childrenByMask: map[string][]domainrepo.BOMChild{"1|": {}}, purchaseCosts: map[int64]float64{1: 10}}
	_, err := New(repo).WithRouting(costRoutingStub{}).WithThirdPartyPrices(failingThirdPartyCost{}).RollUp(context.Background(), request.CostRollupDTO{ItemCode: 1, LotSize: 1, CalculatedBy: uuid.NewString()})
	if err == nil {
		t.Fatal("standard cost must not silently ignore a missing external-service price")
	}
}

func TestRollUp_UsesMaskResolvedBOM(t *testing.T) {
	repo := &fakeCostRepo{
		childrenByMask: map[string][]domainrepo.BOMChild{
			"1|":  {{ChildCode: 20, Quantity: 99}},
			"1|A": {{ChildCode: 10, Quantity: 2}},
		},
		purchaseCosts: map[int64]float64{
			10: 7,
			20: 1000,
		},
	}
	uc := New(repo)

	res, err := uc.RollUp(context.Background(), request.CostRollupDTO{
		ItemCode:     1,
		Mask:         "A",
		LotSize:      1,
		CalculatedBy: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.calls) == 0 || repo.calls[0] != "1|A" {
		t.Fatalf("first GetDirectChildren call = %v, want 1|A", repo.calls)
	}
	if res.MaterialCost != 14 {
		t.Fatalf("material cost = %v, want 14 from masked child only", res.MaterialCost)
	}
}

type fakeCostRepo struct {
	domainrepo.StandardCostRepository
	childrenByMask map[string][]domainrepo.BOMChild
	purchaseCosts  map[int64]float64
	calls          []string
}

func (f *fakeCostRepo) GetDirectChildren(_ context.Context, parentCode int64, mask string) ([]domainrepo.BOMChild, error) {
	f.calls = append(f.calls, costKey(parentCode, mask))
	return f.childrenByMask[costKey(parentCode, mask)], nil
}

func (f *fakeCostRepo) GetItemPurchaseCost(_ context.Context, itemCode int64) (*scentity.ItemPurchaseCost, error) {
	return &scentity.ItemPurchaseCost{ItemCode: itemCode, UnitCost: f.purchaseCosts[itemCode], Currency: "BRL"}, nil
}

func (f *fakeCostRepo) GetRouteHoursByItem(context.Context, int64, string) (float64, error) {
	return 0, nil
}

func (f *fakeCostRepo) ListWorkCenterCosts(context.Context) ([]*scentity.WorkCenterCost, error) {
	return nil, nil
}

func (f *fakeCostRepo) UpsertItemStandardCost(_ context.Context, cost *scentity.ItemStandardCost) (*scentity.ItemStandardCost, error) {
	cost.TotalCost = cost.MaterialCost + cost.LaborCost + cost.OverheadCost
	return cost, nil
}

func (f *fakeCostRepo) InsertRollupLog(context.Context, *scentity.CostRollupLogEntry) error {
	return nil
}

func costKey(parentCode int64, mask string) string {
	return fmt.Sprintf("%d|%s", parentCode, mask)
}

// O stub acima devolve um erro genérico, e o teste que o usa só verifica err != nil.
// Isso passava enquanto o usuário recebia HTTP 500 "erro interno do servidor": a
// frase que explica o problema ficava só no log do servidor. O que importa para
// quem usa é o TIPO do erro (que decide o status) e o conteúdo da mensagem.
type thirdPartyPrecoAusente struct{}

func (thirdPartyPrecoAusente) StandardCostPerUnit(context.Context, int64, string, int64, time.Time) (decimal.Decimal, error) {
	return decimal.Zero, thirdpartyentity.ErrNotFound
}

type costRoutingComNomeDaOperacao struct{ routingrepo.RoutingRepository }

func (costRoutingComNomeDaOperacao) GetRouteForItem(context.Context, int64, string) (*routingentity.ManufacturingRoute, error) {
	return &routingentity.ManufacturingRoute{ID: 1}, nil
}
func (costRoutingComNomeDaOperacao) GetRouteOperations(context.Context, int64) ([]*routingentity.RouteOperation, error) {
	return []*routingentity.RouteOperation{{
		OperationID: 7, OperationName: "GALVANIZAÇÃO A FRIO EXTERNA",
		OperationOrigin: routingentity.OriginThirdPart, Situation: routingentity.RouteOpApproved,
	}}, nil
}

func TestPrecoDeTerceiroAusenteEhPendenciaDeCadastroENaoFalhaDoSistema(t *testing.T) {
	repo := &fakeCostRepo{childrenByMask: map[string][]domainrepo.BOMChild{"1|": {}}, purchaseCosts: map[int64]float64{1: 10}}
	_, err := New(repo).
		WithRouting(costRoutingComNomeDaOperacao{}).
		WithThirdPartyPrices(thirdPartyPrecoAusente{}).
		RollUp(context.Background(), request.CostRollupDTO{ItemCode: 1, LotSize: 1, CalculatedBy: uuid.NewString()})
	if err == nil {
		t.Fatal("preço de terceiro ausente não pode passar em silêncio")
	}

	// Tipado: é o que faz o handler devolver 422 em vez de 500.
	var validacao *errorsuc.ValidationError
	if !errors.As(err, &validacao) {
		t.Fatalf("erro deveria ser de validação (→ 422), veio %T: %v", err, err)
	}

	msg := err.Error()
	// A mensagem precisa dizer QUAL operação e ONDE resolver.
	for _, esperado := range []string{"GALVANIZAÇÃO A FRIO EXTERNA", "VTER0100"} {
		if !strings.Contains(msg, esperado) {
			t.Errorf("mensagem não menciona %q: %s", esperado, msg)
		}
	}
	// E não pode expor o código INTERNO do item, que o usuário não reconhece —
	// é o mesmo problema dos campos "(ID)".
	if strings.Contains(msg, "item 1 ") || strings.Contains(msg, "item 7") {
		t.Errorf("mensagem expõe código interno: %s", msg)
	}
}
