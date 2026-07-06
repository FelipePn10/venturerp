package recurring_sales_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/entity"
	orderentity "github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
	"github.com/google/uuid"
)

type SalesOrderCreator interface {
	Execute(context.Context, request.CreateSalesOrderDTO) (*response.SalesOrderResponse, error)
}

type SalesOrderItemCreator interface {
	Execute(context.Context, request.CreateSalesOrderItemDTO) (*response.SalesOrderItemResponse, error)
}

func (uc *UseCase) GenerateSalesOrder(ctx context.Context, code int64, dto request.MarkRecurringSaleOrderDTO) (*response.RecurringSaleResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	if dto.OrderCode != 0 {
		return uc.MarkOrderGenerated(ctx, code, dto.OrderCode)
	}
	if uc.SalesOrders == nil || uc.SalesOrderItems == nil {
		return nil, errorsuc.NewValidationError("sales order generation is not configured")
	}
	rec, err := uc.Repo.Get(ctx, code)
	if err != nil {
		return nil, err
	}
	if rec.GeneratedOrderCode != nil {
		return nil, errorsuc.NewValidationError("recurring sale already has a generated order")
	}
	if rec.MovementType != entity.MovementSale && rec.MovementType != entity.MovementUpgrade && rec.MovementType != entity.MovementAdjustment {
		return nil, errorsuc.NewValidationError("only SALE, UPGRADE and ADJUSTMENT movements can generate sales orders")
	}
	primary := primaryRepresentative(rec)
	if primary == nil {
		return nil, errorsuc.NewValidationError("primary representative is required to generate sales order")
	}
	params := defaultParameters(rec.EnterpriseCode, dto.CreatedBy)
	if p, err := uc.Repo.GetParameters(ctx, rec.EnterpriseCode); err == nil && p != nil {
		params = p
	}
	lines, err := buildOrderLines(rec, params)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, errorsuc.NewValidationError("no recurring billing lines were generated")
	}
	createdBy := dto.CreatedBy
	if createdBy == uuid.Nil {
		createdBy = rec.CreatedBy
	}
	status := dto.Status
	if status == "" {
		status = string(orderentity.SalesOrderStatusDraft)
	}
	if dto.ConfirmOrder {
		status = string(orderentity.SalesOrderStatusOrder)
	}
	notes := fmt.Sprintf("Pedido gerado automaticamente pela venda recorrente %d.", rec.Code)
	order, err := uc.SalesOrders.Execute(ctx, request.CreateSalesOrderDTO{
		EnterpriseCode: rec.EnterpriseCode, Status: status, Origin: string(orderentity.SalesOrderOriginNormal),
		EmissionDate: dateStringOrDefault(dto.EmissionDate, rec.SaleDate), DeliveryDate: &lines[0].deliveryDate,
		DeliveryDateFirm: true, CustomerCode: &rec.CustomerCode, RepresentativeCode: &primary.RepresentativeCode,
		PlanCode: rec.SalesPlanCode, SalesDivisionCode: dto.SalesDivisionCode, CommissionPct: primary.CommissionPercent,
		PriceTableCode: dto.PriceTableCode, CurrencyCode: "BRL", PaymentTermCode: dto.PaymentTermCode,
		SaleDate: stringPtr(rec.SaleDate.Format("2006-01-02")), Notes: &notes, CreatedBy: createdBy,
	})
	if err != nil {
		return nil, err
	}
	for _, line := range lines {
		_, err := uc.SalesOrderItems.Execute(ctx, request.CreateSalesOrderItemDTO{
			SalesOrderCode: order.Code, Sequence: line.sequence, ItemCode: rec.ItemCode, Mask: stringValue(rec.ItemMask),
			DigitDate: order.EmissionDate.Format("2006-01-02"), SalesUOM: dto.SalesUOM, WarehouseCode: dto.WarehouseCode,
			PriceTableCode: dto.PriceTableCode, RequestedQty: line.quantity, UnitPrice: line.unitValue,
			DeliveryDate: &line.deliveryDate, DeliveryDateFirm: true, Notes: &line.notes,
		})
		if err != nil {
			return nil, err
		}
	}
	row, err := uc.Repo.MarkOrderGenerated(ctx, rec.Code, order.Code)
	if err != nil {
		return nil, err
	}
	return toRecurringSaleResponse(row), nil
}

type orderLine struct {
	sequence     int
	deliveryDate string
	quantity     float64
	unitValue    float64
	notes        string
}

func buildOrderLines(rec *entity.RecurringSale, params *entity.Parameters) ([]orderLine, error) {
	if rec.TermType == entity.TermFixed {
		return buildFixedLines(rec, params)
	}
	return buildIndefiniteLines(rec, params)
}

func buildIndefiniteLines(rec *entity.RecurringSale, params *entity.Parameters) ([]orderLine, error) {
	if rec.NextAdjustmentDate == nil {
		return nil, errorsuc.NewValidationError("next_adjustment_date is required to generate indefinite recurring order")
	}
	start := firstBillingMonth(rec.SaleDate, params.CurrentMonthBillingLimitDay)
	end := monthStart(*rec.NextAdjustmentDate)
	out := make([]orderLine, 0)
	seq := 1
	for m := start; m.Before(end); m = m.AddDate(0, 1, 0) {
		qty, unit := rec.Quantity, rec.UnitValue
		if params.GroupOrderItemTotal {
			qty = 1
			unit = monthlyValue(rec)
		}
		out = append(out, orderLine{
			sequence: seq, deliveryDate: dateWithDay(m, params.IndefiniteDeliveryDay).Format("2006-01-02"),
			quantity: qty, unitValue: unit, notes: fmt.Sprintf("Recorrencia %d - competencia %s", rec.Code, m.Format("2006-01")),
		})
		seq++
	}
	return out, nil
}

func buildFixedLines(rec *entity.RecurringSale, params *entity.Parameters) ([]orderLine, error) {
	if rec.PaymentsQuantity == nil || rec.PaymentValue == nil {
		return nil, errorsuc.NewValidationError("fixed recurring sale requires payments_quantity and payment_value")
	}
	payments := *rec.PaymentsQuantity - rec.GraceMonths
	if payments <= 0 {
		return nil, errorsuc.NewValidationError("payments_quantity must be greater than grace_months")
	}
	start := monthStart(rec.SaleDate).AddDate(0, rec.GraceMonths, 0)
	out := make([]orderLine, 0, payments)
	for i := 0; i < payments; i++ {
		m := start.AddDate(0, i, 0)
		out = append(out, orderLine{
			sequence: i + 1, deliveryDate: dateWithDay(m, params.FixedTermDeliveryDay).Format("2006-01-02"),
			quantity: 1, unitValue: *rec.PaymentValue, notes: fmt.Sprintf("Recorrencia %d - parcela %d/%d", rec.Code, i+1, payments),
		})
	}
	return out, nil
}

func primaryRepresentative(rec *entity.RecurringSale) *entity.Representative {
	for _, rep := range rec.Representatives {
		if rep.IsPrimary {
			return rep
		}
	}
	return nil
}

func defaultParameters(enterpriseCode int64, updatedBy uuid.UUID) *entity.Parameters {
	return &entity.Parameters{
		EnterpriseCode: enterpriseCode, CurrentMonthBillingLimitDay: 10,
		IndefiniteDeliveryDay: 10, FixedTermDeliveryDay: 10, UpdatedBy: updatedBy,
	}
}

func firstBillingMonth(saleDate time.Time, limitDay int) time.Time {
	m := monthStart(saleDate)
	if saleDate.Day() > limitDay {
		return m.AddDate(0, 1, 0)
	}
	return m
}

func dateWithDay(month time.Time, day int) time.Time {
	last := time.Date(month.Year(), month.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if day > last {
		day = last
	}
	if day < 1 {
		day = 1
	}
	return time.Date(month.Year(), month.Month(), day, 0, 0, 0, 0, time.UTC)
}

func dateStringOrDefault(raw string, fallback time.Time) string {
	if raw != "" {
		return raw
	}
	return fallback.Format("2006-01-02")
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func stringPtr(v string) *string { return &v }
