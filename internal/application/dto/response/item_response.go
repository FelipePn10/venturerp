package response

import (
	"time"

	"github.com/google/uuid"
)

// ItemResponse is the API representation of an item with all its folders.
type ItemResponse struct {
	ID          int64                   `json:"id"`
	Code        int64                   `json:"code"`
	Complement  *string                 `json:"complement,omitempty"`
	Nature      int                     `json:"nature"`
	PDM         ItemPDMResponse         `json:"pdm"`
	Situation   string                  `json:"situation"`
	Health      string                  `json:"health"`
	Warehouse   ItemWarehouseResponse   `json:"warehouse"`
	Engineering ItemEngineeringResponse `json:"engineering"`
	Planning    ItemPlanningResponse    `json:"planning"`
	Supplies    ItemSuppliesResponse    `json:"supplies"`
	CreatedBy   uuid.UUID               `json:"created_by"`
	CreatedAt   time.Time               `json:"created_at"`
}

// ItemPDMResponse is the PDM (descriptive) folder of an item.
type ItemPDMResponse struct {
	GroupCode            int32                   `json:"group_code"`
	ModifierCode         int32                   `json:"modifier_code"`
	Attributes           []ItemAttributeResponse `json:"attributes,omitempty"`
	DescriptionTechnique string                  `json:"description_technique"`
}

// ItemAttributeResponse is a single PDM attribute.
type ItemAttributeResponse struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ItemWarehouseResponse is the warehouse folder of an item.
type ItemWarehouseResponse struct {
	WarehouseCode                   int    `json:"warehouse_code"`
	UnitOfMeasurement               string `json:"unit_of_measurement"`
	AutomaticLow                    bool   `json:"automatic_low"`
	CyclicalCountDaysInterval       *int   `json:"cyclical_count_days_interval,omitempty"`
	MinimumStock                    int32  `json:"minimum_stock"`
	AverageMonthlyConsumptionManual *int   `json:"average_monthly_consumption_manual,omitempty"`
}

// ItemEngineeringResponse is the engineering folder of an item.
type ItemEngineeringResponse struct {
	ItemBaseCod *int                    `json:"item_base_cod,omitempty"`
	Weight      ItemWeightResponse      `json:"weight"`
	Dimensions  *ItemDimensionsResponse `json:"dimensions,omitempty"`
	Type        string                  `json:"type"`
	TypeStruct  string                  `json:"type_struct"`
	OEM         bool                    `json:"oem"`
}

// ItemWeightResponse is an item weight value.
type ItemWeightResponse struct {
	Gross float64 `json:"gross"`
	Net   float64 `json:"net"`
	Unit  string  `json:"unit"`
}

// ItemDimensionsResponse is an item dimensions value.
type ItemDimensionsResponse struct {
	Length int `json:"length"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// ItemPlanningResponse is the planning folder of an item.
type ItemPlanningResponse struct {
	TypeMRP      string                    `json:"type_mrp"`
	LLC          int                       `json:"llc"`
	ReorderPoint *ItemReorderPointResponse `json:"reorder_point,omitempty"`
	TankCode     *int                      `json:"tank_code,omitempty"`
	Ghost        bool                      `json:"ghost"`
}

// ItemReorderPointResponse is an item reorder point value.
type ItemReorderPointResponse struct {
	TR int16 `json:"tr"`
	CM int16 `json:"cm"`
	CR int   `json:"cr"`
	ES int16 `json:"es"`
}

// ItemSuppliesResponse is the supplies folder of an item.
type ItemSuppliesResponse struct {
	TypeOfUse string `json:"type_of_use"`
}

// MaskSummaryResponse is a compact representation of an item mask.
type MaskSummaryResponse struct {
	ID        int64     `json:"id"`
	Mask      string    `json:"mask"`
	MaskHash  string    `json:"mask_hash"`
	CreatedAt time.Time `json:"created_at"`
}

// ItemWithMasksResponse pairs an item with its registered masks.
type ItemWithMasksResponse struct {
	Item  *ItemResponse         `json:"item"`
	Masks []MaskSummaryResponse `json:"masks,omitempty"`
}
