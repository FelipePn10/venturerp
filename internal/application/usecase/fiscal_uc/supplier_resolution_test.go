package fiscal_uc

import (
	"context"
	"testing"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	"github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
)

type supplierResolverStub struct {
	result *ports.SupplierItemResolution
	calls  int
}

func (s *supplierResolverStub) ResolveExternal(context.Context, int64, string, string) (*ports.SupplierItemResolution, error) {
	s.calls++
	return s.result, nil
}

func TestResolveSupplierItemsPersistsTraceAndPreservesManualChoice(t *testing.T) {
	item := int64(10)
	external := "TEA452-0"
	items := []*entity.FiscalEntryItem{{SupplierItemCode: &external}, {ItemCode: &item}}
	stub := &supplierResolverStub{result: &ports.SupplierItemResolution{LinkID: 8, ItemCode: 77, Strategy: "CODIGO_EXATO"}}
	if err := resolveSupplierItems(context.Background(), stub, &item, items); err != nil {
		t.Fatal(err)
	}
	if items[0].ItemCode == nil || *items[0].ItemCode != 77 || items[0].ItemSupplierID == nil || *items[0].ResolutionStrategy != "CODIGO_EXATO" || items[0].ResolvedAt == nil {
		t.Fatalf("rastreio incompleto: %#v", items[0])
	}
	if *items[1].ItemCode != 10 || *items[1].ResolutionStrategy != "MANUAL" || stub.calls != 1 {
		t.Fatalf("escolha manual alterada: %#v", items[1])
	}
}

func TestResolveSupplierItemsMarksUnresolved(t *testing.T) {
	external := "DESCONHECIDO"
	item := &entity.FiscalEntryItem{SupplierItemCode: &external}
	supplier := int64(5)
	if err := resolveSupplierItems(context.Background(), &supplierResolverStub{}, &supplier, []*entity.FiscalEntryItem{item}); err != nil {
		t.Fatal(err)
	}
	if item.ItemCode != nil || item.ResolutionStrategy == nil || *item.ResolutionStrategy != "NAO_RESOLVIDO" {
		t.Fatalf("resultado inesperado: %#v", item)
	}
}
