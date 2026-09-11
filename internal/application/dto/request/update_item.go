package request

import (
	"bytes"
	"encoding/json"

	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	itementity "github.com/FelipePn10/panossoerp/internal/domain/items/entity"
	"github.com/FelipePn10/panossoerp/internal/domain/items/valueobject"
	"github.com/shopspring/decimal"
)

type UpdateItemDTO struct {
	// Identificação. Ponteiros: ausente preserva o valor gravado, presente
	// substitui. Sem isso não havia como corrigir nome, natureza ou situação de
	// um item já cadastrado.
	Name          *string                  `json:"name,omitempty"`
	Complement    **string                 `json:"complement,omitempty"`
	Nature        *itementity.ItemNature   `json:"nature,omitempty"`
	IsBase        *bool                    `json:"is_base,omitempty"`
	IsConfigured  *bool                    `json:"is_configured,omitempty"`
	IsPrototype   *bool                    `json:"is_prototype,omitempty"`
	IsTool        *bool                    `json:"is_tool,omitempty"`
	IsProcessItem *bool                    `json:"is_process_item,omitempty"`
	Situation     *types.TypeSituationItem `json:"situation,omitempty"`
	Health        *types.Health            `json:"health,omitempty"`

	PDM         *UpdatePDMDTO         `json:"pdm,omitempty"`
	Engineering *UpdateEngineeringDTO `json:"engineering,omitempty"`
	Planning    *UpdatePlanningDTO    `json:"planning,omitempty"`
	Supplies    *UpdateSuppliesDTO    `json:"supplies,omitempty"`
	Commercial  *UpdateCommercialDTO  `json:"commercial,omitempty"`
	Accounting  *UpdateAccountingDTO  `json:"accounting,omitempty"`
	Warehouse   *UpdateWarehouseDTO   `json:"warehouse,omitempty"`
}

type UpdatePDMDTO struct {
	GroupCode            *int32  `json:"group_code,omitempty"`
	ModifierCode         *int32  `json:"modifier_code,omitempty"`
	DescriptionTechnique *string `json:"description_technique,omitempty"`
}

type UpdateEngineeringDTO struct {
	Weight     *valueobject.Weight      `json:"weight,omitempty"`
	Dimensions **valueobject.Dimensions `json:"dimensions,omitempty"`
	Type       *types.TypeItem          `json:"type,omitempty"`
	TypeStruct *types.TypeStructItem    `json:"type_struct,omitempty"`
	OEM        *bool                    `json:"oem,omitempty"`
}

type UpdatePlanningDTO struct {
	TypeMRP     *types.TypeMRPItem `json:"type_mrp,omitempty"`
	LLC         *int               `json:"llc,omitempty"`
	Ghost       *bool              `json:"ghost,omitempty"`
	ABCClass    **string           `json:"abc_class,omitempty"`
	MinimumLot  *int64             `json:"minimum_lot,omitempty"`
	MultipleLot *int64             `json:"multiple_lot,omitempty"`
	SafetyStock *int64             `json:"safety_stock,omitempty"`
	Critical    *bool              `json:"critical,omitempty"`
	Exclusive   *bool              `json:"exclusive,omitempty"`
	Active      *bool              `json:"active,omitempty"`
}

type UpdateSuppliesDTO struct {
	TypeOfUse   *types.TypeOfUseItem              `json:"type_of_use,omitempty"`
	PurchaseUOM **types.TypeUnitOfMeasurementItem `json:"purchase_uom,omitempty"`
	// WarehouseCode não existia na alteração: o almoxarifado de suprimentos
	// escolhido na criação ficava para sempre, e trocá-lo respondia 200 sem
	// gravar nada.
	WarehouseCode      *int64  `json:"warehouse_code,omitempty"`
	ReceivingChecklist *bool   `json:"receiving_checklist,omitempty"`
	Harvest            *bool   `json:"harvest,omitempty"`
	Notes              *string `json:"notes,omitempty"`
}

type UpdateWarehouseDTO struct {
	// WarehouseCode e AverageMonthlyConsumptionManual não existiam na
	// alteração: o almoxarifado principal e o consumo médio informado à mão
	// ficavam como na criação, e a tela respondia "salvo".
	WarehouseCode                   *int64                           `json:"warehouse_code,omitempty"`
	CyclicalCountConfig             OptionalCyclicalCountConfig      `json:"cyclical_count_config,omitempty"`
	UnitOfMeasurement               *types.TypeUnitOfMeasurementItem `json:"unit_of_measurement,omitempty"`
	AutomaticLow                    *bool                            `json:"automatic_low,omitempty"`
	MinimumStock                    *int32                           `json:"minimum_stock,omitempty"`
	AverageMonthlyConsumptionManual *int                             `json:"average_monthly_consumption_manual,omitempty"`
}

// OptionalCyclicalCountConfig distinguishes omission (preserve) from an
// explicit null (disable), which plain pointers cannot do with encoding/json.
type OptionalCyclicalCountConfig struct {
	Set   bool
	Value *valueobject.CyclicalCountConfig
}

func (o *OptionalCyclicalCountConfig) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		o.Value = nil
		return nil
	}
	var value valueobject.CyclicalCountConfig
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	o.Value = &value
	return nil
}

type UpdateCommercialDTO struct {
	Description                      **string          `json:"description,omitempty"`
	SaleType                         **string          `json:"sale_type,omitempty"`
	VolumeConversionFactor           **decimal.Decimal `json:"volume_conversion_factor,omitempty"`
	SaleMultiple                     **decimal.Decimal `json:"sale_multiple,omitempty"`
	MinimumSaleQuantity              **decimal.Decimal `json:"minimum_sale_quantity,omitempty"`
	EstimatedDeliveryDays            **int             `json:"estimated_delivery_days,omitempty"`
	WarrantyDays                     *int              `json:"warranty_days,omitempty"`
	TransferWarehouseCode            *CodigoFlexivel   `json:"transfer_warehouse_code,omitempty"`
	TechnicalAssistanceWarehouseCode *CodigoFlexivel   `json:"technical_assistance_warehouse_code,omitempty"`
	// PackagingItemCode chega como código de negócio (texto), como a tela o conhece.
	PackagingItemCode             **TextCode `json:"packaging_item_code,omitempty"`
	AllowBillingDescriptionChange *bool      `json:"allow_billing_description_change,omitempty"`
	IssueLoadingLabels            *bool      `json:"issue_loading_labels,omitempty"`
	AssembleShippingVolumes       *bool      `json:"assemble_shipping_volumes,omitempty"`
	RequiresSpecialPackaging      *bool      `json:"requires_special_packaging,omitempty"`
	WithholdPISCOFINS             *bool      `json:"withhold_pis_cofins,omitempty"`
	IsPackaging                   *bool      `json:"is_packaging,omitempty"`
	MobileEnabled                 *bool      `json:"mobile_enabled,omitempty"`
	ExportPackaging               *bool      `json:"export_packaging,omitempty"`
	ClassificationCode            **string   `json:"classification_code,omitempty"`
	Notes                         **string   `json:"notes,omitempty"`
}

type UpdateAccountingDTO struct {
	SaleFiscalClassificationCode     **string                          `json:"sale_fiscal_classification_code,omitempty"`
	PurchaseFiscalClassificationCode **string                          `json:"purchase_fiscal_classification_code,omitempty"`
	Origin                           **int                             `json:"origin,omitempty"`
	SaleIPIType                      **string                          `json:"sale_ipi_type,omitempty"`
	SaleIPIRate                      **decimal.Decimal                 `json:"sale_ipi_rate,omitempty"`
	PurchaseIPIType                  **string                          `json:"purchase_ipi_type,omitempty"`
	PurchaseIPIRate                  **decimal.Decimal                 `json:"purchase_ipi_rate,omitempty"`
	ICMSRate                         **decimal.Decimal                 `json:"icms_rate,omitempty"`
	SaleUnitOfMeasurement            **types.TypeUnitOfMeasurementItem `json:"sale_unit_of_measurement,omitempty"`
	PurchaseUnitOfMeasurement        **types.TypeUnitOfMeasurementItem `json:"purchase_unit_of_measurement,omitempty"`
	InventoryGroupCode               **int64                           `json:"inventory_group_code,omitempty"`
	AccountingClassificationCode     **string                          `json:"accounting_classification_code,omitempty"`
	CEST                             **string                          `json:"cest,omitempty"`
	InputCode                        **string                          `json:"input_code,omitempty"`
	CalculatePISCOFINS               *bool                             `json:"calculate_pis_cofins,omitempty"`
	Notes                            **string                          `json:"notes,omitempty"`
}
