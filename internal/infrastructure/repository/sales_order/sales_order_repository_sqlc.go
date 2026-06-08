package sales_order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func toPgDateFromPtr(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func (r *SalesOrderRepositorySQLC) NextOrderNumber(ctx context.Context, enterpriseCode int64) (int64, error) {
	n, err := r.q.NextSalesOrderNumber(ctx, enterpriseCode)
	if err != nil {
		return 0, fmt.Errorf("next sales order number: %w", err)
	}
	return n, nil
}

func (r *SalesOrderRepositorySQLC) Create(ctx context.Context, o *entity.SalesOrder) (*entity.SalesOrder, error) {
	row, err := r.q.CreateSalesOrder(ctx, sqlc.CreateSalesOrderParams{
		OrderNumber:         o.OrderNumber,
		EnterpriseCode:      o.EnterpriseCode,
		Status:              string(o.Status),
		Origin:              string(o.Origin),
		EmissionDate:        pgutil.ToPgDate(o.EmissionDate),
		DeliveryDate:        toPgDateFromPtr(o.DeliveryDate),
		DeliveryDateFirm:    o.DeliveryDateFirm,
		DigitDate:           pgutil.ToPgDate(o.DigitDate),
		CustomerCode:        o.CustomerCode,
		BillingAddressCode:  o.BillingAddressCode,
		ShippingAddressCode: o.ShippingAddressCode,
		RepresentativeCode:  o.RepresentativeCode,
		PlanCode:            o.PlanCode,
		SalesDivisionCode:   o.SalesDivisionCode,
		CommissionPct:       pgutil.ToPgNumericFromFloat64(o.CommissionPct),
		TaxTypeCode:         o.TaxTypeCode,
		PresenceIndicator:   pgutil.ToPgTextFromPtr(o.PresenceIndicator),
		SalesChannel:        pgutil.ToPgTextFromPtr(o.SalesChannel),
		DefaultNfType:       pgutil.ToPgTextFromPtr(o.DefaultNFType),
		PriceTableCode:      o.PriceTableCode,
		CurrencyCode:        o.CurrencyCode,
		PaymentTermCode:     o.PaymentTermCode,
		AdditionalDays:      int32(o.AdditionalDays),
		BearerCode:          o.BearerCode,
		SaleDate:            toPgDateFromPtr(o.SaleDate),
		TotalWeightNet:      pgutil.ToPgNumericFromFloat64(o.TotalWeightNet),
		TotalWeightGross:    pgutil.ToPgNumericFromFloat64(o.TotalWeightGross),
		TotalGross:          pgutil.ToPgNumericFromFloat64(o.TotalGross),
		TotalNet:            pgutil.ToPgNumericFromFloat64(o.TotalNet),
		TotalNetNoSt:        pgutil.ToPgNumericFromFloat64(o.TotalNetNoST),
		TotalWithIpiWithSt:  pgutil.ToPgNumericFromFloat64(o.TotalWithIPIWithST),
		Notes:               pgutil.ToPgTextFromPtr(o.Notes),
		ObsCustomer:         pgutil.ToPgTextFromPtr(o.ObsCustomer),
		IsBlocked:           o.IsBlocked,
		BlockReason:         pgutil.ToPgTextFromPtr(o.BlockReason),
		IsFirm:              o.IsFirm,
		CreatedBy:           pgutil.ToPgUUID(o.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating sales order: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *SalesOrderRepositorySQLC) Update(ctx context.Context, o *entity.SalesOrder) (*entity.SalesOrder, error) {
	row, err := r.q.UpdateSalesOrder(ctx, sqlc.UpdateSalesOrderParams{
		Code:                o.Code,
		Status:              string(o.Status),
		Origin:              string(o.Origin),
		DeliveryDate:        toPgDateFromPtr(o.DeliveryDate),
		DeliveryDateFirm:    o.DeliveryDateFirm,
		CustomerCode:        o.CustomerCode,
		BillingAddressCode:  o.BillingAddressCode,
		ShippingAddressCode: o.ShippingAddressCode,
		RepresentativeCode:  o.RepresentativeCode,
		PlanCode:            o.PlanCode,
		SalesDivisionCode:   o.SalesDivisionCode,
		CommissionPct:       pgutil.ToPgNumericFromFloat64(o.CommissionPct),
		TaxTypeCode:         o.TaxTypeCode,
		PresenceIndicator:   pgutil.ToPgTextFromPtr(o.PresenceIndicator),
		SalesChannel:        pgutil.ToPgTextFromPtr(o.SalesChannel),
		DefaultNfType:       pgutil.ToPgTextFromPtr(o.DefaultNFType),
		PriceTableCode:      o.PriceTableCode,
		CurrencyCode:        o.CurrencyCode,
		PaymentTermCode:     o.PaymentTermCode,
		AdditionalDays:      int32(o.AdditionalDays),
		BearerCode:          o.BearerCode,
		SaleDate:            toPgDateFromPtr(o.SaleDate),
		TotalWeightNet:      pgutil.ToPgNumericFromFloat64(o.TotalWeightNet),
		TotalWeightGross:    pgutil.ToPgNumericFromFloat64(o.TotalWeightGross),
		TotalGross:          pgutil.ToPgNumericFromFloat64(o.TotalGross),
		TotalNet:            pgutil.ToPgNumericFromFloat64(o.TotalNet),
		TotalNetNoSt:        pgutil.ToPgNumericFromFloat64(o.TotalNetNoST),
		TotalWithIpiWithSt:  pgutil.ToPgNumericFromFloat64(o.TotalWithIPIWithST),
		Notes:               pgutil.ToPgTextFromPtr(o.Notes),
		ObsCustomer:         pgutil.ToPgTextFromPtr(o.ObsCustomer),
		IsFirm:              o.IsFirm,
	})
	if err != nil {
		return nil, fmt.Errorf("updating sales order: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *SalesOrderRepositorySQLC) GetByCode(ctx context.Context, code int64) (*entity.SalesOrder, error) {
	row, err := r.q.GetSalesOrderByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("sales order %d not found", code)
		}
		return nil, fmt.Errorf("fetching sales order: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *SalesOrderRepositorySQLC) List(ctx context.Context) ([]*entity.SalesOrder, error) {
	rows, err := r.q.ListSalesOrders(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing sales orders: %w", err)
	}
	return rowsToEntities(rows), nil
}

func (r *SalesOrderRepositorySQLC) ListByCustomer(ctx context.Context, customerCode int64) ([]*entity.SalesOrder, error) {
	rows, err := r.q.ListSalesOrdersByCustomer(ctx, &customerCode)
	if err != nil {
		return nil, fmt.Errorf("listing sales orders by customer: %w", err)
	}
	return rowsToEntities(rows), nil
}

func (r *SalesOrderRepositorySQLC) ListByStatus(ctx context.Context, status entity.SalesOrderStatus) ([]*entity.SalesOrder, error) {
	rows, err := r.q.ListSalesOrdersByStatus(ctx, string(status))
	if err != nil {
		return nil, fmt.Errorf("listing sales orders by status: %w", err)
	}
	return rowsToEntities(rows), nil
}

func (r *SalesOrderRepositorySQLC) ListByDateRange(ctx context.Context, from, to time.Time) ([]*entity.SalesOrder, error) {
	rows, err := r.q.ListSalesOrdersByDateRange(ctx, sqlc.ListSalesOrdersByDateRangeParams{
		EmissionDate:   pgutil.ToPgDate(from),
		EmissionDate_2: pgutil.ToPgDate(to),
	})
	if err != nil {
		return nil, fmt.Errorf("listing sales orders by date range: %w", err)
	}
	return rowsToEntities(rows), nil
}

func (r *SalesOrderRepositorySQLC) Cancel(ctx context.Context, code int64) error {
	if err := r.q.CancelSalesOrder(ctx, code); err != nil {
		return fmt.Errorf("cancelling sales order %d: %w", code, err)
	}
	return nil
}

func (r *SalesOrderRepositorySQLC) Block(ctx context.Context, code int64, reason string) error {
	if err := r.q.BlockSalesOrder(ctx, sqlc.BlockSalesOrderParams{
		Code:        code,
		BlockReason: pgutil.ToPgText(reason),
	}); err != nil {
		return fmt.Errorf("blocking sales order %d: %w", code, err)
	}
	return nil
}

func (r *SalesOrderRepositorySQLC) Unblock(ctx context.Context, code int64) error {
	if err := r.q.UnblockSalesOrder(ctx, code); err != nil {
		return fmt.Errorf("unblocking sales order %d: %w", code, err)
	}
	return nil
}

func (r *SalesOrderRepositorySQLC) ChangeStatus(ctx context.Context, code int64, status entity.SalesOrderStatus) error {
	if err := r.q.ChangeSalesOrderStatus(ctx, sqlc.ChangeSalesOrderStatusParams{
		Code:   code,
		Status: string(status),
	}); err != nil {
		return fmt.Errorf("changing status of sales order %d: %w", code, err)
	}
	return nil
}

func (r *SalesOrderRepositorySQLC) CreateItem(ctx context.Context, item *entity.SalesOrderItem) (*entity.SalesOrderItem, error) {
	row, err := r.q.CreateSalesOrderItem(ctx, sqlc.CreateSalesOrderItemParams{
		SalesOrderCode:   item.SalesOrderCode,
		Sequence:         int32(item.Sequence),
		ItemCode:         item.ItemCode,
		Mask:             item.Mask,
		DigitDate:        pgutil.ToPgDate(item.DigitDate),
		NfType:           pgutil.ToPgTextFromPtr(item.NFType),
		SalesUom:         pgutil.ToPgTextFromPtr(item.SalesUOM),
		WarehouseCode:    item.WarehouseCode,
		PriceTableCode:   item.PriceTableCode,
		RequestedQty:     pgutil.ToPgNumericFromFloat64(item.RequestedQty),
		UnitPrice:        pgutil.ToPgNumericFromFloat64(item.UnitPrice),
		AttendedQty:      pgutil.ToPgNumericFromFloat64(item.AttendedQty),
		CancelledQty:     pgutil.ToPgNumericFromFloat64(item.CancelledQty),
		DeliveryDate:     toPgDateFromPtr(item.DeliveryDate),
		DeliveryDateFirm: item.DeliveryDateFirm,
		CustomerDelivery: pgutil.ToPgTextFromPtr(item.CustomerDelivery),
		Lot:              pgutil.ToPgTextFromPtr(item.Lot),
		CouponDelivery:   pgutil.ToPgTextFromPtr(item.CouponDelivery),
		PaidAtCashier:    item.PaidAtCashier,
		IpiPct:           pgutil.ToPgNumericFromFloat64(item.IPIPct),
		IcmsPct:          pgutil.ToPgNumericFromFloat64(item.ICMSPct),
		PisPct:           pgutil.ToPgNumericFromFloat64(item.PISPct),
		CofinsPct:        pgutil.ToPgNumericFromFloat64(item.COFINSPct),
		StPct:            pgutil.ToPgNumericFromFloat64(item.STPct),
		DiscountPct:      pgutil.ToPgNumericFromFloat64(item.DiscountPct),
		TotalGross:       pgutil.ToPgNumericFromFloat64(item.TotalGross),
		TotalNet:         pgutil.ToPgNumericFromFloat64(item.TotalNet),
		TotalNetWithIpi:  pgutil.ToPgNumericFromFloat64(item.TotalNetWithIPI),
		TotalIpi:         pgutil.ToPgNumericFromFloat64(item.TotalIPI),
		TotalSt:          pgutil.ToPgNumericFromFloat64(item.TotalST),
		UnitWeightNet:    pgutil.ToPgNumericFromFloat64(item.UnitWeightNet),
		UnitWeightGross:  pgutil.ToPgNumericFromFloat64(item.UnitWeightGross),
		Status:           string(item.Status),
		Notes:            pgutil.ToPgTextFromPtr(item.Notes),
	})
	if err != nil {
		return nil, fmt.Errorf("creating sales order item: %w", err)
	}
	return rowItemToEntity(row), nil
}

func (r *SalesOrderRepositorySQLC) UpdateItem(ctx context.Context, item *entity.SalesOrderItem) (*entity.SalesOrderItem, error) {
	row, err := r.q.UpdateSalesOrderItem(ctx, sqlc.UpdateSalesOrderItemParams{
		Code:             item.Code,
		RequestedQty:     pgutil.ToPgNumericFromFloat64(item.RequestedQty),
		UnitPrice:        pgutil.ToPgNumericFromFloat64(item.UnitPrice),
		AttendedQty:      pgutil.ToPgNumericFromFloat64(item.AttendedQty),
		CancelledQty:     pgutil.ToPgNumericFromFloat64(item.CancelledQty),
		DeliveryDate:     toPgDateFromPtr(item.DeliveryDate),
		DeliveryDateFirm: item.DeliveryDateFirm,
		CustomerDelivery: pgutil.ToPgTextFromPtr(item.CustomerDelivery),
		Lot:              pgutil.ToPgTextFromPtr(item.Lot),
		CouponDelivery:   pgutil.ToPgTextFromPtr(item.CouponDelivery),
		PaidAtCashier:    item.PaidAtCashier,
		IpiPct:           pgutil.ToPgNumericFromFloat64(item.IPIPct),
		IcmsPct:          pgutil.ToPgNumericFromFloat64(item.ICMSPct),
		PisPct:           pgutil.ToPgNumericFromFloat64(item.PISPct),
		CofinsPct:        pgutil.ToPgNumericFromFloat64(item.COFINSPct),
		StPct:            pgutil.ToPgNumericFromFloat64(item.STPct),
		DiscountPct:      pgutil.ToPgNumericFromFloat64(item.DiscountPct),
		TotalGross:       pgutil.ToPgNumericFromFloat64(item.TotalGross),
		TotalNet:         pgutil.ToPgNumericFromFloat64(item.TotalNet),
		TotalNetWithIpi:  pgutil.ToPgNumericFromFloat64(item.TotalNetWithIPI),
		TotalIpi:         pgutil.ToPgNumericFromFloat64(item.TotalIPI),
		TotalSt:          pgutil.ToPgNumericFromFloat64(item.TotalST),
		UnitWeightNet:    pgutil.ToPgNumericFromFloat64(item.UnitWeightNet),
		UnitWeightGross:  pgutil.ToPgNumericFromFloat64(item.UnitWeightGross),
		Status:           string(item.Status),
		Notes:            pgutil.ToPgTextFromPtr(item.Notes),
	})
	if err != nil {
		return nil, fmt.Errorf("updating sales order item: %w", err)
	}
	return rowItemToEntity(row), nil
}

func (r *SalesOrderRepositorySQLC) ListItems(ctx context.Context, salesOrderCode int64) ([]*entity.SalesOrderItem, error) {
	rows, err := r.q.ListSalesOrderItems(ctx, salesOrderCode)
	if err != nil {
		return nil, fmt.Errorf("listing sales order items: %w", err)
	}
	out := make([]*entity.SalesOrderItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowItemToEntity(row))
	}
	return out, nil
}

func (r *SalesOrderRepositorySQLC) CancelItem(ctx context.Context, itemCode int64) error {
	if err := r.q.CancelSalesOrderItem(ctx, itemCode); err != nil {
		return fmt.Errorf("cancelling sales order item %d: %w", itemCode, err)
	}
	return nil
}

func rowToEntity(row sqlc.SalesOrder) *entity.SalesOrder {
	e := &entity.SalesOrder{
		Code:                row.Code,
		OrderNumber:         row.OrderNumber,
		EnterpriseCode:      row.EnterpriseCode,
		Status:              entity.SalesOrderStatus(row.Status),
		Origin:              entity.SalesOrderOrigin(row.Origin),
		EmissionDate:        pgutil.FromPgDate(row.EmissionDate),
		DeliveryDateFirm:    row.DeliveryDateFirm,
		DigitDate:           pgutil.FromPgDate(row.DigitDate),
		CustomerCode:        row.CustomerCode,
		BillingAddressCode:  row.BillingAddressCode,
		ShippingAddressCode: row.ShippingAddressCode,
		RepresentativeCode:  row.RepresentativeCode,
		PlanCode:            row.PlanCode,
		SalesDivisionCode:   row.SalesDivisionCode,
		CommissionPct:       pgutil.FromPgNumericToFloat64(row.CommissionPct),
		TaxTypeCode:         row.TaxTypeCode,
		PriceTableCode:      row.PriceTableCode,
		CurrencyCode:        row.CurrencyCode,
		PaymentTermCode:     row.PaymentTermCode,
		AdditionalDays:      int(row.AdditionalDays),
		BearerCode:          row.BearerCode,
		TotalWeightNet:      pgutil.FromPgNumericToFloat64(row.TotalWeightNet),
		TotalWeightGross:    pgutil.FromPgNumericToFloat64(row.TotalWeightGross),
		TotalGross:          pgutil.FromPgNumericToFloat64(row.TotalGross),
		TotalNet:            pgutil.FromPgNumericToFloat64(row.TotalNet),
		TotalNetNoST:        pgutil.FromPgNumericToFloat64(row.TotalNetNoSt),
		TotalWithIPIWithST:  pgutil.FromPgNumericToFloat64(row.TotalWithIpiWithSt),
		IsBlocked:           row.IsBlocked,
		IsFirm:              row.IsFirm,
		IsActive:            row.IsActive,
		CreatedAt:           pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:           pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:           pgutil.FromPgUUID(row.CreatedBy),
	}

	if row.DeliveryDate.Valid {
		t := pgutil.FromPgDate(row.DeliveryDate)
		e.DeliveryDate = &t
	}
	if row.SaleDate.Valid {
		t := pgutil.FromPgDate(row.SaleDate)
		e.SaleDate = &t
	}
	if row.PresenceIndicator.Valid {
		v := row.PresenceIndicator.String
		e.PresenceIndicator = &v
	}
	if row.SalesChannel.Valid {
		v := row.SalesChannel.String
		e.SalesChannel = &v
	}
	if row.DefaultNfType.Valid {
		v := row.DefaultNfType.String
		e.DefaultNFType = &v
	}
	if row.Notes.Valid {
		v := row.Notes.String
		e.Notes = &v
	}
	if row.ObsCustomer.Valid {
		v := row.ObsCustomer.String
		e.ObsCustomer = &v
	}
	if row.BlockReason.Valid {
		v := row.BlockReason.String
		e.BlockReason = &v
	}

	return e
}

func rowsToEntities(rows []sqlc.SalesOrder) []*entity.SalesOrder {
	out := make([]*entity.SalesOrder, 0, len(rows))
	for _, row := range rows {
		out = append(out, rowToEntity(row))
	}
	return out
}

func rowItemToEntity(row sqlc.SalesOrderItem) *entity.SalesOrderItem {
	item := &entity.SalesOrderItem{
		Code:             row.Code,
		SalesOrderCode:   row.SalesOrderCode,
		Sequence:         int(row.Sequence),
		ItemCode:         row.ItemCode,
		Mask:             row.Mask,
		DigitDate:        pgutil.FromPgDate(row.DigitDate),
		WarehouseCode:    row.WarehouseCode,
		PriceTableCode:   row.PriceTableCode,
		RequestedQty:     pgutil.FromPgNumericToFloat64(row.RequestedQty),
		UnitPrice:        pgutil.FromPgNumericToFloat64(row.UnitPrice),
		AttendedQty:      pgutil.FromPgNumericToFloat64(row.AttendedQty),
		CancelledQty:     pgutil.FromPgNumericToFloat64(row.CancelledQty),
		DeliveryDateFirm: row.DeliveryDateFirm,
		PaidAtCashier:    row.PaidAtCashier,
		IPIPct:           pgutil.FromPgNumericToFloat64(row.IpiPct),
		ICMSPct:          pgutil.FromPgNumericToFloat64(row.IcmsPct),
		PISPct:           pgutil.FromPgNumericToFloat64(row.PisPct),
		COFINSPct:        pgutil.FromPgNumericToFloat64(row.CofinsPct),
		STPct:            pgutil.FromPgNumericToFloat64(row.StPct),
		DiscountPct:      pgutil.FromPgNumericToFloat64(row.DiscountPct),
		TotalGross:       pgutil.FromPgNumericToFloat64(row.TotalGross),
		TotalNet:         pgutil.FromPgNumericToFloat64(row.TotalNet),
		TotalNetWithIPI:  pgutil.FromPgNumericToFloat64(row.TotalNetWithIpi),
		TotalIPI:         pgutil.FromPgNumericToFloat64(row.TotalIpi),
		TotalST:          pgutil.FromPgNumericToFloat64(row.TotalSt),
		UnitWeightNet:    pgutil.FromPgNumericToFloat64(row.UnitWeightNet),
		UnitWeightGross:  pgutil.FromPgNumericToFloat64(row.UnitWeightGross),
		Status:           entity.SalesOrderItemStatus(row.Status),
		IsActive:         row.IsActive,
		CreatedAt:        pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:        pgutil.FromPgTimestamptz(row.UpdatedAt),
	}

	// computed balance
	item.Balance = item.RequestedQty - item.AttendedQty - item.CancelledQty

	if row.DeliveryDate.Valid {
		t := pgutil.FromPgDate(row.DeliveryDate)
		item.DeliveryDate = &t
	}
	if row.NfType.Valid {
		v := row.NfType.String
		item.NFType = &v
	}
	if row.SalesUom.Valid {
		v := row.SalesUom.String
		item.SalesUOM = &v
	}
	if row.CustomerDelivery.Valid {
		v := row.CustomerDelivery.String
		item.CustomerDelivery = &v
	}
	if row.Lot.Valid {
		v := row.Lot.String
		item.Lot = &v
	}
	if row.CouponDelivery.Valid {
		v := row.CouponDelivery.String
		item.CouponDelivery = &v
	}
	if row.Notes.Valid {
		v := row.Notes.String
		item.Notes = &v
	}

	return item
}
