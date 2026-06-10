package overhead_allocation_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/overhead_allocation/entity"
)

func toOverheadAllocationResponse(oa *entity.OverheadAllocation) *response.OverheadAllocationResponse {
	if oa == nil {
		return nil
	}
	return &response.OverheadAllocationResponse{
		Code:            oa.Code,
		CostCenterCode:  oa.CostCenterCode,
		PlanAccountCode: oa.PlanAccountCode,
		AccountCode:     oa.AccountCode,
		PeriodStart:     oa.PeriodStart,
		PeriodEnd:       oa.PeriodEnd,
		AllocationType:  oa.AllocationType,
		BaseCode:        oa.BaseCode,
		Targets:         toAllocationTargetValues(oa.Targets),
		CreatedAt:       oa.CreatedAt,
		UpdatedAt:       oa.UpdatedAt,
		CreatedBy:       oa.CreatedBy,
	}
}

func toOverheadAllocationResponses(list []*entity.OverheadAllocation) []*response.OverheadAllocationResponse {
	out := make([]*response.OverheadAllocationResponse, 0, len(list))
	for _, oa := range list {
		out = append(out, toOverheadAllocationResponse(oa))
	}
	return out
}

func toAllocationTargetValues(targets []*entity.AllocationTarget) []response.AllocationTargetResponse {
	if len(targets) == 0 {
		return nil
	}
	out := make([]response.AllocationTargetResponse, 0, len(targets))
	for _, t := range targets {
		out = append(out, response.AllocationTargetResponse{
			Code:           t.Code,
			OverheadCode:   t.OverheadCode,
			CostCenterCode: t.CostCenterCode,
			Percentage:     t.Percentage,
			Amount:         t.Amount,
			CreatedAt:      t.CreatedAt,
		})
	}
	return out
}
