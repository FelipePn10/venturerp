package mrp_calculation_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/mrp_calculation/entity"
)

func toMRPCalculationLogResponse(l *entity.MRPCalculationLog) *response.MRPCalculationLogResponse {
	if l == nil {
		return nil
	}
	return &response.MRPCalculationLogResponse{
		Code:        l.Code,
		PlanCode:    l.PlanCode,
		StartedAt:   l.StartedAt,
		FinishedAt:  l.FinishedAt,
		Status:      l.Status,
		Errors:      l.Errors,
		TotalItems:  l.TotalItems,
		TotalOrders: l.TotalOrders,
		CreatedAt:   l.CreatedAt,
	}
}

func toMRPExceptionResponse(m *entity.MRPExceptionMessage) *response.MRPExceptionMessageResponse {
	if m == nil {
		return nil
	}
	return &response.MRPExceptionMessageResponse{
		Code:        m.Code,
		PlanCode:    m.PlanCode,
		ItemCode:    m.ItemCode,
		MessageType: string(m.MessageType),
		SourceCode:  m.SourceCode,
		SourceType:  m.SourceType,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
	}
}

func toMRPExceptionResponses(list []*entity.MRPExceptionMessage) []*response.MRPExceptionMessageResponse {
	out := make([]*response.MRPExceptionMessageResponse, 0, len(list))
	for _, m := range list {
		out = append(out, toMRPExceptionResponse(m))
	}
	return out
}

func toMRPItemProfileResponse(p *entity.MRPItemProfile) *response.MRPItemProfileResponse {
	if p == nil {
		return nil
	}
	return &response.MRPItemProfileResponse{
		ItemCode:        p.ItemCode,
		PlanCode:        p.PlanCode,
		CalculationDate: p.CalculationDate,
		Demand:          p.Demand,
		OrdersPlanned:   p.OrdersPlanned,
		OrdersFirm:      p.OrdersFirm,
		StockProjected:  p.StockProjected,
		LLC:             p.LLC,
		NeedDate:        p.NeedDate,
		CreatedAt:       p.CreatedAt,
	}
}

func toMRPItemProfileResponses(list []*entity.MRPItemProfile) []*response.MRPItemProfileResponse {
	out := make([]*response.MRPItemProfileResponse, 0, len(list))
	for _, p := range list {
		out = append(out, toMRPItemProfileResponse(p))
	}
	return out
}

func toConfiguredItemRuleResponse(r *entity.ConfiguredItemRule) *response.ConfiguredItemRuleResponse {
	if r == nil {
		return nil
	}
	return &response.ConfiguredItemRuleResponse{
		Code:      r.Code,
		ItemCode:  r.ItemCode,
		TableType: r.TableType,
		FieldName: r.FieldName,
		RuleType:  r.RuleType,
		RuleValue: r.RuleValue,
		Sequence:  r.Sequence,
		IsActive:  r.IsActive,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
		CreatedBy: r.CreatedBy,
	}
}

func toConfiguredItemRuleResponses(list []*entity.ConfiguredItemRule) []*response.ConfiguredItemRuleResponse {
	out := make([]*response.ConfiguredItemRuleResponse, 0, len(list))
	for _, r := range list {
		out = append(out, toConfiguredItemRuleResponse(r))
	}
	return out
}
