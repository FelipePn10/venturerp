package item_conversion_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
)

func toItemConversionResponse(c *entity.ItemUnitConversion, item *itementity.Item) *response.ItemUnitConversionResponse {
	if c == nil {
		return nil
	}
	out := &response.ItemUnitConversionResponse{
		ID:              c.ID,
		LegacyCode:      c.ItemCode,
		Mask:            c.Mask,
		FromUOM:         c.FromUOM,
		ToUOM:           c.ToUOM,
		Factor:          c.Factor,
		RoundingPercent: c.RoundingPercent, ToleranceValue: c.ToleranceValue, ToleranceType: c.ToleranceType,
		IsActive:  c.IsActive,
		CreatedAt: c.CreatedAt,
		CreatedBy: c.CreatedBy,
	}
	if item != nil {
		out.ItemCode, out.ItemName = string(item.BusinessCode), item.Name
	}
	return out
}

func toItemConversionResponses(list []*entity.ItemUnitConversion, item *itementity.Item) []*response.ItemUnitConversionResponse {
	out := make([]*response.ItemUnitConversionResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toItemConversionResponse(c, item))
	}
	return out
}
