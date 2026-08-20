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
		ID:              it.ID,
		Code:            string(it.BusinessCode),
		LegacyCode:      int64(it.Code),
		Name:            it.Name,
		Complement:      it.Complement,
		Nature:          int(it.Nature),
		PDM:             toItemPDMResponse(it.PDM),
		Situation:       it.Situation.String(),
		Health:          it.Health.String(),
		Warehouse:       toItemWarehouseResponse(it.Warehouse),
		Engineering:     toItemEngineeringResponse(it.Engineering),
		Planning:        toItemPlanningResponse(it.Planning),
		Supplies:        toItemSuppliesResponse(it.Supplies),
		Commercial:      toCommercialResponse(it.Commercial),
		Accounting:      toAccountingResponse(it.Accounting),
		FiscalEffective: toFiscalEffectiveResponse(it.FiscalEffective),
		CreatedBy:       it.CreatedBy,
		CreatedAt:       it.CreatedAt,
	}
}

func toFiscalEffectiveResponse(v entity.FiscalEffective) response.ItemFiscalEffectiveResponse {
	mapContext := func(c *entity.EffectiveFiscalContext) *response.EffectiveFiscalContextResponse {
		if c == nil {
			return nil
		}
		sources := map[string]string{}
		for k, v := range c.Sources {
			sources[k] = string(v)
		}
		return &response.EffectiveFiscalContextResponse{ClassificationID: c.ClassificationID, ClassificationCode: c.ClassificationCode, NCM: c.NCM, CEST: c.CEST, Unit: c.Unit, Origin: c.Origin, IPIRate: c.IPIRate, ICMSRate: c.ICMSRate, PISRate: c.PISRate, COFINSRate: c.COFINSRate, CalculatePISCOFINS: c.CalculatePISCOFINS, Sources: sources}
	}
	return response.ItemFiscalEffectiveResponse{Purchase: mapContext(v.Purchase), Sale: mapContext(v.Sale)}
}

func toCommercialResponse(v entity.Commercial) response.ItemCommercialResponse {
	return response.ItemCommercialResponse{
		Description: v.Description, SaleType: v.SaleType, VolumeConversionFactor: v.VolumeConversionFactor, SaleMultiple: v.SaleMultiple, MinimumSaleQuantity: v.MinimumSaleQuantity,
		EstimatedDeliveryDays: v.EstimatedDeliveryDays, WarrantyDays: v.WarrantyDays, TransferWarehouseCode: v.TransferWarehouseCode, TechnicalAssistanceWarehouseCode: v.TechnicalAssistanceWarehouseCode,
		PackagingItemCode: v.PackagingItemCode, AllowBillingDescriptionChange: v.AllowBillingDescriptionChange, IssueLoadingLabels: v.IssueLoadingLabels, AssembleShippingVolumes: v.AssembleShippingVolumes,
		RequiresSpecialPackaging: v.RequiresSpecialPackaging, WithholdPISCOFINS: v.WithholdPISCOFINS, IsPackaging: v.IsPackaging, MobileEnabled: v.MobileEnabled, ExportPackaging: v.ExportPackaging, ClassificationCode: v.ClassificationCode, Notes: v.Notes,
	}
}

func toAccountingResponse(v entity.Accounting) response.ItemAccountingResponse {
	var sale, purchase *string
	if v.SaleUnitOfMeasurement != nil {
		s := v.SaleUnitOfMeasurement.String()
		sale = &s
	}
	if v.PurchaseUnitOfMeasurement != nil {
		s := v.PurchaseUnitOfMeasurement.String()
		purchase = &s
	}
	return response.ItemAccountingResponse{SaleFiscalClassificationCode: v.SaleFiscalClassificationCode, PurchaseFiscalClassificationCode: v.PurchaseFiscalClassificationCode, Origin: v.Origin,
		SaleIPIType: v.SaleIPIType, SaleIPIRate: v.SaleIPIRate, PurchaseIPIType: v.PurchaseIPIType, PurchaseIPIRate: v.PurchaseIPIRate, ICMSRate: v.ICMSRate,
		SaleUnitOfMeasurement: sale, PurchaseUnitOfMeasurement: purchase, InventoryGroupCode: v.InventoryGroupCode, AccountingClassificationCode: v.AccountingClassificationCode,
		CEST: v.CEST, InputCode: v.InputCode, CalculatePISCOFINS: v.CalculatePISCOFINS, Notes: v.Notes}
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
	var cyclical *response.ItemCyclicalCountConfigResponse
	if w.CyclicalCountConfig != nil {
		cyclical = &response.ItemCyclicalCountConfigResponse{DaysInterval: w.CyclicalCountConfig.DaysInterval}
	}
	return response.ItemWarehouseResponse{
		WarehouseCode:                   w.WarehouseCode,
		UnitOfMeasurement:               w.UnitOfMeasurement.String(),
		AutomaticLow:                    w.AutomaticLow,
		CyclicalCountConfig:             cyclical,
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
		ABCClass:     p.ABCClass,
		MinimumLot:   p.MinimumLot,
		MultipleLot:  p.MultipleLot,
		SafetyStock:  p.SafetyStock,
		Critical:     p.Critical,
		Exclusive:    p.Exclusive,
		Active:       p.Active,
	}
}

func toItemSuppliesResponse(s entity.Supplies) response.ItemSuppliesResponse {
	var purchaseUOM *string
	if s.PurchaseUOM != nil {
		value := s.PurchaseUOM.String()
		purchaseUOM = &value
	}
	return response.ItemSuppliesResponse{
		TypeOfUse: s.TypeOfUse.String(), PurchaseUOM: purchaseUOM, WarehouseCode: s.WarehouseCode,
		ReceivingChecklist: s.ReceivingChecklist, Harvest: s.Harvest,
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
