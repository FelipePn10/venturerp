package recurring_sales_uc

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/representativevalidation"
	"github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/entity"
	rsrepo "github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/repository"
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
		return nil, errorsuc.NewValidationError("a geração de pedido não está configurada")
	}
	rec, err := uc.Repo.Get(ctx, code)
	if err != nil {
		return nil, err
	}
	if rec.MovementType != entity.MovementSale && rec.MovementType != entity.MovementUpgrade && rec.MovementType != entity.MovementAdjustment {
		return nil, errorsuc.NewValidationError("somente venda, upgrade e reajuste podem gerar pedidos")
	}
	primary := primaryRepresentative(rec)
	if primary == nil {
		return nil, errorsuc.NewValidationError("o representante principal é obrigatório para gerar o pedido")
	}
	if err := representativevalidation.ValidateRequired(ctx, uc.Representatives, primary.RepresentativeCode); err != nil {
		return nil, err
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, errorsuc.ErrUnauthorized
	}
	operationRepo, ok := uc.Repo.(rsrepo.OperationRepository)
	if !ok {
		return nil, errorsuc.NewValidationError("o controle idempotente da geração recorrente não está configurado")
	}
	dto.IdempotencyKey = strings.TrimSpace(dto.IdempotencyKey)
	competence := strings.TrimSpace(dto.Competence)
	if competence == "" {
		emission, parseErr := time.Parse("2006-01-02", dateStringOrDefault(dto.EmissionDate, rec.SaleDate))
		if parseErr != nil {
			return nil, errorsuc.NewValidationError("emission_date deve usar o formato AAAA-MM-DD")
		}
		competence = emission.Format("2006-01")
	}
	if _, err = time.Parse("2006-01", competence); err != nil {
		return nil, errorsuc.NewValidationError("competence deve usar o formato AAAA-MM")
	}
	if dto.IdempotencyKey == "" {
		// Compatibilidade temporária com clientes publicados: a competência forma
		// uma chave determinística até todos enviarem o cabeçalho canônico.
		dto.IdempotencyKey = fmt.Sprintf("legacy:recurring:%d:order:%s", rec.Code, competence)
	}
	payload, _ := json.Marshal(dto)
	digest := sha256.Sum256(payload)
	operation := &entity.Operation{RecurringSaleCode: rec.Code, OperationType: "PEDIDO", Competence: competence,
		IdempotencyKey: dto.IdempotencyKey, RequestHash: fmt.Sprintf("%x", digest), ActorID: actor}
	reserved, replay, err := operationRepo.ReserveOperation(ctx, operation)
	if errors.Is(err, rsrepo.ErrOperationConflict) {
		return nil, errorsuc.NewConflictError("já existe geração para esta competência ou a chave de idempotência foi reutilizada com outro conteúdo")
	}
	if errors.Is(err, rsrepo.ErrOperationInProgress) {
		return nil, errorsuc.NewConflictError("a geração desta competência já está em processamento")
	}
	if err != nil {
		return nil, err
	}
	if replay {
		row, getErr := uc.Repo.Get(ctx, rec.Code)
		if getErr != nil {
			return nil, getErr
		}
		return toRecurringSaleResponse(row), nil
	}
	operation = reserved
	completed := false
	defer func() {
		if !completed {
			_ = operationRepo.FailOperation(ctx, operation)
		}
	}()
	if rec.GeneratedOrderCode != nil {
		return nil, errorsuc.NewConflictError("a recorrência já possui pedido gerado")
	}
	params := defaultParameters(rec.EnterpriseCode, actor)
	if p, err := uc.Repo.GetParameters(ctx, rec.EnterpriseCode); err == nil && p != nil {
		params = p
	}
	lines, err := buildOrderLines(rec, params)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, errorsuc.NewValidationError("nenhuma parcela de cobrança recorrente foi gerada")
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
		// SalesPlanCode é uma regra/carteira comercial da recorrência e não o
		// production_plans.code usado pelo campo PlanCode do pedido.
		SalesDivisionCode: dto.SalesDivisionCode, CommissionPct: primary.CommissionPercent,
		PriceTableCode: dto.PriceTableCode, CurrencyCode: "BRL", PaymentTermCode: dto.PaymentTermCode,
		SaleDate: stringPtr(rec.SaleDate.Format("2006-01-02")), Notes: &notes,
	})
	if err != nil {
		return nil, err
	}
	for _, line := range lines {
		_, err := uc.SalesOrderItems.Execute(ctx, request.CreateSalesOrderItemDTO{
			SalesOrderCode: order.Code, Sequence: line.sequence, ItemCode: request.TextCode(fmt.Sprintf("%d", rec.ItemCode)), Mask: stringValue(rec.ItemMask),
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
	if err = operationRepo.CompleteOperation(ctx, operation, order.Code); err != nil {
		return nil, err
	}
	completed = true
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
		return nil, errorsuc.NewValidationError("a próxima data de reajuste é obrigatória para gerar pedido de recorrência indeterminada")
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
		return nil, errorsuc.NewValidationError("a recorrência determinada exige quantidade de parcelas e valor de pagamento")
	}
	payments := *rec.PaymentsQuantity - rec.GraceMonths
	if payments <= 0 {
		return nil, errorsuc.NewValidationError("a quantidade de parcelas deve ser maior que os meses de carência")
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
