package recurring_sales_uc

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/entity"
	rsrepo "github.com/FelipePn10/panossoerp/internal/domain/recurring_sales/repository"
	"github.com/FelipePn10/panossoerp/internal/pkg/datetime"
	"github.com/google/uuid"
)

type UseCase struct {
	Repo            rsrepo.Repository
	Auth            ports.AuthService
	SalesOrders     SalesOrderCreator
	SalesOrderItems SalesOrderItemCreator
}

func (uc *UseCase) ensureAllowed(ctx context.Context) error {
	if uc.Auth == nil || !uc.Auth.CanCreateSalesOrder(ctx) {
		return errorsuc.ErrUnauthorized
	}
	return nil
}

func (uc *UseCase) UpsertParameters(ctx context.Context, dto request.UpsertRecurringSalesParametersDTO) (*response.RecurringSalesParametersResponse, error) {
	if err := uc.ensureAllowed(ctx); err != nil {
		return nil, err
	}
	if dto.EnterpriseCode == 0 {
		return nil, errorsuc.NewValidationError("enterprise_code is required")
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
		return nil, errorsuc.NewValidationError("billing and delivery days must be between 1 and 31")
	}
	created, err := uc.Repo.UpsertParameters(ctx, &entity.Parameters{
		EnterpriseCode: dto.EnterpriseCode, CurrentMonthBillingLimitDay: dto.CurrentMonthBillingLimitDay,
		GroupOrderItemTotal: dto.GroupOrderItemTotal, IndefiniteDeliveryDay: dto.IndefiniteDeliveryDay,
		FixedTermDeliveryDay: dto.FixedTermDeliveryDay, ConsiderDiscountsAdditions: dto.ConsiderDiscountsAdditions,
		GenericRepresentativeCode: dto.GenericRepresentativeCode, GenericSalesPlanCode: dto.GenericSalesPlanCode,
		UpdatedBy: dto.UpdatedBy,
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
	date, ok := datetime.ParseDate(dto.AdjustmentDate)
	if dto.EnterpriseCode == 0 || dto.CustomerCode == 0 || !ok {
		return nil, errorsuc.NewValidationError("enterprise_code, customer_code and valid adjustment_date are required")
	}
	created, err := uc.Repo.CreateAdjustmentDate(ctx, &entity.AdjustmentDate{
		EnterpriseCode: dto.EnterpriseCode, CustomerCode: dto.CustomerCode, EstablishmentCode: dto.EstablishmentCode,
		AdjustmentDate: date, Notes: dto.Notes, CreatedBy: dto.CreatedBy,
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
	rec, err := uc.dtoToRecurringSale(dto)
	if err != nil {
		return nil, err
	}
	if err := validateRepresentatives(dto.Representatives); err != nil {
		return nil, err
	}
	created, err := uc.Repo.Create(ctx, rec)
	if err != nil {
		return nil, err
	}
	for _, repDTO := range dto.Representatives {
		repDTO.RecurringSaleCode = created.Code
		if _, err := uc.AddRepresentative(ctx, repDTO); err != nil {
			return nil, err
		}
	}
	return uc.Get(ctx, created.Code)
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
		return nil, errorsuc.NewValidationError("recurring_sale_code and representative_code are required")
	}
	base := normalizeCommissionBase(dto.CommissionBase)
	if !dto.IsLifetime && (dto.CommissionInstallments == nil || *dto.CommissionInstallments <= 0) {
		return nil, errorsuc.NewValidationError("commission_installments is required when commission is not lifetime")
	}
	if dto.CommissionPercent < 0 {
		return nil, errorsuc.NewValidationError("commission_percent cannot be negative")
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
		return nil, errorsuc.NewValidationError("order_code is required")
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
	current, err := uc.Repo.Get(ctx, code)
	if err != nil {
		return nil, err
	}
	if current.GeneratedOrderCode == nil {
		return nil, errorsuc.NewValidationError("only recurring sales with generated order can be cancelled")
	}
	cancel := *current
	cancel.Code = 0
	cancel.MovementType = entity.MovementCancellation
	cancel.SourceRecurringSaleCode = &current.Code
	cancel.GeneratedOrderCode = nil
	cancel.GeneratedOrderAt = nil
	cancel.Reason = dto.Reason
	cancel.CreatedBy = dto.CreatedBy
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
	adjustDate, err := requiredDate(dto.AdjustmentDate, "adjustment_date")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(dto.Reason) == "" {
		return nil, errorsuc.NewValidationError("reason is required")
	}
	mtSale, mtUpgrade, mtDowngrade := entity.MovementSale, entity.MovementUpgrade, entity.MovementDowngrade
	filter := rsrepo.Filter{EnterpriseCode: dto.EnterpriseCode, CustomerCode: dto.CustomerCode, EstablishmentCode: dto.EstablishmentCode, ItemCode: dto.ItemCode, OnlyActive: true}
	rows, err := uc.Repo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	impacts := make([]response.RecurringSaleResponse, 0)
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
		total += monthlyValue(adjustment)
		if dto.Confirm {
			created, err := uc.Repo.Create(ctx, adjustment)
			if err != nil {
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
			adjustment = created
		}
		impacts = append(impacts, *toRecurringSaleResponse(adjustment))
	}
	return &response.RecurringSalesAdjustmentImpactResponse{Rows: impacts, TotalRows: len(impacts), TotalValue: round2(total), Confirmed: dto.Confirm}, nil
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
		return nil, errorsuc.NewValidationError("only adjustment movements can be recalculated")
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
	if filter.From.IsZero() || filter.To.IsZero() || filter.To.Before(filter.From) {
		return nil, errorsuc.NewValidationError("valid from and to are required")
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
		return nil, errorsuc.NewValidationError("only SALE and UPGRADE can be created directly")
	}
	term := normalizeTerm(dto.TermType)
	rec := &entity.RecurringSale{
		EnterpriseCode: dto.EnterpriseCode, CustomerCode: dto.CustomerCode, EstablishmentCode: dto.EstablishmentCode,
		ItemCode: dto.ItemCode, ItemMask: dto.ItemMask, SalesPlanCode: dto.SalesPlanCode, MovementType: movement,
		TermType: term, SaleDate: saleDate, NextAdjustmentDate: datetime.ParseDatePtr(&dto.NextAdjustmentDate),
		MonthsQuantity: dto.MonthsQuantity, PaymentsQuantity: dto.PaymentsQuantity, GraceMonths: dto.GraceMonths,
		PaymentValue: dto.PaymentValue, Quantity: dto.Quantity, UnitValue: dto.UnitValue, Reason: dto.Reason,
		IsActive: true, CreatedBy: dto.CreatedBy,
	}
	if rec.Quantity == 0 {
		rec.Quantity = 1
	}
	return rec, validateRecurringSale(rec)
}

func validateRecurringSale(v *entity.RecurringSale) error {
	if v.EnterpriseCode == 0 || v.CustomerCode == 0 || v.ItemCode == 0 {
		return errorsuc.NewValidationError("enterprise_code, customer_code and item_code are required")
	}
	if v.Quantity <= 0 || v.UnitValue < 0 || v.GraceMonths < 0 {
		return errorsuc.NewValidationError("quantity must be positive and values cannot be negative")
	}
	if v.TermType == entity.TermIndefinite && v.NextAdjustmentDate == nil {
		return errorsuc.NewValidationError("next_adjustment_date is required for indefinite term")
	}
	if v.TermType == entity.TermFixed {
		if v.MonthsQuantity == nil || *v.MonthsQuantity <= 0 || v.PaymentsQuantity == nil || *v.PaymentsQuantity <= 0 || v.PaymentValue == nil {
			return errorsuc.NewValidationError("months_quantity, payments_quantity and payment_value are required for fixed term")
		}
		if *v.PaymentsQuantity <= v.GraceMonths {
			return errorsuc.NewValidationError("payments_quantity must be greater than grace_months")
		}
	}
	return nil
}

func validateRepresentatives(rows []request.CreateRecurringSaleRepresentativeDTO) error {
	if len(rows) == 0 {
		return errorsuc.NewValidationError("at least one representative is required")
	}
	primary := 0
	for _, row := range rows {
		if row.IsPrimary {
			primary++
		}
	}
	if primary != 1 {
		return errorsuc.NewValidationError("exactly one primary representative is required")
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
		return "", errorsuc.NewValidationError("invalid movement_type")
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
	return time.Time{}, errorsuc.NewValidationError(field + " is required or invalid")
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
