package response

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ---- Receiving notice + divergences (FAVR) ----

type ReceivingNoticeResponse struct {
	ID                int64                         `json:"id"`
	EnterpriseCode    int64                         `json:"enterprise_code"`
	NoticeNumber      int64                         `json:"notice_number"`
	SupplierCode      *int64                        `json:"supplier_code,omitempty"`
	PurchaseOrderCode *int64                        `json:"purchase_order_code,omitempty"`
	CarrierCode       *int64                        `json:"carrier_code,omitempty"`
	Status            string                        `json:"status"`
	Dock              *string                       `json:"dock,omitempty"`
	ScheduledAt       *time.Time                    `json:"scheduled_at,omitempty"`
	ArrivedAt         *time.Time                    `json:"arrived_at,omitempty"`
	InvoiceNumber     *string                       `json:"invoice_number,omitempty"`
	Blocked           bool                          `json:"blocked"`
	Notes             *string                       `json:"notes,omitempty"`
	CreatedBy         *uuid.UUID                    `json:"created_by,omitempty"`
	CreatedAt         time.Time                     `json:"created_at"`
	Items             []ReceivingNoticeItemResponse `json:"items"`
}

type ReceivingNoticeItemResponse struct {
	ID                    int64   `json:"id"`
	PurchaseOrderItemCode *int64  `json:"purchase_order_item_code,omitempty"`
	ItemCode              int64   `json:"item_code"`
	Mask                  string  `json:"mask"`
	ExpectedQty           float64 `json:"expected_qty"`
	ReceivedQty           float64 `json:"received_qty"`
	Unit                  *string `json:"unit,omitempty"`
	Notes                 *string `json:"notes,omitempty"`
}

type ReceivingDivergenceResponse struct {
	ID                    int64      `json:"id"`
	NoticeID              *int64     `json:"notice_id,omitempty"`
	PurchaseOrderCode     *int64     `json:"purchase_order_code,omitempty"`
	PurchaseOrderItemCode *int64     `json:"purchase_order_item_code,omitempty"`
	SupplierCode          *int64     `json:"supplier_code,omitempty"`
	ItemCode              *int64     `json:"item_code,omitempty"`
	Mask                  string     `json:"mask"`
	DivergenceType        string     `json:"divergence_type"`
	ExpectedQty           float64    `json:"expected_qty"`
	ActualQty             float64    `json:"actual_qty"`
	ExpectedPrice         *float64   `json:"expected_price,omitempty"`
	ActualPrice           *float64   `json:"actual_price,omitempty"`
	Resolution            string     `json:"resolution"`
	AffectsSupplierScore  bool       `json:"affects_supplier_score"`
	Notes                 *string    `json:"notes,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	ResolvedAt            *time.Time `json:"resolved_at,omitempty"`
}

// ---- Supplier EDI (FEDS) ----

type SupplierEDIMessageResponse struct {
	ID                int64                     `json:"id"`
	EnterpriseCode    int64                     `json:"enterprise_code"`
	SupplierCode      *int64                    `json:"supplier_code,omitempty"`
	Direction         string                    `json:"direction"`
	MessageType       string                    `json:"message_type"`
	PurchaseOrderCode *int64                    `json:"purchase_order_code,omitempty"`
	ExternalReference *string                   `json:"external_reference,omitempty"`
	Status            string                    `json:"status"`
	DivergenceCount   int                       `json:"divergence_count"`
	Payload           json.RawMessage           `json:"payload,omitempty"`
	Notes             *string                   `json:"notes,omitempty"`
	CreatedAt         time.Time                 `json:"created_at"`
	ProcessedAt       *time.Time                `json:"processed_at,omitempty"`
	Lines             []SupplierEDILineResponse `json:"lines"`
}

type SupplierEDILineResponse struct {
	ID                    int64      `json:"id"`
	PurchaseOrderItemCode *int64     `json:"purchase_order_item_code,omitempty"`
	ItemCode              *int64     `json:"item_code,omitempty"`
	Mask                  string     `json:"mask"`
	ConfirmedQty          float64    `json:"confirmed_qty"`
	ConfirmedPrice        float64    `json:"confirmed_price"`
	ConfirmedDate         *time.Time `json:"confirmed_date,omitempty"`
	Divergence            *string    `json:"divergence,omitempty"`
	Notes                 *string    `json:"notes,omitempty"`
}

// ---- Import landed cost (FREC0203 / FIMP) ----

type ImportProcessResponse struct {
	ID                int64                       `json:"id"`
	EnterpriseCode    int64                       `json:"enterprise_code"`
	ProcessNumber     int64                       `json:"process_number"`
	SupplierCode      *int64                      `json:"supplier_code,omitempty"`
	PurchaseOrderCode *int64                      `json:"purchase_order_code,omitempty"`
	Reference         *string                     `json:"reference,omitempty"`
	Incoterm          *string                     `json:"incoterm,omitempty"`
	Currency          string                      `json:"currency"`
	ExchangeRate      float64                     `json:"exchange_rate"`
	ApportionBasis    string                      `json:"apportion_basis"`
	Status            string                      `json:"status"`
	Notes             *string                     `json:"notes,omitempty"`
	CreatedAt         time.Time                   `json:"created_at"`
	NationalizedAt    *time.Time                  `json:"nationalized_at,omitempty"`
	Items             []ImportProcessItemResponse `json:"items"`
	Expenses          []ImportExpenseResponse     `json:"expenses"`
	TotalExpenses     float64                     `json:"total_expenses"`
	TotalLandedValue  float64                     `json:"total_landed_value"`
}

type ImportProcessItemResponse struct {
	ID                  int64   `json:"id"`
	ItemCode            int64   `json:"item_code"`
	Mask                string  `json:"mask"`
	Quantity            float64 `json:"quantity"`
	Weight              float64 `json:"weight"`
	FobUnitPrice        float64 `json:"fob_unit_price"`
	ApportionedExpenses float64 `json:"apportioned_expenses"`
	LandedUnitCost      float64 `json:"landed_unit_cost"`
	Notes               *string `json:"notes,omitempty"`
}

type ImportExpenseResponse struct {
	ID          int64   `json:"id"`
	ExpenseType string  `json:"expense_type"`
	Amount      float64 `json:"amount"`
	InItemCost  bool    `json:"in_item_cost"`
	Notes       *string `json:"notes,omitempty"`
}

// ---- Procurement parameters (FUTL0125) ----

type ProcurementParameterResponse struct {
	ID          int64      `json:"id"`
	Domain      string     `json:"domain"`
	Key         string     `json:"param_key"`
	Value       string     `json:"param_value"`
	ValueType   string     `json:"value_type"`
	Description *string    `json:"description,omitempty"`
	UpdatedBy   *uuid.UUID `json:"updated_by,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ---- Supplier homologation (FAVF0203) ----

type SupplierHomologationResponse struct {
	ID           int64      `json:"id"`
	SupplierCode int64      `json:"supplier_code"`
	Status       string     `json:"status"`
	IQFScore     *float64   `json:"iqf_score,omitempty"`
	Category     *string    `json:"category,omitempty"`
	ValidUntil   *time.Time `json:"valid_until,omitempty"`
	Notes        *string    `json:"notes,omitempty"`
	DecidedBy    *uuid.UUID `json:"decided_by,omitempty"`
	DecidedAt    time.Time  `json:"decided_at"`
}
