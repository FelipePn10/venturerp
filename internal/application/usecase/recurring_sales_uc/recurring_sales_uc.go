package recurring_sales_uc

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/application/usecase/representativevalidation"
	"github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/entity"
	rsrepo "github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/repository"
	representativerepo "github.com/FelipePn10/panossoerp/internal/domain/representative/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"github.com/google/uuid"
)

type UseCase struct {
	Repo            rsrepo.Repository
	Auth            ports.AuthService
	SalesOrders     SalesOrderCreator
	SalesOrderItems SalesOrderItemCreator
	Representatives representativerepo.RepresentativeRepository
}

func (uc *UseCase) ensureAllowed(ctx context.Context) error {
	if uc.Auth == nil || !uc.Auth.CanCreateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return nil
}

func (uc *UseCase) identity(ctx context.Context) (int64, uuid.UUID, error) {
	enterpriseCode, err := uc.Auth.EnterpriseCode(ctx)
	if err != nil || enterpriseCode <= 0 {
		return 0, uuid.Nil, errorsuc.ErrUnauthorized
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil || actor == uuid.Nil {
		return 0, uuid.Nil, errorsuc.ErrUnauthorized
	}
	return enterpriseCode, actor, nil
}

func (uc *UseCase) UpsertParameters(ctx context.Context, dto request.UpsertRecurringSalesParametersDTO) (*response.RecurringSalesParametersResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	enterpriseCode, actor, err := uc.identity(ctx)
	if err != nil {
		return nil, err
	}
	if err := representativevalidation.Validate(ctx, uc.Representatives, dto.GenericRepresentativeCode); err != nil {
		return nil, err
	}
	if dto.CurrentMonthBillingLimitDay == 0 {
		dto.CurrentMonthBillingLimitDay = 10
	}
	if dto.IndefiniteDeliveryDay == 0 {
		dto.IndefiniteDeliveryDay = 10
	}
	if dto.FixedTermDeliveryDay == 0 {
		dto.FixedTermDeliveryDay = 10
	}
	if !validDay(dto.CurrentMonthBillingLimitDay) || !validDay(dto.IndefiniteDeliveryDay) || !validDay(dto.FixedTermDeliveryDay) {
		return nil, errorsuc.NewValidationError("os dias de faturamento e entrega devem estar entre 1 e 31")
	}
	created, err := uc.Repo.UpsertParameters(ctx, &entity.Parameters{
		EnterpriseCode: enterpriseCode, CurrentMonthBillingLimitDay: dto.CurrentMonthBillingLimitDay,
		GroupOrderItemTotal: dto.GroupOrderItemTotal, IndefiniteDeliveryDay: dto.IndefiniteDeliveryDay,
		FixedTermDeliveryDay: dto.FixedTermDeliveryDay, ConsiderDiscountsAdditions: dto.ConsiderDiscountsAdditions,
		GenericRepresentativeCode: dto.GenericRepresentativeCode, GenericSalesPlanCode: dto.GenericSalesPlanCode,
		UpdatedBy: actor,
	})
	if err != nil {
		return nil, err
	}
	return toParametersResponse(created), nil
}

func (uc *UseCase) GetParameters(ctx context.Context, enterpriseCode int64) (*response.RecurringSalesParametersResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	enterpriseCode, _, err := uc.identity(ctx)
	if err != nil {
		return nil, err
	}
	row, err := uc.Repo.GetParameters(ctx, enterpriseCode)
	if err != nil {
		return nil, err
	}
	return toParametersResponse(row), nil
}

func (uc *UseCase) CreateAdjustmentDate(ctx context.Context, dto request.CreateRecurringSalesAdjustmentDateDTO) (*response.RecurringSalesAdjustmentDateResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	enterpriseCode, actor, err := uc.identity(ctx)
	if err != nil {
		return nil, err
	}
	date, ok := datetime.ParseDate(dto.AdjustmentDate)
	if dto.CustomerCode == 0 || !ok {
		return nil, errorsuc.NewValidationError("cliente e data de reajuste válida são obrigatórios")
	}
	created, err := uc.Repo.CreateAdjustmentDate(ctx, &entity.AdjustmentDate{
		EnterpriseCode: enterpriseCode, CustomerCode: dto.CustomerCode, EstablishmentCode: dto.EstablishmentCode,
		AdjustmentDate: date, Notes: dto.Notes, CreatedBy: actor,
	})
	if err != nil {
		return nil, err
	}
	return toAdjustmentDateResponse(created), nil
}

func (uc *UseCase) ListAdjustmentDates(ctx context.Context, filter rsrepo.Filter) ([]*response.RecurringSalesAdjustmentDateResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	enterpriseCode, _, err := uc.identity(ctx)
	if err != nil {
		return nil, err
	}
	filter.EnterpriseCode = &enterpriseCode
	rows, err := uc.Repo.ListAdjustmentDates(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]*response.RecurringSalesAdjustmentDateResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAdjustmentDateResponse(row))
	}
	return out, nil
}

func (uc *UseCase) Create(ctx context.Context, dto request.CreateRecurringSaleDTO) (*response.RecurringSaleResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	enterpriseCode, actor, err := uc.identity(ctx)
	if err != nil {
		return nil, err
	}
	dto.EnterpriseCode, dto.CreatedBy = enterpriseCode, actor
	rec, err := uc.dtoToRecurringSale(dto)
	if err != nil {
		return nil, err
	}
	if err := validateRepresentatives(dto.Representatives); err != nil {
		return nil, err
	}
	for _, representative := range dto.Representatives {
		if err := representativevalidation.ValidateRequired(ctx, uc.Representatives, representative.RepresentativeCode); err != nil {
			return nil, err
		}
	}
	rec.Representatives = make([]*entity.Representative, 0, len(dto.Representatives))
	for _, repDTO := range dto.Representatives {
		rec.Representatives = append(rec.Representatives, &entity.Representative{RepresentativeCode: repDTO.RepresentativeCode, IsPrimary: repDTO.IsPrimary, CommissionPercent: repDTO.CommissionPercent, CommissionBase: normalizeCommissionBase(repDTO.CommissionBase), IsLifetime: repDTO.IsLifetime, CommissionInstallments: repDTO.CommissionInstallments})
	}
	created, err := uc.Repo.CreateWithRepresentatives(ctx, rec)
	if err != nil {
		return nil, err
	}
	return toRecurringSaleResponse(created), nil
}

func (uc *UseCase) Update(ctx context.Context, code int64, dto request.UpdateRecurringSaleDTO) (*response.RecurringSaleResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	current, err := uc.Repo.Get(ctx, code)
	if err != nil {
		return nil, err
	}
	if dto.SaleDate != "" {
		if current.SaleDate, err = requiredDate(dto.SaleDate, "sale_date"); err != nil {
			return nil, err
		}
	}
	current.SalesPlanCode, current.MonthsQuantity, current.PaymentsQuantity = dto.SalesPlanCode, dto.MonthsQuantity, dto.PaymentsQuantity
	current.NextAdjustmentDate = datetime.ParseDatePtr(&dto.NextAdjustmentDate)
	current.GraceMonths, current.PaymentValue, current.Reason = dto.GraceMonths, dto.PaymentValue, dto.Reason
	if dto.Quantity != 0 {
		current.Quantity = dto.Quantity
	}
	if dto.UnitValue != 0 {
		current.UnitValue = dto.UnitValue
	}
	if dto.IsActive != nil {
		current.IsActive = *dto.IsActive
	}
	if err := validateRecurringSale(current); err != nil {
		return nil, err
	}
	updated, err := uc.Repo.Update(ctx, current)
	if err != nil {
		return nil, err
	}
	return toRecurringSaleResponse(updated), nil
}

func (uc *UseCase) Get(ctx context.Context, code int64) (*response.RecurringSaleResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	row, err := uc.Repo.Get(ctx, code)
	if err != nil {
		return nil, err
	}
	return toRecurringSaleResponse(row), nil
}

func (uc *UseCase) List(ctx context.Context, filter rsrepo.Filter) ([]*response.RecurringSaleResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	enterpriseCode, _, err := uc.identity(ctx)
	if err != nil {
		return nil, err
	}
	filter.EnterpriseCode = &enterpriseCode
	rows, err := uc.Repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]*response.RecurringSaleResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toRecurringSaleResponse(row))
	}
	return out, nil
}

func (uc *UseCase) AddRepresentative(ctx context.Context, dto request.CreateRecurringSaleRepresentativeDTO) (*response.RecurringSaleRepresentativeResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	if dto.RecurringSaleCode == 0 || dto.RepresentativeCode == 0 {
		return nil, errorsuc.NewValidationError("recorrência e representante são obrigatórios")
	}
	if err := representativevalidation.ValidateRequired(ctx, uc.Representatives, dto.RepresentativeCode); err != nil {
		return nil, err
	}
	base := normalizeCommissionBase(dto.CommissionBase)
	if !dto.IsLifetime && (dto.CommissionInstallments == nil || *dto.CommissionInstallments <= 0) {
		return nil, errorsuc.NewValidationError("a quantidade de parcelas da comissão é obrigatória quando ela não é vitalícia")
	}
	if dto.CommissionPercent < 0 {
		return nil, errorsuc.NewValidationError("o percentual de comissão não pode ser negativo")
	}
	created, err := uc.Repo.AddRepresentative(ctx, &entity.Representative{
		RecurringSaleCode: dto.RecurringSaleCode, RepresentativeCode: dto.RepresentativeCode, IsPrimary: dto.IsPrimary,
		CommissionPercent: dto.CommissionPercent, CommissionBase: base, IsLifetime: dto.IsLifetime,
		CommissionInstallments: dto.CommissionInstallments,
	})
	if err != nil {
		return nil, err
	}
	return &response.RecurringSaleRepresentativeResponse{
		Code: created.Code, RepresentativeCode: created.RepresentativeCode, IsPrimary: created.IsPrimary,
		CommissionPercent: created.CommissionPercent, CommissionBase: string(created.CommissionBase),
		IsLifetime: created.IsLifetime, CommissionInstallments: created.CommissionInstallments,
	}, nil
}

func (uc *UseCase) MarkOrderGenerated(ctx context.Context, code int64, orderCode int64) (*response.RecurringSaleResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	if orderCode == 0 {
		return nil, errorsuc.NewValidationError("o código do pedido é obrigatório")
	}
	row, err := uc.Repo.MarkOrderGenerated(ctx, code, orderCode)
	if err != nil {
		return nil, err
	}
	return toRecurringSaleResponse(row), nil
}

func (uc *UseCase) ClearGeneratedOrder(ctx context.Context, code int64) (*response.RecurringSaleResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	row, err := uc.Repo.ClearGeneratedOrder(ctx, code)
	if err != nil {
		return nil, err
	}
	return toRecurringSaleResponse(row), nil
}

func (uc *UseCase) Cancel(ctx context.Context, code int64, dto request.CancelRecurringSaleDTO) (*response.RecurringSaleResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	_, actor, err := uc.identity(ctx)
	if err != nil {
		return nil, err
	}
	if dto.Reason == nil || strings.TrimSpace(*dto.Reason) == "" {
		return nil, errorsuc.NewValidationError("o motivo do cancelamento é obrigatório")
	}
	effectiveDate, ok := datetime.ParseDate(dto.EffectiveDate)
	if !ok {
		return nil, errorsuc.NewValidationError("effective_date deve ser uma data ISO válida no formato AAAA-MM-DD")
	}
	policy := strings.ToUpper(strings.TrimSpace(dto.FutureOrdersPolicy))
	if policy != "MANTER" && policy != "CANCELAR_NAO_FATURADOS" && policy != "NAO_GERAR_NOVOS" {
		return nil, errorsuc.NewValidationError("future_orders_policy deve ser MANTER, CANCELAR_NAO_FATURADOS ou NAO_GERAR_NOVOS")
	}
	if atomicRepo, supported := uc.Repo.(rsrepo.AtomicCanceller); supported {
		row, cancelErr := atomicRepo.CancelAtomic(ctx, rsrepo.CancellationCommand{
			Code: code, EffectiveDate: effectiveDate, FutureOrdersPolicy: policy,
			Reason: strings.TrimSpace(*dto.Reason), ActorID: actor, CorrelationID: strings.TrimSpace(dto.CorrelationID),
		})
		if cancelErr != nil {
			return nil, cancelErr
		}
		return toRecurringSaleResponse(row), nil
	}
	current, err := uc.Repo.Get(ctx, code)
	if err != nil {
		return nil, err
	}
	if current.GeneratedOrderCode == nil {
		return nil, errorsuc.NewValidationError("somente recorrências com pedido gerado podem ser canceladas")
	}
	cancel := *current
	cancel.Code = 0
	cancel.MovementType = entity.MovementCancellation
	cancel.SourceRecurringSaleCode = &current.Code
	cancel.GeneratedOrderCode = nil
	cancel.GeneratedOrderAt = nil
	cancel.Reason = dto.Reason
	cancel.CreatedBy = actor
	created, err := uc.Repo.Create(ctx, &cancel)
	if err != nil {
		return nil, err
	}
	if _, err := uc.Repo.Deactivate(ctx, code, dto.Reason); err != nil {
		return nil, err
	}
	return toRecurringSaleResponse(created), nil
}

func (uc *UseCase) CalculateAdjustment(ctx context.Context, dto request.CalculateRecurringSalesAdjustmentDTO) (*response.RecurringSalesAdjustmentImpactResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	enterpriseCode, actor, err := uc.identity(ctx)
	if err != nil {
		return nil, err
	}
	dto.EnterpriseCode, dto.CreatedBy = &enterpriseCode, actor
	adjustDate, err := requiredDate(dto.AdjustmentDate, "adjustment_date")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(dto.Reason) == "" {
		return nil, errorsuc.NewValidationError("o motivo é obrigatório")
	}
	var operationRepo rsrepo.OperationRepository
	if dto.Confirm {
		dto.IdempotencyKey = strings.TrimSpace(dto.IdempotencyKey)
		if dto.IdempotencyKey == "" {
			dto.IdempotencyKey = "legacy:recurring:adjustment:" + adjustDate.Format("2006-01")
		}
		var ok bool
		operationRepo, ok = uc.Repo.(rsrepo.OperationRepository)
		if !ok {
			return nil, errorsuc.NewValidationError("o controle idempotente do reajuste não está configurado")
		}
	}
	mtSale, mtUpgrade, mtDowngrade := entity.MovementSale, entity.MovementUpgrade, entity.MovementDowngrade
	filter := rsrepo.Filter{EnterpriseCode: dto.EnterpriseCode, CustomerCode: dto.CustomerCode, EstablishmentCode: dto.EstablishmentCode, ItemCode: dto.ItemCode, OnlyActive: true}
	rows, err := uc.Repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	impacts := make([]response.RecurringSaleResponse, 0)
	lineImpacts := make([]response.RecurringSalesAdjustmentLineImpactResponse, 0)
	total := 0.0
	groups := map[string][]*entity.RecurringSale{}
	for _, row := range rows {
		if row.TermType != entity.TermIndefinite || row.GeneratedOrderCode == nil || row.SaleDate.Month() == adjustDate.Month() && row.SaleDate.Year() == adjustDate.Year() {
			continue
		}
		if row.NextAdjustmentDate == nil || !sameDate(*row.NextAdjustmentDate, adjustDate) {
			continue
		}
		if row.MovementType != mtSale && row.MovementType != mtUpgrade && row.MovementType != mtDowngrade {
			continue
		}
		key := adjustmentGroupKey(row)
		groups[key] = append(groups[key], row)
	}
	for _, groupRows := range groups {
		adjustment := buildAdjustment(groupRows, adjustDate, dto.AdjustmentPercent, dto.Reason, dto.CreatedBy)
		sourceCodes := make([]int64, 0, len(groupRows))
		previousTotal := 0.0
		for _, source := range groupRows {
			sourceCodes = append(sourceCodes, source.Code)
			previousTotal += monthlyValue(source)
		}
		previousUnit := 0.0
		if adjustment.Quantity > 0 {
			previousUnit = previousTotal / adjustment.Quantity
		}
		index := strings.TrimSpace(dto.AdjustmentIndex)
		if index == "" && groupRows[0].AdjustmentIndex != nil {
			index = strings.TrimSpace(*groupRows[0].AdjustmentIndex)
		}
		lineImpacts = append(lineImpacts, response.RecurringSalesAdjustmentLineImpactResponse{
			SourceCodes: sourceCodes, ItemCode: adjustment.ItemCode, ItemMask: adjustment.ItemMask,
			PreviousUnitValue: round4(previousUnit), NewUnitValue: adjustment.UnitValue,
			Quantity: adjustment.Quantity, PreviousTotal: round2(previousTotal), NewTotal: monthlyValue(adjustment),
			AdjustmentPercent: dto.AdjustmentPercent, AdjustmentIndex: index,
			LegalBasis: strings.TrimSpace(dto.LegalBasis), EffectiveDate: adjustDate, Reason: strings.TrimSpace(dto.Reason),
		})
		total += monthlyValue(adjustment)
		if dto.Confirm {
			payload, _ := json.Marshal(dto)
			digest := sha256.Sum256(payload)
			operations := make([]*entity.Operation, 0, len(groupRows))
			var replayCode *int64
			for _, source := range groupRows {
				operation := &entity.Operation{RecurringSaleCode: source.Code, OperationType: "REAJUSTE", Competence: adjustDate.Format("2006-01"),
					IdempotencyKey: fmt.Sprintf("%s:%d", dto.IdempotencyKey, source.Code), RequestHash: fmt.Sprintf("%x", digest), ActorID: actor}
				reserved, replay, reserveErr := operationRepo.ReserveOperation(ctx, operation)
				if errors.Is(reserveErr, rsrepo.ErrOperationConflict) {
					return nil, errorsuc.NewConflictError("já existe reajuste para esta competência ou a chave de idempotência foi reutilizada com outro conteúdo")
				}
				if errors.Is(reserveErr, rsrepo.ErrOperationInProgress) {
					return nil, errorsuc.NewConflictError("o reajuste desta competência já está em processamento")
				}
				if reserveErr != nil {
					return nil, reserveErr
				}
				operations = append(operations, reserved)
				if replay && reserved.ResultCode != nil {
					replayCode = reserved.ResultCode
				}
			}
			if replayCode != nil {
				persisted, getErr := uc.Repo.Get(ctx, *replayCode)
				if getErr != nil {
					return nil, getErr
				}
				impacts = append(impacts, *toRecurringSaleResponse(persisted))
				continue
			}
			created, err := uc.Repo.Create(ctx, adjustment)
			if err != nil {
				for _, operation := range operations {
					_ = operationRepo.FailOperation(ctx, operation)
				}
				return nil, err
			}
			for _, rep := range adjustment.Representatives {
				rep.RecurringSaleCode = created.Code
				if _, err := uc.Repo.AddRepresentative(ctx, rep); err != nil {
					return nil, err
				}
			}
			for _, source := range groupRows {
				if err := uc.Repo.CreateAdjustmentLink(ctx, created.Code, source.Code); err != nil {
					return nil, err
				}
			}
			for _, operation := range operations {
				if err := operationRepo.CompleteOperation(ctx, operation, created.Code); err != nil {
					return nil, err
				}
			}
			adjustment = created
		}
		impacts = append(impacts, *toRecurringSaleResponse(adjustment))
	}
	return &response.RecurringSalesAdjustmentImpactResponse{Rows: impacts, Impacts: lineImpacts, TotalRows: len(impacts), TotalValue: round2(total), Confirmed: dto.Confirm}, nil
}

func (uc *UseCase) RecalculateAdjustment(ctx context.Context, code int64, dto request.RecalculateRecurringSalesAdjustmentDTO) (*response.RecurringSaleResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	current, err := uc.Repo.Get(ctx, code)
	if err != nil {
		return nil, err
	}
	if current.MovementType != entity.MovementAdjustment {
		return nil, errorsuc.NewValidationError("somente movimentos de reajuste podem ser recalculados")
	}
	base := current.UnitValue
	if current.AdjustmentPercent != nil {
		base = current.UnitValue / (1 + (*current.AdjustmentPercent / 100))
	}
	current.UnitValue = round4(base * (1 + dto.AdjustmentPercent/100))
	current.AdjustmentPercent = &dto.AdjustmentPercent
	current.Reason = &dto.Reason
	updated, err := uc.Repo.Update(ctx, current)
	if err != nil {
		return nil, err
	}
	return toRecurringSaleResponse(updated), nil
}

func (uc *UseCase) RevenueProjection(ctx context.Context, filter rsrepo.ProjectionFilter) ([]entity.ProjectionRow, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	enterpriseCode, _, err := uc.identity(ctx)
	if err != nil {
		return nil, err
	}
	filter.EnterpriseCode = &enterpriseCode
	if filter.From.IsZero() || filter.To.IsZero() || filter.To.Before(filter.From) {
		return nil, errorsuc.NewValidationError("informe um período inicial e final válido")
	}
	rows, err := uc.Repo.List(ctx, rsrepo.Filter{EnterpriseCode: filter.EnterpriseCode, CustomerCode: filter.CustomerCode, ItemCode: filter.ItemCode, OnlyActive: true})
	if err != nil {
		return nil, err
	}
	return projectRevenue(rows, filter), nil
}

func (uc *UseCase) CommissionProjection(ctx context.Context, filter rsrepo.ProjectionFilter) ([]entity.CommissionProjectionRow, error) {
	revenue, err := uc.RevenueProjection(ctx, filter)
	if err != nil {
		return nil, err
	}
	rows, err := uc.Repo.List(ctx, rsrepo.Filter{EnterpriseCode: filter.EnterpriseCode, CustomerCode: filter.CustomerCode, ItemCode: filter.ItemCode, RepresentativeCode: filter.RepresentativeCode, OnlyActive: true})
	if err != nil {
		return nil, err
	}
	byCode := map[int64]*entity.RecurringSale{}
	for _, row := range rows {
		byCode[row.Code] = row
	}
	out := make([]entity.CommissionProjectionRow, 0)
	for _, rev := range revenue {
		rec := byCode[rev.RecurringSaleCode]
		if rec == nil {
			continue
		}
		for _, rep := range rec.Representatives {
			if filter.RepresentativeCode != nil && rep.RepresentativeCode != *filter.RepresentativeCode {
				continue
			}
			baseValue := rev.ProjectedValue
			if rep.CommissionBase == entity.CommissionBaseOriginal {
				baseValue = monthlyValue(rec)
			}
			out = append(out, entity.CommissionProjectionRow{
				ProjectionRow: rev, RepresentativeCode: rep.RepresentativeCode,
				CommissionPercent: rep.CommissionPercent, CommissionValue: round2(baseValue * rep.CommissionPercent / 100),
			})
		}
	}
	return out, nil
}

func (uc *UseCase) dtoToRecurringSale(dto request.CreateRecurringSaleDTO) (*entity.RecurringSale, error) {
	saleDate, err := requiredDate(dto.SaleDate, "sale_date")
	if err != nil {
		return nil, err
	}
	movement, err := normalizeMovement(dto.MovementType)
	if err != nil {
		return nil, err
	}
	if movement != entity.MovementSale && movement != entity.MovementUpgrade {
		return nil, errorsuc.NewValidationError("somente VENDA e UPGRADE podem ser criados diretamente")
	}
	term := normalizeTerm(dto.TermType)
	rec := &entity.RecurringSale{
		EnterpriseCode: dto.EnterpriseCode, CustomerCode: dto.CustomerCode, EstablishmentCode: dto.EstablishmentCode,
		ItemCode: dto.ItemCode, ItemMask: dto.ItemMask, SalesPlanCode: dto.SalesPlanCode, MovementType: movement,
		TermType: term, SaleDate: saleDate, NextAdjustmentDate: datetime.ParseDatePtr(&dto.NextAdjustmentDate),
		MonthsQuantity: dto.MonthsQuantity, PaymentsQuantity: dto.PaymentsQuantity, GraceMonths: dto.GraceMonths,
		PaymentValue: dto.PaymentValue, Quantity: dto.Quantity, UnitValue: dto.UnitValue, Reason: dto.Reason,
		IsActive: true, LifecycleStatus: entity.LifecycleActive, CreatedBy: dto.CreatedBy,
		EffectiveFrom: datetime.ParseDatePtr(&dto.EffectiveFrom), EffectiveUntil: datetime.ParseDatePtr(&dto.EffectiveUntil),
		Frequency: strings.ToUpper(strings.TrimSpace(dto.Frequency)), PriceTableCode: dto.PriceTableCode,
		CurrencyCode: strings.ToUpper(strings.TrimSpace(dto.CurrencyCode)), AdjustmentIndex: dto.AdjustmentIndex,
		AdjustmentPeriodMonths: dto.AdjustmentPeriodMonths, AdjustmentFloorPct: dto.AdjustmentFloorPct,
		AdjustmentCapPct: dto.AdjustmentCapPct, BillingPolicy: dto.BillingPolicy, DeliveryPolicy: dto.DeliveryPolicy,
		TaxPolicy: dto.TaxPolicy, CostCenterCode: dto.CostCenterCode, RenewalPolicy: strings.ToUpper(strings.TrimSpace(dto.RenewalPolicy)),
	}
	if rec.EffectiveFrom == nil {
		rec.EffectiveFrom = &rec.SaleDate
	}
	if rec.Frequency == "" {
		rec.Frequency = "MENSAL"
	}
	if rec.CurrencyCode == "" {
		rec.CurrencyCode = "BRL"
	}
	if rec.RenewalPolicy == "" {
		rec.RenewalPolicy = "AUTOMATICA"
	}
	if len(rec.BillingPolicy) == 0 {
		rec.BillingPolicy = json.RawMessage(`{}`)
	}
	if len(rec.DeliveryPolicy) == 0 {
		rec.DeliveryPolicy = json.RawMessage(`{}`)
	}
	if len(rec.TaxPolicy) == 0 {
		rec.TaxPolicy = json.RawMessage(`{}`)
	}
	if rec.Quantity == 0 {
		rec.Quantity = 1
	}
	return rec, validateRecurringSale(rec)
}

func validateRecurringSale(v *entity.RecurringSale) error {
	if v.EnterpriseCode == 0 || v.CustomerCode == 0 || v.ItemCode == 0 {
		return errorsuc.NewValidationError("empresa, cliente e item são obrigatórios")
	}
	if v.Quantity <= 0 || v.UnitValue < 0 || v.GraceMonths < 0 {
		return errorsuc.NewValidationError("a quantidade deve ser positiva e os valores não podem ser negativos")
	}
	if v.EffectiveUntil != nil && v.EffectiveFrom != nil && v.EffectiveUntil.Before(*v.EffectiveFrom) {
		return errorsuc.NewValidationError("effective_until não pode ser anterior a effective_from")
	}
	validFrequency := map[string]bool{"SEMANAL": true, "QUINZENAL": true, "MENSAL": true, "BIMESTRAL": true, "TRIMESTRAL": true, "SEMESTRAL": true, "ANUAL": true}
	if !validFrequency[v.Frequency] {
		return errorsuc.NewValidationError("frequência inválida")
	}
	if v.AdjustmentPeriodMonths != nil && *v.AdjustmentPeriodMonths <= 0 {
		return errorsuc.NewValidationError("adjustment_period_months deve ser positivo")
	}
	if v.AdjustmentFloorPct != nil && v.AdjustmentCapPct != nil && *v.AdjustmentFloorPct > *v.AdjustmentCapPct {
		return errorsuc.NewValidationError("o piso do reajuste não pode superar o teto")
	}
	if v.TermType == entity.TermIndefinite && v.NextAdjustmentDate == nil {
		return errorsuc.NewValidationError("a próxima data de reajuste é obrigatória para vigência indeterminada")
	}
	if v.TermType == entity.TermFixed {
		if v.MonthsQuantity == nil || *v.MonthsQuantity <= 0 || v.PaymentsQuantity == nil || *v.PaymentsQuantity <= 0 || v.PaymentValue == nil {
			return errorsuc.NewValidationError("meses, parcelas e valor da parcela são obrigatórios para vigência determinada")
		}
		if *v.PaymentsQuantity <= v.GraceMonths {
			return errorsuc.NewValidationError("a quantidade de parcelas deve ser maior que a carência")
		}
	}
	return nil
}

func validateRepresentatives(rows []request.CreateRecurringSaleRepresentativeDTO) error {
	if len(rows) == 0 {
		return errorsuc.NewValidationError("informe pelo menos um representante")
	}
	primary := 0
	for _, row := range rows {
		if row.IsPrimary {
			primary++
		}
	}
	if primary != 1 {
		return errorsuc.NewValidationError("informe exatamente um representante principal")
	}
	return nil
}

func normalizeMovement(raw string) (entity.MovementType, error) {
	v := entity.MovementType(strings.ToUpper(strings.TrimSpace(raw)))
	if v == "" {
		return entity.MovementSale, nil
	}
	switch v {
	case entity.MovementSale, entity.MovementUpgrade, entity.MovementDowngrade, entity.MovementAdjustment, entity.MovementRecalculation, entity.MovementCancellation:
		return v, nil
	default:
		return "", errorsuc.NewValidationError("tipo de movimento inválido")
	}
}

func normalizeTerm(raw string) entity.TermType {
	if strings.ToUpper(strings.TrimSpace(raw)) == string(entity.TermFixed) {
		return entity.TermFixed
	}
	return entity.TermIndefinite
}

func normalizeCommissionBase(raw string) entity.CommissionBase {
	if strings.ToUpper(strings.TrimSpace(raw)) == string(entity.CommissionBaseOriginal) {
		return entity.CommissionBaseOriginal
	}
	return entity.CommissionBaseAdjusted
}

func requiredDate(raw, field string) (time.Time, error) {
	if t, ok := datetime.ParseDate(raw); ok {
		return t, nil
	}
	return time.Time{}, errorsuc.NewValidationError("o campo " + field + " é obrigatório e deve conter uma data válida")
}

func validDay(v int) bool { return v >= 1 && v <= 31 }

func monthlyValue(v *entity.RecurringSale) float64 {
	if v.TermType == entity.TermFixed && v.PaymentValue != nil {
		return round2(*v.PaymentValue)
	}
	return round2(v.Quantity * v.UnitValue)
}

func buildAdjustment(rows []*entity.RecurringSale, date time.Time, pct float64, reason string, createdBy uuid.UUID) *entity.RecurringSale {
	first := rows[0]
	totalQty, totalValue := 0.0, 0.0
	for _, row := range rows {
		totalQty += row.Quantity
		totalValue += monthlyValue(row)
	}
	avg := 0.0
	if totalQty > 0 {
		avg = totalValue / totalQty
	}
	adjusted := round4(avg * (1 + pct/100))
	createdUUID := createdBy
	if createdUUID == uuid.Nil {
		createdUUID = first.CreatedBy
	}
	return &entity.RecurringSale{
		EnterpriseCode: first.EnterpriseCode, CustomerCode: first.CustomerCode, EstablishmentCode: first.EstablishmentCode,
		ItemCode: first.ItemCode, ItemMask: first.ItemMask, SalesPlanCode: first.SalesPlanCode, MovementType: entity.MovementAdjustment,
		TermType: entity.TermIndefinite, SaleDate: date, NextAdjustmentDate: timePtr(date.AddDate(1, 0, 0)),
		Quantity: totalQty, UnitValue: adjusted, Reason: &reason, AdjustmentPercent: &pct, IsActive: true, CreatedBy: createdUUID,
		Representatives: cloneRepresentatives(first.Representatives),
	}
}

func cloneRepresentatives(rows []*entity.Representative) []*entity.Representative {
	out := make([]*entity.Representative, 0, len(rows))
	for _, row := range rows {
		cp := *row
		cp.Code = 0
		cp.RecurringSaleCode = 0
		out = append(out, &cp)
	}
	return out
}

func adjustmentGroupKey(v *entity.RecurringSale) string {
	mask := ""
	if v.ItemMask != nil {
		mask = *v.ItemMask
	}
	est := int64(0)
	if v.EstablishmentCode != nil {
		est = *v.EstablishmentCode
	}
	plan := int64(0)
	if v.SalesPlanCode != nil {
		plan = *v.SalesPlanCode
	}
	return strings.Join([]string{
		intString(v.EnterpriseCode), intString(v.CustomerCode), intString(est), intString(v.ItemCode), mask, intString(plan),
	}, "|")
}

func projectRevenue(rows []*entity.RecurringSale, filter rsrepo.ProjectionFilter) []entity.ProjectionRow {
	out := make([]entity.ProjectionRow, 0)
	from := monthStart(filter.From)
	to := monthStart(filter.To)
	for _, row := range rows {
		if row.GeneratedOrderCode == nil || !row.IsActive {
			continue
		}
		for m := from; !m.After(to); m = m.AddDate(0, 1, 0) {
			if !recursInMonth(row, m) {
				continue
			}
			value := monthlyValue(row)
			applied := false
			if row.NextAdjustmentDate != nil && !monthStart(*row.NextAdjustmentDate).After(m) && filter.AdjustmentPercent != 0 {
				value = round2(value * (1 + filter.AdjustmentPercent/100))
				applied = true
			}
			out = append(out, entity.ProjectionRow{
				Month: m, EnterpriseCode: row.EnterpriseCode, CustomerCode: row.CustomerCode, EstablishmentCode: row.EstablishmentCode,
				ItemCode: row.ItemCode, ItemMask: row.ItemMask, RecurringSaleCode: row.Code, Quantity: row.Quantity,
				UnitValue: row.UnitValue, ProjectedValue: value, AppliedAdjustment: applied,
			})
		}
	}
	return out
}

func recursInMonth(v *entity.RecurringSale, month time.Time) bool {
	start := monthStart(v.SaleDate)
	if month.Before(start) {
		return false
	}
	if v.TermType != entity.TermFixed || v.MonthsQuantity == nil {
		return true
	}
	end := start.AddDate(0, *v.MonthsQuantity, 0)
	return month.Before(end)
}

func monthStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}
func sameDate(a, b time.Time) bool   { return a.Year() == b.Year() && a.YearDay() == b.YearDay() }
func timePtr(t time.Time) *time.Time { return &t }
func round2(v float64) float64       { return math.Round(v*100) / 100 }
func round4(v float64) float64       { return math.Round(v*10000) / 10000 }

func intString(v int64) string {
	if v == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	return string(b[i:])
}
