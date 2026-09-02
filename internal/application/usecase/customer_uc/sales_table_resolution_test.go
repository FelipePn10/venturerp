package customer_uc

import (
	"context"
	"errors"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
	customerrepo "github.com/FelipePn10/panossoerp/internal/domain/customer/repository"
)

type salesTableResolutionRepo struct {
	customerrepo.CustomerRepository
	tables []*entity.SalesTable
	prices map[int64]*entity.SalesTablePrice
}

func (r salesTableResolutionRepo) ListSalesTables(context.Context, bool) ([]*entity.SalesTable, error) {
	return r.tables, nil
}

func (r salesTableResolutionRepo) ListCommercialPolicies(context.Context, bool, *entity.CommercialPolicyKind) ([]*entity.CommercialPolicy, error) {
	return nil, nil
}

func (r salesTableResolutionRepo) GetSalesTablePrice(_ context.Context, tableID int64, _ string) (*entity.SalesTablePrice, error) {
	price := r.prices[tableID]
	if price == nil {
		return nil, errors.New("não encontrado")
	}
	return price, nil
}

func TestResolveSalesTablesForItemAutoSelectsOnlyValidCandidate(t *testing.T) {
	activeUnit := "UN"
	uc := NewCustomerUseCase(salesTableResolutionRepo{
		tables: []*entity.SalesTable{{ID: 1, Code: 10, Description: "Varejo", IsActive: true}, {ID: 2, Code: 20, Description: "Bloqueada", IsActive: true}},
		prices: map[int64]*entity.SalesTablePrice{1: {Price: 12.5, UME: &activeUnit, Situation: entity.PriceSituationAtivo}, 2: {Price: 9, Blocked: true, Situation: entity.PriceSituationAtivo}},
	})
	result, err := uc.ResolveSalesTablesForItem(context.Background(), request.ResolveSalesTablesForItemDTO{ItemCode: "100", Quantity: 2, Unit: "UN", Currency: "BRL"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.AutoSelected || result.Selected == nil || result.Selected.SalesTableCode != 10 || len(result.Candidates) != 1 {
		t.Fatalf("resolução inesperada: %+v", result)
	}
}

func TestResolveSalesTablesForItemReturnsTieForExplicitDecision(t *testing.T) {
	uc := NewCustomerUseCase(salesTableResolutionRepo{
		tables: []*entity.SalesTable{{ID: 1, Code: 10, IsActive: true}, {ID: 2, Code: 20, IsActive: true}},
		prices: map[int64]*entity.SalesTablePrice{1: {Price: 12, Situation: entity.PriceSituationAtivo}, 2: {Price: 13, Situation: entity.PriceSituationAtivo}},
	})
	result, err := uc.ResolveSalesTablesForItem(context.Background(), request.ResolveSalesTablesForItemDTO{ItemCode: "100", Quantity: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.AutoSelected || result.Selected != nil || len(result.Candidates) != 2 {
		t.Fatalf("esperava empate explícito: %+v", result)
	}
}
