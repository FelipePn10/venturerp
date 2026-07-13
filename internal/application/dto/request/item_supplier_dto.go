package request

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type UpsertItemPreferredSupplierDTO struct {
	ItemCode              int64            `json:"item_code"`
	SupplierCode          int64            `json:"supplier_code"`
	Mask                  string           `json:"mask,omitempty"`
	Ranking               int32            `json:"ranking"`
	SupplierItemCode      *string          `json:"supplier_item_code,omitempty"`
	SupplierDescription   *string          `json:"supplier_description,omitempty"`
	UOM                   *string          `json:"uom,omitempty"`
	XMLUOM                *string          `json:"xml_uom,omitempty"`
	ConversionFactor      *decimal.Decimal `json:"conversion_factor,omitempty"`
	PackageQuantity       decimal.Decimal  `json:"package_quantity"`
	IsPreferred           bool             `json:"is_preferred"`
	ClassificationID      *int64           `json:"classification_id,omitempty"`
	ClassificationDate    *string          `json:"classification_date,omitempty"`
	ClassificationGrade   *decimal.Decimal `json:"classification_grade,omitempty"`
	DirectBilling         bool             `json:"direct_billing"`
	ThirdPartyOrder       bool             `json:"third_party_order"`
	IgnoreAvgCostAddition bool             `json:"ignore_avg_cost_addition"`
	Ecommerce             bool             `json:"ecommerce"`
	Barcode               *string          `json:"barcode,omitempty"`
	Notes                 *string          `json:"notes,omitempty"`
	ValidUntil            *string          `json:"valid_until,omitempty"`
	LeadTimeDays          int32            `json:"lead_time_days,omitempty"`
	CreatedBy             uuid.UUID        `json:"created_by,omitempty"`
}
type CreateItemSupplierQualityReportDTO struct {
	RegisteredOn string  `json:"registered_on"`
	Status       string  `json:"status"`
	FileName     *string `json:"file_name,omitempty"`
	ContentType  *string `json:"content_type,omitempty"`
	Content      []byte  `json:"content,omitempty"`
	Notes        *string `json:"notes,omitempty"`
}
