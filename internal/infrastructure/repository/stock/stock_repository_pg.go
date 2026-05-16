package stock

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type StockRepositorySQLC struct {
	pool *pgxpool.Pool
}

func NewStockRepositorySQLC(pool *pgxpool.Pool) *StockRepositorySQLC {
	return &StockRepositorySQLC{pool: pool}
}

var _ repository.StockRepository = (*StockRepositorySQLC)(nil)

// ---------- Stock Movements ----------

func (r *StockRepositorySQLC) CreateMovement(ctx context.Context, m *entity.StockMovement) (*entity.StockMovement, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO public.stock_movements
			(item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
			 reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		 RETURNING id, created_at`,
		m.ItemCode, m.Mask, m.WarehouseID, m.MovementType, m.Quantity, m.UnitPrice, m.TotalPrice,
		m.ReferenceType, m.ReferenceCode, m.Lot, m.SerialNumber, m.Batch, m.ExpirationDate, m.Notes, m.CreatedBy,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating stock movement: %w", err)
	}
	return m, nil
}

func (r *StockRepositorySQLC) ListMovements(ctx context.Context) ([]*entity.StockMovement, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
		        reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, created_at, created_by
		 FROM public.stock_movements ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing stock movements: %w", err)
	}
	defer rows.Close()
	return scanMovements(rows)
}

func (r *StockRepositorySQLC) ListMovementsByItem(ctx context.Context, itemCode int64) ([]*entity.StockMovement, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
		        reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, created_at, created_by
		 FROM public.stock_movements WHERE item_code = $1 ORDER BY created_at DESC`, itemCode)
	if err != nil {
		return nil, fmt.Errorf("listing stock movements by item: %w", err)
	}
	defer rows.Close()
	return scanMovements(rows)
}

func (r *StockRepositorySQLC) ListMovementsByWarehouse(ctx context.Context, warehouseID int64) ([]*entity.StockMovement, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
		        reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, created_at, created_by
		 FROM public.stock_movements WHERE warehouse_id = $1 ORDER BY created_at DESC`, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("listing stock movements by warehouse: %w", err)
	}
	defer rows.Close()
	return scanMovements(rows)
}

func (r *StockRepositorySQLC) ListMovementsByDateRange(ctx context.Context, from, to time.Time) ([]*entity.StockMovement, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
		        reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, created_at, created_by
		 FROM public.stock_movements WHERE created_at >= $1 AND created_at <= $2 ORDER BY created_at DESC`, from, to)
	if err != nil {
		return nil, fmt.Errorf("listing stock movements by date range: %w", err)
	}
	defer rows.Close()
	return scanMovements(rows)
}

func scanMovements(rows pgx.Rows) ([]*entity.StockMovement, error) {
	var result []*entity.StockMovement
	for rows.Next() {
		var m entity.StockMovement
		if err := rows.Scan(
			&m.ID, &m.ItemCode, &m.Mask, &m.WarehouseID, &m.MovementType, &m.Quantity, &m.UnitPrice, &m.TotalPrice,
			&m.ReferenceType, &m.ReferenceCode, &m.Lot, &m.SerialNumber, &m.Batch, &m.ExpirationDate, &m.Notes, &m.CreatedAt, &m.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scanning stock movement: %w", err)
		}
		result = append(result, &m)
	}
	return result, rows.Err()
}

// ---------- Stock Balance ----------

func (r *StockRepositorySQLC) GetBalance(ctx context.Context, itemCode int64, mask string, warehouseID int64) (*entity.StockBalance, error) {
	var b entity.StockBalance
	err := r.pool.QueryRow(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reserved_qty, available_qty,
		        minimum_stock, maximum_stock, safety_stock, avg_cost, last_cost, total_cost,
		        last_movement_at, updated_at
		 FROM public.stock_balances WHERE item_code = $1 AND mask = $2 AND warehouse_id = $3`,
		itemCode, mask, warehouseID,
	).Scan(&b.ID, &b.ItemCode, &b.Mask, &b.WarehouseID, &b.Quantity, &b.ReservedQty, &b.AvailableQty,
		&b.MinimumStock, &b.MaximumStock, &b.SafetyStock, &b.AvgCost, &b.LastCost, &b.TotalCost,
		&b.LastMovementAt, &b.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("stock balance not found for item %d mask %s warehouse %d", itemCode, mask, warehouseID)
		}
		return nil, fmt.Errorf("getting stock balance: %w", err)
	}
	return &b, nil
}

func (r *StockRepositorySQLC) ListBalances(ctx context.Context) ([]*entity.StockBalance, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reserved_qty, available_qty,
		        minimum_stock, maximum_stock, safety_stock, avg_cost, last_cost, total_cost,
		        last_movement_at, updated_at
		 FROM public.stock_balances ORDER BY item_code`)
	if err != nil {
		return nil, fmt.Errorf("listing stock balances: %w", err)
	}
	defer rows.Close()
	return scanBalances(rows)
}

func (r *StockRepositorySQLC) ListBalancesByWarehouse(ctx context.Context, warehouseID int64) ([]*entity.StockBalance, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reserved_qty, available_qty,
		        minimum_stock, maximum_stock, safety_stock, avg_cost, last_cost, total_cost,
		        last_movement_at, updated_at
		 FROM public.stock_balances WHERE warehouse_id = $1 ORDER BY item_code`, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("listing stock balances by warehouse: %w", err)
	}
	defer rows.Close()
	return scanBalances(rows)
}

func (r *StockRepositorySQLC) ListBalancesByItem(ctx context.Context, itemCode int64) ([]*entity.StockBalance, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reserved_qty, available_qty,
		        minimum_stock, maximum_stock, safety_stock, avg_cost, last_cost, total_cost,
		        last_movement_at, updated_at
		 FROM public.stock_balances WHERE item_code = $1 ORDER BY warehouse_id`, itemCode)
	if err != nil {
		return nil, fmt.Errorf("listing stock balances by item: %w", err)
	}
	defer rows.Close()
	return scanBalances(rows)
}

func (r *StockRepositorySQLC) UpsertBalance(ctx context.Context, b *entity.StockBalance) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.stock_balances (item_code, mask, warehouse_id, quantity, reserved_qty,
		     minimum_stock, maximum_stock, safety_stock, avg_cost, last_cost, total_cost, last_movement_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		 ON CONFLICT (item_code, mask, warehouse_id) DO UPDATE SET
		     quantity = EXCLUDED.quantity,
		     reserved_qty = EXCLUDED.reserved_qty,
		     minimum_stock = EXCLUDED.minimum_stock,
		     maximum_stock = EXCLUDED.maximum_stock,
		     safety_stock = EXCLUDED.safety_stock,
		     avg_cost = EXCLUDED.avg_cost,
		     last_cost = EXCLUDED.last_cost,
		     total_cost = EXCLUDED.total_cost,
		     last_movement_at = EXCLUDED.last_movement_at,
		     updated_at = NOW()`,
		b.ItemCode, b.Mask, b.WarehouseID, b.Quantity, b.ReservedQty,
		b.MinimumStock, b.MaximumStock, b.SafetyStock, b.AvgCost, b.LastCost, b.TotalCost, b.LastMovementAt)
	if err != nil {
		return fmt.Errorf("upserting stock balance: %w", err)
	}
	return nil
}

func scanBalances(rows pgx.Rows) ([]*entity.StockBalance, error) {
	var result []*entity.StockBalance
	for rows.Next() {
		var b entity.StockBalance
		if err := rows.Scan(
			&b.ID, &b.ItemCode, &b.Mask, &b.WarehouseID, &b.Quantity, &b.ReservedQty, &b.AvailableQty,
			&b.MinimumStock, &b.MaximumStock, &b.SafetyStock, &b.AvgCost, &b.LastCost, &b.TotalCost,
			&b.LastMovementAt, &b.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning stock balance: %w", err)
		}
		result = append(result, &b)
	}
	return result, rows.Err()
}

// ---------- Stock Reservations ----------

func (r *StockRepositorySQLC) CreateReservation(ctx context.Context, res *entity.StockReservation) (*entity.StockReservation, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO public.stock_reservations
			(item_code, mask, warehouse_id, quantity, reference_type, reference_code, reference_item_code,
			 reservation_date, expiration_date, status, notes, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		 RETURNING id, created_at, updated_at`,
		res.ItemCode, res.Mask, res.WarehouseID, res.Quantity, res.ReferenceType, res.ReferenceCode,
		res.ReferenceItemCode, res.ReservationDate, res.ExpirationDate, res.Status, res.Notes, res.CreatedBy,
	).Scan(&res.ID, &res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating stock reservation: %w", err)
	}
	return res, nil
}

func (r *StockRepositorySQLC) GetReservation(ctx context.Context, id int64) (*entity.StockReservation, error) {
	var res entity.StockReservation
	err := r.pool.QueryRow(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reference_type, reference_code, reference_item_code,
		        reservation_date, expiration_date, status, notes, created_at, updated_at, created_by
		 FROM public.stock_reservations WHERE id = $1`, id,
	).Scan(&res.ID, &res.ItemCode, &res.Mask, &res.WarehouseID, &res.Quantity, &res.ReferenceType,
		&res.ReferenceCode, &res.ReferenceItemCode, &res.ReservationDate, &res.ExpirationDate,
		&res.Status, &res.Notes, &res.CreatedAt, &res.UpdatedAt, &res.CreatedBy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("stock reservation %d not found", id)
		}
		return nil, fmt.Errorf("getting stock reservation: %w", err)
	}
	return &res, nil
}

func (r *StockRepositorySQLC) ListReservations(ctx context.Context) ([]*entity.StockReservation, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reference_type, reference_code, reference_item_code,
		        reservation_date, expiration_date, status, notes, created_at, updated_at, created_by
		 FROM public.stock_reservations ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing reservations: %w", err)
	}
	defer rows.Close()
	return scanReservations(rows)
}

func (r *StockRepositorySQLC) ListReservationsByItem(ctx context.Context, itemCode int64) ([]*entity.StockReservation, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reference_type, reference_code, reference_item_code,
		        reservation_date, expiration_date, status, notes, created_at, updated_at, created_by
		 FROM public.stock_reservations WHERE item_code = $1 ORDER BY created_at DESC`, itemCode)
	if err != nil {
		return nil, fmt.Errorf("listing reservations by item: %w", err)
	}
	defer rows.Close()
	return scanReservations(rows)
}

func (r *StockRepositorySQLC) ListActiveReservations(ctx context.Context) ([]*entity.StockReservation, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reference_type, reference_code, reference_item_code,
		        reservation_date, expiration_date, status, notes, created_at, updated_at, created_by
		 FROM public.stock_reservations WHERE status = 'ACTIVE' ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing active reservations: %w", err)
	}
	defer rows.Close()
	return scanReservations(rows)
}

func (r *StockRepositorySQLC) CancelReservation(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.stock_reservations SET status = 'CANCELLED', updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("cancelling reservation %d: %w", id, err)
	}
	return nil
}

func (r *StockRepositorySQLC) ConsumeReservation(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.stock_reservations SET status = 'CONSUMED', updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("consuming reservation %d: %w", id, err)
	}
	return nil
}

func scanReservations(rows pgx.Rows) ([]*entity.StockReservation, error) {
	var result []*entity.StockReservation
	for rows.Next() {
		var res entity.StockReservation
		if err := rows.Scan(
			&res.ID, &res.ItemCode, &res.Mask, &res.WarehouseID, &res.Quantity, &res.ReferenceType,
			&res.ReferenceCode, &res.ReferenceItemCode, &res.ReservationDate, &res.ExpirationDate,
			&res.Status, &res.Notes, &res.CreatedAt, &res.UpdatedAt, &res.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scanning reservation: %w", err)
		}
		result = append(result, &res)
	}
	return result, rows.Err()
}

// ---------- Physical Inventory ----------

func (r *StockRepositorySQLC) CreateInventory(ctx context.Context, inv *entity.PhysicalInventory) (*entity.PhysicalInventory, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO public.physical_inventories
			(code, description, warehouse_id, start_date, end_date, status, notes, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING id, created_at, updated_at`,
		inv.Code, inv.Description, inv.WarehouseID, inv.StartDate, inv.EndDate, inv.Status, inv.Notes, inv.CreatedBy,
	).Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating physical inventory: %w", err)
	}
	return inv, nil
}

func (r *StockRepositorySQLC) GetInventory(ctx context.Context, id int64) (*entity.PhysicalInventory, error) {
	var inv entity.PhysicalInventory
	err := r.pool.QueryRow(ctx,
		`SELECT id, code, description, warehouse_id, start_date, end_date, status,
		        total_items, counted_items, notes, created_at, updated_at, created_by
		 FROM public.physical_inventories WHERE id = $1`, id,
	).Scan(&inv.ID, &inv.Code, &inv.Description, &inv.WarehouseID, &inv.StartDate, &inv.EndDate,
		&inv.Status, &inv.TotalItems, &inv.CountedItems, &inv.Notes, &inv.CreatedAt, &inv.UpdatedAt, &inv.CreatedBy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("physical inventory %d not found", id)
		}
		return nil, fmt.Errorf("getting physical inventory: %w", err)
	}
	return &inv, nil
}

func (r *StockRepositorySQLC) GetInventoryByCode(ctx context.Context, code int64) (*entity.PhysicalInventory, error) {
	var inv entity.PhysicalInventory
	err := r.pool.QueryRow(ctx,
		`SELECT id, code, description, warehouse_id, start_date, end_date, status,
		        total_items, counted_items, notes, created_at, updated_at, created_by
		 FROM public.physical_inventories WHERE code = $1`, code,
	).Scan(&inv.ID, &inv.Code, &inv.Description, &inv.WarehouseID, &inv.StartDate, &inv.EndDate,
		&inv.Status, &inv.TotalItems, &inv.CountedItems, &inv.Notes, &inv.CreatedAt, &inv.UpdatedAt, &inv.CreatedBy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("physical inventory %d not found", code)
		}
		return nil, fmt.Errorf("getting physical inventory by code: %w", err)
	}
	return &inv, nil
}

func (r *StockRepositorySQLC) ListInventories(ctx context.Context) ([]*entity.PhysicalInventory, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, code, description, warehouse_id, start_date, end_date, status,
		        total_items, counted_items, notes, created_at, updated_at, created_by
		 FROM public.physical_inventories ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing physical inventories: %w", err)
	}
	defer rows.Close()
	return scanInventories(rows)
}

func (r *StockRepositorySQLC) ListInventoriesByStatus(ctx context.Context, status string) ([]*entity.PhysicalInventory, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, code, description, warehouse_id, start_date, end_date, status,
		        total_items, counted_items, notes, created_at, updated_at, created_by
		 FROM public.physical_inventories WHERE status = $1 ORDER BY created_at DESC`, status)
	if err != nil {
		return nil, fmt.Errorf("listing physical inventories by status: %w", err)
	}
	defer rows.Close()
	return scanInventories(rows)
}

func (r *StockRepositorySQLC) UpdateInventoryStatus(ctx context.Context, id int64, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.physical_inventories SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	if err != nil {
		return fmt.Errorf("updating inventory status: %w", err)
	}
	return nil
}

func (r *StockRepositorySQLC) CloseInventory(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.physical_inventories SET status = 'CLOSED', end_date = CURRENT_DATE, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("closing inventory: %w", err)
	}
	return nil
}

func scanInventories(rows pgx.Rows) ([]*entity.PhysicalInventory, error) {
	var result []*entity.PhysicalInventory
	for rows.Next() {
		var inv entity.PhysicalInventory
		if err := rows.Scan(
			&inv.ID, &inv.Code, &inv.Description, &inv.WarehouseID, &inv.StartDate, &inv.EndDate,
			&inv.Status, &inv.TotalItems, &inv.CountedItems, &inv.Notes, &inv.CreatedAt, &inv.UpdatedAt, &inv.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scanning physical inventory: %w", err)
		}
		result = append(result, &inv)
	}
	return result, rows.Err()
}

// ---------- Physical Inventory Items ----------

func (r *StockRepositorySQLC) UpsertInventoryItem(ctx context.Context, item *entity.PhysicalInventoryItem) error {
	var existingID int64
	err := r.pool.QueryRow(ctx,
		`SELECT id FROM public.physical_inventory_items
		 WHERE inventory_id = $1 AND item_code = $2 AND mask = $3 AND warehouse_id = $4`,
		item.InventoryID, item.ItemCode, item.Mask, item.WarehouseID,
	).Scan(&existingID)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("checking existing inventory item: %w", err)
	}
	if err == pgx.ErrNoRows {
		_, err = r.pool.Exec(ctx,
			`INSERT INTO public.physical_inventory_items
				(inventory_id, item_code, mask, warehouse_id, system_qty)
			 VALUES ($1,$2,$3,$4,$5)`,
			item.InventoryID, item.ItemCode, item.Mask, item.WarehouseID, item.SystemQty)
	} else {
		_, err = r.pool.Exec(ctx,
			`UPDATE public.physical_inventory_items SET system_qty = $1 WHERE id = $2`,
			item.SystemQty, existingID)
	}
	if err != nil {
		return fmt.Errorf("upserting inventory item: %w", err)
	}
	return nil
}

func (r *StockRepositorySQLC) ListInventoryItems(ctx context.Context, inventoryID int64) ([]*entity.PhysicalInventoryItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, inventory_id, item_code, mask, warehouse_id, system_qty, counted_qty,
		        difference_qty, unit_cost, adjustment_type, adjustment_reason, counted_by,
		        counted_at, is_adjusted, created_at
		 FROM public.physical_inventory_items WHERE inventory_id = $1 ORDER BY item_code`, inventoryID)
	if err != nil {
		return nil, fmt.Errorf("listing inventory items: %w", err)
	}
	defer rows.Close()
	return scanInventoryItems(rows)
}

func (r *StockRepositorySQLC) CountInventoryItem(ctx context.Context, item *entity.PhysicalInventoryItem) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.physical_inventory_items SET
		     counted_qty = $1, unit_cost = $2, counted_by = $3, counted_at = NOW()
		 WHERE inventory_id = $4 AND item_code = $5 AND mask = $6 AND warehouse_id = $7`,
		item.CountedQty, item.UnitCost, item.CountedBy, item.InventoryID, item.ItemCode, item.Mask, item.WarehouseID)
	if err != nil {
		return fmt.Errorf("counting inventory item: %w", err)
	}
	return nil
}

func (r *StockRepositorySQLC) AdjustInventoryItem(ctx context.Context, item *entity.PhysicalInventoryItem) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.physical_inventory_items SET
		     adjustment_type = $1, adjustment_reason = $2, is_adjusted = true
		 WHERE inventory_id = $3 AND item_code = $4 AND mask = $5 AND warehouse_id = $6`,
		item.AdjustmentType, item.AdjustmentReason, item.InventoryID, item.ItemCode, item.Mask, item.WarehouseID)
	if err != nil {
		return fmt.Errorf("adjusting inventory item: %w", err)
	}
	return nil
}

func scanInventoryItems(rows pgx.Rows) ([]*entity.PhysicalInventoryItem, error) {
	var result []*entity.PhysicalInventoryItem
	for rows.Next() {
		var item entity.PhysicalInventoryItem
		if err := rows.Scan(
			&item.ID, &item.InventoryID, &item.ItemCode, &item.Mask, &item.WarehouseID, &item.SystemQty,
			&item.CountedQty, &item.DifferenceQty, &item.UnitCost, &item.AdjustmentType, &item.AdjustmentReason,
			&item.CountedBy, &item.CountedAt, &item.IsAdjusted, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning inventory item: %w", err)
		}
		result = append(result, &item)
	}
	return result, rows.Err()
}
