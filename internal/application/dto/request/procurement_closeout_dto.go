package request

import (
	"encoding/json"
	"github.com/shopspring/decimal"
)

// ---- Receiving notice + divergences (FAVR) ----

type CreateReceivingNoticeDTO struct {
	EnterpriseCode    int64                    `json:"-"`
	SupplierCode      *int64                   `json:"supplier_code"`
	PurchaseOrderCode *int64                   `json:"purchase_order_code"`
	CarrierCode       *int64                   `json:"carrier_code"`
	Dock              *string                  `json:"dock"`
	ScheduledAt       *string                  `json:"scheduled_at"`
	InvoiceNumber     *string                  `json:"invoice_number"`
	Notes             *string                  `json:"notes"`
	Items             []ReceivingNoticeItemDTO `json:"items"`
}

type ReceivingNoticeItemDTO struct {
	PurchaseOrderItemCode *int64          `json:"purchase_order_item_code"`
	ItemCode              int64           `json:"item_code"`
	Mask                  string          `json:"mask"`
	ExpectedQty           decimal.Decimal `json:"expected_qty"`
	Unit                  *string         `json:"unit"`
	Notes                 *string         `json:"notes"`
}

type UpdateReceivingNoticeStatusDTO struct {
	Status  string `json:"status"`
	Blocked bool   `json:"blocked"`
}

type CreateReceivingDivergenceDTO struct {
	NoticeID              *int64           `json:"notice_id"`
	PurchaseOrderCode     *int64           `json:"purchase_order_code"`
	PurchaseOrderItemCode *int64           `json:"purchase_order_item_code"`
	SupplierCode          *int64           `json:"supplier_code"`
	ItemCode              *int64           `json:"item_code"`
	Mask                  string           `json:"mask"`
	DivergenceType        string           `json:"divergence_type"`
	ExpectedQty           decimal.Decimal  `json:"expected_qty"`
	ActualQty             decimal.Decimal  `json:"actual_qty"`
	ExpectedPrice         *decimal.Decimal `json:"expected_price"`
	ActualPrice           *decimal.Decimal `json:"actual_price"`
	AffectsSupplierScore  bool             `json:"affects_supplier_score"`
	Notes                 *string          `json:"notes"`
}

type ResolveReceivingDivergenceDTO struct {
	Resolution string `json:"resolution"`
}

// ---- Supplier EDI (FEDS) ----

type CreateEDIMessageDTO struct {
	EnterpriseCode    int64           `json:"-"`
	SupplierCode      *int64          `json:"supplier_code"`
	Direction         string          `json:"direction"`
	MessageType       string          `json:"message_type"`
	PurchaseOrderCode *int64          `json:"purchase_order_code"`
	ExternalReference *string         `json:"external_reference"`
	Payload           json.RawMessage `json:"payload"`
	Notes             *string         `json:"notes"`
	QtyTolerance      decimal.Decimal `json:"qty_tolerance"`
	PriceTolerance    decimal.Decimal `json:"price_tolerance"`
	Lines             []EDILineDTO    `json:"lines"`
}

// EDILineDTO carries the supplier-confirmed values plus the purchase order
// reference values (po_qty/po_price/po_date) so the divergence can be detected
// without coupling to the purchase order repository.
type EDILineDTO struct {
	PurchaseOrderItemCode *int64          `json:"purchase_order_item_code"`
	ItemCode              *int64          `json:"item_code"`
	Mask                  string          `json:"mask"`
	ConfirmedQty          decimal.Decimal `json:"confirmed_qty"`
	ConfirmedPrice        decimal.Decimal `json:"confirmed_price"`
	ConfirmedDate         *string         `json:"confirmed_date"`
	PoQty                 decimal.Decimal `json:"po_qty"`
	PoPrice               decimal.Decimal `json:"po_price"`
	PoDate                *string         `json:"po_date"`
	Notes                 *string         `json:"notes"`
}

// ---- Import landed cost (FREC0203 / FIMP) ----

type CreateImportProcessDTO struct {
	EnterpriseCode    int64              `json:"-"`
	SupplierCode      *int64             `json:"supplier_code"`
	PurchaseOrderCode *int64             `json:"purchase_order_code"`
	Reference         *string            `json:"reference"`
	Incoterm          *string            `json:"incoterm"`
	Currency          string             `json:"currency"`
	ExchangeRate      decimal.Decimal    `json:"exchange_rate"`
	ApportionBasis    string             `json:"apportion_basis"`
	Notes             *string            `json:"notes"`
	Items             []ImportItemDTO    `json:"items"`
	Expenses          []ImportExpenseDTO `json:"expenses"`
}

type ImportItemDTO struct {
	ItemCode     int64           `json:"item_code"`
	Mask         string          `json:"mask"`
	Quantity     decimal.Decimal `json:"quantity"`
	Weight       decimal.Decimal `json:"weight"`
	FobUnitPrice decimal.Decimal `json:"fob_unit_price"`
	Notes        *string         `json:"notes"`
}

type ImportExpenseDTO struct {
	ExpenseType string          `json:"expense_type"`
	Amount      decimal.Decimal `json:"amount"`
	InItemCost  *bool           `json:"in_item_cost"`
	Notes       *string         `json:"notes"`
}

type UpdateImportProcessStatusDTO struct {
	Status string `json:"status"`
}

// ---- Procurement parameters (FUTL0125) ----

type UpsertProcurementParameterDTO struct {
	EnterpriseCode int64   `json:"-"`
	Domain         string  `json:"domain"`
	Key            string  `json:"param_key"`
	Value          string  `json:"param_value"`
	ValueType      string  `json:"value_type"`
	Description    *string `json:"description"`
}

// ---- Supplier homologation (FAVF0203) ----

type CreateSupplierHomologationDTO struct {
	SupplierCode   int64   `json:"supplier_code"`
	PeriodStart    string  `json:"period_start"`
	PeriodEnd      string  `json:"period_end"`
	HomologatedMin float64 `json:"homologated_min"`
	ConditionalMin float64 `json:"conditional_min"`
	Status         string  `json:"status"`
	Category       *string `json:"category"`
	ValidUntil     *string `json:"valid_until"`
	Notes          *string `json:"notes"`
}
