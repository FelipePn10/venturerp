package purchase_order_uc

import (
	"context"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

// AddPurchaseOrderItemUseCase adds an item to a purchase order, resolving the
// unit price (price table), the internal UM/qty/price (conversões por item) and
// the IPI% (classificação fiscal) when not explicitly provided. All three
// providers are optional (nil-safe).
type AddPurchaseOrderItemUseCase struct {
	Repo          repository.PurchaseOrderRepository
	Auth          ports.AuthService
	PriceProvider ports.PurchasePriceProvider
	UOMConverter  ports.UOMConverter
	FiscalClass   ports.FiscalClassificationProvider
}

func (uc *AddPurchaseOrderItemUseCase) Execute(ctx context.Context, dto request.CreatePurchaseOrderItemDTO) (*entity.PurchaseOrderItem, error) {
	if !uc.Auth.CanCreatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}

	po, err := uc.Repo.GetByCode(ctx, dto.PurchaseOrderCode)
	if err != nil {
		return nil, err
	}

	// 1) Unit price — from the price table when not informed.
	unitPrice := dto.UnitPrice
	purchaseUOM := dto.PurchaseUOM
	if unitPrice == 0 && po.PriceTableCode != nil && uc.PriceProvider != nil {
		if price, uom, found, perr := uc.PriceProvider.GetItemPrice(ctx, *po.PriceTableCode, dto.ItemCode, po.SupplierCode); perr == nil && found {
			unitPrice = price
			if purchaseUOM == nil && uom != "" {
				u := uom
				purchaseUOM = &u
			}
		}
	}

	// 2) IPI% — from the fiscal classification when not informed.
	ipiPct := 0.0
	if dto.IPIPct != nil {
		ipiPct = *dto.IPIPct
	} else if dto.FiscalClassificationCode != nil && uc.FiscalClass != nil {
		if rate, found, ferr := uc.FiscalClass.GetIPIRate(ctx, *dto.FiscalClassificationCode); ferr == nil && found {
			ipiPct = rate
		}
	}

	// 3) Internal UM/qty/price — via conversões por item.
	internalQty := dto.RequestedQty
	internalPrice := unitPrice
	if purchaseUOM != nil && dto.InternalUOM != nil && uc.UOMConverter != nil {
		if q, found, cerr := uc.UOMConverter.ConvertQuantity(ctx, dto.ItemCode, dto.RequestedQty, *purchaseUOM, *dto.InternalUOM); cerr == nil && found {
			internalQty = q
		}
		if p, found, cerr := uc.UOMConverter.ConvertUnitPrice(ctx, dto.ItemCode, unitPrice, *purchaseUOM, *dto.InternalUOM); cerr == nil && found {
			internalPrice = p
		}
	}

	// Total = (qty × price) − desconto.
	gross := dto.RequestedQty * unitPrice
	total := gross - gross*dto.DiscountPct/100

	// Next sequence.
	seq := 1
	if existing, lerr := uc.Repo.ListItems(ctx, po.Code); lerr == nil {
		seq = len(existing) + 1
	}

	item := &entity.PurchaseOrderItem{
		PurchaseOrderCode:        po.Code,
		Sequence:                 seq,
		ItemCode:                 dto.ItemCode,
		Mask:                     dto.Mask,
		RequestedQty:             dto.RequestedQty,
		UnitPrice:                unitPrice,
		TotalPrice:               total,
		DiscountPct:              dto.DiscountPct,
		IPIPct:                   ipiPct,
		ICMSPct:                  dto.ICMSPct,
		ICMSSTPct:                dto.ICMSSTPct,
		TolerancePct:             dto.TolerancePct,
		Status:                   entity.PurchaseOrderItemStatusOPEN,
		PurchaseUOM:              purchaseUOM,
		InternalUOM:              dto.InternalUOM,
		InternalQty:              internalQty,
		InternalPrice:            internalPrice,
		OperationTypeCode:        dto.OperationTypeCode,
		InvoiceTypeCode:          dto.InvoiceTypeCode,
		AccountingAccount:        dto.AccountingAccount,
		CostCenterCode:           dto.CostCenterCode,
		FiscalClassificationCode: dto.FiscalClassificationCode,
		RequesterEmployeeCode:    dto.RequesterEmployeeCode,
		ContractCode:             dto.ContractCode,
		QuotationCode:            dto.QuotationCode,
		UtilizationType:          dto.UtilizationType,
		Notes:                    dto.Notes,
		IsActive:                 true,
	}
	if dto.DeliveryDate != nil {
		if t, perr := time.Parse("2006-01-02", *dto.DeliveryDate); perr == nil {
			item.DeliveryDate = &t
		}
	}
	if dto.PromisedDate != nil {
		if t, perr := time.Parse("2006-01-02", *dto.PromisedDate); perr == nil {
			item.PromisedDate = &t
		}
	}

	return uc.Repo.CreateItem(ctx, item)
}
