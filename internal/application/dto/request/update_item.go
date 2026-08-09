package request

import (
	"github.com/FelipePn10/panossoerp/internal/domain/enums/types"
	"github.com/shopspring/decimal"
)

type UpdateItemDTO struct {
	Commercial *UpdateCommercialDTO `json:"commercial,omitempty"`
	Accounting *UpdateAccountingDTO `json:"accounting,omitempty"`
}

type UpdateCommercialDTO struct {
	Description                      **string          `json:"description,omitempty"`
	SaleType                         **string          `json:"sale_type,omitempty"`
	VolumeConversionFactor           **decimal.Decimal `json:"volume_conversion_factor,omitempty"`
	SaleMultiple                     **decimal.Decimal `json:"sale_multiple,omitempty"`
	MinimumSaleQuantity              **decimal.Decimal `json:"minimum_sale_quantity,omitempty"`
	EstimatedDeliveryDays            **int             `json:"estimated_delivery_days,omitempty"`
	WarrantyDays                     *int              `json:"warranty_days,omitempty"`
	TransferWarehouseCode            **int64           `json:"transfer_warehouse_code,omitempty"`
	TechnicalAssistanceWarehouseCode **int64           `json:"technical_assistance_warehouse_code,omitempty"`
	PackagingItemCode                **int64           `json:"packaging_item_code,omitempty"`
	AllowBillingDescriptionChange    *bool             `json:"allow_billing_description_change,omitempty"`
	IssueLoadingLabels               *bool             `json:"issue_loading_labels,omitempty"`
	AssembleShippingVolumes          *bool             `json:"assemble_shipping_volumes,omitempty"`
	RequiresSpecialPackaging         *bool             `json:"requires_special_packaging,omitempty"`
	WithholdPISCOFINS                *bool             `json:"withhold_pis_cofins,omitempty"`
	IsPackaging                      *bool             `json:"is_packaging,omitempty"`
	MobileEnabled                    *bool             `json:"mobile_enabled,omitempty"`
	ExportPackaging                  *bool             `json:"export_packaging,omitempty"`
	ClassificationCode               **string          `json:"classification_code,omitempty"`
	Notes                            **string          `json:"notes,omitempty"`
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
