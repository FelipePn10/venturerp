package stock_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	"github.com/shopspring/decimal"
)

// num e numPtr convertem na fronteira do JSON: o domínio calcula em decimal e a
// resposta continua número, que é o contrato que o frontend já consome.
func num(d decimal.Decimal) float64 { return d.InexactFloat64() }

func numPtr(d *decimal.Decimal) *float64 {
	if d == nil {
		return nil
	}
	f := d.InexactFloat64()
	return &f
}

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
		Quantity:       num(m.Quantity),
		UnitPrice:      num(m.UnitPrice),
		TotalPrice:     num(m.TotalPrice),
		ReferenceType:  m.ReferenceType,
		ReferenceCode:  m.ReferenceCode,
		Lot:            m.Lot,
		SerialNumber:   m.SerialNumber,
		Batch:          m.Batch,
		ExpirationDate: m.ExpirationDate,
		Notes:          m.Notes,
		Address:        m.Address,
		AddressTo:      m.AddressTo,
		CreatedAt:      m.CreatedAt,
		CreatedBy:      m.CreatedBy,

		MovementTypeCode:     m.MovementTypeCode,
		CostCenterCode:       m.CostCenterCode,
		ReversesMovementID:   m.ReversesMovementID,
		ReversedByMovementID: m.ReversedByMovementID,
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
		Quantity:          num(r.Quantity),
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
		Quantity:       num(b.Quantity),
		ReservedQty:    num(b.ReservedQty),
		AvailableQty:   num(b.AvailableQty),
		MinimumStock:   num(b.MinimumStock),
		MaximumStock:   num(b.MaximumStock),
		SafetyStock:    num(b.SafetyStock),
		AvgCost:        num(b.AvgCost),
		LastCost:       num(b.LastCost),
		TotalCost:      num(b.TotalCost),
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

		AdjustmentValue: numPtr(i.AdjustmentValue),
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
		SystemQty:        num(it.SystemQty),
		CountedQty:       numPtr(it.CountedQty),
		DifferenceQty:    numPtr(it.DifferenceQty),
		UnitCost:         numPtr(it.UnitCost),
		AdjustmentType:   it.AdjustmentType,
		AdjustmentReason: it.AdjustmentReason,
		CountedBy:        it.CountedBy,
		CountedAt:        it.CountedAt,
		IsAdjusted:       it.IsAdjusted,

		CountNumber:          it.CountNumber,
		PreviousCountedQty:   numPtr(it.PreviousCountedQty),
		AdjustmentMovementID: it.AdjustmentMovementID,
		CreatedAt:            it.CreatedAt,
	}
}

func toPhysicalInventoryItemResponses(list []*entity.PhysicalInventoryItem) []*response.PhysicalInventoryItemResponse {
	out := make([]*response.PhysicalInventoryItemResponse, 0, len(list))
	for _, it := range list {
		out = append(out, toPhysicalInventoryItemResponse(it))
	}
	return out
}

// ToStockMovementResponse expõe o mapeamento para outros pacotes (operações de
// estoque devolvem o movimento no mesmo formato do lançamento).
func ToStockMovementResponse(m *entity.StockMovement) *response.StockMovementResponse {
	return toStockMovementResponse(m)
}

// ToStockMovementResponses idem, para listas.
func ToStockMovementResponses(list []*entity.StockMovement) []*response.StockMovementResponse {
	return toStockMovementResponses(list)
}
