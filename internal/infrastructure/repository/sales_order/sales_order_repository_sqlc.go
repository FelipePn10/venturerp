package sales_order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	repository "github.com/FelipePn10/panossoerp/internal/domain/sales_order/repository"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func toPgDateFromPtr(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

func datePtrToPg(t *time.Time) pgtype.Date {
	return toPgDateFromPtr(t)
}

func timestamptzPtrToPg(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func boolPtrToPg(v *bool) pgtype.Bool {
	if v == nil {
		return pgtype.Bool{Valid: false}
	}
	return pgtype.Bool{Bool: *v, Valid: true}
}

func textFromStatus(v *entity.SalesOrderStatus) pgtype.Text {
	if v == nil {
		return pgtype.Text{Valid: false}
	}
	return pgutil.ToPgText(string(*v))
}

func textFromAnalysisStatus(v *entity.SalesOrderAnalysisStatus) pgtype.Text {
	if v == nil {
		return pgtype.Text{Valid: false}
	}
	return pgutil.ToPgText(string(*v))
}

func textFromReleaseStatus(v *entity.SalesOrderReleaseStatus) pgtype.Text {
	if v == nil {
		return pgtype.Text{Valid: false}
	}
	return pgutil.ToPgText(string(*v))
}

func textFromConferenceStatus(v *entity.SalesOrderConferenceStatus) pgtype.Text {
	if v == nil {
		return pgtype.Text{Valid: false}
	}
	return pgutil.ToPgText(string(*v))
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
		OrderNumber:                 o.OrderNumber,
		EnterpriseCode:              o.EnterpriseCode,
		Status:                      string(o.Status),
		Origin:                      string(o.Origin),
		EmissionDate:                pgutil.ToPgDate(o.EmissionDate),
		DeliveryDate:                toPgDateFromPtr(o.DeliveryDate),
		DeliveryDateFirm:            o.DeliveryDateFirm,
		DigitDate:                   pgutil.ToPgDate(o.DigitDate),
		CustomerCode:                o.CustomerCode,
		BillingAddressCode:          o.BillingAddressCode,
		ShippingAddressCode:         o.ShippingAddressCode,
		RepresentativeCode:          o.RepresentativeCode,
		PlanCode:                    o.PlanCode,
		SalesDivisionCode:           o.SalesDivisionCode,
		CommissionPct:               pgutil.ToPgNumericFromFloat64(o.CommissionPct),
		TaxTypeCode:                 o.TaxTypeCode,
		PresenceIndicator:           pgutil.ToPgTextFromPtr(o.PresenceIndicator),
		SalesChannel:                pgutil.ToPgTextFromPtr(o.SalesChannel),
		DefaultNfType:               pgutil.ToPgTextFromPtr(o.DefaultNFType),
		PriceTableCode:              o.PriceTableCode,
		CurrencyCode:                o.CurrencyCode,
		PaymentTermCode:             o.PaymentTermCode,
		AdditionalDays:              int32(o.AdditionalDays),
		BearerCode:                  o.BearerCode,
		SaleDate:                    toPgDateFromPtr(o.SaleDate),
		TotalWeightNet:              pgutil.ToPgNumericFromFloat64(o.TotalWeightNet),
		TotalWeightGross:            pgutil.ToPgNumericFromFloat64(o.TotalWeightGross),
		TotalGross:                  pgutil.ToPgNumericFromFloat64(o.TotalGross),
		TotalNet:                    pgutil.ToPgNumericFromFloat64(o.TotalNet),
		TotalNetNoSt:                pgutil.ToPgNumericFromFloat64(o.TotalNetNoST),
		TotalWithIpiWithSt:          pgutil.ToPgNumericFromFloat64(o.TotalWithIPIWithST),
		Notes:                       pgutil.ToPgTextFromPtr(o.Notes),
		ObsCustomer:                 pgutil.ToPgTextFromPtr(o.ObsCustomer),
		IsBlocked:                   o.IsBlocked,
		BlockReason:                 pgutil.ToPgTextFromPtr(o.BlockReason),
		IsFirm:                      o.IsFirm,
		RepresentativeOrderNumber:   o.RepresentativeOrderNumber,
		IsNfce:                      o.IsNFCe,
		Street:                      pgutil.ToPgTextFromPtr(o.Street),
		StreetNumber:                pgutil.ToPgTextFromPtr(o.StreetNumber),
		ForeignDocument:             pgutil.ToPgTextFromPtr(o.ForeignDocument),
		CollectionEstablishmentCode: o.CollectionEstablishmentCode,
		NfTypeDescription:           pgutil.ToPgTextFromPtr(o.NFTypeDescription),
		CarrierCode:                 o.CarrierCode,
		FreightType:                 pgutil.ToPgTextFromPtr(o.FreightType),
		FreightValue:                pgutil.ToPgNumericFromFloat64(o.FreightValue),
		InsuranceValue:              pgutil.ToPgNumericFromFloat64(o.InsuranceValue),
		VolumeQuantity:              pgutil.ToPgNumericFromFloat64(o.VolumeQuantity),
		VolumeType:                  pgutil.ToPgTextFromPtr(o.VolumeType),
		NetWeight:                   pgutil.ToPgNumericFromFloat64(o.NetWeight),
		GrossWeight:                 pgutil.ToPgNumericFromFloat64(o.GrossWeight),
		DiscountValue:               pgutil.ToPgNumericFromFloat64(o.DiscountValue),
		SurchargeValue:              pgutil.ToPgNumericFromFloat64(o.SurchargeValue),
		ProjectCode:                 pgutil.ToPgTextFromPtr(o.ProjectCode),
		ProjectName:                 pgutil.ToPgTextFromPtr(o.ProjectName),
		CreatedBy:                   pgutil.ToPgUUID(o.CreatedBy),
	})
	if err != nil {
		return nil, fmt.Errorf("creating sales order: %w", err)
	}
	return rowToEntity(row), nil
}

func (r *SalesOrderRepositorySQLC) Update(ctx context.Context, o *entity.SalesOrder) (*entity.SalesOrder, error) {
	row, err := r.q.UpdateSalesOrder(ctx, sqlc.UpdateSalesOrderParams{
		Code:                        o.Code,
		Status:                      string(o.Status),
		Origin:                      string(o.Origin),
		DeliveryDate:                toPgDateFromPtr(o.DeliveryDate),
		DeliveryDateFirm:            o.DeliveryDateFirm,
		CustomerCode:                o.CustomerCode,
		BillingAddressCode:          o.BillingAddressCode,
		ShippingAddressCode:         o.ShippingAddressCode,
		RepresentativeCode:          o.RepresentativeCode,
		PlanCode:                    o.PlanCode,
		SalesDivisionCode:           o.SalesDivisionCode,
		CommissionPct:               pgutil.ToPgNumericFromFloat64(o.CommissionPct),
		TaxTypeCode:                 o.TaxTypeCode,
		PresenceIndicator:           pgutil.ToPgTextFromPtr(o.PresenceIndicator),
		SalesChannel:                pgutil.ToPgTextFromPtr(o.SalesChannel),
		DefaultNfType:               pgutil.ToPgTextFromPtr(o.DefaultNFType),
		PriceTableCode:              o.PriceTableCode,
		CurrencyCode:                o.CurrencyCode,
		PaymentTermCode:             o.PaymentTermCode,
		AdditionalDays:              int32(o.AdditionalDays),
		BearerCode:                  o.BearerCode,
		SaleDate:                    toPgDateFromPtr(o.SaleDate),
		TotalWeightNet:              pgutil.ToPgNumericFromFloat64(o.TotalWeightNet),
		TotalWeightGross:            pgutil.ToPgNumericFromFloat64(o.TotalWeightGross),
		TotalGross:                  pgutil.ToPgNumericFromFloat64(o.TotalGross),
		TotalNet:                    pgutil.ToPgNumericFromFloat64(o.TotalNet),
		TotalNetNoSt:                pgutil.ToPgNumericFromFloat64(o.TotalNetNoST),
		TotalWithIpiWithSt:          pgutil.ToPgNumericFromFloat64(o.TotalWithIPIWithST),
		Notes:                       pgutil.ToPgTextFromPtr(o.Notes),
		ObsCustomer:                 pgutil.ToPgTextFromPtr(o.ObsCustomer),
		IsFirm:                      o.IsFirm,
		RepresentativeOrderNumber:   o.RepresentativeOrderNumber,
		IsNfce:                      o.IsNFCe,
		Street:                      pgutil.ToPgTextFromPtr(o.Street),
		StreetNumber:                pgutil.ToPgTextFromPtr(o.StreetNumber),
		ForeignDocument:             pgutil.ToPgTextFromPtr(o.ForeignDocument),
		CollectionEstablishmentCode: o.CollectionEstablishmentCode,
		NfTypeDescription:           pgutil.ToPgTextFromPtr(o.NFTypeDescription),
		CarrierCode:                 o.CarrierCode,
		FreightType:                 pgutil.ToPgTextFromPtr(o.FreightType),
		FreightValue:                pgutil.ToPgNumericFromFloat64(o.FreightValue),
		InsuranceValue:              pgutil.ToPgNumericFromFloat64(o.InsuranceValue),
		VolumeQuantity:              pgutil.ToPgNumericFromFloat64(o.VolumeQuantity),
		VolumeType:                  pgutil.ToPgTextFromPtr(o.VolumeType),
		NetWeight:                   pgutil.ToPgNumericFromFloat64(o.NetWeight),
		GrossWeight:                 pgutil.ToPgNumericFromFloat64(o.GrossWeight),
		DiscountValue:               pgutil.ToPgNumericFromFloat64(o.DiscountValue),
		SurchargeValue:              pgutil.ToPgNumericFromFloat64(o.SurchargeValue),
		ProjectCode:                 pgutil.ToPgTextFromPtr(o.ProjectCode),
		ProjectName:                 pgutil.ToPgTextFromPtr(o.ProjectName),
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

func (r *SalesOrderRepositorySQLC) ListAdvanced(ctx context.Context, filter repository.SalesOrderFilter) ([]*entity.SalesOrder, error) {
	rows, err := r.q.ListSalesOrdersAdvanced(ctx, sqlc.ListSalesOrdersAdvancedParams{
		CustomerCode:             filter.CustomerCode,
		RepresentativeCode:       filter.RepresentativeCode,
		PaymentTermCode:          filter.PaymentTermCode,
		Status:                   textFromStatus(filter.Status),
		CommercialAnalysisStatus: textFromAnalysisStatus(filter.CommercialAnalysisStatus),
		FinancialAnalysisStatus:  textFromAnalysisStatus(filter.FinancialAnalysisStatus),
		ReleaseStatus:            textFromReleaseStatus(filter.ReleaseStatus),
		ConferenceStatus:         textFromConferenceStatus(filter.ConferenceStatus),
		IsBlocked:                boolPtrToPg(filter.IsBlocked),
		EmissionFrom:             datePtrToPg(filter.EmissionFrom),
		EmissionTo:               datePtrToPg(filter.EmissionTo),
		DeliveryFrom:             datePtrToPg(filter.DeliveryFrom),
		DeliveryTo:               datePtrToPg(filter.DeliveryTo),
	})
	if err != nil {
		return nil, fmt.Errorf("listing sales orders advanced: %w", err)
	}
	return rowsToEntities(rows), nil
}

func (r *SalesOrderRepositorySQLC) Report(ctx context.Context, filter repository.SalesOrderFilter) (*repository.SalesOrderReport, error) {
	row, err := r.q.SalesOrderReport(ctx, sqlc.SalesOrderReportParams{
		CustomerCode:       filter.CustomerCode,
		RepresentativeCode: filter.RepresentativeCode,
		PaymentTermCode:    filter.PaymentTermCode,
		Status:             textFromStatus(filter.Status),
		EmissionFrom:       datePtrToPg(filter.EmissionFrom),
		EmissionTo:         datePtrToPg(filter.EmissionTo),
		DeliveryFrom:       datePtrToPg(filter.DeliveryFrom),
		DeliveryTo:         datePtrToPg(filter.DeliveryTo),
	})
	if err != nil {
		return nil, fmt.Errorf("sales order report: %w", err)
	}
	return &repository.SalesOrderReport{
		TotalOrders:            row.TotalOrders,
		TotalGross:             pgutil.FromPgNumericToFloat64(row.TotalGross),
		TotalNet:               pgutil.FromPgNumericToFloat64(row.TotalNet),
		OpenCount:              row.OpenCount,
		ConfirmedCount:         row.ConfirmedCount,
		InvoicedCount:          row.InvoicedCount,
		CancelledCount:         row.CancelledCount,
		BlockedCount:           row.BlockedCount,
		CommercialPendingCount: row.CommercialPendingCount,
		FinancialPendingCount:  row.FinancialPendingCount,
		ConferencePendingCount: row.ConferencePendingCount,
		DelayedCount:           row.DelayedCount,
	}, nil
}

func (r *SalesOrderRepositorySQLC) Cancel(ctx context.Context, code int64, reason string, complement *string) error {
	if err := r.q.CancelSalesOrder(ctx, sqlc.CancelSalesOrderParams{
		Code:             code,
		CancelReason:     pgutil.ToPgText(reason),
		CancelComplement: pgutil.ToPgTextFromPtr(complement),
	}); err != nil {
		return fmt.Errorf("cancelling sales order %d: %w", code, err)
	}
	_ = r.insertEvent(ctx, code, "CANCEL", "", reason, complement, nil, nil)
	return nil
}

func (r *SalesOrderRepositorySQLC) Block(ctx context.Context, code int64, reason string) error {
	if err := r.q.BlockSalesOrder(ctx, sqlc.BlockSalesOrderParams{
		Code:        code,
		BlockReason: pgutil.ToPgText(reason),
	}); err != nil {
		return fmt.Errorf("blocking sales order %d: %w", code, err)
	}
	_ = r.insertEvent(ctx, code, "BLOCK", "", reason, nil, nil, nil)
	return nil
}

func (r *SalesOrderRepositorySQLC) Unblock(ctx context.Context, code int64) error {
	if err := r.q.UnblockSalesOrder(ctx, code); err != nil {
		return fmt.Errorf("unblocking sales order %d: %w", code, err)
	}
	_ = r.insertEvent(ctx, code, "UNBLOCK", "", "Desbloqueio manual", nil, nil, nil)
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

func (r *SalesOrderRepositorySQLC) Analyze(ctx context.Context, code int64, area string, status entity.SalesOrderAnalysisStatus, reason string, createdBy uuid.UUID) error {
	if err := r.q.AnalyzeSalesOrder(ctx, sqlc.AnalyzeSalesOrderParams{
		Code:                     code,
		Column2:                  area,
		CommercialAnalysisStatus: string(status),
	}); err != nil {
		return fmt.Errorf("analyzing sales order %d: %w", code, err)
	}
	return r.insertEvent(ctx, code, "ANALYZE", area, reason, nil, nil, &createdBy)
}

func (r *SalesOrderRepositorySQLC) Release(ctx context.Context, code int64, releaseStatus entity.SalesOrderReleaseStatus, reason string, area string, createdBy uuid.UUID) error {
	if err := r.q.ReleaseSalesOrder(ctx, sqlc.ReleaseSalesOrderParams{
		Code:          code,
		ReleaseStatus: string(releaseStatus),
		BlockReason:   pgutil.ToPgText(reason),
	}); err != nil {
		return fmt.Errorf("releasing sales order %d: %w", code, err)
	}
	return r.insertEvent(ctx, code, "RELEASE", area, reason, nil, nil, &createdBy)
}

func (r *SalesOrderRepositorySQLC) Attend(ctx context.Context, code int64, reason string, eventDate *time.Time, createdBy uuid.UUID) error {
	if err := r.q.AttendSalesOrder(ctx, sqlc.AttendSalesOrderParams{
		Code:           code,
		AttendedReason: pgutil.ToPgText(reason),
		AttendedAt:     timestamptzPtrToPg(eventDate),
	}); err != nil {
		return fmt.Errorf("attending sales order %d: %w", code, err)
	}
	return r.insertEvent(ctx, code, "ATTEND", "", reason, nil, eventDate, &createdBy)
}

func (r *SalesOrderRepositorySQLC) Confer(ctx context.Context, code int64, status entity.SalesOrderConferenceStatus, reason string, createdBy uuid.UUID) error {
	if err := r.q.ConferSalesOrder(ctx, sqlc.ConferSalesOrderParams{
		Code:             code,
		ConferenceStatus: string(status),
	}); err != nil {
		return fmt.Errorf("conferencing sales order %d: %w", code, err)
	}
	return r.insertEvent(ctx, code, "CONFER", "LOGISTICS", reason, nil, nil, &createdBy)
}

func (r *SalesOrderRepositorySQLC) SaveDelayReason(ctx context.Context, code int64, reason, action string, createdBy uuid.UUID) error {
	if err := r.q.SaveSalesOrderDelayReason(ctx, sqlc.SaveSalesOrderDelayReasonParams{
		Code:        code,
		DelayReason: pgutil.ToPgText(reason),
		DelayAction: pgutil.ToPgText(action),
	}); err != nil {
		return fmt.Errorf("saving delay reason for sales order %d: %w", code, err)
	}
	return r.insertEvent(ctx, code, "DELAY_REASON", "", reason, &action, nil, &createdBy)
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

func (r *SalesOrderRepositorySQLC) insertEvent(ctx context.Context, code int64, eventType, area, reason string, complement *string, eventDate *time.Time, createdBy *uuid.UUID) error {
	var created pgtype.UUID
	if createdBy != nil {
		created = pgutil.ToPgUUID(*createdBy)
	}
	var areaText pgtype.Text
	if area != "" {
		areaText = pgutil.ToPgText(area)
	}
	return r.q.InsertSalesOrderEvent(ctx, sqlc.InsertSalesOrderEventParams{
		SalesOrderCode: code,
		EventType:      eventType,
		Area:           areaText,
		Reason:         pgutil.ToPgText(reason),
		Complement:     pgutil.ToPgTextFromPtr(complement),
		Column6:        timestamptzPtrToPg(eventDate),
		CreatedBy:      created,
	})
}

func rowToEntity(row sqlc.SalesOrder) *entity.SalesOrder {
	e := &entity.SalesOrder{
		Code:                        row.Code,
		OrderNumber:                 row.OrderNumber,
		EnterpriseCode:              row.EnterpriseCode,
		Status:                      entity.SalesOrderStatus(row.Status),
		Origin:                      entity.SalesOrderOrigin(row.Origin),
		EmissionDate:                pgutil.FromPgDate(row.EmissionDate),
		DeliveryDateFirm:            row.DeliveryDateFirm,
		DigitDate:                   pgutil.FromPgDate(row.DigitDate),
		CustomerCode:                row.CustomerCode,
		BillingAddressCode:          row.BillingAddressCode,
		ShippingAddressCode:         row.ShippingAddressCode,
		RepresentativeCode:          row.RepresentativeCode,
		PlanCode:                    row.PlanCode,
		SalesDivisionCode:           row.SalesDivisionCode,
		CommissionPct:               pgutil.FromPgNumericToFloat64(row.CommissionPct),
		TaxTypeCode:                 row.TaxTypeCode,
		PriceTableCode:              row.PriceTableCode,
		CurrencyCode:                row.CurrencyCode,
		PaymentTermCode:             row.PaymentTermCode,
		AdditionalDays:              int(row.AdditionalDays),
		BearerCode:                  row.BearerCode,
		TotalWeightNet:              pgutil.FromPgNumericToFloat64(row.TotalWeightNet),
		TotalWeightGross:            pgutil.FromPgNumericToFloat64(row.TotalWeightGross),
		TotalGross:                  pgutil.FromPgNumericToFloat64(row.TotalGross),
		TotalNet:                    pgutil.FromPgNumericToFloat64(row.TotalNet),
		TotalNetNoST:                pgutil.FromPgNumericToFloat64(row.TotalNetNoSt),
		TotalWithIPIWithST:          pgutil.FromPgNumericToFloat64(row.TotalWithIpiWithSt),
		IsBlocked:                   row.IsBlocked,
		IsFirm:                      row.IsFirm,
		IsActive:                    row.IsActive,
		RepresentativeOrderNumber:   row.RepresentativeOrderNumber,
		IsNFCe:                      row.IsNfce,
		CollectionEstablishmentCode: row.CollectionEstablishmentCode,
		CarrierCode:                 row.CarrierCode,
		FreightValue:                pgutil.FromPgNumericToFloat64(row.FreightValue),
		InsuranceValue:              pgutil.FromPgNumericToFloat64(row.InsuranceValue),
		VolumeQuantity:              pgutil.FromPgNumericToFloat64(row.VolumeQuantity),
		NetWeight:                   pgutil.FromPgNumericToFloat64(row.NetWeight),
		GrossWeight:                 pgutil.FromPgNumericToFloat64(row.GrossWeight),
		DiscountValue:               pgutil.FromPgNumericToFloat64(row.DiscountValue),
		SurchargeValue:              pgutil.FromPgNumericToFloat64(row.SurchargeValue),
		CommercialAnalysisStatus:    entity.SalesOrderAnalysisStatus(row.CommercialAnalysisStatus),
		FinancialAnalysisStatus:     entity.SalesOrderAnalysisStatus(row.FinancialAnalysisStatus),
		ReleaseStatus:               entity.SalesOrderReleaseStatus(row.ReleaseStatus),
		ConferenceStatus:            entity.SalesOrderConferenceStatus(row.ConferenceStatus),
		CreatedAt:                   pgutil.FromPgTimestamptz(row.CreatedAt),
		UpdatedAt:                   pgutil.FromPgTimestamptz(row.UpdatedAt),
		CreatedBy:                   pgutil.FromPgUUID(row.CreatedBy),
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
	if row.Street.Valid {
		v := row.Street.String
		e.Street = &v
	}
	if row.StreetNumber.Valid {
		v := row.StreetNumber.String
		e.StreetNumber = &v
	}
	if row.ForeignDocument.Valid {
		v := row.ForeignDocument.String
		e.ForeignDocument = &v
	}
	if row.NfTypeDescription.Valid {
		v := row.NfTypeDescription.String
		e.NFTypeDescription = &v
	}
	if row.FreightType.Valid {
		v := row.FreightType.String
		e.FreightType = &v
	}
	if row.VolumeType.Valid {
		v := row.VolumeType.String
		e.VolumeType = &v
	}
	if row.ProjectCode.Valid {
		v := row.ProjectCode.String
		e.ProjectCode = &v
	}
	if row.ProjectName.Valid {
		v := row.ProjectName.String
		e.ProjectName = &v
	}
	if row.CancelReason.Valid {
		v := row.CancelReason.String
		e.CancelReason = &v
	}
	if row.CancelComplement.Valid {
		v := row.CancelComplement.String
		e.CancelComplement = &v
	}
	if row.AttendedReason.Valid {
		v := row.AttendedReason.String
		e.AttendedReason = &v
	}
	if row.AttendedAt.Valid {
		t := pgutil.FromPgTimestamptz(row.AttendedAt)
		e.AttendedAt = &t
	}
	if row.DelayReason.Valid {
		v := row.DelayReason.String
		e.DelayReason = &v
	}
	if row.DelayAction.Valid {
		v := row.DelayAction.String
		e.DelayAction = &v
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
