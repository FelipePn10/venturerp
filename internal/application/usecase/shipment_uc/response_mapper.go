package shipment_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
)

func toShipmentResponse(s *entity.Shipment) *response.ShipmentResponse {
	if s == nil {
		return nil
	}
	return &response.ShipmentResponse{
		ID:             s.ID,
		Code:           s.Code,
		SalesOrderCode: s.SalesOrderCode,
		CarrierCode:    s.CarrierCode,
		Status:         string(s.Status),
		TotalVolumes:   s.TotalVolumes,
		TotalWeight:    s.TotalWeight,
		Notes:          s.Notes,
		ShippedAt:      s.ShippedAt,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
		CreatedBy:      s.CreatedBy,
		Items:          toShipmentItemValues(s.Items),
	}
}

func toShipmentResponses(list []*entity.Shipment) []*response.ShipmentResponse {
	out := make([]*response.ShipmentResponse, 0, len(list))
	for _, s := range list {
		out = append(out, toShipmentResponse(s))
	}
	return out
}

func toShipmentItemResponse(it *entity.ShipmentItem) *response.ShipmentItemResponse {
	if it == nil {
		return nil
	}
	return &response.ShipmentItemResponse{
		ID:                 it.ID,
		ShipmentID:         it.ShipmentID,
		Sequence:           it.Sequence,
		ItemCode:           it.ItemCode,
		SalesOrderItemCode: it.SalesOrderItemCode,
		WarehouseID:        it.WarehouseID,
		Quantity:           it.Quantity,
		ConferredQty:       it.ConferredQty,
		IsConferred:        it.IsConferred,
		Notes:              it.Notes,
		CreatedAt:          it.CreatedAt,
	}
}

func toShipmentItemValues(items []*entity.ShipmentItem) []response.ShipmentItemResponse {
	if len(items) == 0 {
		return nil
	}
	out := make([]response.ShipmentItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, *toShipmentItemResponse(it))
	}
	return out
}
