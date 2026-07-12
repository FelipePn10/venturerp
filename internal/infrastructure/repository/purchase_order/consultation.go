package purchase_order

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/FelipePn10/panossoerp/internal/application/usecase/purchase_order_uc"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

func (r *PurchaseOrderRepositorySQLC) Consult(ctx context.Context, f purchase_order_uc.PurchaseOrderConsultationFilter) ([]purchase_order_uc.PurchaseOrderConsultationResult, error) {
	args := []any{f.EnterpriseID}
	where := []string{"e.id = $1", "po.is_active"}
	add := func(expr string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(expr, len(args)))
	}
	if f.OrderFrom != nil {
		add("po.order_number >= $%d", *f.OrderFrom)
	}
	if f.OrderTo != nil {
		add("po.order_number <= $%d", *f.OrderTo)
	}
	if f.SupplierFrom != nil {
		add("po.supplier_code >= $%d", *f.SupplierFrom)
	}
	if f.SupplierTo != nil {
		add("po.supplier_code <= $%d", *f.SupplierTo)
	}
	if f.RequestTypeCode != nil {
		add("po.request_type_code = $%d", *f.RequestTypeCode)
	}
	if f.EmissionFrom != nil {
		add("po.emission_date >= $%d", *f.EmissionFrom)
	}
	if f.EmissionTo != nil {
		add("po.emission_date <= $%d", *f.EmissionTo)
	}
	if f.DeliveryFrom != nil {
		add("po.delivery_date >= $%d", *f.DeliveryFrom)
	}
	if f.DeliveryTo != nil {
		add("po.delivery_date <= $%d", *f.DeliveryTo)
	}
	if f.BuyerCode != nil {
		add("po.buyer_employee_code = $%d", *f.BuyerCode)
	}
	if f.OrderType != "" {
		add("po.order_type = $%d", f.OrderType)
	}
	if f.OnlyKanban {
		where = append(where, "po.kanban_origin")
	}
	itemPred := []string{"pi.purchase_order_code = po.code", "pi.is_active"}
	if f.ItemFrom != nil {
		args = append(args, *f.ItemFrom)
		itemPred = append(itemPred, fmt.Sprintf("pi.item_code >= $%d", len(args)))
	}
	if f.ItemTo != nil {
		args = append(args, *f.ItemTo)
		itemPred = append(itemPred, fmt.Sprintf("pi.item_code <= $%d", len(args)))
	}
	switch f.Position {
	case purchase_order_uc.PositionAttended:
		itemPred = append(itemPred, "pi.received_qty + pi.cancelled_qty = pi.requested_qty", "pi.cancelled_qty <> pi.requested_qty")
	case purchase_order_uc.PositionPending:
		itemPred = append(itemPred, "pi.received_qty + pi.cancelled_qty <> pi.requested_qty")
	case purchase_order_uc.PositionCancelled:
		itemPred = append(itemPred, "pi.cancelled_qty > 0")
	}
	if f.ItemFrom != nil || f.ItemTo != nil || f.Position != "" {
		where = append(where, "EXISTS (SELECT 1 FROM purchase_order_items pi WHERE "+strings.Join(itemPred, " AND ")+")")
	}
	if f.ImportFrom != nil || f.ImportTo != nil {
		imp := []string{"ip.enterprise_code = po.enterprise_code", "ip.purchase_order_code = po.code"}
		if f.ImportFrom != nil {
			args = append(args, *f.ImportFrom)
			imp = append(imp, fmt.Sprintf("ip.process_number >= $%d", len(args)))
		}
		if f.ImportTo != nil {
			args = append(args, *f.ImportTo)
			imp = append(imp, fmt.Sprintf("ip.process_number <= $%d", len(args)))
		}
		where = append(where, "EXISTS (SELECT 1 FROM import_processes ip WHERE "+strings.Join(imp, " AND ")+")")
	}
	convertSelect := "1::numeric"
	currencyArg, dateArg := 0, 0
	if f.Convert {
		args = append(args, f.TargetCurrency, *f.BaseDate)
		currencyArg, dateArg = len(args)-1, len(args)
		convertSelect = fmt.Sprintf(`CASE WHEN po.currency_code = $%d THEN 1::numeric
			ELSE (CASE WHEN po.currency_code = 'BRL' THEN 1::numeric ELSE src.rate_to_base END) /
			     (CASE WHEN $%d = 'BRL' THEN 1::numeric ELSE dst.rate_to_base END) END`, currencyArg, currencyArg)
	}
	joins := ""
	if f.Convert {
		joins = fmt.Sprintf(`LEFT JOIN purchase_order_currency_rates src ON src.enterprise_id=e.id AND src.currency_code=po.currency_code AND src.rate_date=$%d
		LEFT JOIN purchase_order_currency_rates dst ON dst.enterprise_id=e.id AND dst.currency_code=$%d AND dst.rate_date=$%d`, dateArg, currencyArg, dateArg)
	}
	args = append(args, f.Limit, f.Offset)
	query := fmt.Sprintf(`SELECT po.code, po.order_number, po.supplier_code, po.request_type_code, po.emission_date,
		po.delivery_date, po.buyer_employee_code, po.order_type, po.customer_code, po.kanban_origin,
		po.currency_code, %s AS factor,
		COALESCE(t.products,0), po.freight_value, COALESCE(t.discount,0), COALESCE(t.additions,0),
		COALESCE(t.net,0), COALESCE(t.total,0),
		COALESCE((SELECT array_agg(ip.process_number ORDER BY ip.process_number) FROM import_processes ip
		 WHERE ip.enterprise_code=po.enterprise_code AND ip.purchase_order_code=po.code), ARRAY[]::bigint[])
	FROM purchase_orders po JOIN enterprise e ON e.code=po.enterprise_code
	%s
	LEFT JOIN LATERAL (SELECT
		SUM(i.requested_qty*i.unit_price) products,
		SUM(i.requested_qty*i.unit_price*i.discount_pct/100) discount,
		SUM(i.additions) additions,
		SUM(i.requested_qty*i.unit_price*(1-i.discount_pct/100)+i.additions) net,
		SUM(i.requested_qty*i.unit_price*(1-i.discount_pct/100)+i.additions+
		 CASE WHEN po.order_type='OCL' THEN COALESCE(i.ipi_value,(i.requested_qty*i.unit_price*(1-i.discount_pct/100)+i.additions)*i.ipi_pct/100)+COALESCE(i.icms_st_value,(i.requested_qty*i.unit_price*(1-i.discount_pct/100)+i.additions)*i.icms_st_pct/100) ELSE 0 END) total
		FROM purchase_order_items i WHERE i.purchase_order_code=po.code AND i.is_active) t ON true
	WHERE %s ORDER BY po.order_number DESC LIMIT $%d OFFSET $%d`, convertSelect, joins, strings.Join(where, " AND "), len(args)-1, len(args))
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("consult purchase orders: %w", err)
	}
	defer rows.Close()
	results := make([]purchase_order_uc.PurchaseOrderConsultationResult, 0)
	for rows.Next() {
		var x purchase_order_uc.PurchaseOrderConsultationResult
		var factor *decimal.Decimal
		if err := rows.Scan(&x.Code, &x.OrderNumber, &x.SupplierCode, &x.RequestTypeCode, &x.EmissionDate, &x.DeliveryDate, &x.BuyerCode, &x.OrderType, &x.CustomerCode, &x.KanbanOrigin, &x.CurrencyCode, &factor, &x.ProductsTotal, &x.Freight, &x.Discount, &x.Additions, &x.Net, &x.Total, &x.ImportProcesses); err != nil {
			return nil, fmt.Errorf("scan purchase order consultation: %w", err)
		}
		if factor == nil {
			return nil, fmt.Errorf("exchange rate missing for order %d on %s", x.OrderNumber, f.BaseDate.Format("2006-01-02"))
		}
		x.ConversionRate = *factor
		x.DisplayCurrency = x.CurrencyCode
		if f.Convert {
			x.DisplayCurrency = f.TargetCurrency
		}
		x.ProductsTotal = x.ProductsTotal.Mul(x.ConversionRate)
		x.Freight = x.Freight.Mul(x.ConversionRate)
		x.ProductsWithFreight = x.ProductsTotal.Add(x.Freight)
		x.Discount = x.Discount.Mul(x.ConversionRate)
		x.Additions = x.Additions.Mul(x.ConversionRate)
		x.Net = x.Net.Mul(x.ConversionRate)
		x.Total = x.Total.Add(x.Freight).Mul(x.ConversionRate)
		results = append(results, x)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return results, nil
	}
	if err := r.loadConsultationItems(ctx, results, f); err != nil {
		return nil, err
	}
	if err := r.loadConsultationAttachments(ctx, results); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *PurchaseOrderRepositorySQLC) loadConsultationItems(ctx context.Context, results []purchase_order_uc.PurchaseOrderConsultationResult, f purchase_order_uc.PurchaseOrderConsultationFilter) error {
	codes := make([]int64, len(results))
	factors := make(map[int64]decimal.Decimal, len(results))
	for i := range results {
		codes[i] = results[i].Code
		factors[results[i].Code] = results[i].ConversionRate
	}
	args := []any{codes}
	conditions := []string{"i.purchase_order_code=ANY($1)", "i.is_active"}
	if !f.AllItems {
		if f.ItemFrom != nil {
			args = append(args, *f.ItemFrom)
			conditions = append(conditions, fmt.Sprintf("i.item_code >= $%d", len(args)))
		}
		if f.ItemTo != nil {
			args = append(args, *f.ItemTo)
			conditions = append(conditions, fmt.Sprintf("i.item_code <= $%d", len(args)))
		}
		switch f.Position {
		case purchase_order_uc.PositionAttended:
			conditions = append(conditions, "i.received_qty+i.cancelled_qty=i.requested_qty", "i.cancelled_qty<>i.requested_qty")
		case purchase_order_uc.PositionPending:
			conditions = append(conditions, "i.received_qty+i.cancelled_qty<>i.requested_qty")
		case purchase_order_uc.PositionCancelled:
			conditions = append(conditions, "i.cancelled_qty>0")
		}
	}
	q := `SELECT i.purchase_order_code,i.code,i.sequence,i.item_code,i.requested_qty,i.received_qty,i.cancelled_qty,i.unit_price,
		i.requested_qty*i.unit_price gross, i.requested_qty*i.unit_price*i.discount_pct/100 discount, i.additions,
		i.requested_qty*i.unit_price*(1-i.discount_pct/100)+i.additions net,
		CASE WHEN po.order_type='OCL' THEN COALESCE(i.ipi_base,i.requested_qty*i.unit_price*(1-i.discount_pct/100)+i.additions) END,
		CASE WHEN po.order_type='OCL' THEN i.ipi_pct END,
		CASE WHEN po.order_type='OCL' THEN COALESCE(i.ipi_value,COALESCE(i.ipi_base,i.requested_qty*i.unit_price*(1-i.discount_pct/100)+i.additions)*i.ipi_pct/100) END,
		CASE WHEN po.order_type='OCL' THEN COALESCE(i.icms_base,i.requested_qty*i.unit_price*(1-i.discount_pct/100)+i.additions) END,
		CASE WHEN po.order_type='OCL' THEN i.icms_pct END,
		CASE WHEN po.order_type='OCL' THEN COALESCE(i.icms_value,COALESCE(i.icms_base,i.requested_qty*i.unit_price*(1-i.discount_pct/100)+i.additions)*i.icms_pct/100) END,
		CASE WHEN po.order_type='OCL' THEN COALESCE(i.icms_st_base,i.requested_qty*i.unit_price*(1-i.discount_pct/100)+i.additions) END,
		CASE WHEN po.order_type='OCL' THEN i.icms_st_pct END,
		CASE WHEN po.order_type='OCL' THEN COALESCE(i.icms_st_value,COALESCE(i.icms_st_base,i.requested_qty*i.unit_price*(1-i.discount_pct/100)+i.additions)*i.icms_st_pct/100) END
		FROM purchase_order_items i JOIN purchase_orders po ON po.code=i.purchase_order_code WHERE ` + strings.Join(conditions, " AND ") + ` ORDER BY i.purchase_order_code,i.sequence`
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("consult purchase order items: %w", err)
	}
	defer rows.Close()
	idx := map[int64]int{}
	for i := range results {
		idx[results[i].Code] = i
		results[i].Items = []purchase_order_uc.PurchaseOrderConsultationItem{}
	}
	for rows.Next() {
		var order int64
		var x purchase_order_uc.PurchaseOrderConsultationItem
		if err := rows.Scan(&order, &x.Code, &x.Sequence, &x.ItemCode, &x.RequestedQty, &x.ReceivedQty, &x.CancelledQty, &x.UnitPrice, &x.Gross, &x.Discount, &x.Additions, &x.Net, &x.IPIBase, &x.IPIRate, &x.IPIValue, &x.ICMSBase, &x.ICMSRate, &x.ICMSValue, &x.ICMSSTBase, &x.ICMSSTRate, &x.ICMSSTValue); err != nil {
			return err
		}
		if x.CancelledQty.Equal(x.RequestedQty) {
			x.Position = purchase_order_uc.PositionCancelled
		} else if x.ReceivedQty.Add(x.CancelledQty).Equal(x.RequestedQty) {
			x.Position = purchase_order_uc.PositionAttended
		} else {
			x.Position = purchase_order_uc.PositionPending
		}
		factor := factors[order]
		x.UnitPrice = x.UnitPrice.Mul(factor)
		x.Gross = x.Gross.Mul(factor)
		x.Discount = x.Discount.Mul(factor)
		x.Additions = x.Additions.Mul(factor)
		x.Net = x.Net.Mul(factor)
		for _, amount := range []**decimal.Decimal{&x.IPIBase, &x.IPIValue, &x.ICMSBase, &x.ICMSValue, &x.ICMSSTBase, &x.ICMSSTValue} {
			if *amount != nil {
				converted := (*amount).Mul(factor)
				*amount = &converted
			}
		}
		x.Total = x.Net
		if x.IPIValue != nil {
			x.Total = x.Total.Add(*x.IPIValue)
		}
		if x.ICMSSTValue != nil {
			x.Total = x.Total.Add(*x.ICMSSTValue)
		}
		i := idx[order]
		results[i].Items = append(results[i].Items, x)
	}
	return rows.Err()
}

func (r *PurchaseOrderRepositorySQLC) loadConsultationAttachments(ctx context.Context, results []purchase_order_uc.PurchaseOrderConsultationResult) error {
	codes := make([]int64, len(results))
	idx := map[int64]int{}
	for i := range results {
		codes[i] = results[i].Code
		idx[results[i].Code] = i
	}
	rows, err := r.db.Query(ctx, `SELECT purchase_order_code,id,file_name,content_type,file_size,created_at FROM purchase_order_attachments WHERE purchase_order_code=ANY($1) ORDER BY created_at DESC`, codes)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var order int64
		var a purchase_order_uc.PurchaseOrderAttachment
		if err := rows.Scan(&order, &a.ID, &a.FileName, &a.ContentType, &a.FileSize, &a.CreatedAt); err != nil {
			return err
		}
		i := idx[order]
		results[i].Attachments = append(results[i].Attachments, a)
	}
	return rows.Err()
}

func (r *PurchaseOrderRepositorySQLC) GetAttachment(ctx context.Context, enterpriseID, orderCode, attachmentID int64) (*purchase_order_uc.PurchaseOrderAttachmentFile, error) {
	var f purchase_order_uc.PurchaseOrderAttachmentFile
	err := r.db.QueryRow(ctx, `SELECT a.id,a.file_name,a.content_type,a.file_size,a.created_at,a.content FROM purchase_order_attachments a JOIN purchase_orders po ON po.code=a.purchase_order_code JOIN enterprise e ON e.code=po.enterprise_code WHERE e.id=$1 AND po.code=$2 AND a.id=$3 AND po.is_active`, enterpriseID, orderCode, attachmentID).Scan(&f.ID, &f.FileName, &f.ContentType, &f.FileSize, &f.CreatedAt, &f.Content)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, purchase_order_uc.ErrAttachmentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}
