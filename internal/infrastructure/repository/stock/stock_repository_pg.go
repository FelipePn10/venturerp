package stock

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type StockRepositorySQLC struct {
	pool *pgxpool.Pool
}

func NewStockRepositorySQLC(pool *pgxpool.Pool) *StockRepositorySQLC {
	return &StockRepositorySQLC{pool: pool}
}

var _ repository.StockRepository = (*StockRepositorySQLC)(nil)

// ---------- Stock Movements ----------

// CreateMovement records the movement and atomically updates the stock balance
// snapshot (on-hand quantity, weighted average cost and last cost) in the same
// transaction, so balances always reflect the movements that were posted.
func (r *StockRepositorySQLC) CreateMovement(ctx context.Context, m *entity.StockMovement) (*entity.StockMovement, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning stock movement tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = CreateMovementTx(ctx, tx, enterpriseID, m)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing stock movement: %w", err)
	}
	return m, nil
}

// CreateMovementTx records a movement and updates its aggregate balances using
// the caller's transaction. It is exported for infrastructure coordinators
// that must settle production and stock atomically.
func CreateMovementTx(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) error {
	err := tx.QueryRow(ctx,
		`INSERT INTO public.stock_movements
			(item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
			 reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, created_by, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		 RETURNING id, created_at`,
		m.ItemCode, m.Mask, m.WarehouseID, m.MovementType, movementQuantity(m), m.UnitPrice, m.TotalPrice,
		m.ReferenceType, m.ReferenceCode, m.Lot, m.SerialNumber, m.Batch, m.ExpirationDate, m.Notes, m.CreatedBy, enterpriseID,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return fmt.Errorf("creating stock movement: %w", err)
	}

	if err := applyMovementToBalance(ctx, tx, enterpriseID, m); err != nil {
		return err
	}

	if err := applyMovementToLotBalance(ctx, tx, enterpriseID, m); err != nil {
		return err
	}
	return nil
}

// applyMovementToLotBalance keeps the lot-segregated balance in sync when a
// movement carries a lot, so a metallurgy shop can tell how much of each heat
// remains in each warehouse. Movements without a lot are ignored here.
func applyMovementToLotBalance(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) error {
	if m.Lot == nil || *m.Lot == "" {
		return nil
	}
	delta := signedDecimalQuantity(m.MovementType, movementQuantity(m))
	if delta.IsZero() {
		return nil
	}
	lastCost := m.UnitPrice
	_, err := tx.Exec(ctx,
		`INSERT INTO public.stock_lot_balances
			(item_code, mask, warehouse_id, lot, quantity, last_cost, last_movement_at, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 ON CONFLICT (enterprise_id, item_code, mask, warehouse_id, lot) WHERE enterprise_id IS NOT NULL DO UPDATE SET
			 quantity = public.stock_lot_balances.quantity + EXCLUDED.quantity,
			 last_cost = CASE WHEN EXCLUDED.last_cost > 0 THEN EXCLUDED.last_cost ELSE public.stock_lot_balances.last_cost END,
			 last_movement_at = EXCLUDED.last_movement_at,
			 updated_at = NOW()`,
		m.ItemCode, m.Mask, m.WarehouseID, *m.Lot, delta, lastCost, m.CreatedAt, enterpriseID)
	if err != nil {
		return fmt.Errorf("updating lot balance: %w", err)
	}
	return nil
}

// applyMovementToBalance updates stock_balances within the given transaction
// according to the movement direction. Weighted average cost is recomputed on
// inbound movements; outbound movements consume at the current average cost.
func applyMovementToBalance(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) error {
	deltaExact := signedDecimalQuantity(m.MovementType, movementQuantity(m))
	if deltaExact.IsZero() {
		// Movement type does not affect on-hand quantity (e.g. reservation).
		return nil
	}

	var qtyExact decimal.Decimal
	var avgCost, totalCost float64
	exists := true
	err := tx.QueryRow(ctx,
		`SELECT quantity, avg_cost, total_cost FROM public.stock_balances
		 WHERE item_code=$1 AND mask=$2 AND warehouse_id=$3 AND enterprise_id=$4 FOR UPDATE`,
		m.ItemCode, m.Mask, m.WarehouseID, enterpriseID,
	).Scan(&qtyExact, &avgCost, &totalCost)
	if err == pgx.ErrNoRows {
		exists = false
	} else if err != nil {
		return fmt.Errorf("reading stock balance for update: %w", err)
	}

	// Weighted-average costing is computed by the domain (single source of truth,
	// unit-tested); the repository only persists the result.
	qty, _ := qtyExact.Float64()
	delta, _ := deltaExact.Float64()
	next, lastCost := entity.ApplyMovementCosting(
		entity.CostingState{Quantity: qty, AvgCost: avgCost, TotalCost: totalCost}, delta, m.UnitPrice,
	)
	newQty, newAvg, newTotal := qtyExact.Add(deltaExact), next.AvgCost, next.TotalCost

	if exists {
		_, err = tx.Exec(ctx,
			`UPDATE public.stock_balances
			 SET quantity=$4, avg_cost=$5, last_cost=$6, total_cost=$7, last_movement_at=$8, updated_at=NOW()
			 WHERE item_code=$1 AND mask=$2 AND warehouse_id=$3 AND enterprise_id=$9`,
			m.ItemCode, m.Mask, m.WarehouseID, newQty, newAvg, lastCost, newTotal, m.CreatedAt, enterpriseID)
	} else {
		_, err = tx.Exec(ctx,
			`INSERT INTO public.stock_balances
				(item_code, mask, warehouse_id, quantity, avg_cost, last_cost, total_cost, last_movement_at, enterprise_id)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			m.ItemCode, m.Mask, m.WarehouseID, newQty, newAvg, lastCost, newTotal, m.CreatedAt, enterpriseID)
	}
	if err != nil {
		return fmt.Errorf("updating stock balance: %w", err)
	}
	return nil
}

func movementQuantity(m *entity.StockMovement) decimal.Decimal {
	if !m.ExactQuantity.IsZero() {
		return m.ExactQuantity
	}
	return decimal.NewFromFloat(m.Quantity)
}

func signedDecimalQuantity(movementType string, quantity decimal.Decimal) decimal.Decimal {
	sign := entity.SignedQuantity(movementType, 1)
	if sign < 0 {
		return quantity.Neg()
	}
	if sign > 0 {
		return quantity
	}
	return decimal.Zero
}

func (r *StockRepositorySQLC) ListMovements(ctx context.Context) ([]*entity.StockMovement, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
		        reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, created_at, created_by
		 FROM public.stock_movements WHERE enterprise_id=$1 ORDER BY created_at DESC`, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing stock movements: %w", err)
	}
	defer rows.Close()
	return scanMovements(rows)
}

func (r *StockRepositorySQLC) ListMovementsByItem(ctx context.Context, itemCode int64) ([]*entity.StockMovement, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
		        reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, created_at, created_by
		 FROM public.stock_movements WHERE item_code = $1 AND enterprise_id=$2 ORDER BY created_at DESC`, itemCode, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing stock movements by item: %w", err)
	}
	defer rows.Close()
	return scanMovements(rows)
}

func (r *StockRepositorySQLC) ListMovementsByWarehouse(ctx context.Context, warehouseID int64) ([]*entity.StockMovement, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
		        reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, created_at, created_by
		 FROM public.stock_movements WHERE warehouse_id = $1 AND enterprise_id=$2 ORDER BY created_at DESC`, warehouseID, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing stock movements by warehouse: %w", err)
	}
	defer rows.Close()
	return scanMovements(rows)
}

func (r *StockRepositorySQLC) ListMovementsByDateRange(ctx context.Context, from, to time.Time) ([]*entity.StockMovement, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
		        reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, created_at, created_by
		 FROM public.stock_movements WHERE created_at >= $1 AND created_at <= $2 AND enterprise_id=$3 ORDER BY created_at DESC`, from, to, enterpriseID)
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
	enterpriseID, tenantErr := tenant.ID(ctx)
	if tenantErr != nil {
		return nil, tenantErr
	}
	var b entity.StockBalance
	err := r.pool.QueryRow(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reserved_qty, available_qty,
		        minimum_stock, maximum_stock, safety_stock, avg_cost, last_cost, total_cost,
		        last_movement_at, updated_at
		 FROM public.stock_balances WHERE item_code = $1 AND mask = $2 AND warehouse_id = $3 AND enterprise_id=$4`,
		itemCode, mask, warehouseID, enterpriseID,
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
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reserved_qty, available_qty,
		        minimum_stock, maximum_stock, safety_stock, avg_cost, last_cost, total_cost,
		        last_movement_at, updated_at
		 FROM public.stock_balances WHERE enterprise_id=$1 ORDER BY item_code`, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing stock balances: %w", err)
	}
	defer rows.Close()
	return scanBalances(rows)
}

func (r *StockRepositorySQLC) ListBalancesByWarehouse(ctx context.Context, warehouseID int64) ([]*entity.StockBalance, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reserved_qty, available_qty,
		        minimum_stock, maximum_stock, safety_stock, avg_cost, last_cost, total_cost,
		        last_movement_at, updated_at
		 FROM public.stock_balances WHERE warehouse_id = $1 AND enterprise_id=$2 ORDER BY item_code`, warehouseID, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing stock balances by warehouse: %w", err)
	}
	defer rows.Close()
	return scanBalances(rows)
}

func (r *StockRepositorySQLC) ListBalancesByItem(ctx context.Context, itemCode int64) ([]*entity.StockBalance, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reserved_qty, available_qty,
		        minimum_stock, maximum_stock, safety_stock, avg_cost, last_cost, total_cost,
		        last_movement_at, updated_at
		 FROM public.stock_balances WHERE item_code = $1 AND enterprise_id=$2 ORDER BY warehouse_id`, itemCode, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing stock balances by item: %w", err)
	}
	defer rows.Close()
	return scanBalances(rows)
}

func (r *StockRepositorySQLC) UpsertBalance(ctx context.Context, b *entity.StockBalance) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO public.stock_balances (item_code, mask, warehouse_id, quantity, reserved_qty,
		     minimum_stock, maximum_stock, safety_stock, avg_cost, last_cost, total_cost, last_movement_at, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		 ON CONFLICT (enterprise_id, item_code, mask, warehouse_id) WHERE enterprise_id IS NOT NULL DO UPDATE SET
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
		b.MinimumStock, b.MaximumStock, b.SafetyStock, b.AvgCost, b.LastCost, b.TotalCost, b.LastMovementAt, enterpriseID)
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

// CreateReservation records the reservation and atomically increases the
// reserved quantity of the balance, so available_qty (= quantity − reserved_qty)
// reflects the reservation immediately.
func (r *StockRepositorySQLC) CreateReservation(ctx context.Context, res *entity.StockReservation) (*entity.StockReservation, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning reservation tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx,
		`INSERT INTO public.stock_reservations
			(item_code, mask, warehouse_id, quantity, reference_type, reference_code, reference_item_code,
			 reservation_date, expiration_date, status, notes, created_by, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		 RETURNING id, created_at, updated_at`,
		res.ItemCode, res.Mask, res.WarehouseID, res.Quantity, res.ReferenceType, res.ReferenceCode,
		res.ReferenceItemCode, res.ReservationDate, res.ExpirationDate, res.Status, res.Notes, res.CreatedBy, enterpriseID,
	).Scan(&res.ID, &res.CreatedAt, &res.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating stock reservation: %w", err)
	}

	if res.Status == "ACTIVE" {
		if err := adjustReservedTx(ctx, tx, enterpriseID, res.ItemCode, res.Mask, res.WarehouseID, res.Quantity); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing reservation: %w", err)
	}
	return res, nil
}

// adjustReservedTx adds delta to reserved_qty of a balance within a transaction,
// creating the balance row if it does not exist yet.
func adjustReservedTx(ctx context.Context, tx pgx.Tx, enterpriseID, itemCode int64, mask string, warehouseID int64, delta float64) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO public.stock_balances (item_code, mask, warehouse_id, reserved_qty, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (enterprise_id,item_code,mask,warehouse_id) WHERE enterprise_id IS NOT NULL DO UPDATE
		   SET reserved_qty = GREATEST(public.stock_balances.reserved_qty + EXCLUDED.reserved_qty, 0),
		       updated_at = NOW()`,
		itemCode, mask, warehouseID, delta, enterpriseID)
	if err != nil {
		return fmt.Errorf("adjusting reserved quantity: %w", err)
	}
	return nil
}

func (r *StockRepositorySQLC) HasActiveReservationByReference(ctx context.Context, referenceType string, referenceCode int64) (bool, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return false, err
	}
	var exists bool
	err = r.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM public.stock_reservations
			WHERE reference_type = $1 AND reference_code = $2 AND status = 'ACTIVE' AND enterprise_id=$3
		 )`, referenceType, referenceCode, enterpriseID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("checking active reservations: %w", err)
	}
	return exists, nil
}

func (r *StockRepositorySQLC) GetReservation(ctx context.Context, id int64) (*entity.StockReservation, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var res entity.StockReservation
	err = r.pool.QueryRow(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reference_type, reference_code, reference_item_code,
		        reservation_date, expiration_date, status, notes, created_at, updated_at, created_by
		 FROM public.stock_reservations WHERE id = $1 AND enterprise_id=$2`, id, enterpriseID,
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
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reference_type, reference_code, reference_item_code,
		        reservation_date, expiration_date, status, notes, created_at, updated_at, created_by
		 FROM public.stock_reservations WHERE enterprise_id=$1 ORDER BY created_at DESC`, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing reservations: %w", err)
	}
	defer rows.Close()
	return scanReservations(rows)
}

func (r *StockRepositorySQLC) ListReservationsByItem(ctx context.Context, itemCode int64) ([]*entity.StockReservation, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reference_type, reference_code, reference_item_code,
		        reservation_date, expiration_date, status, notes, created_at, updated_at, created_by
		 FROM public.stock_reservations WHERE item_code = $1 AND enterprise_id=$2 ORDER BY created_at DESC`, itemCode, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing reservations by item: %w", err)
	}
	defer rows.Close()
	return scanReservations(rows)
}

func (r *StockRepositorySQLC) ListActiveReservations(ctx context.Context) ([]*entity.StockReservation, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, quantity, reference_type, reference_code, reference_item_code,
		        reservation_date, expiration_date, status, notes, created_at, updated_at, created_by
		 FROM public.stock_reservations WHERE status = 'ACTIVE' AND enterprise_id=$1 ORDER BY created_at DESC`, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing active reservations: %w", err)
	}
	defer rows.Close()
	return scanReservations(rows)
}

func (r *StockRepositorySQLC) CancelReservation(ctx context.Context, id int64) error {
	return r.closeReservation(ctx, id, "CANCELLED")
}

func (r *StockRepositorySQLC) ConsumeReservation(ctx context.Context, id int64) error {
	return r.closeReservation(ctx, id, "CONSUMED")
}

// closeReservation moves a reservation to a terminal status and, if it was still
// ACTIVE, releases its quantity from the balance's reserved_qty so available_qty
// is restored. No-op on the reserved_qty if the reservation was already closed.
func (r *StockRepositorySQLC) closeReservation(ctx context.Context, id int64, status string) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning reservation close tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var itemCode, warehouseID int64
	var mask, prevStatus string
	var qty float64
	err = tx.QueryRow(ctx,
		`SELECT item_code, mask, warehouse_id, quantity, status
		 FROM public.stock_reservations WHERE id = $1 AND enterprise_id=$2 FOR UPDATE`, id, enterpriseID,
	).Scan(&itemCode, &mask, &warehouseID, &qty, &prevStatus)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("stock reservation %d not found", id)
		}
		return fmt.Errorf("reading reservation %d: %w", id, err)
	}

	if _, err = tx.Exec(ctx,
		`UPDATE public.stock_reservations SET status = $2, updated_at = NOW() WHERE id = $1 AND enterprise_id=$3`, id, status, enterpriseID); err != nil {
		return fmt.Errorf("closing reservation %d: %w", id, err)
	}

	if prevStatus == "ACTIVE" {
		if err := adjustReservedTx(ctx, tx, enterpriseID, itemCode, mask, warehouseID, -qty); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing reservation close: %w", err)
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

// ---------- Consumption Average ----------

// RecalcConsumptionAverage computes the average monthly consumption of an item
// from its outbound movements over the trailing window and upserts the result.
func (r *StockRepositorySQLC) RecalcConsumptionAverage(ctx context.Context, itemCode int64, windowMonths int) (*entity.ItemConsumptionAverage, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if windowMonths <= 0 {
		windowMonths = 6
	}

	var totalConsumed float64
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(quantity), 0)
		 FROM public.stock_movements
		 WHERE item_code = $1
		   AND movement_type IN ('OUT', 'TRANSFER_OUT')
		   AND created_at >= NOW() - make_interval(months => $2)
		   AND enterprise_id=$3`, itemCode, windowMonths, enterpriseID).Scan(&totalConsumed)
	if err != nil {
		return nil, fmt.Errorf("summing item consumption: %w", err)
	}

	avg := totalConsumed / float64(windowMonths)

	var out entity.ItemConsumptionAverage
	err = r.pool.QueryRow(ctx,
		`INSERT INTO public.item_consumption_averages
			(item_code, avg_monthly_consumption, total_consumed, window_months, calculated_at, enterprise_id)
		 VALUES ($1,$2,$3,$4,NOW(),$5)
		 ON CONFLICT (enterprise_id,item_code) DO UPDATE SET
			 avg_monthly_consumption = EXCLUDED.avg_monthly_consumption,
			 total_consumed          = EXCLUDED.total_consumed,
			 window_months           = EXCLUDED.window_months,
			 calculated_at           = NOW()
		 RETURNING id, item_code, avg_monthly_consumption, total_consumed, window_months, calculated_at`,
		itemCode, avg, totalConsumed, windowMonths, enterpriseID,
	).Scan(&out.ID, &out.ItemCode, &out.AvgMonthlyConsumption, &out.TotalConsumed, &out.WindowMonths, &out.CalculatedAt)
	if err != nil {
		return nil, fmt.Errorf("upserting consumption average: %w", err)
	}
	return &out, nil
}

// RecalcAllConsumptionAverages recomputes the average for every item that had
// outbound movements within the window, returning how many items were updated.
func (r *StockRepositorySQLC) RecalcAllConsumptionAverages(ctx context.Context, windowMonths int) (int, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	if windowMonths <= 0 {
		windowMonths = 6
	}
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT item_code FROM public.stock_movements
		 WHERE movement_type IN ('OUT', 'TRANSFER_OUT')
		   AND created_at >= NOW() - make_interval(months => $1) AND enterprise_id=$2`, windowMonths, enterpriseID)
	if err != nil {
		return 0, fmt.Errorf("listing items with consumption: %w", err)
	}
	var items []int64
	for rows.Next() {
		var code int64
		if err := rows.Scan(&code); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scanning item code: %w", err)
		}
		items = append(items, code)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	count := 0
	for _, code := range items {
		if _, err := r.RecalcConsumptionAverage(ctx, code, windowMonths); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (r *StockRepositorySQLC) GetConsumptionAverage(ctx context.Context, itemCode int64) (*entity.ItemConsumptionAverage, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var out entity.ItemConsumptionAverage
	err = r.pool.QueryRow(ctx,
		`SELECT id, item_code, avg_monthly_consumption, total_consumed, window_months, calculated_at
		 FROM public.item_consumption_averages WHERE item_code = $1 AND enterprise_id=$2`, itemCode, enterpriseID,
	).Scan(&out.ID, &out.ItemCode, &out.AvgMonthlyConsumption, &out.TotalConsumed, &out.WindowMonths, &out.CalculatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("consumption average not found for item %d", itemCode)
		}
		return nil, fmt.Errorf("getting consumption average: %w", err)
	}
	return &out, nil
}

// ---------- Lot Traceability ----------

func (r *StockRepositorySQLC) UpsertLot(ctx context.Context, lot *entity.StockLot) (*entity.StockLot, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx,
		`INSERT INTO public.stock_lots
			(item_code, lot, heat_number, certificate, supplier_code, received_at, notes, created_by, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 ON CONFLICT (enterprise_id,item_code,lot) DO UPDATE SET
			 heat_number   = EXCLUDED.heat_number,
			 certificate   = EXCLUDED.certificate,
			 supplier_code = EXCLUDED.supplier_code,
			 received_at   = EXCLUDED.received_at,
			 notes         = EXCLUDED.notes
		 RETURNING id, created_at`,
		lot.ItemCode, lot.Lot, lot.HeatNumber, lot.Certificate, lot.SupplierCode, lot.ReceivedAt, lot.Notes, lot.CreatedBy, enterpriseID,
	).Scan(&lot.ID, &lot.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("upserting stock lot: %w", err)
	}
	return lot, nil
}

func (r *StockRepositorySQLC) GetLot(ctx context.Context, itemCode int64, lot string) (*entity.StockLot, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var l entity.StockLot
	err = r.pool.QueryRow(ctx,
		`SELECT id, item_code, lot, heat_number, certificate, supplier_code, received_at, notes, created_at, created_by
		 FROM public.stock_lots WHERE item_code = $1 AND lot = $2 AND enterprise_id=$3`, itemCode, lot, enterpriseID,
	).Scan(&l.ID, &l.ItemCode, &l.Lot, &l.HeatNumber, &l.Certificate, &l.SupplierCode, &l.ReceivedAt, &l.Notes, &l.CreatedAt, &l.CreatedBy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("getting stock lot: %w", err)
	}
	return &l, nil
}

func (r *StockRepositorySQLC) ListLotBalancesByItem(ctx context.Context, itemCode int64) ([]*entity.StockLotBalance, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, lot, quantity, last_cost, last_movement_at, updated_at
		 FROM public.stock_lot_balances WHERE item_code = $1 AND enterprise_id=$2 ORDER BY lot, warehouse_id`, itemCode, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing lot balances: %w", err)
	}
	defer rows.Close()
	return scanLotBalances(rows)
}

func scanLotBalances(rows pgx.Rows) ([]*entity.StockLotBalance, error) {
	var out []*entity.StockLotBalance
	for rows.Next() {
		var b entity.StockLotBalance
		if err := rows.Scan(&b.ID, &b.ItemCode, &b.Mask, &b.WarehouseID, &b.Lot, &b.Quantity, &b.LastCost, &b.LastMovementAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning lot balance: %w", err)
		}
		out = append(out, &b)
	}
	return out, rows.Err()
}

// GetLotGenealogy traces an item lot in both directions: the production orders
// that consumed it (where this raw material went) and the production orders that
// produced it together with the input lots that compose it.
func (r *StockRepositorySQLC) GetLotGenealogy(ctx context.Context, itemCode int64, lot string) (*entity.LotGenealogy, error) {
	enterpriseID, tenantErr := tenant.ID(ctx)
	if tenantErr != nil {
		return nil, tenantErr
	}
	g := &entity.LotGenealogy{ItemCode: itemCode, Lot: lot}

	registry, err := r.GetLot(ctx, itemCode, lot)
	if err != nil {
		return nil, err
	}
	g.Registry = registry

	balRows, err := r.pool.Query(ctx,
		`SELECT id, item_code, mask, warehouse_id, lot, quantity, last_cost, last_movement_at, updated_at
		 FROM public.stock_lot_balances WHERE item_code = $1 AND lot = $2 AND enterprise_id=$3 ORDER BY warehouse_id`, itemCode, lot, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing lot genealogy balances: %w", err)
	}
	balances, err := scanLotBalances(balRows)
	balRows.Close()
	if err != nil {
		return nil, err
	}
	g.Balances = balances

	// Forward: production orders that consumed this lot.
	consRows, err := r.pool.Query(ctx,
		`SELECT pc.production_order_id, po.order_number, po.item_code, pc.consumed_qty
		 FROM public.production_consumptions pc
		 JOIN public.production_orders po ON po.id = pc.production_order_id
		 WHERE pc.item_code = $1 AND pc.lot = $2 AND po.enterprise_id=$3
		 ORDER BY pc.production_order_id`, itemCode, lot, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing lot consumptions: %w", err)
	}
	for consRows.Next() {
		var c entity.LotConsumption
		if err := consRows.Scan(&c.ProductionOrderID, &c.OrderNumber, &c.ProducedItemCode, &c.ConsumedQty); err != nil {
			consRows.Close()
			return nil, fmt.Errorf("scanning lot consumption: %w", err)
		}
		g.ConsumedIn = append(g.ConsumedIn, c)
	}
	consRows.Close()

	// Backward: production orders that produced this lot (recorded on the IN
	// movement) and the input lots that went into each of them.
	prodRows, err := r.pool.Query(ctx,
		`SELECT sm.reference_code, po.order_number, COALESCE(SUM(sm.quantity), 0)
		 FROM public.stock_movements sm
		 JOIN public.production_orders po ON po.id = sm.reference_code
		 WHERE sm.item_code = $1 AND sm.lot = $2
		   AND sm.reference_type = 'PRODUCTION_ORDER' AND sm.movement_type = 'IN'
		   AND sm.enterprise_id=$3 AND po.enterprise_id=$3
		 GROUP BY sm.reference_code, po.order_number
		 ORDER BY sm.reference_code`, itemCode, lot, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing lot productions: %w", err)
	}
	var productions []entity.LotProduction
	for prodRows.Next() {
		var p entity.LotProduction
		if err := prodRows.Scan(&p.ProductionOrderID, &p.OrderNumber, &p.ProducedQty); err != nil {
			prodRows.Close()
			return nil, fmt.Errorf("scanning lot production: %w", err)
		}
		productions = append(productions, p)
	}
	prodRows.Close()

	for i := range productions {
		inputRows, err := r.pool.Query(ctx,
			`SELECT pc.item_code, COALESCE(pc.lot, ''), pc.consumed_qty
			 FROM public.production_consumptions pc
			 JOIN public.production_orders po ON po.id=pc.production_order_id
			 WHERE pc.production_order_id = $1 AND pc.lot IS NOT NULL AND pc.lot <> '' AND po.enterprise_id=$2
			 ORDER BY pc.item_code`, productions[i].ProductionOrderID, enterpriseID)
		if err != nil {
			return nil, fmt.Errorf("listing lot inputs: %w", err)
		}
		for inputRows.Next() {
			var in entity.LotInput
			if err := inputRows.Scan(&in.ItemCode, &in.Lot, &in.ConsumedQty); err != nil {
				inputRows.Close()
				return nil, fmt.Errorf("scanning lot input: %w", err)
			}
			productions[i].InputLots = append(productions[i].InputLots, in)
		}
		inputRows.Close()
	}
	g.ProducedBy = productions

	return g, nil
}

// ---------- Physical Inventory ----------

func (r *StockRepositorySQLC) CreateInventory(ctx context.Context, inv *entity.PhysicalInventory) (*entity.PhysicalInventory, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx,
		`INSERT INTO public.physical_inventories
			(code, description, warehouse_id, start_date, end_date, status, notes, created_by, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING id, created_at, updated_at`,
		inv.Code, inv.Description, inv.WarehouseID, inv.StartDate, inv.EndDate, inv.Status, inv.Notes, inv.CreatedBy, enterpriseID,
	).Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating physical inventory: %w", err)
	}
	return inv, nil
}

func (r *StockRepositorySQLC) GetInventory(ctx context.Context, id int64) (*entity.PhysicalInventory, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var inv entity.PhysicalInventory
	err = r.pool.QueryRow(ctx,
		`SELECT id, code, description, warehouse_id, start_date, end_date, status,
		        total_items, counted_items, notes, created_at, updated_at, created_by
		 FROM public.physical_inventories WHERE id = $1 AND enterprise_id=$2`, id, enterpriseID,
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
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	var inv entity.PhysicalInventory
	err = r.pool.QueryRow(ctx,
		`SELECT id, code, description, warehouse_id, start_date, end_date, status,
		        total_items, counted_items, notes, created_at, updated_at, created_by
		 FROM public.physical_inventories WHERE code = $1 AND enterprise_id=$2`, code, enterpriseID,
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
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, code, description, warehouse_id, start_date, end_date, status,
		        total_items, counted_items, notes, created_at, updated_at, created_by
		 FROM public.physical_inventories WHERE enterprise_id=$1 ORDER BY created_at DESC`, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing physical inventories: %w", err)
	}
	defer rows.Close()
	return scanInventories(rows)
}

func (r *StockRepositorySQLC) ListInventoriesByStatus(ctx context.Context, status string) ([]*entity.PhysicalInventory, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, code, description, warehouse_id, start_date, end_date, status,
		        total_items, counted_items, notes, created_at, updated_at, created_by
		 FROM public.physical_inventories WHERE status = $1 AND enterprise_id=$2 ORDER BY created_at DESC`, status, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing physical inventories by status: %w", err)
	}
	defer rows.Close()
	return scanInventories(rows)
}

func (r *StockRepositorySQLC) UpdateInventoryStatus(ctx context.Context, id int64, status string) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE public.physical_inventories SET status = $1, updated_at = NOW() WHERE id = $2 AND enterprise_id=$3`, status, id, enterpriseID)
	if err != nil {
		return fmt.Errorf("updating inventory status: %w", err)
	}
	return nil
}

func (r *StockRepositorySQLC) CloseInventory(ctx context.Context, id int64) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE public.physical_inventories SET status = 'CLOSED', end_date = CURRENT_DATE, updated_at = NOW() WHERE id = $1 AND enterprise_id=$2`, id, enterpriseID)
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
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	var existingID int64
	err = r.pool.QueryRow(ctx,
		`SELECT item.id FROM public.physical_inventory_items item
		 JOIN physical_inventories inventory ON inventory.id=item.inventory_id
		 WHERE item.inventory_id=$1 AND item.item_code=$2 AND item.mask=$3 AND item.warehouse_id=$4 AND inventory.enterprise_id=$5`,
		item.InventoryID, item.ItemCode, item.Mask, item.WarehouseID, enterpriseID,
	).Scan(&existingID)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("checking existing inventory item: %w", err)
	}
	if err == pgx.ErrNoRows {
		_, err = r.pool.Exec(ctx,
			`INSERT INTO public.physical_inventory_items
				(inventory_id, item_code, mask, warehouse_id, system_qty)
			 SELECT $1,$2,$3,$4,$5 WHERE EXISTS (SELECT 1 FROM physical_inventories WHERE id=$1 AND enterprise_id=$6)`,
			item.InventoryID, item.ItemCode, item.Mask, item.WarehouseID, item.SystemQty, enterpriseID)
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
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, inventory_id, item_code, mask, warehouse_id, system_qty, counted_qty,
		        difference_qty, unit_cost, adjustment_type, adjustment_reason, counted_by,
		        counted_at, is_adjusted, created_at
		 FROM public.physical_inventory_items item WHERE inventory_id=$1 AND EXISTS
		 (SELECT 1 FROM physical_inventories inventory WHERE inventory.id=item.inventory_id AND inventory.enterprise_id=$2)
		 ORDER BY item_code`, inventoryID, enterpriseID)
	if err != nil {
		return nil, fmt.Errorf("listing inventory items: %w", err)
	}
	defer rows.Close()
	return scanInventoryItems(rows)
}

func (r *StockRepositorySQLC) CountInventoryItem(ctx context.Context, item *entity.PhysicalInventoryItem) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE public.physical_inventory_items SET
		     counted_qty = $1, unit_cost = $2, counted_by = $3, counted_at = NOW()
		 WHERE inventory_id=$4 AND item_code=$5 AND mask=$6 AND warehouse_id=$7 AND EXISTS
		 (SELECT 1 FROM physical_inventories inventory WHERE inventory.id=$4 AND inventory.enterprise_id=$8)`,
		item.CountedQty, item.UnitCost, item.CountedBy, item.InventoryID, item.ItemCode, item.Mask, item.WarehouseID, enterpriseID)
	if err != nil {
		return fmt.Errorf("counting inventory item: %w", err)
	}
	return nil
}

func (r *StockRepositorySQLC) AdjustInventoryItem(ctx context.Context, item *entity.PhysicalInventoryItem) error {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`UPDATE public.physical_inventory_items SET
		     adjustment_type = $1, adjustment_reason = $2, is_adjusted = true
		 WHERE inventory_id=$3 AND item_code=$4 AND mask=$5 AND warehouse_id=$6 AND EXISTS
		 (SELECT 1 FROM physical_inventories inventory WHERE inventory.id=$3 AND inventory.enterprise_id=$7)`,
		item.AdjustmentType, item.AdjustmentReason, item.InventoryID, item.ItemCode, item.Mask, item.WarehouseID, enterpriseID)
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
