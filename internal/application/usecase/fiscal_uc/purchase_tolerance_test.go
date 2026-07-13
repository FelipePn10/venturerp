package fiscal_uc

import (
	"context"
	"testing"

	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	poentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	porepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
	"github.com/shopspring/decimal"
)

type tolerancePORepo struct{ porepo.PurchaseOrderRepository }

func (tolerancePORepo) GetByCode(context.Context, int64) (*poentity.PurchaseOrder, error) {
	s := int64(9)
	return &poentity.PurchaseOrder{Code: 1, SupplierCode: &s}, nil
}
func (tolerancePORepo) ListItems(context.Context, int64) ([]*poentity.PurchaseOrderItem, error) {
	return []*poentity.PurchaseOrderItem{{ItemCode: 10, RequestedQty: 5, UnitPrice: 2}}, nil
}

type toleranceEval struct{ action string }

func (e toleranceEval) EvaluatePurchaseTolerance(context.Context, *int64, string, string, decimal.Decimal, decimal.Decimal) (string, string, bool, error) {
	return e.action, "divergence", true, nil
}

func TestValidatePurchaseEntryTolerancesWarnsAndBlocks(t *testing.T) {
	po := int64(1)
	item := int64(10)
	items := []*fiscalentity.FiscalEntryItem{{ItemCode: &item, Quantity: 7, UnitPrice: 3}}
	supplier, warnings, err := validatePurchaseEntryTolerances(context.Background(), tolerancePORepo{}, toleranceEval{action: "WARN"}, &po, items, 21)
	if err != nil || supplier == nil || len(warnings) != 3 {
		t.Fatalf("supplier=%v warnings=%v err=%v", supplier, warnings, err)
	}
	_, _, err = validatePurchaseEntryTolerances(context.Background(), tolerancePORepo{}, toleranceEval{action: "BLOCK"}, &po, items, 21)
	if err == nil {
		t.Fatal("blocking tolerance accepted divergent invoice")
	}
}
