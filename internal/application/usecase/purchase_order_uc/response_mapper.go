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
		TotalGross:             o.TotalGross.InexactFloat64(),
		TotalNet:               o.TotalNet.InexactFloat64(),
		TotalDiscount:          o.TotalDiscount.InexactFloat64(),
		PriceTableCode:         o.PriceTableCode,
		InvoiceTypeCode:        o.InvoiceTypeCode,
		FinancialAccount:       o.FinancialAccount,
		RequestTypeCode:        o.RequestTypeCode,
		CurrencyDate:           o.CurrencyDate,
		FreightType:            o.FreightType,
		FreightValueType:       o.FreightValueType,
		FreightValueMode:       o.FreightValueMode,
		FreightValue:           o.FreightValue.InexactFloat64(),
		CarrierCode:            o.CarrierCode,
		RedispatchCarrierCode:  o.RedispatchCarrierCode,
		RedispatchFreightType:  o.RedispatchFreightType,
		RedispatchFreightValue: o.RedispatchFreightValue.InexactFloat64(),
		AdvanceDate:            o.AdvanceDate,
		AdvanceValue:           o.AdvanceValue.InexactFloat64(),
		IncotermCode:           o.IncotermCode,
		ShipmentDate:           o.ShipmentDate,
		TalaoNumber:            o.TalaoNumber,
		AlcadaStatus:           o.AlcadaStatus,
		IsActive:               o.IsActive,
		IsFirm:                 o.IsFirm,
		CreatedAt:              o.CreatedAt,
		UpdatedAt:              o.UpdatedAt,
		CreatedBy:              o.CreatedBy,
		ResponsibleUserName:    o.ResponsibleUserName,
		Items:                  toPurchaseOrderItemValues(o.Items),
	}
}

// ToPurchaseOrderResponses é o mapeamento da API para quem gera pedidos em lote
// (requisição, cotação): a resposta sai no formato do GET do pedido, com os
// valores em número — a entidade, em decimal, sairia em texto.
func ToPurchaseOrderResponses(orders []*entity.PurchaseOrder) []*response.PurchaseOrderResponse {
	return toPurchaseOrderResponses(orders)
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
		Code:                      it.Code,
		PurchaseOrderCode:         it.PurchaseOrderCode,
		Sequence:                  it.Sequence,
		ItemCode:                  it.ItemCode,
		Mask:                      it.Mask,
		RequestedQty:              it.RequestedQty.InexactFloat64(),
		ReceivedQty:               it.ReceivedQty.InexactFloat64(),
		CancelledQty:              it.CancelledQty.InexactFloat64(),
		InvoicedQty:               it.InvoicedQty.InexactFloat64(),
		UnitPrice:                 it.UnitPrice.InexactFloat64(),
		TotalPrice:                it.TotalPrice.InexactFloat64(),
		DiscountPct:               it.DiscountPct.InexactFloat64(),
		IPIPct:                    it.IPIPct.InexactFloat64(),
		ICMSPct:                   it.ICMSPct.InexactFloat64(),
		ICMSSTPct:                 it.ICMSSTPct.InexactFloat64(),
		Status:                    string(it.Status),
		DeliveryDate:              it.DeliveryDate,
		PromisedDate:              it.PromisedDate,
		Notes:                     it.Notes,
		PurchaseUOM:               it.PurchaseUOM,
		InternalUOM:               it.InternalUOM,
		InternalQty:               it.InternalQty.InexactFloat64(),
		InternalPrice:             it.InternalPrice.InexactFloat64(),
		WarehouseID:               it.WarehouseID,
		TolerancePct:              it.TolerancePct.InexactFloat64(),
		CancelledToleranceQty:     it.CancelledToleranceQty.InexactFloat64(),
		OperationTypeCode:         it.OperationTypeCode,
		InvoiceTypeCode:           it.InvoiceTypeCode,
		AccountingAccount:         it.AccountingAccount,
		CostCenterCode:            it.CostCenterCode,
		FiscalClassificationCode:  it.FiscalClassificationCode,
		RequesterEmployeeCode:     it.RequesterEmployeeCode,
		ContractCode:              it.ContractCode,
		QuotationCode:             it.QuotationCode,
		PlannedOrderCode:          it.PlannedOrderCode,
		DemandType:                it.DemandType,
		DemandCode:                it.DemandCode,
		SalesOrderCode:            it.SalesOrderCode,
		ProductionOrderID:         it.ProductionOrderID,
		PurchaseRequisitionCode:   it.PurchaseRequisitionCode,
		PurchaseRequisitionItemID: it.PurchaseRequisitionItemID,
		UtilizationType:           it.UtilizationType,
		IsActive:                  it.IsActive,
		CreatedAt:                 it.CreatedAt,
		UpdatedAt:                 it.UpdatedAt,
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
