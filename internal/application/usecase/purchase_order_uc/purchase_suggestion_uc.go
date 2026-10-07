package purchase_order_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
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
	Almoxarifados    AlmoxarifadoPadrao
	// Aprovacao passa o pedido gerado pela alçada de valores. Sem ela o pedido
	// fica em rascunho para ser aprovado em VPDC0200.
	Aprovacao *ApprovePurchaseOrderUseCase
}

func (uc *ApprovePurchaseSuggestionUseCase) Execute(ctx context.Context, dto request.ApprovePurchaseSuggestionDTO) (*poentity.PurchaseOrder, error) {
	if !uc.Auth.CanCreatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	enterpriseCode, err := uc.Auth.EnterpriseCode(ctx)
	if err != nil {
		return nil, err
	}
	dto.EnterpriseCode = enterpriseCode
	dto.CreatedBy, err = uc.Auth.UserID(ctx)
	if err != nil {
		return nil, err
	}

	if dto.SupplierCode == nil || *dto.SupplierCode <= 0 {
		return nil, errorsuc.NewValidationError("escolha o fornecedor da compra")
	}
	if dto.UnitPrice < 0 {
		return nil, errorsuc.NewValidationError("preço unitário inválido")
	}
	planned, err := uc.Planned.GetByCode(ctx, dto.PlannedOrderCode)
	if err != nil {
		return nil, err
	}
	if !isPurchaseSuggestion(planned) {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("ordem %d não é uma sugestão de compra aprovável (tipo/status/firme inválidos)", dto.PlannedOrderCode))
	}

	orderNum, err := uc.Repo.NextOrderNumber(ctx, dto.EnterpriseCode)
	if err != nil {
		return nil, err
	}

	// Condição de pagamento do cadastro do fornecedor — que, de passagem, é
	// onde se confere que ele é desta empresa e está ativo.
	var paymentTerm *int64
	if uc.SupplierDefaults != nil {
		def, derr := resolverFornecedorDoTenant(ctx, uc.SupplierDefaults, *dto.SupplierCode, dto.EnterpriseCode)
		if derr != nil {
			return nil, derr
		}
		paymentTerm = def.PaymentConditionID
	}

	qty := planned.QuantityCorrected
	if qty <= 0 {
		qty = planned.Quantity
	}

	po := &poentity.PurchaseOrder{
		OrderNumber:    orderNum,
		EnterpriseCode: dto.EnterpriseCode,
		// Nasce em rascunho e vai pela aprovação normal (alçada) logo abaixo:
		// criar já aprovado deixava a sugestão do MRP passar por cima do limite.
		Status:          poentity.PurchaseOrderStatusDRAFT,
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
		Sequence:         1,
		ItemCode:         planned.ItemCode,
		Mask:             mask,
		RequestedQty:     qty,
		UnitPrice:        dto.UnitPrice,
		TotalPrice:       poentity.TotalDaLinha(qty, dto.UnitPrice, 0),
		Status:           poentity.PurchaseOrderItemStatusOPEN,
		DeliveryDate:     &planned.NeedDate,
		IsActive:         true,
		WarehouseID:      planned.WarehouseCode,
		PlannedOrderCode: &planned.Code,
		DemandCode:       planned.DemandCode,
		SalesOrderCode:   planned.SalesOrderCode,
	}
	if item.WarehouseID, err = completarAlmoxarifado(ctx, uc.Almoxarifados, planned.ItemCode, planned.WarehouseCode, "da sugestão"); err != nil {
		return nil, err
	}
	demandType := string(planned.DemandType)
	item.DemandType = &demandType

	// Atomic: order + item are created in one transaction.
	created, err := uc.Repo.CreateWithItems(ctx, po, []*poentity.PurchaseOrderItem{item})
	if err != nil {
		return nil, fmt.Errorf("creating purchase order from suggestion: %w", err)
	}

	// Firm the planned order (sets is_firm = TRUE, status = RELEASED).
	if _, ferr := uc.Planned.FirmOrder(ctx, planned.Code); ferr != nil {
		return nil, fmt.Errorf("firming planned order: %w", ferr)
	}

	if uc.Aprovacao != nil {
		if _, aerr := uc.Aprovacao.Execute(ctx, created.Code); aerr != nil {
			return nil, aerr
		}
		if atual, gerr := uc.Repo.GetByCode(ctx, created.Code); gerr == nil {
			atual.Items = created.Items
			created = atual
		}
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
		return nil, errorsuc.NewValidationError(fmt.Sprintf("ordem %d não é uma sugestão de compra rejeitável", code))
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

// SugestaoResponse é a sugestão como a tela lê. O código do item sai com a
// chave item_code para o tradutor devolvê-lo como código comercial — a
// entidade crua saía como "ItemCode" e a tela mostrava a chave interna.
type SugestaoResponse struct {
	Code          int64     `json:"code"`
	ItemCode      int64     `json:"item_code"`
	Mask          *string   `json:"mask,omitempty"`
	Quantity      float64   `json:"quantity"`
	NeedDate      time.Time `json:"need_date"`
	Status        string    `json:"status"`
	OrderType     string    `json:"order_type"`
	WarehouseCode *int64    `json:"warehouse_code,omitempty"`
}

// ParaSugestoes converte as ordens planejadas de compra para a resposta.
func ParaSugestoes(lista []*plannedentity.PlannedOrder) []SugestaoResponse {
	out := make([]SugestaoResponse, 0, len(lista))
	for _, o := range lista {
		q := o.QuantityCorrected
		if q <= 0 {
			q = o.Quantity
		}
		out = append(out, SugestaoResponse{Code: o.Code, ItemCode: o.ItemCode, Mask: o.Mask, Quantity: q, NeedDate: o.NeedDate,
			Status: string(o.Status), OrderType: string(o.OrderType), WarehouseCode: o.WarehouseCode})
	}
	return out
}

// ParaPedidoResponse expõe o mapeamento do pedido para o handler da sugestão.
func ParaPedidoResponse(o *poentity.PurchaseOrder) *response.PurchaseOrderResponse {
	return toPurchaseOrderResponse(o)
}
