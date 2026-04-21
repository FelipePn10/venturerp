package mapper

import (
	"github.com/FelipePn10/panossoerp/internal/application/dto/request"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	machineentity "github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
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
		toPlanners(d.Planners),
		toSupplies(d.Supplies),
		d.CreatedBy,
	)
}

func toPDM(d request.PDMDTO) itementity.PDM {
	return itementity.PDM{
		GroupID:              d.GroupID,
		ModifierID:           d.ModifierID,
		Attributes:           d.Attributes,
		DescriptionTechnique: d.DescriptionTechnique,
	}
}

func toWarehouse(d request.WarehouseDTO) itementity.Warehouse {
	return itementity.Warehouse{
		WarehouseID:                     d.WarehouseID,
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
		TankID:       d.TankID,
		Ghost:        d.Ghost,
	}
}

func toPlanners(d request.PlannersDTO) itementity.Planners {
	var machines *[]machineentity.MachineUsage

	if d.Machines != nil {
		list := make([]machineentity.MachineUsage, len(*d.Machines))
		for i, m := range *d.Machines {
			list[i] = machineentity.MachineUsage{
				MachineID: m.MachineID,
				UsageTime: m.UsageTime,
			}
		}
		machines = &list
	}

	return itementity.Planners{
		EmployeeID: d.EmployeeID,
		MachinesID: machines,
	}
}

func toSupplies(d request.SuppliesDTO) itementity.Supplies {
	return itementity.Supplies{
		TypeOfUse: d.TypeOfUse,
	}
}
