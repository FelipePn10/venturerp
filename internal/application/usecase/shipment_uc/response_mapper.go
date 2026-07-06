package shipment_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/shipment/repository"
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

func toShipmentLoadResponse(l *entity.ShipmentLoad) *response.ShipmentLoadResponse {
	if l == nil {
		return nil
	}
	return &response.ShipmentLoadResponse{
		ID:                l.ID,
		Code:              l.Code,
		Status:            string(l.Status),
		Description:       l.Description,
		CarrierCode:       l.CarrierCode,
		VehiclePlate:      l.VehiclePlate,
		DriverName:        l.DriverName,
		DriverDocument:    l.DriverDocument,
		RouteCode:         l.RouteCode,
		Origin:            l.Origin,
		Destination:       l.Destination,
		DispatchBoxCode:   l.DispatchBoxCode,
		PlannedShipDate:   l.PlannedShipDate,
		EstimatedDelivery: l.EstimatedDelivery,
		StartedLoadingAt:  l.StartedLoadingAt,
		LoadedAt:          l.LoadedAt,
		ReleasedAt:        l.ReleasedAt,
		ShippedAt:         l.ShippedAt,
		CancelledAt:       l.CancelledAt,
		TotalShipments:    l.TotalShipments,
		TotalFiscalNotes:  l.TotalFiscalNotes,
		TotalVolumes:      l.TotalVolumes,
		TotalNetWeight:    l.TotalNetWeight,
		TotalGrossWeight:  l.TotalGrossWeight,
		TotalCubageM3:     l.TotalCubageM3,
		Notes:             l.Notes,
		CreatedAt:         l.CreatedAt,
		UpdatedAt:         l.UpdatedAt,
		CreatedBy:         l.CreatedBy,
		Shipments:         toShipmentLoadShipmentValues(l.Shipments),
		FiscalNotes:       toShipmentLoadFiscalNoteValues(l.FiscalNotes),
		Instructions:      toDeliveryInstructionValues(l.Instructions),
	}
}

func toShipmentLoadResponses(list []*entity.ShipmentLoad) []*response.ShipmentLoadResponse {
	out := make([]*response.ShipmentLoadResponse, 0, len(list))
	for _, l := range list {
		out = append(out, toShipmentLoadResponse(l))
	}
	return out
}

func toShipmentLoadShipmentResponse(s *entity.ShipmentLoadShipment) *response.ShipmentLoadShipmentResponse {
	if s == nil {
		return nil
	}
	return &response.ShipmentLoadShipmentResponse{
		ID:           s.ID,
		LoadID:       s.LoadID,
		LoadCode:     s.LoadCode,
		ShipmentID:   s.ShipmentID,
		ShipmentCode: s.ShipmentCode,
		Sequence:     s.Sequence,
		CreatedAt:    s.CreatedAt,
	}
}

func toShipmentLoadShipmentValues(list []*entity.ShipmentLoadShipment) []response.ShipmentLoadShipmentResponse {
	if len(list) == 0 {
		return nil
	}
	out := make([]response.ShipmentLoadShipmentResponse, 0, len(list))
	for _, item := range list {
		out = append(out, *toShipmentLoadShipmentResponse(item))
	}
	return out
}

func toShipmentLoadFiscalNoteResponse(n *entity.ShipmentLoadFiscalNote) *response.ShipmentLoadFiscalNoteResponse {
	if n == nil {
		return nil
	}
	return &response.ShipmentLoadFiscalNoteResponse{
		ID:           n.ID,
		LoadID:       n.LoadID,
		LoadCode:     n.LoadCode,
		ShipmentID:   n.ShipmentID,
		ShipmentCode: n.ShipmentCode,
		FiscalExitID: n.FiscalExitID,
		NFeNumber:    n.NFeNumber,
		NFeKey:       n.NFeKey,
		Sequence:     n.Sequence,
		CreatedAt:    n.CreatedAt,
	}
}

func toShipmentLoadFiscalNoteValues(list []*entity.ShipmentLoadFiscalNote) []response.ShipmentLoadFiscalNoteResponse {
	if len(list) == 0 {
		return nil
	}
	out := make([]response.ShipmentLoadFiscalNoteResponse, 0, len(list))
	for _, item := range list {
		out = append(out, *toShipmentLoadFiscalNoteResponse(item))
	}
	return out
}

func toDeliveryInstructionResponse(d *entity.DeliveryInstruction) *response.DeliveryInstructionResponse {
	if d == nil {
		return nil
	}
	return &response.DeliveryInstructionResponse{
		ID:          d.ID,
		LoadID:      d.LoadID,
		LoadCode:    d.LoadCode,
		CustomerID:  d.CustomerID,
		Title:       d.Title,
		Instruction: d.Instruction,
		Priority:    d.Priority,
		Active:      d.Active,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

func toDeliveryInstructionValues(list []*entity.DeliveryInstruction) []response.DeliveryInstructionResponse {
	if len(list) == 0 {
		return nil
	}
	out := make([]response.DeliveryInstructionResponse, 0, len(list))
	for _, item := range list {
		out = append(out, *toDeliveryInstructionResponse(item))
	}
	return out
}

func toDeliveryInstructionResponses(list []*entity.DeliveryInstruction) []*response.DeliveryInstructionResponse {
	out := make([]*response.DeliveryInstructionResponse, 0, len(list))
	for _, item := range list {
		out = append(out, toDeliveryInstructionResponse(item))
	}
	return out
}

func toDispatchBoxResponse(b *entity.DispatchBox) *response.DispatchBoxResponse {
	if b == nil {
		return nil
	}
	return &response.DispatchBoxResponse{
		ID:          b.ID,
		Code:        b.Code,
		Description: b.Description,
		WarehouseID: b.WarehouseID,
		Zone:        b.Zone,
		Active:      b.Active,
		CurrentLoad: b.CurrentLoad,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
	}
}

func toDispatchBoxResponses(list []*entity.DispatchBox) []*response.DispatchBoxResponse {
	out := make([]*response.DispatchBoxResponse, 0, len(list))
	for _, item := range list {
		out = append(out, toDispatchBoxResponse(item))
	}
	return out
}

func toLoadMonitorResponses(rows []*repository.LoadMonitorRow) []*response.LoadMonitorResponse {
	out := make([]*response.LoadMonitorResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, &response.LoadMonitorResponse{
			LoadCode:           r.LoadCode,
			Status:             string(r.Status),
			CarrierCode:        r.CarrierCode,
			VehiclePlate:       r.VehiclePlate,
			DriverName:         r.DriverName,
			DispatchBoxCode:    r.DispatchBoxCode,
			PlannedShipDate:    r.PlannedShipDate,
			EstimatedDelivery:  r.EstimatedDelivery,
			TotalShipments:     r.TotalShipments,
			TotalFiscalNotes:   r.TotalFiscalNotes,
			TotalVolumes:       r.TotalVolumes,
			TotalNetWeight:     r.TotalNetWeight,
			TotalGrossWeight:   r.TotalGrossWeight,
			TotalCubageM3:      r.TotalCubageM3,
			OpenShipments:      r.OpenShipments,
			SeparatedShipments: r.SeparatedShipments,
			ConferredShipments: r.ConferredShipments,
			ShippedShipments:   r.ShippedShipments,
		})
	}
	return out
}

func toSeparationMonitorResponses(rows []*repository.SeparationMonitorRow) []*response.SeparationMonitorResponse {
	out := make([]*response.SeparationMonitorResponse, 0, len(rows))
	for _, r := range rows {
		var loadStatus *string
		if r.LoadStatus != nil {
			s := string(*r.LoadStatus)
			loadStatus = &s
		}
		out = append(out, &response.SeparationMonitorResponse{
			ShipmentCode:     r.ShipmentCode,
			LoadCode:         r.LoadCode,
			ShipmentStatus:   string(r.ShipmentStatus),
			LoadStatus:       loadStatus,
			SalesOrderCode:   r.SalesOrderCode,
			CarrierCode:      r.CarrierCode,
			DispatchBoxCode:  r.DispatchBoxCode,
			TotalItems:       r.TotalItems,
			ConferredItems:   r.ConferredItems,
			DivergentItems:   r.DivergentItems,
			TotalVolumes:     r.TotalVolumes,
			TotalGrossWeight: r.TotalGrossWeight,
		})
	}
	return out
}

func toLogisticPanelResponse(s *repository.LogisticPanelSummary) *response.LogisticPanelResponse {
	if s == nil {
		return nil
	}
	return &response.LogisticPanelResponse{
		PlannedLoads:       s.PlannedLoads,
		ReleasedLoads:      s.ReleasedLoads,
		LoadingLoads:       s.LoadingLoads,
		LoadedLoads:        s.LoadedLoads,
		ShippedLoads:       s.ShippedLoads,
		CancelledLoads:     s.CancelledLoads,
		OpenShipments:      s.OpenShipments,
		SeparatedShipments: s.SeparatedShipments,
		ConferredShipments: s.ConferredShipments,
		BoxesOccupied:      s.BoxesOccupied,
		BoxesAvailable:     s.BoxesAvailable,
		TotalVolumes:       s.TotalVolumes,
		TotalGrossWeight:   s.TotalGrossWeight,
	}
}
