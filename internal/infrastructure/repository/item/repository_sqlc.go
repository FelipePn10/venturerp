package item

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	machine "github.com/FelipePn10/panossoerp/internal/domain/machine/entity"
	"github.com/FelipePn10/panossoerp/internal/infrastructure/database/nullable"
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

	dimensions, err := nullable.ToNullRawMessage(item.Engineering.Dimensions)
	if err != nil {
		return nil, fmt.Errorf("marshal engineering_dimensions: %w", err)
	}

	reorderPoint, err := nullable.ToNullRawMessage(item.Planning.ReorderPoint)
	if err != nil {
		return nil, fmt.Errorf("marshal planning_reorder_point: %w", err)
	}

	cyclicalCountConfig, err := nullable.ToNullRawMessage(item.Warehouse.CyclicalCountConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal cyclical_count_config: %w", err)
	}

	params := sqlc.CreateItemParams{
		WarehouseID: int32(item.Warehouse.WarehouseID),

		// ✔ bigint alinhado
		Code: int64(item.Code),

		Complement: nullable.ToNullString(item.Complement),

		Nature:    int16(item.Nature),
		Inherit:   item.Inherit,
		Situation: int16(item.Situation),

		// ✔ ENUM string → string
		Health: sqlc.HealthEnum(item.Health),

		PdmGroupID:              item.PDM.GroupID,
		PdmModifierID:           item.PDM.ModifierID,
		PdmAttributes:           attributes,
		PdmDescriptionTechnique: item.PDM.DescriptionTechnique,

		WarehouseUnitOfMeasurement: sqlc.UnitOfMeasurementEnum(item.Warehouse.UnitOfMeasurement),

		WarehouseAutomaticLow:        item.Warehouse.AutomaticLow,
		WarehouseCyclicalCountConfig: cyclicalCountConfig,
		WarehouseMinimumStock:        item.Warehouse.MinimumStock,
		WarehouseAvgMonthlyConsumptionManual: nullable.ToNullInt32FromIntPtr(
			item.Warehouse.AverageMonthlyConsumptionManual,
		),

		EngineeringItemBaseCod: nullable.ToNullInt32FromIntPtr(item.Engineering.ItemBaseCod),
		EngineeringWeight:      weight,
		EngineeringDimensions:  dimensions,
		EngineeringType:        int16(item.Engineering.Type),
		EngineeringTypeStruct:  int16(item.Engineering.TypeStruct),
		EngineeringOem:         item.Engineering.OEM,

		PlanningTypeMrp:      int16(item.Planning.TypeMRP),
		PlanningLlc:          int32(item.Planning.LLC),
		PlanningReorderPoint: reorderPoint,
		PlanningTankID:       nullable.ToNullInt32FromIntPtr(item.Planning.TankID),
		PlanningGhost:        item.Planning.Ghost,

		PlannerEmployeeID: nullable.ToNullInt32FromPtr(item.Planners.EmployeeID),

		SuppliesTypeOfUse: int16(item.Supplies.TypeOfUse),

		CreatedBy: item.CreatedBy,
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

	var pdmAttributes []valueobject.Attribute
	if err := json.Unmarshal(dbItem.PdmAttributes, &pdmAttributes); err != nil {
		return nil, fmt.Errorf("unmarshal pdm_attributes: %w", err)
	}

	var engineeringWeight valueobject.Weight
	if err := json.Unmarshal(dbItem.EngineeringWeight, &engineeringWeight); err != nil {
		return nil, fmt.Errorf("unmarshal engineering_weight: %w", err)
	}

	engineeringDimensions, err := nullable.UnmarshalNullRawMessage[valueobject.Dimensions](dbItem.EngineeringDimensions)
	if err != nil {
		return nil, fmt.Errorf("unmarshal engineering_dimensions: %w", err)
	}

	planningReorderPoint, err := nullable.UnmarshalNullRawMessage[valueobject.ReorderPoint](dbItem.PlanningReorderPoint)
	if err != nil {
		return nil, fmt.Errorf("unmarshal planning_reorder_point: %w", err)
	}

	cyclicalCount, err := nullable.UnmarshalNullRawMessage[valueobject.CyclicalCountConfig](dbItem.WarehouseCyclicalCountConfig)
	if err != nil {
		return nil, fmt.Errorf("unmarshal cyclical_count_config: %w", err)
	}

	var machines *[]machine.MachineUsage
	if item.Planners.MachinesID != nil {
		list := make([]machine.MachineUsage, len(*item.Planners.MachinesID))
		for i, m := range *item.Planners.MachinesID {
			list[i] = machine.MachineUsage{
				ItemID:    dbItem.ID,
				MachineID: m.MachineID,
				UsageTime: m.UsageTime,
			}
		}
		machines = &list
	}

	return &entity.Item{
		ID: dbItem.ID,

		// ✔ bigint → VO
		Code: valueobject.ItemCode(dbItem.Code),

		Complement: nullable.FromNullString(dbItem.Complement),

		Nature:  entity.ItemNature(dbItem.Nature),
		Inherit: dbItem.Inherit,

		PDM: entity.PDM{
			GroupID:              dbItem.PdmGroupID,
			ModifierID:           dbItem.PdmModifierID,
			Attributes:           pdmAttributes,
			DescriptionTechnique: dbItem.PdmDescriptionTechnique,
		},

		Situation: types.TypeSituationItem(dbItem.Situation),

		// ✔ string → enum
		Health: types.Health(dbItem.Health),

		Warehouse: entity.Warehouse{
			WarehouseID: int(dbItem.WarehouseID),

			UnitOfMeasurement: types.TypeUnitOfMeasurementItem(dbItem.WarehouseUnitOfMeasurement),

			AutomaticLow:                    dbItem.WarehouseAutomaticLow,
			CyclicalCountConfig:             cyclicalCount,
			MinimumStock:                    dbItem.WarehouseMinimumStock,
			AverageMonthlyConsumptionManual: nullable.FromNullInt32ToIntPtr(dbItem.WarehouseAvgMonthlyConsumptionManual),
		},

		Engineering: entity.Engineering{
			ItemBaseCod: nullable.FromNullInt32ToIntPtr(dbItem.EngineeringItemBaseCod),
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
			TankID:       nullable.FromNullInt32ToIntPtr(dbItem.PlanningTankID),
			Ghost:        dbItem.PlanningGhost,
		},

		Planners: entity.Planners{
			EmployeeID: nullable.FromNullInt32(dbItem.PlannerEmployeeID),
			MachinesID: machines,
		},

		Supplies: entity.Supplies{
			TypeOfUse: types.TypeOfUseItem(dbItem.SuppliesTypeOfUse),
		},

		CreatedBy: dbItem.CreatedBy,
		CreatedAt: dbItem.CreatedAt,
	}, nil
}

func (r *RepositoryItemSQLC) FindItemByCode(
	ctx context.Context,
	code valueobject.ItemCode,
) (*entity.Item, error) {

	dbItem, err := r.q.FindItemByCode(ctx, int64(code))
	if err != nil {
		return nil, err
	}

	var pdmAttributes []valueobject.Attribute
	if err := json.Unmarshal(dbItem.PdmAttributes, &pdmAttributes); err != nil {
		return nil, fmt.Errorf("unmarshal pdm_attributes: %w", err)
	}

	var engineeringWeight valueobject.Weight
	if err := json.Unmarshal(dbItem.EngineeringWeight, &engineeringWeight); err != nil {
		return nil, fmt.Errorf("unmarshal engineering_weight: %w", err)
	}

	engineeringDimensions, err := nullable.UnmarshalNullRawMessage[valueobject.Dimensions](dbItem.EngineeringDimensions)
	if err != nil {
		return nil, fmt.Errorf("unmarshal engineering_dimensions: %w", err)
	}

	planningReorderPoint, err := nullable.UnmarshalNullRawMessage[valueobject.ReorderPoint](dbItem.PlanningReorderPoint)
	if err != nil {
		return nil, fmt.Errorf("unmarshal planning_reorder_point: %w", err)
	}

	cyclicalCount, err := nullable.UnmarshalNullRawMessage[valueobject.CyclicalCountConfig](dbItem.WarehouseCyclicalCountConfig)
	if err != nil {
		return nil, fmt.Errorf("unmarshal cyclical_count_config: %w", err)
	}

	var machines *[]machine.MachineUsage

	return &entity.Item{
		ID:   dbItem.ID,
		Code: valueobject.ItemCode(dbItem.Code),

		Complement: nullable.FromNullString(dbItem.Complement),
		Nature:     entity.ItemNature(dbItem.Nature),
		Inherit:    dbItem.Inherit,

		PDM: entity.PDM{
			GroupID:              dbItem.PdmGroupID,
			ModifierID:           dbItem.PdmModifierID,
			Attributes:           pdmAttributes,
			DescriptionTechnique: dbItem.PdmDescriptionTechnique,
		},

		Situation: types.TypeSituationItem(dbItem.Situation),
		Health:    types.Health(dbItem.Health),

		Warehouse: entity.Warehouse{
			WarehouseID: int(dbItem.WarehouseID),
			UnitOfMeasurement: types.TypeUnitOfMeasurementItem(
				dbItem.WarehouseUnitOfMeasurement,
			),
			AutomaticLow:                    dbItem.WarehouseAutomaticLow,
			CyclicalCountConfig:             cyclicalCount,
			MinimumStock:                    dbItem.WarehouseMinimumStock,
			AverageMonthlyConsumptionManual: nullable.FromNullInt32ToIntPtr(dbItem.WarehouseAvgMonthlyConsumptionManual),
		},

		Engineering: entity.Engineering{
			ItemBaseCod: nullable.FromNullInt32ToIntPtr(dbItem.EngineeringItemBaseCod),
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
			TankID:       nullable.FromNullInt32ToIntPtr(dbItem.PlanningTankID),
			Ghost:        dbItem.PlanningGhost,
		},

		Planners: entity.Planners{
			EmployeeID: nullable.FromNullInt32(dbItem.PlannerEmployeeID),
			MachinesID: machines,
		},

		Supplies: entity.Supplies{
			TypeOfUse: types.TypeOfUseItem(dbItem.SuppliesTypeOfUse),
		},

		CreatedBy: dbItem.CreatedBy,
		CreatedAt: dbItem.CreatedAt,
	}, nil
}
