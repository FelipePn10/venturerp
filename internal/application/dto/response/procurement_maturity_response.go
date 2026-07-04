package response

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ProcurementRecordResponse struct {
	ID                    int64           `json:"id"`
	RecordType            string          `json:"record_type"`
	Status                string          `json:"status"`
	SupplierCode          *int64          `json:"supplier_code,omitempty"`
	PurchaseOrderCode     *int64          `json:"purchase_order_code,omitempty"`
	PurchaseOrderItemCode *int64          `json:"purchase_order_item_code,omitempty"`
	ItemCode              *int64          `json:"item_code,omitempty"`
	Mask                  string          `json:"mask"`
	WarehouseID           *int64          `json:"warehouse_id,omitempty"`
	Quantity              float64         `json:"quantity"`
	Reference             *string         `json:"reference,omitempty"`
	Payload               json.RawMessage `json:"payload"`
	OpenedAt              time.Time       `json:"opened_at"`
	ClosedAt              *time.Time      `json:"closed_at,omitempty"`
	CreatedBy             *uuid.UUID      `json:"created_by,omitempty"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

type ReceivingInspectionDispositionResponse struct {
	ID                     int64                      `json:"id"`
	RecordID               int64                      `json:"record_id"`
	ApprovedQty            float64                    `json:"approved_qty"`
	RejectedQty            float64                    `json:"rejected_qty"`
	QuarantineWarehouseID  *int64                     `json:"quarantine_warehouse_id,omitempty"`
	DestinationWarehouseID *int64                     `json:"destination_warehouse_id,omitempty"`
	Reason                 *string                    `json:"reason,omitempty"`
	DisposedAt             time.Time                  `json:"disposed_at"`
	DisposedBy             *uuid.UUID                 `json:"disposed_by,omitempty"`
	Inspection             *ProcurementRecordResponse `json:"inspection"`
	Movements              []StockMovementResponse    `json:"movements"`
}

type SupplierScorecardResponse struct {
	ID               int64      `json:"id"`
	SupplierCode     int64      `json:"supplier_code"`
	PeriodStart      time.Time  `json:"period_start"`
	PeriodEnd        time.Time  `json:"period_end"`
	QualityScore     float64    `json:"quality_score"`
	DeliveryScore    float64    `json:"delivery_score"`
	CommercialScore  float64    `json:"commercial_score"`
	ServiceScore     float64    `json:"service_score"`
	OverallScore     float64    `json:"overall_score"`
	TotalReceipts    int        `json:"total_receipts"`
	RejectedReceipts int        `json:"rejected_receipts"`
	LateReceipts     int        `json:"late_receipts"`
	Notes            *string    `json:"notes,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	CreatedBy        *uuid.UUID `json:"created_by,omitempty"`
	// Computed indicates the scorecard was derived from real data (IQF auto-compute)
	// rather than entered manually. Persisted is true when it was also saved.
	Computed  bool `json:"computed,omitempty"`
	Persisted bool `json:"persisted,omitempty"`
}

type ApprovalLimitResponse struct {
	ID             int64      `json:"id"`
	EnterpriseCode int64      `json:"enterprise_code"`
	Scope          string     `json:"scope"`
	ScopeRef       *string    `json:"scope_ref,omitempty"`
	Currency       string     `json:"currency"`
	AutoApproveMax float64    `json:"auto_approve_max"`
	BlockAbove     *float64   `json:"block_above,omitempty"`
	IsActive       bool       `json:"is_active"`
	ValidFrom      time.Time  `json:"valid_from"`
	ValidTo        *time.Time `json:"valid_to,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	CreatedBy      *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type SupplierContractResponse struct {
	ID             int64                          `json:"id"`
	EnterpriseCode int64                          `json:"enterprise_code"`
	SupplierCode   int64                          `json:"supplier_code"`
	ContractNumber string                         `json:"contract_number"`
	Description    *string                        `json:"description,omitempty"`
	Status         string                         `json:"status"`
	Currency       string                         `json:"currency"`
	ValidFrom      time.Time                      `json:"valid_from"`
	ValidTo        *time.Time                     `json:"valid_to,omitempty"`
	PriceIndex     *string                        `json:"price_index,omitempty"`
	Notes          *string                        `json:"notes,omitempty"`
	CreatedBy      *uuid.UUID                     `json:"created_by,omitempty"`
	CreatedAt      time.Time                      `json:"created_at"`
	Items          []SupplierContractItemResponse `json:"items"`
}

type SupplierContractItemResponse struct {
	ID            int64   `json:"id"`
	ContractID    int64   `json:"contract_id"`
	ItemCode      int64   `json:"item_code"`
	Mask          string  `json:"mask"`
	Unit          *string `json:"unit,omitempty"`
	ContractedQty float64 `json:"contracted_qty"`
	ConsumedQty   float64 `json:"consumed_qty"`
	RemainingQty  float64 `json:"remaining_qty"`
	UnitPrice     float64 `json:"unit_price"`
	MinOrderQty   float64 `json:"min_order_qty"`
	Notes         *string `json:"notes,omitempty"`
}

type PurchaseMovementHistoryResponse struct {
	SupplierCode      *int64     `json:"supplier_code,omitempty"`
	PurchaseOrderCode int64      `json:"purchase_order_code"`
	OrderNumber       int64      `json:"order_number"`
	ItemCode          int64      `json:"item_code"`
	Mask              string     `json:"mask"`
	RequestedQty      float64    `json:"requested_qty"`
	ReceivedQty       float64    `json:"received_qty"`
	CancelledQty      float64    `json:"cancelled_qty"`
	OpenQty           float64    `json:"open_qty"`
	UnitPrice         float64    `json:"unit_price"`
	Status            string     `json:"status"`
	EmissionDate      time.Time  `json:"emission_date"`
	DeliveryDate      *time.Time `json:"delivery_date,omitempty"`
}
