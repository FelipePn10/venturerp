package sales_quotation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	quoteentity "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	quoterepo "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/tenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) NextQuotationNumber(ctx context.Context, enterpriseCode int64) (int64, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return 0, err
	}
	if enterpriseCode != tenantID {
		return 0, fmt.Errorf("sales quotation enterprise does not match authenticated tenant")
	}
	var n int64
	err = r.pool.QueryRow(ctx, `
INSERT INTO public.sales_quotation_sequences (enterprise_code, last_number)
VALUES ($1, 1)
ON CONFLICT (enterprise_code)
DO UPDATE SET last_number = sales_quotation_sequences.last_number + 1
RETURNING last_number`, enterpriseCode).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("next sales quotation number: %w", err)
	}
	return n, nil
}

func (r *Repository) Create(ctx context.Context, q *quoteentity.SalesQuotation) (*quoteentity.SalesQuotation, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if q.EnterpriseCode != tenantID {
		return nil, fmt.Errorf("sales quotation enterprise does not match authenticated tenant")
	}
	row := r.pool.QueryRow(ctx, `
INSERT INTO public.sales_quotations (
 quotation_number, enterprise_code, status, quotation_type, emission_date, digit_date,
 valid_until, delivery_date, delivery_date_firm, purchase_order_number,
 customer_code, billing_address_code, shipping_address_code,
 representative_code, sales_division_code, price_table_code, payment_term_code,
 currency_code, probability_pct, commission_pct, is_nfce, street, street_number,
 foreign_document, release_status, commercial_blocked, commercial_block_reason,
 carrier_code, freight_type, verify_freight, freight_value, redelivery_freight_value,
 insurance_value, discount_value, surcharge_value, retained_tax_value,
 delivery_authorization, notes, obs_customer, created_by, delivery_with_receipt, consumer_address
) VALUES (
 $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
 $11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
 $21,$22,$23,$24,$25,$26,$27,$28,$29,$30,
 $31,$32,$33,$34,$35,$36,$37,$38,$39,$40,$41,$42
) RETURNING `+quotationColumns,
		q.QuotationNumber, q.EnterpriseCode, string(q.Status), string(q.QuotationType), q.EmissionDate, q.DigitDate,
		q.ValidUntil, q.DeliveryDate, q.DeliveryDateFirm, q.PurchaseOrderNumber,
		q.CustomerCode, q.BillingAddressCode, q.ShippingAddressCode,
		q.RepresentativeCode, q.SalesDivisionCode, q.PriceTableCode, q.PaymentTermCode,
		q.CurrencyCode, q.ProbabilityPct, q.CommissionPct, q.IsNFCe, q.Street, q.StreetNumber,
		q.ForeignDocument, string(q.ReleaseStatus), q.CommercialBlocked, q.CommercialBlockReason,
		q.CarrierCode, q.FreightType, q.VerifyFreight, q.FreightValue, q.RedeliveryFreightValue,
		q.InsuranceValue, q.DiscountValue, q.SurchargeValue, q.RetainedTaxValue,
		q.DeliveryAuthorization, q.Notes, q.ObsCustomer, q.CreatedBy, q.DeliveryWithReceipt, q.ConsumerAddress,
	)
	return scanQuotation(row)
}

func (r *Repository) Update(ctx context.Context, q *quoteentity.SalesQuotation) (*quoteentity.SalesQuotation, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `
UPDATE public.sales_quotations SET
 status=$1, quotation_type=$2, valid_until=$3, delivery_date=$4, delivery_date_firm=$5,
 purchase_order_number=$6, customer_code=$7, billing_address_code=$8, shipping_address_code=$9,
 representative_code=$10, sales_division_code=$11, price_table_code=$12,
 payment_term_code=$13, currency_code=$14, probability_pct=$15, commission_pct=$16,
 is_nfce=$17, street=$18, street_number=$19, foreign_document=$20,
 release_status=$21, commercial_blocked=$22, commercial_block_reason=$23,
 carrier_code=$24, freight_type=$25, verify_freight=$26, freight_value=$27,
 redelivery_freight_value=$28, insurance_value=$29, discount_value=$30,
 surcharge_value=$31, retained_tax_value=$32, delivery_authorization=$33,
 notes=$34, obs_customer=$35, delivery_with_receipt=$36, consumer_address=$37, updated_at=NOW()
WHERE code=$38 AND enterprise_code=$39 AND is_active=TRUE
RETURNING `+quotationColumns,
		string(q.Status), string(q.QuotationType), q.ValidUntil, q.DeliveryDate, q.DeliveryDateFirm,
		q.PurchaseOrderNumber, q.CustomerCode, q.BillingAddressCode, q.ShippingAddressCode,
		q.RepresentativeCode, q.SalesDivisionCode, q.PriceTableCode,
		q.PaymentTermCode, q.CurrencyCode, q.ProbabilityPct, q.CommissionPct,
		q.IsNFCe, q.Street, q.StreetNumber, q.ForeignDocument,
		string(q.ReleaseStatus), q.CommercialBlocked, q.CommercialBlockReason,
		q.CarrierCode, q.FreightType, q.VerifyFreight, q.FreightValue,
		q.RedeliveryFreightValue, q.InsuranceValue, q.DiscountValue, q.SurchargeValue,
		q.RetainedTaxValue, q.DeliveryAuthorization, q.Notes, q.ObsCustomer, q.DeliveryWithReceipt, q.ConsumerAddress, q.Code, tenantID,
	)
	return scanQuotation(row)
}

func (r *Repository) GetByCode(ctx context.Context, code int64) (*quoteentity.SalesQuotation, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `SELECT `+quotationColumns+` FROM public.sales_quotations WHERE code=$1 AND enterprise_code=$2`, code, tenantID)
	q, err := scanQuotation(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("sales quotation %d not found", code)
		}
		return nil, err
	}
	return q, nil
}

func (r *Repository) List(ctx context.Context, filter quoterepo.SalesQuotationFilter) ([]*quoteentity.SalesQuotation, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	sql := `SELECT ` + quotationColumns + ` FROM public.sales_quotations WHERE enterprise_code=$1`
	args := []any{tenantID}
	add := func(clause string, arg any) {
		args = append(args, arg)
		sql += fmt.Sprintf(" AND "+clause, len(args))
	}
	if filter.QuotationNumber != nil {
		add("quotation_number=$%d", *filter.QuotationNumber)
	}
	if filter.CustomerCode != nil {
		add("customer_code=$%d", *filter.CustomerCode)
	}
	if filter.SalesDivisionCode != nil {
		add("sales_division_code=$%d", *filter.SalesDivisionCode)
	}
	if filter.QuotationType != nil {
		add("quotation_type=$%d", string(*filter.QuotationType))
	}
	if filter.Status != nil {
		add("status=$%d", string(*filter.Status))
	}
	if filter.From != nil {
		add("emission_date >= $%d", *filter.From)
	}
	if filter.To != nil {
		add("emission_date <= $%d", *filter.To)
	}
	if filter.PurchaseOrderNumber != nil {
		add("purchase_order_number ILIKE $%d", "%"+*filter.PurchaseOrderNumber+"%")
	}
	if filter.FreightType != nil {
		add("freight_type=$%d", *filter.FreightType)
	}
	sql += " ORDER BY emission_date DESC, quotation_number DESC"
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	addLimit := len(args) + 1
	addOffset := len(args) + 2
	sql += fmt.Sprintf(" LIMIT $%d OFFSET $%d", addLimit, addOffset)
	args = append(args, limit, max(filter.Offset, 0))
	rows, err := r.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("listing sales quotations: %w", err)
	}
	defer rows.Close()
	out := []*quoteentity.SalesQuotation{}
	for rows.Next() {
		q, err := scanQuotation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

func (r *Repository) Cancel(ctx context.Context, code, reasonCode int64, reason string, complement *string) error {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.withTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE public.sales_quotations SET status='CANCELLED', cancellation_reason_code=$3, cancel_reason=$4, cancel_complement=$5, updated_at=NOW() WHERE code=$1 AND enterprise_code=$2 AND status NOT IN ('CANCELLED','ATTENDED')`, code, tenantID, reasonCode, reason, complement)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("sales quotation %d not found or cannot be cancelled", code)
		}
		_, err = tx.Exec(ctx, `INSERT INTO public.sales_quotation_events (sales_quotation_code, event_type, reason, complement) VALUES ($1,'CANCEL',$2,$3)`, code, reason, complement)
		return err
	})
}

func (r *Repository) Uncancel(ctx context.Context, code, reasonCode int64, reason string, complement *string) error {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.withTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE public.sales_quotations SET status='OV', is_active=TRUE, cancellation_reason_code=NULL, cancel_reason=NULL, cancel_complement=NULL, updated_at=NOW() WHERE code=$1 AND enterprise_code=$2 AND status='CANCELLED' AND cancellation_reason_code=$3`, code, tenantID, reasonCode)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("sales quotation %d not found or is not cancelled", code)
		}
		_, err = tx.Exec(ctx, `INSERT INTO public.sales_quotation_events (sales_quotation_code, event_type, reason, complement) VALUES ($1,'UNCANCEL',$2,$3)`, code, reason, complement)
		return err
	})
}

func (r *Repository) Attend(ctx context.Context, code int64, reason string, complement *string, eventDate time.Time) error {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.withTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE public.sales_quotations SET status='ATTENDED', attended_reason=$3, attended_at=$4, updated_at=NOW() WHERE code=$1 AND enterprise_code=$2 AND status NOT IN ('CANCELLED','ATTENDED','EXPIRED')`, code, tenantID, reason, eventDate)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("sales quotation %d not found or cannot be attended", code)
		}
		_, err = tx.Exec(ctx, `INSERT INTO public.sales_quotation_events (sales_quotation_code, event_type, reason, complement, event_date) VALUES ($1,'ATTEND',$2,$3,$4)`, code, reason, complement, eventDate)
		return err
	})
}

func (r *Repository) ChangeStatus(ctx context.Context, code int64, status quoteentity.SalesQuotationStatus) error {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `UPDATE public.sales_quotations SET status=$3, updated_at=NOW() WHERE code=$1 AND enterprise_code=$2`, code, tenantID, string(status))
	if err == nil && tag.RowsAffected() == 0 {
		return fmt.Errorf("sales quotation %d not found", code)
	}
	return err
}

func (r *Repository) ChangeRelease(ctx context.Context, code int64, status quoteentity.SalesQuotationReleaseStatus, reason string) error {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.withTx(ctx, func(tx pgx.Tx) error {
		var previous string
		err := tx.QueryRow(ctx, `SELECT release_status FROM public.sales_quotations WHERE code=$1 AND enterprise_code=$2 FOR UPDATE`, code, tenantID).Scan(&previous)
		if err != nil {
			return err
		}
		if previous == string(status) {
			return nil
		}
		if _, err = tx.Exec(ctx, `UPDATE public.sales_quotations SET release_status=$3::text,commercial_blocked=($3::text='BLOCKED'),commercial_block_reason=CASE WHEN $3::text='BLOCKED' THEN $4 ELSE NULL END,updated_at=NOW() WHERE code=$1 AND enterprise_code=$2`, code, tenantID, string(status), reason); err != nil {
			return err
		}
		eventType := "RELEASE"
		switch status {
		case quoteentity.SalesQuotationReleaseBlocked:
			eventType = "BLOCK"
		case quoteentity.SalesQuotationReleaseManual:
			eventType = "MANUAL_RELEASE"
		}
		_, err = tx.Exec(ctx, `INSERT INTO public.sales_quotation_events(sales_quotation_code,event_type,reason) VALUES($1,$2,$3)`, code, eventType, reason)
		return err
	})
}

func (r *Repository) ListEvents(ctx context.Context, code int64) ([]*quoteentity.Event, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT e.id,e.sales_quotation_code,e.sales_quotation_item_code,e.event_type,e.reason,e.complement,e.event_date,e.created_at,e.created_by FROM public.sales_quotation_events e JOIN public.sales_quotations q ON q.code=e.sales_quotation_code WHERE e.sales_quotation_code=$1 AND q.enterprise_code=$2 ORDER BY e.event_date DESC,e.created_at DESC,e.id DESC`, code, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*quoteentity.Event
	for rows.Next() {
		e := new(quoteentity.Event)
		if err := rows.Scan(&e.ID, &e.SalesQuotationCode, &e.SalesQuotationItemCode, &e.EventType, &e.Reason, &e.Complement, &e.EventDate, &e.CreatedAt, &e.CreatedBy); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) MarkConverted(ctx context.Context, quotationCode, salesOrderCode int64) error {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.withTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE public.sales_quotations SET status='ATTENDED', attended_reason='Convertido em pedido de venda', attended_at=NOW(), converted_sales_order_code=$3, converted_at=NOW(), updated_at=NOW() WHERE code=$1 AND enterprise_code=$2 AND converted_sales_order_code IS NULL`, quotationCode, tenantID, salesOrderCode)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("sales quotation %d not found or already converted", quotationCode)
		}
		_, err = tx.Exec(ctx, `INSERT INTO public.sales_quotation_events (sales_quotation_code, event_type, reason) VALUES ($1,'CONVERT','Convertido em pedido de venda')`, quotationCode)
		return err
	})
}

func (r *Repository) Report(ctx context.Context, filter quoterepo.SalesQuotationFilter) (*quoterepo.SalesQuotationReport, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	sql := `SELECT COUNT(*), COALESCE(SUM(total_gross),0), COALESCE(SUM(total_net),0),
COUNT(*) FILTER (WHERE status IN ('R','A','OA','OF','OV')),
COUNT(*) FILTER (WHERE status IN ('OF','OV')),
COUNT(*) FILTER (WHERE converted_sales_order_code IS NOT NULL OR status='ATTENDED'),
COUNT(*) FILTER (WHERE status='CANCELLED'),
COUNT(*) FILTER (WHERE status='EXPIRED'),
COALESCE(SUM(total_net * probability_pct / 100),0),
COALESCE(SUM(retained_tax_value),0)
FROM public.sales_quotations WHERE enterprise_code=$1`
	args := []any{tenantID}
	add := func(clause string, arg any) {
		args = append(args, arg)
		sql += fmt.Sprintf(" AND "+clause, len(args))
	}
	if filter.QuotationNumber != nil {
		add("quotation_number=$%d", *filter.QuotationNumber)
	}
	if filter.CustomerCode != nil {
		add("customer_code=$%d", *filter.CustomerCode)
	}
	if filter.SalesDivisionCode != nil {
		add("sales_division_code=$%d", *filter.SalesDivisionCode)
	}
	if filter.QuotationType != nil {
		add("quotation_type=$%d", string(*filter.QuotationType))
	}
	if filter.Status != nil {
		add("status=$%d", string(*filter.Status))
	}
	if filter.From != nil {
		add("emission_date >= $%d", *filter.From)
	}
	if filter.To != nil {
		add("emission_date <= $%d", *filter.To)
	}
	if filter.PurchaseOrderNumber != nil {
		add("purchase_order_number ILIKE $%d", "%"+*filter.PurchaseOrderNumber+"%")
	}
	if filter.FreightType != nil {
		add("freight_type=$%d", *filter.FreightType)
	}
	var report quoterepo.SalesQuotationReport
	var gross, net, weighted, retained pgtype.Numeric
	err = r.pool.QueryRow(ctx, sql, args...).Scan(&report.TotalQuotations, &gross, &net, &report.OpenCount, &report.ApprovedCount, &report.ConvertedCount, &report.CancelledCount, &report.ExpiredCount, &weighted, &retained)
	report.TotalGross = numericToDecimal(gross)
	report.TotalNet = numericToDecimal(net)
	report.WeightedNet = numericToDecimal(weighted)
	report.RetainedTax = numericToDecimal(retained)
	return &report, err
}

func (r *Repository) CreateItem(ctx context.Context, item *quoteentity.SalesQuotationItem) (*quoteentity.SalesQuotationItem, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `
INSERT INTO public.sales_quotation_items (
 sales_quotation_code, sequence, item_code, mask, sales_uom, warehouse_code,
 price_table_code, requested_qty, unit_price, attended_qty, cancelled_qty,
 delivery_date, delivery_date_firm, discount_pct, ipi_pct, st_pct,
 total_gross, total_net, total_net_with_ipi, status, notes
) SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,0,0,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19
WHERE EXISTS (SELECT 1 FROM public.sales_quotations q WHERE q.code=$1 AND q.enterprise_code=$20 AND q.is_active)
RETURNING `+itemColumns,
		item.SalesQuotationCode, item.Sequence, item.ItemCode, item.Mask, item.SalesUOM, item.WarehouseCode,
		item.PriceTableCode, item.RequestedQty, item.UnitPrice, item.DeliveryDate, item.DeliveryDateFirm,
		item.DiscountPct, item.IPIPct, item.STPct, item.TotalGross, item.TotalNet, item.TotalNetWithIPI,
		string(item.Status), item.Notes, tenantID,
	)
	return scanItem(row)
}

func (r *Repository) UpdateItem(ctx context.Context, item *quoteentity.SalesQuotationItem) (*quoteentity.SalesQuotationItem, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `
UPDATE public.sales_quotation_items SET
 requested_qty=$1, unit_price=$2, attended_qty=$3, cancelled_qty=$4,
 delivery_date=$5, delivery_date_firm=$6, discount_pct=$7, ipi_pct=$8,
 st_pct=$9, total_gross=$10, total_net=$11, total_net_with_ipi=$12,
 status=$13, notes=$14, updated_at=NOW()
WHERE code=$15 AND is_active=TRUE AND EXISTS (
 SELECT 1 FROM public.sales_quotations q
 WHERE q.code=public.sales_quotation_items.sales_quotation_code AND q.enterprise_code=$16
)
RETURNING `+itemColumns,
		item.RequestedQty, item.UnitPrice, item.AttendedQty, item.CancelledQty,
		item.DeliveryDate, item.DeliveryDateFirm, item.DiscountPct, item.IPIPct,
		item.STPct, item.TotalGross, item.TotalNet, item.TotalNetWithIPI,
		string(item.Status), item.Notes, item.Code, tenantID,
	)
	return scanItem(row)
}

func (r *Repository) GetItem(ctx context.Context, itemCode int64) (*quoteentity.SalesQuotationItem, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	row := r.pool.QueryRow(ctx, `SELECT `+itemColumns+` FROM public.sales_quotation_items i WHERE i.code=$1 AND EXISTS(SELECT 1 FROM public.sales_quotations q WHERE q.code=i.sales_quotation_code AND q.enterprise_code=$2)`, itemCode, tenantID)
	return scanItem(row)
}

func (r *Repository) ListItems(ctx context.Context, quotationCode int64) ([]*quoteentity.SalesQuotationItem, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+itemColumns+` FROM public.sales_quotation_items i WHERE sales_quotation_code=$1 AND is_active=TRUE AND EXISTS (SELECT 1 FROM public.sales_quotations q WHERE q.code=i.sales_quotation_code AND q.enterprise_code=$2) ORDER BY sequence`, quotationCode, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*quoteentity.SalesQuotationItem{}
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *Repository) CancelItem(ctx context.Context, itemCode, reasonCode int64, reason string, complement *string) error {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.withTx(ctx, func(tx pgx.Tx) error {
		var quotationCode int64
		err = tx.QueryRow(ctx, `UPDATE public.sales_quotation_items i SET status='CANCELLED',cancelled_qty=requested_qty,is_active=FALSE,updated_at=NOW() WHERE code=$1 AND is_active AND EXISTS(SELECT 1 FROM public.sales_quotations q WHERE q.code=i.sales_quotation_code AND q.enterprise_code=$2 AND q.status NOT IN ('CANCELLED','ATTENDED','EXPIRED')) RETURNING sales_quotation_code`, itemCode, tenantID).Scan(&quotationCode)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO public.sales_quotation_events(sales_quotation_code,sales_quotation_item_code,event_type,reason,complement) VALUES($1,$2,'CANCEL',$3,$4)`, quotationCode, itemCode, fmt.Sprintf("[%d] %s", reasonCode, reason), complement); err != nil {
			return err
		}
		return r.recalculateTotalsTx(ctx, tx, tenantID, quotationCode)
	})
}

func (r *Repository) RecalculateTotals(ctx context.Context, quotationCode int64) error {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	return r.recalculateTotalsTx(ctx, r.pool, tenantID, quotationCode)
}

type dbExecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func (r *Repository) recalculateTotalsTx(ctx context.Context, db dbExecutor, tenantID, quotationCode int64) error {
	_, err := db.Exec(ctx, `
UPDATE public.sales_quotations q SET
 total_gross = COALESCE((SELECT SUM(total_gross) FROM public.sales_quotation_items WHERE sales_quotation_code=$1 AND is_active=TRUE),0),
 total_net = COALESCE((SELECT SUM(total_net) FROM public.sales_quotation_items WHERE sales_quotation_code=$1 AND is_active=TRUE),0)
   + q.freight_value + q.redelivery_freight_value + q.insurance_value + q.surcharge_value
   - q.discount_value - q.retained_tax_value,
 updated_at = NOW()
WHERE q.code=$1 AND q.enterprise_code=$2`, quotationCode, tenantID)
	return err
}

func (r *Repository) withTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

const quotationColumns = `code, quotation_number, enterprise_code, status, emission_date, valid_until, delivery_date,
quotation_type, digit_date, delivery_date_firm, purchase_order_number, customer_code,
billing_address_code, shipping_address_code, representative_code, sales_division_code,
price_table_code, payment_term_code, currency_code, probability_pct, commission_pct,
is_nfce, street, street_number, foreign_document, release_status, commercial_blocked,
commercial_block_reason, carrier_code, freight_type, verify_freight, freight_value,
redelivery_freight_value, insurance_value, discount_value, surcharge_value,
retained_tax_value, total_gross, total_net, delivery_authorization, notes, obs_customer,
cancel_reason, cancel_complement, attended_reason, attended_at,
converted_sales_order_code, converted_at, is_active, created_at, updated_at, created_by,
delivery_with_receipt, dav_generated_at, dav_report_key, consumer_address`

const itemColumns = `code, sales_quotation_code, sequence, item_code, mask, sales_uom, warehouse_code,
price_table_code, requested_qty, unit_price, attended_qty, cancelled_qty, delivery_date,
delivery_date_firm, discount_pct, ipi_pct, st_pct, total_gross, total_net, total_net_with_ipi,
status, notes, is_active, created_at, updated_at`

type scanner interface {
	Scan(dest ...any) error
}

func scanQuotation(s scanner) (*quoteentity.SalesQuotation, error) {
	var q quoteentity.SalesQuotation
	var validUntil, deliveryDate pgtype.Date
	var attendedAt, convertedAt, davGeneratedAt pgtype.Timestamptz
	var probability, commission, freight, redeliveryFreight, insurance, discount, surcharge, retained, gross, net pgtype.Numeric
	err := s.Scan(
		&q.Code, &q.QuotationNumber, &q.EnterpriseCode, &q.Status, &q.EmissionDate, &validUntil, &deliveryDate,
		&q.QuotationType, &q.DigitDate, &q.DeliveryDateFirm, &q.PurchaseOrderNumber, &q.CustomerCode,
		&q.BillingAddressCode, &q.ShippingAddressCode, &q.RepresentativeCode, &q.SalesDivisionCode,
		&q.PriceTableCode, &q.PaymentTermCode, &q.CurrencyCode, &probability, &commission,
		&q.IsNFCe, &q.Street, &q.StreetNumber, &q.ForeignDocument, &q.ReleaseStatus, &q.CommercialBlocked,
		&q.CommercialBlockReason, &q.CarrierCode, &q.FreightType, &q.VerifyFreight, &freight,
		&redeliveryFreight, &insurance, &discount, &surcharge, &retained, &gross, &net,
		&q.DeliveryAuthorization, &q.Notes, &q.ObsCustomer, &q.CancelReason, &q.CancelComplement,
		&q.AttendedReason, &attendedAt, &q.ConvertedSalesOrderCode, &convertedAt, &q.IsActive,
		&q.CreatedAt, &q.UpdatedAt, &q.CreatedBy, &q.DeliveryWithReceipt, &davGeneratedAt,
		&q.DAVReportKey, &q.ConsumerAddress,
	)
	if err != nil {
		return nil, err
	}
	q.ProbabilityPct = numericToDecimal(probability)
	q.CommissionPct = numericToDecimal(commission)
	q.FreightValue = numericToDecimal(freight)
	q.RedeliveryFreightValue = numericToDecimal(redeliveryFreight)
	q.InsuranceValue = numericToDecimal(insurance)
	q.DiscountValue = numericToDecimal(discount)
	q.SurchargeValue = numericToDecimal(surcharge)
	q.RetainedTaxValue = numericToDecimal(retained)
	q.TotalGross = numericToDecimal(gross)
	q.TotalNet = numericToDecimal(net)
	if validUntil.Valid {
		t := validUntil.Time
		q.ValidUntil = &t
	}
	if deliveryDate.Valid {
		t := deliveryDate.Time
		q.DeliveryDate = &t
	}
	if attendedAt.Valid {
		t := attendedAt.Time
		q.AttendedAt = &t
	}
	if convertedAt.Valid {
		t := convertedAt.Time
		q.ConvertedAt = &t
	}
	if davGeneratedAt.Valid {
		t := davGeneratedAt.Time
		q.DAVGeneratedAt = &t
	}
	return &q, nil
}

func scanItem(s scanner) (*quoteentity.SalesQuotationItem, error) {
	var item quoteentity.SalesQuotationItem
	var deliveryDate pgtype.Date
	var requested, unit, attended, cancelled, discount, ipi, st, gross, net, netIPI pgtype.Numeric
	err := s.Scan(
		&item.Code, &item.SalesQuotationCode, &item.Sequence, &item.ItemCode, &item.Mask, &item.SalesUOM, &item.WarehouseCode,
		&item.PriceTableCode, &requested, &unit, &attended, &cancelled, &deliveryDate,
		&item.DeliveryDateFirm, &discount, &ipi, &st, &gross, &net, &netIPI,
		&item.Status, &item.Notes, &item.IsActive, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "cannot scan NULL") {
			return nil, fmt.Errorf("scan sales quotation item: %w", err)
		}
		return nil, err
	}
	item.RequestedQty = numericToDecimal(requested)
	item.UnitPrice = numericToDecimal(unit)
	item.AttendedQty = numericToDecimal(attended)
	item.CancelledQty = numericToDecimal(cancelled)
	item.DiscountPct = numericToDecimal(discount)
	item.IPIPct = numericToDecimal(ipi)
	item.STPct = numericToDecimal(st)
	item.TotalGross = numericToDecimal(gross)
	item.TotalNet = numericToDecimal(net)
	item.TotalNetWithIPI = numericToDecimal(netIPI)
	item.Balance = item.RequestedQty.Sub(item.AttendedQty).Sub(item.CancelledQty)
	if deliveryDate.Valid {
		t := deliveryDate.Time
		item.DeliveryDate = &t
	}
	return &item, nil
}

func numericToDecimal(v pgtype.Numeric) decimal.Decimal {
	if !v.Valid || v.Int == nil || v.NaN || v.InfinityModifier != pgtype.Finite {
		return decimal.Zero
	}
	return decimal.NewFromBigInt(v.Int, v.Exp)
}

var _ quoterepo.SalesQuotationRepository = (*Repository)(nil)

func (r *Repository) GetParameters(ctx context.Context) (*quoteentity.Parameters, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	p := quoteentity.DefaultParameters(tenantID)
	var minimum string
	err = r.pool.QueryRow(ctx, `SELECT purchase_order_prompt,delivery_authorization_prompt,final_consumer_customer_code,allow_service_items_nfce,default_nfce,minimum_cif_freight::text,add_redelivery_to_freight FROM public.sales_quotation_parameters WHERE enterprise_code=$1`, tenantID).Scan(&p.PurchaseOrderPrompt, &p.DeliveryAuthorizationPrompt, &p.FinalConsumerCustomerCode, &p.AllowServiceItemsNFCe, &p.DefaultNFCe, &minimum, &p.AddRedeliveryToFreight)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, nil
	}
	if err != nil {
		return nil, err
	}
	p.MinimumCIFFreight, err = decimal.NewFromString(minimum)
	return p, err
}

func (r *Repository) SaveParameters(ctx context.Context, p *quoteentity.Parameters) (*quoteentity.Parameters, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	p.EnterpriseCode = tenantID
	if err := p.Validate(); err != nil {
		return nil, err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO public.sales_quotation_parameters(enterprise_code,purchase_order_prompt,delivery_authorization_prompt,final_consumer_customer_code,allow_service_items_nfce,default_nfce,minimum_cif_freight,add_redelivery_to_freight) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(enterprise_code) DO UPDATE SET purchase_order_prompt=EXCLUDED.purchase_order_prompt,delivery_authorization_prompt=EXCLUDED.delivery_authorization_prompt,final_consumer_customer_code=EXCLUDED.final_consumer_customer_code,allow_service_items_nfce=EXCLUDED.allow_service_items_nfce,default_nfce=EXCLUDED.default_nfce,minimum_cif_freight=EXCLUDED.minimum_cif_freight,add_redelivery_to_freight=EXCLUDED.add_redelivery_to_freight,updated_at=NOW()`, tenantID, p.PurchaseOrderPrompt, p.DeliveryAuthorizationPrompt, p.FinalConsumerCustomerCode, p.AllowServiceItemsNFCe, p.DefaultNFCe, p.MinimumCIFFreight, p.AddRedeliveryToFreight)
	if err != nil {
		return nil, err
	}
	return r.GetParameters(ctx)
}

func (r *Repository) SaveCommissionPattern(ctx context.Context, p *quoteentity.CommissionPattern) (*quoteentity.CommissionPattern, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	p.EnterpriseCode = tenantID
	if p.Code == 0 {
		err = r.pool.QueryRow(ctx, `INSERT INTO public.sales_quotation_commission_pattern_sequences(enterprise_code,last_code) VALUES($1,1) ON CONFLICT(enterprise_code) DO UPDATE SET last_code=sales_quotation_commission_pattern_sequences.last_code+1 RETURNING last_code`, tenantID).Scan(&p.Code)
		if err != nil {
			return nil, err
		}
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx, `INSERT INTO public.sales_quotation_commission_patterns(enterprise_code,code,description,commission_pct,invoice_pct,payment_pct) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(enterprise_code,code) DO UPDATE SET description=EXCLUDED.description,commission_pct=EXCLUDED.commission_pct,invoice_pct=EXCLUDED.invoice_pct,payment_pct=EXCLUDED.payment_pct,is_active=TRUE,updated_at=NOW() RETURNING id,is_active,created_at,updated_at`, tenantID, p.Code, p.Description, p.CommissionPct, p.InvoicePct, p.PaymentPct).Scan(&p.ID, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (r *Repository) ListCommissionPatterns(ctx context.Context) ([]*quoteentity.CommissionPattern, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT id,enterprise_code,code,description,commission_pct::text,invoice_pct::text,payment_pct::text,is_active,created_at,updated_at FROM public.sales_quotation_commission_patterns WHERE enterprise_code=$1 AND is_active ORDER BY code`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*quoteentity.CommissionPattern
	for rows.Next() {
		p := new(quoteentity.CommissionPattern)
		var a, b, c string
		if err := rows.Scan(&p.ID, &p.EnterpriseCode, &p.Code, &p.Description, &a, &b, &c, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.CommissionPct, _ = decimal.NewFromString(a)
		p.InvoicePct, _ = decimal.NewFromString(b)
		p.PaymentPct, _ = decimal.NewFromString(c)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) SaveCancellationReason(ctx context.Context, v *quoteentity.CancellationReason) (*quoteentity.CancellationReason, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	v.EnterpriseCode = tenantID
	if err := v.Validate(); err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx, `INSERT INTO public.sales_quotation_cancellation_reasons(enterprise_code,code,description,allow_uncancel,require_complement) VALUES($1,$2,$3,$4,$5) ON CONFLICT(enterprise_code,code) DO UPDATE SET description=EXCLUDED.description,allow_uncancel=EXCLUDED.allow_uncancel,require_complement=EXCLUDED.require_complement,is_active=TRUE,updated_at=NOW() RETURNING id,is_active,created_at,updated_at`, tenantID, v.Code, v.Description, v.AllowUncancel, v.RequireComplement).Scan(&v.ID, &v.IsActive, &v.CreatedAt, &v.UpdatedAt)
	return v, err
}

func (r *Repository) ListCancellationReasons(ctx context.Context) ([]*quoteentity.CancellationReason, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT id,enterprise_code,code,description,allow_uncancel,require_complement,is_active,created_at,updated_at FROM public.sales_quotation_cancellation_reasons WHERE enterprise_code=$1 AND is_active ORDER BY code`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*quoteentity.CancellationReason
	for rows.Next() {
		v := new(quoteentity.CancellationReason)
		if err := rows.Scan(&v.ID, &v.EnterpriseCode, &v.Code, &v.Description, &v.AllowUncancel, &v.RequireComplement, &v.IsActive, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) GetCancellationReason(ctx context.Context, code int64) (*quoteentity.CancellationReason, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	v := new(quoteentity.CancellationReason)
	err = r.pool.QueryRow(ctx, `SELECT id,enterprise_code,code,description,allow_uncancel,require_complement,is_active,created_at,updated_at FROM public.sales_quotation_cancellation_reasons WHERE enterprise_code=$1 AND code=$2 AND is_active`, tenantID, code).Scan(&v.ID, &v.EnterpriseCode, &v.Code, &v.Description, &v.AllowUncancel, &v.RequireComplement, &v.IsActive, &v.CreatedAt, &v.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, quoterepo.ErrCancellationReasonNotFound
	}
	return v, err
}

func (r *Repository) GenerateDAV(ctx context.Context, code int64) (*quoteentity.SalesQuotation, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	key := uuid.New()
	row := r.pool.QueryRow(ctx, `UPDATE public.sales_quotations SET dav_generated_at=COALESCE(dav_generated_at,NOW()),dav_report_key=COALESCE(dav_report_key,$3),updated_at=NOW() WHERE code=$1 AND enterprise_code=$2 AND status NOT IN ('CANCELLED','EXPIRED') RETURNING `+quotationColumns, code, tenantID, key)
	return scanQuotation(row)
}

func (r *Repository) CreateAttachment(ctx context.Context, a *quoteentity.Attachment) (*quoteentity.Attachment, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	if err := a.Validate(); err != nil {
		return nil, err
	}
	err = r.pool.QueryRow(ctx, `INSERT INTO public.sales_quotation_attachments(sales_quotation_code,file_name,content_type,file_size,storage_key,uploaded_by,content) SELECT $1,$2,$3,$4,$5,$6,$7 WHERE EXISTS(SELECT 1 FROM public.sales_quotations WHERE code=$1 AND enterprise_code=$8) RETURNING id,uploaded_at`, a.SalesQuotationCode, a.FileName, a.ContentType, a.FileSize, a.StorageKey, a.UploadedBy, a.Content, tenantID).Scan(&a.ID, &a.UploadedAt)
	return a, err
}

func (r *Repository) ListAttachments(ctx context.Context, quotationCode int64) ([]*quoteentity.Attachment, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT a.id,a.sales_quotation_code,a.file_name,COALESCE(a.content_type,''),a.file_size,a.storage_key,a.uploaded_at,COALESCE(a.uploaded_by,'00000000-0000-0000-0000-000000000000'::uuid) FROM public.sales_quotation_attachments a JOIN public.sales_quotations q ON q.code=a.sales_quotation_code WHERE a.sales_quotation_code=$1 AND q.enterprise_code=$2 ORDER BY a.uploaded_at DESC`, quotationCode, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*quoteentity.Attachment
	for rows.Next() {
		a := new(quoteentity.Attachment)
		if err := rows.Scan(&a.ID, &a.SalesQuotationCode, &a.FileName, &a.ContentType, &a.FileSize, &a.StorageKey, &a.UploadedAt, &a.UploadedBy); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repository) GetAttachment(ctx context.Context, quotationCode, attachmentID int64) (*quoteentity.Attachment, error) {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return nil, err
	}
	a := new(quoteentity.Attachment)
	err = r.pool.QueryRow(ctx, `SELECT a.id,a.sales_quotation_code,a.file_name,COALESCE(a.content_type,''),a.file_size,a.storage_key,a.uploaded_at,COALESCE(a.uploaded_by,'00000000-0000-0000-0000-000000000000'::uuid),a.content FROM public.sales_quotation_attachments a JOIN public.sales_quotations q ON q.code=a.sales_quotation_code WHERE a.id=$1 AND a.sales_quotation_code=$2 AND q.enterprise_code=$3`, attachmentID, quotationCode, tenantID).Scan(&a.ID, &a.SalesQuotationCode, &a.FileName, &a.ContentType, &a.FileSize, &a.StorageKey, &a.UploadedAt, &a.UploadedBy, &a.Content)
	return a, err
}

func (r *Repository) DeleteAttachment(ctx context.Context, quotationCode, attachmentID int64) error {
	tenantID, err := tenant.ID(ctx)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM public.sales_quotation_attachments a USING public.sales_quotations q WHERE a.id=$1 AND a.sales_quotation_code=$2 AND q.code=a.sales_quotation_code AND q.enterprise_code=$3`, attachmentID, quotationCode, tenantID)
	if err == nil && tag.RowsAffected() == 0 {
		return fmt.Errorf("attachment not found")
	}
	return err
}
