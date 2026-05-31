package purchase_requisition_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	poentity "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	porepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
	reqrepo "github.com/FelipePn10/panossoerp/internal/domain/purchase_requisition/repository"
)

// GeneratePurchaseOrdersUseCase turns selected requisition items into purchase
// orders, grouped by supplier (preferred supplier when not given), resolving
// price from the supplier's price table and registering the attended quantity
// back on the requisition.
type GeneratePurchaseOrdersUseCase struct {
	Reqs             reqrepo.PurchaseRequisitionRepository
	POs              porepo.PurchaseOrderRepository
	Auth             ports.AuthService
	Preferred        ports.PreferredSupplierProvider
	SupplierDefaults ports.SupplierPurchasingDefaultsProvider
	PriceProvider    ports.PurchasePriceProvider
}

type GeneratePurchaseOrdersResult struct {
	Orders  []*poentity.PurchaseOrder `json:"orders"`
	Skipped []string                  `json:"skipped,omitempty"`
}

type genLine struct {
	reqItemID int64
	itemCode  int64
	qty       float64
	balance   float64
	uom       *string
	costCtr   *int64
	acct      *string
	utiliz    *string
	delivery  *time.Time
	suggested float64
}

func (uc *GeneratePurchaseOrdersUseCase) Execute(ctx context.Context, dto request.GeneratePurchaseOrdersDTO) (*GeneratePurchaseOrdersResult, error) {
	if !uc.Auth.CanCreatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	result := &GeneratePurchaseOrdersResult{}

	// Group selections by resolved supplier, preserving supplier order.
	grouped := map[int64][]genLine{}
	var supplierOrder []int64

	for _, sel := range dto.Selections {
		if sel.QtyToAttend <= 0 {
			continue
		}
		reqItem, err := uc.Reqs.GetItem(ctx, sel.RequisitionItemID)
		if err != nil {
			result.Skipped = append(result.Skipped, fmt.Sprintf("item %d: não encontrado", sel.RequisitionItemID))
			continue
		}

		var supplierCode int64
		if sel.SupplierCode != nil {
			supplierCode = *sel.SupplierCode
		} else if uc.Preferred != nil {
			if code, found, _ := uc.Preferred.GetPreferredSupplier(ctx, reqItem.ItemCode); found {
				supplierCode = code
			}
		}
		if supplierCode == 0 {
			result.Skipped = append(result.Skipped, fmt.Sprintf("item %d (%d): sem fornecedor (informe ou cadastre preferencial)", reqItem.ItemCode, sel.RequisitionItemID))
			continue
		}

		if _, ok := grouped[supplierCode]; !ok {
			supplierOrder = append(supplierOrder, supplierCode)
		}
		grouped[supplierCode] = append(grouped[supplierCode], genLine{
			reqItemID: reqItem.ID,
			itemCode:  reqItem.ItemCode,
			qty:       sel.QtyToAttend,
			balance:   reqItem.Balance(),
			uom:       reqItem.UOM,
			costCtr:   reqItem.CostCenterCode,
			acct:      reqItem.AccountingAccount,
			utiliz:    reqItem.UtilizationType,
			delivery:  reqItem.DeliveryDate,
			suggested: reqItem.SuggestedPrice,
		})
	}

	for _, supplierCode := range supplierOrder {
		lines := grouped[supplierCode]
		sc := supplierCode

		var priceTable, paymentTerm, invoiceType *int64
		var financialAccount *string
		freightType := ""
		if uc.SupplierDefaults != nil {
			if def, derr := uc.SupplierDefaults.GetPurchasingDefaults(ctx, supplierCode, dto.EnterpriseCode); derr == nil && def != nil {
				priceTable = def.PurchasePriceTableID
				paymentTerm = def.PaymentConditionID
				invoiceType = def.DefaultInvoiceTypeID
				financialAccount = def.FinancialAccount
				freightType = def.FreightType
			}
		}
		if freightType == "" {
			freightType = "SEM_FRETE"
		}

		orderNum, err := uc.POs.NextOrderNumber(ctx, dto.EnterpriseCode)
		if err != nil {
			return nil, err
		}

		po := &poentity.PurchaseOrder{
			OrderNumber:      orderNum,
			EnterpriseCode:   dto.EnterpriseCode,
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

		items := make([]*poentity.PurchaseOrderItem, 0, len(lines))
		for i, ln := range lines {
			unitPrice := ln.suggested
			if unitPrice == 0 && priceTable != nil && uc.PriceProvider != nil {
				if p, _, found, _ := uc.PriceProvider.GetItemPrice(ctx, *priceTable, ln.itemCode, &sc); found {
					unitPrice = p
				}
			}
			items = append(items, &poentity.PurchaseOrderItem{
				Sequence:          i + 1,
				ItemCode:          ln.itemCode,
				RequestedQty:      ln.qty,
				UnitPrice:         unitPrice,
				TotalPrice:        ln.qty * unitPrice,
				Status:            poentity.PurchaseOrderItemStatusOPEN,
				PurchaseUOM:       ln.uom,
				DeliveryDate:      ln.delivery,
				AccountingAccount: ln.acct,
				CostCenterCode:    ln.costCtr,
				UtilizationType:   ln.utiliz,
				IsActive:          true,
			})
		}

		created, err := uc.POs.CreateWithItems(ctx, po, items)
		if err != nil {
			return nil, fmt.Errorf("creating purchase order for supplier %d: %w", supplierCode, err)
		}
		result.Orders = append(result.Orders, created)

		// Register attendance back on the requisition items (capped at balance).
		for _, ln := range lines {
			attend := ln.qty
			if attend > ln.balance {
				attend = ln.balance
			}
			if attend > 0 {
				_, _ = uc.Reqs.RegisterAttendance(ctx, ln.reqItemID, attend)
			}
		}
	}

	return result, nil
}
