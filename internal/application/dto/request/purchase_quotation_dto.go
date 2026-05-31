package request

import "github.com/google/uuid"

// CreatePurchaseQuotationDTO releases requisition items and/or planned orders to
// a new quotation, optionally inviting suppliers.
type CreatePurchaseQuotationDTO struct {
	EnterpriseCode     int64     `json:"enterprise_code"`
	Notes              *string   `json:"notes,omitempty"`
	CreatedBy          uuid.UUID `json:"created_by"`
	RequisitionItemIDs []int64   `json:"requisition_item_ids,omitempty"`
	PlannedOrderCodes  []int64   `json:"planned_order_codes,omitempty"`
	SupplierCodes      []int64   `json:"supplier_codes,omitempty"`
}

type AddQuotationSupplierDTO struct {
	QuotationCode int64 `json:"-"`
	SupplierCode  int64 `json:"supplier_code"`
}

type RecordQuotationPriceDTO struct {
	QuotationItemID int64   `json:"quotation_item_id"`
	SupplierCode    int64   `json:"supplier_code"`
	UnitPrice       float64 `json:"unit_price"`
	LeadTimeDays    int32   `json:"lead_time_days,omitempty"`
	PaymentTermCode *int64  `json:"payment_term_code,omitempty"`
	Notes           *string `json:"notes,omitempty"`
}

type GenerateOrdersFromQuotationDTO struct {
	QuotationCode int64     `json:"-"`
	CreatedBy     uuid.UUID `json:"created_by"`
}
