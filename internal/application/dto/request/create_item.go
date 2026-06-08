package request

import (
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/google/uuid"
)

type CreateItemDTO struct {
	Code        valueobject.ItemCode    `json:"code"`
	Complement  *string                 `json:"complement,omitempty"`
	Nature      itementity.ItemNature   `json:"nature"`
	PDM         PDMDTO                  `json:"pdm"`
	Situation   types.TypeSituationItem `json:"situation"`
	Health      types.Health            `json:"health"`
	Warehouse   WarehouseDTO            `json:"warehouse"`
	Engineering EngineeringDTO          `json:"engineering"`
	Planning    PlanningDTO             `json:"planning"`
	Supplies    SuppliesDTO             `json:"supplies"`
	CreatedBy   uuid.UUID               `json:"created_by"`
}

type PDMDTO struct {
	GroupCode            int32                   `json:"group_code"`
	ModifierCode         int32                   `json:"modifier_code"`
	Attributes           []valueobject.Attribute `json:"attributes"`
	DescriptionTechnique string                  `json:"description_technique"`
}

type WarehouseDTO struct {
	WarehouseCode                   int                              `json:"warehouse_code"`
	UnitOfMeasurement               types.TypeUnitOfMeasurementItem  `json:"unit_of_measurement"`
	AutomaticLow                    bool                             `json:"automatic_low"`
	CyclicalCountConfig             *valueobject.CyclicalCountConfig `json:"cyclical_count_config,omitempty"`
	MinimumStock                    int32                            `json:"minimum_stock"`
	AverageMonthlyConsumptionManual *int                             `json:"average_monthly_consumption_manual,omitempty"`
}

type EngineeringDTO struct {
	ItemBaseCod *int                    `json:"item_base_cod,omitempty"`
	Weight      valueobject.Weight      `json:"weight"`
	Dimensions  *valueobject.Dimensions `json:"dimensions,omitempty"`
	Type        types.TypeItem          `json:"type"`
	TypeStruct  types.TypeStructItem    `json:"type_struct"`
	OEM         bool                    `json:"oem"`
}

type PlanningDTO struct {
	TypeMRP      types.TypeMRPItem         `json:"type_mrp"`
	LLC          int                       `json:"llc"`
	ReorderPoint *valueobject.ReorderPoint `json:"reorder_point,omitempty"`
	TankCode     *int                      `json:"tank_code,omitempty"`
	Ghost        bool                      `json:"ghost"`
}

type SuppliesDTO struct {
	TypeOfUse types.TypeOfUseItem `json:"type_of_use"`
}
