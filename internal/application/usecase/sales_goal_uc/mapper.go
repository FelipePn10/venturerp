package sales_goal_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_goal/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_goal/repository"
)

func toPeriodResponse(p *entity.Period) *response.SalesGoalPeriodResponse {
	if p == nil {
		return nil
	}
	return &response.SalesGoalPeriodResponse{Code: p.Code, Description: p.Description, PeriodType: p.PeriodType, StartDate: p.StartDate, EndDate: p.EndDate, IsActive: p.IsActive, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt}
}

func toGoalResponse(g *entity.Goal) *response.SalesGoalResponse {
	if g == nil {
		return nil
	}
	out := &response.SalesGoalResponse{Code: g.Code, RepresentativeCode: g.RepresentativeCode, PeriodCode: g.PeriodCode, AnalysisBase: g.AnalysisBase, AwardPct: g.AwardPct.InexactFloat64(), Notes: g.Notes, IsActive: g.IsActive, CreatedAt: g.CreatedAt, UpdatedAt: g.UpdatedAt}
	for _, item := range g.Items {
		out.Items = append(out.Items, *toGoalItemResponse(item))
	}
	return out
}

func toGoalItemResponse(i *entity.GoalItem) *response.SalesGoalItemResponse {
	if i == nil {
		return nil
	}
	return &response.SalesGoalItemResponse{ID: i.ID, GoalCode: i.GoalCode, TargetType: i.TargetType, ItemCode: i.ItemCode, ItemClassificationCode: i.ItemClassificationCode, ItemGroupCode: i.ItemGroupCode, SalesUOM: i.SalesUOM, TargetQuantity: i.TargetQuantity.InexactFloat64(), TargetValue: i.TargetValue.InexactFloat64(), BonusPct: i.BonusPct.InexactFloat64(), IsActive: i.IsActive, CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt}
}

func toGroupTargetResponse(g *entity.GroupTarget) *response.SalesGoalGroupTargetResponse {
	if g == nil {
		return nil
	}
	return &response.SalesGoalGroupTargetResponse{ID: g.ID, PeriodCode: g.PeriodCode, CommercialGroupCode: g.CommercialGroupCode, GoalType: g.GoalType, MinimumValue: g.MinimumValue.InexactFloat64(), MinimumBonusPct: g.MinimumBonusPct.InexactFloat64(), ProbableValue: g.ProbableValue.InexactFloat64(), ProbableBonusPct: g.ProbableBonusPct.InexactFloat64(), IdealValue: g.IdealValue.InexactFloat64(), IdealBonusPct: g.IdealBonusPct.InexactFloat64(), IsActive: g.IsActive, CreatedAt: g.CreatedAt, UpdatedAt: g.UpdatedAt}
}

func toGroupCustomerResponse(c *entity.GroupCustomer) *response.SalesGoalGroupCustomerResponse {
	if c == nil {
		return nil
	}
	return &response.SalesGoalGroupCustomerResponse{ID: c.ID, GroupGoalID: c.GroupGoalID, CustomerCode: c.CustomerCode, RepresentativeCode: c.RepresentativeCode, MinimumValue: c.MinimumValue.InexactFloat64(), MinimumBonusPct: c.MinimumBonusPct.InexactFloat64(), ProbableValue: c.ProbableValue.InexactFloat64(), ProbableBonusPct: c.ProbableBonusPct.InexactFloat64(), IdealValue: c.IdealValue.InexactFloat64(), IdealBonusPct: c.IdealBonusPct.InexactFloat64(), IsActive: c.IsActive, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}
}

func toBalanceResponse(b *entity.Balance) *response.SalesGoalBalanceResponse {
	if b == nil {
		return nil
	}
	return &response.SalesGoalBalanceResponse{ID: b.ID, PeriodCode: b.PeriodCode, NextPeriodCode: b.NextPeriodCode, BalanceScope: b.BalanceScope, RepresentativeCode: b.RepresentativeCode, CommercialGroupCode: b.CommercialGroupCode, CustomerCode: b.CustomerCode, GoalType: b.GoalType, RealizedValue: b.RealizedValue.InexactFloat64(), IdealValue: b.IdealValue.InexactFloat64(), BalanceValue: b.BalanceValue.InexactFloat64(), Notes: b.Notes, CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt}
}

func toReportRowResponse(row repository.ReportRow) response.SalesGoalReportRowResponse {
	return response.SalesGoalReportRowResponse{Scope: row.Scope, RepresentativeCode: row.RepresentativeCode, CommercialGroupCode: row.CommercialGroupCode, CustomerCode: row.CustomerCode, PeriodCode: row.PeriodCode, PeriodDescription: row.PeriodDescription, AnalysisBase: row.AnalysisBase, TargetValue: row.TargetValue.InexactFloat64(), TargetQuantity: row.TargetQuantity.InexactFloat64(), RealizedValue: row.RealizedValue.InexactFloat64(), RealizedQuantity: row.RealizedQuantity.InexactFloat64(), BalanceValue: row.BalanceValue.InexactFloat64(), AchievementPct: row.AchievementPct.InexactFloat64(), BonusPct: row.BonusPct.InexactFloat64(), Status: row.Status}
}
