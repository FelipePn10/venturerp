package production_order

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	stockrepository "github.com/FelipePn10/panossoerp/internal/infrastructure/repository/stock"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
)

const materialColumns = `id,production_order_id,material_kind,item_code,mask,substituted_item_code,
	quantity,attended_quantity,warehouse_id,automatic_issue,notes,created_at,updated_at,created_by`
const qualifiedMaterialColumns = `material.id,material.production_order_id,material.material_kind,material.item_code,material.mask,material.substituted_item_code,
	material.quantity,material.attended_quantity,material.warehouse_id,material.automatic_issue,material.notes,material.created_at,material.updated_at,material.created_by`

func (r *ProductionOrderRepositoryPGX) GetManualOrderPlanner(ctx context.Context, itemCode int64) (*int64, error) {
	if _, err := tenant.ID(ctx); err != nil {
		return nil, err
	}
	var planner *int64
	err := r.pool.QueryRow(ctx, `SELECT planner_employee_code FROM items WHERE code=$1`, itemCode).Scan(&planner)
	return planner, err
}

func (r *ProductionOrderRepositoryPGX) CreateWithMaterials(ctx context.Context, order *entity.ProductionOrder, materials []*entity.ProductionOrderMaterial) (*entity.ProductionOrder, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	created, err := scanProductionOrder(tx.QueryRow(ctx, `INSERT INTO production_orders
		(order_number,planned_order_id,item_code,mask,planned_qty,produced_qty,scrapped_qty,status,start_date,end_date,
		machine_id,cost_center_id,employee_id,priority,notes,is_active,created_by,warehouse_id,enterprise_id,origin_type)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,
		CASE WHEN $2::bigint IS NULL THEN 'MANUAL' WHEN EXISTS(SELECT 1 FROM kanban_cards k WHERE k.item_code=$3 AND k.enterprise_id=$19) THEN 'KANBAN' ELSE 'MRP' END)
		RETURNING id,order_number,planned_order_id,item_code,mask,planned_qty,produced_qty,scrapped_qty,status,
		start_date,end_date,machine_id,cost_center_id,employee_id,priority,notes,is_active,created_at,updated_at,created_by,warehouse_id`,
		order.OrderNumber, order.PlannedOrderID, order.ItemCode, order.Mask, pgutil.ToPgNumericFromFloat64(order.PlannedQty),
		pgutil.ToPgNumericFromFloat64(order.ProducedQty), pgutil.ToPgNumericFromFloat64(order.ScrappedQty), string(order.Status),
		pgDatePtr(order.StartDate), pgDatePtr(order.EndDate), order.MachineID, order.CostCenterID, order.EmployeeID, order.Priority,
		order.Notes, order.IsActive, pgutil.ToPgUUID(order.CreatedBy), order.WarehouseID, enterpriseID))
	if err != nil {
		return nil, err
	}
	var orderIssueType string
	_ = tx.QueryRow(ctx, `SELECT automatic_issue_type FROM manufacturing_stock_item_controls WHERE enterprise_id=$1 AND item_code=$2`, enterpriseID, order.ItemCode).Scan(&orderIssueType)
	for _, material := range materials {
		if !material.Quantity.IsPositive() {
			return nil, fmt.Errorf("material quantity must be positive")
		}
		if orderIssueType == "TRANSFER" && material.AutomaticIssue {
			var lineWarehouse *int64
			if err := tx.QueryRow(ctx, `SELECT line_warehouse_id FROM manufacturing_stock_item_controls WHERE enterprise_id=$1 AND item_code=$2`, enterpriseID, material.ItemCode).Scan(&lineWarehouse); err != nil || lineWarehouse == nil {
				return nil, fmt.Errorf("automatic transfer requires a line warehouse for component %d", material.ItemCode)
			}
			var balance decimal.Decimal
			if err := tx.QueryRow(ctx, `SELECT quantity FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 FOR UPDATE`, enterpriseID, material.ItemCode, material.Mask, material.WarehouseID).Scan(&balance); err != nil || balance.LessThan(material.Quantity) {
				return nil, fmt.Errorf("insufficient stock for automatic component transfer %d", material.ItemCode)
			}
			referenceType, referenceCode := "PRODUCTION_ORDER_TRANSFER", created.ID
			q, _ := material.Quantity.Float64()
			out := &stockentity.StockMovement{ItemCode: material.ItemCode, Mask: material.Mask, WarehouseID: material.WarehouseID, MovementType: stockentity.MovementTypeTransferOut, Quantity: q, ExactQuantity: material.Quantity, ReferenceType: &referenceType, ReferenceCode: &referenceCode, CreatedBy: material.CreatedBy}
			if err := stockrepository.CreateMovementTx(ctx, tx, enterpriseID, out); err != nil {
				return nil, err
			}
			in := &stockentity.StockMovement{ItemCode: material.ItemCode, Mask: material.Mask, WarehouseID: *lineWarehouse, MovementType: stockentity.MovementTypeTransferIn, Quantity: q, ExactQuantity: material.Quantity, ReferenceType: &referenceType, ReferenceCode: &referenceCode, CreatedBy: material.CreatedBy}
			if err := stockrepository.CreateMovementTx(ctx, tx, enterpriseID, in); err != nil {
				return nil, err
			}
			material.WarehouseID = *lineWarehouse
		}
		_, err = tx.Exec(ctx, `INSERT INTO production_order_materials
			(production_order_id,enterprise_id,material_kind,item_code,mask,substituted_item_code,quantity,warehouse_id,automatic_issue,notes,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, created.ID, enterpriseID, material.Kind, material.ItemCode,
			material.Mask, material.SubstitutedItemCode, material.Quantity, material.WarehouseID, material.AutomaticIssue, material.Notes, material.CreatedBy)
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

func scanMaterial(row pgx.Row) (*entity.ProductionOrderMaterial, error) {
	material := &entity.ProductionOrderMaterial{}
	err := row.Scan(&material.ID, &material.ProductionOrderID, &material.Kind, &material.ItemCode, &material.Mask,
		&material.SubstitutedItemCode, &material.Quantity, &material.AttendedQuantity, &material.WarehouseID,
		&material.AutomaticIssue, &material.Notes, &material.CreatedAt, &material.UpdatedAt, &material.CreatedBy)
	return material, err
}

func (r *ProductionOrderRepositoryPGX) ListMaterials(ctx context.Context, productionOrderID int64, kind entity.MaterialKind) ([]*entity.ProductionOrderMaterial, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+materialColumns+` FROM production_order_materials
		WHERE enterprise_id=$1 AND production_order_id=$2 AND ($3='' OR material_kind=$3)
		ORDER BY material_kind,id`, enterpriseID, productionOrderID, string(kind))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []*entity.ProductionOrderMaterial{}
	for rows.Next() {
		material, err := scanMaterial(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, material)
	}
	return result, rows.Err()
}

func (r *ProductionOrderRepositoryPGX) AddMaterial(ctx context.Context, material *entity.ProductionOrderMaterial) (*entity.ProductionOrderMaterial, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if !material.Quantity.IsPositive() {
		return nil, fmt.Errorf("material quantity must be positive")
	}
	var orderItem int64
	var planned decimal.Decimal
	if err := r.pool.QueryRow(ctx, `SELECT item_code,planned_qty FROM production_orders
		WHERE id=$1 AND enterprise_id=$2`, material.ProductionOrderID, enterpriseID).Scan(&orderItem, &planned); err != nil {
		return nil, err
	}
	if material.Kind == entity.MaterialDemand && material.ItemCode == orderItem {
		var enabled bool
		_ = r.pool.QueryRow(ctx, `SELECT UPPER(value) IN ('S','SIM','1','TRUE','YES') FROM planning_params
			WHERE enterprise_id=$1 AND (param_number=66 OR param_key='VALIDA_QUANTIDADE_RETRABALHO') LIMIT 1`, enterpriseID).Scan(&enabled)
		if enabled && !material.Quantity.Equal(planned) {
			return nil, fmt.Errorf("parameter 66 requires rework demand quantity to equal production order quantity")
		}
	}
	return scanMaterial(r.pool.QueryRow(ctx, `INSERT INTO production_order_materials
		(production_order_id,enterprise_id,material_kind,item_code,mask,substituted_item_code,quantity,warehouse_id,automatic_issue,notes,created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING `+materialColumns,
		material.ProductionOrderID, enterpriseID, material.Kind, material.ItemCode, material.Mask,
		material.SubstitutedItemCode, material.Quantity, material.WarehouseID, material.AutomaticIssue, material.Notes, material.CreatedBy))
}

func (r *ProductionOrderRepositoryPGX) HasActiveWMSRequest(ctx context.Context, materialID int64) (bool, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var exists bool
	err = r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM production_order_wms_requests
		WHERE enterprise_id=$1 AND production_order_material_id=$2 AND status<>'CANCELLED')`, enterpriseID, materialID).Scan(&exists)
	return exists, err
}

func (r *ProductionOrderRepositoryPGX) DeleteMaterial(ctx context.Context, materialID int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	command, err := r.pool.Exec(ctx, `DELETE FROM production_order_materials material
		WHERE material.id=$1 AND material.enterprise_id=$2 AND material.attended_quantity=0
		AND NOT EXISTS (SELECT 1 FROM production_order_wms_requests request
			WHERE request.production_order_material_id=material.id AND request.enterprise_id=$2 AND request.status<>'CANCELLED')
		AND NOT EXISTS (SELECT 1 FROM stock_movements movement
			WHERE movement.enterprise_id=$2 AND movement.reference_type='PRODUCTION_ORDER'
			AND movement.reference_code=material.production_order_id)`, materialID, enterpriseID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return fmt.Errorf("material cannot be deleted after attendance, movement or WMS separation")
	}
	return nil
}

func (r *ProductionOrderRepositoryPGX) ReplaceMaterial(ctx context.Context, materialID int64, replacements []entity.MaterialSubstitution, createdBy uuid.UUID) ([]*entity.ProductionOrderMaterial, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	original, err := scanMaterial(tx.QueryRow(ctx, `SELECT `+materialColumns+` FROM production_order_materials
		WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, materialID, enterpriseID))
	if err != nil {
		return nil, err
	}
	if original.SubstitutedItemCode != nil || original.AttendedQuantity.IsPositive() {
		return nil, fmt.Errorf("attended or substitute materials cannot be replaced")
	}
	var wms bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM production_order_wms_requests
		WHERE enterprise_id=$1 AND production_order_material_id=$2 AND status<>'CANCELLED')`, enterpriseID, materialID).Scan(&wms); err != nil {
		return nil, err
	}
	if wms {
		return nil, fmt.Errorf("material has a non-cancelled WMS separation request")
	}
	total := decimal.Zero
	for _, replacement := range replacements {
		if replacement.ItemCode == 0 || !replacement.Quantity.IsPositive() {
			return nil, fmt.Errorf("replacement item and positive quantity are required")
		}
		total = total.Add(replacement.Quantity)
	}
	remaining := original.Quantity.Sub(original.AttendedQuantity)
	if total.GreaterThan(remaining) {
		return nil, fmt.Errorf("replacement quantity exceeds remaining material quantity")
	}
	if _, err := tx.Exec(ctx, `UPDATE production_order_materials SET quantity=$1,updated_at=NOW()
		WHERE id=$2 AND enterprise_id=$3`, remaining.Sub(total), materialID, enterpriseID); err != nil {
		return nil, err
	}
	result := []*entity.ProductionOrderMaterial{}
	for _, replacement := range replacements {
		material, err := scanMaterial(tx.QueryRow(ctx, `INSERT INTO production_order_materials
			(production_order_id,enterprise_id,material_kind,item_code,mask,substituted_item_code,quantity,warehouse_id,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING `+materialColumns,
			original.ProductionOrderID, enterpriseID, original.Kind, replacement.ItemCode, replacement.Mask,
			original.ItemCode, replacement.Quantity, replacement.WarehouseID, createdBy))
		if err != nil {
			return nil, err
		}
		result = append(result, material)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *ProductionOrderRepositoryPGX) AllocateLots(ctx context.Context, materialID int64, movementKind string, allocations []entity.LotAllocation, createdBy uuid.UUID) ([]entity.LotAllocation, error) {
	return r.AllocateLotsWithPolicy(ctx, materialID, movementKind, allocations, true, createdBy)
}
func (r *ProductionOrderRepositoryPGX) AllocateLotsWithPolicy(ctx context.Context, materialID int64, movementKind string, allocations []entity.LotAllocation, confirmPartial bool, createdBy uuid.UUID) ([]entity.LotAllocation, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	movementKind = strings.ToUpper(strings.TrimSpace(movementKind))
	if movementKind != "REQUISITION" && movementKind != "RETURN" {
		return nil, fmt.Errorf("movement_kind must be REQUISITION or RETURN")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	material, err := scanMaterial(tx.QueryRow(ctx, `SELECT `+materialColumns+` FROM production_order_materials
		WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, materialID, enterpriseID))
	if err != nil {
		return nil, err
	}
	limit := material.Quantity.Sub(material.AttendedQuantity)
	if movementKind == "RETURN" {
		limit = material.AttendedQuantity
	}
	var returnMode string
	var autoIssue bool
	err = tx.QueryRow(ctx, `SELECT lot_return_mode,auto_issue_lots FROM manufacturing_stock_parameters WHERE enterprise_id=$1`, enterpriseID).Scan(&returnMode, &autoIssue)
	if err == pgx.ErrNoRows {
		returnMode = "I"
		autoIssue = false
	} else if err != nil {
		return nil, err
	}
	if movementKind == "REQUISITION" && len(allocations) == 0 && !autoIssue {
		return nil, fmt.Errorf("parameter 53 requires explicit lot selection")
	}
	if movementKind == "RETURN" && returnMode == "A" && len(allocations) == 0 {
		var number int64
		if err := tx.QueryRow(ctx, `SELECT order_number FROM production_orders WHERE id=$1 AND enterprise_id=$2`, material.ProductionOrderID, enterpriseID).Scan(&number); err != nil {
			return nil, err
		}
		allocations = []entity.LotAllocation{{WarehouseID: material.WarehouseID, Lot: fmt.Sprintf("OF-%d", number), Quantity: limit}}
	}
	if movementKind == "REQUISITION" && len(allocations) == 0 {
		selectionWarehouse := material.WarehouseID
		var isWMS bool
		var intermediate *int64
		settingsErr := tx.QueryRow(ctx, `SELECT is_wms,intermediate_out_warehouse_id FROM warehouse_wms_settings
			WHERE enterprise_id=$1 AND warehouse_id=$2`, enterpriseID, material.WarehouseID).Scan(&isWMS, &intermediate)
		if settingsErr != nil && settingsErr != pgx.ErrNoRows {
			return nil, settingsErr
		}
		if isWMS && intermediate == nil {
			return nil, fmt.Errorf("WMS warehouse has no intermediate outbound warehouse")
		}
		if isWMS {
			selectionWarehouse = *intermediate
		}
		rows, err := tx.Query(ctx, `SELECT warehouse_id,lot,quantity FROM stock_lot_balances
			WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND quantity>0
			ORDER BY last_movement_at NULLS FIRST,lot`, enterpriseID, material.ItemCode, material.Mask, selectionWarehouse)
		if err != nil {
			return nil, err
		}
		remaining := limit
		for rows.Next() {
			var allocation entity.LotAllocation
			var balance decimal.Decimal
			if err := rows.Scan(&allocation.WarehouseID, &allocation.Lot, &balance); err != nil {
				rows.Close()
				return nil, err
			}
			allocation.Quantity = balance
			if allocation.Quantity.GreaterThan(remaining) {
				allocation.Quantity = remaining
			}
			if allocation.Quantity.IsPositive() {
				allocations = append(allocations, allocation)
				remaining = remaining.Sub(allocation.Quantity)
			}
			if remaining.IsZero() {
				break
			}
		}
		rows.Close()
		if remaining.IsPositive() {
			return nil, fmt.Errorf("insufficient lot balance to fulfill material quantity")
		}
	}
	total := decimal.Zero
	for _, allocation := range allocations {
		if allocation.Lot == "" || !allocation.Quantity.IsPositive() {
			return nil, fmt.Errorf("lot and positive quantity are required")
		}
		total = total.Add(allocation.Quantity)
		var controlsAddress bool
		_ = tx.QueryRow(ctx, `SELECT controls_address FROM manufacturing_stock_item_controls WHERE enterprise_id=$1 AND item_code=$2`, enterpriseID, material.ItemCode).Scan(&controlsAddress)
		if controlsAddress && (allocation.Address == nil || strings.TrimSpace(*allocation.Address) == "") {
			return nil, fmt.Errorf("stock address is required for item %d", material.ItemCode)
		}
		if allocation.Address != nil {
			var valid bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM manufacturing_warehouse_addresses WHERE enterprise_id=$1 AND warehouse_id=$2 AND address=$3 AND is_active)`, enterpriseID, allocation.WarehouseID, *allocation.Address).Scan(&valid); err != nil || !valid {
				return nil, fmt.Errorf("invalid or inactive stock address")
			}
		}
		if movementKind == "RETURN" && returnMode == "E" {
			var used bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM production_order_lot_allocations WHERE enterprise_id=$1 AND production_order_material_id=$2 AND movement_kind='REQUISITION' AND warehouse_id=$3 AND lot=$4)`, enterpriseID, materialID, allocation.WarehouseID, allocation.Lot).Scan(&used); err != nil || !used {
				return nil, fmt.Errorf("parameter 44 mode E requires a lot used by the material requisition")
			}
		}
		if movementKind == "REQUISITION" {
			var isWMS bool
			var intermediate *int64
			settingsErr := tx.QueryRow(ctx, `SELECT is_wms,intermediate_out_warehouse_id FROM warehouse_wms_settings
				WHERE enterprise_id=$1 AND warehouse_id=$2`, enterpriseID, material.WarehouseID).Scan(&isWMS, &intermediate)
			if settingsErr != nil && settingsErr != pgx.ErrNoRows {
				return nil, settingsErr
			}
			if isWMS && intermediate == nil {
				return nil, fmt.Errorf("WMS warehouse has no intermediate outbound warehouse")
			}
			if isWMS && allocation.WarehouseID != *intermediate {
				return nil, fmt.Errorf("lot must be selected from the WMS intermediate outbound warehouse")
			}
		}
		if movementKind == "REQUISITION" {
			var balance decimal.Decimal
			err := tx.QueryRow(ctx, `SELECT quantity FROM stock_lot_balances WHERE enterprise_id=$1
				AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND lot=$5 FOR UPDATE`,
				enterpriseID, material.ItemCode, material.Mask, allocation.WarehouseID, allocation.Lot).Scan(&balance)
			if err != nil || balance.LessThan(allocation.Quantity) {
				return nil, fmt.Errorf("insufficient balance for lot %s", allocation.Lot)
			}
		}
	}
	if total.GreaterThan(limit) {
		return nil, fmt.Errorf("allocated quantity exceeds material quantity available for %s", strings.ToLower(movementKind))
	}
	if total.LessThan(limit) && !confirmPartial {
		return nil, fmt.Errorf("partial lot allocation requires explicit confirmation")
	}
	if _, err := tx.Exec(ctx, `DELETE FROM production_order_lot_allocations
		WHERE enterprise_id=$1 AND production_order_material_id=$2 AND movement_kind=$3`, enterpriseID, materialID, movementKind); err != nil {
		return nil, err
	}
	result := make([]entity.LotAllocation, 0, len(allocations))
	for _, allocation := range allocations {
		allocation.ProductionOrderMaterialID = materialID
		allocation.MovementKind = movementKind
		allocation.CreatedBy = createdBy
		err := tx.QueryRow(ctx, `INSERT INTO production_order_lot_allocations
			(production_order_material_id,enterprise_id,movement_kind,warehouse_id,lot,address,quantity,created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id,created_at`, materialID, enterpriseID,
			movementKind, allocation.WarehouseID, allocation.Lot, allocation.Address, allocation.Quantity, createdBy).Scan(&allocation.ID, &allocation.CreatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, allocation)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *ProductionOrderRepositoryPGX) AllocateLotsBatch(ctx context.Context, materialIDs []int64, movementKind string, lots []entity.LotAllocation, createdBy uuid.UUID) ([]entity.LotAllocation, error) {
	return r.AllocateLotsBatchWithPolicy(ctx, materialIDs, movementKind, lots, true, createdBy)
}
func (r *ProductionOrderRepositoryPGX) AllocateLotsBatchWithPolicy(ctx context.Context, materialIDs []int64, movementKind string, lots []entity.LotAllocation, confirmPartial bool, createdBy uuid.UUID) ([]entity.LotAllocation, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	movementKind = strings.ToUpper(strings.TrimSpace(movementKind))
	if movementKind != "REQUISITION" && movementKind != "RETURN" {
		return nil, fmt.Errorf("movement_kind must be REQUISITION or RETURN")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT `+qualifiedMaterialColumns+` FROM production_order_materials material
		JOIN production_orders po ON po.id=material.production_order_id
		WHERE material.enterprise_id=$1 AND material.id=ANY($2::bigint[]) ORDER BY po.order_number,material.id FOR UPDATE OF material`, enterpriseID, materialIDs)
	if err != nil {
		return nil, err
	}
	materials := []*entity.ProductionOrderMaterial{}
	for rows.Next() {
		material, err := scanMaterial(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		materials = append(materials, material)
	}
	rows.Close()
	if len(materials) != len(materialIDs) {
		return nil, fmt.Errorf("one or more materials do not belong to the authenticated enterprise")
	}
	itemCode, mask := materials[0].ItemCode, materials[0].Mask
	totalNeed := decimal.Zero
	remaining := map[int64]decimal.Decimal{}
	for _, material := range materials {
		if material.ItemCode != itemCode || material.Mask != mask {
			return nil, fmt.Errorf("batch lot selection requires the same item and mask")
		}
		need := material.Quantity.Sub(material.AttendedQuantity)
		if movementKind == "RETURN" {
			need = material.AttendedQuantity
		}
		remaining[material.ID] = need
		totalNeed = totalNeed.Add(need)
	}
	automaticSelection := len(lots) == 0
	var returnMode string
	var autoIssue bool
	err = tx.QueryRow(ctx, `SELECT lot_return_mode,auto_issue_lots FROM manufacturing_stock_parameters WHERE enterprise_id=$1`, enterpriseID).Scan(&returnMode, &autoIssue)
	if err == pgx.ErrNoRows {
		returnMode = "I"
	} else if err != nil {
		return nil, err
	}
	if movementKind == "REQUISITION" && automaticSelection && !autoIssue {
		return nil, fmt.Errorf("parameter 53 requires explicit lot selection")
	}
	selectionWarehouse := materials[0].WarehouseID
	var isWMS bool
	var intermediate *int64
	settingsErr := tx.QueryRow(ctx, `SELECT is_wms,intermediate_out_warehouse_id FROM warehouse_wms_settings WHERE enterprise_id=$1 AND warehouse_id=$2`, enterpriseID, selectionWarehouse).Scan(&isWMS, &intermediate)
	if settingsErr != nil && settingsErr != pgx.ErrNoRows {
		return nil, settingsErr
	}
	if isWMS && intermediate == nil {
		return nil, fmt.Errorf("WMS warehouse has no intermediate outbound warehouse")
	}
	if isWMS {
		selectionWarehouse = *intermediate
	}
	for _, m := range materials[1:] {
		effective := m.WarehouseID
		var mwms bool
		var mout *int64
		e := tx.QueryRow(ctx, `SELECT is_wms,intermediate_out_warehouse_id FROM warehouse_wms_settings WHERE enterprise_id=$1 AND warehouse_id=$2`, enterpriseID, m.WarehouseID).Scan(&mwms, &mout)
		if e != nil && e != pgx.ErrNoRows {
			return nil, e
		}
		if mwms && mout == nil {
			return nil, fmt.Errorf("WMS warehouse has no intermediate outbound warehouse")
		}
		if mwms {
			effective = *mout
		}
		if effective != selectionWarehouse {
			return nil, fmt.Errorf("batch lot selection requires the same effective warehouse")
		}
	}
	if automaticSelection && movementKind == "RETURN" && returnMode == "A" {
		for _, m := range materials {
			var number int64
			if err := tx.QueryRow(ctx, `SELECT order_number FROM production_orders WHERE id=$1 AND enterprise_id=$2`, m.ProductionOrderID, enterpriseID).Scan(&number); err != nil {
				return nil, err
			}
			lots = append(lots, entity.LotAllocation{WarehouseID: m.WarehouseID, Lot: fmt.Sprintf("OF-%d", number), Quantity: remaining[m.ID]})
		}
	}
	if automaticSelection && movementKind == "REQUISITION" {
		warehouse := selectionWarehouse
		lotRows, err := tx.Query(ctx, `SELECT warehouse_id,lot,quantity FROM stock_lot_balances WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND quantity>0 ORDER BY last_movement_at NULLS FIRST,lot`, enterpriseID, itemCode, mask, warehouse)
		if err != nil {
			return nil, err
		}
		for lotRows.Next() {
			var lot entity.LotAllocation
			if err := lotRows.Scan(&lot.WarehouseID, &lot.Lot, &lot.Quantity); err != nil {
				lotRows.Close()
				return nil, err
			}
			lots = append(lots, lot)
		}
		lotRows.Close()
	}
	if _, err := tx.Exec(ctx, `DELETE FROM production_order_lot_allocations WHERE enterprise_id=$1 AND production_order_material_id=ANY($2::bigint[]) AND movement_kind=$3`, enterpriseID, materialIDs, movementKind); err != nil {
		return nil, err
	}
	result := []entity.LotAllocation{}
	allocatedTotal := decimal.Zero
	for _, lot := range lots {
		if lot.Lot == "" || !lot.Quantity.IsPositive() {
			return nil, fmt.Errorf("lot and positive quantity are required")
		}
		lotRemaining := lot.Quantity
		if isWMS && movementKind == "REQUISITION" && lot.WarehouseID != selectionWarehouse {
			return nil, fmt.Errorf("lot must be selected from the WMS intermediate outbound warehouse")
		}
		if movementKind == "REQUISITION" {
			var balance decimal.Decimal
			if err := tx.QueryRow(ctx, `SELECT quantity FROM stock_lot_balances WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND lot=$5 FOR UPDATE`, enterpriseID, itemCode, mask, lot.WarehouseID, lot.Lot).Scan(&balance); err != nil {
				return nil, err
			}
			if balance.LessThan(lot.Quantity) {
				return nil, fmt.Errorf("insufficient balance for lot %s", lot.Lot)
			}
		}
		for _, material := range materials {
			need := remaining[material.ID]
			if !need.IsPositive() || !lotRemaining.IsPositive() {
				continue
			}
			quantity := need
			if quantity.GreaterThan(lotRemaining) {
				quantity = lotRemaining
			}
			allocation := entity.LotAllocation{ProductionOrderMaterialID: material.ID, MovementKind: movementKind, WarehouseID: lot.WarehouseID, Lot: lot.Lot, Address: lot.Address, Quantity: quantity, CreatedBy: createdBy}
			err := tx.QueryRow(ctx, `INSERT INTO production_order_lot_allocations(production_order_material_id,enterprise_id,movement_kind,warehouse_id,lot,address,quantity,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id,created_at`, material.ID, enterpriseID, movementKind, lot.WarehouseID, lot.Lot, lot.Address, quantity, createdBy).Scan(&allocation.ID, &allocation.CreatedAt)
			if err != nil {
				return nil, err
			}
			result = append(result, allocation)
			remaining[material.ID] = need.Sub(quantity)
			lotRemaining = lotRemaining.Sub(quantity)
			allocatedTotal = allocatedTotal.Add(quantity)
		}
		if lotRemaining.IsPositive() {
			return nil, fmt.Errorf("lot quantities exceed the selected orders' requirements")
		}
	}
	if movementKind == "REQUISITION" && allocatedTotal.LessThan(totalNeed) && automaticSelection {
		return nil, fmt.Errorf("insufficient lot balance")
	}
	if allocatedTotal.LessThan(totalNeed) && !confirmPartial {
		return nil, fmt.Errorf("partial lot allocation requires explicit confirmation")
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *ProductionOrderRepositoryPGX) AddScrapDestination(ctx context.Context, destination *entity.ScrapDestination) (*entity.ScrapDestination, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	result, err := r.addScrapDestinationTx(ctx, tx, enterpriseID, destination)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *ProductionOrderRepositoryPGX) addScrapDestinationTx(ctx context.Context, tx pgx.Tx, enterpriseID int64, destination *entity.ScrapDestination) (*entity.ScrapDestination, error) {
	var err error
	if !destination.Quantity.IsPositive() && !destination.ReturnQuantity.Add(destination.ScrapQuantity).IsPositive() {
		return nil, fmt.Errorf("scrap or return quantity must be positive")
	}
	destination.DestinationKind = strings.ToUpper(strings.TrimSpace(destination.DestinationKind))
	if destination.DestinationKind == "" {
		destination.DestinationKind = "ORDER_ITEM"
	}
	if destination.DestinationKind != "ORDER_ITEM" && destination.DestinationKind != "DEMAND" {
		return nil, fmt.Errorf("destination_kind must be ORDER_ITEM or DEMAND")
	}
	if destination.DestinationKind == "DEMAND" && destination.ProductionOrderMaterialID == nil {
		return nil, fmt.Errorf("demand destination requires production_order_material_id")
	}
	var valued bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM production_order_costs WHERE production_order_id=$1) AND $2 < date_trunc('month',CURRENT_DATE)::date`, destination.ProductionOrderID, destination.DestinationDate).Scan(&valued); err != nil {
		return nil, err
	}
	if valued {
		return nil, fmt.Errorf("valued production order cannot receive a destination in a prior period")
	}
	if destination.ReturnQuantity.IsZero() && destination.ScrapQuantity.IsZero() {
		destination.ScrapQuantity = destination.Quantity
	}
	destination.Quantity = destination.ReturnQuantity.Add(destination.ScrapQuantity)
	if !destination.Quantity.IsPositive() {
		return nil, fmt.Errorf("return_quantity or scrap_quantity must be positive")
	}
	var closed bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM manufacturing_stock_closed_periods WHERE enterprise_id=$1 AND $2 BETWEEN period_from AND period_to)`, enterpriseID, destination.DestinationDate).Scan(&closed); err != nil {
		return nil, err
	}
	if closed {
		return nil, fmt.Errorf("destination date belongs to a closed stock period")
	}
	var inInterval bool
	err = tx.QueryRow(ctx, `SELECT CASE WHEN movement_from IS NULL AND movement_to IS NULL THEN TRUE ELSE $2 BETWEEN COALESCE(movement_from,'-infinity'::date) AND COALESCE(movement_to,'infinity'::date) END FROM manufacturing_stock_parameters WHERE enterprise_id=$1`, enterpriseID, destination.DestinationDate).Scan(&inInterval)
	if err == pgx.ErrNoRows {
		inInterval = true
	} else if err != nil {
		return nil, err
	}
	if !inInterval {
		return nil, fmt.Errorf("destination date is outside the accounting stock movement interval")
	}
	var sourceItem int64
	if destination.ProductionOrderMaterialID == nil {
		if err := tx.QueryRow(ctx, `SELECT item_code FROM production_orders WHERE id=$1 AND enterprise_id=$2`, destination.ProductionOrderID, enterpriseID).Scan(&sourceItem); err != nil {
			return nil, err
		}
	} else {
		if err := tx.QueryRow(ctx, `SELECT item_code FROM production_order_materials WHERE id=$1 AND production_order_id=$2 AND enterprise_id=$3`, *destination.ProductionOrderMaterialID, destination.ProductionOrderID, enterpriseID).Scan(&sourceItem); err != nil {
			return nil, err
		}
	}
	if destination.DestinationKind == "DEMAND" {
		var sourceLot, sourceAddress bool
		_ = tx.QueryRow(ctx, `SELECT controls_lot,controls_address FROM manufacturing_stock_item_controls WHERE enterprise_id=$1 AND item_code=$2`, enterpriseID, sourceItem).Scan(&sourceLot, &sourceAddress)
		if sourceLot && (destination.Lot == nil || strings.TrimSpace(*destination.Lot) == "") {
			return nil, fmt.Errorf("lot is required for demand return")
		}
		if sourceAddress && (destination.Address == nil || strings.TrimSpace(*destination.Address) == "") {
			return nil, fmt.Errorf("address is required for demand return")
		}
		if destination.Lot != nil {
			var used bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM production_order_lot_allocations WHERE enterprise_id=$1 AND production_order_material_id=$2 AND movement_kind='REQUISITION' AND lot=$3)`, enterpriseID, *destination.ProductionOrderMaterialID, *destination.Lot).Scan(&used); err != nil || !used {
				return nil, fmt.Errorf("demand return lot must have been used in its requisition")
			}
		}
		if destination.Address != nil {
			var used bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM production_order_lot_allocations WHERE enterprise_id=$1 AND production_order_material_id=$2 AND movement_kind='REQUISITION' AND address=$3)`, enterpriseID, *destination.ProductionOrderMaterialID, *destination.Address).Scan(&used); err != nil || !used {
				return nil, fmt.Errorf("demand return address must have been used in its requisition")
			}
		}
	}
	var scrapGroup, scrapUOM string
	var controlsLot, controlsAddress bool
	err = tx.QueryRow(ctx, `SELECT inventory_group_type,stock_uom,controls_lot,controls_address FROM manufacturing_stock_item_controls WHERE enterprise_id=$1 AND item_code=$2`, enterpriseID, destination.ScrapItemCode).Scan(&scrapGroup, &scrapUOM, &controlsLot, &controlsAddress)
	if err != nil {
		return nil, fmt.Errorf("scrap item stock controls are required: %w", err)
	}
	if scrapGroup != "SECONDARY_MATERIAL" {
		return nil, fmt.Errorf("scrap item must belong to the secondary material inventory group")
	}
	if controlsLot && (destination.Lot == nil || strings.TrimSpace(*destination.Lot) == "") {
		return nil, fmt.Errorf("lot is required for scrap item")
	}
	if controlsAddress && (destination.Address == nil || strings.TrimSpace(*destination.Address) == "") {
		return nil, fmt.Errorf("stock address is required for scrap item")
	}
	if destination.Address != nil {
		var valid bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM manufacturing_warehouse_addresses WHERE enterprise_id=$1 AND warehouse_id=$2 AND address=$3 AND is_active)`, enterpriseID, destination.WarehouseID, *destination.Address).Scan(&valid); err != nil || !valid {
			return nil, fmt.Errorf("invalid or inactive stock address")
		}
	}
	if destination.ScrapUOM == "" {
		destination.ScrapUOM = scrapUOM
	}
	if destination.SourceUOM == "" {
		_ = tx.QueryRow(ctx, `SELECT stock_uom FROM manufacturing_stock_item_controls WHERE enterprise_id=$1 AND item_code=$2`, enterpriseID, sourceItem).Scan(&destination.SourceUOM)
		if destination.SourceUOM == "" {
			destination.SourceUOM = destination.ScrapUOM
		}
	}
	convertedScrap := destination.ScrapQuantity
	if destination.ScrapQuantity.IsPositive() && destination.SourceUOM != destination.ScrapUOM {
		var factor decimal.Decimal
		err := tx.QueryRow(ctx, `SELECT factor FROM item_unit_conversions WHERE item_code=$1 AND from_uom=$2 AND to_uom=$3 AND is_active UNION ALL SELECT 1/factor FROM item_unit_conversions WHERE item_code=$1 AND from_uom=$3 AND to_uom=$2 AND is_active LIMIT 1`, destination.ScrapItemCode, destination.SourceUOM, destination.ScrapUOM).Scan(&factor)
		if err != nil {
			return nil, fmt.Errorf("unit conversion from %s to %s is required", destination.SourceUOM, destination.ScrapUOM)
		}
		convertedScrap = convertedScrap.Mul(factor)
	}
	var limit decimal.Decimal
	if destination.ProductionOrderMaterialID == nil {
		err = tx.QueryRow(ctx, `SELECT scrapped_qty FROM production_orders WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, destination.ProductionOrderID, enterpriseID).Scan(&limit)
	} else {
		err = tx.QueryRow(ctx, `SELECT material.attended_quantity FROM production_order_materials material
			WHERE material.id=$1 AND material.production_order_id=$2 AND material.enterprise_id=$3 FOR UPDATE`, *destination.ProductionOrderMaterialID, destination.ProductionOrderID, enterpriseID).Scan(&limit)
	}
	if err != nil {
		return nil, err
	}
	var already decimal.Decimal
	err = tx.QueryRow(ctx, `SELECT COALESCE(SUM(return_quantity+scrap_quantity),0) FROM production_order_scrap_destinations
		WHERE enterprise_id=$1 AND production_order_id=$2 AND production_order_material_id IS NOT DISTINCT FROM $3`, enterpriseID, destination.ProductionOrderID, destination.ProductionOrderMaterialID).Scan(&already)
	if err != nil {
		return nil, err
	}
	if already.Add(destination.Quantity).GreaterThan(limit) {
		return nil, fmt.Errorf("scrap destination quantity exceeds pending scrapped quantity")
	}
	if destination.DestinationKind == "ORDER_ITEM" && already.Add(destination.Quantity).Equal(limit) {
		var demandExists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM production_order_scrap_destinations WHERE enterprise_id=$1 AND production_order_id=$2 AND destination_kind='DEMAND')`, enterpriseID, destination.ProductionOrderID).Scan(&demandExists); err != nil {
			return nil, err
		}
		if demandExists {
			return nil, fmt.Errorf("all order scrap cannot be destined while demand destinations exist")
		}
	}
	if destination.DestinationKind == "DEMAND" {
		var allOrderDestined bool
		if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(return_quantity+scrap_quantity),0)>0 FROM production_order_scrap_destinations WHERE enterprise_id=$1 AND production_order_id=$2 AND destination_kind='ORDER_ITEM'`, enterpriseID, destination.ProductionOrderID).Scan(&allOrderDestined); err != nil {
			return nil, err
		}
		if allOrderDestined {
			return nil, fmt.Errorf("demand scrap cannot be destined after order item destination")
		}
	}
	if destination.ID > 0 {
		err = tx.QueryRow(ctx, `INSERT INTO production_order_scrap_destinations
		(id,production_order_id,production_order_material_id,enterprise_id,scrap_item_code,warehouse_id,lot,address,quantity,destination_date,created_by,destination_kind,return_quantity,scrap_quantity,source_uom,scrap_uom)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16) RETURNING created_at`, destination.ID, destination.ProductionOrderID, destination.ProductionOrderMaterialID, enterpriseID, destination.ScrapItemCode, destination.WarehouseID, destination.Lot, destination.Address, destination.Quantity, destination.DestinationDate, destination.CreatedBy, destination.DestinationKind, destination.ReturnQuantity, destination.ScrapQuantity, destination.SourceUOM, destination.ScrapUOM).Scan(&destination.CreatedAt)
	} else {
		err = tx.QueryRow(ctx, `INSERT INTO production_order_scrap_destinations
		(production_order_id,production_order_material_id,enterprise_id,scrap_item_code,warehouse_id,lot,address,quantity,destination_date,created_by,destination_kind,return_quantity,scrap_quantity,source_uom,scrap_uom)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING id,created_at`, destination.ProductionOrderID,
			destination.ProductionOrderMaterialID, enterpriseID, destination.ScrapItemCode, destination.WarehouseID,
			destination.Lot, destination.Address, destination.Quantity, destination.DestinationDate, destination.CreatedBy, destination.DestinationKind, destination.ReturnQuantity, destination.ScrapQuantity, destination.SourceUOM, destination.ScrapUOM).Scan(&destination.ID, &destination.CreatedAt)
	}
	if err != nil {
		return nil, err
	}
	referenceType, referenceCode := "PRODUCTION_SCRAP", destination.ID
	if destination.ScrapQuantity.IsPositive() {
		quantityFloat, _ := convertedScrap.Float64()
		movement := &stockentity.StockMovement{ItemCode: destination.ScrapItemCode, WarehouseID: destination.WarehouseID, MovementType: stockentity.MovementTypeIn, Quantity: quantityFloat, ExactQuantity: convertedScrap, ReferenceType: &referenceType, ReferenceCode: &referenceCode, Lot: destination.Lot, CreatedBy: destination.CreatedBy}
		if err := stockrepository.CreateMovementTx(ctx, tx, enterpriseID, movement); err != nil {
			return nil, err
		}
	}
	if destination.ReturnQuantity.IsPositive() {
		quantityFloat, _ := destination.ReturnQuantity.Float64()
		movement := &stockentity.StockMovement{ItemCode: sourceItem, WarehouseID: destination.WarehouseID, MovementType: stockentity.MovementTypeIn, Quantity: quantityFloat, ExactQuantity: destination.ReturnQuantity, ReferenceType: &referenceType, ReferenceCode: &referenceCode, Lot: destination.Lot, CreatedBy: destination.CreatedBy}
		if err := stockrepository.CreateMovementTx(ctx, tx, enterpriseID, movement); err != nil {
			return nil, err
		}
	}
	return destination, nil
}

func (r *ProductionOrderRepositoryPGX) DeleteScrapDestination(ctx context.Context, id int64, createdBy uuid.UUID) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := r.removeScrapDestinationTx(ctx, tx, enterpriseID, id, createdBy); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *ProductionOrderRepositoryPGX) removeScrapDestinationTx(ctx context.Context, tx pgx.Tx, enterpriseID, id int64, createdBy uuid.UUID) error {
	var err error
	var date time.Time
	if err := tx.QueryRow(ctx, `SELECT destination_date FROM production_order_scrap_destinations WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, id, enterpriseID).Scan(&date); err != nil {
		return err
	}
	var allowed bool
	err = tx.QueryRow(ctx, `SELECT NOT EXISTS(SELECT 1 FROM manufacturing_stock_closed_periods WHERE enterprise_id=$1 AND $2 BETWEEN period_from AND period_to) AND COALESCE((SELECT CASE WHEN movement_from IS NULL AND movement_to IS NULL THEN TRUE ELSE $2 BETWEEN COALESCE(movement_from,'-infinity'::date) AND COALESCE(movement_to,'infinity'::date) END FROM manufacturing_stock_parameters WHERE enterprise_id=$1),TRUE)`, enterpriseID, date).Scan(&allowed)
	if err != nil || !allowed {
		return fmt.Errorf("scrap destination belongs to a closed or disallowed stock period")
	}
	rows, err := tx.Query(ctx, `SELECT item_code,mask,warehouse_id,quantity,lot FROM stock_movements WHERE enterprise_id=$1 AND reference_type='PRODUCTION_SCRAP' AND reference_code=$2 AND movement_type='IN' FOR UPDATE`, enterpriseID, id)
	if err != nil {
		return err
	}
	movements := []stockentity.StockMovement{}
	for rows.Next() {
		var m stockentity.StockMovement
		if err := rows.Scan(&m.ItemCode, &m.Mask, &m.WarehouseID, &m.ExactQuantity, &m.Lot); err != nil {
			rows.Close()
			return err
		}
		movements = append(movements, m)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for i := range movements {
		m := &movements[i]
		var balance decimal.Decimal
		if err := tx.QueryRow(ctx, `SELECT quantity FROM stock_balances WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 FOR UPDATE`, enterpriseID, m.ItemCode, m.Mask, m.WarehouseID).Scan(&balance); err != nil || balance.LessThan(m.ExactQuantity) {
			return fmt.Errorf("scrap destination cannot be removed because it would make stock negative (balance=%s required=%s: %v)", balance, m.ExactQuantity, err)
		}
	}
	referenceType, referenceCode := "PRODUCTION_SCRAP_REVERSAL", id
	for i := range movements {
		m := &movements[i]
		q, _ := m.ExactQuantity.Float64()
		m.Quantity = q
		m.MovementType = stockentity.MovementTypeOut
		m.ReferenceType = &referenceType
		m.ReferenceCode = &referenceCode
		m.CreatedBy = createdBy
		if err := stockrepository.CreateMovementTx(ctx, tx, enterpriseID, m); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE stock_movements SET reference_type='PRODUCTION_SCRAP_REVERSED' WHERE enterprise_id=$1 AND reference_type='PRODUCTION_SCRAP' AND reference_code=$2 AND movement_type='IN'`, enterpriseID, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM production_order_scrap_destinations WHERE id=$1 AND enterprise_id=$2`, id, enterpriseID); err != nil {
		return err
	}
	return nil
}

func (r *ProductionOrderRepositoryPGX) UpdateScrapDestination(ctx context.Context, id int64, destination *entity.ScrapDestination) (*entity.ScrapDestination, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT true FROM production_order_scrap_destinations WHERE id=$1 AND enterprise_id=$2 FOR UPDATE`, id, enterpriseID).Scan(&exists); err != nil {
		return nil, err
	}
	if err := r.removeScrapDestinationTx(ctx, tx, enterpriseID, id, destination.CreatedBy); err != nil {
		return nil, err
	}
	destination.ID = id
	result, err := r.addScrapDestinationTx(ctx, tx, enterpriseID, destination)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *ProductionOrderRepositoryPGX) HasProductionActivity(ctx context.Context, productionOrderID int64) (bool, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var exists bool
	err = r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM production_orders po WHERE po.id=$1 AND po.enterprise_id=$2 AND (
		EXISTS(SELECT 1 FROM production_appointments a WHERE a.production_order_id=po.id) OR
		EXISTS(SELECT 1 FROM production_consumptions c WHERE c.production_order_id=po.id) OR
		EXISTS(SELECT 1 FROM stock_movements m WHERE m.enterprise_id=$2 AND m.reference_type='PRODUCTION_ORDER' AND m.reference_code=po.id) OR
		EXISTS(SELECT 1 FROM production_order_materials material JOIN production_order_wms_requests request ON request.production_order_material_id=material.id
			WHERE material.production_order_id=po.id AND request.enterprise_id=$2 AND request.status<>'CANCELLED')))`, productionOrderID, enterpriseID).Scan(&exists)
	return exists, err
}

func (r *ProductionOrderRepositoryPGX) CanChangeOrderQuantity(ctx context.Context, productionOrderID int64) (bool, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var allowed bool
	err = r.pool.QueryRow(ctx, `SELECT po.allow_quantity_change AND po.origin_type NOT IN ('KANBAN','COMMERCIAL')
		AND COALESCE((SELECT UPPER(value) IN ('S','SIM','1','TRUE','YES') FROM planning_params WHERE enterprise_id=$2 AND (param_number=10 OR param_key='PERMITE_ALTERAR_QUANTIDADE_OF') LIMIT 1),TRUE)
		AND NOT EXISTS(SELECT 1 FROM production_order_service_links WHERE enterprise_id=$2 AND production_order_id=po.id)
		AND NOT EXISTS(SELECT 1 FROM production_order_materials m JOIN production_order_wms_requests w ON w.production_order_material_id=m.id WHERE m.production_order_id=po.id AND w.enterprise_id=$2 AND w.status<>'CANCELLED')
		FROM production_orders po WHERE po.id=$1 AND po.enterprise_id=$2`, productionOrderID, enterpriseID).Scan(&allowed)
	return allowed, err
}
func (r *ProductionOrderRepositoryPGX) CanChangeOrderDates(ctx context.Context, productionOrderID int64) (bool, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var allowed bool
	err = r.pool.QueryRow(ctx, `SELECT po.allow_date_change AND po.origin_type NOT IN ('KANBAN','COMMERCIAL')
		AND CASE WHEN po.origin_type='MRP' THEN COALESCE((SELECT UPPER(value) IN ('S','SIM','1','TRUE','YES') FROM planning_params WHERE enterprise_id=$2 AND (param_number=14 OR param_key='PERMITE_ALTERAR_DATAS_OF') LIMIT 1),FALSE) ELSE TRUE END
		FROM production_orders po WHERE po.id=$1 AND po.enterprise_id=$2`, productionOrderID, enterpriseID).Scan(&allowed)
	return allowed, err
}
func (r *ProductionOrderRepositoryPGX) AcceptsFractionalQuantity(ctx context.Context, itemCode int64) (bool, error) {
	if _, err := tenant.ID(ctx); err != nil {
		return false, err
	}
	var allowed bool
	err := r.pool.QueryRow(ctx, `SELECT accepts_fractional_quantity FROM items WHERE code=$1`, itemCode).Scan(&allowed)
	return allowed, err
}

func (r *ProductionOrderRepositoryPGX) UpsertWMSSettings(ctx context.Context, settings entity.WMSWarehouseSettings) (*entity.WMSWarehouseSettings, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if settings.IsWMS && settings.IntermediateOutWarehouseID == nil {
		return nil, fmt.Errorf("WMS warehouse requires an intermediate outbound warehouse")
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO warehouse_wms_settings(enterprise_id,warehouse_id,is_wms,intermediate_out_warehouse_id) VALUES($1,$2,$3,$4) ON CONFLICT(enterprise_id,warehouse_id) DO UPDATE SET is_wms=EXCLUDED.is_wms,intermediate_out_warehouse_id=EXCLUDED.intermediate_out_warehouse_id`, enterpriseID, settings.WarehouseID, settings.IsWMS, settings.IntermediateOutWarehouseID)
	if err != nil {
		return nil, err
	}
	return &settings, nil
}

func (r *ProductionOrderRepositoryPGX) ConfigureManufacturingStock(ctx context.Context, p entity.ManufacturingStockParameters) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO manufacturing_stock_parameters(enterprise_id,lot_return_mode,auto_issue_lots,movement_from,movement_to) VALUES($1,$2,$3,$4,$5) ON CONFLICT(enterprise_id) DO UPDATE SET lot_return_mode=EXCLUDED.lot_return_mode,auto_issue_lots=EXCLUDED.auto_issue_lots,movement_from=EXCLUDED.movement_from,movement_to=EXCLUDED.movement_to`, enterpriseID, p.LotReturnMode, p.AutoIssueLots, p.MovementFrom, p.MovementTo)
	return err
}
func (r *ProductionOrderRepositoryPGX) ConfigureManufacturingItemStock(ctx context.Context, c entity.ManufacturingItemStockControl) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	if c.AutomaticIssueType == "" {
		c.AutomaticIssueType = "ISSUE"
	}
	command, err := r.pool.Exec(ctx, `INSERT INTO manufacturing_stock_item_controls(enterprise_id,item_code,stock_uom,controls_lot,controls_address,inventory_group_type,automatic_issue_type,line_warehouse_id) SELECT $1,$2,$3,$4,$5,$6,$7,$8 WHERE EXISTS(SELECT 1 FROM items WHERE code=$2) ON CONFLICT(enterprise_id,item_code) DO UPDATE SET stock_uom=EXCLUDED.stock_uom,controls_lot=EXCLUDED.controls_lot,controls_address=EXCLUDED.controls_address,inventory_group_type=EXCLUDED.inventory_group_type,automatic_issue_type=EXCLUDED.automatic_issue_type,line_warehouse_id=EXCLUDED.line_warehouse_id,updated_at=NOW()`, enterpriseID, c.ItemCode, c.StockUOM, c.ControlsLot, c.ControlsAddress, c.InventoryGroupType, c.AutomaticIssueType, c.LineWarehouseID)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
func (r *ProductionOrderRepositoryPGX) ConfigureWarehouseAddress(ctx context.Context, warehouseID int64, address string, active bool) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO manufacturing_warehouse_addresses(enterprise_id,warehouse_id,address,is_active) VALUES($1,$2,$3,$4) ON CONFLICT(enterprise_id,warehouse_id,address) DO UPDATE SET is_active=EXCLUDED.is_active`, enterpriseID, warehouseID, address, active)
	return err
}
func (r *ProductionOrderRepositoryPGX) ConfigureTemporaryLot(ctx context.Context, lot entity.TemporaryProductionLot) (*entity.TemporaryProductionLot, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	command, err := r.pool.Exec(ctx, `UPDATE production_orders po SET temporary_lot_code=$1,temporary_lot_manufactured_on=$2,temporary_lot_expires_on=$3,updated_at=NOW() WHERE po.id=$4 AND po.enterprise_id=$5 AND po.origin_type NOT IN('KANBAN','COMMERCIAL') AND NOT EXISTS(SELECT 1 FROM production_appointments a WHERE a.production_order_id=po.id) AND NOT EXISTS(SELECT 1 FROM production_consumptions c WHERE c.production_order_id=po.id)`, lot.Lot, lot.ManufacturedOn, lot.ExpiresOn, lot.ProductionOrderID, enterpriseID)
	if err != nil {
		return nil, err
	}
	if command.RowsAffected() == 0 {
		return nil, fmt.Errorf("temporary lot cannot be changed for moved, Kanban or commercial order")
	}
	return &lot, nil
}
func (r *ProductionOrderRepositoryPGX) GetMaintenance(ctx context.Context, id *int64) ([]entity.ProductionOrderMaintenanceView, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	args := []any{enterpriseID}
	where := "enterprise_id=$1 AND origin_type NOT IN ('KANBAN','COMMERCIAL')"
	if id != nil {
		args = append(args, *id)
		where = "enterprise_id=$1 AND id=$2"
	}
	rows, err := r.pool.Query(ctx, `SELECT id,order_number,planned_order_id,item_code,mask,planned_qty,produced_qty,scrapped_qty,
		status,start_date,end_date,machine_id,cost_center_id,employee_id,priority,notes,is_active,created_at,updated_at,created_by,warehouse_id,
		origin_type,temporary_lot_code,temporary_lot_manufactured_on,temporary_lot_expires_on
		FROM production_orders WHERE `+where+` ORDER BY order_number`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []entity.ProductionOrderMaintenanceView{}
	for rows.Next() {
		var order entity.ProductionOrder
		var origin, status string
		var code *string
		var manufactured, expires *time.Time
		if err := rows.Scan(&order.ID, &order.OrderNumber, &order.PlannedOrderID, &order.ItemCode, &order.Mask,
			&order.PlannedQty, &order.ProducedQty, &order.ScrappedQty, &status, &order.StartDate, &order.EndDate,
			&order.MachineID, &order.CostCenterID, &order.EmployeeID, &order.Priority, &order.Notes, &order.IsActive,
			&order.CreatedAt, &order.UpdatedAt, &order.CreatedBy, &order.WarehouseID,
			&origin, &code, &manufactured, &expires); err != nil {
			return nil, err
		}
		order.Status = entity.ProductionOrderStatus(status)
		if origin == "KANBAN" {
			return nil, fmt.Errorf("this order was generated by Kanban activation and cannot be maintained; use order consultation")
		}
		if origin == "COMMERCIAL" {
			return nil, fmt.Errorf("commercial production orders cannot be maintained")
		}
		view := entity.ProductionOrderMaintenanceView{ProductionOrder: &order, OriginType: origin, OrderType: maintenanceOrderType(origin, order.Status), Rework: order.Notes != nil && strings.Contains(*order.Notes, "ORDEM DE RETRABALHO")}
		if code != nil && manufactured != nil && expires != nil {
			view.TemporaryLot = &entity.TemporaryProductionLot{ProductionOrderID: order.ID, Lot: *code, ManufacturedOn: *manufactured, ExpiresOn: *expires}
		}
		result = append(result, view)
	}
	return result, rows.Err()
}
func maintenanceOrderType(origin string, status entity.ProductionOrderStatus) string {
	if status == entity.StatusCompleted || status == entity.StatusClosed {
		return "OFE"
	}
	if status == entity.StatusOpen && origin == "MRP" {
		return "OFF"
	}
	if origin == "MRP" {
		return "OFA"
	}
	return "OFM"
}
