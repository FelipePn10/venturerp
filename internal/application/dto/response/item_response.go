package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ItemResponse is the API representation of an item with all its folders.
type ItemResponse struct {
	ID         int64   `json:"id"`
	Code       string  `json:"code"`
	LegacyCode int64   `json:"legacy_code"`
	Name       string  `json:"name"`
	Complement *string `json:"complement,omitempty"`
	Nature     int     `json:"nature"`

	LongDescription        *string `json:"long_description,omitempty"`
	GeneralNotes           *string `json:"general_notes,omitempty"`
	Revision               *string `json:"revision,omitempty"`
	PreviousCode           *string `json:"previous_code,omitempty"`
	ManufacturerName       *string `json:"manufacturer_name,omitempty"`
	ManufacturerPartNumber *string `json:"manufacturer_part_number,omitempty"`
	IsScrap                bool    `json:"is_scrap"`
	DescriptionEN          *string `json:"description_en,omitempty"`
	// Marcadores independentes: um item pode ser base e configurado ao mesmo tempo.
	IsBase          bool                        `json:"is_base"`
	IsConfigured    bool                        `json:"is_configured"`
	IsPrototype     bool                        `json:"is_prototype"`
	IsTool          bool                        `json:"is_tool"`
	IsProcessItem   bool                        `json:"is_process_item"`
	PDM             ItemPDMResponse             `json:"pdm"`
	Situation       string                      `json:"situation"`
	Health          string                      `json:"health"`
	Warehouse       ItemWarehouseResponse       `json:"warehouse"`
	Engineering     ItemEngineeringResponse     `json:"engineering"`
	Planning        ItemPlanningResponse        `json:"planning"`
	Supplies        ItemSuppliesResponse        `json:"supplies"`
	Commercial      ItemCommercialResponse      `json:"commercial"`
	Accounting      ItemAccountingResponse      `json:"accounting"`
	FiscalEffective ItemFiscalEffectiveResponse `json:"fiscal_effective"`
	CreatedBy       uuid.UUID                   `json:"created_by"`
	CreatedAt       time.Time                   `json:"created_at"`
}

type EffectiveFiscalContextResponse struct {
	ClassificationID   int64             `json:"classification_id"`
	ClassificationCode int64             `json:"classification_code"`
	NCM                *string           `json:"ncm,omitempty"`
	CEST               *string           `json:"cest,omitempty"`
	Unit               *string           `json:"unit,omitempty"`
	Origin             *int              `json:"origin,omitempty"`
	IPIRate            *decimal.Decimal  `json:"ipi_rate,omitempty"`
	ICMSRate           *decimal.Decimal  `json:"icms_rate,omitempty"`
	PISRate            *decimal.Decimal  `json:"pis_rate,omitempty"`
	COFINSRate         *decimal.Decimal  `json:"cofins_rate,omitempty"`
	CalculatePISCOFINS *bool             `json:"calculate_pis_cofins,omitempty"`
	Sources            map[string]string `json:"sources"`
}
type ItemFiscalEffectiveResponse struct {
	Purchase *EffectiveFiscalContextResponse `json:"purchase,omitempty"`
	Sale     *EffectiveFiscalContextResponse `json:"sale,omitempty"`
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
	WarehouseCode                   int                              `json:"warehouse_code"`
	UnitOfMeasurement               string                           `json:"unit_of_measurement"`
	AutomaticLow                    bool                             `json:"automatic_low"`
	CyclicalCountConfig             *ItemCyclicalCountConfigResponse `json:"cyclical_count_config,omitempty"`
	MinimumStock                    float64                          `json:"minimum_stock"`
	AverageMonthlyConsumptionManual *float64                         `json:"average_monthly_consumption_manual,omitempty"`
	ShelfLifeDays                   *int                             `json:"shelf_life_days,omitempty"`
	MinRemainingLifeDays            *int                             `json:"min_remaining_life_days,omitempty"`
	StorageConditions               *string                          `json:"storage_conditions,omitempty"`
	Hazardous                       bool                             `json:"hazardous"`
	UNNumber                        *string                          `json:"un_number,omitempty"`
	HazardClass                     *string                          `json:"hazard_class,omitempty"`
	AcceptsFractionalQuantity       bool                             `json:"accepts_fractional_quantity"`
	SecondaryUOM                    *string                          `json:"secondary_uom,omitempty"`
	SecondaryFactor                 *float64                         `json:"secondary_factor,omitempty"`
}

type ItemCyclicalCountConfigResponse struct {
	DaysInterval int `json:"days_interval"`
}

// ItemEngineeringResponse is the engineering folder of an item.
type ItemEngineeringResponse struct {
	// ItemBaseCod é o código de negócio (texto) do item-base;
	// LegacyItemBaseCod mantém a chave numérica interna.
	ItemBaseCod       *string                 `json:"item_base_cod,omitempty"`
	LegacyItemBaseCod *int                    `json:"legacy_item_base_cod,omitempty"`
	Weight            ItemWeightResponse      `json:"weight"`
	Dimensions        *ItemDimensionsResponse `json:"dimensions,omitempty"`
	Type              string                  `json:"type"`
	TypeStruct        string                  `json:"type_struct"`
	OEM               bool                    `json:"oem"`
	Volume            *float64                `json:"volume,omitempty"`
	RequiresDrawing   bool                    `json:"requires_drawing"`
	CuttingGrain      string                  `json:"cutting_grain"`
	CuttingNoRotation bool                    `json:"cutting_no_rotation"`
}

// ItemWeightResponse is an item weight value.
type ItemWeightResponse struct {
	Gross float64 `json:"gross"`
	Net   float64 `json:"net"`
	Unit  string  `json:"unit"`
}

// ItemDimensionsResponse is an item dimensions value.
type ItemDimensionsResponse struct {
	Length int     `json:"length"`
	Width  int     `json:"width"`
	Height float64 `json:"height"`
}

// ItemPlanningResponse is the planning folder of an item.
type ItemPlanningResponse struct {
	TypeMRP      string                    `json:"type_mrp"`
	LLC          int                       `json:"llc"`
	ReorderPoint *ItemReorderPointResponse `json:"reorder_point,omitempty"`
	TankCode     *int                      `json:"tank_code,omitempty"`
	Ghost        bool                      `json:"ghost"`
	ABCClass     *string                   `json:"abc_class,omitempty"`
	MinimumLot   float64                   `json:"minimum_lot"`
	MultipleLot  float64                   `json:"multiple_lot"`
	SafetyStock  float64                   `json:"safety_stock"`
	Critical     bool                      `json:"critical"`
	Exclusive    bool                      `json:"exclusive"`
	Active       bool                      `json:"active"`

	PlannerEmployeeCode       *int64   `json:"planner_employee_code,omitempty"`
	BuyerEmployeeCode         *int64   `json:"buyer_employee_code,omitempty"`
	PurchaseLeadTimeDays      *int     `json:"purchase_lead_time_days,omitempty"`
	ManufacturingLeadTimeDays *int     `json:"manufacturing_lead_time_days,omitempty"`
	Notes                     *string  `json:"notes,omitempty"`
	MaterialIssueTiming       string   `json:"material_issue_timing"`
	ProductionReportingType   string   `json:"production_reporting_type"`
	MaximumStock              float64  `json:"maximum_stock"`
	Coverage                  int      `json:"coverage"`
	GroupingKey               *string  `json:"grouping_key,omitempty"`
	SafetyTimeDays            *int     `json:"safety_time_days,omitempty"`
	MaximumLot                *float64 `json:"maximum_lot,omitempty"`
}

// ItemReorderPointResponse is an item reorder point value.
type ItemReorderPointResponse struct {
	TR int     `json:"tr"`
	CM float64 `json:"cm"`
	CR int     `json:"cr"`
	ES float64 `json:"es"`
}

// ItemSuppliesResponse is the supplies folder of an item.
type ItemSuppliesResponse struct {
	TypeOfUse          string  `json:"type_of_use"`
	PurchaseUOM        *string `json:"purchase_uom,omitempty"`
	WarehouseCode      *int64  `json:"warehouse_code,omitempty"`
	ReceivingChecklist bool    `json:"receiving_checklist"`
	Harvest            bool    `json:"harvest"`
	Notes              *string `json:"notes,omitempty"`

	UnderDeliveryTolerancePct *float64 `json:"under_delivery_tolerance_pct,omitempty"`
	OverDeliveryTolerancePct  *float64 `json:"over_delivery_tolerance_pct,omitempty"`
	ReceiptRouting            string   `json:"receipt_routing"`
	ClassificationCode        *string  `json:"classification_code,omitempty"`
	RequiresCertificate       bool     `json:"requires_certificate"`
}

type ItemCommercialResponse struct {
	Description                      *string          `json:"description,omitempty"`
	SaleType                         *string          `json:"sale_type,omitempty"`
	VolumeConversionFactor           *decimal.Decimal `json:"volume_conversion_factor,omitempty"`
	SaleMultiple                     *decimal.Decimal `json:"sale_multiple,omitempty"`
	MinimumSaleQuantity              *decimal.Decimal `json:"minimum_sale_quantity,omitempty"`
	EstimatedDeliveryDays            *int             `json:"estimated_delivery_days,omitempty"`
	WarrantyDays                     int              `json:"warranty_days"`
	TransferWarehouseCode            *string          `json:"transfer_warehouse_code,omitempty"`
	TechnicalAssistanceWarehouseCode *string          `json:"technical_assistance_warehouse_code,omitempty"`
	// PackagingItemCode é o código de negócio (texto) do item de embalagem, como
	// a tela o exibe; LegacyPackagingItemCode mantém a chave numérica interna.
	PackagingItemCode             *string `json:"packaging_item_code,omitempty"`
	LegacyPackagingItemCode       *int64  `json:"legacy_packaging_item_code,omitempty"`
	AllowBillingDescriptionChange bool    `json:"allow_billing_description_change"`
	IssueLoadingLabels            bool    `json:"issue_loading_labels"`
	AssembleShippingVolumes       bool    `json:"assemble_shipping_volumes"`
	RequiresSpecialPackaging      bool    `json:"requires_special_packaging"`
	WithholdPISCOFINS             bool    `json:"withhold_pis_cofins"`
	IsPackaging                   bool    `json:"is_packaging"`
	MobileEnabled                 bool    `json:"mobile_enabled"`
	ExportPackaging               bool    `json:"export_packaging"`
	ClassificationCode            *string `json:"classification_code,omitempty"`
	Notes                         *string `json:"notes,omitempty"`
	ExclusiveCustomerCode         *int64  `json:"exclusive_customer_code,omitempty"`
	VolumesPerUnit                *int    `json:"volumes_per_unit,omitempty"`
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
	CalculatePISCOFINS               *bool            `json:"calculate_pis_cofins,omitempty"`
	Notes                            *string          `json:"notes,omitempty"`

	GTIN             *string  `json:"gtin,omitempty"`
	GTINTax          *string  `json:"gtin_tax,omitempty"`
	EFDItemType      *string  `json:"efd_item_type,omitempty"`
	TaxUnit          *string  `json:"tax_unit,omitempty"`
	TaxUnitFactor    *float64 `json:"tax_unit_factor,omitempty"`
	ExTIPI           *string  `json:"ex_tipi,omitempty"`
	RelevantScale    *bool    `json:"relevant_scale,omitempty"`
	ManufacturerCNPJ *string  `json:"manufacturer_cnpj,omitempty"`
	FCINumber        *string  `json:"fci_number,omitempty"`
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
