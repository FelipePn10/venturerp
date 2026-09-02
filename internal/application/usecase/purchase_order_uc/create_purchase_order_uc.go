package purchase_order_uc

import (
	"context"
	"fmt"
	"time"

	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/application/ports"
	errorsuc "github.com/FelipePn10/panossoerp/internal/application/usecase/errors"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/repository"
)

type CreatePurchaseOrderUseCase struct {
	Repo repository.PurchaseOrderRepository
	Auth ports.AuthService
	// SupplierDefaults is optional. When set, missing header fields (payment
	// term) are defaulted from the supplier's registration. Nil disables it.
	SupplierDefaults ports.SupplierPurchasingDefaultsProvider
}

func (uc *CreatePurchaseOrderUseCase) Execute(
	ctx context.Context,
	dto request.CreatePurchaseOrderDTO,
) (*response.PurchaseOrderResponse, error) {
	if !uc.Auth.CanCreatePurchaseOrder(ctx) {
		return nil, errorsuc.ErrUnauthorized
	}
	if dto.SupplierCode == nil || *dto.SupplierCode <= 0 {
		return nil, fmt.Errorf("fornecedor é obrigatório")
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

	orderNum, err := uc.Repo.NextOrderNumber(ctx, dto.EnterpriseCode)
	if err != nil {
		return nil, err
	}

	emissionDate := time.Now()
	if dto.EmissionDate != "" {
		emissionDate, err = time.Parse("2006-01-02", dto.EmissionDate)
		if err != nil {
			return nil, fmt.Errorf("data de emissão inválida; use AAAA-MM-DD")
		}
	}

	status := entity.PurchaseOrderStatusDRAFT
	if dto.Status != "" {
		status = entity.PurchaseOrderStatus(dto.Status)
	}
	origin := entity.PurchaseOrderOriginNORMAL
	if dto.Origin != "" {
		origin = entity.PurchaseOrderOrigin(dto.Origin)
	}
	currency := "BRL"
	if dto.CurrencyCode != "" {
		currency = dto.CurrencyCode
	}

	// Default header fields from the supplier registration when not provided.
	paymentTerm := dto.PaymentTermCode
	priceTable := dto.PriceTableCode
	invoiceType := dto.InvoiceTypeCode
	financialAccount := dto.FinancialAccount
	freightType := dto.FreightType
	if uc.SupplierDefaults != nil && dto.SupplierCode != nil {
		if def, derr := uc.SupplierDefaults.GetPurchasingDefaults(ctx, *dto.SupplierCode, dto.EnterpriseCode); derr == nil && def != nil {
			if paymentTerm == nil {
				paymentTerm = def.PaymentConditionID
			}
			if priceTable == nil {
				priceTable = def.PurchasePriceTableID
			}
			if invoiceType == nil {
				invoiceType = def.DefaultInvoiceTypeID
			}
			if financialAccount == nil {
				financialAccount = def.FinancialAccount
			}
			if freightType == "" {
				freightType = def.FreightType
			}
		}
	}

	o := &entity.PurchaseOrder{
		OrderNumber:         orderNum,
		EnterpriseCode:      dto.EnterpriseCode,
		Status:              status,
		Origin:              origin,
		EmissionDate:        emissionDate,
		SupplierCode:        dto.SupplierCode,
		PaymentTermCode:     paymentTerm,
		CurrencyCode:        currency,
		ShippingAddressCode: dto.ShippingAddressCode,
		Notes:               dto.Notes,
		TotalGross:          dto.TotalGross,
		TotalNet:            dto.TotalNet,
		TotalDiscount:       dto.TotalDiscount,
		IsFirm:              dto.IsFirm,
		CreatedBy:           dto.CreatedBy,
		// extended header
		PriceTableCode:         priceTable,
		InvoiceTypeCode:        invoiceType,
		FinancialAccount:       financialAccount,
		RequestTypeCode:        dto.RequestTypeCode,
		FreightType:            freightType,
		FreightValueType:       dto.FreightValueType,
		FreightValueMode:       dto.FreightValueMode,
		FreightValue:           dto.FreightValue,
		CarrierCode:            dto.CarrierCode,
		RedispatchCarrierCode:  dto.RedispatchCarrierCode,
		RedispatchFreightType:  dto.RedispatchFreightType,
		RedispatchFreightValue: dto.RedispatchFreightValue,
		AdvanceValue:           dto.AdvanceValue,
		IncotermCode:           dto.IncotermCode,
		TalaoNumber:            dto.TalaoNumber,
	}

	if dto.DeliveryDate != nil {
		t, parseErr := time.Parse("2006-01-02", *dto.DeliveryDate)
		if parseErr != nil {
			return nil, fmt.Errorf("data de entrega inválida; use AAAA-MM-DD")
		}
		o.DeliveryDate = &t
	}
	o.CurrencyDate = parsePODate(dto.CurrencyDate)
	o.AdvanceDate = parsePODate(dto.AdvanceDate)
	o.ShipmentDate = parsePODate(dto.ShipmentDate)

	items, err := canonicalPurchaseOrderItems(dto.Items)
	if err != nil {
		return nil, err
	}
	var created *entity.PurchaseOrder
	if len(items) > 0 {
		created, err = uc.Repo.CreateWithItems(ctx, o, items)
	} else {
		created, err = uc.Repo.Create(ctx, o)
	}
	if err != nil {
		return nil, err
	}
	return toPurchaseOrderResponse(created), nil
}

func canonicalPurchaseOrderItems(inputs []request.CreatePurchaseOrderItemDTO) ([]*entity.PurchaseOrderItem, error) {
	items := make([]*entity.PurchaseOrderItem, 0, len(inputs))
	for i, input := range inputs {
		if input.ItemCode <= 0 || input.RequestedQty <= 0 {
			return nil, fmt.Errorf("item e quantidade positiva são obrigatórios na linha %d", i+1)
		}
		if input.WarehouseID == nil || *input.WarehouseID <= 0 {
			return nil, fmt.Errorf("depósito é obrigatório na linha %d", i+1)
		}
		if input.UnitPrice < 0 || input.DiscountPct < 0 || input.DiscountPct > 100 {
			return nil, fmt.Errorf("preço ou desconto inválido na linha %d", i+1)
		}
		gross := input.RequestedQty * input.UnitPrice
		line := &entity.PurchaseOrderItem{
			Sequence: i + 1, ItemCode: input.ItemCode, Mask: input.Mask,
			RequestedQty: input.RequestedQty, UnitPrice: input.UnitPrice,
			TotalPrice:  gross - gross*input.DiscountPct/100,
			DiscountPct: input.DiscountPct, ICMSPct: input.ICMSPct, ICMSSTPct: input.ICMSSTPct,
			TolerancePct: input.TolerancePct, Status: entity.PurchaseOrderItemStatusOPEN,
			PurchaseUOM: input.PurchaseUOM, InternalUOM: input.InternalUOM,
			WarehouseID: input.WarehouseID,
			InternalQty: input.RequestedQty, InternalPrice: input.UnitPrice,
			OperationTypeCode: input.OperationTypeCode, InvoiceTypeCode: input.InvoiceTypeCode,
			AccountingAccount: input.AccountingAccount, CostCenterCode: input.CostCenterCode,
			FiscalClassificationCode: input.FiscalClassificationCode,
			RequesterEmployeeCode:    input.RequesterEmployeeCode, ContractCode: input.ContractCode,
			QuotationCode: input.QuotationCode, UtilizationType: input.UtilizationType,
			PlannedOrderCode: input.PlannedOrderCode, DemandType: input.DemandType,
			DemandCode: input.DemandCode, SalesOrderCode: input.SalesOrderCode,
			ProductionOrderID:         input.ProductionOrderID,
			PurchaseRequisitionCode:   input.PurchaseRequisitionCode,
			PurchaseRequisitionItemID: input.PurchaseRequisitionItemID,
			Notes:                     input.Notes, IsActive: true,
		}
		if input.IPIPct != nil {
			line.IPIPct = *input.IPIPct
		}
		if input.DeliveryDate != nil {
			d, err := time.Parse("2006-01-02", *input.DeliveryDate)
			if err != nil {
				return nil, fmt.Errorf("data de entrega inválida na linha %d", i+1)
			}
			line.DeliveryDate = &d
		}
		if input.PromisedDate != nil {
			d, err := time.Parse("2006-01-02", *input.PromisedDate)
			if err != nil {
				return nil, fmt.Errorf("data prometida inválida na linha %d", i+1)
			}
			line.PromisedDate = &d
		}
		items = append(items, line)
	}
	return items, nil
}

func parsePODate(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil
	}
	return &t
}
