package item_conversion_uc

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/item_conversion/entity"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
)

func toItemConversionResponse(c *entity.ItemUnitConversion, item *itementity.Item, hoje time.Time) *response.ItemUnitConversionResponse {
	if c == nil {
		return nil
	}
	fromQty, toQty := c.FromQty, c.ToQty
	if !fromQty.IsPositive() || !toQty.IsPositive() {
		fromQty, toQty = decimal.NewFromInt(1), c.Factor
	}
	out := &response.ItemUnitConversionResponse{
		ID:              c.ID,
		LegacyCode:      c.ItemCode,
		Mask:            c.Mask,
		FromUOM:         c.FromUOM,
		ToUOM:           c.ToUOM,
		Factor:          c.Factor.InexactFloat64(),
		FromQuantity:    fromQty,
		ToQuantity:      toQty,
		FactorExact:     toQty.DivRound(fromQty, 10),
		RoundingPercent: c.RoundingPercent.InexactFloat64(), ToleranceValue: c.ToleranceValue.InexactFloat64(), ToleranceType: c.ToleranceType,
		ValidFrom: c.ValidFrom.Format("2006-01-02"),
		Situation: c.Situacao(hoje),
		Origin:    c.Origin,
		Notes:     c.Notes,
		IsActive:  c.IsActive,
		CreatedAt: c.CreatedAt,
		CreatedBy: c.CreatedBy,
		UpdatedAt: c.UpdatedAt,
		UpdatedBy: c.UpdatedBy,
	}
	if toQty.IsPositive() {
		out.InverseFactor = fromQty.DivRound(toQty, 10)
	}
	if c.ValidTo != nil {
		s := c.ValidTo.Format("2006-01-02")
		out.ValidTo = &s
	}
	if out.Origin == "" {
		out.Origin = entity.OrigemManual
	}
	if item != nil {
		out.ItemCode, out.ItemName = string(item.BusinessCode), item.Name
	}
	return out
}

func toItemConversionResponses(list []*entity.ItemUnitConversion, item *itementity.Item, hoje time.Time) []*response.ItemUnitConversionResponse {
	out := make([]*response.ItemUnitConversionResponse, 0, len(list))
	for _, c := range list {
		out = append(out, toItemConversionResponse(c, item, hoje))
	}
	return out
}

func toLotResponse(l *entity.LotConversion, itemCode string) *response.LotConversionResponse {
	return &response.LotConversionResponse{ID: l.ID, ItemCode: itemCode, Lot: l.Lot, FromUOM: l.FromUOM, ToUOM: l.ToUOM,
		FromQuantity: l.FromQty, ToQuantity: l.ToQty, Factor: l.Factor, Notes: l.Notes, CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt}
}
