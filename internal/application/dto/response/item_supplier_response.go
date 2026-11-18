package response

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"time"
)

type ItemPreferredSupplierResponse struct {
	ID                    int64            `json:"id"`
	EnterpriseID          int64            `json:"enterprise_id"`
	ItemCode              int64            `json:"item_code"`
	SupplierCode          int64            `json:"supplier_code"`
	Mask                  string           `json:"mask"`
	Ranking               int32            `json:"ranking"`
	SupplierItemCode      *string          `json:"supplier_item_code,omitempty"`
	SupplierDescription   *string          `json:"supplier_description,omitempty"`
	UOM                   *string          `json:"uom,omitempty"`
	XMLUOM                *string          `json:"xml_uom,omitempty"`
	ConversionFactor      *decimal.Decimal `json:"conversion_factor,omitempty"`
	PackageQuantity       decimal.Decimal  `json:"package_quantity"`
	IsPreferred           bool             `json:"is_preferred"`
	SupplierUF            *string          `json:"supplier_uf,omitempty"`
	ClassificationID      *int64           `json:"classification_id,omitempty"`
	ClassificationDate    *time.Time       `json:"classification_date,omitempty"`
	ClassificationGrade   *decimal.Decimal `json:"classification_grade,omitempty"`
	DirectBilling         bool             `json:"direct_billing"`
	ThirdPartyOrder       bool             `json:"third_party_order"`
	IgnoreAvgCostAddition bool             `json:"ignore_avg_cost_addition"`
	Ecommerce             bool             `json:"ecommerce"`
	Barcode               *string          `json:"barcode,omitempty"`
	Notes                 *string          `json:"notes,omitempty"`
	ValidFrom             *time.Time       `json:"valid_from,omitempty"`
	ValidUntil            *time.Time       `json:"valid_until,omitempty"`
	LeadTimeDays          int32            `json:"lead_time_days"`
	MinOrderQty           decimal.Decimal  `json:"min_order_qty"`
	OrderMultiple         decimal.Decimal  `json:"order_multiple"`
	QuotaPercent          *decimal.Decimal `json:"quota_percent,omitempty"`
	Blocked               bool             `json:"blocked"`
	BlockedReason         *string          `json:"blocked_reason,omitempty"`
	BlockedAt             *time.Time       `json:"blocked_at,omitempty"`
	BlockedBy             *uuid.UUID       `json:"blocked_by,omitempty"`
	// Situacao hoje: VIGENTE, BLOQUEADO, VENCIDO ou FUTURO. Só VIGENTE é fonte.
	Situacao  string    `json:"situacao"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy uuid.UUID `json:"created_by"`
	UpdatedAt time.Time `json:"updated_at"`
}
type ItemSupplierQualityReportResponse struct {
	ID             int64     `json:"id"`
	ItemSupplierID int64     `json:"item_supplier_id"`
	RegisteredOn   time.Time `json:"registered_on"`
	Status         string    `json:"status"`
	FileName       *string   `json:"file_name,omitempty"`
	ContentType    *string   `json:"content_type,omitempty"`
	HasAttachment  bool      `json:"has_attachment"`
	Notes          *string   `json:"notes,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	CreatedBy      uuid.UUID `json:"created_by"`
}

type ItemSupplierBlockEventResponse struct {
	ID             int64     `json:"id"`
	ItemSupplierID int64     `json:"item_supplier_id"`
	Action         string    `json:"action"`
	Reason         string    `json:"reason"`
	ActorID        uuid.UUID `json:"actor_id"`
	CreatedAt      time.Time `json:"created_at"`
}
