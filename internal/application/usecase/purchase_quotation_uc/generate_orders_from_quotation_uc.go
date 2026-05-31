package purchase_quotation_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	poentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	porepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
	qentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/entity"
	qrepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/repository"
	reqrepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/repository"
)

// GenerateOrdersFromQuotationUseCase turns the selected quotation prices into
// purchase orders (one per supplier), registering attendance back on the source
// requisition items and closing the quotation.
type GenerateOrdersFromQuotationUseCase struct {
	Quotations       qrepo.PurchaseQuotationRepository
	Reqs             reqrepo.PurchaseRequisitionRepository
	POs              porepo.PurchaseOrderRepository
	Auth             ports.AuthService
	SupplierDefaults ports.SupplierPurchasingDefaultsProvider
}

type GenerateOrdersFromQuotationResult struct {
	Orders  []*poentity.PurchaseOrder `json:"orders"`
	Skipped []string                  `json:"skipped,omitempty"`
}

func (uc *GenerateOrdersFromQuotationUseCase) Execute(ctx context.Context, dto request.GenerateOrdersFromQuotationDTO) (*GenerateOrdersFromQuotationResult, error) {
	if !uc.Auth.CanCreatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	q, err := uc.Quotations.GetByCode(ctx, dto.QuotationCode)
	if err != nil {
		return nil, err
	}

	items, err := uc.Quotations.ListItems(ctx, dto.QuotationCode)
	if err != nil {
		return nil, err
	}
	itemByID := make(map[int64]*qentity.PurchaseQuotationItem, len(items))
	for _, it := range items {
		itemByID[it.ID] = it
	}

	selected, err := uc.Quotations.ListSelectedPrices(ctx, dto.QuotationCode)
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("nenhum preço selecionado na cotação %d", dto.QuotationCode)
	}

	result := &GenerateOrdersFromQuotationResult{}

	// Group selected prices by supplier (preserving order).
	grouped := map[int64][]*qentity.PurchaseQuotationPrice{}
	var supplierOrder []int64
	for _, p := range selected {
		if _, ok := grouped[p.SupplierCode]; !ok {
			supplierOrder = append(supplierOrder, p.SupplierCode)
		}
		grouped[p.SupplierCode] = append(grouped[p.SupplierCode], p)
	}

	for _, supplierCode := range supplierOrder {
		prices := grouped[supplierCode]
		sc := supplierCode

		var priceTable, paymentTerm, invoiceType *int64
		var financialAccount *string
		freightType := "SEM_FRETE"
		if uc.SupplierDefaults != nil {
			if def, derr := uc.SupplierDefaults.GetPurchasingDefaults(ctx, supplierCode, q.EnterpriseCode); derr == nil && def != nil {
				priceTable = def.PurchasePriceTableID
				paymentTerm = def.PaymentConditionID
				invoiceType = def.DefaultInvoiceTypeID
				financialAccount = def.FinancialAccount
				if def.FreightType != "" {
					freightType = def.FreightType
				}
			}
		}

		orderNum, oerr := uc.POs.NextOrderNumber(ctx, q.EnterpriseCode)
		if oerr != nil {
			return nil, oerr
		}

		po := &poentity.PurchaseOrder{
			OrderNumber:      orderNum,
			EnterpriseCode:   q.EnterpriseCode,
			Status:           poentity.PurchaseOrderStatusAPPROVED,
			Origin:           poentity.PurchaseOrderOriginNORMAL,
			EmissionDate:     time.Now(),
			SupplierCode:     &sc,
			PaymentTermCode:  paymentTerm,
			PriceTableCode:   priceTable,
			InvoiceTypeCode:  invoiceType,
			FinancialAccount: financialAccount,
			FreightType:      freightType,
			CurrencyCode:     "BRL",
			IsFirm:           true,
			AlcadaStatus:     "A",
			CreatedBy:        dto.CreatedBy,
		}

		poItems := make([]*poentity.PurchaseOrderItem, 0, len(prices))
		seq := 0
		for _, p := range prices {
			qi := itemByID[p.QuotationItemID]
			if qi == nil {
				continue
			}
			seq++
			poItems = append(poItems, &poentity.PurchaseOrderItem{
				Sequence:     seq,
				ItemCode:     qi.ItemCode,
				RequestedQty: qi.Quantity,
				UnitPrice:    p.UnitPrice,
				TotalPrice:   qi.Quantity * p.UnitPrice,
				Status:       poentity.PurchaseOrderItemStatusOPEN,
				PurchaseUOM:  qi.UOM,
				DeliveryDate: qi.DeliveryDate,
				IsActive:     true,
			})
		}

		created, cerr := uc.POs.CreateWithItems(ctx, po, poItems)
		if cerr != nil {
			return nil, fmt.Errorf("creating purchase order for supplier %d: %w", supplierCode, cerr)
		}
		result.Orders = append(result.Orders, created)

		// Register attendance for requisition-sourced items.
		for _, p := range prices {
			qi := itemByID[p.QuotationItemID]
			if qi != nil && qi.SourceType == qentity.SourceRequisition && qi.SourceItemID != nil {
				_, _ = uc.Reqs.RegisterAttendance(ctx, *qi.SourceItemID, qi.Quantity)
			}
		}
	}

	_ = uc.Quotations.UpdateStatus(ctx, dto.QuotationCode, string(qentity.QuotationClosed))
	return result, nil
}
