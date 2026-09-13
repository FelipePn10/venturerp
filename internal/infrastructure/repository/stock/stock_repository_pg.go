package stock

import (
	"context"
	"fmt"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/google/uuid"
	"sort"
	"strings"
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
	if err := valorizarSaidaPelaMedia(ctx, tx, enterpriseID, m); err != nil {
		return err
	}
	err := tx.QueryRow(ctx,
		`INSERT INTO public.stock_movements
			(item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
			 reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, created_by, enterprise_id,
			 address, address_to)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
		 RETURNING id, created_at`,
		m.ItemCode, m.Mask, m.WarehouseID, m.MovementType, movementQuantity(m), m.UnitPrice, m.TotalPrice,
		m.ReferenceType, m.ReferenceCode, m.Lot, m.SerialNumber, m.Batch, m.ExpirationDate, m.Notes, m.CreatedBy, enterpriseID,
		m.Address, m.AddressTo,
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
			(item_code, mask, warehouse_id, lot, quantity, last_cost, last_movement_at, enterprise_id, address)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 ON CONFLICT (enterprise_id, item_code, mask, warehouse_id, lot, address) WHERE enterprise_id IS NOT NULL DO UPDATE SET
			 quantity = public.stock_lot_balances.quantity + EXCLUDED.quantity,
			 last_cost = CASE WHEN EXCLUDED.last_cost > 0 THEN EXCLUDED.last_cost ELSE public.stock_lot_balances.last_cost END,
			 last_movement_at = EXCLUDED.last_movement_at,
			 updated_at = NOW()`,
		m.ItemCode, m.Mask, m.WarehouseID, *m.Lot, delta, lastCost, m.CreatedAt, enterpriseID, enderecoDoSaldo(m))
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
		 WHERE item_code=$1 AND mask=$2 AND warehouse_id=$3 AND enterprise_id=$4 AND address=$5 FOR UPDATE`,
		m.ItemCode, m.Mask, m.WarehouseID, enterpriseID, enderecoDoSaldo(m),
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
			 WHERE item_code=$1 AND mask=$2 AND warehouse_id=$3 AND enterprise_id=$9 AND address=$10`,
			m.ItemCode, m.Mask, m.WarehouseID, newQty, newAvg, lastCost, newTotal, m.CreatedAt, enterpriseID, enderecoDoSaldo(m))
	} else {
		_, err = tx.Exec(ctx,
			`INSERT INTO public.stock_balances
				(item_code, mask, warehouse_id, quantity, avg_cost, last_cost, total_cost, last_movement_at, enterprise_id, address)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			 ON CONFLICT (enterprise_id, item_code, mask, warehouse_id, address) DO UPDATE
			 SET quantity = public.stock_balances.quantity + EXCLUDED.quantity,
			     total_cost = public.stock_balances.total_cost + EXCLUDED.total_cost,
			     avg_cost = CASE WHEN public.stock_balances.quantity + EXCLUDED.quantity > 0
			                     THEN (public.stock_balances.total_cost + EXCLUDED.total_cost)
			                          / (public.stock_balances.quantity + EXCLUDED.quantity)
			                     ELSE EXCLUDED.avg_cost END,
			     last_cost = EXCLUDED.last_cost,
			     last_movement_at = EXCLUDED.last_movement_at,
			     updated_at = NOW()`,
			m.ItemCode, m.Mask, m.WarehouseID, newQty, newAvg, lastCost, newTotal, m.CreatedAt, enterpriseID, enderecoDoSaldo(m))
	}
	if err != nil {
		return fmt.Errorf("updating stock balance: %w", err)
	}
	return nil
}

// valorizarSaidaPelaMedia preenche o valor de uma saída que veio sem preço.
//
// O saldo já baixa pela média (ApplyMovementCosting consome ao custo médio
// corrente), mas o MOVIMENTO era gravado com o unit_price que o chamador
// mandasse — e as telas de baixa não mandam preço. O extrato e os relatórios de
// valorização ficavam com consumo a zero, embora o estoque tivesse sido baixado
// pelo valor certo. Aqui a saída passa a registrar o mesmo custo que consumiu.
func valorizarSaidaPelaMedia(ctx context.Context, tx pgx.Tx, enterpriseID int64, m *entity.StockMovement) error {
	if m.UnitPrice > 0 || m.TotalPrice > 0 {
		return nil
	}
	if !signedDecimalQuantity(m.MovementType, movementQuantity(m)).IsNegative() {
		return nil
	}
	var avgCost float64
	err := tx.QueryRow(ctx,
		`SELECT avg_cost FROM public.stock_balances
		 WHERE item_code=$1 AND mask=$2 AND warehouse_id=$3 AND enterprise_id=$4`,
		m.ItemCode, m.Mask, m.WarehouseID, enterpriseID).Scan(&avgCost)
	if err == pgx.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading average cost to value the outbound movement: %w", err)
	}
	if avgCost <= 0 {
		return nil
	}
	quantidade, _ := movementQuantity(m).Abs().Float64()
	m.UnitPrice = avgCost
	m.TotalPrice = quantidade * avgCost
	return nil
}

// enderecoDoSaldo devolve o endereço que identifica a linha de saldo. Vazio
// significa almoxarifado sem endereçamento — o saldo continua sendo um só, como
// antes da migração 344.
func enderecoDoSaldo(m *entity.StockMovement) string {
	if m.Address == nil {
		return ""
	}
	return strings.TrimSpace(*m.Address)
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
		        reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, address, address_to, created_at, created_by
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
		        reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, address, address_to, created_at, created_by
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
		        reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, address, address_to, created_at, created_by
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
		        reference_type, reference_code, lot, serial_number, batch, expiration_date, notes, address, address_to, created_at, created_by
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
			&m.ReferenceType, &m.ReferenceCode, &m.Lot, &m.SerialNumber, &m.Batch, &m.ExpirationDate, &m.Notes,
			&m.Address, &m.AddressTo, &m.CreatedAt, &m.CreatedBy,
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
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("saldo de estoque não encontrado para o item %d, máscara %s e almoxarifado %d", itemCode, mask, warehouseID))
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

// adjustReservedTx adds delta to reserved_qty of an existing balance. A
// reservation cannot manufacture availability by creating a zero-quantity row.
func adjustReservedTx(ctx context.Context, tx pgx.Tx, enterpriseID, itemCode int64, mask string, warehouseID int64, delta float64) error {
	command, err := tx.Exec(ctx,
		`UPDATE public.stock_balances
		 SET reserved_qty = GREATEST(reserved_qty + $5, 0), updated_at = NOW()
		 WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4
		   AND ($5::numeric <= 0 OR quantity - reserved_qty >= $5::numeric)`,
		enterpriseID, itemCode, mask, warehouseID, delta)
	if err != nil {
		return fmt.Errorf("adjusting reserved quantity: %w", err)
	}
	if command.RowsAffected() == 0 {
		return repository.ErrInsufficientStock
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
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("reserva de estoque %d não encontrada", id))
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
			return errorsuc.NewNotFoundError(fmt.Sprintf("reserva de estoque %d não encontrada", id))
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
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("consumo médio não encontrado para o item %d", itemCode))
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
			(item_code, mask, lot, heat_number, certificate, supplier_code, received_at, expires_at, notes, created_by, enterprise_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 ON CONFLICT (enterprise_id,item_code,mask,lot) DO UPDATE SET
			 heat_number   = EXCLUDED.heat_number,
			 certificate   = EXCLUDED.certificate,
			 supplier_code = EXCLUDED.supplier_code,
			 received_at   = EXCLUDED.received_at,
			 expires_at    = EXCLUDED.expires_at,
			 notes         = EXCLUDED.notes
		 RETURNING id, created_at`,
		lot.ItemCode, lot.Mask, lot.Lot, lot.HeatNumber, lot.Certificate, lot.SupplierCode, lot.ReceivedAt, lot.ExpiresAt, lot.Notes, lot.CreatedBy, enterpriseID,
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
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("inventário físico %d não encontrado", id))
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
			return nil, errorsuc.NewNotFoundError(fmt.Sprintf("inventário físico %d não encontrado", code))
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

// SugerirSeparacaoFEFO devolve de quais lotes e endereços tirar a quantidade
// pedida, na ordem da regra escolhida.
//
// FEFO ("first expired, first out") ordena por validade: o que vence antes sai
// antes. É o que a norma sanitária e a maioria das indústrias exige, e o que o
// sistema não tinha — havia apenas FIFO, e só dentro do plano de corte. Lote sem
// validade cai para o fim e desempata pela data de recebimento, que é o FIFO.
//
// Lote vencido nunca é sugerido: ele é contado à parte para que a tela possa
// avisar em vez de fingir que o saldo está disponível.
func (r *StockRepositorySQLC) SugerirSeparacaoFEFO(
	ctx context.Context, itemCode int64, mask string, warehouseID int64, necessario float64, regra string,
) (*entity.ResultadoFEFO, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	ordem := `l.expires_at IS NULL, l.expires_at, l.received_at NULLS LAST, b.lot`
	if strings.EqualFold(regra, "FIFO") {
		regra = "FIFO"
		ordem = `l.received_at NULLS LAST, l.expires_at IS NULL, l.expires_at, b.lot`
	} else {
		regra = "FEFO"
	}

	linhas, err := r.pool.Query(ctx, `
		SELECT b.lot, b.address, b.warehouse_id, GREATEST(b.quantity - b.reserved_qty, 0) AS disponivel,
		       l.heat_number, l.certificate, l.expires_at, l.received_at,
		       COALESCE(l.expires_at < CURRENT_DATE, false) AS vencido,
		       COALESCE(a.zone, ''), COALESCE(a.pick_sequence, 0),
		       COALESCE(a.is_blocked, false) AS bloqueado
		  FROM public.stock_lot_balances b
		  LEFT JOIN public.stock_lots l
		         ON l.enterprise_id = b.enterprise_id AND l.item_code = b.item_code
		        AND l.mask = b.mask AND l.lot = b.lot
		  LEFT JOIN public.manufacturing_warehouse_addresses a
		         ON a.enterprise_id = b.enterprise_id AND a.warehouse_id = b.warehouse_id
		        AND a.address = b.address
		 WHERE b.enterprise_id = $1 AND b.item_code = $2 AND b.mask = $3
		   AND ($4 = 0 OR b.warehouse_id = $4)
		   -- Desconta o reservado: sem isso duas ondas geradas no mesmo minuto
		   -- prometem o mesmo lote no mesmo endereço, e a segunda chega lá e não
		   -- acha nada.
		   AND b.quantity - b.reserved_qty > 0
		 ORDER BY `+ordem,
		enterpriseID, itemCode, mask, warehouseID)
	if err != nil {
		return nil, fmt.Errorf("consultando lotes para separação: %w", err)
	}
	defer linhas.Close()

	resultado := &entity.ResultadoFEFO{
		ItemCode: itemCode, Mask: mask, Regra: regra, Necessario: necessario,
		Linhas: []entity.SugestaoFEFO{},
	}
	restante := necessario
	for linhas.Next() {
		var s entity.SugestaoFEFO
		var vencido, bloqueado bool
		if err := linhas.Scan(&s.Lot, &s.Address, &s.WarehouseID, &s.Disponivel,
			&s.HeatNumber, &s.Certificate, &s.ExpiresAt, &s.ReceivedAt, &vencido,
			&s.Zone, &s.PickSequence, &bloqueado); err != nil {
			return nil, fmt.Errorf("lendo lote para separação: %w", err)
		}
		if vencido {
			resultado.VencidosFora++
			continue
		}
		// Endereço bloqueado (inventário, avaria, quarentena) não entra na
		// separação: o saldo existe, mas não pode sair.
		if bloqueado {
			resultado.BloqueadosFora++
			continue
		}
		if restante <= 0 {
			continue
		}
		s.Sugerido = s.Disponivel
		if s.Sugerido > restante {
			s.Sugerido = restante
		}
		restante -= s.Sugerido
		resultado.Linhas = append(resultado.Linhas, s)
	}
	if err := linhas.Err(); err != nil {
		return nil, fmt.Errorf("lendo lotes para separação: %w", err)
	}
	resultado.Atendido = necessario - restante
	if restante > 0 {
		resultado.EmFalta = restante
	}
	// O FEFO decide QUAIS lotes; a lista é entregue na ordem de caminhada, que é
	// o que o separador segue no galpão. Sem sequência cadastrada (0) a ordem
	// vira a do endereço, que é melhor que a ordem de validade para caminhar.
	sort.SliceStable(resultado.Linhas, func(i, j int) bool {
		a, b := resultado.Linhas[i], resultado.Linhas[j]
		if a.PickSequence != b.PickSequence {
			if a.PickSequence == 0 || b.PickSequence == 0 {
				return b.PickSequence == 0
			}
			return a.PickSequence < b.PickSequence
		}
		return a.Address < b.Address
	})
	return resultado, nil
}

// ListarSaldoPorEndereco devolve o saldo quebrado por endereço de um
// almoxarifado. É a consulta que o Focco expõe na FEST0332; aqui ela também
// traz o lote, para que a conferência física não precise de uma segunda tela.
func (r *StockRepositorySQLC) ListarSaldoPorEndereco(
	ctx context.Context, warehouseID int64, itemCode int64,
) ([]*entity.StockLotBalance, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	linhas, err := r.pool.Query(ctx, `
		SELECT id, item_code, mask, warehouse_id, lot, address, quantity, last_cost, last_movement_at, updated_at
		  FROM public.stock_lot_balances
		 WHERE enterprise_id = $1
		   AND ($2 = 0 OR warehouse_id = $2)
		   AND ($3 = 0 OR item_code = $3)
		   AND quantity <> 0
		 ORDER BY warehouse_id, address, item_code, lot`,
		enterpriseID, warehouseID, itemCode)
	if err != nil {
		return nil, fmt.Errorf("consultando saldo por endereço: %w", err)
	}
	defer linhas.Close()
	var saldos []*entity.StockLotBalance
	for linhas.Next() {
		var b entity.StockLotBalance
		if err := linhas.Scan(&b.ID, &b.ItemCode, &b.Mask, &b.WarehouseID, &b.Lot, &b.Address,
			&b.Quantity, &b.LastCost, &b.LastMovementAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("lendo saldo por endereço: %w", err)
		}
		saldos = append(saldos, &b)
	}
	return saldos, linhas.Err()
}

// TransferirEntreEnderecos move quantidade de um endereço para outro dentro do
// MESMO almoxarifado, numa única transação.
//
// É um movimento só, com origem e destino, e não dois espelhados: dois
// movimentos dobrariam a quantidade nos relatórios de giro e fariam a mesma peça
// aparecer duas vezes no extrato. O saldo do almoxarifado não muda — muda de
// endereço —, por isso a baixa e o crédito são aplicados aqui explicitamente em
// vez de passarem pela apuração normal de saldo.
func (r *StockRepositorySQLC) TransferirEntreEnderecos(
	ctx context.Context, m *entity.StockMovement, origem, destino string,
) (*entity.StockMovement, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	quantidade := movementQuantity(m)
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrindo transação da transferência: %w", err)
	}
	defer tx.Rollback(ctx)

	// A origem precisa cobrir a quantidade; sem isso o endereço ficaria negativo
	// e o inventário nunca fecharia.
	var disponivel decimal.Decimal
	err = tx.QueryRow(ctx,
		`SELECT quantity FROM public.stock_balances
		  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND address=$5 FOR UPDATE`,
		enterpriseID, m.ItemCode, m.Mask, m.WarehouseID, origem).Scan(&disponivel)
	if err == pgx.ErrNoRows {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("não há saldo do item no endereço %q", origem))
	}
	if err != nil {
		return nil, fmt.Errorf("lendo saldo da origem: %w", err)
	}
	if disponivel.LessThan(quantidade) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf(
			"o endereço %q tem %s e a transferência pede %s", origem, disponivel.String(), quantidade.String()))
	}

	var custoMedio, custoTotal float64
	_ = tx.QueryRow(ctx,
		`SELECT avg_cost, total_cost FROM public.stock_balances
		  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND address=$5`,
		enterpriseID, m.ItemCode, m.Mask, m.WarehouseID, origem).Scan(&custoMedio, &custoTotal)
	valorMovido, _ := quantidade.Float64()
	valorMovido *= custoMedio

	err = tx.QueryRow(ctx,
		`INSERT INTO public.stock_movements
			(item_code, mask, warehouse_id, movement_type, quantity, unit_price, total_price,
			 reference_type, reference_code, lot, notes, created_by, enterprise_id, address, address_to)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		 RETURNING id, created_at`,
		m.ItemCode, m.Mask, m.WarehouseID, entity.MovementTypeAddressTransfer, quantidade,
		custoMedio, valorMovido, m.ReferenceType, m.ReferenceCode, m.Lot, m.Notes, m.CreatedBy,
		enterpriseID, origem, destino,
	).Scan(&m.ID, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("gravando a transferência: %w", err)
	}

	// Baixa na origem, mantendo o custo médio (o material é o mesmo).
	if _, err = tx.Exec(ctx,
		`UPDATE public.stock_balances
		    SET quantity = quantity - $6, total_cost = GREATEST(total_cost - $7, 0), last_movement_at = $8, updated_at = NOW()
		  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND address=$5`,
		enterpriseID, m.ItemCode, m.Mask, m.WarehouseID, origem, quantidade, valorMovido, m.CreatedAt); err != nil {
		return nil, fmt.Errorf("baixando o endereço de origem: %w", err)
	}

	if _, err = tx.Exec(ctx,
		`INSERT INTO public.stock_balances
			(item_code, mask, warehouse_id, quantity, avg_cost, last_cost, total_cost, last_movement_at, enterprise_id, address)
		 VALUES ($1,$2,$3,$4,$5,$5,$6,$7,$8,$9)
		 ON CONFLICT (enterprise_id, item_code, mask, warehouse_id, address) DO UPDATE
		 SET quantity = public.stock_balances.quantity + EXCLUDED.quantity,
		     total_cost = public.stock_balances.total_cost + EXCLUDED.total_cost,
		     avg_cost = CASE WHEN public.stock_balances.quantity + EXCLUDED.quantity > 0
		                     THEN (public.stock_balances.total_cost + EXCLUDED.total_cost)
		                          / (public.stock_balances.quantity + EXCLUDED.quantity)
		                     ELSE EXCLUDED.avg_cost END,
		     last_movement_at = EXCLUDED.last_movement_at, updated_at = NOW()`,
		m.ItemCode, m.Mask, m.WarehouseID, quantidade, custoMedio, valorMovido, m.CreatedAt, enterpriseID, destino); err != nil {
		return nil, fmt.Errorf("creditando o endereço de destino: %w", err)
	}

	// Lote acompanha o material: sem isso a rastreabilidade apontaria o endereço
	// antigo depois da transferência.
	if m.Lot != nil && *m.Lot != "" {
		if _, err = tx.Exec(ctx,
			`UPDATE public.stock_lot_balances SET quantity = quantity - $7, updated_at = NOW()
			  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND address=$5 AND lot=$6`,
			enterpriseID, m.ItemCode, m.Mask, m.WarehouseID, origem, *m.Lot, quantidade); err != nil {
			return nil, fmt.Errorf("baixando o lote na origem: %w", err)
		}
		if _, err = tx.Exec(ctx,
			`INSERT INTO public.stock_lot_balances
				(item_code, mask, warehouse_id, lot, quantity, last_cost, last_movement_at, enterprise_id, address)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			 ON CONFLICT (enterprise_id, item_code, mask, warehouse_id, lot, address) WHERE enterprise_id IS NOT NULL DO UPDATE
			 SET quantity = public.stock_lot_balances.quantity + EXCLUDED.quantity,
			     last_movement_at = EXCLUDED.last_movement_at, updated_at = NOW()`,
			m.ItemCode, m.Mask, m.WarehouseID, *m.Lot, quantidade, custoMedio, m.CreatedAt, enterpriseID, destino); err != nil {
			return nil, fmt.Errorf("creditando o lote no destino: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("confirmando a transferência: %w", err)
	}
	return m, nil
}

// ApurarCurvaABC classifica os itens pelo VALOR consumido na janela, não pela
// quantidade: mil parafusos baratos não são um item A.
//
// Pareto clássico — A até 80% do valor acumulado, B até 95%, C o resto. Grava em
// items.planning_abc_class (a coluna que o resto do sistema já lê) junto do valor
// e da participação, para a tela poder explicar a classe em vez de exibir só a
// letra. Antes a classe era digitada à mão e envelhecia sem ninguém notar.
func (r *StockRepositorySQLC) ApurarCurvaABC(ctx context.Context, janelaMeses int, corteA, corteB float64) (*entity.ResumoABC, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if janelaMeses <= 0 {
		janelaMeses = 12
	}
	if corteA <= 0 || corteA >= 100 {
		corteA = 80
	}
	if corteB <= corteA || corteB >= 100 {
		corteB = 95
	}

	linhas, err := r.pool.Query(ctx, `
		WITH consumo AS (
			SELECT m.item_code,
			       SUM(ABS(m.quantity) * COALESCE(NULLIF(m.unit_price,0), b.avg_cost, 0)) AS valor
			  FROM stock_movements m
			  LEFT JOIN LATERAL (
			       SELECT AVG(avg_cost) avg_cost FROM stock_balances s
			        WHERE s.enterprise_id = m.enterprise_id AND s.item_code = m.item_code
			  ) b ON TRUE
			 WHERE m.enterprise_id = $1
			   AND m.created_at >= NOW() - make_interval(months => $2)
			   AND m.movement_type IN ('OUT','SAIDA','TRANSFER_OUT','REP')
			 GROUP BY m.item_code
			HAVING SUM(ABS(m.quantity) * COALESCE(NULLIF(m.unit_price,0), b.avg_cost, 0)) > 0
		), total AS (
			SELECT COALESCE(SUM(valor),0) t FROM consumo
		)
		SELECT c.item_code, c.valor,
		       100 * c.valor / NULLIF(t.t,0) AS participacao,
		       100 * SUM(c.valor) OVER (ORDER BY c.valor DESC, c.item_code) / NULLIF(t.t,0) AS acumulado,
		       t.t
		  FROM consumo c CROSS JOIN total t
		 ORDER BY c.valor DESC, c.item_code`, enterpriseID, janelaMeses)
	if err != nil {
		return nil, fmt.Errorf("apurando consumo para a curva ABC: %w", err)
	}
	defer linhas.Close()

	resumo := &entity.ResumoABC{JanelaMeses: janelaMeses, CorteA: corteA, CorteB: corteB, Itens: []entity.ClasseABC{}}
	for linhas.Next() {
		var it entity.ClasseABC
		if err := linhas.Scan(&it.ItemCode, &it.ValorConsumido, &it.ParticipacaoPct, &it.AcumuladoPct, &resumo.ValorTotal); err != nil {
			return nil, fmt.Errorf("lendo consumo do item: %w", err)
		}
		it.Classe = entity.ClassificarABC(it.AcumuladoPct, it.ParticipacaoPct, corteA, corteB)
		resumo.Itens = append(resumo.Itens, it)
	}
	if err := linhas.Err(); err != nil {
		return nil, fmt.Errorf("lendo a curva ABC: %w", err)
	}

	for _, it := range resumo.Itens {
		if _, err := r.pool.Exec(ctx, `
			UPDATE items SET planning_abc_class = $3, abc_consumption_value = $4,
			                 abc_share_pct = $5, abc_calculated_at = NOW()
			 WHERE enterprise_id = $1 AND code = $2`,
			enterpriseID, it.ItemCode, it.Classe, it.ValorConsumido, it.AcumuladoPct); err != nil {
			return nil, fmt.Errorf("gravando a classe ABC do item %d: %w", it.ItemCode, err)
		}
		resumo.Classificados++
	}
	return resumo, nil
}

// SugerirEnderecoDeGuarda recomenda onde guardar o material que chegou.
//
// A ordem das estratégias é a que reduz deslocamento e fragmentação, e é a mesma
// que SAP e Focco usam como padrão:
//  1. endereço fixo do item (o "fixed bin") — se existe, é ele e acabou;
//  2. consolidação: endereço que JÁ tem o mesmo item, para não espalhar o saldo;
//  3. endereço vazio da zona pedida, na ordem da rota de separação;
//  4. qualquer endereço ativo com espaço.
//
// Endereço bloqueado nunca é sugerido, e a capacidade declarada é respeitada —
// sugerir um endereço que não cabe é pior que não sugerir nada.
func (r *StockRepositorySQLC) SugerirEnderecoDeGuarda(
	ctx context.Context, itemCode int64, mask string, warehouseID int64, quantidade float64, zona string,
) ([]*entity.SugestaoGuarda, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	linhas, err := r.pool.Query(ctx, `
		WITH enderecos AS (
			SELECT a.address, a.zone, a.capacity, a.pick_sequence, a.fixed_item_code,
			       COALESCE(s.quantity, 0) AS saldo,
			       COALESCE(ocupado.total, 0) AS ocupado
			  FROM manufacturing_warehouse_addresses a
			  LEFT JOIN stock_balances s
			         ON s.enterprise_id = a.enterprise_id AND s.warehouse_id = a.warehouse_id
			        AND s.address = a.address AND s.item_code = $2 AND s.mask = $3
			  LEFT JOIN LATERAL (
			       SELECT SUM(quantity) total FROM stock_balances x
			        WHERE x.enterprise_id = a.enterprise_id AND x.warehouse_id = a.warehouse_id
			          AND x.address = a.address
			  ) ocupado ON TRUE
			 WHERE a.enterprise_id = $1 AND a.warehouse_id = $4
			   AND a.is_active AND NOT a.is_blocked
		)
		SELECT address, zone, capacity, pick_sequence, saldo,
		       CASE
		         WHEN fixed_item_code = $2                     THEN 'endereço fixo do item'
		         WHEN saldo > 0                                THEN 'consolida com o saldo que já está aqui'
		         WHEN ocupado = 0 AND ($5 = '' OR zone = $5)    THEN 'endereço vazio na zona pedida'
		         ELSE 'endereço ativo com espaço'
		       END AS motivo,
		       (capacity IS NULL OR capacity - ocupado >= $6) AS cabe
		  FROM enderecos
		 WHERE (capacity IS NULL OR capacity - ocupado >= $6)
		 ORDER BY (fixed_item_code = $2) DESC NULLS LAST,
		          (saldo > 0) DESC,
		          ($5 <> '' AND zone = $5) DESC,
		          pick_sequence,
		          address
		 LIMIT 10`, enterpriseID, itemCode, mask, warehouseID, strings.TrimSpace(zona), quantidade)
	if err != nil {
		return nil, fmt.Errorf("sugerindo endereço de guarda: %w", err)
	}
	defer linhas.Close()
	var sugestoes []*entity.SugestaoGuarda
	for linhas.Next() {
		var s entity.SugestaoGuarda
		if err := linhas.Scan(&s.Address, &s.Zone, &s.Capacidade, &s.PickSeq, &s.SaldoAtual, &s.Motivo, &s.Cabe); err != nil {
			return nil, fmt.Errorf("lendo sugestão de guarda: %w", err)
		}
		sugestoes = append(sugestoes, &s)
	}
	return sugestoes, linhas.Err()
}

// CriarOndaDeSeparacao aloca as necessidades por FEFO, RESERVA o que alocou no
// endereço/lote e devolve a lista já na ordem da rota.
//
// Reservar é o ponto: sem isso duas ondas criadas no mesmo minuto mandam o
// separador ao mesmo endereço atrás da mesma peça. A reserva é feita dentro da
// mesma transação da alocação, com o saldo travado — entre ler o disponível e
// reservar não pode caber outra onda.
//
// A onda é criada mesmo quando falta saldo: separar o que existe e sinalizar a
// falta é mais útil ao almoxarifado que recusar a onda inteira.
func (r *StockRepositorySQLC) CriarOndaDeSeparacao(
	ctx context.Context, codigo, warehouseID int64, regra string, necessidades []entity.NecessidadeOnda, ator uuid.UUID,
) (*entity.OndaDeSeparacao, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(regra, "FIFO") {
		regra = "FIFO"
	} else {
		regra = "FEFO"
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrindo transação da onda: %w", err)
	}
	defer tx.Rollback(ctx)

	onda := &entity.OndaDeSeparacao{
		Code: codigo, WarehouseID: warehouseID, Status: "ABERTA", Rule: regra,
		Linhas: []entity.LinhaOnda{}, EmFalta: []entity.FaltaNaOnda{},
	}
	if err = tx.QueryRow(ctx,
		`INSERT INTO stock_picking_waves(enterprise_id,code,warehouse_id,status,rule,created_by)
		 VALUES($1,$2,$3,'ABERTA',$4,$5) RETURNING id, created_at`,
		enterpriseID, codigo, warehouseID, regra, ator).Scan(&onda.ID, &onda.CreatedAt); err != nil {
		return nil, fmt.Errorf("criando a onda: %w", err)
	}

	ordem := `l.expires_at IS NULL, l.expires_at, l.received_at NULLS LAST, b.lot`
	if regra == "FIFO" {
		ordem = `l.received_at NULLS LAST, l.expires_at IS NULL, l.expires_at, b.lot`
	}

	for _, necessidade := range necessidades {
		if necessidade.Quantity <= 0 {
			continue
		}
		restante := necessidade.Quantity
		candidatos, err := tx.Query(ctx, `
			SELECT b.id, b.lot, b.address, GREATEST(b.quantity - b.reserved_qty, 0) AS disponivel,
			       l.heat_number, l.expires_at,
			       COALESCE(a.zone,''), COALESCE(a.pick_sequence,0)
			  FROM stock_lot_balances b
			  LEFT JOIN stock_lots l ON l.enterprise_id=b.enterprise_id AND l.item_code=b.item_code
			                        AND l.mask=b.mask AND l.lot=b.lot
			  LEFT JOIN manufacturing_warehouse_addresses a
			         ON a.enterprise_id=b.enterprise_id AND a.warehouse_id=b.warehouse_id AND a.address=b.address
			 WHERE b.enterprise_id=$1 AND b.item_code=$2 AND b.mask=$3 AND b.warehouse_id=$4
			   AND b.quantity - b.reserved_qty > 0
			   AND COALESCE(l.expires_at >= CURRENT_DATE, true)
			   AND COALESCE(a.is_blocked,false) = false
			 ORDER BY `+ordem+`
			 FOR UPDATE OF b`,
			enterpriseID, necessidade.ItemCode, necessidade.Mask, warehouseID)
		if err != nil {
			return nil, fmt.Errorf("alocando o item %d na onda: %w", necessidade.ItemCode, err)
		}
		type alocacao struct {
			saldoID              int64
			lote, endereco, zona string
			quantidade           float64
			corrida              *string
			validade             *time.Time
			sequencia            int
		}
		var alocacoes []alocacao
		for candidatos.Next() {
			var c alocacao
			var disponivel float64
			if err := candidatos.Scan(&c.saldoID, &c.lote, &c.endereco, &disponivel,
				&c.corrida, &c.validade, &c.zona, &c.sequencia); err != nil {
				candidatos.Close()
				return nil, fmt.Errorf("lendo candidato da onda: %w", err)
			}
			if restante <= 0 {
				continue
			}
			c.quantidade = disponivel
			if c.quantidade > restante {
				c.quantidade = restante
			}
			restante -= c.quantidade
			alocacoes = append(alocacoes, c)
		}
		errCandidatos := candidatos.Err()
		candidatos.Close()
		if errCandidatos != nil {
			return nil, fmt.Errorf("lendo candidatos da onda: %w", errCandidatos)
		}

		for _, c := range alocacoes {
			var reservaID int64
			if err = tx.QueryRow(ctx,
				`INSERT INTO stock_reservations(item_code,mask,warehouse_id,quantity,reference_type,reference_code,
				     status,created_by,enterprise_id,address,lot)
				 VALUES($1,$2,$3,$4,'ONDA_SEPARACAO',$5,'ACTIVE',$6,$7,$8,$9) RETURNING id`,
				necessidade.ItemCode, necessidade.Mask, warehouseID, c.quantidade, codigo, ator,
				enterpriseID, c.endereco, c.lote).Scan(&reservaID); err != nil {
				return nil, fmt.Errorf("reservando o lote %s: %w", c.lote, err)
			}
			if _, err = tx.Exec(ctx,
				`UPDATE stock_lot_balances SET reserved_qty = reserved_qty + $2, updated_at=NOW() WHERE id=$1`,
				c.saldoID, c.quantidade); err != nil {
				return nil, fmt.Errorf("marcando a reserva no saldo do lote: %w", err)
			}
			if _, err = tx.Exec(ctx,
				`UPDATE stock_balances SET reserved_qty = reserved_qty + $6, updated_at=NOW()
				  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND address=$5`,
				enterpriseID, necessidade.ItemCode, necessidade.Mask, warehouseID, c.endereco, c.quantidade); err != nil {
				return nil, fmt.Errorf("marcando a reserva no saldo do endereço: %w", err)
			}
			var linhaID int64
			if err = tx.QueryRow(ctx,
				`INSERT INTO stock_picking_wave_lines(enterprise_id,wave_id,item_code,mask,lot,address,
				     pick_sequence,quantity,reservation_id,reference_type,reference_code)
				 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
				enterpriseID, onda.ID, necessidade.ItemCode, necessidade.Mask, c.lote, c.endereco,
				c.sequencia, c.quantidade, reservaID, necessidade.ReferenceType, necessidade.ReferenceCode).Scan(&linhaID); err != nil {
				return nil, fmt.Errorf("gravando a linha da onda: %w", err)
			}
			onda.Linhas = append(onda.Linhas, entity.LinhaOnda{
				ID: linhaID, ItemCode: necessidade.ItemCode, Mask: necessidade.Mask, Lot: c.lote,
				Address: c.endereco, Zone: c.zona, PickSequence: c.sequencia, Quantity: c.quantidade,
				HeatNumber: c.corrida, ExpiresAt: c.validade,
				ReferenceType: necessidade.ReferenceType, ReferenceCode: necessidade.ReferenceCode,
			})
		}
		if restante > 0 {
			onda.EmFalta = append(onda.EmFalta, entity.FaltaNaOnda{
				ItemCode: necessidade.ItemCode, Quantity: restante,
			})
		}
	}

	// Uma caminhada só: a ordem é a do galpão, não a das necessidades.
	sort.SliceStable(onda.Linhas, func(i, j int) bool {
		a, b := onda.Linhas[i], onda.Linhas[j]
		if a.PickSequence != b.PickSequence {
			if a.PickSequence == 0 || b.PickSequence == 0 {
				return b.PickSequence == 0
			}
			return a.PickSequence < b.PickSequence
		}
		return a.Address < b.Address
	})

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("confirmando a criação da onda: %w", err)
	}
	return onda, nil
}

// ConfirmarOndaDeSeparacao baixa o estoque do que foi separado e libera as
// reservas. Sem este passo a reserva ficaria presa para sempre e o saldo, embora
// fisicamente entregue, continuaria aparecendo como disponível.
func (r *StockRepositorySQLC) ConfirmarOndaDeSeparacao(ctx context.Context, codigo int64, ator uuid.UUID) (*entity.OndaDeSeparacao, error) {
	return r.encerrarOnda(ctx, codigo, ator, true)
}

// CancelarOndaDeSeparacao devolve o reservado sem mexer no estoque físico.
func (r *StockRepositorySQLC) CancelarOndaDeSeparacao(ctx context.Context, codigo int64, ator uuid.UUID) (*entity.OndaDeSeparacao, error) {
	return r.encerrarOnda(ctx, codigo, ator, false)
}

func (r *StockRepositorySQLC) encerrarOnda(ctx context.Context, codigo int64, ator uuid.UUID, confirmar bool) (*entity.OndaDeSeparacao, error) {
	enterpriseID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("abrindo transação da onda: %w", err)
	}
	defer tx.Rollback(ctx)

	var ondaID, warehouseID int64
	var status, regra string
	if err = tx.QueryRow(ctx,
		`SELECT id, warehouse_id, status, rule FROM stock_picking_waves
		  WHERE enterprise_id=$1 AND code=$2 FOR UPDATE`, enterpriseID, codigo).
		Scan(&ondaID, &warehouseID, &status, &regra); err == pgx.ErrNoRows {
		return nil, errorsuc.NewNotFoundError(fmt.Sprintf("onda de separação %d não encontrada", codigo))
	} else if err != nil {
		return nil, fmt.Errorf("lendo a onda: %w", err)
	}
	if status != "ABERTA" {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("a onda %d já está %s", codigo, strings.ToLower(status)))
	}

	linhas, err := tx.Query(ctx,
		`SELECT id, item_code, mask, lot, address, quantity, reservation_id
		   FROM stock_picking_wave_lines WHERE wave_id=$1 ORDER BY pick_sequence, address`, ondaID)
	if err != nil {
		return nil, fmt.Errorf("lendo as linhas da onda: %w", err)
	}
	type linha struct {
		id, itemCode         int64
		mask, lote, endereco string
		quantidade           float64
		reservaID            *int64
	}
	var todas []linha
	for linhas.Next() {
		var l linha
		if err := linhas.Scan(&l.id, &l.itemCode, &l.mask, &l.lote, &l.endereco, &l.quantidade, &l.reservaID); err != nil {
			linhas.Close()
			return nil, fmt.Errorf("lendo linha da onda: %w", err)
		}
		todas = append(todas, l)
	}
	errLinhas := linhas.Err()
	linhas.Close()
	if errLinhas != nil {
		return nil, fmt.Errorf("lendo linhas da onda: %w", errLinhas)
	}

	for _, l := range todas {
		// A reserva sai nos dois casos: confirmada vira baixa física, cancelada
		// simplesmente volta a ficar disponível.
		if _, err = tx.Exec(ctx,
			`UPDATE stock_lot_balances SET reserved_qty = GREATEST(reserved_qty - $5, 0), updated_at=NOW()
			  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND lot=$4 AND address=$6 AND warehouse_id=$7`,
			enterpriseID, l.itemCode, l.mask, l.lote, l.quantidade, l.endereco, warehouseID); err != nil {
			return nil, fmt.Errorf("liberando a reserva do lote: %w", err)
		}
		if _, err = tx.Exec(ctx,
			`UPDATE stock_balances SET reserved_qty = GREATEST(reserved_qty - $6, 0), updated_at=NOW()
			  WHERE enterprise_id=$1 AND item_code=$2 AND mask=$3 AND warehouse_id=$4 AND address=$5`,
			enterpriseID, l.itemCode, l.mask, warehouseID, l.endereco, l.quantidade); err != nil {
			return nil, fmt.Errorf("liberando a reserva do endereço: %w", err)
		}
		if l.reservaID != nil {
			estado := "CANCELLED"
			if confirmar {
				estado = "CONSUMED"
			}
			if _, err = tx.Exec(ctx, `UPDATE stock_reservations SET status=$2, updated_at=NOW() WHERE id=$1`,
				*l.reservaID, estado); err != nil {
				return nil, fmt.Errorf("encerrando a reserva: %w", err)
			}
		}
		if !confirmar {
			continue
		}
		lote := l.lote
		endereco := l.endereco
		movimento := &entity.StockMovement{
			ItemCode: l.itemCode, Mask: l.mask, WarehouseID: warehouseID,
			MovementType: entity.MovementTypeOut, Quantity: l.quantidade,
			Lot: &lote, Address: &endereco, CreatedBy: ator,
		}
		referencia := "ONDA_SEPARACAO"
		movimento.ReferenceType = &referencia
		movimento.ReferenceCode = &codigo
		if err = CreateMovementTx(ctx, tx, enterpriseID, movimento); err != nil {
			return nil, fmt.Errorf("baixando o estoque da linha da onda: %w", err)
		}
		if _, err = tx.Exec(ctx, `UPDATE stock_picking_wave_lines SET picked_qty=$2 WHERE id=$1`, l.id, l.quantidade); err != nil {
			return nil, fmt.Errorf("marcando a separação da linha: %w", err)
		}
	}

	novoStatus := "CANCELADA"
	coluna := "cancelled_at"
	if confirmar {
		novoStatus = "SEPARADA"
		coluna = "confirmed_at"
	}
	if _, err = tx.Exec(ctx,
		`UPDATE stock_picking_waves SET status=$2, `+coluna+`=NOW() WHERE id=$1`, ondaID, novoStatus); err != nil {
		return nil, fmt.Errorf("encerrando a onda: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("confirmando o encerramento da onda: %w", err)
	}
	return &entity.OndaDeSeparacao{
		ID: ondaID, Code: codigo, WarehouseID: warehouseID, Status: novoStatus, Rule: regra,
		Linhas: []entity.LinhaOnda{}, EmFalta: []entity.FaltaNaOnda{},
	}, nil
}
