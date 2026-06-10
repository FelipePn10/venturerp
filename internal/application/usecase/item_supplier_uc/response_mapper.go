package item_supplier_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/item_supplier/entity"
)

func toItemPreferredSupplierResponse(s *entity.ItemPreferredSupplier) *response.ItemPreferredSupplierResponse {
	if s == nil {
		return nil
	}
	return &response.ItemPreferredSupplierResponse{
		ID:                  s.ID,
		ItemCode:            s.ItemCode,
		SupplierCode:        s.SupplierCode,
		Ranking:             s.Ranking,
		SupplierItemCode:    s.SupplierItemCode,
		SupplierDescription: s.SupplierDescription,
		UOM:                 s.UOM,
		LeadTimeDays:        s.LeadTimeDays,
		IsActive:            s.IsActive,
		CreatedAt:           s.CreatedAt,
		CreatedBy:           s.CreatedBy,
	}
}

func toItemPreferredSupplierResponses(list []*entity.ItemPreferredSupplier) []*response.ItemPreferredSupplierResponse {
	out := make([]*response.ItemPreferredSupplierResponse, 0, len(list))
	for _, s := range list {
		out = append(out, toItemPreferredSupplierResponse(s))
	}
	return out
}
