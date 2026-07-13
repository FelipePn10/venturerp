package item_conversion_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity"
)

func toItemConversionResponse(c *entity.ItemUnitConversion) *response.ItemUnitConversionResponse {
	if c == nil {
		return nil
	}
	return &response.ItemUnitConversionResponse{
		ID:              c.ID,
		ItemCode:        c.ItemCode,
		Mask:            c.Mask,
		FromUOM:         c.FromUOM,
		ToUOM:           c.ToUOM,
		Factor:          c.Factor,
		RoundingPercent: c.RoundingPercent, ToleranceValue: c.ToleranceValue, ToleranceType: c.ToleranceType,
		IsActive:  c.IsActive,
		CreatedAt: c.CreatedAt,
		CreatedBy: c.CreatedBy,
	}
}

func toItemConversionResponses(list []*entity.ItemUnitConversion) []*response.ItemUnitConversionResponse {
	out := make([]*response.ItemUnitConversionResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toItemConversionResponse(c))
	}
	return out
}
