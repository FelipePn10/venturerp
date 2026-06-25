package shipment_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
)

func toShipmentResponse(s *entity.Shipment) *response.ShipmentResponse {
	if s == nil {
		return nil
	}
	var refTypeStr *string
	if s.ReferenceType != nil {
		rt := string(*s.ReferenceType)
		refTypeStr = &rt
	}
	return &response.ShipmentResponse{
		ID:                  s.ID,
		Code:                s.Code,
		ReferenceType:       refTypeStr,
		SalesOrderCode:      s.SalesOrderCode,
		PurchaseOrderCode:   s.PurchaseOrderCode,
		ProductionOrderCode: s.ProductionOrderCode,
		CarrierCode:         s.CarrierCode,
		Status:              string(s.Status),
		TotalVolumes:        s.TotalVolumes,
		TotalNetWeight:      s.TotalNetWeight,
		TotalGrossWeight:    s.TotalGrossWeight,
		TotalCubageM3:       s.TotalCubageM3,
		FreightModality:     s.FreightModality,
		FreightValue:        s.FreightValue,
		InsuranceValue:      s.InsuranceValue,
		VehiclePlate:        s.VehiclePlate,
		DriverName:          s.DriverName,
		DriverDocument:      s.DriverDocument,
		ANTTCode:            s.ANTTCode,
		Seals:               s.Seals,
		EstimatedDelivery:   s.EstimatedDelivery,
		FiscalExitID:        s.FiscalExitID,
		NFeNumber:           s.NFeNumber,
		NFeKey:              s.NFeKey,
		Notes:               s.Notes,
		SeparatedAt:         s.SeparatedAt,
		ConferredAt:         s.ConferredAt,
		ShippedAt:           s.ShippedAt,
		CancelledAt:         s.CancelledAt,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
		CreatedBy:           s.CreatedBy,
		Items:               toShipmentItemValues(s.Items),
		Volumes:             toShipmentVolumeValues(s.Volumes),
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
		HasDivergence:      it.HasDivergence(),
		UnitNetWeight:      it.UnitNetWeight,
		UnitGrossWeight:    it.UnitGrossWeight,
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

func toShipmentVolumeResponse(v *entity.ShipmentVolume) *response.ShipmentVolumeResponse {
	if v == nil {
		return nil
	}
	return &response.ShipmentVolumeResponse{
		ID:           v.ID,
		ShipmentID:   v.ShipmentID,
		VolumeNumber: v.VolumeNumber,
		PackageType:  v.PackageType,
		NetWeight:    v.NetWeight,
		GrossWeight:  v.GrossWeight,
		LengthCm:     v.LengthCm,
		WidthCm:      v.WidthCm,
		HeightCm:     v.HeightCm,
		CubageM3:     v.CubageM3,
		Marking:      v.Marking,
		Contents:     v.Contents,
		CreatedAt:    v.CreatedAt,
	}
}

func toShipmentVolumeValues(vols []*entity.ShipmentVolume) []response.ShipmentVolumeResponse {
	if len(vols) == 0 {
		return nil
	}
	out := make([]response.ShipmentVolumeResponse, 0, len(vols))
	for _, v := range vols {
		out = append(out, *toShipmentVolumeResponse(v))
	}
	return out
}

func toShipmentVolumeResponses(vols []*entity.ShipmentVolume) []*response.ShipmentVolumeResponse {
	out := make([]*response.ShipmentVolumeResponse, 0, len(vols))
	for _, v := range vols {
		out = append(out, toShipmentVolumeResponse(v))
	}
	return out
}

func toShipmentEventResponses(events []*entity.ShipmentEvent) []*response.ShipmentEventResponse {
	out := make([]*response.ShipmentEventResponse, 0, len(events))
	for _, e := range events {
		out = append(out, &response.ShipmentEventResponse{
			ID:        e.ID,
			Event:     e.Event,
			Note:      e.Note,
			CreatedBy: e.CreatedBy,
			CreatedAt: e.CreatedAt,
		})
	}
	return out
}
