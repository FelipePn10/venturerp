package mrp_calculation

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
	orderpriority "github.com/FelipePn10/panossoerp/internal/domain/order_priority/entity"
	salesdivision "github.com/FelipePn10/panossoerp/internal/domain/sales_division/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

func (r *MRPCalculationRepositorySQLC) ListOpenSalesOrderDemands(ctx context.Context, planCode int64, salesOrderItemCode *int64) ([]*entity.MRPInput, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.q.ListOpenSalesOrderDemands(ctx, sqlc.ListOpenSalesOrderDemandsParams{
		EnterpriseID: enterpriseID, PlanCode: planCode, SalesOrderItemCode: salesOrderItemCode,
	})
	if err != nil {
		return nil, fmt.Errorf("listing open sales-order demands: %w", err)
	}
	result := make([]*entity.MRPInput, 0, len(rows))
	for _, row := range rows {
		sourceCode := row.SourceCode
		var sourceEnterpriseCode *int64
		if row.InterFactory {
			code := row.SourceEnterpriseCode
			sourceEnterpriseCode = &code
		}
		result = append(result, &entity.MRPInput{
			ItemCode: row.ItemCode, Mask: row.Mask,
			Quantity: pgutil.FromPgNumericToFloat64(row.Quantity),
			NeedDate: row.NeedDate.Time, DemandType: "SALES_ORDER", SourceCode: &sourceCode,
			WarehouseCode: row.WarehouseCode, TechnicalAssistance: row.TechnicalAssistance,
			InterFactory: row.InterFactory, SourceEnterpriseCode: sourceEnterpriseCode, AutoRelease: row.AutoRelease,
		})
	}
	return result, nil
}

func (r *MRPCalculationRepositorySQLC) ResolveClassificationItemCodes(ctx context.Context, classification string, classCodes []string) ([]int64, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	items, err := r.q.ResolveClassificationItemCodes(ctx, sqlc.ResolveClassificationItemCodesParams{
		EnterpriseID: enterpriseID, Classification: classification, ClassCodes: classCodes,
	})
	if err != nil {
		return nil, fmt.Errorf("resolving classification items: %w", err)
	}
	return items, nil
}

// ---------- Planning Params ----------

func (r *MRPCalculationRepositorySQLC) LoadTypedPlanningParams(ctx context.Context) (*entity.TypedPlanningParams, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx,
		`SELECT param_number, value FROM planning_params WHERE enterprise_id = $1 ORDER BY param_number`, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("loading typed planning params: %w", err)
	}
	defer rows.Close()

	raw := make(map[int]string)
	for rows.Next() {
		var code int
		var value string
		if err := rows.Scan(&code, &value); err != nil {
			return nil, fmt.Errorf("scanning planning param: %w", err)
		}
		raw[code] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating planning params: %w", err)
	}

	params := &entity.TypedPlanningParams{}
	params.LoadFromDB(raw)
	return params, nil
}

// ---------- Order Priorities ----------

func (r *MRPCalculationRepositorySQLC) ListAllOrderPriorities(ctx context.Context) ([]*orderpriority.OrderPriority, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx,
		`SELECT code, interval_start, interval_end, priority, description, is_active, created_at, updated_at, created_by
		 FROM order_priorities WHERE enterprise_id = $1 AND is_active = true ORDER BY interval_start`, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing order priorities: %w", err)
	}
	defer rows.Close()

	var result []*orderpriority.OrderPriority
	for rows.Next() {
		var op orderpriority.OrderPriority
		if err := rows.Scan(
			&op.Code, &op.IntervalStart, &op.IntervalEnd, &op.Priority,
			&op.Description, &op.IsActive, &op.CreatedAt, &op.UpdatedAt, &op.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scanning order priority: %w", err)
		}
		result = append(result, &op)
	}
	return result, rows.Err()
}

// ---------- Machine Times ----------

func (r *MRPCalculationRepositorySQLC) ListItemMachineTimes(ctx context.Context, itemCodes []int64) (map[int64][]*entity.MachineTimeInfo, error) {
	if len(itemCodes) == 0 {
		return make(map[int64][]*entity.MachineTimeInfo), nil
	}

	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx,
		`SELECT item_code, machine_id, priority, production_time
		 FROM item_machine_times
		 WHERE item_code = ANY($1) AND enterprise_id = $2 AND is_active = true
		 ORDER BY priority`,
		itemCodes, enterpriseID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing item machine times: %w", err)
	}
	defer rows.Close()

	m := make(map[int64][]*entity.MachineTimeInfo)
	for rows.Next() {
		var info entity.MachineTimeInfo
		if err := rows.Scan(&info.ItemCode, &info.MachineID, &info.Priority, &info.ProductionTime); err != nil {
			return nil, fmt.Errorf("scanning machine time: %w", err)
		}
		m[info.ItemCode] = append(m[info.ItemCode], &info)
	}
	return m, rows.Err()
}

// ---------- Kanban Cards ----------

func (r *MRPCalculationRepositorySQLC) ListKanbanCards(ctx context.Context) ([]*entity.KanbanCardInfo, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx,
		`SELECT code, item_code, reorder_point, quantity_per_card, card_count, is_active
		 FROM kanban_cards WHERE enterprise_id = $1 AND is_active = true ORDER BY item_code`, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing kanban cards: %w", err)
	}
	defer rows.Close()

	var result []*entity.KanbanCardInfo
	for rows.Next() {
		var k entity.KanbanCardInfo
		if err := rows.Scan(&k.CardCode, &k.ItemCode, &k.ReorderPoint,
			&k.QuantityPerCard, &k.CardCount, &k.IsActive); err != nil {
			return nil, fmt.Errorf("scanning kanban card: %w", err)
		}
		result = append(result, &k)
	}
	return result, rows.Err()
}

// ---------- MPS Items ----------

func (r *MRPCalculationRepositorySQLC) ListMPSItems(ctx context.Context, planCode int64) ([]*entity.MPSItemInfo, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx,
		`SELECT item_code, mask, period_type, period_value, year, quantity, is_firm
		 FROM mps_schedule WHERE enterprise_id = $1 AND is_firm = false
		 ORDER BY year, period_type, period_value`, enterpriseID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing MPS items: %w", err)
	}
	defer rows.Close()

	var result []*entity.MPSItemInfo
	for rows.Next() {
		var m entity.MPSItemInfo
		if err := rows.Scan(&m.ItemCode, &m.Mask, &m.PeriodType, &m.PeriodValue, &m.Year, &m.Quantity, &m.IsFirm); err != nil {
			return nil, fmt.Errorf("scanning MPS item: %w", err)
		}
		result = append(result, &m)
	}
	return result, rows.Err()
}

// ---------- Item Planning Extras ----------

func (r *MRPCalculationRepositorySQLC) ListItemPlanningExtras(ctx context.Context) (map[int64]*entity.ItemPlanningExtra, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx,
		`SELECT item_code, maximum_stock, safety_time, coverage, grouping_key, is_critical, use_tank_date
		 FROM item_planning_extras WHERE enterprise_id = $1 AND is_active = true`, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing item planning extras: %w", err)
	}
	defer rows.Close()

	m := make(map[int64]*entity.ItemPlanningExtra)
	for rows.Next() {
		var e entity.ItemPlanningExtra
		if err := rows.Scan(&e.ItemCode, &e.MaximumStock, &e.SafetyTime, &e.Coverage,
			&e.GroupingKey, &e.IsCritical, &e.UseTankDate); err != nil {
			return nil, fmt.Errorf("scanning item planning extra: %w", err)
		}
		m[e.ItemCode] = &e
	}
	return m, rows.Err()
}

// ---------- Machine Schedule ----------

func (r *MRPCalculationRepositorySQLC) CreateMachineSchedule(ctx context.Context, schedule *entity.MachineScheduleInfo) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`INSERT INTO machine_schedules (plan_code, planned_order_code, machine_id, schedule_date, production_time, enterprise_id)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		schedule.PlanCode, schedule.PlannedOrderCode, schedule.MachineID,
		schedule.ScheduleDate, schedule.ProductionTime, enterpriseID,
	)
	return err
}

// ---------- Item Sales Divisions ----------

func (r *MRPCalculationRepositorySQLC) ListItemSalesDivisions(ctx context.Context, itemCodes []int64) (map[int64]*salesdivision.SalesDivision, error) {
	if len(itemCodes) == 0 {
		return make(map[int64]*salesdivision.SalesDivision), nil
	}

	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT ON (sd.code)
		        sd.code, sd.description, sd.commercial_analysis, sd.financial_analysis,
		        sd.is_technical_assistance, sd.consider_delivery_promise, sd.consider_mrp,
		        sd.allow_outside_limits, sd.minimum_delivery_days, sd.financial_delay_days,
		        sd.pis_percentage, sd.cofins_percentage, sd.parent_division_id, sd.is_active,
		        sd.created_at, sd.updated_at, sd.created_by
		 FROM sales_divisions sd
		 INNER JOIN item_sales_divisions isd ON isd.sales_division_code = sd.code
		 WHERE isd.item_code = ANY($1) AND sd.enterprise_id = $2 AND sd.is_active = true`,
		itemCodes, enterpriseID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing item sales divisions: %w", err)
	}
	defer rows.Close()

	m := make(map[int64]*salesdivision.SalesDivision)
	for rows.Next() {
		var sd salesdivision.SalesDivision
		var createdAt, updatedAt time.Time
		if err := rows.Scan(
			&sd.Code, &sd.Description, &sd.CommercialAnalysis, &sd.FinancialAnalysis,
			&sd.IsTechnicalAssistance, &sd.ConsiderDeliveryPromise, &sd.ConsiderMRP,
			&sd.AllowOutsideLimits, &sd.MinimumDeliveryDays, &sd.FinancialDelayDays,
			&sd.PISPercentage, &sd.CofinsPercentage, &sd.ParentDivisionID, &sd.IsActive,
			&createdAt, &updatedAt, &sd.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scanning sales division: %w", err)
		}
		sd.CreatedAt = createdAt
		sd.UpdatedAt = updatedAt
		m[sd.Code] = &sd
	}
	return m, rows.Err()
}

// ---------- Update Planned Order Priority ----------

func (r *MRPCalculationRepositorySQLC) UpdatePlannedOrderPriority(ctx context.Context, suggestionCode int64, priority string) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`UPDATE mrp_planned_suggestions SET priority = $2 WHERE code = $1 AND enterprise_id = $3`,
		suggestionCode, priority, enterpriseID,
	)
	return err
}

// ---------- Update Planned Order Machine ----------

func (r *MRPCalculationRepositorySQLC) UpdatePlannedOrderMachine(ctx context.Context, suggestionCode int64, machineID int64, productionTime float64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`UPDATE mrp_planned_suggestions SET machine_id = $2, production_time = $3 WHERE code = $1 AND enterprise_id = $4`,
		suggestionCode, machineID, productionTime, enterpriseID,
	)
	return err
}
