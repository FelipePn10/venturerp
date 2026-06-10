package item_uc

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/response"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
)

func toItemResponse(it *entity.Item) *response.ItemResponse {
	if it == nil {
		return nil
	}
	return &response.ItemResponse{
		ID:          it.ID,
		Code:        int64(it.Code),
		Complement:  it.Complement,
		Nature:      int(it.Nature),
		PDM:         toItemPDMResponse(it.PDM),
		Situation:   it.Situation.String(),
		Health:      it.Health.String(),
		Warehouse:   toItemWarehouseResponse(it.Warehouse),
		Engineering: toItemEngineeringResponse(it.Engineering),
		Planning:    toItemPlanningResponse(it.Planning),
		Supplies:    response.ItemSuppliesResponse{TypeOfUse: it.Supplies.TypeOfUse.String()},
		CreatedBy:   it.CreatedBy,
		CreatedAt:   it.CreatedAt,
	}
}

func toItemResponses(list []*entity.Item) []*response.ItemResponse {
	out := make([]*response.ItemResponse, 0, len(list))
	for _, it := range list {
		out = append(out, toItemResponse(it))
	}
	return out
}

func toItemPDMResponse(p entity.PDM) response.ItemPDMResponse {
	attrs := make([]response.ItemAttributeResponse, 0, len(p.Attributes))
	for _, a := range p.Attributes {
		attrs = append(attrs, response.ItemAttributeResponse{Name: a.Name, Value: a.Value})
	}
	return response.ItemPDMResponse{
		GroupCode:            p.GroupCode,
		ModifierCode:         p.ModifierCode,
		Attributes:           attrs,
		DescriptionTechnique: p.DescriptionTechnique,
	}
}

func toItemWarehouseResponse(w entity.Warehouse) response.ItemWarehouseResponse {
	var cyclical *int
	if w.CyclicalCountConfig != nil {
		v := w.CyclicalCountConfig.DaysInterval
		cyclical = &v
	}
	return response.ItemWarehouseResponse{
		WarehouseCode:                   w.WarehouseCode,
		UnitOfMeasurement:               w.UnitOfMeasurement.String(),
		AutomaticLow:                    w.AutomaticLow,
		CyclicalCountDaysInterval:       cyclical,
		MinimumStock:                    w.MinimumStock,
		AverageMonthlyConsumptionManual: w.AverageMonthlyConsumptionManual,
	}
}

func toItemEngineeringResponse(e entity.Engineering) response.ItemEngineeringResponse {
	var dims *response.ItemDimensionsResponse
	if e.Dimensions != nil {
		dims = &response.ItemDimensionsResponse{
			Length: e.Dimensions.Length,
			Width:  e.Dimensions.Width,
			Height: e.Dimensions.Height,
		}
	}
	return response.ItemEngineeringResponse{
		ItemBaseCod: e.ItemBaseCod,
		Weight: response.ItemWeightResponse{
			Gross: e.Weight.Gross,
			Net:   e.Weight.Net,
			Unit:  e.Weight.Unit,
		},
		Dimensions: dims,
		Type:       e.Type.String(),
		TypeStruct: e.TypeStruct.String(),
		OEM:        e.OEM,
	}
}

func toItemPlanningResponse(p entity.Planning) response.ItemPlanningResponse {
	var rop *response.ItemReorderPointResponse
	if p.ReorderPoint != nil {
		rop = &response.ItemReorderPointResponse{
			TR: p.ReorderPoint.TR,
			CM: p.ReorderPoint.CM,
			CR: p.ReorderPoint.CR,
			ES: p.ReorderPoint.ES,
		}
	}
	return response.ItemPlanningResponse{
		TypeMRP:      p.TypeMRP.String(),
		LLC:          p.LLC,
		ReorderPoint: rop,
		TankCode:     p.TankCode,
		Ghost:        p.Ghost,
	}
}

func toItemWithMasksResponses(list []entity.ItemWithMasks) []response.ItemWithMasksResponse {
	out := make([]response.ItemWithMasksResponse, 0, len(list))
	for _, iwm := range list {
		masks := make([]response.MaskSummaryResponse, 0, len(iwm.Masks))
		for _, m := range iwm.Masks {
			masks = append(masks, response.MaskSummaryResponse{
				ID:        m.ID,
				Mask:      m.Mask,
				MaskHash:  m.MaskHash,
				CreatedAt: m.CreatedAt,
			})
		}
		out = append(out, response.ItemWithMasksResponse{
			Item:  toItemResponse(iwm.Item),
			Masks: masks,
		})
	}
	return out
}
