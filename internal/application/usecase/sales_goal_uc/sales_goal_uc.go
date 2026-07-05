package sales_goal_uc

import (
	"context"
	"strings"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_goal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_goal/repository"
)

type UseCase struct {
	Repo repository.Repository
	Auth ports.AuthService
}

func (uc *UseCase) CreatePeriod(ctx context.Context, dto request.CreateSalesGoalPeriodDTO) (*response.SalesGoalPeriodResponse, error) {
	if !uc.Auth.CanCreateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	p, err := periodFromDTO(dto)
	if err != nil {
		return nil, err
	}
	created, err := uc.Repo.CreatePeriod(ctx, p)
	if err != nil {
		return nil, err
	}
	return toPeriodResponse(created), nil
}

func (uc *UseCase) ListPeriods(ctx context.Context, filter repository.PeriodFilter) ([]response.SalesGoalPeriodResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	rows, err := uc.Repo.ListPeriods(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]response.SalesGoalPeriodResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, *toPeriodResponse(row))
	}
	return out, nil
}

func (uc *UseCase) CreateGoal(ctx context.Context, dto request.CreateSalesGoalDTO) (*response.SalesGoalResponse, error) {
	if !uc.Auth.CanCreateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	g, err := goalFromCreate(dto)
	if err != nil {
		return nil, err
	}
	created, err := uc.Repo.CreateGoal(ctx, g)
	if err != nil {
		return nil, err
	}
	return toGoalResponse(created), nil
}

func (uc *UseCase) UpdateGoal(ctx context.Context, dto request.UpdateSalesGoalDTO) (*response.SalesGoalResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	g, err := goalFromUpdate(dto)
	if err != nil {
		return nil, err
	}
	updated, err := uc.Repo.UpdateGoal(ctx, g)
	if err != nil {
		return nil, err
	}
	return toGoalResponse(updated), nil
}

func (uc *UseCase) GetGoal(ctx context.Context, code int64) (*response.SalesGoalResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	g, err := uc.Repo.GetGoal(ctx, code)
	if err != nil {
		return nil, err
	}
	return toGoalResponse(g), nil
}

func (uc *UseCase) ListGoals(ctx context.Context, filter repository.GoalFilter) ([]response.SalesGoalResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	filter.AnalysisBase = normalizeAnalysisBase(filter.AnalysisBase)
	rows, err := uc.Repo.ListGoals(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]response.SalesGoalResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, *toGoalResponse(row))
	}
	return out, nil
}

func (uc *UseCase) AddGoalItem(ctx context.Context, dto request.SalesGoalItemDTO) (*response.SalesGoalItemResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	item, err := goalItemFromDTO(dto)
	if err != nil {
		return nil, err
	}
	created, err := uc.Repo.AddGoalItem(ctx, item)
	if err != nil {
		return nil, err
	}
	return toGoalItemResponse(created), nil
}

func (uc *UseCase) UpsertGroupTarget(ctx context.Context, dto request.SalesGoalGroupTargetDTO) (*response.SalesGoalGroupTargetResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	target, err := groupTargetFromDTO(dto)
	if err != nil {
		return nil, err
	}
	row, err := uc.Repo.UpsertGroupTarget(ctx, target)
	if err != nil {
		return nil, err
	}
	return toGroupTargetResponse(row), nil
}

func (uc *UseCase) AddGroupCustomer(ctx context.Context, dto request.SalesGoalGroupCustomerDTO) (*response.SalesGoalGroupCustomerResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	customer, err := groupCustomerFromDTO(dto)
	if err != nil {
		return nil, err
	}
	row, err := uc.Repo.AddGroupCustomer(ctx, customer)
	if err != nil {
		return nil, err
	}
	return toGroupCustomerResponse(row), nil
}

func (uc *UseCase) UpsertBalance(ctx context.Context, dto request.SalesGoalBalanceDTO) (*response.SalesGoalBalanceResponse, error) {
	if !uc.Auth.CanUpdateSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	balance, err := balanceFromDTO(dto)
	if err != nil {
		return nil, err
	}
	row, err := uc.Repo.UpsertBalance(ctx, balance)
	if err != nil {
		return nil, err
	}
	return toBalanceResponse(row), nil
}

func (uc *UseCase) Report(ctx context.Context, filter repository.ReportFilter) ([]response.SalesGoalReportRowResponse, error) {
	if !uc.Auth.CanGetSalesOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	filter.AnalysisBase = normalizeAnalysisBase(filter.AnalysisBase)
	filter.Layout = strings.ToUpper(strings.TrimSpace(filter.Layout))
	filter.BreakBy = strings.ToUpper(strings.TrimSpace(filter.BreakBy))
	rows, err := uc.Repo.Report(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make([]response.SalesGoalReportRowResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, toReportRowResponse(row))
	}
	return out, nil
}

func periodFromDTO(dto request.CreateSalesGoalPeriodDTO) (*entity.Period, error) {
	if strings.TrimSpace(dto.Description) == "" {
		return nil, errorsuc.NewValidationError("description is required")
	}
	start, err := parseDate(dto.StartDate)
	if err != nil {
		return nil, errorsuc.NewValidationError("valid start_date is required")
	}
	end, err := parseDate(dto.EndDate)
	if err != nil {
		return nil, errorsuc.NewValidationError("valid end_date is required")
	}
	if end.Before(start) {
		return nil, errorsuc.NewValidationError("end_date cannot be before start_date")
	}
	periodType := strings.ToUpper(strings.TrimSpace(dto.PeriodType))
	switch periodType {
	case "MONTH", "WEEK", "CUSTOM":
	default:
		periodType = "MONTH"
	}
	return &entity.Period{Description: strings.TrimSpace(dto.Description), PeriodType: periodType, StartDate: start, EndDate: end, IsActive: true}, nil
}

func goalFromCreate(dto request.CreateSalesGoalDTO) (*entity.Goal, error) {
	if dto.RepresentativeCode == 0 || dto.PeriodCode == 0 {
		return nil, errorsuc.NewValidationError("representative_code and period_code are required")
	}
	if dto.AwardPct < 0 {
		return nil, errorsuc.NewValidationError("award_pct cannot be negative")
	}
	base := normalizeAnalysisBase(dto.AnalysisBase)
	if base == "" {
		return nil, errorsuc.NewValidationError("analysis_base must be SALES or INVOICING")
	}
	return &entity.Goal{RepresentativeCode: dto.RepresentativeCode, PeriodCode: dto.PeriodCode, AnalysisBase: base, AwardPct: dto.AwardPct, Notes: dto.Notes, IsActive: true}, nil
}

func goalFromUpdate(dto request.UpdateSalesGoalDTO) (*entity.Goal, error) {
	if dto.Code == 0 {
		return nil, errorsuc.NewValidationError("code is required")
	}
	g, err := goalFromCreate(request.CreateSalesGoalDTO{RepresentativeCode: dto.RepresentativeCode, PeriodCode: dto.PeriodCode, AnalysisBase: dto.AnalysisBase, AwardPct: dto.AwardPct, Notes: dto.Notes})
	if err != nil {
		return nil, err
	}
	g.Code = dto.Code
	g.IsActive = dto.IsActive
	return g, nil
}

func goalItemFromDTO(dto request.SalesGoalItemDTO) (*entity.GoalItem, error) {
	if dto.GoalCode == 0 {
		return nil, errorsuc.NewValidationError("goal_code is required")
	}
	if dto.TargetQuantity < 0 || dto.TargetValue < 0 || dto.BonusPct < 0 {
		return nil, errorsuc.NewValidationError("targets and bonus_pct cannot be negative")
	}
	targetType := strings.ToUpper(strings.TrimSpace(dto.TargetType))
	targets := 0
	if dto.ItemCode != nil {
		targets++
	}
	if dto.ItemClassificationCode != nil {
		targets++
	}
	if dto.ItemGroupCode != nil {
		targets++
	}
	if targets != 1 {
		return nil, errorsuc.NewValidationError("inform exactly one of item_code, item_classification_code or item_group_code")
	}
	switch {
	case targetType == "" && dto.ItemCode != nil:
		targetType = "ITEM"
	case targetType == "" && dto.ItemClassificationCode != nil:
		targetType = "CLASSIFICATION"
	case targetType == "" && dto.ItemGroupCode != nil:
		targetType = "GROUP"
	}
	if (targetType == "ITEM" && dto.ItemCode == nil) || (targetType == "CLASSIFICATION" && dto.ItemClassificationCode == nil) || (targetType == "GROUP" && dto.ItemGroupCode == nil) {
		return nil, errorsuc.NewValidationError("target_type does not match informed target")
	}
	if targetType != "ITEM" && targetType != "CLASSIFICATION" && targetType != "GROUP" {
		return nil, errorsuc.NewValidationError("target_type must be ITEM, CLASSIFICATION or GROUP")
	}
	return &entity.GoalItem{GoalCode: dto.GoalCode, TargetType: targetType, ItemCode: dto.ItemCode, ItemClassificationCode: dto.ItemClassificationCode, ItemGroupCode: dto.ItemGroupCode, SalesUOM: dto.SalesUOM, TargetQuantity: dto.TargetQuantity, TargetValue: dto.TargetValue, BonusPct: dto.BonusPct, IsActive: dto.IsActive}, nil
}

func groupTargetFromDTO(dto request.SalesGoalGroupTargetDTO) (*entity.GroupTarget, error) {
	if dto.PeriodCode == 0 || dto.CommercialGroupCode == 0 {
		return nil, errorsuc.NewValidationError("period_code and commercial_group_code are required")
	}
	goalType := normalizeAnalysisBase(dto.GoalType)
	if goalType == "" {
		return nil, errorsuc.NewValidationError("goal_type must be SALES or INVOICING")
	}
	if hasNegative(dto.MinimumValue, dto.MinimumBonusPct, dto.ProbableValue, dto.ProbableBonusPct, dto.IdealValue, dto.IdealBonusPct) {
		return nil, errorsuc.NewValidationError("goal values and bonus percentages cannot be negative")
	}
	return &entity.GroupTarget{PeriodCode: dto.PeriodCode, CommercialGroupCode: dto.CommercialGroupCode, GoalType: goalType, MinimumValue: dto.MinimumValue, MinimumBonusPct: dto.MinimumBonusPct, ProbableValue: dto.ProbableValue, ProbableBonusPct: dto.ProbableBonusPct, IdealValue: dto.IdealValue, IdealBonusPct: dto.IdealBonusPct, IsActive: dto.IsActive}, nil
}

func groupCustomerFromDTO(dto request.SalesGoalGroupCustomerDTO) (*entity.GroupCustomer, error) {
	if dto.GroupGoalID == 0 || dto.CustomerCode == 0 {
		return nil, errorsuc.NewValidationError("group_goal_id and customer_code are required")
	}
	if hasNegative(dto.MinimumValue, dto.MinimumBonusPct, dto.ProbableValue, dto.ProbableBonusPct, dto.IdealValue, dto.IdealBonusPct) {
		return nil, errorsuc.NewValidationError("goal values and bonus percentages cannot be negative")
	}
	return &entity.GroupCustomer{GroupGoalID: dto.GroupGoalID, CustomerCode: dto.CustomerCode, RepresentativeCode: dto.RepresentativeCode, MinimumValue: dto.MinimumValue, MinimumBonusPct: dto.MinimumBonusPct, ProbableValue: dto.ProbableValue, ProbableBonusPct: dto.ProbableBonusPct, IdealValue: dto.IdealValue, IdealBonusPct: dto.IdealBonusPct, IsActive: dto.IsActive}, nil
}

func balanceFromDTO(dto request.SalesGoalBalanceDTO) (*entity.Balance, error) {
	if dto.PeriodCode == 0 {
		return nil, errorsuc.NewValidationError("period_code is required")
	}
	scope := strings.ToUpper(strings.TrimSpace(dto.BalanceScope))
	if (scope == "REPRESENTATIVE" && dto.RepresentativeCode == nil) || (scope == "GROUP" && dto.CommercialGroupCode == nil) || (scope == "CUSTOMER" && dto.CustomerCode == nil) {
		return nil, errorsuc.NewValidationError("balance_scope does not match informed owner")
	}
	if scope != "REPRESENTATIVE" && scope != "GROUP" && scope != "CUSTOMER" {
		return nil, errorsuc.NewValidationError("balance_scope must be REPRESENTATIVE, GROUP or CUSTOMER")
	}
	goalType := normalizeAnalysisBase(dto.GoalType)
	if goalType == "" {
		return nil, errorsuc.NewValidationError("goal_type must be SALES or INVOICING")
	}
	if hasNegative(dto.RealizedValue, dto.IdealValue, dto.BalanceValue) {
		return nil, errorsuc.NewValidationError("balance values cannot be negative")
	}
	return &entity.Balance{PeriodCode: dto.PeriodCode, NextPeriodCode: dto.NextPeriodCode, BalanceScope: scope, RepresentativeCode: dto.RepresentativeCode, CommercialGroupCode: dto.CommercialGroupCode, CustomerCode: dto.CustomerCode, GoalType: goalType, RealizedValue: dto.RealizedValue, IdealValue: dto.IdealValue, BalanceValue: dto.BalanceValue, Notes: dto.Notes}, nil
}

func normalizeAnalysisBase(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	switch value {
	case "SALE", "SALES", "VENDA", "VENDAS":
		return "SALES"
	case "INVOICE", "INVOICING", "FATURAMENTO":
		return "INVOICING"
	default:
		return ""
	}
}

func parseDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(value))
}

func hasNegative(values ...float64) bool {
	for _, value := range values {
		if value < 0 {
			return true
		}
	}
	return false
}
