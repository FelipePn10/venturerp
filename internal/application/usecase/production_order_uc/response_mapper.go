package production_order_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/production_order/entity"
	"time"
)

func toProductionOrderResponse(o *entity.ProductionOrder) *response.ProductionOrderResponse {
	if o == nil {
		return nil
	}
	return &response.ProductionOrderResponse{
		ID:             o.ID,
		OrderNumber:    o.OrderNumber,
		PlannedOrderID: o.PlannedOrderID,
		ItemCode:       o.ItemCode,
		Mask:           o.Mask,
		PlannedQty:     o.PlannedQty,
		ProducedQty:    o.ProducedQty,
		ScrappedQty:    o.ScrappedQty,
		Status:         string(o.Status),
		StartDate:      o.StartDate,
		EndDate:        o.EndDate,
		MachineID:      o.MachineID,
		CostCenterID:   o.CostCenterID,
		EmployeeID:     o.EmployeeID,
		Priority:       o.Priority,
		Notes:          o.Notes,
		IsActive:       o.IsActive,
		CreatedAt:      o.CreatedAt,
		UpdatedAt:      o.UpdatedAt,
		CreatedBy:      o.CreatedBy,
	}
}

func toProductionOrderResponses(list []*entity.ProductionOrder) []*response.ProductionOrderResponse {
	out := make([]*response.ProductionOrderResponse, 0, len(list))
	for _, o := range list {
		out = append(out, toProductionOrderResponse(o))
	}
	return out
}

func toProductionAppointmentResponse(a *entity.ProductionAppointment) *response.ProductionAppointmentResponse {
	if a == nil {
		return nil
	}
	return &response.ProductionAppointmentResponse{
		ID:                a.ID,
		ProductionOrderID: a.ProductionOrderID,
		MachineID:         a.MachineID,
		EmployeeID:        a.EmployeeID,
		AppointmentDate:   a.AppointmentDate,
		StartTime:         a.StartTime,
		EndTime:           a.EndTime,
		ProducedQty:       a.ProducedQty,
		ScrappedQty:       a.ScrappedQty,
		ScrapReason:       a.ScrapReason,
		Notes:             a.Notes,
		CreatedAt:         a.CreatedAt,
		UpdatedAt:         a.UpdatedAt,
		CreatedBy:         a.CreatedBy,
	}
}

func toProductionAppointmentResponses(list []*entity.ProductionAppointment) []*response.ProductionAppointmentResponse {
	out := make([]*response.ProductionAppointmentResponse, 0, len(list))
	for _, a := range list {
		out = append(out, toProductionAppointmentResponse(a))
	}
	return out
}

func toProductionConsumptionResponse(c *entity.ProductionConsumption) *response.ProductionConsumptionResponse {
	if c == nil {
		return nil
	}
	return &response.ProductionConsumptionResponse{
		ID:                c.ID,
		ProductionOrderID: c.ProductionOrderID,
		AppointmentID:     c.AppointmentID,
		ItemCode:          c.ItemCode,
		ConsumedQty:       c.ConsumedQty,
		WarehouseID:       c.WarehouseID,
		Lot:               c.Lot,
		ConsumptionDate:   c.ConsumptionDate,
		Notes:             c.Notes,
		CreatedAt:         c.CreatedAt,
		CreatedBy:         c.CreatedBy,
	}
}

func toProductionConsumptionResponses(list []*entity.ProductionConsumption) []*response.ProductionConsumptionResponse {
	out := make([]*response.ProductionConsumptionResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toProductionConsumptionResponse(c))
	}
	return out
}

// ToProductionOrderCostResponse traduz o fechamento de custo para o contrato da
// API, com nomes estáveis em snake_case.
func ToProductionOrderCostResponse(c *entity.ProductionOrderCost) *response.ProductionOrderCostResponse {
	if c == nil {
		return nil
	}
	out := &response.ProductionOrderCostResponse{
		ProductionOrderID: c.ProductionOrderID,
		ProducedQty:       c.ProducedQty,
		MaterialCostReal:  c.MaterialCostReal,
		LaborCostReal:     c.LaborCostReal,
		OverheadCostReal:  c.OverheadCostReal,
		TotalCostReal:     c.TotalCostReal,
		UnitCostReal:      c.UnitCostReal,
		MaterialCostStd:   c.MaterialCostStd,
		LaborCostStd:      c.LaborCostStd,
		OverheadCostStd:   c.OverheadCostStd,
		TotalCostStd:      c.TotalCostStd,
		MaterialVariance:  c.MaterialVariance,
		LaborVariance:     c.LaborVariance,
		OverheadVariance:  c.OverheadVariance,
		TotalVariance:     c.TotalVariance,
		Currency:          c.Currency,
	}
	if c.TotalCostStd != 0 {
		out.VariancePct = c.TotalVariance / c.TotalCostStd * 100
	}
	if !c.SettledAt.IsZero() {
		out.SettledAt = c.SettledAt.Format(time.RFC3339)
	}
	return out
}
