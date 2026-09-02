package salespricing

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/customer/entity"
)

type pricingRepositoryStub struct {
	table *entity.SalesTable
	price *entity.SalesTablePrice
	err   error
}

func (s pricingRepositoryStub) GetSalesTableByCode(context.Context, int64) (*entity.SalesTable, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.table, nil
}

func (s pricingRepositoryStub) GetSalesTablePrice(context.Context, int64, string) (*entity.SalesTablePrice, error) {
	if s.price == nil {
		return nil, errors.New("não encontrado")
	}
	return s.price, nil
}

func TestResolveUsesActiveCurrentSalesTablePrice(t *testing.T) {
	result, err := Resolve(context.Background(), pricingRepositoryStub{
		table: &entity.SalesTable{ID: 10, Code: 20, IsActive: true},
		price: &entity.SalesTablePrice{Price: 123.45, Situation: entity.PriceSituationAtivo},
	}, 20, 30, 2)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if result.AppliedPrice != 123.45 || result.BasePrice != 123.45 || result.Source != "SALES_TABLE" {
		t.Fatalf("Resolve() = %+v", result)
	}
}

func TestResolveRejectsExpiredBlockedAndMissingPricesInPortuguese(t *testing.T) {
	yesterday := time.Now().Add(-48 * time.Hour)
	tests := []struct {
		name string
		repo pricingRepositoryStub
		want string
	}{
		{"tabela vencida", pricingRepositoryStub{table: &entity.SalesTable{IsActive: true, ValidityEnd: &yesterday}}, "vencida"},
		{"preço bloqueado", pricingRepositoryStub{table: &entity.SalesTable{ID: 1, IsActive: true}, price: &entity.SalesTablePrice{Price: 10, Blocked: true}}, "bloqueado"},
		{"preço ausente", pricingRepositoryStub{table: &entity.SalesTable{ID: 1, IsActive: true}}, "não existe preço válido"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Resolve(context.Background(), tt.repo, 2, 3, 1)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Resolve() error = %v, want %q", err, tt.want)
			}
		})
	}
}
