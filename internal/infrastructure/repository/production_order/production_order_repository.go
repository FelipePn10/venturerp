package production_order

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	stockrepository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type ProductionOrderRepositoryPGX struct {
	pool *pgxpool.Pool
}

func NewProductionOrderRepositoryPGX(pool *pgxpool.Pool) *ProductionOrderRepositoryPGX {
	return &ProductionOrderRepositoryPGX{pool: pool}
}

func (r *ProductionOrderRepositoryPGX) Create(ctx context.Context, o *entity.ProductionOrder) (*entity.ProductionOrder, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx,
		`INSERT INTO public.production_orders
			(order_number, planned_order_id, item_code, mask, planned_qty, produced_qty, scrapped_qty,
			 status, start_date, end_date, machine_id, cost_center_id, employee_id, priority, notes,
			 is_active, created_by, warehouse_id, enterprise_id, origin_type)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,
		 CASE WHEN $2::bigint IS NULL THEN 'MANUAL' WHEN EXISTS(SELECT 1 FROM kanban_cards k WHERE k.item_code=$3 AND k.enterprise_id=$19) THEN 'KANBAN' ELSE 'MRP' END)
		 RETURNING id, order_number, planned_order_id, item_code, mask, planned_qty, produced_qty, scrapped_qty,
		           status, start_date, end_date, machine_id, cost_center_id, employee_id, priority, notes,
		           is_active, created_at, updated_at, created_by, warehouse_id`,
		o.OrderNumber, o.PlannedOrderID, o.ItemCode, o.Mask,
		pgutil.ToPgNumericFromFloat64(o.PlannedQty),
		pgutil.ToPgNumericFromFloat64(o.ProducedQty),
		pgutil.ToPgNumericFromFloat64(o.ScrappedQty),
		string(o.Status),
		pgDatePtr(o.StartDate), pgDatePtr(o.EndDate),
		o.MachineID, o.CostCenterID, o.EmployeeID,
		o.Priority, o.Notes, o.IsActive, pgutil.ToPgUUID(o.CreatedBy), o.WarehouseID, enterpriseID,
	)
	return scanProductionOrder(row)
}

func (r *ProductionOrderRepositoryPGX) Update(ctx context.Context, o *entity.ProductionOrder) (*entity.ProductionOrder, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx,
		`UPDATE public.production_orders
		 SET planned_order_id=$2, item_code=$3, mask=$4, planned_qty=$5, produced_qty=$6, scrapped_qty=$7,
		     status=$8, start_date=$9, end_date=$10, machine_id=$11, cost_center_id=$12, employee_id=$13,
		     priority=$14, notes=$15, is_active=$16, warehouse_id=$17, updated_at=NOW()
		 WHERE id=$1 AND enterprise_id=$18
		 RETURNING id, order_number, planned_order_id, item_code, mask, planned_qty, produced_qty, scrapped_qty,
		           status, start_date, end_date, machine_id, cost_center_id, employee_id, priority, notes,
		           is_active, created_at, updated_at, created_by, warehouse_id`,
		o.ID, o.PlannedOrderID, o.ItemCode, o.Mask,
		pgutil.ToPgNumericFromFloat64(o.PlannedQty),
		pgutil.ToPgNumericFromFloat64(o.ProducedQty),
		pgutil.ToPgNumericFromFloat64(o.ScrappedQty),
		string(o.Status),
		pgDatePtr(o.StartDate), pgDatePtr(o.EndDate),
		o.MachineID, o.CostCenterID, o.EmployeeID,
		o.Priority, o.Notes, o.IsActive, o.WarehouseID, enterpriseID,
	)
	return scanProductionOrder(row)
}

func (r *ProductionOrderRepositoryPGX) GetByCode(ctx context.Context, id int64) (*entity.ProductionOrder, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx,
		`SELECT id, order_number, planned_order_id, item_code, mask, planned_qty, produced_qty, scrapped_qty,
		        status, start_date, end_date, machine_id, cost_center_id, employee_id, priority, notes,
		        is_active, created_at, updated_at, created_by, warehouse_id
		 FROM public.production_orders WHERE id=$1 AND enterprise_id=$2`, id, enterpriseID)
	return scanProductionOrder(row)
}

func (r *ProductionOrderRepositoryPGX) GetDeliveryByIdempotencyKey(ctx context.Context, key string) (*entity.ProductionDelivery, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var d entity.ProductionDelivery
	err = r.pool.QueryRow(ctx, `SELECT id, production_order_id, quantity, movement_class, warehouse_id, lot, is_final, delivered_at, created_by
		FROM production_deliveries WHERE enterprise_id=$1 AND idempotency_key=$2`, enterpriseID, key).Scan(
		&d.ID, &d.ProductionOrderID, &d.Quantity, &d.MovementClass, &d.WarehouseID, &d.Lot, &d.IsFinal, &d.DeliveredAt, &d.CreatedBy)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *ProductionOrderRepositoryPGX) RegisterDelivery(ctx context.Context, d *entity.ProductionDelivery) (*entity.ProductionOrder, error) {
	return r.RegisterDeliveryWithMovements(ctx, d, nil)
}

// RegisterDeliveryWithMovements settles the delivery header, production order,
// stock movements and balance snapshots in one database transaction.
func (r *ProductionOrderRepositoryPGX) RegisterDeliveryWithMovements(ctx context.Context, d *entity.ProductionDelivery, movements []*stockentity.StockMovement) (*entity.ProductionOrder, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var deliveryID int64
	err = tx.QueryRow(ctx, `INSERT INTO production_deliveries
		(production_order_id, enterprise_id, idempotency_key, quantity, movement_class, warehouse_id, lot, is_final, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`, d.ProductionOrderID, enterpriseID, d.IdempotencyKey, d.Quantity, d.MovementClass, d.WarehouseID, d.Lot, d.IsFinal, d.CreatedBy).Scan(&deliveryID)
	if err != nil {
		return nil, fmt.Errorf("registering production delivery: %w", err)
	}
	lines := d.Lines
	if len(lines) == 0 && d.Quantity.IsPositive() {
		lines = []entity.ProductionDeliveryLine{{MovementClass: d.MovementClass, Quantity: d.Quantity}}
	}
	for _, line := range lines {
		if _, err := tx.Exec(ctx, `INSERT INTO production_delivery_lines
			(production_delivery_id, movement_class, quantity) VALUES ($1,$2,$3)`,
			deliveryID, line.MovementClass, line.Quantity); err != nil {
			return nil, fmt.Errorf("registering production delivery line: %w", err)
		}
	}
	for _, movement := range movements {
		if err := stockrepository.CreateMovementTx(ctx, tx, enterpriseID, movement); err != nil {
			return nil, err
		}
	}
	status := "IN_PROGRESS"
	if d.IsFinal {
		status = "COMPLETED"
	}
	row := tx.QueryRow(ctx, `UPDATE production_orders SET produced_qty=GREATEST(produced_qty,
		(SELECT COALESCE(SUM(quantity),0) FROM production_deliveries WHERE enterprise_id=$4 AND production_order_id=$3)), status=$1,
		end_date=CASE WHEN $2 THEN CURRENT_DATE ELSE end_date END, updated_at=NOW()
		WHERE id=$3 AND enterprise_id=$4
		RETURNING id, order_number, planned_order_id, item_code, mask, planned_qty, produced_qty, scrapped_qty,
		status, start_date, end_date, machine_id, cost_center_id, employee_id, priority, notes,
		is_active, created_at, updated_at, created_by, warehouse_id`, status, d.IsFinal, d.ProductionOrderID, enterpriseID)
	order, err := scanProductionOrder(row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return order, nil
}

func (r *ProductionOrderRepositoryPGX) HasPendingServicePurchaseOrders(ctx context.Context, productionOrderID int64) (bool, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var pending bool
	err = r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM production_order_service_links link
		JOIN purchase_orders po ON po.code=link.purchase_order_code
		WHERE link.enterprise_id=$1 AND link.production_order_id=$2
		AND po.status NOT IN ('RECEIVED','CANCELLED'))`, enterpriseID, productionOrderID).Scan(&pending)
	return pending, err
}

func (r *ProductionOrderRepositoryPGX) LinkServiceRequisition(ctx context.Context, productionOrderID, requisitionCode int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	command, err := r.pool.Exec(ctx, `INSERT INTO production_order_service_requisition_links
		(production_order_id,purchase_requisition_code,enterprise_id)
		SELECT po.id,$2,$3 FROM production_orders po
		JOIN enterprise e ON e.id=$3
		JOIN purchase_requisitions pr ON pr.code=$2 AND pr.enterprise_code=e.code
		WHERE po.id=$1 AND po.enterprise_id=$3
		ON CONFLICT DO NOTHING`, productionOrderID, requisitionCode, enterpriseID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		var exists bool
		err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM production_order_service_requisition_links
			WHERE enterprise_id=$1 AND production_order_id=$2 AND purchase_requisition_code=$3)`, enterpriseID, productionOrderID, requisitionCode).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return pgx.ErrNoRows
		}
	}
	return nil
}

func (r *ProductionOrderRepositoryPGX) CurrentEnterpriseCode(ctx context.Context) (int64, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	var code int64
	err = r.pool.QueryRow(ctx, `SELECT code FROM enterprise WHERE id=$1`, enterpriseID).Scan(&code)
	return code, err
}

func (r *ProductionOrderRepositoryPGX) ValidateProductionRelease(ctx context.Context, itemCode int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var enabled bool
	_ = r.pool.QueryRow(ctx, `SELECT UPPER(value) IN ('S','SIM','1','TRUE','YES') FROM planning_params
		WHERE enterprise_id=$1 AND (param_number=45 OR param_key='OBRIGAR_CONTROLE_ESTOQUE_TERCEIROS') LIMIT 1`, enterpriseID).Scan(&enabled)
	if !enabled {
		return nil
	}
	var reporting, issue string
	if err := r.pool.QueryRow(ctx, `SELECT production_reporting_type,material_issue_timing FROM items
		WHERE code=$1`, itemCode).Scan(&reporting, &issue); err != nil {
		return err
	}
	if reporting == "ORDER" && issue == "REGISTRATION_RELEASE" {
		return fmt.Errorf("parameter 45 blocks release: order reporting with issue at registration/release")
	}
	rows, err := r.pool.Query(ctx, `SELECT operation.origin::text,
		COALESCE(route_operation.third_party_remittance,operation.third_party_remittance)
		FROM manufacturing_routes route
		JOIN route_operations route_operation ON route_operation.route_id=route.id AND route_operation.is_active
		JOIN operations operation ON operation.id=route_operation.operation_id AND operation.is_active
		WHERE route.item_code=$1 AND route.is_active AND route.is_standard`, itemCode)
	if err != nil {
		return err
	}
	defer rows.Close()
	total, external := 0, 0
	invalidRemittance := false
	for rows.Next() {
		var origin, remittance string
		if err := rows.Scan(&origin, &remittance); err != nil {
			return err
		}
		total++
		if origin == "EXTERNA" || origin == "TERCEIROS" {
			external++
			invalidRemittance = invalidRemittance || remittance != "DEMAND_ITEMS"
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if external > 0 && total > external {
		return fmt.Errorf("parameter 45 blocks release: third-party operations cannot coexist with other operations")
	}
	if invalidRemittance {
		return fmt.Errorf("parameter 45 blocks release: third-party remittance must use demand items")
	}
	return nil
}

func (r *ProductionOrderRepositoryPGX) LinkServicePurchaseOrder(ctx context.Context, requisitionItemID, purchaseOrderCode int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	command, err := r.pool.Exec(ctx, `INSERT INTO production_order_service_links
		(production_order_id,purchase_order_code,enterprise_id)
		SELECT link.production_order_id,$2,$3
		FROM purchase_requisition_items item
		JOIN production_order_service_requisition_links link
		  ON link.purchase_requisition_code=item.requisition_code AND link.enterprise_id=$3
		JOIN enterprise e ON e.id=$3
		JOIN purchase_orders po ON po.code=$2 AND po.enterprise_code=e.code
		WHERE item.id=$1
		ON CONFLICT DO NOTHING`, requisitionItemID, purchaseOrderCode, enterpriseID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		var exists bool
		err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM purchase_requisition_items item
			JOIN production_order_service_requisition_links req_link ON req_link.purchase_requisition_code=item.requisition_code AND req_link.enterprise_id=$1
			JOIN production_order_service_links po_link ON po_link.production_order_id=req_link.production_order_id AND po_link.enterprise_id=$1 AND po_link.purchase_order_code=$3
			WHERE item.id=$2)`, enterpriseID, requisitionItemID, purchaseOrderCode).Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			return pgx.ErrNoRows
		}
	}
	return nil
}

func (r *ProductionOrderRepositoryPGX) TreatProductionExcess(ctx context.Context) (bool, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var value string
	err = r.pool.QueryRow(ctx, `SELECT value FROM planning_params WHERE enterprise_id=$1 AND param_key='production_excess_treatment'`, enterpriseID).Scan(&value)
	if err != nil {
		return false, nil
	}
	switch value {
	case "S", "SIM", "1", "TRUE", "true":
		return true, nil
	}
	return false, nil
}

func (r *ProductionOrderRepositoryPGX) GetDeliveredQuantity(ctx context.Context, productionOrderID int64) (decimal.Decimal, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return decimal.Zero, err
	}
	var quantity decimal.Decimal
	err = r.pool.QueryRow(ctx, `SELECT COALESCE(SUM(quantity),0) FROM production_deliveries WHERE enterprise_id=$1 AND production_order_id=$2`, enterpriseID, productionOrderID).Scan(&quantity)
	return quantity, err
}

func (r *ProductionOrderRepositoryPGX) ListDeliveries(ctx context.Context, productionOrderID int64) ([]*entity.ProductionDelivery, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT id, production_order_id, idempotency_key, quantity, movement_class,
		warehouse_id, lot, is_final, delivered_at, created_by FROM production_deliveries
		WHERE enterprise_id=$1 AND production_order_id=$2 ORDER BY delivered_at,id`, enterpriseID, productionOrderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*entity.ProductionDelivery
	for rows.Next() {
		d := &entity.ProductionDelivery{}
		if err := rows.Scan(&d.ID, &d.ProductionOrderID, &d.IdempotencyKey, &d.Quantity, &d.MovementClass,
			&d.WarehouseID, &d.Lot, &d.IsFinal, &d.DeliveredAt, &d.CreatedBy); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (r *ProductionOrderRepositoryPGX) GetItemAutomaticIssue(ctx context.Context, itemCode int64) (bool, int64, error) {
	var automatic bool
	var warehouseID int64
	err := r.pool.QueryRow(ctx, `SELECT warehouse_automatic_low, warehouse_code FROM items WHERE code=$1`, itemCode).Scan(&automatic, &warehouseID)
	return automatic, warehouseID, err
}

func (r *ProductionOrderRepositoryPGX) List(ctx context.Context) ([]*entity.ProductionOrder, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, order_number, planned_order_id, item_code, mask, planned_qty, produced_qty, scrapped_qty,
		        status, start_date, end_date, machine_id, cost_center_id, employee_id, priority, notes,
		        is_active, created_at, updated_at, created_by, warehouse_id
		 FROM public.production_orders WHERE enterprise_id=$1 ORDER BY id DESC`, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing production orders: %w", err)
	}
	defer rows.Close()

	var orders []*entity.ProductionOrder
	for rows.Next() {
		o, err := scanProductionOrderFromRows(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (r *ProductionOrderRepositoryPGX) CreateFromPlannedOrder(ctx context.Context, plannedOrderID int64) (*entity.ProductionOrder, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	nextNum, _ := r.GetNextOrderNumber(ctx)

	row := r.pool.QueryRow(ctx,
		`INSERT INTO public.production_orders
			(order_number, planned_order_id, item_code, mask, planned_qty, status, is_active, created_by, warehouse_id, enterprise_id, origin_type)
		 SELECT $1, po.id, po.item_code, COALESCE(po.mask, ''), po.quantity, 'OPEN', TRUE, po.created_by, po.warehouse_code, po.enterprise_id,
		 CASE WHEN EXISTS(SELECT 1 FROM kanban_cards k WHERE k.item_code=po.item_code AND k.enterprise_id=po.enterprise_id) THEN 'KANBAN' ELSE 'MRP' END
		 FROM public.planned_orders po WHERE po.id = $2 AND po.enterprise_id=$3
		 RETURNING id, order_number, planned_order_id, item_code, mask, planned_qty, produced_qty, scrapped_qty,
		           status, start_date, end_date, machine_id, cost_center_id, employee_id, priority, notes,
		           is_active, created_at, updated_at, created_by, warehouse_id`,
		nextNum, plannedOrderID, enterpriseID,
	)
	return scanProductionOrder(row)
}

func (r *ProductionOrderRepositoryPGX) Start(ctx context.Context, id int64, startDate time.Time) (*entity.ProductionOrder, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx,
		`UPDATE public.production_orders
		 SET status='IN_PROGRESS', start_date=$2, updated_at=NOW()
		 WHERE id=$1 AND enterprise_id=$3
		 RETURNING id, order_number, planned_order_id, item_code, mask, planned_qty, produced_qty, scrapped_qty,
		           status, start_date, end_date, machine_id, cost_center_id, employee_id, priority, notes,
		           is_active, created_at, updated_at, created_by, warehouse_id`,
		id, startDate, enterpriseID,
	)
	return scanProductionOrder(row)
}

func (r *ProductionOrderRepositoryPGX) AddAppointment(ctx context.Context, a *entity.ProductionAppointment) (*entity.ProductionAppointment, error) {
	if _, err := r.GetByCode(ctx, a.ProductionOrderID); err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx,
		`INSERT INTO public.production_appointments
			(production_order_id, machine_id, employee_id, appointment_date,
			 start_time, end_time, produced_qty, scrapped_qty, scrap_reason, notes, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 RETURNING id, production_order_id, machine_id, employee_id, appointment_date,
		           start_time, end_time, produced_qty, scrapped_qty, scrap_reason, notes,
		           created_at, updated_at, created_by`,
		a.ProductionOrderID, a.MachineID, a.EmployeeID, a.AppointmentDate,
		a.StartTime, a.EndTime,
		pgutil.ToPgNumericFromFloat64(a.ProducedQty),
		pgutil.ToPgNumericFromFloat64(a.ScrappedQty),
		a.ScrapReason, a.Notes, pgutil.ToPgUUID(a.CreatedBy),
	)

	var startTime, endTime, scrapReason, notes *string
	var machineID, employeeID *int64
	var id, prodOrderID int64
	var appointmentDate time.Time
	var producedQty, scrappedQty float64
	var createdAt, updatedAt time.Time
	var createdBy uuid.UUID

	err := row.Scan(&id, &prodOrderID, &machineID, &employeeID, &appointmentDate,
		&startTime, &endTime, &producedQty, &scrappedQty, &scrapReason, &notes,
		&createdAt, &updatedAt, &createdBy)
	if err != nil {
		return nil, fmt.Errorf("adding appointment: %w", err)
	}

	return &entity.ProductionAppointment{
		ID:                id,
		ProductionOrderID: prodOrderID,
		MachineID:         machineID,
		EmployeeID:        employeeID,
		AppointmentDate:   appointmentDate,
		StartTime:         startTime,
		EndTime:           endTime,
		ProducedQty:       producedQty,
		ScrappedQty:       scrappedQty,
		ScrapReason:       scrapReason,
		Notes:             notes,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
		CreatedBy:         createdBy,
	}, nil
}

func (r *ProductionOrderRepositoryPGX) AddConsumption(ctx context.Context, c *entity.ProductionConsumption) (*entity.ProductionConsumption, error) {
	if _, err := r.GetByCode(ctx, c.ProductionOrderID); err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx,
		`INSERT INTO public.production_consumptions
			(production_order_id, appointment_id, item_code, consumed_qty,
			 warehouse_id, lot, consumption_date, notes, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING id, production_order_id, appointment_id, item_code, consumed_qty,
		           warehouse_id, lot, consumption_date, notes, created_at, created_by`,
		c.ProductionOrderID, c.AppointmentID, c.ItemCode,
		pgutil.ToPgNumericFromFloat64(c.ConsumedQty),
		c.WarehouseID, c.Lot, c.ConsumptionDate, c.Notes,
		pgutil.ToPgUUID(c.CreatedBy),
	)

	var appointmentID, warehouseID *int64
	var lot, notes *string
	var id, prodOrderID, itemCode int64
	var consumedQty float64
	var consumptionDate, createdAt time.Time
	var createdBy uuid.UUID

	err := row.Scan(&id, &prodOrderID, &appointmentID, &itemCode, &consumedQty,
		&warehouseID, &lot, &consumptionDate, &notes, &createdAt, &createdBy)
	if err != nil {
		return nil, fmt.Errorf("adding consumption: %w", err)
	}

	return &entity.ProductionConsumption{
		ID:                id,
		ProductionOrderID: prodOrderID,
		AppointmentID:     appointmentID,
		ItemCode:          itemCode,
		ConsumedQty:       consumedQty,
		WarehouseID:       warehouseID,
		Lot:               lot,
		ConsumptionDate:   consumptionDate,
		Notes:             notes,
		CreatedAt:         createdAt,
		CreatedBy:         createdBy,
	}, nil
}

func (r *ProductionOrderRepositoryPGX) Complete(ctx context.Context, id int64, endDate time.Time) (*entity.ProductionOrder, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx,
		`UPDATE public.production_orders
		 SET status='COMPLETED', end_date=$2, updated_at=NOW()
		 WHERE id=$1 AND enterprise_id=$3
		 RETURNING id, order_number, planned_order_id, item_code, mask, planned_qty, produced_qty, scrapped_qty,
		           status, start_date, end_date, machine_id, cost_center_id, employee_id, priority, notes,
		           is_active, created_at, updated_at, created_by, warehouse_id`,
		id, endDate, enterpriseID,
	)
	return scanProductionOrder(row)
}

func (r *ProductionOrderRepositoryPGX) Close(ctx context.Context, id int64) (*entity.ProductionOrder, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx,
		`UPDATE public.production_orders
		 SET status='CLOSED', updated_at=NOW()
		 WHERE id=$1 AND enterprise_id=$2
		 RETURNING id, order_number, planned_order_id, item_code, mask, planned_qty, produced_qty, scrapped_qty,
		           status, start_date, end_date, machine_id, cost_center_id, employee_id, priority, notes,
		           is_active, created_at, updated_at, created_by, warehouse_id`,
		id, enterpriseID,
	)
	return scanProductionOrder(row)
}

func (r *ProductionOrderRepositoryPGX) Cancel(ctx context.Context, id int64) (*entity.ProductionOrder, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx,
		`UPDATE public.production_orders
		 SET status='CANCELLED', is_active=FALSE, updated_at=NOW()
		 WHERE id=$1 AND enterprise_id=$2 AND origin_type NOT IN ('KANBAN','COMMERCIAL')
		 AND NOT EXISTS(SELECT 1 FROM production_appointments a WHERE a.production_order_id=production_orders.id)
		 AND NOT EXISTS(SELECT 1 FROM production_consumptions c WHERE c.production_order_id=production_orders.id)
		 AND NOT EXISTS(SELECT 1 FROM stock_movements m WHERE m.enterprise_id=$2 AND m.reference_type='PRODUCTION_ORDER' AND m.reference_code=production_orders.id)
		 RETURNING id, order_number, planned_order_id, item_code, mask, planned_qty, produced_qty, scrapped_qty,
		           status, start_date, end_date, machine_id, cost_center_id, employee_id, priority, notes,
		           is_active, created_at, updated_at, created_by, warehouse_id`,
		id, enterpriseID,
	)
	return scanProductionOrder(row)
}

func (r *ProductionOrderRepositoryPGX) GetAppointments(ctx context.Context, productionOrderID int64) ([]*entity.ProductionAppointment, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, production_order_id, machine_id, employee_id, appointment_date,
		        start_time, end_time, produced_qty, scrapped_qty, scrap_reason, notes,
		        created_at, updated_at, created_by
		 FROM public.production_appointments
		 WHERE production_order_id=$1 AND EXISTS (SELECT 1 FROM production_orders po WHERE po.id=$1 AND po.enterprise_id=$2)
		 ORDER BY id DESC`, productionOrderID, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing appointments: %w", err)
	}
	defer rows.Close()

	var appointments []*entity.ProductionAppointment
	for rows.Next() {
		var startTime, endTime, scrapReason, notes *string
		var machineID, employeeID *int64
		var id, prodOrderID int64
		var appointmentDate time.Time
		var producedQty, scrappedQty float64
		var createdAt, updatedAt time.Time
		var createdBy uuid.UUID

		err := rows.Scan(&id, &prodOrderID, &machineID, &employeeID, &appointmentDate,
			&startTime, &endTime, &producedQty, &scrappedQty, &scrapReason, &notes,
			&createdAt, &updatedAt, &createdBy)
		if err != nil {
			return nil, fmt.Errorf("scanning appointment: %w", err)
		}

		appointments = append(appointments, &entity.ProductionAppointment{
			ID:                id,
			ProductionOrderID: prodOrderID,
			MachineID:         machineID,
			EmployeeID:        employeeID,
			AppointmentDate:   appointmentDate,
			StartTime:         startTime,
			EndTime:           endTime,
			ProducedQty:       producedQty,
			ScrappedQty:       scrappedQty,
			ScrapReason:       scrapReason,
			Notes:             notes,
			CreatedAt:         createdAt,
			UpdatedAt:         updatedAt,
			CreatedBy:         createdBy,
		})
	}
	return appointments, rows.Err()
}

func (r *ProductionOrderRepositoryPGX) GetConsumptions(ctx context.Context, productionOrderID int64) ([]*entity.ProductionConsumption, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, production_order_id, appointment_id, item_code, consumed_qty,
		        warehouse_id, lot, consumption_date, notes, created_at, created_by
		 FROM public.production_consumptions
		 WHERE production_order_id=$1 AND EXISTS (SELECT 1 FROM production_orders po WHERE po.id=$1 AND po.enterprise_id=$2)
		 ORDER BY id DESC`, productionOrderID, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing consumptions: %w", err)
	}
	defer rows.Close()

	var consumptions []*entity.ProductionConsumption
	for rows.Next() {
		var appointmentID, warehouseID *int64
		var lot, notes *string
		var id, prodOrderID, itemCode int64
		var consumedQty float64
		var consumptionDate, createdAt time.Time
		var createdBy uuid.UUID

		err := rows.Scan(&id, &prodOrderID, &appointmentID, &itemCode, &consumedQty,
			&warehouseID, &lot, &consumptionDate, &notes, &createdAt, &createdBy)
		if err != nil {
			return nil, fmt.Errorf("scanning consumption: %w", err)
		}

		consumptions = append(consumptions, &entity.ProductionConsumption{
			ID:                id,
			ProductionOrderID: prodOrderID,
			AppointmentID:     appointmentID,
			ItemCode:          itemCode,
			ConsumedQty:       consumedQty,
			WarehouseID:       warehouseID,
			Lot:               lot,
			ConsumptionDate:   consumptionDate,
			Notes:             notes,
			CreatedAt:         createdAt,
			CreatedBy:         createdBy,
		})
	}
	return consumptions, rows.Err()
}

// ComputeActualCostInputs aggregates the actual material and labor cost incurred
// by an order. Material is the sum of each consumption valued at the item's
// weighted-average cost (preferring the consumption warehouse, falling back to
// the most recently updated balance for the item). Labor is the sum of appointed
// durations × the cost/hour of the work center the appointment's machine belongs
// to (a work center is a machine_type, which is what work_center_costs keys on).
func (r *ProductionOrderRepositoryPGX) ComputeActualCostInputs(ctx context.Context, productionOrderID int64) (float64, float64, error) {
	var materialReal float64
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(c.consumed_qty * COALESCE(sb.avg_cost, 0)), 0)
		 FROM public.production_consumptions c
		 LEFT JOIN LATERAL (
		     SELECT b.avg_cost FROM public.stock_balances b
		     WHERE b.item_code = c.item_code
		     ORDER BY CASE WHEN b.warehouse_id = c.warehouse_id THEN 0 ELSE 1 END, b.updated_at DESC
		     LIMIT 1
		 ) sb ON TRUE
		 WHERE c.production_order_id = $1`, productionOrderID).Scan(&materialReal)
	if err != nil {
		return 0, 0, fmt.Errorf("computing actual material cost: %w", err)
	}

	var laborReal float64
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(
		     EXTRACT(EPOCH FROM (a.end_time - a.start_time)) / 3600.0 * wcc.cost_per_hour
		 ), 0)
		 FROM public.production_appointments a
		 JOIN public.machines m ON m.id = a.machine_id
		 JOIN public.machine_types mt ON mt.code = m.machine_type_code
		 JOIN public.work_center_costs wcc ON wcc.work_center_id = mt.id
		 WHERE a.production_order_id = $1
		   AND a.start_time IS NOT NULL AND a.end_time IS NOT NULL`, productionOrderID).Scan(&laborReal)
	if err != nil {
		return 0, 0, fmt.Errorf("computing actual labor cost: %w", err)
	}

	return materialReal, laborReal, nil
}

// SettleCost upserts the cost settlement of a production order (one row per OF).
func (r *ProductionOrderRepositoryPGX) SettleCost(ctx context.Context, c *entity.ProductionOrderCost) (*entity.ProductionOrderCost, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO public.production_order_costs
			(production_order_id, produced_qty,
			 material_cost_real, labor_cost_real, overhead_cost_real, total_cost_real, unit_cost_real,
			 material_cost_std, labor_cost_std, overhead_cost_std, total_cost_std,
			 material_variance, labor_variance, overhead_variance, total_variance,
			 currency, settled_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		 ON CONFLICT (production_order_id) DO UPDATE SET
			 produced_qty       = EXCLUDED.produced_qty,
			 material_cost_real = EXCLUDED.material_cost_real,
			 labor_cost_real    = EXCLUDED.labor_cost_real,
			 overhead_cost_real = EXCLUDED.overhead_cost_real,
			 total_cost_real    = EXCLUDED.total_cost_real,
			 unit_cost_real     = EXCLUDED.unit_cost_real,
			 material_cost_std  = EXCLUDED.material_cost_std,
			 labor_cost_std     = EXCLUDED.labor_cost_std,
			 overhead_cost_std  = EXCLUDED.overhead_cost_std,
			 total_cost_std     = EXCLUDED.total_cost_std,
			 material_variance  = EXCLUDED.material_variance,
			 labor_variance     = EXCLUDED.labor_variance,
			 overhead_variance  = EXCLUDED.overhead_variance,
			 total_variance     = EXCLUDED.total_variance,
			 currency           = EXCLUDED.currency,
			 settled_at         = NOW(),
			 settled_by         = EXCLUDED.settled_by
		 RETURNING id, settled_at`,
		c.ProductionOrderID, pgutil.ToPgNumericFromFloat64(c.ProducedQty),
		pgutil.ToPgNumericFromFloat64(c.MaterialCostReal), pgutil.ToPgNumericFromFloat64(c.LaborCostReal),
		pgutil.ToPgNumericFromFloat64(c.OverheadCostReal), pgutil.ToPgNumericFromFloat64(c.TotalCostReal),
		pgutil.ToPgNumericFromFloat64(c.UnitCostReal),
		pgutil.ToPgNumericFromFloat64(c.MaterialCostStd), pgutil.ToPgNumericFromFloat64(c.LaborCostStd),
		pgutil.ToPgNumericFromFloat64(c.OverheadCostStd), pgutil.ToPgNumericFromFloat64(c.TotalCostStd),
		pgutil.ToPgNumericFromFloat64(c.MaterialVariance), pgutil.ToPgNumericFromFloat64(c.LaborVariance),
		pgutil.ToPgNumericFromFloat64(c.OverheadVariance), pgutil.ToPgNumericFromFloat64(c.TotalVariance),
		c.Currency, pgutil.ToPgUUID(c.SettledBy),
	)
	if err := row.Scan(&c.ID, &c.SettledAt); err != nil {
		return nil, fmt.Errorf("settling production order cost: %w", err)
	}
	return c, nil
}

func (r *ProductionOrderRepositoryPGX) GetCost(ctx context.Context, productionOrderID int64) (*entity.ProductionOrderCost, error) {
	var c entity.ProductionOrderCost
	err := r.pool.QueryRow(ctx,
		`SELECT id, production_order_id, produced_qty,
		        material_cost_real, labor_cost_real, overhead_cost_real, total_cost_real, unit_cost_real,
		        material_cost_std, labor_cost_std, overhead_cost_std, total_cost_std,
		        material_variance, labor_variance, overhead_variance, total_variance,
		        currency, settled_at, settled_by
		 FROM public.production_order_costs WHERE production_order_id = $1`, productionOrderID,
	).Scan(&c.ID, &c.ProductionOrderID, &c.ProducedQty,
		&c.MaterialCostReal, &c.LaborCostReal, &c.OverheadCostReal, &c.TotalCostReal, &c.UnitCostReal,
		&c.MaterialCostStd, &c.LaborCostStd, &c.OverheadCostStd, &c.TotalCostStd,
		&c.MaterialVariance, &c.LaborVariance, &c.OverheadVariance, &c.TotalVariance,
		&c.Currency, &c.SettledAt, &c.SettledBy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("cost settlement not found for production order %d", productionOrderID)
		}
		return nil, fmt.Errorf("getting production order cost: %w", err)
	}
	return &c, nil
}

func (r *ProductionOrderRepositoryPGX) GetNextOrderNumber(ctx context.Context) (int64, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	var num int64
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(order_number), 0) + 1 FROM public.production_orders WHERE enterprise_id=$1`, enterpriseID).Scan(&num)
	if err != nil {
		return 1, nil
	}
	return num, nil
}

func pgDatePtr(t *time.Time) *time.Time {
	return t
}

func scanProductionOrder(row pgx.Row) (*entity.ProductionOrder, error) {
	return scanProductionOrderFromRows(row)
}

func scanProductionOrderFromRows(row pgx.Row) (*entity.ProductionOrder, error) {
	var id, orderNumber, itemCode int64
	var plannedOrderID, machineID, costCenterID, employeeID, warehouseID *int64
	var mask, status string
	var priority, notes *string
	var plannedQty, producedQty, scrappedQty float64
	var startDate, endDate *time.Time
	var isActive bool
	var createdAt, updatedAt time.Time
	var createdBy uuid.UUID

	err := row.Scan(&id, &orderNumber, &plannedOrderID, &itemCode, &mask,
		&plannedQty, &producedQty, &scrappedQty, &status,
		&startDate, &endDate, &machineID, &costCenterID, &employeeID,
		&priority, &notes, &isActive, &createdAt, &updatedAt, &createdBy, &warehouseID)
	if err != nil {
		return nil, fmt.Errorf("scanning production order: %w", err)
	}

	return &entity.ProductionOrder{
		ID:             id,
		OrderNumber:    orderNumber,
		PlannedOrderID: plannedOrderID,
		ItemCode:       itemCode,
		Mask:           mask,
		PlannedQty:     plannedQty,
		ProducedQty:    producedQty,
		ScrappedQty:    scrappedQty,
		Status:         entity.ProductionOrderStatus(status),
		StartDate:      startDate,
		EndDate:        endDate,
		MachineID:      machineID,
		CostCenterID:   costCenterID,
		EmployeeID:     employeeID,
		WarehouseID:    warehouseID,
		Priority:       priority,
		Notes:          notes,
		IsActive:       isActive,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
		CreatedBy:      createdBy,
	}, nil
}
