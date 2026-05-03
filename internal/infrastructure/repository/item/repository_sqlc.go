package item

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	machine "github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/pgutil"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/sqlc"
)

func (r *RepositoryItemSQLC) Create(
	ctx context.Context,
	item *entity.Item,
) (*entity.Item, error) {

	attributes, err := json.Marshal(item.PDM.Attributes)
	if err != nil {
		return nil, fmt.Errorf("marshal pdm_attributes: %w", err)
	}

	weight, err := json.Marshal(item.Engineering.Weight)
	if err != nil {
		return nil, fmt.Errorf("marshal engineering_weight: %w", err)
	}

	dimensions, err := json.Marshal(item.Engineering.Dimensions)
	if err != nil {
		return nil, fmt.Errorf("marshal engineering_dimensions: %w", err)
	}

	reorderPoint, err := json.Marshal(item.Planning.ReorderPoint)
	if err != nil {
		return nil, fmt.Errorf("marshal planning_reorder_point: %w", err)
	}

	cyclicalCountConfig, err := json.Marshal(item.Warehouse.CyclicalCountConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal cyclical_count_config: %w", err)
	}

	params := sqlc.CreateItemParams{
		WarehouseID: int32(item.Warehouse.WarehouseID),
		Code:        int64(item.Code),

		Complement: pgutil.ToPgTextFromPtr(item.Complement),

		Nature:    int16(item.Nature),
		Inherit:   item.Inherit,
		Situation: int16(item.Situation),
		Health:    sqlc.HealthEnum(item.Health),

		PdmGroupID:              item.PDM.GroupID,
		PdmModifierID:           item.PDM.ModifierID,
		PdmAttributes:           attributes,
		PdmDescriptionTechnique: item.PDM.DescriptionTechnique,

		WarehouseUnitOfMeasurement:           sqlc.UnitOfMeasurementEnum(item.Warehouse.UnitOfMeasurement),
		WarehouseAutomaticLow:                item.Warehouse.AutomaticLow,
		WarehouseCyclicalCountConfig:         cyclicalCountConfig,
		WarehouseMinimumStock:                item.Warehouse.MinimumStock,
		WarehouseAvgMonthlyConsumptionManual: intPtrToInt32Ptr(item.Warehouse.AverageMonthlyConsumptionManual),

		EngineeringItemBaseCod: intPtrToInt32Ptr(item.Engineering.ItemBaseCod),
		EngineeringWeight:      weight,
		EngineeringDimensions:  dimensions,
		EngineeringType:        int16(item.Engineering.Type),
		EngineeringTypeStruct:  int16(item.Engineering.TypeStruct),
		EngineeringOem:         item.Engineering.OEM,

		PlanningTypeMrp:      int16(item.Planning.TypeMRP),
		PlanningLlc:          int32(item.Planning.LLC),
		PlanningReorderPoint: reorderPoint,
		PlanningTankID:       intPtrToInt32Ptr(item.Planning.TankID),
		PlanningGhost:        item.Planning.Ghost,

		PlannerEmployeeID: item.Planners.EmployeeID,

		SuppliesTypeOfUse: int16(item.Supplies.TypeOfUse),

		CreatedBy: pgutil.ToPgUUID(item.CreatedBy),
	}

	dbItem, err := r.q.CreateItem(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create item: %w", err)
	}

	if item.Planners.MachinesID != nil {
		for _, m := range *item.Planners.MachinesID {
			_, err := r.q.CreateItemMachineUsage(ctx, sqlc.CreateItemMachineUsageParams{
				ItemID:    dbItem.ID,
				MachineID: int32(m.MachineID),
				UsageTime: int32(m.UsageTime),
			})
			if err != nil {
				return nil, fmt.Errorf("create machine usage (machine_id=%d): %w", m.MachineID, err)
			}
		}
	}

	return mapDBItemToEntity(dbItem, item.Planners.MachinesID)
}

func (r *RepositoryItemSQLC) FindItemByCode(
	ctx context.Context,
	code valueobject.ItemCode,
) (*entity.Item, error) {

	dbItem, err := r.q.FindItemByCode(ctx, int64(code))
	if err != nil {
		return nil, fmt.Errorf("finding item by code: %w", err)
	}

	// Ainda não tem carregamento de machines aqui
	// irei manter como nil
	var machines *[]machine.MachineUsage

	return mapDBItemToEntity(dbItem, machines)
}

func mapDBItemToEntity(
	dbItem sqlc.Item,
	machines *[]machine.MachineUsage,
) (*entity.Item, error) {

	var complement *string
	if dbItem.Complement.Valid {
		v := dbItem.Complement.String
		complement = &v
	}

	var pdmAttributes []valueobject.Attribute
	if err := json.Unmarshal(dbItem.PdmAttributes, &pdmAttributes); err != nil {
		return nil, fmt.Errorf("unmarshal pdm_attributes: %w", err)
	}

	var engineeringWeight valueobject.Weight
	if err := json.Unmarshal(dbItem.EngineeringWeight, &engineeringWeight); err != nil {
		return nil, fmt.Errorf("unmarshal engineering_weight: %w", err)
	}

	var engineeringDimensions *valueobject.Dimensions
	if len(dbItem.EngineeringDimensions) > 0 {
		var v valueobject.Dimensions
		if err := json.Unmarshal(dbItem.EngineeringDimensions, &v); err != nil {
			return nil, fmt.Errorf("unmarshal engineering_dimensions: %w", err)
		}
		engineeringDimensions = &v
	}

	var planningReorderPoint *valueobject.ReorderPoint
	if len(dbItem.PlanningReorderPoint) > 0 {
		var v valueobject.ReorderPoint
		if err := json.Unmarshal(dbItem.PlanningReorderPoint, &v); err != nil {
			return nil, fmt.Errorf("unmarshal planning_reorder_point: %w", err)
		}
		planningReorderPoint = &v
	}

	var cyclicalCount *valueobject.CyclicalCountConfig
	if len(dbItem.WarehouseCyclicalCountConfig) > 0 {
		var v valueobject.CyclicalCountConfig
		if err := json.Unmarshal(dbItem.WarehouseCyclicalCountConfig, &v); err != nil {
			return nil, fmt.Errorf("unmarshal cyclical_count_config: %w", err)
		}
		cyclicalCount = &v
	}

	return &entity.Item{
		ID:         dbItem.ID,
		Code:       valueobject.ItemCode(dbItem.Code),
		Complement: complement,

		Nature:  entity.ItemNature(dbItem.Nature),
		Inherit: dbItem.Inherit,

		PDM: entity.PDM{
			GroupID:              dbItem.PdmGroupID,
			ModifierID:           dbItem.PdmModifierID,
			Attributes:           pdmAttributes,
			DescriptionTechnique: dbItem.PdmDescriptionTechnique,
		},

		Situation: types.TypeSituationItem(dbItem.Situation),
		Health:    types.Health(dbItem.Health),

		Warehouse: entity.Warehouse{
			WarehouseID:                     int(dbItem.WarehouseID),
			UnitOfMeasurement:               types.TypeUnitOfMeasurementItem(dbItem.WarehouseUnitOfMeasurement),
			AutomaticLow:                    dbItem.WarehouseAutomaticLow,
			CyclicalCountConfig:             cyclicalCount,
			MinimumStock:                    dbItem.WarehouseMinimumStock,
			AverageMonthlyConsumptionManual: int32PtrToIntPtr(dbItem.WarehouseAvgMonthlyConsumptionManual),
		},

		Engineering: entity.Engineering{
			ItemBaseCod: int32PtrToIntPtr(dbItem.EngineeringItemBaseCod),
			Weight:      engineeringWeight,
			Dimensions:  engineeringDimensions,
			Type:        types.TypeItem(dbItem.EngineeringType),
			TypeStruct:  types.TypeStructItem(dbItem.EngineeringTypeStruct),
			OEM:         dbItem.EngineeringOem,
		},

		Planning: entity.Planning{
			TypeMRP:      types.TypeMRPItem(dbItem.PlanningTypeMrp),
			LLC:          int(dbItem.PlanningLlc),
			ReorderPoint: planningReorderPoint,
			TankID:       int32PtrToIntPtr(dbItem.PlanningTankID),
			Ghost:        dbItem.PlanningGhost,
		},

		Planners: entity.Planners{
			EmployeeID: dbItem.PlannerEmployeeID,
			MachinesID: machines,
		},

		Supplies: entity.Supplies{
			TypeOfUse: types.TypeOfUseItem(dbItem.SuppliesTypeOfUse),
		},

		CreatedBy: pgutil.FromPgUUID(dbItem.CreatedBy),
		CreatedAt: pgutil.FromPgTimestamp(dbItem.CreatedAt),
	}, nil
}

func intPtrToInt32Ptr(v *int) *int32 {
	if v == nil {
		return nil
	}
	value := int32(*v)
	return &value
}

func int32PtrToIntPtr(v *int32) *int {
	if v == nil {
		return nil
	}
	value := int(*v)
	return &value
}
