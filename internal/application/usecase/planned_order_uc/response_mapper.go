package planned_order_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
)

func toPlannedOrderResponse(o *entity.PlannedOrder) *response.PlannedOrderResponse {
	if o == nil {
		return nil
	}
	return &response.PlannedOrderResponse{
		Code:              o.Code,
		OrderNumber:       o.OrderNumber,
		ItemCode:          o.ItemCode,
		Mask:              o.Mask,
		Quantity:          o.Quantity,
		QuantityLoss:      o.QuantityLoss,
		QuantityCorrected: o.QuantityCorrected,
		OrderType:         string(o.OrderType),
		Status:            string(o.Status),
		PlanCode:          o.PlanCode,
		DemandType:        string(o.DemandType),
		DemandCode:        o.DemandCode,
		NeedDate:          o.NeedDate,
		StartDate:         o.StartDate,
		EndDate:           o.EndDate,
		CostCenterCode:    o.CostCenterCode,
		EmployeeCode:      o.EmployeeCode,
		MachineCode:       o.MachineCode,
		ProductionTime:    o.ProductionTime,
		Priority:          o.Priority,
		LLC:               o.LLC,
		Notes:             o.Notes,
		ParentOrderCode:   o.ParentOrderCode,
		SalesOrderCode:    o.SalesOrderCode,
		IsFirm:            o.IsFirm,
		IsActive:          o.IsActive,
		CreatedAt:         o.CreatedAt,
		UpdatedAt:         o.UpdatedAt,
		CreatedBy:         o.CreatedBy,
	}
}

func toPlannedOrderResponses(list []*entity.PlannedOrder) []*response.PlannedOrderResponse {
	out := make([]*response.PlannedOrderResponse, 0, len(list))
	for _, o := range list {
		out = append(out, toPlannedOrderResponse(o))
	}
	return out
}
