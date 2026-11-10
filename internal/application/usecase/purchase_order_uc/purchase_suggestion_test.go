package purchase_order_uc

import (
	"testing"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	plannedentity "github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
)

func TestIsPurchaseSuggestion(t *testing.T) {
	base := func() *plannedentity.PlannedOrder {
		return &plannedentity.PlannedOrder{
			OrderType: types.OrderPurchase,
			Status:    types.StatusPlanned,
			IsFirm:    false,
			IsActive:  true,
		}
	}

	if !isPurchaseSuggestion(base()) {
		t.Error("a PURCHASE planned order, PLANNED, not firm and active should be a suggestion")
	}
	// Firmada no MRP (VMRP0100 → RELEASED, firme): continua na fila do
	// comprador até virar pedido. Antes sumia sem pedido nem requisição.
	firmada := base()
	firmada.IsFirm, firmada.Status = true, types.StatusReleased
	if !isPurchaseSuggestion(firmada) {
		t.Error("compra firmada pelo planejador tem de continuar na fila do comprador")
	}

	cases := []struct {
		name   string
		mutate func(*plannedentity.PlannedOrder)
	}{
		{"not active", func(o *plannedentity.PlannedOrder) { o.IsActive = false }},
		{"not purchase", func(o *plannedentity.PlannedOrder) { o.OrderType = types.OrderProduction }},
		{"cancelled", func(o *plannedentity.PlannedOrder) { o.Status = types.StatusCancelled }},
		{"finished", func(o *plannedentity.PlannedOrder) { o.Status = types.StatusFinished }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := base()
			tc.mutate(o)
			if isPurchaseSuggestion(o) {
				t.Errorf("%s: should NOT be a suggestion", tc.name)
			}
		})
	}
}
