package request

import "encoding/json"

type CreateProcurementRecordDTO struct {
	RecordType            string          `json:"record_type"`
	Status                string          `json:"status"`
	SupplierCode          *int64          `json:"supplier_code"`
	PurchaseOrderCode     *int64          `json:"purchase_order_code"`
	PurchaseOrderItemCode *int64          `json:"purchase_order_item_code"`
	ItemCode              *int64          `json:"item_code"`
	Mask                  string          `json:"mask"`
	WarehouseID           *int64          `json:"warehouse_id"`
	Quantity              float64         `json:"quantity"`
	Reference             *string         `json:"reference"`
	Payload               json.RawMessage `json:"payload"`
}

type UpdateProcurementRecordStatusDTO struct {
	Status string `json:"status"`
}

type DisposeReceivingInspectionDTO struct {
	ApprovedQty            float64 `json:"approved_qty"`
	RejectedQty            float64 `json:"rejected_qty"`
	QuarantineWarehouseID  *int64  `json:"quarantine_warehouse_id"`
	DestinationWarehouseID *int64  `json:"destination_warehouse_id"`
	Reason                 *string `json:"reason"`
}

type CreateSupplierScorecardDTO struct {
	SupplierCode     int64   `json:"supplier_code"`
	PeriodStart      string  `json:"period_start"`
	PeriodEnd        string  `json:"period_end"`
	QualityScore     float64 `json:"quality_score"`
	DeliveryScore    float64 `json:"delivery_score"`
	CommercialScore  float64 `json:"commercial_score"`
	ServiceScore     float64 `json:"service_score"`
	TotalReceipts    int     `json:"total_receipts"`
	RejectedReceipts int     `json:"rejected_receipts"`
	LateReceipts     int     `json:"late_receipts"`
	Notes            *string `json:"notes"`
}

// ComputeSupplierScorecardDTO drives IQF auto-computation from real receiving
// inspection and delivery data. CommercialScore/ServiceScore are optional manual
// inputs (they have no objective source yet) and default to 100.
type ComputeSupplierScorecardDTO struct {
	SupplierCode    int64   `json:"supplier_code"`
	PeriodStart     string  `json:"period_start"`
	PeriodEnd       string  `json:"period_end"`
	CommercialScore float64 `json:"commercial_score"`
	ServiceScore    float64 `json:"service_score"`
	Persist         bool    `json:"persist"`
	Notes           *string `json:"notes"`
}

type CreateApprovalLimitDTO struct {
	EnterpriseCode int64    `json:"enterprise_code"`
	Scope          string   `json:"scope"`
	ScopeRef       *string  `json:"scope_ref"`
	Currency       string   `json:"currency"`
	AutoApproveMax float64  `json:"auto_approve_max"`
	BlockAbove     *float64 `json:"block_above"`
	ValidFrom      string   `json:"valid_from"`
	ValidTo        *string  `json:"valid_to"`
	Notes          *string  `json:"notes"`
}

type CreateSupplierContractDTO struct {
	EnterpriseCode int64                          `json:"enterprise_code"`
	SupplierCode   int64                          `json:"supplier_code"`
	ContractNumber string                         `json:"contract_number"`
	Description    *string                        `json:"description"`
	Status         string                         `json:"status"`
	Currency       string                         `json:"currency"`
	ValidFrom      string                         `json:"valid_from"`
	ValidTo        *string                        `json:"valid_to"`
	PriceIndex     *string                        `json:"price_index"`
	Notes          *string                        `json:"notes"`
	Items          []SupplierContractItemInputDTO `json:"items"`
}

type SupplierContractItemInputDTO struct {
	ItemCode      int64   `json:"item_code"`
	Mask          string  `json:"mask"`
	Unit          *string `json:"unit"`
	ContractedQty float64 `json:"contracted_qty"`
	UnitPrice     float64 `json:"unit_price"`
	MinOrderQty   float64 `json:"min_order_qty"`
	Notes         *string `json:"notes"`
}

type UpdateSupplierContractStatusDTO struct {
	Status string `json:"status"`
}

type ConsumeSupplierContractDTO struct {
	ItemCode int64   `json:"item_code"`
	Mask     string  `json:"mask"`
	Quantity float64 `json:"quantity"`
}
