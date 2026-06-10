package purchase_order_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_order/entity"
)

func toPurchaseOrderResponse(o *entity.PurchaseOrder) *response.PurchaseOrderResponse {
	if o == nil {
		return nil
	}
	return &response.PurchaseOrderResponse{
		Code:                   o.Code,
		OrderNumber:            o.OrderNumber,
		EnterpriseCode:         o.EnterpriseCode,
		Status:                 string(o.Status),
		Origin:                 string(o.Origin),
		EmissionDate:           o.EmissionDate,
		DeliveryDate:           o.DeliveryDate,
		SupplierCode:           o.SupplierCode,
		PaymentTermCode:        o.PaymentTermCode,
		CurrencyCode:           o.CurrencyCode,
		ShippingAddressCode:    o.ShippingAddressCode,
		Notes:                  o.Notes,
		TotalGross:             o.TotalGross,
		TotalNet:               o.TotalNet,
		TotalDiscount:          o.TotalDiscount,
		PriceTableCode:         o.PriceTableCode,
		InvoiceTypeCode:        o.InvoiceTypeCode,
		FinancialAccount:       o.FinancialAccount,
		RequestTypeCode:        o.RequestTypeCode,
		CurrencyDate:           o.CurrencyDate,
		FreightType:            o.FreightType,
		FreightValueType:       o.FreightValueType,
		FreightValueMode:       o.FreightValueMode,
		FreightValue:           o.FreightValue,
		CarrierCode:            o.CarrierCode,
		RedispatchCarrierCode:  o.RedispatchCarrierCode,
		RedispatchFreightType:  o.RedispatchFreightType,
		RedispatchFreightValue: o.RedispatchFreightValue,
		AdvanceDate:            o.AdvanceDate,
		AdvanceValue:           o.AdvanceValue,
		IncotermCode:           o.IncotermCode,
		ShipmentDate:           o.ShipmentDate,
		TalaoNumber:            o.TalaoNumber,
		AlcadaStatus:           o.AlcadaStatus,
		IsActive:               o.IsActive,
		IsFirm:                 o.IsFirm,
		CreatedAt:              o.CreatedAt,
		UpdatedAt:              o.UpdatedAt,
		CreatedBy:              o.CreatedBy,
		Items:                  toPurchaseOrderItemValues(o.Items),
	}
}

func toPurchaseOrderResponses(orders []*entity.PurchaseOrder) []*response.PurchaseOrderResponse {
	out := make([]*response.PurchaseOrderResponse, 0, len(orders))
	for _, o := range orders {
		out = append(out, toPurchaseOrderResponse(o))
	}
	return out
}

func toPurchaseOrderItemResponse(it *entity.PurchaseOrderItem) *response.PurchaseOrderItemResponse {
	if it == nil {
		return nil
	}
	return &response.PurchaseOrderItemResponse{
		Code:                     it.Code,
		PurchaseOrderCode:        it.PurchaseOrderCode,
		Sequence:                 it.Sequence,
		ItemCode:                 it.ItemCode,
		Mask:                     it.Mask,
		RequestedQty:             it.RequestedQty,
		ReceivedQty:              it.ReceivedQty,
		CancelledQty:             it.CancelledQty,
		UnitPrice:                it.UnitPrice,
		TotalPrice:               it.TotalPrice,
		DiscountPct:              it.DiscountPct,
		IPIPct:                   it.IPIPct,
		ICMSPct:                  it.ICMSPct,
		ICMSSTPct:                it.ICMSSTPct,
		Status:                   string(it.Status),
		DeliveryDate:             it.DeliveryDate,
		PromisedDate:             it.PromisedDate,
		Notes:                    it.Notes,
		PurchaseUOM:              it.PurchaseUOM,
		InternalUOM:              it.InternalUOM,
		InternalQty:              it.InternalQty,
		InternalPrice:            it.InternalPrice,
		TolerancePct:             it.TolerancePct,
		CancelledToleranceQty:    it.CancelledToleranceQty,
		OperationTypeCode:        it.OperationTypeCode,
		InvoiceTypeCode:          it.InvoiceTypeCode,
		AccountingAccount:        it.AccountingAccount,
		CostCenterCode:           it.CostCenterCode,
		FiscalClassificationCode: it.FiscalClassificationCode,
		RequesterEmployeeCode:    it.RequesterEmployeeCode,
		ContractCode:             it.ContractCode,
		QuotationCode:            it.QuotationCode,
		UtilizationType:          it.UtilizationType,
		IsActive:                 it.IsActive,
		CreatedAt:                it.CreatedAt,
		UpdatedAt:                it.UpdatedAt,
	}
}

// toPurchaseOrderItemValues maps lines to value DTOs for embedding in the header.
func toPurchaseOrderItemValues(items []*entity.PurchaseOrderItem) []response.PurchaseOrderItemResponse {
	if len(items) == 0 {
		return nil
	}
	out := make([]response.PurchaseOrderItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, *toPurchaseOrderItemResponse(it))
	}
	return out
}
