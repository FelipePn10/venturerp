package sales_quotation_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/entity"
	quoterepo "github.com/FelipePn10/panossoerp/internal/domain/sales_quotation/repository"
)

func toResponse(q *entity.SalesQuotation) *response.SalesQuotationResponse {
	if q == nil {
		return nil
	}
	out := &response.SalesQuotationResponse{
		Code:                    q.Code,
		QuotationNumber:         q.QuotationNumber,
		EnterpriseCode:          q.EnterpriseCode,
		Status:                  string(q.Status),
		QuotationType:           string(q.QuotationType),
		EmissionDate:            q.EmissionDate,
		DigitDate:               q.DigitDate,
		ValidUntil:              q.ValidUntil,
		DeliveryDate:            q.DeliveryDate,
		DeliveryDateFirm:        q.DeliveryDateFirm,
		PurchaseOrderNumber:     q.PurchaseOrderNumber,
		CustomerCode:            q.CustomerCode,
		BillingAddressCode:      q.BillingAddressCode,
		ShippingAddressCode:     q.ShippingAddressCode,
		RepresentativeCode:      q.RepresentativeCode,
		SalesDivisionCode:       q.SalesDivisionCode,
		PriceTableCode:          q.PriceTableCode,
		PaymentTermCode:         q.PaymentTermCode,
		CurrencyCode:            q.CurrencyCode,
		ProbabilityPct:          q.ProbabilityPct,
		CommissionPct:           q.CommissionPct,
		IsNFCe:                  q.IsNFCe,
		Street:                  q.Street,
		StreetNumber:            q.StreetNumber,
		ForeignDocument:         q.ForeignDocument,
		ReleaseStatus:           string(q.ReleaseStatus),
		CommercialBlocked:       q.CommercialBlocked,
		CommercialBlockReason:   q.CommercialBlockReason,
		CarrierCode:             q.CarrierCode,
		FreightType:             q.FreightType,
		VerifyFreight:           q.VerifyFreight,
		FreightValue:            q.FreightValue,
		RedeliveryFreightValue:  q.RedeliveryFreightValue,
		InsuranceValue:          q.InsuranceValue,
		DiscountValue:           q.DiscountValue,
		SurchargeValue:          q.SurchargeValue,
		RetainedTaxValue:        q.RetainedTaxValue,
		TotalGross:              q.TotalGross,
		TotalNet:                q.TotalNet,
		DeliveryAuthorization:   q.DeliveryAuthorization,
		Notes:                   q.Notes,
		ObsCustomer:             q.ObsCustomer,
		CancelReason:            q.CancelReason,
		CancelComplement:        q.CancelComplement,
		AttendedReason:          q.AttendedReason,
		AttendedAt:              q.AttendedAt,
		ConvertedSalesOrderCode: q.ConvertedSalesOrderCode,
		ConvertedAt:             q.ConvertedAt,
		IsActive:                q.IsActive,
		CreatedAt:               q.CreatedAt,
		UpdatedAt:               q.UpdatedAt,
		CreatedBy:               q.CreatedBy,
	}
	if len(q.Items) > 0 {
		out.Items = make([]response.SalesQuotationItemResponse, 0, len(q.Items))
		for _, item := range q.Items {
			out.Items = append(out.Items, *toItemResponse(item))
		}
	}
	return out
}

func toResponses(items []*entity.SalesQuotation) []*response.SalesQuotationResponse {
	out := make([]*response.SalesQuotationResponse, 0, len(items))
	for _, item := range items {
		out = append(out, toResponse(item))
	}
	return out
}

func toItemResponse(item *entity.SalesQuotationItem) *response.SalesQuotationItemResponse {
	if item == nil {
		return nil
	}
	return &response.SalesQuotationItemResponse{
		Code:               item.Code,
		SalesQuotationCode: item.SalesQuotationCode,
		Sequence:           item.Sequence,
		ItemCode:           item.ItemCode,
		Mask:               item.Mask,
		SalesUOM:           item.SalesUOM,
		WarehouseCode:      item.WarehouseCode,
		PriceTableCode:     item.PriceTableCode,
		RequestedQty:       item.RequestedQty,
		UnitPrice:          item.UnitPrice,
		AttendedQty:        item.AttendedQty,
		CancelledQty:       item.CancelledQty,
		Balance:            item.Balance,
		DeliveryDate:       item.DeliveryDate,
		DeliveryDateFirm:   item.DeliveryDateFirm,
		DiscountPct:        item.DiscountPct,
		IPIPct:             item.IPIPct,
		STPct:              item.STPct,
		TotalGross:         item.TotalGross,
		TotalNet:           item.TotalNet,
		TotalNetWithIPI:    item.TotalNetWithIPI,
		Status:             string(item.Status),
		Notes:              item.Notes,
		IsActive:           item.IsActive,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}
}

func toItemResponses(items []*entity.SalesQuotationItem) []*response.SalesQuotationItemResponse {
	out := make([]*response.SalesQuotationItemResponse, 0, len(items))
	for _, item := range items {
		out = append(out, toItemResponse(item))
	}
	return out
}

func toReportResponse(r *quoterepo.SalesQuotationReport) *response.SalesQuotationReportResponse {
	return &response.SalesQuotationReportResponse{
		TotalQuotations: r.TotalQuotations,
		TotalGross:      r.TotalGross,
		TotalNet:        r.TotalNet,
		OpenCount:       r.OpenCount,
		ApprovedCount:   r.ApprovedCount,
		ConvertedCount:  r.ConvertedCount,
		CancelledCount:  r.CancelledCount,
		ExpiredCount:    r.ExpiredCount,
		WeightedNet:     r.WeightedNet,
		RetainedTax:     r.RetainedTax,
	}
}
