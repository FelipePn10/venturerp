package planned_order

import (
	"context"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/ports"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
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

	rows, err := a.q.ListFirmPlannedOrdersByItems(ctx, itemCodes)
	if err != nil {
		return nil, err
	}

	result := make(map[int64][]ports.SupplyEntry, len(rows))
	for _, row := range rows {
		result[row.ItemCode] = append(result[row.ItemCode], ports.SupplyEntry{
			ItemCode:    row.ItemCode,
			Quantity:    pgutil.FromPgNumericToFloat64(row.Quantity),
			ArrivalDate: pgutil.FromPgDate(row.NeedDate),
			SourceType:  ports.SupplySourcePlannedOrder,
			SourceCode:  row.Code,
		})
	}
	return result, nil
}
