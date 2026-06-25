package shipment

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/repository"
	"github.com/google/uuid"
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

// shipmentCols is the full column projection shared by every shipment SELECT, so
// the scanner stays in lockstep with the queries.
const shipmentCols = `id, code, sales_order_code, carrier_code, status, total_volumes,
	total_net_weight, total_gross_weight, total_cubage_m3,
	freight_modality, freight_value, insurance_value,
	vehicle_plate, driver_name, driver_document, antt_code, seals, estimated_delivery,
	fiscal_exit_id, nfe_number, nfe_key,
	notes, separated_at, conferred_at, shipped_at, cancelled_at,
	created_at, updated_at, created_by, updated_by,
	reference_type, purchase_order_code, production_order_code`

type rowScanner interface{ Scan(dest ...any) error }

func scanShipmentRow(row rowScanner) (*entity.Shipment, error) {
	var s entity.Shipment
	var status string
	var refType *string
	if err := row.Scan(
		&s.ID, &s.Code, &s.SalesOrderCode, &s.CarrierCode, &status, &s.TotalVolumes,
		&s.TotalNetWeight, &s.TotalGrossWeight, &s.TotalCubageM3,
		&s.FreightModality, &s.FreightValue, &s.InsuranceValue,
		&s.VehiclePlate, &s.DriverName, &s.DriverDocument, &s.ANTTCode, &s.Seals, &s.EstimatedDelivery,
		&s.FiscalExitID, &s.NFeNumber, &s.NFeKey,
		&s.Notes, &s.SeparatedAt, &s.ConferredAt, &s.ShippedAt, &s.CancelledAt,
		&s.CreatedAt, &s.UpdatedAt, &s.CreatedBy, &s.UpdatedBy,
		&refType, &s.PurchaseOrderCode, &s.ProductionOrderCode,
	); err != nil {
		return nil, err
	}
	s.Status = entity.ShipmentStatus(status)
	if refType != nil {
		rt := entity.ShipmentReferenceType(*refType)
		s.ReferenceType = &rt
	}
	return &s, nil
}

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
			(code, sales_order_code, carrier_code, status, total_volumes,
			 total_weight, total_net_weight, total_gross_weight, total_cubage_m3,
			 notes, created_by, reference_type, purchase_order_code, production_order_code)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		 RETURNING id, created_at, updated_at`,
		s.Code, s.SalesOrderCode, s.CarrierCode, string(s.Status), s.TotalVolumes,
		s.TotalGrossWeight, s.TotalNetWeight, s.TotalGrossWeight, s.TotalCubageM3,
		s.Notes, s.CreatedBy, s.ReferenceType, s.PurchaseOrderCode, s.ProductionOrderCode,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating shipment: %w", err)
	}
	_ = r.AddEvent(ctx, &entity.ShipmentEvent{ShipmentID: s.ID, Event: "CREATED", CreatedBy: ptrUUID(s.CreatedBy)})
	return s, nil
}

func (r *ShipmentRepositoryPG) GetByCode(ctx context.Context, code int64) (*entity.Shipment, error) {
	s, err := scanShipmentRow(r.pool.QueryRow(ctx,
		`SELECT `+shipmentCols+` FROM public.shipments WHERE code = $1`, code))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("shipment %d not found", code)
		}
		return nil, fmt.Errorf("getting shipment: %w", err)
	}
	if s.Items, err = r.ListItems(ctx, s.ID); err != nil {
		return nil, err
	}
	if s.Volumes, err = r.ListVolumes(ctx, s.ID); err != nil {
		return nil, err
	}
	return s, nil
}

func (r *ShipmentRepositoryPG) List(ctx context.Context) ([]*entity.Shipment, error) {
	return r.ListFiltered(ctx, repository.ShipmentFilter{})
}

func (r *ShipmentRepositoryPG) ListFiltered(ctx context.Context, f repository.ShipmentFilter) ([]*entity.Shipment, error) {
	var conds []string
	var args []any
	add := func(cond string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	if f.Status != nil {
		add("status = $%d", string(*f.Status))
	}
	if f.CarrierCode != nil {
		add("carrier_code = $%d", *f.CarrierCode)
	}
	if f.From != nil {
		add("created_at >= $%d", *f.From)
	}
	if f.To != nil {
		add("created_at < $%d", *f.To)
	}
	q := `SELECT ` + shipmentCols + ` FROM public.shipments`
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY code DESC"
	if f.Limit > 0 {
		args = append(args, f.Limit)
		q += fmt.Sprintf(" LIMIT $%d", len(args))
		if f.Offset > 0 {
			args = append(args, f.Offset)
			q += fmt.Sprintf(" OFFSET $%d", len(args))
		}
	}
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("listing shipments: %w", err)
	}
	defer rows.Close()
	return scanShipments(rows)
}

func (r *ShipmentRepositoryPG) ListBySalesOrder(ctx context.Context, code int64) ([]*entity.Shipment, error) {
	return r.listByCol(ctx, "sales_order_code", code)
}

func (r *ShipmentRepositoryPG) ListByPurchaseOrder(ctx context.Context, code int64) ([]*entity.Shipment, error) {
	return r.listByCol(ctx, "purchase_order_code", code)
}

func (r *ShipmentRepositoryPG) ListByProductionOrder(ctx context.Context, code int64) ([]*entity.Shipment, error) {
	return r.listByCol(ctx, "production_order_code", code)
}

func (r *ShipmentRepositoryPG) listByCol(ctx context.Context, col string, code int64) ([]*entity.Shipment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+shipmentCols+` FROM public.shipments WHERE `+col+` = $1 ORDER BY code DESC`, code)
	if err != nil {
		return nil, fmt.Errorf("listing shipments by %s: %w", col, err)
	}
	defer rows.Close()
	return scanShipments(rows)
}

func (r *ShipmentRepositoryPG) ListByReference(ctx context.Context, refType entity.ShipmentReferenceType, refCode int64) ([]*entity.Shipment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+shipmentCols+` FROM public.shipments WHERE reference_type = $1
		   AND ((reference_type = 'SALES_ORDER' AND sales_order_code = $2)
		     OR (reference_type = 'PURCHASE_ORDER' AND purchase_order_code = $2)
		     OR (reference_type = 'PRODUCTION_ORDER' AND production_order_code = $2))
		 ORDER BY code DESC`, string(refType), refCode)
	if err != nil {
		return nil, fmt.Errorf("listing shipments by reference %s/%d: %w", refType, refCode, err)
	}
	defer rows.Close()
	return scanShipments(rows)
}

func scanShipments(rows pgx.Rows) ([]*entity.Shipment, error) {
	var result []*entity.Shipment
	for rows.Next() {
		s, err := scanShipmentRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning shipment: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

// statusTimestampColumn maps a status to the column stamped on entering it.
func statusTimestampColumn(status entity.ShipmentStatus) string {
	switch status {
	case entity.ShipmentStatusSeparated:
		return "separated_at"
	case entity.ShipmentStatusConferred:
		return "conferred_at"
	case entity.ShipmentStatusShipped:
		return "shipped_at"
	case entity.ShipmentStatusCancelled:
		return "cancelled_at"
	}
	return ""
}

func (r *ShipmentRepositoryPG) UpdateStatus(ctx context.Context, code int64, status entity.ShipmentStatus, by *uuid.UUID, note string) error {
	setTS := ""
	if col := statusTimestampColumn(status); col != "" {
		setTS = ", " + col + " = NOW()"
	}
	var id int64
	err := r.pool.QueryRow(ctx,
		`UPDATE public.shipments SET status = $2, updated_at = NOW(), updated_by = $3`+setTS+
			` WHERE code = $1 RETURNING id`,
		code, string(status), by).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("shipment %d not found", code)
		}
		return fmt.Errorf("updating shipment status: %w", err)
	}
	return r.AddEvent(ctx, &entity.ShipmentEvent{ShipmentID: id, Event: string(status), Note: nilIfEmpty(note), CreatedBy: by})
}

func (r *ShipmentRepositoryPG) UpdateTransport(ctx context.Context, code int64, t repository.TransportInput, by *uuid.UUID) error {
	var id int64
	err := r.pool.QueryRow(ctx,
		`UPDATE public.shipments SET
			carrier_code = COALESCE($2, carrier_code),
			freight_modality = $3, freight_value = $4, insurance_value = $5,
			vehicle_plate = $6, driver_name = $7, driver_document = $8,
			antt_code = $9, seals = $10, estimated_delivery = $11,
			updated_at = NOW(), updated_by = $12
		 WHERE code = $1 RETURNING id`,
		code, t.CarrierCode, t.FreightModality, t.FreightValue, t.InsuranceValue,
		t.VehiclePlate, t.DriverName, t.DriverDocument, t.ANTTCode, t.Seals, t.EstimatedDelivery, by,
	).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("shipment %d not found", code)
		}
		return fmt.Errorf("updating shipment transport: %w", err)
	}
	return r.AddEvent(ctx, &entity.ShipmentEvent{ShipmentID: id, Event: "TRANSPORT", CreatedBy: by})
}

func (r *ShipmentRepositoryPG) SetFiscalExit(ctx context.Context, code int64, fiscalExitID, nfeNumber *int64, nfeKey *string, by *uuid.UUID) error {
	var id int64
	err := r.pool.QueryRow(ctx,
		`UPDATE public.shipments SET fiscal_exit_id = $2, nfe_number = $3, nfe_key = $4,
		        updated_at = NOW(), updated_by = $5
		 WHERE code = $1 RETURNING id`,
		code, fiscalExitID, nfeNumber, nfeKey, by).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("shipment %d not found", code)
		}
		return fmt.Errorf("linking shipment to NF-e: %w", err)
	}
	return r.AddEvent(ctx, &entity.ShipmentEvent{ShipmentID: id, Event: "NFE_LINKED", CreatedBy: by})
}

// RecalcTotals recomputes header totals (volumes, net/gross weight, cubage) from
// the persisted volumes; falls back to item weights when there are no volumes.
func (r *ShipmentRepositoryPG) RecalcTotals(ctx context.Context, code int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.shipments s SET
			total_volumes = COALESCE(v.cnt, 0),
			total_net_weight = COALESCE(v.net, i.net, 0),
			total_gross_weight = COALESCE(v.gross, i.gross, 0),
			total_weight = COALESCE(v.gross, i.gross, 0),
			total_cubage_m3 = COALESCE(v.cub, 0),
			updated_at = NOW()
		 FROM (SELECT id FROM public.shipments WHERE code = $1) sx
		 LEFT JOIN LATERAL (
			SELECT COUNT(*) cnt, SUM(net_weight) net, SUM(gross_weight) gross, SUM(cubage_m3) cub
			FROM public.shipment_volumes WHERE shipment_id = sx.id
		 ) v ON TRUE
		 LEFT JOIN LATERAL (
			SELECT SUM(quantity*unit_net_weight) net, SUM(quantity*unit_gross_weight) gross
			FROM public.shipment_items WHERE shipment_id = sx.id
		 ) i ON TRUE
		 WHERE s.id = sx.id`, code)
	if err != nil {
		return fmt.Errorf("recalculating shipment totals: %w", err)
	}
	return nil
}

func (r *ShipmentRepositoryPG) AddItem(ctx context.Context, item *entity.ShipmentItem) (*entity.ShipmentItem, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO public.shipment_items
			(shipment_id, sequence, item_code, sales_order_item_code, warehouse_id,
			 quantity, conferred_qty, is_conferred, unit_net_weight, unit_gross_weight, notes)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 RETURNING id, created_at`,
		item.ShipmentID, item.Sequence, item.ItemCode, item.SalesOrderItemCode, item.WarehouseID,
		item.Quantity, item.ConferredQty, item.IsConferred, item.UnitNetWeight, item.UnitGrossWeight, item.Notes,
	).Scan(&item.ID, &item.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("adding shipment item: %w", err)
	}
	return item, nil
}

const shipmentItemCols = `id, shipment_id, sequence, item_code, sales_order_item_code, warehouse_id,
	quantity, conferred_qty, is_conferred, unit_net_weight, unit_gross_weight, notes, created_at`

func scanShipmentItem(row rowScanner) (*entity.ShipmentItem, error) {
	var it entity.ShipmentItem
	if err := row.Scan(&it.ID, &it.ShipmentID, &it.Sequence, &it.ItemCode, &it.SalesOrderItemCode,
		&it.WarehouseID, &it.Quantity, &it.ConferredQty, &it.IsConferred,
		&it.UnitNetWeight, &it.UnitGrossWeight, &it.Notes, &it.CreatedAt); err != nil {
		return nil, err
	}
	return &it, nil
}

func (r *ShipmentRepositoryPG) ListItems(ctx context.Context, shipmentID int64) ([]*entity.ShipmentItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+shipmentItemCols+` FROM public.shipment_items WHERE shipment_id = $1 ORDER BY sequence, id`, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("listing shipment items: %w", err)
	}
	defer rows.Close()
	var result []*entity.ShipmentItem
	for rows.Next() {
		it, err := scanShipmentItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning shipment item: %w", err)
		}
		result = append(result, it)
	}
	return result, rows.Err()
}

func (r *ShipmentRepositoryPG) GetItem(ctx context.Context, itemID int64) (*entity.ShipmentItem, error) {
	it, err := scanShipmentItem(r.pool.QueryRow(ctx,
		`SELECT `+shipmentItemCols+` FROM public.shipment_items WHERE id = $1`, itemID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("shipment item %d not found", itemID)
		}
		return nil, fmt.Errorf("getting shipment item: %w", err)
	}
	return it, nil
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

func (r *ShipmentRepositoryPG) AddVolume(ctx context.Context, v *entity.ShipmentVolume) (*entity.ShipmentVolume, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO public.shipment_volumes
			(shipment_id, volume_number, package_type, net_weight, gross_weight,
			 length_cm, width_cm, height_cm, cubage_m3, marking, contents)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 RETURNING id, created_at`,
		v.ShipmentID, v.VolumeNumber, v.PackageType, v.NetWeight, v.GrossWeight,
		v.LengthCm, v.WidthCm, v.HeightCm, v.CubageM3, v.Marking, v.Contents,
	).Scan(&v.ID, &v.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("adding shipment volume: %w", err)
	}
	return v, nil
}

func (r *ShipmentRepositoryPG) ListVolumes(ctx context.Context, shipmentID int64) ([]*entity.ShipmentVolume, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, shipment_id, volume_number, package_type, net_weight, gross_weight,
		        length_cm, width_cm, height_cm, cubage_m3, marking, contents, created_at
		 FROM public.shipment_volumes WHERE shipment_id = $1 ORDER BY volume_number, id`, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("listing shipment volumes: %w", err)
	}
	defer rows.Close()
	var result []*entity.ShipmentVolume
	for rows.Next() {
		var v entity.ShipmentVolume
		if err := rows.Scan(&v.ID, &v.ShipmentID, &v.VolumeNumber, &v.PackageType, &v.NetWeight, &v.GrossWeight,
			&v.LengthCm, &v.WidthCm, &v.HeightCm, &v.CubageM3, &v.Marking, &v.Contents, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning shipment volume: %w", err)
		}
		result = append(result, &v)
	}
	return result, rows.Err()
}

func (r *ShipmentRepositoryPG) DeleteVolume(ctx context.Context, volumeID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM public.shipment_volumes WHERE id = $1`, volumeID)
	if err != nil {
		return fmt.Errorf("deleting shipment volume: %w", err)
	}
	return nil
}

func (r *ShipmentRepositoryPG) AddEvent(ctx context.Context, e *entity.ShipmentEvent) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.shipment_events (shipment_id, event, note, created_by)
		 VALUES ($1,$2,$3,$4)`, e.ShipmentID, e.Event, e.Note, e.CreatedBy)
	if err != nil {
		return fmt.Errorf("adding shipment event: %w", err)
	}
	return nil
}

func (r *ShipmentRepositoryPG) ListEvents(ctx context.Context, shipmentID int64) ([]*entity.ShipmentEvent, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, shipment_id, event, note, created_by, created_at
		 FROM public.shipment_events WHERE shipment_id = $1 ORDER BY created_at, id`, shipmentID)
	if err != nil {
		return nil, fmt.Errorf("listing shipment events: %w", err)
	}
	defer rows.Close()
	var result []*entity.ShipmentEvent
	for rows.Next() {
		var e entity.ShipmentEvent
		if err := rows.Scan(&e.ID, &e.ShipmentID, &e.Event, &e.Note, &e.CreatedBy, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning shipment event: %w", err)
		}
		result = append(result, &e)
	}
	return result, rows.Err()
}

func ptrUUID(u uuid.UUID) *uuid.UUID { return &u }

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
