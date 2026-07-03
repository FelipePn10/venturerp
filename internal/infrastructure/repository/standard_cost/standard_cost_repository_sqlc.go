package standard_cost

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/standard_cost/entity"
	domainrepo "github.com/FelipePn10/panossoerp/internal/domain/standard_cost/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

type StandardCostRepositorySQLC struct {
	q *sqlc.Queries
}

func New(q *sqlc.Queries) domainrepo.StandardCostRepository {
	return &StandardCostRepositorySQLC{q: q}
}

// ─── item_standard_costs ──────────────────────────────────────────────────────

func (r *StandardCostRepositorySQLC) UpsertItemStandardCost(ctx context.Context, cost *entity.ItemStandardCost) (*entity.ItemStandardCost, error) {
	row, err := r.q.UpsertItemStandardCost(ctx, sqlc.UpsertItemStandardCostParams{
		ItemCode:     cost.ItemCode,
		Mask:         cost.Mask,
		MaterialCost: pgutil.ToPgNumericFromFloat64(cost.MaterialCost),
		LaborCost:    pgutil.ToPgNumericFromFloat64(cost.LaborCost),
		OverheadCost: pgutil.ToPgNumericFromFloat64(cost.OverheadCost),
		Currency:     orDefault(cost.Currency, "BRL"),
		CalculatedBy: pgutil.ToPgUUID(cost.CalculatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("upserting item standard cost: %w", err)
	}
	return isCostRowToEntity(row), nil
}

func (r *StandardCostRepositorySQLC) GetItemStandardCost(ctx context.Context, itemCode int64, mask string) (*entity.ItemStandardCost, error) {
	row, err := r.q.GetItemStandardCost(ctx, itemCode, mask)
	if err != nil {
		return nil, fmt.Errorf("fetching standard cost for item %d mask %q: %w", itemCode, mask, err)
	}
	return isCostRowToEntity(row), nil
}

func (r *StandardCostRepositorySQLC) ListItemStandardCosts(ctx context.Context, itemCode int64) ([]*entity.ItemStandardCost, error) {
	rows, err := r.q.ListItemStandardCosts(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("listing standard costs for item %d: %w", itemCode, err)
	}
	out := make([]*entity.ItemStandardCost, 0, len(rows))
	for _, row := range rows {
		out = append(out, isCostRowToEntity(row))
	}
	return out, nil
}

// ─── work_center_costs ────────────────────────────────────────────────────────

func (r *StandardCostRepositorySQLC) UpsertWorkCenterCost(ctx context.Context, wcc *entity.WorkCenterCost) (*entity.WorkCenterCost, error) {
	row, err := r.q.UpsertWorkCenterCost(ctx, sqlc.UpsertWorkCenterCostParams{
		WorkCenterID:       wcc.WorkCenterID,
		CostPerHour:        pgutil.ToPgNumericFromFloat64(wcc.CostPerHour),
		MachineCostPerHour: pgutil.ToPgNumericFromFloat64(wcc.MachineCostPerHour),
		LaborCostPerHour:   pgutil.ToPgNumericFromFloat64(wcc.LaborCostPerHour),
		Currency:           orDefault(wcc.Currency, "BRL"),
		UpdatedBy:          pgutil.ToPgUUID(wcc.UpdatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("upserting work center cost: %w", err)
	}
	return wccRowToEntity(row), nil
}

func (r *StandardCostRepositorySQLC) GetWorkCenterCost(ctx context.Context, workCenterID int64) (*entity.WorkCenterCost, error) {
	row, err := r.q.GetWorkCenterCost(ctx, workCenterID)
	if err != nil {
		return nil, fmt.Errorf("fetching work center cost for CT %d: %w", workCenterID, err)
	}
	return wccRowToEntity(row), nil
}

func (r *StandardCostRepositorySQLC) ListWorkCenterCosts(ctx context.Context) ([]*entity.WorkCenterCost, error) {
	rows, err := r.q.ListWorkCenterCosts(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing work center costs: %w", err)
	}
	out := make([]*entity.WorkCenterCost, 0, len(rows))
	for _, row := range rows {
		out = append(out, wccRowToEntity(row))
	}
	return out, nil
}

// ─── item_purchase_costs ──────────────────────────────────────────────────────

func (r *StandardCostRepositorySQLC) UpsertItemPurchaseCost(ctx context.Context, ipc *entity.ItemPurchaseCost) (*entity.ItemPurchaseCost, error) {
	row, err := r.q.UpsertItemPurchaseCost(ctx, sqlc.UpsertItemPurchaseCostParams{
		ItemCode:  ipc.ItemCode,
		UnitCost:  pgutil.ToPgNumericFromFloat64(ipc.UnitCost),
		Currency:  orDefault(ipc.Currency, "BRL"),
		UpdatedBy: pgutil.ToPgUUID(ipc.UpdatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("upserting purchase cost for item %d: %w", ipc.ItemCode, err)
	}
	return ipcRowToEntity(row), nil
}

func (r *StandardCostRepositorySQLC) GetItemPurchaseCost(ctx context.Context, itemCode int64) (*entity.ItemPurchaseCost, error) {
	row, err := r.q.GetItemPurchaseCost(ctx, itemCode)
	if err != nil {
		return nil, fmt.Errorf("fetching purchase cost for item %d: %w", itemCode, err)
	}
	return ipcRowToEntity(row), nil
}

// ─── cost_rollup_log ──────────────────────────────────────────────────────────

func (r *StandardCostRepositorySQLC) InsertRollupLog(ctx context.Context, entry *entity.CostRollupLogEntry) error {
	_, err := r.q.InsertRollupLog(ctx, sqlc.InsertRollupLogParams{
		ItemCode:     entry.ItemCode,
		Mask:         entry.Mask,
		BOMLevel:     int32(entry.BOMLevel),
		MaterialCost: pgutil.ToPgNumericFromFloat64(entry.MaterialCost),
		LaborCost:    pgutil.ToPgNumericFromFloat64(entry.LaborCost),
		OverheadCost: pgutil.ToPgNumericFromFloat64(entry.OverheadCost),
	})
	return err
}

// ─── BOM helpers ──────────────────────────────────────────────────────────────

func (r *StandardCostRepositorySQLC) GetDirectChildren(ctx context.Context, parentCode int64) ([]domainrepo.BOMChild, error) {
	rows, err := r.q.GetAllDirectChildren(ctx, parentCode)
	if err != nil {
		return nil, fmt.Errorf("fetching BOM children for item %d: %w", parentCode, err)
	}
	out := make([]domainrepo.BOMChild, 0, len(rows))
	for _, row := range rows {
		if !row.IsActive {
			continue
		}
		out = append(out, domainrepo.BOMChild{
			ChildCode:      row.ChildCode,
			Quantity:       row.Quantity,
			LossPercentage: row.LossPercentage,
			IsCoproduct:    row.IsCoproduct,
			IsFixedQty:     row.IsFixedQty,
		})
	}
	return out, nil
}

func (r *StandardCostRepositorySQLC) GetRouteHoursByItem(ctx context.Context, itemCode int64, mask string) (float64, error) {
	return r.q.GetRouteHoursForItem(ctx, itemCode, mask)
}

// ─── mappers ──────────────────────────────────────────────────────────────────

func isCostRowToEntity(row sqlc.DBItemStandardCost) *entity.ItemStandardCost {
	return &entity.ItemStandardCost{
		ID:           row.ID,
		ItemCode:     row.ItemCode,
		Mask:         row.Mask,
		MaterialCost: pgutil.FromPgNumericToFloat64(row.MaterialCost),
		LaborCost:    pgutil.FromPgNumericToFloat64(row.LaborCost),
		OverheadCost: pgutil.FromPgNumericToFloat64(row.OverheadCost),
		TotalCost:    pgutil.FromPgNumericToFloat64(row.TotalCost),
		Currency:     row.Currency,
		CalculatedAt: pgutil.FromPgTimestamptz(row.CalculatedAt),
		CalculatedBy: pgutil.FromPgUUID(row.CalculatedBy),
	}
}

func wccRowToEntity(row sqlc.DBWorkCenterCost) *entity.WorkCenterCost {
	return &entity.WorkCenterCost{
		ID:                 row.ID,
		WorkCenterID:       row.WorkCenterID,
		CostPerHour:        pgutil.FromPgNumericToFloat64(row.CostPerHour),
		MachineCostPerHour: pgutil.FromPgNumericToFloat64(row.MachineCostPerHour),
		LaborCostPerHour:   pgutil.FromPgNumericToFloat64(row.LaborCostPerHour),
		Currency:           row.Currency,
		UpdatedAt:          pgutil.FromPgTimestamptz(row.UpdatedAt),
		UpdatedBy:          pgutil.FromPgUUID(row.UpdatedBy),
	}
}

func ipcRowToEntity(row sqlc.DBItemPurchaseCost) *entity.ItemPurchaseCost {
	return &entity.ItemPurchaseCost{
		ID:        row.ID,
		ItemCode:  row.ItemCode,
		UnitCost:  pgutil.FromPgNumericToFloat64(row.UnitCost),
		Currency:  row.Currency,
		UpdatedAt: pgutil.FromPgTimestamptz(row.UpdatedAt),
		UpdatedBy: pgutil.FromPgUUID(row.UpdatedBy),
	}
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
