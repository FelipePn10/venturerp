package purchase_order_uc

import (
	"context"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
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
	Almoxarifados AlmoxarifadoPadrao
}

func (uc *AddPurchaseOrderItemUseCase) Execute(ctx context.Context, dto request.CreatePurchaseOrderItemDTO) (*response.PurchaseOrderItemResponse, error) {
	if !uc.Auth.CanCreatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.ItemCode <= 0 {
		return nil, errorsuc.NewValidationError("escolha o item")
	}
	if dto.RequestedQty <= 0 {
		return nil, errorsuc.NewValidationError("a quantidade deve ser maior que zero")
	}
	if dto.UnitPrice < 0 || dto.DiscountPct < 0 || dto.DiscountPct > 100 {
		return nil, errorsuc.NewValidationError("preço ou desconto inválido")
	}

	po, err := uc.Repo.GetByCode(ctx, dto.PurchaseOrderCode)
	if err != nil {
		return nil, err
	}
	if !po.IsActive {
		return nil, errorsuc.NewValidationError(fmt.Sprintf("o pedido %d está cancelado", po.Code))
	}
	if err := po.Editavel(); err != nil {
		return nil, errorsuc.NewValidationError(err.Error())
	}
	dto.WarehouseID, err = completarAlmoxarifado(ctx, uc.Almoxarifados, dto.ItemCode, dto.WarehouseID, "da linha")
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

	total := entity.TotalDaLinha(dto.RequestedQty, unitPrice, dto.DiscountPct)

	// Próxima sequência: a maior + 1 (contar as linhas repetia o número de
	// uma linha depois que outra era removida).
	existing, err := uc.Repo.ListItems(ctx, po.Code)
	if err != nil {
		return nil, err
	}
	seq := 1
	for _, l := range existing {
		if l.Sequence >= seq {
			seq = l.Sequence + 1
		}
	}

	item := &entity.PurchaseOrderItem{
		PurchaseOrderCode:         po.Code,
		Sequence:                  seq,
		ItemCode:                  dto.ItemCode,
		Mask:                      dto.Mask,
		RequestedQty:              dto.RequestedQty,
		UnitPrice:                 unitPrice,
		TotalPrice:                total,
		DiscountPct:               dto.DiscountPct,
		IPIPct:                    ipiPct,
		ICMSPct:                   dto.ICMSPct,
		ICMSSTPct:                 dto.ICMSSTPct,
		TolerancePct:              dto.TolerancePct,
		Status:                    entity.PurchaseOrderItemStatusOPEN,
		PurchaseUOM:               purchaseUOM,
		InternalUOM:               dto.InternalUOM,
		InternalQty:               internalQty,
		InternalPrice:             internalPrice,
		WarehouseID:               dto.WarehouseID,
		OperationTypeCode:         dto.OperationTypeCode,
		InvoiceTypeCode:           dto.InvoiceTypeCode,
		AccountingAccount:         dto.AccountingAccount,
		CostCenterCode:            dto.CostCenterCode,
		FiscalClassificationCode:  dto.FiscalClassificationCode,
		RequesterEmployeeCode:     dto.RequesterEmployeeCode,
		ContractCode:              dto.ContractCode,
		QuotationCode:             dto.QuotationCode,
		PlannedOrderCode:          dto.PlannedOrderCode,
		DemandType:                dto.DemandType,
		DemandCode:                dto.DemandCode,
		SalesOrderCode:            dto.SalesOrderCode,
		ProductionOrderID:         dto.ProductionOrderID,
		PurchaseRequisitionCode:   dto.PurchaseRequisitionCode,
		PurchaseRequisitionItemID: dto.PurchaseRequisitionItemID,
		UtilizationType:           dto.UtilizationType,
		Notes:                     dto.Notes,
		IsActive:                  true,
	}
	if item.DeliveryDate, err = dataOpcional(dto.DeliveryDate, "entrega"); err != nil {
		return nil, err
	}
	if item.PromisedDate, err = dataOpcional(dto.PromisedDate, "prometida"); err != nil {
		return nil, err
	}

	created, err := uc.Repo.CreateItem(ctx, item)
	if err != nil {
		return nil, err
	}
	if _, err := regravarComTotais(ctx, uc.Repo, po); err != nil {
		return nil, err
	}
	return toPurchaseOrderItemResponse(created), nil
}
