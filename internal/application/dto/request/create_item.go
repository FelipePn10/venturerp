package request

import (
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreateItemDTO struct {
	Code             string                  `json:"code"`
	Name             string                  `json:"name"`
	Complement       *string                 `json:"complement,omitempty"`
	Nature           itementity.ItemNature   `json:"nature"`
	PDM              PDMDTO                  `json:"pdm"`
	Situation        types.TypeSituationItem `json:"situation"`
	Health           types.Health            `json:"health"`
	Warehouse        WarehouseDTO            `json:"warehouse"`
	Engineering      EngineeringDTO          `json:"engineering"`
	Planning         PlanningDTO             `json:"planning"`
	Supplies         SuppliesDTO             `json:"supplies"`
	Commercial       *CommercialDTO          `json:"commercial,omitempty"`
	Accounting       *AccountingDTO          `json:"accounting,omitempty"`
	AccountingFiscal *LegacyAccountingDTO    `json:"accounting_fiscal,omitempty"`
	CreatedBy        uuid.UUID               `json:"created_by"`
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
	ABCClass     *string                   `json:"abc_class,omitempty"`
	MinimumLot   int64                     `json:"minimum_lot"`
	MultipleLot  int64                     `json:"multiple_lot"`
	SafetyStock  int64                     `json:"safety_stock"`
	Critical     bool                      `json:"critical"`
	Exclusive    bool                      `json:"exclusive"`
	Active       bool                      `json:"active"`
}

type SuppliesDTO struct {
	TypeOfUse          types.TypeOfUseItem              `json:"type_of_use"`
	PurchaseUOM        *types.TypeUnitOfMeasurementItem `json:"purchase_uom,omitempty"`
	WarehouseCode      *int64                           `json:"warehouse_code,omitempty"`
	ReceivingChecklist bool                             `json:"receiving_checklist"`
	Harvest            bool                             `json:"harvest"`
}

type CommercialDTO struct {
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

type AccountingDTO struct {
	SaleFiscalClassificationCode     *string                          `json:"sale_fiscal_classification_code,omitempty"`
	PurchaseFiscalClassificationCode *string                          `json:"purchase_fiscal_classification_code,omitempty"`
	Origin                           *int                             `json:"origin,omitempty"`
	SaleIPIType                      *string                          `json:"sale_ipi_type,omitempty"`
	SaleIPIRate                      *decimal.Decimal                 `json:"sale_ipi_rate,omitempty"`
	PurchaseIPIType                  *string                          `json:"purchase_ipi_type,omitempty"`
	PurchaseIPIRate                  *decimal.Decimal                 `json:"purchase_ipi_rate,omitempty"`
	ICMSRate                         *decimal.Decimal                 `json:"icms_rate,omitempty"`
	SaleUnitOfMeasurement            *types.TypeUnitOfMeasurementItem `json:"sale_unit_of_measurement,omitempty"`
	PurchaseUnitOfMeasurement        *types.TypeUnitOfMeasurementItem `json:"purchase_unit_of_measurement,omitempty"`
	InventoryGroupCode               *int64                           `json:"inventory_group_code,omitempty"`
	AccountingClassificationCode     *string                          `json:"accounting_classification_code,omitempty"`
	CEST                             *string                          `json:"cest,omitempty"`
	InputCode                        *string                          `json:"input_code,omitempty"`
	CalculatePISCOFINS               bool                             `json:"calculate_pis_cofins"`
	Notes                            *string                          `json:"notes,omitempty"`
}

// LegacyAccountingDTO keeps v1.1.2 request bodies valid during the frontend rollout.
type LegacyAccountingDTO struct {
	Active             bool `json:"active"`
	CalculatePISCOFINS bool `json:"calculate_pis_cofins"`
}
