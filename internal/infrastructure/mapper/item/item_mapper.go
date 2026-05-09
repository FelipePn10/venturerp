package mapper

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
)

func ToItemEntity(d request.CreateItemDTO) (*itementity.Item, error) {
	return itementity.NewItem(
		d.Code,
		d.Complement,
		d.Nature,
		d.Inherit,
		toPDM(d.PDM),
		d.Situation,
		d.Health,
		toWarehouse(d.Warehouse),
		toEngineering(d.Engineering),
		toPlanning(d.Planning),
		toSupplies(d.Supplies),
		d.CreatedBy,
	)
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
	}
}

func toSupplies(d request.SuppliesDTO) itementity.Supplies {
	return itementity.Supplies{
		TypeOfUse: d.TypeOfUse,
	}
}
