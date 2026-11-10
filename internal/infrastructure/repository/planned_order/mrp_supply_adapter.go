package planned_order

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/ports"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

// PlannedOrderSupplyAdapter implements ports.PlannedOrderSupplyPort.
// Returns only firm (is_firm = TRUE) planned orders for time-phased netting.
type PlannedOrderSupplyAdapter struct {
	q *sqlc.Queries
}

func NewPlannedOrderSupplyAdapter(q *sqlc.Queries) *PlannedOrderSupplyAdapter {
	return &PlannedOrderSupplyAdapter{q: q}
}

func (a *PlannedOrderSupplyAdapter) ListFirmSupplyForItems(
	ctx context.Context,
	itemCodes []int64,
) (map[int64][]ports.SupplyEntry, error) {
	if len(itemCodes) == 0 {
		return make(map[int64][]ports.SupplyEntry), nil
	}

	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := a.q.ListFirmPlannedOrdersByItems(ctx, sqlc.ListFirmPlannedOrdersByItemsParams{ItemCodes: itemCodes, EnterpriseID: enterpriseID})
	if err != nil {
		return nil, err
	}

	result := make(map[int64][]ports.SupplyEntry, len(rows))
	for _, row := range rows {
		result[row.ItemCode] = append(result[row.ItemCode], ports.SupplyEntry{
			ItemCode:    row.ItemCode,
			Quantity:    pgutil.FromPgNumericToDecimal(row.Quantity),
			ArrivalDate: pgutil.FromPgDate(row.NeedDate),
			SourceType:  ports.SupplySourcePlannedOrder,
			SourceCode:  row.Code,
		})
	}

	// OF e pedido de compra que não nasceram de ordem planejada firme também
	// chegam: sem eles o MRP sugeria de novo o que a fábrica já está fazendo e
	// o que o comprador já pediu.
	ofs, err := a.q.ListOpenProductionOrderSupply(ctx, sqlc.ListOpenProductionOrderSupplyParams{EnterpriseID: enterpriseID, ItemCodes: itemCodes})
	if err != nil {
		return nil, fmt.Errorf("lendo OFs abertas como suprimento: %w", err)
	}
	for _, row := range ofs {
		result[row.ItemCode] = append(result[row.ItemCode], ports.SupplyEntry{
			ItemCode: row.ItemCode, Quantity: pgutil.FromPgNumericToDecimal(row.Quantity),
			ArrivalDate: pgutil.FromPgDate(row.ArrivalDate), SourceType: ports.SupplySourceProductionOrder, SourceCode: row.ID,
		})
	}
	if enterpriseID != nil {
		pedidos, err := a.q.ListOpenPurchaseOrderSupply(ctx, sqlc.ListOpenPurchaseOrderSupplyParams{EnterpriseID: *enterpriseID, ItemCodes: itemCodes})
		if err != nil {
			return nil, fmt.Errorf("lendo pedidos de compra em aberto como suprimento: %w", err)
		}
		for _, row := range pedidos {
			result[row.ItemCode] = append(result[row.ItemCode], ports.SupplyEntry{
				ItemCode: row.ItemCode, Quantity: pgutil.FromPgNumericToDecimal(row.Quantity),
				ArrivalDate: pgutil.FromPgDate(row.ArrivalDate), SourceType: ports.SupplySourcePurchaseOrder, SourceCode: row.Code,
			})
		}
	}
	return result, nil
}

// ListOpenOrderAllocations: o empenho das OFs abertas (necessário − atendido).
func (a *PlannedOrderSupplyAdapter) ListOpenOrderAllocations(ctx context.Context) ([]ports.OrderAllocation, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := a.q.ListOpenProductionOrderAllocations(ctx, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("lendo o empenho das OFs abertas: %w", err)
	}
	out := make([]ports.OrderAllocation, 0, len(rows))
	for _, row := range rows {
		al := ports.OrderAllocation{
			ProductionOrderID: row.ProductionOrderID, ParentItemCode: row.ParentItemCode, ParentMask: row.ParentMask, ParentQuantity: pgutil.FromPgNumericToDecimal(row.ParentQuantity),
			ItemCode: row.ItemCode, Mask: row.Mask, Quantity: pgutil.FromPgNumericToDecimal(row.Quantity),
			OrderStart: pgutil.FromPgDate(row.StartDate), OrderEnd: pgutil.FromPgDate(row.EndDate),
		}
		if row.RouteOperationSequence.Valid {
			etapa := row.RouteOperationSequence.Int16
			al.Step = &etapa
		}
		out = append(out, al)
	}
	return out, nil
}

// ListFirmPlannedProductionWithoutOrder: planejada firme de fabricação sem OF.
func (a *PlannedOrderSupplyAdapter) ListFirmPlannedProductionWithoutOrder(ctx context.Context) ([]ports.FirmPlannedWithoutOrder, error) {
	enterpriseID, err := tenant.IDPtr(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := a.q.ListFirmPlannedProductionWithoutOrder(ctx, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("lendo planejadas firmes sem OF: %w", err)
	}
	out := make([]ports.FirmPlannedWithoutOrder, 0, len(rows))
	for _, r := range rows {
		out = append(out, ports.FirmPlannedWithoutOrder{
			Code: r.Code, ItemCode: r.ItemCode, Mask: r.Mask, Quantity: pgutil.FromPgNumericToDecimal(r.Quantity),
			Start: pgutil.FromPgDate(r.StartDate), Need: pgutil.FromPgDate(r.NeedDate),
		})
	}
	return out, nil
}
