package planned_order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PlannedOrderRepositorySQLC struct {
	q *sqlc.Queries
}

func NewPlannedOrderRepositorySQLC(
	q *sqlc.Queries,
) *PlannedOrderRepositorySQLC {

	return &PlannedOrderRepositorySQLC{
		q: q,
	}
}

func (r *PlannedOrderRepositorySQLC) Create(
	ctx context.Context,
	o *entity.PlannedOrder,
) (*entity.PlannedOrder, error) {

	mask := ""
	if o.Mask != nil {
		mask = *o.Mask
	}

	var planCode *int64 = o.PlanCode

	var demandCode int32
	if o.DemandCode != nil {
		demandCode = int32(*o.DemandCode)
	}

	var costCenterCode *int64 = o.CostCenterCode
	var employeeCode *int64 = o.EmployeeCode
	var machineCode *int64 = o.MachineCode
	var parentOrderCode *int64 = o.ParentOrderCode
	var salesOrderCode *int64 = o.SalesOrderCode

	row, err := r.q.CreatePlannedOrder(
		ctx,
		sqlc.CreatePlannedOrderParams{
			OrderNumber:       o.OrderNumber,
			ItemCode:          o.ItemCode,
			Mask:              mask,
			Quantity:          pgutil.ToPgNumericFromFloat64(o.Quantity),
			QuantityLoss:      pgutil.ToPgNumericFromFloat64(o.QuantityLoss),
			QuantityCorrected: pgutil.ToPgNumericFromFloat64(o.QuantityCorrected),
			OrderType:         sqlc.OrderTypeEnum(o.OrderType),
			Status:            sqlc.OrderStatusEnum(o.Status),
			PlanCode:          planCode,
			DemandType:        sqlc.DemandTypeEnum(o.DemandType),
			DemandCode:        demandCode,
			NeedDate:          pgutil.ToPgDate(o.NeedDate),
			StartDate:         toPgDatePtr(o.StartDate),
			EndDate:           toPgDatePtr(o.EndDate),
			CostCenterCode:    costCenterCode,
			EmployeeCode:      employeeCode,
			MachineCode:       machineCode,
			ProductionTime:    pgutil.ToPgNumericFromFloat64(o.ProductionTime),
			Priority:          pgutil.ToPgTextFromPtr(o.Priority),
			Llc:               int32(o.LLC),
			Notes:             pgutil.ToPgTextFromPtr(o.Notes),
			ParentOrderCode:   parentOrderCode,
			SalesOrderCode:    salesOrderCode,
			CreatedBy:         pgutil.ToPgUUID(o.CreatedBy),
		},
	)

	if err != nil {
		return nil, fmt.Errorf("creating planned order: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *PlannedOrderRepositorySQLC) GetByCode(
	ctx context.Context,
	code int64,
) (*entity.PlannedOrder, error) {

	row, err := r.q.GetPlannedOrderByCode(ctx, code)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("planned order %d not found", code)
		}

		return nil, fmt.Errorf("fetching planned order: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *PlannedOrderRepositorySQLC) GetByNumber(
	ctx context.Context,
	number int64,
) (*entity.PlannedOrder, error) {

	row, err := r.q.GetPlannedOrderByNumber(ctx, number)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("planned order number %d not found", number)
		}

		return nil, fmt.Errorf("fetching planned order by number: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *PlannedOrderRepositorySQLC) List(
	ctx context.Context,
) ([]*entity.PlannedOrder, error) {

	rows, err := r.q.ListPlannedOrders(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing planned orders: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *PlannedOrderRepositorySQLC) ListByPlan(
	ctx context.Context,
	planCode int64,
) ([]*entity.PlannedOrder, error) {

	rows, err := r.q.ListPlannedOrdersByPlan(ctx, &planCode)

	if err != nil {
		return nil, fmt.Errorf("listing planned orders by plan: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *PlannedOrderRepositorySQLC) ListByItem(
	ctx context.Context,
	itemCode int64,
) ([]*entity.PlannedOrder, error) {

	rows, err := r.q.ListPlannedOrdersByItem(ctx, itemCode)

	if err != nil {
		return nil, fmt.Errorf("listing planned orders by item: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *PlannedOrderRepositorySQLC) ListByType(
	ctx context.Context,
	orderType string,
) ([]*entity.PlannedOrder, error) {

	rows, err := r.q.ListPlannedOrdersByType(
		ctx,
		sqlc.OrderTypeEnum(orderType),
	)

	if err != nil {
		return nil, fmt.Errorf("listing planned orders by type: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *PlannedOrderRepositorySQLC) ListByStatus(
	ctx context.Context,
	status string,
) ([]*entity.PlannedOrder, error) {

	rows, err := r.q.ListPlannedOrdersByStatus(
		ctx,
		sqlc.OrderStatusEnum(status),
	)

	if err != nil {
		return nil, fmt.Errorf("listing planned orders by status: %w", err)
	}

	return rowsToEntities(rows), nil
}

func (r *PlannedOrderRepositorySQLC) UpdateStatus(
	ctx context.Context,
	code int64,
	status string,
) (*entity.PlannedOrder, error) {

	row, err := r.q.UpdatePlannedOrderStatus(
		ctx,
		sqlc.UpdatePlannedOrderStatusParams{
			Status: sqlc.OrderStatusEnum(status),
			Code:   code,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("updating planned order status: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *PlannedOrderRepositorySQLC) FirmOrder(
	ctx context.Context,
	code int64,
) (*entity.PlannedOrder, error) {

	row, err := r.q.FirmPlannedOrder(ctx, code)

	if err != nil {
		return nil, fmt.Errorf("firming planned order: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *PlannedOrderRepositorySQLC) UpdateDates(
	ctx context.Context,
	code int64,
	start,
	end *time.Time,
) (*entity.PlannedOrder, error) {

	row, err := r.q.UpdatePlannedOrderDates(
		ctx,
		sqlc.UpdatePlannedOrderDatesParams{
			StartDate: toPgDatePtr(start),
			EndDate:   toPgDatePtr(end),
			Code:      code,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("updating planned order dates: %w", err)
	}

	return rowToEntity(row), nil
}

func (r *PlannedOrderRepositorySQLC) Delete(
	ctx context.Context,
	code int64,
) error {

	return r.q.DeletePlannedOrder(ctx, code)
}

func (r *PlannedOrderRepositorySQLC) DeleteByPlan(
	ctx context.Context,
	planCode int64,
) error {

	return r.q.DeleteOrdersByPlan(ctx, &planCode)
}

func (r *PlannedOrderRepositorySQLC) GetNextOrderNumber(
	ctx context.Context,
) (int64, error) {

	result, err := r.q.GetNextOrderNumber(ctx)
	if err != nil {
		return 1, nil
	}

	return int64(result), nil
}

func rowToEntity(
	row sqlc.PlannedOrder,
) *entity.PlannedOrder {

	e := &entity.PlannedOrder{
		Code:              row.Code,
		OrderNumber:       row.OrderNumber,
		ItemCode:          row.ItemCode,
		Quantity:          pgutil.FromPgNumericToFloat64(row.Quantity),
		QuantityLoss:      pgutil.FromPgNumericToFloat64(row.QuantityLoss),
		QuantityCorrected: pgutil.FromPgNumericToFloat64(row.QuantityCorrected),
		OrderType:         types.OrderType(row.OrderType),
		Status:            types.OrderStatus(row.Status),
		DemandType:        types.DemandType(row.DemandType),
		NeedDate:          pgutil.FromPgDate(row.NeedDate),
		ProductionTime:    pgutil.FromPgNumericToFloat64(row.ProductionTime),
		LLC:               int(row.Llc),
		IsFirm:            row.IsFirm,
		IsActive:          row.IsActive,
		CreatedAt:         pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:         pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:         pgutil.FromPgUUID(row.CreatedBy),
	}

	if row.Mask != "" {
		v := row.Mask
		e.Mask = &v
	}

	if row.PlanCode != nil {
		v := *row.PlanCode
		e.PlanCode = &v
	}

	if row.DemandCode != 0 {
		v := int64(row.DemandCode)
		e.DemandCode = &v
	}

	e.StartDate = fromPgDatePtr(row.StartDate)
	e.EndDate = fromPgDatePtr(row.EndDate)

	if row.CostCenterCode != nil {
		v := *row.CostCenterCode
		e.CostCenterCode = &v
	}

	if row.EmployeeCode != nil {
		v := *row.EmployeeCode
		e.EmployeeCode = &v
	}

	if row.MachineCode != nil {
		v := *row.MachineCode
		e.MachineCode = &v
	}

	if row.Priority.Valid {
		v := row.Priority.String
		e.Priority = &v
	}

	if row.Notes.Valid {
		v := row.Notes.String
		e.Notes = &v
	}

	if row.ParentOrderCode != nil {
		v := *row.ParentOrderCode
		e.ParentOrderCode = &v
	}

	if row.SalesOrderCode != nil {
		v := *row.SalesOrderCode
		e.SalesOrderCode = &v
	}

	return e
}

func rowsToEntities(
	rows []sqlc.PlannedOrder,
) []*entity.PlannedOrder {

	out := make([]*entity.PlannedOrder, 0, len(rows))

	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}

	return out
}

func toPgDatePtr(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{
			Valid: false,
		}
	}

	return pgtype.Date{
		Time:  *t,
		Valid: true,
	}
}

func fromPgDatePtr(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}

	v := d.Time
	return &v
}
