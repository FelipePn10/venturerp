package purchase_order

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *PurchaseOrderRepositorySQLC) NextOrderNumber(ctx context.Context, enterpriseCode int64) (int64, error) {
	var lastNum int64
	err := r.db.QueryRow(ctx,
		`INSERT INTO purchase_order_sequences (enterprise_code, last_number)
		 VALUES ($1, 1)
		 ON CONFLICT (enterprise_code)
		 DO UPDATE SET last_number = purchase_order_sequences.last_number + 1
		 RETURNING last_number`,
		enterpriseCode,
	).Scan(&lastNum)
	if err != nil {
		return 0, fmt.Errorf("next purchase order number: %w", err)
	}
	return lastNum, nil
}

// pgQuerier is satisfied by both *pgxpool.Pool and pgx.Tx, letting the insert
// helpers run either standalone or inside a transaction.
type pgQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (r *PurchaseOrderRepositorySQLC) Create(ctx context.Context, o *entity.PurchaseOrder) (*entity.PurchaseOrder, error) {
	return insertOrder(ctx, r.db, o)
}

// CreateWithItems creates the order and its items atomically in one transaction.
func (r *PurchaseOrderRepositorySQLC) CreateWithItems(ctx context.Context, o *entity.PurchaseOrder, items []*entity.PurchaseOrderItem) (*entity.PurchaseOrder, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	created, err := insertOrder(ctx, tx, o)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		item.PurchaseOrderCode = created.Code
		ci, ierr := insertItem(ctx, tx, item)
		if ierr != nil {
			return nil, ierr
		}
		created.Items = append(created.Items, ci)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}
	return created, nil
}

func insertOrder(ctx context.Context, q pgQuerier, o *entity.PurchaseOrder) (*entity.PurchaseOrder, error) {
	var deliveryDate pgtype.Date
	if o.DeliveryDate != nil {
		deliveryDate = pgtype.Date{Time: *o.DeliveryDate, Valid: true}
	}
	freightType := o.FreightType
	if freightType == "" {
		freightType = "SEM_FRETE"
	}
	alcada := o.AlcadaStatus
	if alcada == "" {
		alcada = "N"
	}

	var row purchaseOrderRow
	err := q.QueryRow(ctx,
		`INSERT INTO public.purchase_orders (
			order_number, enterprise_code, status, origin, emission_date,
			delivery_date, supplier_code, payment_term_code, currency_code,
			shipping_address_code, notes, total_gross, total_net, total_discount,
			is_firm, created_by,
			price_table_code, invoice_type_code, financial_account, request_type_code, currency_date,
			freight_type, freight_value_type, freight_value_mode, freight_value, carrier_code,
			redispatch_carrier_code, redispatch_freight_type, redispatch_freight_value,
			advance_date, advance_value, incoterm_code, shipment_date, talao_number, alcada_status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,
			$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35)
		RETURNING code, order_number, enterprise_code, status, origin, emission_date,
			delivery_date, supplier_code, payment_term_code, currency_code,
			shipping_address_code, notes, total_gross, total_net, total_discount,
			is_active, is_firm, created_at, updated_at, created_by,
			price_table_code, invoice_type_code, financial_account, request_type_code, currency_date,
			freight_type, freight_value_type, freight_value_mode, freight_value, carrier_code,
			redispatch_carrier_code, redispatch_freight_type, redispatch_freight_value,
			advance_date, advance_value, incoterm_code, shipment_date, talao_number, alcada_status`,
		o.OrderNumber, o.EnterpriseCode, string(o.Status), string(o.Origin), o.EmissionDate,
		deliveryDate, o.SupplierCode, o.PaymentTermCode, o.CurrencyCode,
		o.ShippingAddressCode, o.Notes, o.TotalGross, o.TotalNet, o.TotalDiscount,
		o.IsFirm, o.CreatedBy,
		o.PriceTableCode, o.InvoiceTypeCode, pgutil.ToPgTextFromPtr(o.FinancialAccount), o.RequestTypeCode, pgutil.ToPgDateFromPtr(o.CurrencyDate),
		freightType, pgutil.ToPgTextFromPtr(o.FreightValueType), pgutil.ToPgTextFromPtr(o.FreightValueMode), o.FreightValue, o.CarrierCode,
		o.RedispatchCarrierCode, pgutil.ToPgTextFromPtr(o.RedispatchFreightType), o.RedispatchFreightValue,
		pgutil.ToPgDateFromPtr(o.AdvanceDate), o.AdvanceValue, pgutil.ToPgTextFromPtr(o.IncotermCode), pgutil.ToPgDateFromPtr(o.ShipmentDate), pgutil.ToPgTextFromPtr(o.TalaoNumber), alcada,
	).Scan(
		&row.Code, &row.OrderNumber, &row.EnterpriseCode, &row.Status, &row.Origin, &row.EmissionDate,
		&row.DeliveryDate, &row.SupplierCode, &row.PaymentTermCode, &row.CurrencyCode,
		&row.ShippingAddressCode, &row.Notes, &row.TotalGross, &row.TotalNet, &row.TotalDiscount,
		&row.IsActive, &row.IsFirm, &row.CreatedAt, &row.UpdatedAt, &row.CreatedBy,
		&row.PriceTableCode, &row.InvoiceTypeCode, &row.FinancialAccount, &row.RequestTypeCode, &row.CurrencyDate,
		&row.FreightType, &row.FreightValueType, &row.FreightValueMode, &row.FreightValue, &row.CarrierCode,
		&row.RedispatchCarrierCode, &row.RedispatchFreightType, &row.RedispatchFreightValue,
		&row.AdvanceDate, &row.AdvanceValue, &row.IncotermCode, &row.ShipmentDate, &row.TalaoNumber, &row.AlcadaStatus,
	)
	if err != nil {
		return nil, fmt.Errorf("creating purchase order: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *PurchaseOrderRepositorySQLC) Update(ctx context.Context, o *entity.PurchaseOrder) (*entity.PurchaseOrder, error) {
	var deliveryDate pgtype.Date
	if o.DeliveryDate != nil {
		deliveryDate = pgtype.Date{Time: *o.DeliveryDate, Valid: true}
	}

	var row purchaseOrderRow
	err := r.db.QueryRow(ctx,
		`UPDATE public.purchase_orders SET
			status = $2, origin = $3, delivery_date = $4, supplier_code = $5,
			payment_term_code = $6, currency_code = $7, shipping_address_code = $8,
			notes = $9, total_gross = $10, total_net = $11, total_discount = $12,
			is_firm = $13, updated_at = NOW()
		WHERE code = $1 AND is_active = true
		RETURNING code, order_number, enterprise_code, status, origin, emission_date,
			delivery_date, supplier_code, payment_term_code, currency_code,
			shipping_address_code, notes, total_gross, total_net, total_discount,
			is_active, is_firm, created_at, updated_at, created_by`,
		o.Code, string(o.Status), string(o.Origin), deliveryDate, o.SupplierCode,
		o.PaymentTermCode, o.CurrencyCode, o.ShippingAddressCode,
		o.Notes, o.TotalGross, o.TotalNet, o.TotalDiscount, o.IsFirm,
	).Scan(
		&row.Code, &row.OrderNumber, &row.EnterpriseCode, &row.Status, &row.Origin, &row.EmissionDate,
		&row.DeliveryDate, &row.SupplierCode, &row.PaymentTermCode, &row.CurrencyCode,
		&row.ShippingAddressCode, &row.Notes, &row.TotalGross, &row.TotalNet, &row.TotalDiscount,
		&row.IsActive, &row.IsFirm, &row.CreatedAt, &row.UpdatedAt, &row.CreatedBy,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("purchase order %d not found or inactive", o.Code)
		}
		return nil, fmt.Errorf("updating purchase order: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *PurchaseOrderRepositorySQLC) GetByCode(ctx context.Context, code int64) (*entity.PurchaseOrder, error) {
	var row purchaseOrderRow
	err := r.db.QueryRow(ctx,
		`SELECT code, order_number, enterprise_code, status, origin, emission_date,
			delivery_date, supplier_code, payment_term_code, currency_code,
			shipping_address_code, notes, total_gross, total_net, total_discount,
			is_active, is_firm, created_at, updated_at, created_by,
			price_table_code, invoice_type_code, financial_account, request_type_code, currency_date,
			freight_type, freight_value_type, freight_value_mode, freight_value, carrier_code,
			redispatch_carrier_code, redispatch_freight_type, redispatch_freight_value,
			advance_date, advance_value, incoterm_code, shipment_date, talao_number, alcada_status
		FROM public.purchase_orders WHERE code = $1`, code,
	).Scan(
		&row.Code, &row.OrderNumber, &row.EnterpriseCode, &row.Status, &row.Origin, &row.EmissionDate,
		&row.DeliveryDate, &row.SupplierCode, &row.PaymentTermCode, &row.CurrencyCode,
		&row.ShippingAddressCode, &row.Notes, &row.TotalGross, &row.TotalNet, &row.TotalDiscount,
		&row.IsActive, &row.IsFirm, &row.CreatedAt, &row.UpdatedAt, &row.CreatedBy,
		&row.PriceTableCode, &row.InvoiceTypeCode, &row.FinancialAccount, &row.RequestTypeCode, &row.CurrencyDate,
		&row.FreightType, &row.FreightValueType, &row.FreightValueMode, &row.FreightValue, &row.CarrierCode,
		&row.RedispatchCarrierCode, &row.RedispatchFreightType, &row.RedispatchFreightValue,
		&row.AdvanceDate, &row.AdvanceValue, &row.IncotermCode, &row.ShipmentDate, &row.TalaoNumber, &row.AlcadaStatus,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("purchase order %d not found", code)
		}
		return nil, fmt.Errorf("fetching purchase order: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *PurchaseOrderRepositorySQLC) List(ctx context.Context) ([]*entity.PurchaseOrder, error) {
	rows, err := r.db.Query(ctx,
		`SELECT code, order_number, enterprise_code, status, origin, emission_date,
			delivery_date, supplier_code, payment_term_code, currency_code,
			shipping_address_code, notes, total_gross, total_net, total_discount,
			is_active, is_firm, created_at, updated_at, created_by
		FROM public.purchase_orders WHERE is_active = true ORDER BY code DESC`)
	if err != nil {
		return nil, fmt.Errorf("listing purchase orders: %w", err)
	}
	defer rows.Close()

	var result []*entity.PurchaseOrder
	for rows.Next() {
		var row purchaseOrderRow
		if err := rows.Scan(
			&row.Code, &row.OrderNumber, &row.EnterpriseCode, &row.Status, &row.Origin, &row.EmissionDate,
			&row.DeliveryDate, &row.SupplierCode, &row.PaymentTermCode, &row.CurrencyCode,
			&row.ShippingAddressCode, &row.Notes, &row.TotalGross, &row.TotalNet, &row.TotalDiscount,
			&row.IsActive, &row.IsFirm, &row.CreatedAt, &row.UpdatedAt, &row.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scanning purchase order: %w", err)
		}
		result = append(result, rowToEntity(row))
	}
	return result, rows.Err()
}

func (r *PurchaseOrderRepositorySQLC) Cancel(ctx context.Context, code int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE public.purchase_orders SET status = 'CANCELLED', is_active = false, updated_at = NOW()
		 WHERE code = $1`, code)
	if err != nil {
		return fmt.Errorf("cancelling purchase order %d: %w", code, err)
	}
	return nil
}

func (r *PurchaseOrderRepositorySQLC) ListBySupplier(ctx context.Context, supplierCode int64) ([]*entity.PurchaseOrder, error) {
	rows, err := r.db.Query(ctx,
		`SELECT code, order_number, enterprise_code, status, origin, emission_date,
			delivery_date, supplier_code, payment_term_code, currency_code,
			shipping_address_code, notes, total_gross, total_net, total_discount,
			is_active, is_firm, created_at, updated_at, created_by
		FROM public.purchase_orders WHERE supplier_code = $1 AND is_active = true ORDER BY code DESC`,
		supplierCode)
	if err != nil {
		return nil, fmt.Errorf("listing purchase orders by supplier: %w", err)
	}
	defer rows.Close()

	var result []*entity.PurchaseOrder
	for rows.Next() {
		var row purchaseOrderRow
		if err := rows.Scan(
			&row.Code, &row.OrderNumber, &row.EnterpriseCode, &row.Status, &row.Origin, &row.EmissionDate,
			&row.DeliveryDate, &row.SupplierCode, &row.PaymentTermCode, &row.CurrencyCode,
			&row.ShippingAddressCode, &row.Notes, &row.TotalGross, &row.TotalNet, &row.TotalDiscount,
			&row.IsActive, &row.IsFirm, &row.CreatedAt, &row.UpdatedAt, &row.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scanning purchase order: %w", err)
		}
		result = append(result, rowToEntity(row))
	}
	return result, rows.Err()
}

func (r *PurchaseOrderRepositorySQLC) ListByStatus(ctx context.Context, status entity.PurchaseOrderStatus) ([]*entity.PurchaseOrder, error) {
	rows, err := r.db.Query(ctx,
		`SELECT code, order_number, enterprise_code, status, origin, emission_date,
			delivery_date, supplier_code, payment_term_code, currency_code,
			shipping_address_code, notes, total_gross, total_net, total_discount,
			is_active, is_firm, created_at, updated_at, created_by
		FROM public.purchase_orders WHERE status = $1 AND is_active = true ORDER BY code DESC`,
		string(status))
	if err != nil {
		return nil, fmt.Errorf("listing purchase orders by status: %w", err)
	}
	defer rows.Close()

	var result []*entity.PurchaseOrder
	for rows.Next() {
		var row purchaseOrderRow
		if err := rows.Scan(
			&row.Code, &row.OrderNumber, &row.EnterpriseCode, &row.Status, &row.Origin, &row.EmissionDate,
			&row.DeliveryDate, &row.SupplierCode, &row.PaymentTermCode, &row.CurrencyCode,
			&row.ShippingAddressCode, &row.Notes, &row.TotalGross, &row.TotalNet, &row.TotalDiscount,
			&row.IsActive, &row.IsFirm, &row.CreatedAt, &row.UpdatedAt, &row.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("scanning purchase order: %w", err)
		}
		result = append(result, rowToEntity(row))
	}
	return result, rows.Err()
}

func (r *PurchaseOrderRepositorySQLC) CreateItem(ctx context.Context, item *entity.PurchaseOrderItem) (*entity.PurchaseOrderItem, error) {
	return insertItem(ctx, r.db, item)
}

func insertItem(ctx context.Context, q pgQuerier, item *entity.PurchaseOrderItem) (*entity.PurchaseOrderItem, error) {
	var deliveryDate pgtype.Date
	if item.DeliveryDate != nil {
		deliveryDate = pgtype.Date{Time: *item.DeliveryDate, Valid: true}
	}

	var notes pgtype.Text
	if item.Notes != nil {
		notes = pgtype.Text{String: *item.Notes, Valid: true}
	}

	var promisedDate pgtype.Date
	if item.PromisedDate != nil {
		promisedDate = pgtype.Date{Time: *item.PromisedDate, Valid: true}
	}

	var result purchaseOrderItemRow
	err := q.QueryRow(ctx,
		`INSERT INTO public.purchase_order_items (
			purchase_order_code, sequence, item_code, mask, requested_qty,
			received_qty, cancelled_qty, unit_price, total_price, discount_pct,
			ipi_pct, icms_pct, status, delivery_date, notes,
			icms_st_pct, promised_date, purchase_uom, internal_uom, internal_qty, internal_price,
			tolerance_pct, cancelled_tolerance_qty, operation_type_code, invoice_type_code,
			accounting_account, cost_center_code, requester_employee_code, contract_code,
			quotation_code, utilization_type, fiscal_classification_code
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,
			$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32)
		RETURNING code, purchase_order_code, sequence, item_code, mask,
			requested_qty, received_qty, cancelled_qty, unit_price, total_price,
			discount_pct, ipi_pct, icms_pct, status, delivery_date, notes,
			is_active, created_at, updated_at,
			icms_st_pct, promised_date, purchase_uom, internal_uom, internal_qty, internal_price,
			tolerance_pct, cancelled_tolerance_qty, operation_type_code, invoice_type_code,
			accounting_account, cost_center_code, requester_employee_code, contract_code,
			quotation_code, utilization_type, fiscal_classification_code`,
		item.PurchaseOrderCode, item.Sequence, item.ItemCode, item.Mask,
		item.RequestedQty, item.ReceivedQty, item.CancelledQty,
		item.UnitPrice, item.TotalPrice, item.DiscountPct,
		item.IPIPct, item.ICMSPct, string(item.Status), deliveryDate, notes,
		item.ICMSSTPct, promisedDate, pgutil.ToPgTextFromPtr(item.PurchaseUOM), pgutil.ToPgTextFromPtr(item.InternalUOM), item.InternalQty, item.InternalPrice,
		item.TolerancePct, item.CancelledToleranceQty, item.OperationTypeCode, item.InvoiceTypeCode,
		pgutil.ToPgTextFromPtr(item.AccountingAccount), item.CostCenterCode, item.RequesterEmployeeCode, item.ContractCode,
		item.QuotationCode, pgutil.ToPgTextFromPtr(item.UtilizationType), item.FiscalClassificationCode,
	).Scan(
		&result.Code, &result.PurchaseOrderCode, &result.Sequence, &result.ItemCode, &result.Mask,
		&result.RequestedQty, &result.ReceivedQty, &result.CancelledQty,
		&result.UnitPrice, &result.TotalPrice, &result.DiscountPct,
		&result.IPIPct, &result.ICMSPct, &result.Status, &result.DeliveryDate,
		&result.Notes, &result.IsActive, &result.CreatedAt, &result.UpdatedAt,
		&result.ICMSSTPct, &result.PromisedDate, &result.PurchaseUOM, &result.InternalUOM, &result.InternalQty, &result.InternalPrice,
		&result.TolerancePct, &result.CancelledToleranceQty, &result.OperationTypeCode, &result.InvoiceTypeCode,
		&result.AccountingAccount, &result.CostCenterCode, &result.RequesterEmployeeCode, &result.ContractCode,
		&result.QuotationCode, &result.UtilizationType, &result.FiscalClassificationCode,
	)
	if err != nil {
		return nil, fmt.Errorf("creating purchase order item: %w", err)
	}
	return rowItemToEntity(result), nil
}

func (r *PurchaseOrderRepositorySQLC) UpdateItem(ctx context.Context, item *entity.PurchaseOrderItem) (*entity.PurchaseOrderItem, error) {
	var deliveryDate pgtype.Date
	if item.DeliveryDate != nil {
		deliveryDate = pgtype.Date{Time: *item.DeliveryDate, Valid: true}
	}

	var notes pgtype.Text
	if item.Notes != nil {
		notes = pgtype.Text{String: *item.Notes, Valid: true}
	}

	var result purchaseOrderItemRow
	err := r.db.QueryRow(ctx,
		`UPDATE public.purchase_order_items SET
			sequence = $2, item_code = $3, mask = $4, requested_qty = $5,
			received_qty = $6, cancelled_qty = $7, unit_price = $8, total_price = $9,
			discount_pct = $10, ipi_pct = $11, icms_pct = $12, status = $13,
			delivery_date = $14, notes = $15, updated_at = NOW()
		WHERE code = $1 AND is_active = true
		RETURNING code, purchase_order_code, sequence, item_code, mask,
			requested_qty, received_qty, cancelled_qty, unit_price, total_price,
			discount_pct, ipi_pct, icms_pct, status, delivery_date, notes,
			is_active, created_at, updated_at`,
		item.Code, item.Sequence, item.ItemCode, item.Mask,
		item.RequestedQty, item.ReceivedQty, item.CancelledQty,
		item.UnitPrice, item.TotalPrice, item.DiscountPct,
		item.IPIPct, item.ICMSPct, string(item.Status), deliveryDate, notes,
	).Scan(
		&result.Code, &result.PurchaseOrderCode, &result.Sequence, &result.ItemCode, &result.Mask,
		&result.RequestedQty, &result.ReceivedQty, &result.CancelledQty,
		&result.UnitPrice, &result.TotalPrice, &result.DiscountPct,
		&result.IPIPct, &result.ICMSPct, &result.Status, &result.DeliveryDate,
		&result.Notes, &result.IsActive, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("purchase order item %d not found", item.Code)
		}
		return nil, fmt.Errorf("updating purchase order item: %w", err)
	}
	return rowItemToEntity(result), nil
}

func (r *PurchaseOrderRepositorySQLC) ListItems(ctx context.Context, purchaseOrderCode int64) ([]*entity.PurchaseOrderItem, error) {
	rows, err := r.db.Query(ctx,
		`SELECT code, purchase_order_code, sequence, item_code, mask,
			requested_qty, received_qty, cancelled_qty, unit_price, total_price,
			discount_pct, ipi_pct, icms_pct, status, delivery_date, notes,
			is_active, created_at, updated_at,
			icms_st_pct, promised_date, purchase_uom, internal_uom, internal_qty, internal_price,
			tolerance_pct, cancelled_tolerance_qty, operation_type_code, invoice_type_code,
			accounting_account, cost_center_code, requester_employee_code, contract_code,
			quotation_code, utilization_type, fiscal_classification_code
		FROM public.purchase_order_items
		WHERE purchase_order_code = $1 AND is_active = true ORDER BY sequence`,
		purchaseOrderCode)
	if err != nil {
		return nil, fmt.Errorf("listing purchase order items: %w", err)
	}
	defer rows.Close()

	var result []*entity.PurchaseOrderItem
	for rows.Next() {
		var row purchaseOrderItemRow
		if err := rows.Scan(
			&row.Code, &row.PurchaseOrderCode, &row.Sequence, &row.ItemCode, &row.Mask,
			&row.RequestedQty, &row.ReceivedQty, &row.CancelledQty,
			&row.UnitPrice, &row.TotalPrice, &row.DiscountPct,
			&row.IPIPct, &row.ICMSPct, &row.Status, &row.DeliveryDate,
			&row.Notes, &row.IsActive, &row.CreatedAt, &row.UpdatedAt,
			&row.ICMSSTPct, &row.PromisedDate, &row.PurchaseUOM, &row.InternalUOM, &row.InternalQty, &row.InternalPrice,
			&row.TolerancePct, &row.CancelledToleranceQty, &row.OperationTypeCode, &row.InvoiceTypeCode,
			&row.AccountingAccount, &row.CostCenterCode, &row.RequesterEmployeeCode, &row.ContractCode,
			&row.QuotationCode, &row.UtilizationType, &row.FiscalClassificationCode,
		); err != nil {
			return nil, fmt.Errorf("scanning purchase order item: %w", err)
		}
		result = append(result, rowItemToEntity(row))
	}
	return result, rows.Err()
}

func (r *PurchaseOrderRepositorySQLC) CancelItem(ctx context.Context, itemCode int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE public.purchase_order_items SET status = 'CANCELLED', is_active = false, updated_at = NOW()
		 WHERE code = $1`, itemCode)
	if err != nil {
		return fmt.Errorf("cancelling purchase order item %d: %w", itemCode, err)
	}
	return nil
}

type purchaseOrderRow struct {
	Code               int64
	OrderNumber        int64
	EnterpriseCode     int64
	Status             string
	Origin             string
	EmissionDate       time.Time
	DeliveryDate       pgtype.Date
	SupplierCode       *int64
	PaymentTermCode    *int64
	CurrencyCode       string
	ShippingAddressCode *int64
	Notes              pgtype.Text
	TotalGross         float64
	TotalNet           float64
	TotalDiscount      float64
	IsActive           bool
	IsFirm             bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
	CreatedBy          [16]byte
	// extended (migration 000140)
	PriceTableCode         *int64
	InvoiceTypeCode        *int64
	FinancialAccount       pgtype.Text
	RequestTypeCode        *int64
	CurrencyDate           pgtype.Date
	FreightType            string
	FreightValueType       pgtype.Text
	FreightValueMode       pgtype.Text
	FreightValue           float64
	CarrierCode            *int64
	RedispatchCarrierCode  *int64
	RedispatchFreightType  pgtype.Text
	RedispatchFreightValue float64
	AdvanceDate            pgtype.Date
	AdvanceValue           float64
	IncotermCode           pgtype.Text
	ShipmentDate           pgtype.Date
	TalaoNumber            pgtype.Text
	AlcadaStatus           string
}

type purchaseOrderItemRow struct {
	Code              int64
	PurchaseOrderCode int64
	Sequence          int32
	ItemCode          int64
	Mask              string
	RequestedQty      float64
	ReceivedQty       float64
	CancelledQty      float64
	UnitPrice         float64
	TotalPrice        float64
	DiscountPct       float64
	IPIPct            float64
	ICMSPct           float64
	Status            string
	DeliveryDate      pgtype.Date
	Notes             pgtype.Text
	IsActive          bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
	// extended (migration 000140)
	ICMSSTPct                float64
	PromisedDate             pgtype.Date
	PurchaseUOM              pgtype.Text
	InternalUOM              pgtype.Text
	InternalQty              float64
	InternalPrice            float64
	TolerancePct             float64
	CancelledToleranceQty    float64
	OperationTypeCode        *int64
	InvoiceTypeCode          *int64
	AccountingAccount        pgtype.Text
	CostCenterCode           *int64
	RequesterEmployeeCode    *int64
	ContractCode             *int64
	QuotationCode            *int64
	UtilizationType          pgtype.Text
	FiscalClassificationCode *int64
}

func rowToEntity(row purchaseOrderRow) *entity.PurchaseOrder {
	e := &entity.PurchaseOrder{
		Code:               row.Code,
		OrderNumber:        row.OrderNumber,
		EnterpriseCode:     row.EnterpriseCode,
		Status:             entity.PurchaseOrderStatus(row.Status),
		Origin:             entity.PurchaseOrderOrigin(row.Origin),
		EmissionDate:       row.EmissionDate,
		SupplierCode:       row.SupplierCode,
		PaymentTermCode:    row.PaymentTermCode,
		CurrencyCode:       row.CurrencyCode,
		ShippingAddressCode: row.ShippingAddressCode,
		TotalGross:         row.TotalGross,
		TotalNet:           row.TotalNet,
		TotalDiscount:      row.TotalDiscount,
		IsActive:           row.IsActive,
		IsFirm:             row.IsFirm,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}

	if row.DeliveryDate.Valid {
		e.DeliveryDate = &row.DeliveryDate.Time
	}
	if row.Notes.Valid {
		e.Notes = &row.Notes.String
	}

	// extended fields
	e.PriceTableCode = row.PriceTableCode
	e.InvoiceTypeCode = row.InvoiceTypeCode
	e.FinancialAccount = pgutil.FromPgTextPtr(row.FinancialAccount)
	e.RequestTypeCode = row.RequestTypeCode
	e.CurrencyDate = pgutil.FromPgDateToPtr(row.CurrencyDate)
	e.FreightType = row.FreightType
	e.FreightValueType = pgutil.FromPgTextPtr(row.FreightValueType)
	e.FreightValueMode = pgutil.FromPgTextPtr(row.FreightValueMode)
	e.FreightValue = row.FreightValue
	e.CarrierCode = row.CarrierCode
	e.RedispatchCarrierCode = row.RedispatchCarrierCode
	e.RedispatchFreightType = pgutil.FromPgTextPtr(row.RedispatchFreightType)
	e.RedispatchFreightValue = row.RedispatchFreightValue
	e.AdvanceDate = pgutil.FromPgDateToPtr(row.AdvanceDate)
	e.AdvanceValue = row.AdvanceValue
	e.IncotermCode = pgutil.FromPgTextPtr(row.IncotermCode)
	e.ShipmentDate = pgutil.FromPgDateToPtr(row.ShipmentDate)
	e.TalaoNumber = pgutil.FromPgTextPtr(row.TalaoNumber)
	e.AlcadaStatus = row.AlcadaStatus

	copy(e.CreatedBy[:], row.CreatedBy[:])

	return e
}

func rowItemToEntity(row purchaseOrderItemRow) *entity.PurchaseOrderItem {
	item := &entity.PurchaseOrderItem{
		Code:              row.Code,
		PurchaseOrderCode: row.PurchaseOrderCode,
		Sequence:          int(row.Sequence),
		ItemCode:          row.ItemCode,
		Mask:              row.Mask,
		RequestedQty:      row.RequestedQty,
		ReceivedQty:       row.ReceivedQty,
		CancelledQty:      row.CancelledQty,
		UnitPrice:         row.UnitPrice,
		TotalPrice:        row.TotalPrice,
		DiscountPct:       row.DiscountPct,
		IPIPct:            row.IPIPct,
		ICMSPct:           row.ICMSPct,
		Status:            entity.PurchaseOrderItemStatus(row.Status),
		IsActive:          row.IsActive,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}

	if row.DeliveryDate.Valid {
		item.DeliveryDate = &row.DeliveryDate.Time
	}
	if row.Notes.Valid {
		item.Notes = &row.Notes.String
	}

	// extended fields
	item.ICMSSTPct = row.ICMSSTPct
	item.PromisedDate = pgutil.FromPgDateToPtr(row.PromisedDate)
	item.PurchaseUOM = pgutil.FromPgTextPtr(row.PurchaseUOM)
	item.InternalUOM = pgutil.FromPgTextPtr(row.InternalUOM)
	item.InternalQty = row.InternalQty
	item.InternalPrice = row.InternalPrice
	item.TolerancePct = row.TolerancePct
	item.CancelledToleranceQty = row.CancelledToleranceQty
	item.OperationTypeCode = row.OperationTypeCode
	item.InvoiceTypeCode = row.InvoiceTypeCode
	item.AccountingAccount = pgutil.FromPgTextPtr(row.AccountingAccount)
	item.CostCenterCode = row.CostCenterCode
	item.RequesterEmployeeCode = row.RequesterEmployeeCode
	item.ContractCode = row.ContractCode
	item.QuotationCode = row.QuotationCode
	item.UtilizationType = pgutil.FromPgTextPtr(row.UtilizationType)
	item.FiscalClassificationCode = row.FiscalClassificationCode

	return item
}
