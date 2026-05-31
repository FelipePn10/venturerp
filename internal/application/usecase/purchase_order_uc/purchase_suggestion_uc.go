package purchase_order_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	plannedentity "github.com/FelipePn10/panossoerp/internal/domain/planned_order/entity"
	plannedrepo "github.com/FelipePn10/panossoerp/internal/domain/planned_order/repository"
	poentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	porepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

// A purchase suggestion is a planned order of type PURCHASE that is not yet firm
// (is_firm = false, status PLANNED). The PCP/Compras approves it (→ purchase
// order, origin MRP) or rejects it (→ cancelled).

func isPurchaseSuggestion(o *plannedentity.PlannedOrder) bool {
	return o.IsActive && o.OrderType == types.OrderPurchase && !o.IsFirm && o.Status == types.StatusPlanned
}

// ─── List suggestions ──────────────────────────────────────────────────────

type ListPurchaseSuggestionsUseCase struct {
	Planned plannedrepo.PlannedOrderRepository
	Auth    ports.AuthService
}

func (uc *ListPurchaseSuggestionsUseCase) Execute(ctx context.Context) ([]*plannedentity.PlannedOrder, error) {
	if !uc.Auth.CanListPurchaseOrders(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	all, err := uc.Planned.ListByType(ctx, string(types.OrderPurchase))
	if err != nil {
		return nil, err
	}
	out := make([]*plannedentity.PlannedOrder, 0, len(all))
	for _, o := range all {
		if isPurchaseSuggestion(o) {
			out = append(out, o)
		}
	}
	return out, nil
}

// ─── Approve suggestion ─────────────────────────────────────────────────────

type ApprovePurchaseSuggestionUseCase struct {
	Planned          plannedrepo.PlannedOrderRepository
	Repo             porepo.PurchaseOrderRepository
	Auth             ports.AuthService
	SupplierDefaults ports.SupplierPurchasingDefaultsProvider
}

func (uc *ApprovePurchaseSuggestionUseCase) Execute(ctx context.Context, dto request.ApprovePurchaseSuggestionDTO) (*poentity.PurchaseOrder, error) {
	if !uc.Auth.CanCreatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	planned, err := uc.Planned.GetByCode(ctx, dto.PlannedOrderCode)
	if err != nil {
		return nil, err
	}
	if !isPurchaseSuggestion(planned) {
		return nil, fmt.Errorf("ordem %d não é uma sugestão de compra aprovável (tipo/status/firme inválidos)", dto.PlannedOrderCode)
	}

	orderNum, err := uc.Repo.NextOrderNumber(ctx, dto.EnterpriseCode)
	if err != nil {
		return nil, err
	}

	// Default the payment term from the supplier registration when available.
	var paymentTerm *int64
	if uc.SupplierDefaults != nil && dto.SupplierCode != nil {
		if def, derr := uc.SupplierDefaults.GetPurchasingDefaults(ctx, *dto.SupplierCode, dto.EnterpriseCode); derr == nil && def != nil {
			paymentTerm = def.PaymentConditionID
		}
	}

	qty := planned.QuantityCorrected
	if qty <= 0 {
		qty = planned.Quantity
	}

	po := &poentity.PurchaseOrder{
		OrderNumber:     orderNum,
		EnterpriseCode:  dto.EnterpriseCode,
		Status:          poentity.PurchaseOrderStatusAPPROVED,
		Origin:          poentity.PurchaseOrderOriginMRP,
		EmissionDate:    time.Now(),
		DeliveryDate:    &planned.NeedDate,
		SupplierCode:    dto.SupplierCode,
		PaymentTermCode: paymentTerm,
		CurrencyCode:    "BRL",
		Notes:           dto.Notes,
		IsFirm:          true,
		CreatedBy:       dto.CreatedBy,
	}

	mask := ""
	if planned.Mask != nil {
		mask = *planned.Mask
	}
	item := &poentity.PurchaseOrderItem{
		Sequence:     1,
		ItemCode:     planned.ItemCode,
		Mask:         mask,
		RequestedQty: qty,
		UnitPrice:    dto.UnitPrice,
		TotalPrice:   qty * dto.UnitPrice,
		Status:       poentity.PurchaseOrderItemStatusOPEN,
		DeliveryDate: &planned.NeedDate,
		IsActive:     true,
	}

	// Atomic: order + item are created in one transaction.
	created, err := uc.Repo.CreateWithItems(ctx, po, []*poentity.PurchaseOrderItem{item})
	if err != nil {
		return nil, fmt.Errorf("creating purchase order from suggestion: %w", err)
	}

	// Firm the planned order (sets is_firm = TRUE, status = RELEASED).
	if _, ferr := uc.Planned.FirmOrder(ctx, planned.Code); ferr != nil {
		return nil, fmt.Errorf("firming planned order: %w", ferr)
	}

	return created, nil
}

// ─── Reject suggestion ──────────────────────────────────────────────────────

type RejectPurchaseSuggestionUseCase struct {
	Planned plannedrepo.PlannedOrderRepository
	Auth    ports.AuthService
}

func (uc *RejectPurchaseSuggestionUseCase) Execute(ctx context.Context, code int64) (*plannedentity.PlannedOrder, error) {
	if !uc.Auth.CanReleaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	planned, err := uc.Planned.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if !isPurchaseSuggestion(planned) {
		return nil, fmt.Errorf("ordem %d não é uma sugestão de compra rejeitável", code)
	}
	updated, err := uc.Planned.UpdateStatus(ctx, code, string(types.StatusCancelled))
	if err != nil {
		return nil, err
	}
	// Mark inactive so it no longer shows up as a suggestion.
	if derr := uc.Planned.Delete(ctx, code); derr != nil {
		return nil, derr
	}
	updated.IsActive = false
	return updated, nil
}
