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

const loadCols = `id, code, status, description, carrier_code, vehicle_plate, driver_name, driver_document,
	route_code, origin, destination, dispatch_box_code, planned_ship_date, estimated_delivery,
	started_loading_at, loaded_at, released_at, shipped_at, cancelled_at,
	total_shipments, total_fiscal_notes, total_volumes, total_net_weight, total_gross_weight, total_cubage_m3,
	notes, created_at, updated_at, created_by, updated_by`

func scanLoadRow(row rowScanner) (*entity.ShipmentLoad, error) {
	var l entity.ShipmentLoad
	var status string
	if err := row.Scan(
		&l.ID, &l.Code, &status, &l.Description, &l.CarrierCode, &l.VehiclePlate, &l.DriverName, &l.DriverDocument,
		&l.RouteCode, &l.Origin, &l.Destination, &l.DispatchBoxCode, &l.PlannedShipDate, &l.EstimatedDelivery,
		&l.StartedLoadingAt, &l.LoadedAt, &l.ReleasedAt, &l.ShippedAt, &l.CancelledAt,
		&l.TotalShipments, &l.TotalFiscalNotes, &l.TotalVolumes, &l.TotalNetWeight, &l.TotalGrossWeight, &l.TotalCubageM3,
		&l.Notes, &l.CreatedAt, &l.UpdatedAt, &l.CreatedBy, &l.UpdatedBy,
	); err != nil {
		return nil, err
	}
	l.Status = entity.LoadStatus(status)
	return &l, nil
}

func (r *ShipmentRepositoryPG) NextLoadCode(ctx context.Context) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx,
		`INSERT INTO public.shipment_load_sequences (id, last_number) VALUES (1, 1)
		 ON CONFLICT (id) DO UPDATE SET last_number = shipment_load_sequences.last_number + 1
		 RETURNING last_number`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("next shipment load code: %w", err)
	}
	return n, nil
}

func (r *ShipmentRepositoryPG) CreateLoad(ctx context.Context, in repository.CreateLoadInput) (*entity.ShipmentLoad, error) {
	code, err := r.NextLoadCode(ctx)
	if err != nil {
		return nil, err
	}
	load, err := scanLoadRow(r.pool.QueryRow(ctx,
		`INSERT INTO public.shipment_loads
			(code, status, description, carrier_code, vehicle_plate, driver_name, driver_document,
			 route_code, origin, destination, dispatch_box_code, planned_ship_date, estimated_delivery,
			 notes, created_by)
		 VALUES ($1,'PLANNED',$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		 RETURNING `+loadCols,
		code, in.Description, in.CarrierCode, in.VehiclePlate, in.DriverName, in.DriverDocument,
		in.RouteCode, in.Origin, in.Destination, in.DispatchBoxCode, in.PlannedShipDate, in.EstimatedDelivery,
		in.Notes, in.CreatedBy,
	))
	if err != nil {
		return nil, fmt.Errorf("creating shipment load: %w", err)
	}
	if in.DispatchBoxCode != nil && *in.DispatchBoxCode != "" {
		_ = r.AssignBoxToLoad(ctx, code, *in.DispatchBoxCode, &in.CreatedBy)
	}
	return load, nil
}

func (r *ShipmentRepositoryPG) GetLoadByCode(ctx context.Context, code int64) (*entity.ShipmentLoad, error) {
	load, err := scanLoadRow(r.pool.QueryRow(ctx, `SELECT `+loadCols+` FROM public.shipment_loads WHERE code = $1`, code))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("shipment load %d not found", code)
		}
		return nil, fmt.Errorf("getting shipment load: %w", err)
	}
	if load.Shipments, err = r.listLoadShipments(ctx, load.ID); err != nil {
		return nil, err
	}
	if load.FiscalNotes, err = r.listLoadFiscalNotes(ctx, load.ID); err != nil {
		return nil, err
	}
	loadCode := load.Code
	if load.Instructions, err = r.ListDeliveryInstructions(ctx, &loadCode, true); err != nil {
		return nil, err
	}
	return load, nil
}

func (r *ShipmentRepositoryPG) ListLoads(ctx context.Context, f repository.LoadFilter) ([]*entity.ShipmentLoad, error) {
	q, args := buildLoadListQuery(f, `SELECT `+loadCols+` FROM public.shipment_loads l`)
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("listing shipment loads: %w", err)
	}
	defer rows.Close()
	var result []*entity.ShipmentLoad
	for rows.Next() {
		l, err := scanLoadRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning shipment load: %w", err)
		}
		result = append(result, l)
	}
	return result, rows.Err()
}

func buildLoadListQuery(f repository.LoadFilter, base string) (string, []any) {
	var conds []string
	var args []any
	add := func(cond string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf(cond, len(args)))
	}
	if f.Status != nil {
		add("l.status = $%d", string(*f.Status))
	}
	if f.CarrierCode != nil {
		add("l.carrier_code = $%d", *f.CarrierCode)
	}
	if f.BoxCode != nil && *f.BoxCode != "" {
		add("l.dispatch_box_code = $%d", *f.BoxCode)
	}
	if f.From != nil {
		add("l.planned_ship_date >= $%d", *f.From)
	}
	if f.To != nil {
		add("l.planned_ship_date < $%d", *f.To)
	}
	if len(conds) > 0 {
		base += " WHERE " + strings.Join(conds, " AND ")
	}
	base += " ORDER BY code DESC"
	if f.Limit > 0 {
		args = append(args, f.Limit)
		base += fmt.Sprintf(" LIMIT $%d", len(args))
		if f.Offset > 0 {
			args = append(args, f.Offset)
			base += fmt.Sprintf(" OFFSET $%d", len(args))
		}
	}
	return base, args
}

func (r *ShipmentRepositoryPG) AddShipmentToLoad(ctx context.Context, loadCode, shipmentCode int64, sequence int) (*entity.ShipmentLoadShipment, error) {
	var out entity.ShipmentLoadShipment
	err := r.pool.QueryRow(ctx,
		`WITH l AS (SELECT id, code FROM public.shipment_loads WHERE code = $1),
		      s AS (SELECT id, code FROM public.shipments WHERE code = $2)
		 INSERT INTO public.shipment_load_shipments (load_id, shipment_id, sequence)
		 SELECT l.id, s.id, $3 FROM l, s
		 ON CONFLICT (load_id, shipment_id) DO UPDATE SET sequence = EXCLUDED.sequence
		 RETURNING id, load_id, (SELECT code FROM l), shipment_id, (SELECT code FROM s), sequence, created_at`,
		loadCode, shipmentCode, sequence,
	).Scan(&out.ID, &out.LoadID, &out.LoadCode, &out.ShipmentID, &out.ShipmentCode, &out.Sequence, &out.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("adding shipment to load: %w", err)
	}
	return &out, nil
}

func (r *ShipmentRepositoryPG) RemoveShipmentFromLoad(ctx context.Context, loadCode, shipmentCode int64) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM public.shipment_load_shipments lsi
		 USING public.shipment_loads l, public.shipments s
		 WHERE lsi.load_id = l.id AND lsi.shipment_id = s.id AND l.code = $1 AND s.code = $2`,
		loadCode, shipmentCode)
	if err != nil {
		return fmt.Errorf("removing shipment from load: %w", err)
	}
	return nil
}

func (r *ShipmentRepositoryPG) listLoadShipments(ctx context.Context, loadID int64) ([]*entity.ShipmentLoadShipment, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT lsi.id, lsi.load_id, l.code, lsi.shipment_id, s.code, lsi.sequence, lsi.created_at
		 FROM public.shipment_load_shipments lsi
		 JOIN public.shipment_loads l ON l.id = lsi.load_id
		 JOIN public.shipments s ON s.id = lsi.shipment_id
		 WHERE lsi.load_id = $1 ORDER BY lsi.sequence, lsi.id`, loadID)
	if err != nil {
		return nil, fmt.Errorf("listing load shipments: %w", err)
	}
	defer rows.Close()
	var result []*entity.ShipmentLoadShipment
	for rows.Next() {
		var s entity.ShipmentLoadShipment
		if err := rows.Scan(&s.ID, &s.LoadID, &s.LoadCode, &s.ShipmentID, &s.ShipmentCode, &s.Sequence, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning load shipment: %w", err)
		}
		result = append(result, &s)
	}
	return result, rows.Err()
}

func (r *ShipmentRepositoryPG) AddFiscalNoteToLoad(ctx context.Context, in repository.AddFiscalNoteToLoadInput) (*entity.ShipmentLoadFiscalNote, error) {
	var out entity.ShipmentLoadFiscalNote
	err := r.pool.QueryRow(ctx,
		`WITH l AS (SELECT id, code FROM public.shipment_loads WHERE code = $1),
		      s AS (SELECT id, code FROM public.shipments WHERE code = $2)
		 INSERT INTO public.shipment_load_fiscal_notes (load_id, shipment_id, fiscal_exit_id, nfe_number, nfe_key, sequence)
		 SELECT l.id, CASE WHEN $2::bigint IS NULL THEN NULL ELSE s.id END, $3, $4, $5, $6 FROM l LEFT JOIN s ON TRUE
		 ON CONFLICT (load_id, fiscal_exit_id) DO UPDATE SET
		    shipment_id = EXCLUDED.shipment_id, nfe_number = EXCLUDED.nfe_number,
		    nfe_key = EXCLUDED.nfe_key, sequence = EXCLUDED.sequence
		 RETURNING id, load_id, (SELECT code FROM l), shipment_id,
		           (SELECT code FROM public.shipments WHERE id = shipment_id),
		           fiscal_exit_id, nfe_number, nfe_key, sequence, created_at`,
		in.LoadCode, in.ShipmentCode, in.FiscalExitID, in.NFeNumber, in.NFeKey, in.Sequence,
	).Scan(&out.ID, &out.LoadID, &out.LoadCode, &out.ShipmentID, &out.ShipmentCode,
		&out.FiscalExitID, &out.NFeNumber, &out.NFeKey, &out.Sequence, &out.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("adding fiscal note to load: %w", err)
	}
	return &out, nil
}

func (r *ShipmentRepositoryPG) listLoadFiscalNotes(ctx context.Context, loadID int64) ([]*entity.ShipmentLoadFiscalNote, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT n.id, n.load_id, l.code, n.shipment_id, s.code, n.fiscal_exit_id,
		        n.nfe_number, n.nfe_key, n.sequence, n.created_at
		 FROM public.shipment_load_fiscal_notes n
		 JOIN public.shipment_loads l ON l.id = n.load_id
		 LEFT JOIN public.shipments s ON s.id = n.shipment_id
		 WHERE n.load_id = $1 ORDER BY n.sequence, n.id`, loadID)
	if err != nil {
		return nil, fmt.Errorf("listing load fiscal notes: %w", err)
	}
	defer rows.Close()
	var result []*entity.ShipmentLoadFiscalNote
	for rows.Next() {
		var n entity.ShipmentLoadFiscalNote
		if err := rows.Scan(&n.ID, &n.LoadID, &n.LoadCode, &n.ShipmentID, &n.ShipmentCode,
			&n.FiscalExitID, &n.NFeNumber, &n.NFeKey, &n.Sequence, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning load fiscal note: %w", err)
		}
		result = append(result, &n)
	}
	return result, rows.Err()
}

func loadStatusTimestampColumn(status entity.LoadStatus) string {
	switch status {
	case entity.LoadStatusReleased:
		return "released_at"
	case entity.LoadStatusLoading:
		return "started_loading_at"
	case entity.LoadStatusLoaded:
		return "loaded_at"
	case entity.LoadStatusShipped:
		return "shipped_at"
	case entity.LoadStatusCancelled:
		return "cancelled_at"
	}
	return ""
}

func (r *ShipmentRepositoryPG) UpdateLoadStatus(ctx context.Context, code int64, status entity.LoadStatus, by *uuid.UUID, note string) error {
	setTS := ""
	if col := loadStatusTimestampColumn(status); col != "" {
		setTS = ", " + col + " = NOW()"
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE public.shipment_loads SET status = $2, updated_at = NOW(), updated_by = $3`+setTS+` WHERE code = $1`,
		code, string(status), by)
	if err != nil {
		return fmt.Errorf("updating shipment load status: %w", err)
	}
	return nil
}

func (r *ShipmentRepositoryPG) RecalcLoadTotals(ctx context.Context, code int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.shipment_loads l SET
			total_shipments = COALESCE(s.cnt, 0),
			total_fiscal_notes = COALESCE(n.cnt, 0),
			total_volumes = COALESCE(s.volumes, 0),
			total_net_weight = COALESCE(s.net_weight, 0),
			total_gross_weight = COALESCE(s.gross_weight, 0),
			total_cubage_m3 = COALESCE(s.cubage, 0),
			updated_at = NOW()
		 FROM (SELECT id FROM public.shipment_loads WHERE code = $1) lx
		 LEFT JOIN LATERAL (
			SELECT COUNT(*) cnt, SUM(sh.total_volumes) volumes, SUM(sh.total_net_weight) net_weight,
			       SUM(sh.total_gross_weight) gross_weight, SUM(sh.total_cubage_m3) cubage
			FROM public.shipment_load_shipments lsi
			JOIN public.shipments sh ON sh.id = lsi.shipment_id
			WHERE lsi.load_id = lx.id
		 ) s ON TRUE
		 LEFT JOIN LATERAL (
			SELECT COUNT(*) cnt FROM public.shipment_load_fiscal_notes WHERE load_id = lx.id
		 ) n ON TRUE
		 WHERE l.id = lx.id`, code)
	if err != nil {
		return fmt.Errorf("recalculating load totals: %w", err)
	}
	return nil
}

func (r *ShipmentRepositoryPG) CreateDeliveryInstruction(ctx context.Context, d *entity.DeliveryInstruction) (*entity.DeliveryInstruction, error) {
	err := r.pool.QueryRow(ctx,
		`WITH l AS (SELECT id, code FROM public.shipment_loads WHERE code = $1)
		 INSERT INTO public.shipment_delivery_instructions (load_id, customer_id, title, instruction, priority, active)
		 VALUES ((SELECT id FROM l), $2, $3, $4, $5, $6)
		 RETURNING id, load_id, (SELECT code FROM l), customer_id, title, instruction, priority, active, created_at, updated_at`,
		d.LoadCode, d.CustomerID, d.Title, d.Instruction, d.Priority, d.Active,
	).Scan(&d.ID, &d.LoadID, &d.LoadCode, &d.CustomerID, &d.Title, &d.Instruction, &d.Priority, &d.Active, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating delivery instruction: %w", err)
	}
	return d, nil
}

func (r *ShipmentRepositoryPG) ListDeliveryInstructions(ctx context.Context, loadCode *int64, activeOnly bool) ([]*entity.DeliveryInstruction, error) {
	var conds []string
	var args []any
	if loadCode != nil {
		args = append(args, *loadCode)
		conds = append(conds, fmt.Sprintf("l.code = $%d", len(args)))
	}
	if activeOnly {
		conds = append(conds, "i.active = TRUE")
	}
	q := `SELECT i.id, i.load_id, l.code, i.customer_id, i.title, i.instruction, i.priority,
	             i.active, i.created_at, i.updated_at
	      FROM public.shipment_delivery_instructions i
	      LEFT JOIN public.shipment_loads l ON l.id = i.load_id`
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += " ORDER BY i.priority, i.id"
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("listing delivery instructions: %w", err)
	}
	defer rows.Close()
	var result []*entity.DeliveryInstruction
	for rows.Next() {
		var d entity.DeliveryInstruction
		if err := rows.Scan(&d.ID, &d.LoadID, &d.LoadCode, &d.CustomerID, &d.Title, &d.Instruction, &d.Priority, &d.Active, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning delivery instruction: %w", err)
		}
		result = append(result, &d)
	}
	return result, rows.Err()
}

func (r *ShipmentRepositoryPG) CreateDispatchBox(ctx context.Context, b *entity.DispatchBox) (*entity.DispatchBox, error) {
	err := r.pool.QueryRow(ctx,
		`INSERT INTO public.shipment_dispatch_boxes (code, description, warehouse_id, zone, active)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (code) DO UPDATE SET description = EXCLUDED.description, warehouse_id = EXCLUDED.warehouse_id,
		    zone = EXCLUDED.zone, active = EXCLUDED.active, updated_at = NOW()
		 RETURNING id, code, description, warehouse_id, zone, active, current_load, created_at, updated_at`,
		b.Code, b.Description, b.WarehouseID, b.Zone, b.Active,
	).Scan(&b.ID, &b.Code, &b.Description, &b.WarehouseID, &b.Zone, &b.Active, &b.CurrentLoad, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("creating dispatch box: %w", err)
	}
	return b, nil
}

func (r *ShipmentRepositoryPG) ListDispatchBoxes(ctx context.Context, activeOnly bool) ([]*entity.DispatchBox, error) {
	q := `SELECT id, code, description, warehouse_id, zone, active, current_load, created_at, updated_at
	      FROM public.shipment_dispatch_boxes`
	if activeOnly {
		q += " WHERE active = TRUE"
	}
	q += " ORDER BY code"
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("listing dispatch boxes: %w", err)
	}
	defer rows.Close()
	var result []*entity.DispatchBox
	for rows.Next() {
		var b entity.DispatchBox
		if err := rows.Scan(&b.ID, &b.Code, &b.Description, &b.WarehouseID, &b.Zone, &b.Active, &b.CurrentLoad, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning dispatch box: %w", err)
		}
		result = append(result, &b)
	}
	return result, rows.Err()
}

func (r *ShipmentRepositoryPG) AssignBoxToLoad(ctx context.Context, loadCode int64, boxCode string, by *uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin assign box: %w", err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx,
		`UPDATE public.shipment_loads SET dispatch_box_code = $2, updated_at = NOW(), updated_by = $3 WHERE code = $1`,
		loadCode, boxCode, by)
	if err != nil {
		return fmt.Errorf("assigning box to load: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("shipment load %d not found", loadCode)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE public.shipment_dispatch_boxes SET current_load = NULL, updated_at = NOW() WHERE current_load = $1`,
		loadCode); err != nil {
		return fmt.Errorf("clearing previous box assignment: %w", err)
	}
	tag, err = tx.Exec(ctx,
		`UPDATE public.shipment_dispatch_boxes SET current_load = $2, updated_at = NOW() WHERE code = $1`,
		boxCode, loadCode)
	if err != nil {
		return fmt.Errorf("updating dispatch box assignment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("dispatch box %s not found", boxCode)
	}
	return tx.Commit(ctx)
}

func (r *ShipmentRepositoryPG) LoadMonitor(ctx context.Context, f repository.LoadFilter) ([]*repository.LoadMonitorRow, error) {
	q, args := buildLoadListQuery(f,
		`SELECT l.code, l.status, l.carrier_code, l.vehicle_plate, l.driver_name, l.dispatch_box_code,
		        l.planned_ship_date, l.estimated_delivery, l.total_shipments, l.total_fiscal_notes,
		        l.total_volumes, l.total_net_weight, l.total_gross_weight, l.total_cubage_m3,
		        COALESCE(SUM(CASE WHEN s.status = 'OPEN' THEN 1 ELSE 0 END),0)::int,
		        COALESCE(SUM(CASE WHEN s.status = 'SEPARATED' THEN 1 ELSE 0 END),0)::int,
		        COALESCE(SUM(CASE WHEN s.status = 'CONFERRED' THEN 1 ELSE 0 END),0)::int,
		        COALESCE(SUM(CASE WHEN s.status = 'SHIPPED' THEN 1 ELSE 0 END),0)::int
		 FROM public.shipment_loads l
		 LEFT JOIN public.shipment_load_shipments lsi ON lsi.load_id = l.id
		 LEFT JOIN public.shipments s ON s.id = lsi.shipment_id`)
	q = strings.Replace(q, " ORDER BY code DESC", " GROUP BY l.id ORDER BY l.code DESC", 1)
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("load monitor: %w", err)
	}
	defer rows.Close()
	var result []*repository.LoadMonitorRow
	for rows.Next() {
		var row repository.LoadMonitorRow
		var status string
		if err := rows.Scan(&row.LoadCode, &status, &row.CarrierCode, &row.VehiclePlate, &row.DriverName, &row.DispatchBoxCode,
			&row.PlannedShipDate, &row.EstimatedDelivery, &row.TotalShipments, &row.TotalFiscalNotes,
			&row.TotalVolumes, &row.TotalNetWeight, &row.TotalGrossWeight, &row.TotalCubageM3,
			&row.OpenShipments, &row.SeparatedShipments, &row.ConferredShipments, &row.ShippedShipments); err != nil {
			return nil, fmt.Errorf("scanning load monitor: %w", err)
		}
		row.Status = entity.LoadStatus(status)
		result = append(result, &row)
	}
	return result, rows.Err()
}

func (r *ShipmentRepositoryPG) SeparationMonitor(ctx context.Context, f repository.LoadFilter) ([]*repository.SeparationMonitorRow, error) {
	q, args := buildLoadListQuery(f,
		`SELECT s.code, l.code, s.status, l.status, s.sales_order_code, COALESCE(l.carrier_code, s.carrier_code),
		        l.dispatch_box_code, COUNT(si.id)::int,
		        COALESCE(SUM(CASE WHEN si.is_conferred THEN 1 ELSE 0 END),0)::int,
		        COALESCE(SUM(CASE WHEN si.is_conferred AND si.conferred_qty <> si.quantity THEN 1 ELSE 0 END),0)::int,
		        s.total_volumes, s.total_gross_weight
		 FROM public.shipments s
		 LEFT JOIN public.shipment_load_shipments lsi ON lsi.shipment_id = s.id
		 LEFT JOIN public.shipment_loads l ON l.id = lsi.load_id
		 LEFT JOIN public.shipment_items si ON si.shipment_id = s.id`)
	q = strings.Replace(q, " ORDER BY code DESC", " GROUP BY s.id, l.id ORDER BY COALESCE(l.code, 0) DESC, s.code DESC", 1)
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("separation monitor: %w", err)
	}
	defer rows.Close()
	var result []*repository.SeparationMonitorRow
	for rows.Next() {
		var row repository.SeparationMonitorRow
		var shipStatus string
		var loadStatus *string
		if err := rows.Scan(&row.ShipmentCode, &row.LoadCode, &shipStatus, &loadStatus, &row.SalesOrderCode, &row.CarrierCode,
			&row.DispatchBoxCode, &row.TotalItems, &row.ConferredItems, &row.DivergentItems,
			&row.TotalVolumes, &row.TotalGrossWeight); err != nil {
			return nil, fmt.Errorf("scanning separation monitor: %w", err)
		}
		row.ShipmentStatus = entity.ShipmentStatus(shipStatus)
		if loadStatus != nil {
			st := entity.LoadStatus(*loadStatus)
			row.LoadStatus = &st
		}
		result = append(result, &row)
	}
	return result, rows.Err()
}

func (r *ShipmentRepositoryPG) LogisticPanel(ctx context.Context) (*repository.LogisticPanelSummary, error) {
	var s repository.LogisticPanelSummary
	err := r.pool.QueryRow(ctx,
		`SELECT
			COALESCE(SUM(CASE WHEN l.status = 'PLANNED' THEN 1 ELSE 0 END),0)::int,
			COALESCE(SUM(CASE WHEN l.status = 'RELEASED' THEN 1 ELSE 0 END),0)::int,
			COALESCE(SUM(CASE WHEN l.status = 'LOADING' THEN 1 ELSE 0 END),0)::int,
			COALESCE(SUM(CASE WHEN l.status = 'LOADED' THEN 1 ELSE 0 END),0)::int,
			COALESCE(SUM(CASE WHEN l.status = 'SHIPPED' THEN 1 ELSE 0 END),0)::int,
			COALESCE(SUM(CASE WHEN l.status = 'CANCELLED' THEN 1 ELSE 0 END),0)::int,
			(SELECT COUNT(*) FROM public.shipments WHERE status = 'OPEN')::int,
			(SELECT COUNT(*) FROM public.shipments WHERE status = 'SEPARATED')::int,
			(SELECT COUNT(*) FROM public.shipments WHERE status = 'CONFERRED')::int,
			(SELECT COUNT(*) FROM public.shipment_dispatch_boxes WHERE active AND current_load IS NOT NULL)::int,
			(SELECT COUNT(*) FROM public.shipment_dispatch_boxes WHERE active AND current_load IS NULL)::int,
			COALESCE(SUM(l.total_volumes),0)::int,
			COALESCE(SUM(l.total_gross_weight),0)
		 FROM public.shipment_loads l`).Scan(
		&s.PlannedLoads, &s.ReleasedLoads, &s.LoadingLoads, &s.LoadedLoads, &s.ShippedLoads, &s.CancelledLoads,
		&s.OpenShipments, &s.SeparatedShipments, &s.ConferredShipments, &s.BoxesOccupied, &s.BoxesAvailable,
		&s.TotalVolumes, &s.TotalGrossWeight,
	)
	if err != nil {
		return nil, fmt.Errorf("logistic panel: %w", err)
	}
	return &s, nil
}
