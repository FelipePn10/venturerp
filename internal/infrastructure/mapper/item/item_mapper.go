package mapper

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"strings"
)

func ToItemEntity(d request.CreateItemDTO) (*itementity.Item, error) {
	return itementity.NewItem(
		d.Code,
		d.Name,
		d.Complement,
		d.Nature,
		toPDM(d.PDM),
		d.Situation,
		d.Health,
		toWarehouse(d.Warehouse),
		toEngineering(d.Engineering),
		toPlanning(d.Planning),
		toSupplies(d.Supplies),
		toCommercial(d.Commercial),
		toAccounting(d.Accounting, d.AccountingFiscal),
		d.CreatedBy,
	)
}

func clean(value *string) *string {
	if value == nil {
		return nil
	}
	v := strings.TrimSpace(*value)
	if v == "" {
		return nil
	}
	return &v
}

func toCommercial(d *request.CommercialDTO) itementity.Commercial {
	if d == nil {
		return itementity.Commercial{}
	}
	return itementity.Commercial{
		Description: clean(d.Description), SaleType: clean(d.SaleType), VolumeConversionFactor: d.VolumeConversionFactor,
		SaleMultiple: d.SaleMultiple, MinimumSaleQuantity: d.MinimumSaleQuantity, EstimatedDeliveryDays: d.EstimatedDeliveryDays,
		WarrantyDays: d.WarrantyDays, TransferWarehouseCode: d.TransferWarehouseCode,
		TechnicalAssistanceWarehouseCode: d.TechnicalAssistanceWarehouseCode, PackagingItemCode: d.PackagingItemCode,
		AllowBillingDescriptionChange: d.AllowBillingDescriptionChange, IssueLoadingLabels: d.IssueLoadingLabels,
		AssembleShippingVolumes: d.AssembleShippingVolumes, RequiresSpecialPackaging: d.RequiresSpecialPackaging,
		WithholdPISCOFINS: d.WithholdPISCOFINS, IsPackaging: d.IsPackaging, MobileEnabled: d.MobileEnabled,
		ExportPackaging: d.ExportPackaging, ClassificationCode: clean(d.ClassificationCode), Notes: clean(d.Notes),
	}
}

func toAccounting(d *request.AccountingDTO, legacy *request.LegacyAccountingDTO) itementity.Accounting {
	if d == nil {
		if legacy != nil {
			return itementity.Accounting{CalculatePISCOFINS: &legacy.CalculatePISCOFINS}
		}
		return itementity.Accounting{}
	}
	return itementity.Accounting{
		SaleFiscalClassificationCode: clean(d.SaleFiscalClassificationCode), PurchaseFiscalClassificationCode: clean(d.PurchaseFiscalClassificationCode),
		Origin: d.Origin, SaleIPIType: clean(d.SaleIPIType), SaleIPIRate: d.SaleIPIRate,
		PurchaseIPIType: clean(d.PurchaseIPIType), PurchaseIPIRate: d.PurchaseIPIRate, ICMSRate: d.ICMSRate,
		SaleUnitOfMeasurement: d.SaleUnitOfMeasurement, PurchaseUnitOfMeasurement: d.PurchaseUnitOfMeasurement,
		InventoryGroupCode: d.InventoryGroupCode, AccountingClassificationCode: clean(d.AccountingClassificationCode),
		CEST: clean(d.CEST), InputCode: clean(d.InputCode), CalculatePISCOFINS: d.CalculatePISCOFINS, Notes: clean(d.Notes),
	}
}

func toPDM(d request.PDMDTO) itementity.PDM {
	return itementity.PDM{
		GroupCode:            d.GroupCode,
		ModifierCode:         d.ModifierCode,
		Attributes:           d.Attributes,
		DescriptionTechnique: d.DescriptionTechnique,
	}
}

func toWarehouse(d request.WarehouseDTO) itementity.Warehouse {
	return itementity.Warehouse{
		WarehouseCode:                   d.WarehouseCode,
		UnitOfMeasurement:               d.UnitOfMeasurement,
		AutomaticLow:                    d.AutomaticLow,
		CyclicalCountConfig:             d.CyclicalCountConfig,
		MinimumStock:                    d.MinimumStock,
		AverageMonthlyConsumptionManual: d.AverageMonthlyConsumptionManual,
	}
}

func toEngineering(d request.EngineeringDTO) itementity.Engineering {
	return itementity.Engineering{
		ItemBaseCod: d.ItemBaseCod,
		Weight:      d.Weight,
		Dimensions:  d.Dimensions,
		Type:        d.Type,
		TypeStruct:  d.TypeStruct,
		OEM:         d.OEM,
	}
}

func toPlanning(d request.PlanningDTO) itementity.Planning {
	return itementity.Planning{
		TypeMRP:      d.TypeMRP,
		LLC:          d.LLC,
		ReorderPoint: d.ReorderPoint,
		TankCode:     d.TankCode,
		Ghost:        d.Ghost,
		ABCClass:     d.ABCClass,
		MinimumLot:   d.MinimumLot,
		MultipleLot:  d.MultipleLot,
		SafetyStock:  d.SafetyStock,
		Critical:     d.Critical,
		Exclusive:    d.Exclusive,
		Active:       d.Active,
	}
}

func toSupplies(d request.SuppliesDTO) itementity.Supplies {
	return itementity.Supplies{
		TypeOfUse:          d.TypeOfUse,
		PurchaseUOM:        d.PurchaseUOM,
		WarehouseCode:      d.WarehouseCode,
		ReceivingChecklist: d.ReceivingChecklist,
		Harvest:            d.Harvest,
	}
}
