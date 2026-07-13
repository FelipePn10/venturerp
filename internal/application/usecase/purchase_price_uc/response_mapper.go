package purchase_price_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/purchase_price/entity"
)

func toPriceTableResponse(t *entity.PurchasePriceTable) *response.PurchasePriceTableResponse {
	if t == nil {
		return nil
	}
	return &response.PurchasePriceTableResponse{
		ID:            t.ID,
		EnterpriseID:  t.EnterpriseID,
		Code:          t.Code,
		SupplierCode:  t.SupplierCode,
		Description:   t.Description,
		CurrencyCode:  t.CurrencyCode,
		ValidityStart: t.ValidityStart,
		ValidityEnd:   t.ValidityEnd,
		IsActive:      t.IsActive,
		CreatedAt:     t.CreatedAt,
		CreatedBy:     t.CreatedBy,
		UpdatedAt:     t.UpdatedAt,
		Items:         toPriceTableItemValues(t.Items),
	}
}

func toPriceTableResponses(tables []*entity.PurchasePriceTable) []*response.PurchasePriceTableResponse {
	out := make([]*response.PurchasePriceTableResponse, 0, len(tables))
	for _, t := range tables {
		out = append(out, toPriceTableResponse(t))
	}
	return out
}

func toPriceTableItemResponse(it *entity.PurchasePriceTableItem) *response.PurchasePriceTableItemResponse {
	if it == nil {
		return nil
	}
	return &response.PurchasePriceTableItemResponse{
		ID:                     it.ID,
		TableID:                it.TableID,
		ItemCode:               it.ItemCode,
		SupplierCode:           it.SupplierCode,
		UOM:                    it.UOM,
		Price:                  it.Price,
		MinQty:                 it.MinQty,
		UpdateReplacementValue: it.UpdateReplacementValue,
		IsActive:               it.IsActive,
		CreatedAt:              it.CreatedAt,
		UpdatedAt:              it.UpdatedAt,
		Adjustments:            toAdjustmentResponses(it.Adjustments),
	}
}

func toAdjustmentResponses(items []*entity.PriceAdjustment) []response.PriceAdjustmentResponse {
	out := make([]response.PriceAdjustmentResponse, 0, len(items))
	for _, it := range items {
		out = append(out, response.PriceAdjustmentResponse{ID: it.ID, Sequence: it.Sequence, Kind: it.Kind, CalculationType: it.CalculationType, Value: it.Value})
	}
	return out
}

func toPriceTableItemValues(items []*entity.PurchasePriceTableItem) []response.PurchasePriceTableItemResponse {
	if len(items) == 0 {
		return nil
	}
	out := make([]response.PurchasePriceTableItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, *toPriceTableItemResponse(it))
	}
	return out
}

func toPriceTableItemResponses(items []*entity.PurchasePriceTableItem) []*response.PurchasePriceTableItemResponse {
	out := make([]*response.PurchasePriceTableItemResponse, 0, len(items))
	for _, it := range items {
		out = append(out, toPriceTableItemResponse(it))
	}
	return out
}
