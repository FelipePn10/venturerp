package sales_order_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_order/entity"
)

// toSalesOrderResponse converts a domain SalesOrder into its API response DTO,
// keeping domain types out of the HTTP boundary.
func toSalesOrderResponse(o *entity.SalesOrder) *response.SalesOrderResponse {
	if o == nil {
		return nil
	}
	return &response.SalesOrderResponse{
		Code:                        o.Code,
		OrderNumber:                 o.OrderNumber,
		EnterpriseCode:              o.EnterpriseCode,
		Status:                      string(o.Status),
		Origin:                      string(o.Origin),
		EmissionDate:                o.EmissionDate,
		DeliveryDate:                o.DeliveryDate,
		DeliveryDateFirm:            o.DeliveryDateFirm,
		DigitDate:                   o.DigitDate,
		CustomerCode:                o.CustomerCode,
		BillingAddressCode:          o.BillingAddressCode,
		ShippingAddressCode:         o.ShippingAddressCode,
		RepresentativeCode:          o.RepresentativeCode,
		RepresentativeOrderNumber:   o.RepresentativeOrderNumber,
		PlanCode:                    o.PlanCode,
		SalesDivisionCode:           o.SalesDivisionCode,
		CommissionPct:               o.CommissionPct,
		TaxTypeCode:                 o.TaxTypeCode,
		PresenceIndicator:           o.PresenceIndicator,
		SalesChannel:                o.SalesChannel,
		DefaultNFType:               o.DefaultNFType,
		PriceTableCode:              o.PriceTableCode,
		CurrencyCode:                o.CurrencyCode,
		PaymentTermCode:             o.PaymentTermCode,
		AdditionalDays:              o.AdditionalDays,
		BearerCode:                  o.BearerCode,
		SaleDate:                    o.SaleDate,
		TotalWeightNet:              o.TotalWeightNet,
		TotalWeightGross:            o.TotalWeightGross,
		TotalGross:                  o.TotalGross,
		TotalNet:                    o.TotalNet,
		TotalNetNoST:                o.TotalNetNoST,
		TotalWithIPIWithST:          o.TotalWithIPIWithST,
		Notes:                       o.Notes,
		ObsCustomer:                 o.ObsCustomer,
		IsBlocked:                   o.IsBlocked,
		BlockReason:                 o.BlockReason,
		IsFirm:                      o.IsFirm,
		IsActive:                    o.IsActive,
		IsNFCe:                      o.IsNFCe,
		Street:                      o.Street,
		StreetNumber:                o.StreetNumber,
		ForeignDocument:             o.ForeignDocument,
		CollectionEstablishmentCode: o.CollectionEstablishmentCode,
		NFTypeDescription:           o.NFTypeDescription,
		CarrierCode:                 o.CarrierCode,
		FreightType:                 o.FreightType,
		FreightValue:                o.FreightValue,
		InsuranceValue:              o.InsuranceValue,
		VolumeQuantity:              o.VolumeQuantity,
		VolumeType:                  o.VolumeType,
		NetWeight:                   o.NetWeight,
		GrossWeight:                 o.GrossWeight,
		DiscountValue:               o.DiscountValue,
		SurchargeValue:              o.SurchargeValue,
		ProjectCode:                 o.ProjectCode,
		ProjectName:                 o.ProjectName,
		Items:                       toSalesOrderItemResponses(o.Items),
		CreatedAt:                   o.CreatedAt,
		UpdatedAt:                   o.UpdatedAt,
		CreatedBy:                   o.CreatedBy,
	}
}

// toSalesOrderResponses maps a slice of domain orders to response DTOs.
func toSalesOrderResponses(orders []*entity.SalesOrder) []*response.SalesOrderResponse {
	out := make([]*response.SalesOrderResponse, 0, len(orders))
	for _, o := range orders {
		out = append(out, toSalesOrderResponse(o))
	}
	return out
}

// toSalesOrderItemResponse converts a domain order line into its response DTO.
func toSalesOrderItemResponse(it *entity.SalesOrderItem) *response.SalesOrderItemResponse {
	if it == nil {
		return nil
	}
	return &response.SalesOrderItemResponse{
		Code:             it.Code,
		SalesOrderCode:   it.SalesOrderCode,
		Sequence:         it.Sequence,
		ItemCode:         it.ItemCode,
		Mask:             it.Mask,
		DigitDate:        it.DigitDate,
		NFType:           it.NFType,
		SalesUOM:         it.SalesUOM,
		WarehouseCode:    it.WarehouseCode,
		PriceTableCode:   it.PriceTableCode,
		RequestedQty:     it.RequestedQty,
		UnitPrice:        it.UnitPrice,
		AttendedQty:      it.AttendedQty,
		CancelledQty:     it.CancelledQty,
		Balance:          it.Balance,
		DeliveryDate:     it.DeliveryDate,
		DeliveryDateFirm: it.DeliveryDateFirm,
		CustomerDelivery: it.CustomerDelivery,
		Lot:              it.Lot,
		CouponDelivery:   it.CouponDelivery,
		PaidAtCashier:    it.PaidAtCashier,
		IPIPct:           it.IPIPct,
		ICMSPct:          it.ICMSPct,
		PISPct:           it.PISPct,
		COFINSPct:        it.COFINSPct,
		STPct:            it.STPct,
		DiscountPct:      it.DiscountPct,
		TotalGross:       it.TotalGross,
		TotalNet:         it.TotalNet,
		TotalNetWithIPI:  it.TotalNetWithIPI,
		TotalIPI:         it.TotalIPI,
		TotalST:          it.TotalST,
		UnitWeightNet:    it.UnitWeightNet,
		UnitWeightGross:  it.UnitWeightGross,
		Status:           string(it.Status),
		Notes:            it.Notes,
		IsActive:         it.IsActive,
		CreatedAt:        it.CreatedAt,
		UpdatedAt:        it.UpdatedAt,
	}
}

// toSalesOrderItemResponses maps a slice of domain order lines to value DTOs
// (used for embedding inside SalesOrderResponse.Items).
func toSalesOrderItemResponses(items []*entity.SalesOrderItem) []response.SalesOrderItemResponse {
	if len(items) == 0 {
		return nil
	}
	out := make([]response.SalesOrderItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, *toSalesOrderItemResponse(it))
	}
	return out
}

// toSalesOrderItemResponsePtrs maps a slice of domain order lines to pointer
// DTOs (used for the standalone "list items" endpoint).
func toSalesOrderItemResponsePtrs(items []*entity.SalesOrderItem) []*response.SalesOrderItemResponse {
	out := make([]*response.SalesOrderItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, toSalesOrderItemResponse(it))
	}
	return out
}
