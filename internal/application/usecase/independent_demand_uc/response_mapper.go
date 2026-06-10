package independent_demand_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/independent_demand/entity"
)

func toIndependentDemandResponse(d *entity.IndependentDemand) *response.IndependentDemandResponse {
	if d == nil {
		return nil
	}
	return &response.IndependentDemandResponse{
		CodeDemand:     d.CodeDemand,
		ItemCode:       d.ItemCode,
		Mask:           d.Mask,
		CostCenterCode: d.CostCenterCode,
		Quantity:       d.Quantity,
		DemandDate:     d.DemandDate,
		IsActive:       d.IsActive,
		CreatedAt:      d.CreatedAt,
		UpdatedAt:      d.UpdatedAt,
		CreatedBy:      d.CreatedBy,
	}
}

func toIndependentDemandResponses(list []*entity.IndependentDemand) []*response.IndependentDemandResponse {
	out := make([]*response.IndependentDemandResponse, 0, len(list))
	for _, d := range list {
		out = append(out, toIndependentDemandResponse(d))
	}
	return out
}
