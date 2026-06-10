package purchase_quotation_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_quotation/entity"
)

func toQuotationResponse(q *entity.PurchaseQuotation) *response.PurchaseQuotationResponse {
	if q == nil {
		return nil
	}
	return &response.PurchaseQuotationResponse{
		ID:             q.ID,
		Code:           q.Code,
		EnterpriseCode: q.EnterpriseCode,
		Status:         string(q.Status),
		EmissionDate:   q.EmissionDate,
		Notes:          q.Notes,
		IsActive:       q.IsActive,
		CreatedAt:      q.CreatedAt,
		CreatedBy:      q.CreatedBy,
		UpdatedAt:      q.UpdatedAt,
		Items:          toQuotationItemValues(q.Items),
		Suppliers:      toQuotationSupplierValues(q.Suppliers),
	}
}

func toQuotationResponses(qs []*entity.PurchaseQuotation) []*response.PurchaseQuotationResponse {
	out := make([]*response.PurchaseQuotationResponse, 0, len(qs))
	for _, q := range qs {
		out = append(out, toQuotationResponse(q))
	}
	return out
}

func toQuotationItemValues(items []*entity.PurchaseQuotationItem) []response.PurchaseQuotationItemResponse {
	if len(items) == 0 {
		return nil
	}
	out := make([]response.PurchaseQuotationItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, response.PurchaseQuotationItemResponse{
			ID:            it.ID,
			QuotationCode: it.QuotationCode,
			Sequence:      it.Sequence,
			ItemCode:      it.ItemCode,
			Quantity:      it.Quantity,
			UOM:           it.UOM,
			DeliveryDate:  it.DeliveryDate,
			SourceType:    string(it.SourceType),
			SourceCode:    it.SourceCode,
			SourceItemID:  it.SourceItemID,
			IsConfigured:  it.IsConfigured,
			CreatedAt:     it.CreatedAt,
			Prices:        toQuotationPriceValues(it.Prices),
		})
	}
	return out
}

func toQuotationSupplierValues(suppliers []*entity.PurchaseQuotationSupplier) []response.PurchaseQuotationSupplierResponse {
	if len(suppliers) == 0 {
		return nil
	}
	out := make([]response.PurchaseQuotationSupplierResponse, 0, len(suppliers))
	for _, s := range suppliers {
		out = append(out, *toQuotationSupplierResponse(s))
	}
	return out
}

func toQuotationPriceValues(prices []*entity.PurchaseQuotationPrice) []response.PurchaseQuotationPriceResponse {
	if len(prices) == 0 {
		return nil
	}
	out := make([]response.PurchaseQuotationPriceResponse, 0, len(prices))
	for _, p := range prices {
		out = append(out, *toQuotationPriceResponse(p))
	}
	return out
}

func toQuotationSupplierResponse(s *entity.PurchaseQuotationSupplier) *response.PurchaseQuotationSupplierResponse {
	if s == nil {
		return nil
	}
	return &response.PurchaseQuotationSupplierResponse{
		ID:            s.ID,
		QuotationCode: s.QuotationCode,
		SupplierCode:  s.SupplierCode,
		InvitedAt:     s.InvitedAt,
	}
}

func toQuotationPriceResponse(p *entity.PurchaseQuotationPrice) *response.PurchaseQuotationPriceResponse {
	if p == nil {
		return nil
	}
	return &response.PurchaseQuotationPriceResponse{
		ID:              p.ID,
		QuotationItemID: p.QuotationItemID,
		SupplierCode:    p.SupplierCode,
		UnitPrice:       p.UnitPrice,
		LeadTimeDays:    p.LeadTimeDays,
		PaymentTermCode: p.PaymentTermCode,
		Notes:           p.Notes,
		IsSelected:      p.IsSelected,
		CreatedAt:       p.CreatedAt,
	}
}
