package production_order

import (
	"context"
	"errors"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	serierepository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/serie"
	stockrepository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/shopspring/decimal"
)

// Accept a pool or an existing transaction; nested writes use savepoints.
type productionDB interface {
	sqlc.DBTX
	Begin(context.Context) (pgx.Tx, error)
}

type ProductionOrderRepositoryPGX struct {
	pool productionDB
}

func NewProductionOrderRepositoryPGX(pool productionDB) *ProductionOrderRepositoryPGX {
	return &ProductionOrderRepositoryPGX{pool: pool}
}

func (r *ProductionOrderRepositoryPGX) Create(ctx context.Context, o *entity.ProductionOrder) (*entity.ProductionOrder, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	row := tx.QueryRow(ctx,
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
		pgutil.ToPgNumericFromDecimal(o.PlannedQty),
		pgutil.ToPgNumericFromDecimal(o.ProducedQty),
		pgutil.ToPgNumericFromDecimal(o.ScrappedQty),
		string(o.Status),
		pgDatePtr(o.StartDate), pgDatePtr(o.EndDate),
		o.MachineID, o.CostCenterID, o.EmployeeID,
		o.Priority, o.Notes, o.IsActive, pgutil.ToPgUUID(o.CreatedBy), o.WarehouseID, enterpriseID,
	)
	created, err := scanProductionOrder(row)
	if err != nil {
		return nil, err
	}
	if err = createThirdPartyOrdersTx(ctx, tx, enterpriseID, created); err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
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
		pgutil.ToPgNumericFromDecimal(o.PlannedQty),
		pgutil.ToPgNumericFromDecimal(o.ProducedQty),
		pgutil.ToPgNumericFromDecimal(o.ScrappedQty),
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
	q := sqlc.New(tx)
	locked, err := q.LockProductionExecution(ctx, sqlc.LockProductionExecutionParams{ID: d.ProductionOrderID, EnterpriseID: &enterpriseID})
	if err != nil {
		return nil, err
	}
	if locked.Status == "COMPLETED" || locked.Status == "CANCELLED" {
		return nil, errorsuc.NewValidationError("não é possível entregar produção de uma ordem encerrada ou cancelada")
	}
	if d.IsFinal {
		pending, err := q.ProductionOperationsPending(ctx, d.ProductionOrderID)
		if err != nil {
			return nil, err
		}
		if pending {
			return nil, errorsuc.NewValidationError("conclua todas as etapas do roteiro antes da entrega final")
		}

		var qualityPending bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM quality_inspection_lots l LEFT JOIN quality_lot_decisions d ON d.lot_id=l.id
			 WHERE l.production_order_id=$1 AND l.enterprise_id=$2 AND l.origem IN('PROCESSO','FINAL')
			   AND (l.status='ABERTO' OR (l.status='DECIDIDO' AND d.decisao='REJEITADO')))`, d.ProductionOrderID, enterpriseID).Scan(&qualityPending); err != nil {
			return nil, err
		}
		if qualityPending {
			return nil, errorsuc.NewValidationError("conclua a avaliação da qualidade antes da entrega final")
		}
		var servicesPending bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM production_order_service_links link JOIN purchase_orders po ON po.code=link.purchase_order_code WHERE link.production_order_id=$1 AND link.enterprise_id=$2 AND po.status NOT IN('RECEIVED','CANCELLED'))`, d.ProductionOrderID, enterpriseID).Scan(&servicesPending); err != nil {
			return nil, err
		}
		if servicesPending {
			return nil, errorsuc.NewValidationError("conclua ou cancele os pedidos de serviço de terceiros antes da entrega final")
		}
	}
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
	// O produto entra por último: o valor dele inclui o que esta entrega baixou.
	ehProduto := func(m *stockentity.StockMovement) bool {
		return m.MovementType == stockentity.MovementTypeProductionEntry || m.MovementType == stockentity.MovementTypePlannedProductionEntry ||
			m.MovementType == stockentity.MovementTypeProductionExcess
	}
	for _, movement := range movements {
		if ehProduto(movement) {
			continue
		}
		if err := stockrepository.CreateMovementTx(ctx, tx, enterpriseID, movement); err != nil {
			return nil, err
		}
	}
	var planejado, entregue decimal.Decimal
	if err := tx.QueryRow(ctx,
		`SELECT o.planned_qty, COALESCE((SELECT SUM(x.quantity) FROM production_deliveries x WHERE x.production_order_id=o.id AND x.enterprise_id=o.enterprise_id), 0)
		   FROM production_orders o WHERE o.id=$1 AND o.enterprise_id=$2`, d.ProductionOrderID, enterpriseID).Scan(&planejado, &entregue); err != nil {
		return nil, fmt.Errorf("lendo o saldo a produzir da ordem: %w", err)
	}
	unitario, err := custoUnitarioDaEntrega(ctx, tx, enterpriseID, d.ProductionOrderID, locked.ItemCode, locked.Mask,
		planejado.Sub(entregue).Add(d.Quantity), d.Quantity, d.IsFinal)
	if err != nil {
		return nil, err
	}
	for _, movement := range movements {
		if !ehProduto(movement) {
			continue
		}
		movement.UnitPrice = unitario
		movement.ValorizarPelaMedia = !unitario.IsPositive()
		if err := stockrepository.CreateMovementTx(ctx, tx, enterpriseID, movement); err != nil {
			return nil, err
		}
	}
	// Item com número de série: as séries nascem em estoque nesta transação.
	if d.ItemCode > 0 && d.Quantity.IsPositive() {
		var por *uuid.UUID
		if d.CreatedBy != uuid.Nil {
			por = &d.CreatedBy
		}
		gravadas, err := serierepository.EntradaPelaProducaoTx(ctx, tx, enterpriseID, d.ProductionOrderID, d.ItemCode, d.Mask,
			d.WarehouseID, d.Lot, d.Quantity, d.NumerosSerie, por, time.Now())
		if err != nil {
			return nil, err
		}
		d.SeriesGravadas = gravadas
	}
	// O produzido para amostra, feira, projeto… fica reservado para a demanda
	// da OF: nenhum pedido de venda o pega antes da saída. Reserva só o que
	// entrou de fato (a OF de beneficiamento não dá entrada), na linha de saldo
	// e no lote do movimento.
	for _, movement := range movements {
		if movement.ItemCode != d.ItemCode {
			continue // coproduto e requisição de componente
		}
		if _, err := stockrepository.ReservarEntradaDaOFTx(ctx, tx, enterpriseID, d.ProductionOrderID, movement, d.CreatedBy); err != nil {
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
	if err := opcional(r.pool.QueryRow(ctx, `SELECT UPPER(value) IN ('S','SIM','1','TRUE','YES') FROM planning_params
		WHERE enterprise_id=$1 AND (param_number=45 OR param_key='OBRIGAR_CONTROLE_ESTOQUE_TERCEIROS') LIMIT 1`, enterpriseID).Scan(&enabled)); err != nil {
		return fmt.Errorf("lendo o parâmetro 45: %w", err)
	}
	if !enabled {
		return nil
	}
	var reporting, issue string
	if err := r.pool.QueryRow(ctx, `SELECT production_reporting_type,material_issue_timing FROM items
		WHERE code=$1 AND enterprise_id=$2`, itemCode, enterpriseID).Scan(&reporting, &issue); err != nil {
		return err
	}
	if reporting == "ORDER" && issue == "REGISTRATION_RELEASE" {
		return errorsuc.NewValidationError("a ordem não pode ser liberada: ela aponta com baixa no cadastro ou na liberação")
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
		return errorsuc.NewValidationError("a ordem não pode ser liberada: operações de terceiros não podem conviver com outras operações")
	}
	if invalidRemittance {
		return errorsuc.NewValidationError("a ordem não pode ser liberada: a remessa a terceiros deve usar itens de demanda")
	}
	return nil
}

func (r *ProductionOrderRepositoryPGX) LinkServicePurchaseOrder(ctx context.Context, requisitionItemID, purchaseOrderCode int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	command, err := tx.Exec(ctx, `INSERT INTO production_order_service_links
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
		var serviceRequisition bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM purchase_requisition_items item
			JOIN production_order_service_requisition_links req_link ON req_link.purchase_requisition_code=item.requisition_code AND req_link.enterprise_id=$1
			WHERE item.id=$2)`, enterpriseID, requisitionItemID).Scan(&serviceRequisition)
		if err != nil {
			return err
		}
		// Ordinary purchase requisitions have no manufacturing-service origin.
		// Linking is intentionally a no-op for them; only service requisitions
		// participate in the OF -> requisition -> purchase-order traceability.
		if !serviceRequisition {
			return tx.Commit(ctx)
		}
		var exists bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM purchase_requisition_items item
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
	_, err = tx.Exec(ctx, `UPDATE third_party_service_orders service_order
		SET purchase_order_code=$3,status='RELEASED_WITH_PO',updated_at=NOW()
		FROM purchase_requisition_items item
		JOIN production_order_service_requisition_links req_link
		  ON req_link.purchase_requisition_code=item.requisition_code AND req_link.enterprise_id=$1
		WHERE item.id=$2 AND service_order.enterprise_id=$1
		  AND service_order.production_order_id=req_link.production_order_id`, enterpriseID, requisitionItemID, purchaseOrderCode)
	if err != nil {
		return err
	}
	// Legacy service requisition links may predate the dedicated external-service
	// order. The purchase-order link is still valid; update the service order only
	// when one exists.
	return tx.Commit(ctx)
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
	nextNum, err := r.GetNextOrderNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("numerando a ordem de produção: %w", err)
	}

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
		 WHERE id=$1 AND enterprise_id=$3 AND status='OPEN'
		 RETURNING id, order_number, planned_order_id, item_code, mask, planned_qty, produced_qty, scrapped_qty,
		           status, start_date, end_date, machine_id, cost_center_id, employee_id, priority, notes,
		           is_active, created_at, updated_at, created_by, warehouse_id`,
		id, startDate, enterpriseID,
	)
	return scanProductionOrder(row)
}

func (r *ProductionOrderRepositoryPGX) AddAppointment(ctx context.Context, a *entity.ProductionAppointment) (*entity.ProductionAppointment, error) {
	order, err := r.GetByCode(ctx, a.ProductionOrderID)
	if err != nil {
		return nil, err
	}
	if order.Status != entity.StatusInProgress {
		return nil, errorsuc.NewValidationError("a ordem de produção precisa estar em andamento para receber apontamento")
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
		pgutil.ToPgNumericFromDecimal(a.ProducedQty),
		pgutil.ToPgNumericFromDecimal(a.ScrappedQty),
		a.ScrapReason, a.Notes, pgutil.ToPgUUID(a.CreatedBy),
	)

	var startTime, endTime, scrapReason, notes *string
	var machineID, employeeID *int64
	var id, prodOrderID int64
	var appointmentDate time.Time
	var producedQty, scrappedQty decimal.Decimal
	var createdAt, updatedAt time.Time
	var createdBy uuid.UUID

	err = row.Scan(&id, &prodOrderID, &machineID, &employeeID, &appointmentDate,
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
		pgutil.ToPgNumericFromDecimal(c.ConsumedQty),
		c.WarehouseID, c.Lot, c.ConsumptionDate, c.Notes,
		pgutil.ToPgUUID(c.CreatedBy),
	)

	var appointmentID, warehouseID *int64
	var lot, notes *string
	var id, prodOrderID, itemCode int64
	var consumedQty decimal.Decimal
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
		 WHERE id=$1 AND enterprise_id=$3 AND status='IN_PROGRESS'
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
		 WHERE id=$1 AND enterprise_id=$2 AND status='COMPLETED'
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
		var producedQty, scrappedQty decimal.Decimal
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
		var consumedQty decimal.Decimal
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
// by an order. Material é o que o estoque baixou para a ordem (o valor do movimento,
// no custo da data do consumo), menos o que voltou; consumo sem almoxarifado não
// gera movimento e entra pelo custo médio atual. Labor is the sum of appointed
// durations × the cost/hour of the work center the appointment's machine belongs
// to (a work center is a machine_type, which is what work_center_costs keys on).
func (r *ProductionOrderRepositoryPGX) ComputeActualCostInputs(ctx context.Context, productionOrderID int64) (decimal.Decimal, decimal.Decimal, error) {
	return custoRealDaOrdem(ctx, r.pool, productionOrderID)
}

// leitorDeLinha aceita o pool ou a transação de quem chamou.
type leitorDeLinha interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func custoRealDaOrdem(ctx context.Context, q leitorDeLinha, productionOrderID int64) (decimal.Decimal, decimal.Decimal, error) {
	var materialReal decimal.Decimal
	err := q.QueryRow(ctx,
		`SELECT COALESCE((SELECT SUM(m.total_price) FROM public.stock_movements m
		                   JOIN public.production_orders o ON o.id = m.reference_code AND o.enterprise_id = m.enterprise_id
		                  WHERE m.reference_type = 'PRODUCTION_ORDER' AND m.reference_code = $1
		                    AND m.movement_type IN ('OUT','REP') AND m.reversed_by_movement_id IS NULL), 0)
		      - COALESCE((SELECT SUM(m.total_price) FROM public.stock_movements m
		                   JOIN public.production_orders o ON o.id = m.reference_code AND o.enterprise_id = m.enterprise_id
		                  WHERE m.reference_type = 'PRODUCTION_ORDER' AND m.reference_code = $1
		                    AND m.movement_type = 'IN' AND m.item_code <> o.item_code AND m.reversed_by_movement_id IS NULL), 0)
		      + COALESCE((SELECT SUM(c.consumed_qty * COALESCE(sb.avg_cost, 0))
		                    FROM public.production_consumptions c
		                    LEFT JOIN LATERAL (
		                        SELECT b.avg_cost FROM public.stock_balances b
		                         WHERE b.item_code = c.item_code
		                         ORDER BY b.updated_at DESC LIMIT 1
		                    ) sb ON TRUE
		                   WHERE c.production_order_id = $1 AND c.warehouse_id IS NULL), 0)`, productionOrderID).Scan(&materialReal)
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("computing actual material cost: %w", err)
	}

	var laborReal decimal.Decimal
	err = q.QueryRow(ctx,
		`SELECT COALESCE(SUM(
		     EXTRACT(EPOCH FROM (a.end_time - a.start_time)) / 3600.0 * wcc.cost_per_hour
		 ), 0)
		 FROM public.production_appointments a
		 JOIN public.machines m ON m.id = a.machine_id
		 JOIN public.machine_types mt ON mt.code = m.machine_type_code
		 JOIN public.production_orders o ON o.id = a.production_order_id
		 JOIN public.work_center_costs wcc ON wcc.work_center_id = mt.id AND wcc.enterprise_id = o.enterprise_id
		 WHERE a.production_order_id = $1
		   AND a.start_time IS NOT NULL AND a.end_time IS NOT NULL`, productionOrderID).Scan(&laborReal)
	if err != nil {
		return decimal.Zero, decimal.Zero, fmt.Errorf("computing actual labor cost: %w", err)
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
		c.ProductionOrderID, pgutil.ToPgNumericFromDecimal(c.ProducedQty),
		pgutil.ToPgNumericFromDecimal(c.MaterialCostReal), pgutil.ToPgNumericFromDecimal(c.LaborCostReal),
		pgutil.ToPgNumericFromDecimal(c.OverheadCostReal), pgutil.ToPgNumericFromDecimal(c.TotalCostReal),
		pgutil.ToPgNumericFromDecimal(c.UnitCostReal),
		pgutil.ToPgNumericFromDecimal(c.MaterialCostStd), pgutil.ToPgNumericFromDecimal(c.LaborCostStd),
		pgutil.ToPgNumericFromDecimal(c.OverheadCostStd), pgutil.ToPgNumericFromDecimal(c.TotalCostStd),
		pgutil.ToPgNumericFromDecimal(c.MaterialVariance), pgutil.ToPgNumericFromDecimal(c.LaborVariance),
		pgutil.ToPgNumericFromDecimal(c.OverheadVariance), pgutil.ToPgNumericFromDecimal(c.TotalVariance),
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
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("apuração de custo não encontrada para a ordem de produção %d", productionOrderID))
		}
		return nil, fmt.Errorf("getting production order cost: %w", err)
	}
	return &c, nil
}

// AlmoxarifadoDaEmpresa diz se o almoxarifado é da empresa da sessão.
func (r *ProductionOrderRepositoryPGX) AlmoxarifadoDaEmpresa(ctx context.Context, id int64) (bool, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var ok bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM warehouse WHERE id = $1 AND enterprise_id = $2)`, id, enterpriseID).Scan(&ok); err != nil {
		return false, fmt.Errorf("checking warehouse %d: %w", id, err)
	}
	return ok, nil
}

func (r *ProductionOrderRepositoryPGX) GetNextOrderNumber(ctx context.Context) (int64, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	var num int64
	err = r.pool.QueryRow(ctx,
		`INSERT INTO production_order_number_counters(enterprise_id,last_number)
 SELECT $1,COALESCE(MAX(order_number),0)+1 FROM production_orders WHERE enterprise_id=$1
 ON CONFLICT(enterprise_id) DO UPDATE SET last_number=GREATEST(production_order_number_counters.last_number,EXCLUDED.last_number-1)+1
 RETURNING last_number`, enterpriseID).Scan(&num)
	if err != nil {
		return 0, err
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
	var plannedQty, producedQty, scrappedQty decimal.Decimal
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

func (r *ProductionOrderRepositoryPGX) HasGeneratedProduction(ctx context.Context, plannedID int64) (bool, error) {
	enterprise, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var exists bool
	err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM production_orders WHERE enterprise_id=$1 AND planned_order_id=$2)`, enterprise, plannedID).Scan(&exists)
	return exists, err
}

// MarcarCustoPendente grava (ou limpa, com motivo nil) o motivo de a OF estar
// sem custo real apurado.
func (r *ProductionOrderRepositoryPGX) MarcarCustoPendente(ctx context.Context, id int64, motivo *string) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE public.production_orders SET custo_pendente=$3 WHERE id=$1 AND enterprise_id=$2`, id, enterpriseID, motivo)
	if err != nil {
		return fmt.Errorf("gravando o custo pendente da OF %d: %w", id, err)
	}
	if tag.RowsAffected() == 0 {
		return errorsuc.NewNotFoundError("ordem de fabricação não encontrada")
	}
	return nil
}

// CustoPendente lê o motivo de a OF estar sem custo real apurado (nil = apurado
// ou ainda não encerrada).
func (r *ProductionOrderRepositoryPGX) CustoPendente(ctx context.Context, id int64) (*string, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var motivo *string
	if err := r.pool.QueryRow(ctx, `SELECT custo_pendente FROM public.production_orders WHERE id=$1 AND enterprise_id=$2`, id, enterpriseID).Scan(&motivo); err != nil {
		return nil, fmt.Errorf("lendo o custo pendente da OF %d: %w", id, err)
	}
	return motivo, nil
}

// CustoUnitarioDaEntrega é o valor da entrega fora de transação, para consulta.
func (r *ProductionOrderRepositoryPGX) CustoUnitarioDaEntrega(ctx context.Context, order *entity.ProductionOrder, qty decimal.Decimal, final bool) (decimal.Decimal, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return decimal.Zero, err
	}
	return custoUnitarioDaEntrega(ctx, r.pool, enterpriseID, order.ID, order.ItemCode, order.Mask, order.PlannedQty.Sub(order.ProducedQty), qty, final)
}

// custoUnitarioDaEntrega diz por quanto a peça entra no estoque. O que a ordem já
// gastou (material baixado e mão de obra apontada) menos o que as entregas anteriores
// levaram é o custo a entregar: a entrega final leva tudo, a parcial, a sua parte do
// que falta produzir. Sem custo apurado vale o custo-padrão; sem padrão, zero, e o
// movimento entra pelo médio do item. Antes a entrada era sempre a custo zero: o
// produto acabado diluía o médio e saía na venda sem custo.
func custoUnitarioDaEntrega(ctx context.Context, q leitorDeLinha, enterpriseID, ordem, item int64, mask string, falta, qty decimal.Decimal, final bool) (decimal.Decimal, error) {
	if !qty.IsPositive() {
		return decimal.Zero, nil
	}
	material, mao, err := custoRealDaOrdem(ctx, q, ordem)
	if err != nil {
		return decimal.Zero, err
	}
	var entregue decimal.Decimal
	if err := q.QueryRow(ctx,
		`SELECT COALESCE(SUM(total_price), 0) FROM public.stock_movements
		  WHERE enterprise_id=$1 AND reference_type='PRODUCTION_ORDER' AND reference_code=$2
		    AND movement_type IN ('EP','EPP','EPE') AND reversed_by_movement_id IS NULL`, enterpriseID, ordem).Scan(&entregue); err != nil {
		return decimal.Zero, fmt.Errorf("lendo o valor já entregue da ordem: %w", err)
	}
	if restante := material.Add(mao).Sub(entregue); restante.IsPositive() {
		base := qty
		if !final && falta.GreaterThan(qty) {
			base = falta
		}
		return restante.DivRound(base, stockentity.CasasCusto), nil
	}
	var padrao decimal.Decimal
	err = q.QueryRow(ctx,
		`SELECT COALESCE(total_cost, 0) FROM public.item_standard_costs WHERE item_code=$1 AND mask=$2 ORDER BY calculated_at DESC LIMIT 1`,
		item, mask).Scan(&padrao)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return decimal.Zero, fmt.Errorf("lendo o custo-padrão do item: %w", err)
	}
	return padrao, nil
}

// BaixasDaOrdem diz quanto de cada componente a ordem já baixou (menos o que voltou),
// quanto já entregou e quanto apontou. A baixa automática da entrega e o backflush do
// apontamento descontam isso: sem o desconto, a mesma peça saía duas vezes.
func (r *ProductionOrderRepositoryPGX) BaixasDaOrdem(ctx context.Context, ordem int64) (map[int64]decimal.Decimal, decimal.Decimal, decimal.Decimal, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, decimal.Zero, decimal.Zero, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT m.item_code, SUM(CASE WHEN m.movement_type = 'IN' THEN -m.quantity ELSE m.quantity END)
		   FROM public.stock_movements m
		   JOIN public.production_orders o ON o.id = m.reference_code AND o.enterprise_id = m.enterprise_id
		  WHERE m.enterprise_id=$1 AND m.reference_type='PRODUCTION_ORDER' AND m.reference_code=$2
		    AND m.movement_type IN ('OUT','REP','IN') AND m.item_code <> o.item_code AND m.reversed_by_movement_id IS NULL
		  GROUP BY m.item_code`, enterpriseID, ordem)
	if err != nil {
		return nil, decimal.Zero, decimal.Zero, fmt.Errorf("lendo as baixas da ordem: %w", err)
	}
	defer rows.Close()
	baixado := map[int64]decimal.Decimal{}
	for rows.Next() {
		var item int64
		var q decimal.Decimal
		if err := rows.Scan(&item, &q); err != nil {
			return nil, decimal.Zero, decimal.Zero, fmt.Errorf("lendo as baixas da ordem: %w", err)
		}
		baixado[item] = q
	}
	if err := rows.Err(); err != nil {
		return nil, decimal.Zero, decimal.Zero, err
	}
	var entregue, apontado decimal.Decimal
	if err := r.pool.QueryRow(ctx,
		`SELECT COALESCE((SELECT SUM(quantity) FROM production_deliveries WHERE production_order_id=$1 AND enterprise_id=$2), 0),
		        COALESCE((SELECT SUM(produced_qty) FROM production_appointments WHERE production_order_id=$1), 0)`, ordem, enterpriseID).Scan(&entregue, &apontado); err != nil {
		return nil, decimal.Zero, decimal.Zero, fmt.Errorf("lendo o produzido da ordem: %w", err)
	}
	return baixado, entregue, apontado, nil
}
