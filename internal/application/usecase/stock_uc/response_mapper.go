package stock_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
)

func toStockMovementResponse(m *entity.StockMovement) *response.StockMovementResponse {
	if m == nil {
		return nil
	}
	return &response.StockMovementResponse{
		ID:             m.ID,
		ItemCode:       m.ItemCode,
		Mask:           m.Mask,
		WarehouseID:    m.WarehouseID,
		MovementType:   m.MovementType,
		Quantity:       m.Quantity,
		UnitPrice:      m.UnitPrice,
		TotalPrice:     m.TotalPrice,
		ReferenceType:  m.ReferenceType,
		ReferenceCode:  m.ReferenceCode,
		Lot:            m.Lot,
		SerialNumber:   m.SerialNumber,
		Batch:          m.Batch,
		ExpirationDate: m.ExpirationDate,
		Notes:          m.Notes,
		CreatedAt:      m.CreatedAt,
		CreatedBy:      m.CreatedBy,
	}
}

func toStockMovementResponses(list []*entity.StockMovement) []*response.StockMovementResponse {
	out := make([]*response.StockMovementResponse, 0, len(list))
	for _, m := range list {
		out = append(out, toStockMovementResponse(m))
	}
	return out
}

func toStockReservationResponse(r *entity.StockReservation) *response.StockReservationResponse {
	if r == nil {
		return nil
	}
	return &response.StockReservationResponse{
		ID:                r.ID,
		ItemCode:          r.ItemCode,
		Mask:              r.Mask,
		WarehouseID:       r.WarehouseID,
		Quantity:          r.Quantity,
		ReferenceType:     r.ReferenceType,
		ReferenceCode:     r.ReferenceCode,
		ReferenceItemCode: r.ReferenceItemCode,
		ReservationDate:   r.ReservationDate,
		ExpirationDate:    r.ExpirationDate,
		Status:            r.Status,
		Notes:             r.Notes,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
		CreatedBy:         r.CreatedBy,
	}
}

func toStockBalanceResponse(b *entity.StockBalance) *response.StockBalanceResponse {
	if b == nil {
		return nil
	}
	return &response.StockBalanceResponse{
		ID:             b.ID,
		ItemCode:       b.ItemCode,
		Mask:           b.Mask,
		WarehouseID:    b.WarehouseID,
		Quantity:       b.Quantity,
		ReservedQty:    b.ReservedQty,
		AvailableQty:   b.AvailableQty,
		MinimumStock:   b.MinimumStock,
		MaximumStock:   b.MaximumStock,
		SafetyStock:    b.SafetyStock,
		AvgCost:        b.AvgCost,
		LastCost:       b.LastCost,
		TotalCost:      b.TotalCost,
		LastMovementAt: b.LastMovementAt,
		UpdatedAt:      b.UpdatedAt,
	}
}

func toStockBalanceResponses(list []*entity.StockBalance) []*response.StockBalanceResponse {
	out := make([]*response.StockBalanceResponse, 0, len(list))
	for _, b := range list {
		out = append(out, toStockBalanceResponse(b))
	}
	return out
}

func toPhysicalInventoryResponse(i *entity.PhysicalInventory) *response.PhysicalInventoryResponse {
	if i == nil {
		return nil
	}
	return &response.PhysicalInventoryResponse{
		ID:           i.ID,
		Code:         i.Code,
		Description:  i.Description,
		WarehouseID:  i.WarehouseID,
		StartDate:    i.StartDate,
		EndDate:      i.EndDate,
		Status:       i.Status,
		TotalItems:   i.TotalItems,
		CountedItems: i.CountedItems,
		Notes:        i.Notes,
		CreatedAt:    i.CreatedAt,
		UpdatedAt:    i.UpdatedAt,
		CreatedBy:    i.CreatedBy,
	}
}

func toPhysicalInventoryResponses(list []*entity.PhysicalInventory) []*response.PhysicalInventoryResponse {
	out := make([]*response.PhysicalInventoryResponse, 0, len(list))
	for _, i := range list {
		out = append(out, toPhysicalInventoryResponse(i))
	}
	return out
}

func toPhysicalInventoryItemResponse(it *entity.PhysicalInventoryItem) *response.PhysicalInventoryItemResponse {
	if it == nil {
		return nil
	}
	return &response.PhysicalInventoryItemResponse{
		ID:               it.ID,
		InventoryID:      it.InventoryID,
		ItemCode:         it.ItemCode,
		Mask:             it.Mask,
		WarehouseID:      it.WarehouseID,
		SystemQty:        it.SystemQty,
		CountedQty:       it.CountedQty,
		DifferenceQty:    it.DifferenceQty,
		UnitCost:         it.UnitCost,
		AdjustmentType:   it.AdjustmentType,
		AdjustmentReason: it.AdjustmentReason,
		CountedBy:        it.CountedBy,
		CountedAt:        it.CountedAt,
		IsAdjusted:       it.IsAdjusted,
		CreatedAt:        it.CreatedAt,
	}
}

func toPhysicalInventoryItemResponses(list []*entity.PhysicalInventoryItem) []*response.PhysicalInventoryItemResponse {
	out := make([]*response.PhysicalInventoryItemResponse, 0, len(list))
	for _, it := range list {
		out = append(out, toPhysicalInventoryItemResponse(it))
	}
	return out
}
