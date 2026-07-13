package fiscal_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/ports"
	fiscalentity "github.com/FelipePn10/panossoerp/internal/domain/fiscal/entity"
	porepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
	"github.com/shopspring/decimal"
)

func validatePurchaseEntryTolerances(ctx context.Context, orders porepo.PurchaseOrderRepository, evaluator ports.PurchaseToleranceEvaluator, purchaseOrderCode *int64, items []*fiscalentity.FiscalEntryItem, productsTotal float64) (*int64, []string, error) {
	if purchaseOrderCode == nil || orders == nil || evaluator == nil {
		return nil, nil, nil
	}
	order, err := orders.GetByCode(ctx, *purchaseOrderCode)
	if err != nil {
		return nil, nil, err
	}
	lines, err := orders.ListItems(ctx, *purchaseOrderCode)
	if err != nil {
		return nil, nil, err
	}
	byItem := map[int64][]int{}
	for i, line := range lines {
		byItem[line.ItemCode] = append(byItem[line.ItemCode], i)
	}
	warnings := []string{}
	expectedTotal := decimal.Zero
	for _, line := range lines {
		expectedTotal = expectedTotal.Add(decimal.NewFromFloat(line.RequestedQty).Mul(decimal.NewFromFloat(line.UnitPrice)))
	}
	check := func(kind string, expected, actual decimal.Decimal) error {
		action, msg, exceeded, err := evaluator.EvaluatePurchaseTolerance(ctx, order.SupplierCode, kind, "ENTRY_INVOICE", expected, actual)
		if err != nil {
			return err
		}
		if !exceeded {
			return nil
		}
		if action == "BLOCK" {
			return fmt.Errorf("purchase tolerance blocked entry invoice: %s", msg)
		}
		if action == "WARN" {
			warnings = append(warnings, msg)
		}
		return nil
	}
	for _, item := range items {
		if item.ItemCode == nil {
			continue
		}
		indexes := byItem[*item.ItemCode]
		if len(indexes) == 0 {
			continue
		}
		line := lines[indexes[0]]
		byItem[*item.ItemCode] = indexes[1:]
		if err = check("QUANTITY", decimal.NewFromFloat(line.RequestedQty), decimal.NewFromFloat(item.Quantity)); err != nil {
			return nil, nil, err
		}
		if err = check("ITEM_PRICE", decimal.NewFromFloat(line.UnitPrice), decimal.NewFromFloat(item.UnitPrice)); err != nil {
			return nil, nil, err
		}
	}
	if err = check("PRODUCTS_TOTAL", expectedTotal, decimal.NewFromFloat(productsTotal)); err != nil {
		return nil, nil, err
	}
	return order.SupplierCode, warnings, nil
}
