package sales_quotation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	quoteentity "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	quoterepo "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) NextQuotationNumber(ctx context.Context, enterpriseCode int64) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `
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
 delivery_authorization, notes, obs_customer, created_by
) VALUES (
 $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
 $11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
 $21,$22,$23,$24,$25,$26,$27,$28,$29,$30,
 $31,$32,$33,$34,$35,$36,$37,$38,$39,$40
) RETURNING `+quotationColumns,
		q.QuotationNumber, q.EnterpriseCode, string(q.Status), string(q.QuotationType), q.EmissionDate, q.DigitDate,
		q.ValidUntil, q.DeliveryDate, q.DeliveryDateFirm, q.PurchaseOrderNumber,
		q.CustomerCode, q.BillingAddressCode, q.ShippingAddressCode,
		q.RepresentativeCode, q.SalesDivisionCode, q.PriceTableCode, q.PaymentTermCode,
		q.CurrencyCode, q.ProbabilityPct, q.CommissionPct, q.IsNFCe, q.Street, q.StreetNumber,
		q.ForeignDocument, string(q.ReleaseStatus), q.CommercialBlocked, q.CommercialBlockReason,
		q.CarrierCode, q.FreightType, q.VerifyFreight, q.FreightValue, q.RedeliveryFreightValue,
		q.InsuranceValue, q.DiscountValue, q.SurchargeValue, q.RetainedTaxValue,
		q.DeliveryAuthorization, q.Notes, q.ObsCustomer, q.CreatedBy,
	)
	return scanQuotation(row)
}

func (r *Repository) Update(ctx context.Context, q *quoteentity.SalesQuotation) (*quoteentity.SalesQuotation, error) {
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
 notes=$34, obs_customer=$35, updated_at=NOW()
WHERE code=$36 AND is_active=TRUE
RETURNING `+quotationColumns,
		string(q.Status), string(q.QuotationType), q.ValidUntil, q.DeliveryDate, q.DeliveryDateFirm,
		q.PurchaseOrderNumber, q.CustomerCode, q.BillingAddressCode, q.ShippingAddressCode,
		q.RepresentativeCode, q.SalesDivisionCode, q.PriceTableCode,
		q.PaymentTermCode, q.CurrencyCode, q.ProbabilityPct, q.CommissionPct,
		q.IsNFCe, q.Street, q.StreetNumber, q.ForeignDocument,
		string(q.ReleaseStatus), q.CommercialBlocked, q.CommercialBlockReason,
		q.CarrierCode, q.FreightType, q.VerifyFreight, q.FreightValue,
		q.RedeliveryFreightValue, q.InsuranceValue, q.DiscountValue, q.SurchargeValue,
		q.RetainedTaxValue, q.DeliveryAuthorization, q.Notes, q.ObsCustomer, q.Code,
	)
	return scanQuotation(row)
}

func (r *Repository) GetByCode(ctx context.Context, code int64) (*quoteentity.SalesQuotation, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+quotationColumns+` FROM public.sales_quotations WHERE code=$1`, code)
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
	sql := `SELECT ` + quotationColumns + ` FROM public.sales_quotations WHERE TRUE`
	args := []any{}
	add := func(clause string, arg any) {
		args = append(args, arg)
		sql += fmt.Sprintf(" AND "+clause, len(args))
	}
	if filter.CustomerCode != nil {
		add("customer_code=$%d", *filter.CustomerCode)
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

func (r *Repository) Cancel(ctx context.Context, code int64, reason string, complement *string) error {
	_, err := r.pool.Exec(ctx, `UPDATE public.sales_quotations SET status='CANCELLED', cancel_reason=$2, cancel_complement=$3, updated_at=NOW() WHERE code=$1`, code, reason, complement)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO public.sales_quotation_events (sales_quotation_code, event_type, reason, complement) VALUES ($1,'CANCEL',$2,$3)`, code, reason, complement)
	return err
}

func (r *Repository) Uncancel(ctx context.Context, code int64, reason string, complement *string) error {
	_, err := r.pool.Exec(ctx, `UPDATE public.sales_quotations SET status='OF', is_active=TRUE, cancel_reason=NULL, cancel_complement=NULL, updated_at=NOW() WHERE code=$1`, code)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO public.sales_quotation_events (sales_quotation_code, event_type, reason, complement) VALUES ($1,'UNCANCEL',$2,$3)`, code, reason, complement)
	return err
}

func (r *Repository) Attend(ctx context.Context, code int64, reason string, complement *string, eventDate time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE public.sales_quotations SET status='ATTENDED', attended_reason=$2, attended_at=$3, updated_at=NOW() WHERE code=$1`, code, reason, eventDate)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO public.sales_quotation_events (sales_quotation_code, event_type, reason, complement, event_date) VALUES ($1,'ATTEND',$2,$3,$4)`, code, reason, complement, eventDate)
	return err
}

func (r *Repository) ChangeStatus(ctx context.Context, code int64, status quoteentity.SalesQuotationStatus) error {
	_, err := r.pool.Exec(ctx, `UPDATE public.sales_quotations SET status=$2, updated_at=NOW() WHERE code=$1`, code, string(status))
	return err
}

func (r *Repository) MarkConverted(ctx context.Context, quotationCode, salesOrderCode int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE public.sales_quotations SET status='ATTENDED', attended_reason='Convertido em pedido de venda', attended_at=NOW(), converted_sales_order_code=$2, converted_at=NOW(), updated_at=NOW() WHERE code=$1`, quotationCode, salesOrderCode)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO public.sales_quotation_events (sales_quotation_code, event_type, reason) VALUES ($1,'CONVERT','Convertido em pedido de venda')`, quotationCode)
	return err
}

func (r *Repository) Report(ctx context.Context, filter quoterepo.SalesQuotationFilter) (*quoterepo.SalesQuotationReport, error) {
	sql := `SELECT COUNT(*), COALESCE(SUM(total_gross),0), COALESCE(SUM(total_net),0),
COUNT(*) FILTER (WHERE status IN ('R','A','OA','OF')),
COUNT(*) FILTER (WHERE status='OF'),
COUNT(*) FILTER (WHERE converted_sales_order_code IS NOT NULL OR status='ATTENDED'),
COUNT(*) FILTER (WHERE status='CANCELLED'),
COUNT(*) FILTER (WHERE status='EXPIRED'),
COALESCE(SUM(total_net * probability_pct / 100),0),
COALESCE(SUM(retained_tax_value),0)
FROM public.sales_quotations WHERE TRUE`
	args := []any{}
	add := func(clause string, arg any) {
		args = append(args, arg)
		sql += fmt.Sprintf(" AND "+clause, len(args))
	}
	if filter.CustomerCode != nil {
		add("customer_code=$%d", *filter.CustomerCode)
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
	err := r.pool.QueryRow(ctx, sql, args...).Scan(&report.TotalQuotations, &gross, &net, &report.OpenCount, &report.ApprovedCount, &report.ConvertedCount, &report.CancelledCount, &report.ExpiredCount, &weighted, &retained)
	report.TotalGross = pgutil.FromPgNumericToFloat64(gross)
	report.TotalNet = pgutil.FromPgNumericToFloat64(net)
	report.WeightedNet = pgutil.FromPgNumericToFloat64(weighted)
	report.RetainedTax = pgutil.FromPgNumericToFloat64(retained)
	return &report, err
}

func (r *Repository) CreateItem(ctx context.Context, item *quoteentity.SalesQuotationItem) (*quoteentity.SalesQuotationItem, error) {
	row := r.pool.QueryRow(ctx, `
INSERT INTO public.sales_quotation_items (
 sales_quotation_code, sequence, item_code, mask, sales_uom, warehouse_code,
 price_table_code, requested_qty, unit_price, attended_qty, cancelled_qty,
 delivery_date, delivery_date_firm, discount_pct, ipi_pct, st_pct,
 total_gross, total_net, total_net_with_ipi, status, notes
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,0,0,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
RETURNING `+itemColumns,
		item.SalesQuotationCode, item.Sequence, item.ItemCode, item.Mask, item.SalesUOM, item.WarehouseCode,
		item.PriceTableCode, item.RequestedQty, item.UnitPrice, item.DeliveryDate, item.DeliveryDateFirm,
		item.DiscountPct, item.IPIPct, item.STPct, item.TotalGross, item.TotalNet, item.TotalNetWithIPI,
		string(item.Status), item.Notes,
	)
	return scanItem(row)
}

func (r *Repository) UpdateItem(ctx context.Context, item *quoteentity.SalesQuotationItem) (*quoteentity.SalesQuotationItem, error) {
	row := r.pool.QueryRow(ctx, `
UPDATE public.sales_quotation_items SET
 requested_qty=$1, unit_price=$2, attended_qty=$3, cancelled_qty=$4,
 delivery_date=$5, delivery_date_firm=$6, discount_pct=$7, ipi_pct=$8,
 st_pct=$9, total_gross=$10, total_net=$11, total_net_with_ipi=$12,
 status=$13, notes=$14, updated_at=NOW()
WHERE code=$15 AND is_active=TRUE
RETURNING `+itemColumns,
		item.RequestedQty, item.UnitPrice, item.AttendedQty, item.CancelledQty,
		item.DeliveryDate, item.DeliveryDateFirm, item.DiscountPct, item.IPIPct,
		item.STPct, item.TotalGross, item.TotalNet, item.TotalNetWithIPI,
		string(item.Status), item.Notes, item.Code,
	)
	return scanItem(row)
}

func (r *Repository) ListItems(ctx context.Context, quotationCode int64) ([]*quoteentity.SalesQuotationItem, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+itemColumns+` FROM public.sales_quotation_items WHERE sales_quotation_code=$1 AND is_active=TRUE ORDER BY sequence`, quotationCode)
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

func (r *Repository) CancelItem(ctx context.Context, itemCode int64) error {
	var quotationCode int64
	err := r.pool.QueryRow(ctx, `UPDATE public.sales_quotation_items SET status='CANCELLED', cancelled_qty=requested_qty, is_active=FALSE, updated_at=NOW() WHERE code=$1 RETURNING sales_quotation_code`, itemCode).Scan(&quotationCode)
	if err != nil {
		return err
	}
	return r.RecalculateTotals(ctx, quotationCode)
}

func (r *Repository) RecalculateTotals(ctx context.Context, quotationCode int64) error {
	_, err := r.pool.Exec(ctx, `
UPDATE public.sales_quotations q SET
 total_gross = COALESCE((SELECT SUM(total_gross) FROM public.sales_quotation_items WHERE sales_quotation_code=$1 AND is_active=TRUE),0),
 total_net = COALESCE((SELECT SUM(total_net) FROM public.sales_quotation_items WHERE sales_quotation_code=$1 AND is_active=TRUE),0)
   + q.freight_value + q.redelivery_freight_value + q.insurance_value + q.surcharge_value
   - q.discount_value - q.retained_tax_value,
 updated_at = NOW()
WHERE q.code=$1`, quotationCode)
	return err
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
converted_sales_order_code, converted_at, is_active, created_at, updated_at, created_by`

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
	var attendedAt, convertedAt pgtype.Timestamptz
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
		&q.CreatedAt, &q.UpdatedAt, &q.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	q.ProbabilityPct = pgutil.FromPgNumericToFloat64(probability)
	q.CommissionPct = pgutil.FromPgNumericToFloat64(commission)
	q.FreightValue = pgutil.FromPgNumericToFloat64(freight)
	q.RedeliveryFreightValue = pgutil.FromPgNumericToFloat64(redeliveryFreight)
	q.InsuranceValue = pgutil.FromPgNumericToFloat64(insurance)
	q.DiscountValue = pgutil.FromPgNumericToFloat64(discount)
	q.SurchargeValue = pgutil.FromPgNumericToFloat64(surcharge)
	q.RetainedTaxValue = pgutil.FromPgNumericToFloat64(retained)
	q.TotalGross = pgutil.FromPgNumericToFloat64(gross)
	q.TotalNet = pgutil.FromPgNumericToFloat64(net)
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
	item.RequestedQty = pgutil.FromPgNumericToFloat64(requested)
	item.UnitPrice = pgutil.FromPgNumericToFloat64(unit)
	item.AttendedQty = pgutil.FromPgNumericToFloat64(attended)
	item.CancelledQty = pgutil.FromPgNumericToFloat64(cancelled)
	item.DiscountPct = pgutil.FromPgNumericToFloat64(discount)
	item.IPIPct = pgutil.FromPgNumericToFloat64(ipi)
	item.STPct = pgutil.FromPgNumericToFloat64(st)
	item.TotalGross = pgutil.FromPgNumericToFloat64(gross)
	item.TotalNet = pgutil.FromPgNumericToFloat64(net)
	item.TotalNetWithIPI = pgutil.FromPgNumericToFloat64(netIPI)
	item.Balance = item.RequestedQty - item.AttendedQty - item.CancelledQty
	if deliveryDate.Valid {
		t := deliveryDate.Time
		item.DeliveryDate = &t
	}
	return &item, nil
}

var _ quoterepo.SalesQuotationRepository = (*Repository)(nil)
