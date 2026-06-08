package shipment

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ShipmentRepositoryPG struct {
	pool *pgxpool.Pool
}

func NewShipmentRepositoryPG(pool *pgxpool.Pool) *ShipmentRepositoryPG {
	return &ShipmentRepositoryPG{pool: pool}
}

var _ repository.ShipmentRepository = (*ShipmentRepositoryPG)(nil)

func (r *ShipmentRepositoryPG) NextCode(ctx context.Context) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO public.shipment_sequences (id, last_number) VALUES (1, 1)
		 ON CONFLICT (id) DO UPDATE SET last_number = shipment_sequences.last_number + 1
		 RETURNING last_number`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("next shipment code: %w", err)
	}
	return n, nil
}

func (r *ShipmentRepositoryPG) Create(ctx context.Context, s *entity.Shipment) (*entity.Shipment, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO public.shipments
			(code, sales_order_code, carrier_code, status, total_volumes, total_weight, notes, created_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING id, created_at, updated_at`,
		s.Code, s.SalesOrderCode, s.CarrierCode, string(s.Status), s.TotalVolumes, s.TotalWeight, s.Notes, s.CreatedBy,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating shipment: %w", err)
	}
	return s, nil
}

func (r *ShipmentRepositoryPG) GetByCode(ctx context.Context, code int64) (*entity.Shipment, error) {
	var s entity.Shipment
	var status string
	err := r.pool.QueryRow(ctx,
		`SELECT id, code, sales_order_code, carrier_code, status, total_volumes, total_weight,
		        notes, shipped_at, created_at, updated_at, created_by
		 FROM public.shipments WHERE code = $1`, code,
	).Scan(&s.ID, &s.Code, &s.SalesOrderCode, &s.CarrierCode, &status, &s.TotalVolumes, &s.TotalWeight,
		&s.Notes, &s.ShippedAt, &s.CreatedAt, &s.UpdatedAt, &s.CreatedBy)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("shipment %d not found", code)
		}
		return nil, fmt.Errorf("getting shipment: %w", err)
	}
	s.Status = entity.ShipmentStatus(status)
	items, err := r.ListItems(ctx, s.ID)
	if err != nil {
		return nil, err
	}
	s.Items = items
	return &s, nil
}

func (r *ShipmentRepositoryPG) List(ctx context.Context) ([]*entity.Shipment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, code, sales_order_code, carrier_code, status, total_volumes, total_weight,
		        notes, shipped_at, created_at, updated_at, created_by
		 FROM public.shipments ORDER BY code DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing shipments: %w", err)
	}
	defer rows.Close()
	return scanShipments(rows)
}

func (r *ShipmentRepositoryPG) ListBySalesOrder(ctx context.Context, salesOrderCode int64) ([]*entity.Shipment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, code, sales_order_code, carrier_code, status, total_volumes, total_weight,
		        notes, shipped_at, created_at, updated_at, created_by
		 FROM public.shipments WHERE sales_order_code = $1 ORDER BY code DESC`, salesOrderCode)
	if err != nil {
		return nil, fmt.Errorf("listing shipments by sales order: %w", err)
	}
	defer rows.Close()
	return scanShipments(rows)
}

func scanShipments(rows pgx.Rows) ([]*entity.Shipment, error) {
	var result []*entity.Shipment
	for rows.Next() {
		var s entity.Shipment
		var status string
		if err := rows.Scan(&s.ID, &s.Code, &s.SalesOrderCode, &s.CarrierCode, &status, &s.TotalVolumes,
			&s.TotalWeight, &s.Notes, &s.ShippedAt, &s.CreatedAt, &s.UpdatedAt, &s.CreatedBy); err != nil {
			return nil, fmt.Errorf("scanning shipment: %w", err)
		}
		s.Status = entity.ShipmentStatus(status)
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (r *ShipmentRepositoryPG) UpdateStatus(ctx context.Context, code int64, status entity.ShipmentStatus) error {
	shippedClause := ""
	if status == entity.ShipmentStatusShipped {
		shippedClause = ", shipped_at = NOW()"
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE public.shipments SET status = $2, updated_at = NOW()`+shippedClause+` WHERE code = $1`,
		code, string(status))
	if err != nil {
		return fmt.Errorf("updating shipment status: %w", err)
	}
	return nil
}

func (r *ShipmentRepositoryPG) AddItem(ctx context.Context, item *entity.ShipmentItem) (*entity.ShipmentItem, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO public.shipment_items
			(shipment_id, sequence, item_code, sales_order_item_code, warehouse_id, quantity, conferred_qty, is_conferred, notes)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		 RETURNING id, created_at`,
		item.ShipmentID, item.Sequence, item.ItemCode, item.SalesOrderItemCode, item.WarehouseID,
		item.Quantity, item.ConferredQty, item.IsConferred, item.Notes,
	).Scan(&item.ID, &item.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("adding shipment item: %w", err)
	}
	return item, nil
}

func (r *ShipmentRepositoryPG) ListItems(ctx context.Context, shipmentID int64) ([]*entity.ShipmentItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, shipment_id, sequence, item_code, sales_order_item_code, warehouse_id,
		        quantity, conferred_qty, is_conferred, notes, created_at
		 FROM public.shipment_items WHERE shipment_id = $1 ORDER BY sequence, id`, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("listing shipment items: %w", err)
	}
	defer rows.Close()
	var result []*entity.ShipmentItem
	for rows.Next() {
		var it entity.ShipmentItem
		if err := rows.Scan(&it.ID, &it.ShipmentID, &it.Sequence, &it.ItemCode, &it.SalesOrderItemCode,
			&it.WarehouseID, &it.Quantity, &it.ConferredQty, &it.IsConferred, &it.Notes, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning shipment item: %w", err)
		}
		result = append(result, &it)
	}
	return result, rows.Err()
}

func (r *ShipmentRepositoryPG) ConferItem(ctx context.Context, itemID int64, conferredQty float64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.shipment_items SET conferred_qty = $2, is_conferred = TRUE WHERE id = $1`,
		itemID, conferredQty)
	if err != nil {
		return fmt.Errorf("conferring shipment item: %w", err)
	}
	return nil
}
