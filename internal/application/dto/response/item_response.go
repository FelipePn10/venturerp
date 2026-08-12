package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ItemResponse is the API representation of an item with all its folders.
type ItemResponse struct {
	ID          int64                   `json:"id"`
	Code        string                  `json:"code"`
	LegacyCode  int64                   `json:"legacy_code"`
	Name        string                  `json:"name"`
	Complement  *string                 `json:"complement,omitempty"`
	Nature      int                     `json:"nature"`
	PDM         ItemPDMResponse         `json:"pdm"`
	Situation   string                  `json:"situation"`
	Health      string                  `json:"health"`
	Warehouse   ItemWarehouseResponse   `json:"warehouse"`
	Engineering ItemEngineeringResponse `json:"engineering"`
	Planning    ItemPlanningResponse    `json:"planning"`
	Supplies    ItemSuppliesResponse    `json:"supplies"`
	Commercial  ItemCommercialResponse  `json:"commercial"`
	Accounting  ItemAccountingResponse  `json:"accounting"`
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
	ABCClass     *string                   `json:"abc_class,omitempty"`
	MinimumLot   int64                     `json:"minimum_lot"`
	MultipleLot  int64                     `json:"multiple_lot"`
	SafetyStock  int64                     `json:"safety_stock"`
	Critical     bool                      `json:"critical"`
	Exclusive    bool                      `json:"exclusive"`
	Active       bool                      `json:"active"`
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
	TypeOfUse          string  `json:"type_of_use"`
	PurchaseUOM        *string `json:"purchase_uom,omitempty"`
	WarehouseCode      *int64  `json:"warehouse_code,omitempty"`
	ReceivingChecklist bool    `json:"receiving_checklist"`
	Harvest            bool    `json:"harvest"`
}

type ItemCommercialResponse struct {
	Description                      *string          `json:"description,omitempty"`
	SaleType                         *string          `json:"sale_type,omitempty"`
	VolumeConversionFactor           *decimal.Decimal `json:"volume_conversion_factor,omitempty"`
	SaleMultiple                     *decimal.Decimal `json:"sale_multiple,omitempty"`
	MinimumSaleQuantity              *decimal.Decimal `json:"minimum_sale_quantity,omitempty"`
	EstimatedDeliveryDays            *int             `json:"estimated_delivery_days,omitempty"`
	WarrantyDays                     int              `json:"warranty_days"`
	TransferWarehouseCode            *int64           `json:"transfer_warehouse_code,omitempty"`
	TechnicalAssistanceWarehouseCode *int64           `json:"technical_assistance_warehouse_code,omitempty"`
	PackagingItemCode                *int64           `json:"packaging_item_code,omitempty"`
	AllowBillingDescriptionChange    bool             `json:"allow_billing_description_change"`
	IssueLoadingLabels               bool             `json:"issue_loading_labels"`
	AssembleShippingVolumes          bool             `json:"assemble_shipping_volumes"`
	RequiresSpecialPackaging         bool             `json:"requires_special_packaging"`
	WithholdPISCOFINS                bool             `json:"withhold_pis_cofins"`
	IsPackaging                      bool             `json:"is_packaging"`
	MobileEnabled                    bool             `json:"mobile_enabled"`
	ExportPackaging                  bool             `json:"export_packaging"`
	ClassificationCode               *string          `json:"classification_code,omitempty"`
	Notes                            *string          `json:"notes,omitempty"`
}

type ItemAccountingResponse struct {
	SaleFiscalClassificationCode     *string          `json:"sale_fiscal_classification_code,omitempty"`
	PurchaseFiscalClassificationCode *string          `json:"purchase_fiscal_classification_code,omitempty"`
	Origin                           *int             `json:"origin,omitempty"`
	SaleIPIType                      *string          `json:"sale_ipi_type,omitempty"`
	SaleIPIRate                      *decimal.Decimal `json:"sale_ipi_rate,omitempty"`
	PurchaseIPIType                  *string          `json:"purchase_ipi_type,omitempty"`
	PurchaseIPIRate                  *decimal.Decimal `json:"purchase_ipi_rate,omitempty"`
	ICMSRate                         *decimal.Decimal `json:"icms_rate,omitempty"`
	SaleUnitOfMeasurement            *string          `json:"sale_unit_of_measurement,omitempty"`
	PurchaseUnitOfMeasurement        *string          `json:"purchase_unit_of_measurement,omitempty"`
	InventoryGroupCode               *int64           `json:"inventory_group_code,omitempty"`
	AccountingClassificationCode     *string          `json:"accounting_classification_code,omitempty"`
	CEST                             *string          `json:"cest,omitempty"`
	InputCode                        *string          `json:"input_code,omitempty"`
	CalculatePISCOFINS               bool             `json:"calculate_pis_cofins"`
	Notes                            *string          `json:"notes,omitempty"`
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
