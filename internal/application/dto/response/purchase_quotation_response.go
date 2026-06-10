package response

import (
	"time"

	"github.com/google/uuid"
)

// PurchaseQuotationResponse is the API representation of a purchase quotation.
type PurchaseQuotationResponse struct {
	ID             int64                                `json:"id"`
	Code           int64                                `json:"code"`
	EnterpriseCode int64                                `json:"enterprise_code"`
	Status         string                               `json:"status"`
	EmissionDate   time.Time                            `json:"emission_date"`
	Notes          *string                              `json:"notes,omitempty"`
	IsActive       bool                                 `json:"is_active"`
	CreatedAt      time.Time                            `json:"created_at"`
	CreatedBy      uuid.UUID                            `json:"created_by"`
	UpdatedAt      time.Time                            `json:"updated_at"`
	Items          []PurchaseQuotationItemResponse      `json:"items,omitempty"`
	Suppliers      []PurchaseQuotationSupplierResponse  `json:"suppliers,omitempty"`
}

// PurchaseQuotationItemResponse is the API representation of a quotation line.
type PurchaseQuotationItemResponse struct {
	ID            int64                            `json:"id"`
	QuotationCode int64                            `json:"quotation_code"`
	Sequence      int32                            `json:"sequence"`
	ItemCode      int64                            `json:"item_code"`
	Quantity      float64                          `json:"quantity"`
	UOM           *string                          `json:"uom,omitempty"`
	DeliveryDate  *time.Time                       `json:"delivery_date,omitempty"`
	SourceType    string                           `json:"source_type"`
	SourceCode    *int64                           `json:"source_code,omitempty"`
	SourceItemID  *int64                           `json:"source_item_id,omitempty"`
	IsConfigured  bool                             `json:"is_configured"`
	CreatedAt     time.Time                        `json:"created_at"`
	Prices        []PurchaseQuotationPriceResponse `json:"prices,omitempty"`
}

// PurchaseQuotationSupplierResponse is the API representation of an invited supplier.
type PurchaseQuotationSupplierResponse struct {
	ID            int64     `json:"id"`
	QuotationCode int64     `json:"quotation_code"`
	SupplierCode  int64     `json:"supplier_code"`
	InvitedAt     time.Time `json:"invited_at"`
}

// PurchaseQuotationPriceResponse is the API representation of a quoted price.
type PurchaseQuotationPriceResponse struct {
	ID              int64     `json:"id"`
	QuotationItemID int64     `json:"quotation_item_id"`
	SupplierCode    int64     `json:"supplier_code"`
	UnitPrice       float64   `json:"unit_price"`
	LeadTimeDays    int32     `json:"lead_time_days"`
	PaymentTermCode *int64    `json:"payment_term_code,omitempty"`
	Notes           *string   `json:"notes,omitempty"`
	IsSelected      bool      `json:"is_selected"`
	CreatedAt       time.Time `json:"created_at"`
}
