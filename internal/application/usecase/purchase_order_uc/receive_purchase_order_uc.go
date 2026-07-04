package purchase_order_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	poentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	porepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
	stockentity "github.com/FelipePn10/panossoerp/internal/domain/stock/entity"
	stockrepo "github.com/FelipePn10/panossoerp/internal/domain/stock/repository"
)

// ReceivingInspectionGate hooks receiving inspection into physical receipt
// (FINS0212). When a received line matches an active inspection route the material
// is routed into the inspection warehouse and an inspection order is opened.
// Implemented by the procurement use case; kept as a port to avoid coupling.
type ReceivingInspectionGate interface {
	// ResolveInspectionRoute returns the inspection warehouse for an active route
	// matching the item, and whether a route matched.
	ResolveInspectionRoute(ctx context.Context, itemCode int64, mask string) (inspectionWarehouseID int64, matched bool)
	// OpenInspectionOrderFromReceipt opens an inspection order for material routed
	// into the inspection warehouse; returns the created order id and number.
	OpenInspectionOrderFromReceipt(ctx context.Context, itemCode int64, mask string, quantity float64, inspectionWarehouseID int64, supplierCode, purchaseOrderCode, purchaseOrderItemCode *int64, lot *string) (int64, int64, error)
}

type ReceivePurchaseOrderUseCase struct {
	Repo      porepo.PurchaseOrderRepository
	StockRepo stockrepo.StockRepository
	Auth      ports.AuthService
	// Inspection is optional; nil disables auto-inspection on receipt.
	Inspection ReceivingInspectionGate
}

func (uc *ReceivePurchaseOrderUseCase) Execute(ctx context.Context, dto request.ReceivePurchaseOrderDTO) (*response.PurchaseOrderReceiptResponse, error) {
	if !uc.Auth.CanUpdatePurchaseOrder(ctx) || !uc.Auth.CanCreateStockMovement(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	actor, err := uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}
	if dto.PurchaseOrderCode <= 0 {
		return nil, fmt.Errorf("purchase_order_code is required")
	}
	if len(dto.Items) == 0 {
		return nil, fmt.Errorf("at least one receipt item is required")
	}

	order, err := uc.Repo.GetByCode(ctx, dto.PurchaseOrderCode)
	if err != nil {
		return nil, err
	}
	if order.Status == poentity.PurchaseOrderStatusCANCELLED || order.Status == poentity.PurchaseOrderStatusRECEIVED {
		return nil, fmt.Errorf("purchase order %d cannot receive in status %s", order.Code, order.Status)
	}
	lines, err := uc.Repo.ListItems(ctx, dto.PurchaseOrderCode)
	if err != nil {
		return nil, err
	}
	byCode := make(map[int64]*poentity.PurchaseOrderItem, len(lines))
	for _, line := range lines {
		byCode[line.Code] = line
	}

	receivedByLine := make(map[int64]float64, len(dto.Items))
	receiptLines := make([]response.PurchaseReceiptLine, 0, len(dto.Items))
	movements := make([]response.StockMovementResponse, 0, len(dto.Items))
	inspectionOrders := make([]response.ReceiptInspectionOrder, 0)
	refType := stockentity.ReferenceTypePurchaseOrder
	refCode := dto.PurchaseOrderCode

	for _, item := range dto.Items {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("quantity must be positive for purchase order item %d", item.PurchaseOrderItemCode)
		}
		if item.WarehouseID <= 0 {
			return nil, fmt.Errorf("warehouse_id is required for purchase order item %d", item.PurchaseOrderItemCode)
		}
		line := byCode[item.PurchaseOrderItemCode]
		if line == nil {
			return nil, fmt.Errorf("purchase order item %d does not belong to order %d", item.PurchaseOrderItemCode, dto.PurchaseOrderCode)
		}
		if line.Status == poentity.PurchaseOrderItemStatusCANCELLED {
			return nil, fmt.Errorf("purchase order item %d is cancelled", item.PurchaseOrderItemCode)
		}
		remaining := line.RequestedQty - line.ReceivedQty - line.CancelledQty
		toleranceRemaining := line.TolerancePct/100*line.RequestedQty + line.CancelledToleranceQty
		if item.Quantity > remaining+toleranceRemaining+0.0001 {
			return nil, fmt.Errorf("receipt quantity %.4f exceeds remaining %.4f for purchase order item %d", item.Quantity, remaining, item.PurchaseOrderItemCode)
		}

		stockQty := item.Quantity
		unitPrice := line.UnitPrice
		if line.InternalQty > 0 && line.RequestedQty > 0 {
			stockQty = item.Quantity * (line.InternalQty / line.RequestedQty)
		}
		if line.InternalPrice > 0 {
			unitPrice = line.InternalPrice
		}
		totalPrice := stockQty * unitPrice
		notes := mergeReceiptNotes(dto.Notes, item.Notes)
		expirationDate, err := parseReceiptDate(item.ExpirationDate)
		if err != nil {
			return nil, err
		}

		// FINS0212: if an active inspection route matches, receive into the
		// inspection warehouse instead of the requested one so nothing reaches
		// available stock without a laudo.
		targetWarehouse := item.WarehouseID
		underInspection := false
		if uc.Inspection != nil {
			if inspWH, matched := uc.Inspection.ResolveInspectionRoute(ctx, line.ItemCode, line.Mask); matched && inspWH > 0 {
				targetWarehouse = inspWH
				underInspection = true
			}
		}

		mov := &stockentity.StockMovement{
			ItemCode:       line.ItemCode,
			Mask:           line.Mask,
			WarehouseID:    targetWarehouse,
			MovementType:   stockentity.MovementTypeIn,
			Quantity:       stockQty,
			UnitPrice:      unitPrice,
			TotalPrice:     totalPrice,
			ReferenceType:  &refType,
			ReferenceCode:  &refCode,
			Lot:            item.Lot,
			SerialNumber:   item.SerialNumber,
			Batch:          item.Batch,
			ExpirationDate: expirationDate,
			Notes:          notes,
			CreatedBy:      actor,
		}
		created, err := uc.StockRepo.CreateMovement(ctx, mov)
		if err != nil {
			return nil, err
		}
		movements = append(movements, stockMovementReceiptResponse(created))

		if underInspection {
			orderID, orderNumber, err := uc.Inspection.OpenInspectionOrderFromReceipt(
				ctx, line.ItemCode, line.Mask, stockQty, targetWarehouse,
				order.SupplierCode, &dto.PurchaseOrderCode, &line.Code, item.Lot)
			if err != nil {
				return nil, err
			}
			inspectionOrders = append(inspectionOrders, response.ReceiptInspectionOrder{
				InspectionOrderID:     orderID,
				OrderNumber:           orderNumber,
				PurchaseOrderItemCode: line.Code,
				ItemCode:              line.ItemCode,
				WarehouseID:           targetWarehouse,
				Quantity:              stockQty,
			})
		}

		receivedByLine[line.Code] += item.Quantity
		receiptLines = append(receiptLines, response.PurchaseReceiptLine{
			PurchaseOrderItemCode: line.Code,
			ItemCode:              line.ItemCode,
			Mask:                  line.Mask,
			WarehouseID:           targetWarehouse,
			Quantity:              item.Quantity,
			StockQuantity:         stockQty,
			RemainingQty:          remaining - item.Quantity,
			UnderInspection:       underInspection,
		})
	}

	matched, err := uc.Repo.RegisterItemReceipts(ctx, dto.PurchaseOrderCode, receivedByLine)
	if err != nil {
		return nil, err
	}
	if matched != len(receivedByLine) {
		return nil, fmt.Errorf("registered %d receipt lines, expected %d", matched, len(receivedByLine))
	}
	updated, err := uc.Repo.GetByCode(ctx, dto.PurchaseOrderCode)
	if err != nil {
		return nil, err
	}
	updated.Items, err = uc.Repo.ListItems(ctx, dto.PurchaseOrderCode)
	if err != nil {
		return nil, err
	}
	return &response.PurchaseOrderReceiptResponse{
		PurchaseOrder:    toPurchaseOrderResponse(updated),
		ReceivedLines:    receiptLines,
		Movements:        movements,
		InspectionOrders: inspectionOrders,
	}, nil
}

func parseReceiptDate(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, fmt.Errorf("invalid expiration_date %q: %w", *s, err)
	}
	return &t, nil
}

func mergeReceiptNotes(header, line *string) *string {
	switch {
	case header != nil && line != nil && *header != "" && *line != "":
		merged := *header + " | " + *line
		return &merged
	case line != nil && *line != "":
		return line
	case header != nil && *header != "":
		return header
	default:
		return nil
	}
}

func stockMovementReceiptResponse(m *stockentity.StockMovement) response.StockMovementResponse {
	return response.StockMovementResponse{
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
