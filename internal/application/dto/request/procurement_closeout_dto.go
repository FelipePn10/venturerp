package request

import "encoding/json"

// ---- Receiving notice + divergences (FAVR) ----

type CreateReceivingNoticeDTO struct {
	EnterpriseCode    int64                    `json:"enterprise_code"`
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
	PurchaseOrderItemCode *int64  `json:"purchase_order_item_code"`
	ItemCode              int64   `json:"item_code"`
	Mask                  string  `json:"mask"`
	ExpectedQty           float64 `json:"expected_qty"`
	Unit                  *string `json:"unit"`
	Notes                 *string `json:"notes"`
}

type UpdateReceivingNoticeStatusDTO struct {
	Status  string `json:"status"`
	Blocked bool   `json:"blocked"`
}

type CreateReceivingDivergenceDTO struct {
	NoticeID              *int64   `json:"notice_id"`
	PurchaseOrderCode     *int64   `json:"purchase_order_code"`
	PurchaseOrderItemCode *int64   `json:"purchase_order_item_code"`
	SupplierCode          *int64   `json:"supplier_code"`
	ItemCode              *int64   `json:"item_code"`
	Mask                  string   `json:"mask"`
	DivergenceType        string   `json:"divergence_type"`
	ExpectedQty           float64  `json:"expected_qty"`
	ActualQty             float64  `json:"actual_qty"`
	ExpectedPrice         *float64 `json:"expected_price"`
	ActualPrice           *float64 `json:"actual_price"`
	AffectsSupplierScore  bool     `json:"affects_supplier_score"`
	Notes                 *string  `json:"notes"`
}

type ResolveReceivingDivergenceDTO struct {
	Resolution string `json:"resolution"`
}

// ---- Supplier EDI (FEDS) ----

type CreateEDIMessageDTO struct {
	EnterpriseCode    int64           `json:"enterprise_code"`
	SupplierCode      *int64          `json:"supplier_code"`
	Direction         string          `json:"direction"`
	MessageType       string          `json:"message_type"`
	PurchaseOrderCode *int64          `json:"purchase_order_code"`
	ExternalReference *string         `json:"external_reference"`
	Payload           json.RawMessage `json:"payload"`
	Notes             *string         `json:"notes"`
	QtyTolerance      float64         `json:"qty_tolerance"`
	PriceTolerance    float64         `json:"price_tolerance"`
	Lines             []EDILineDTO    `json:"lines"`
}

// EDILineDTO carries the supplier-confirmed values plus the purchase order
// reference values (po_qty/po_price/po_date) so the divergence can be detected
// without coupling to the purchase order repository.
type EDILineDTO struct {
	PurchaseOrderItemCode *int64  `json:"purchase_order_item_code"`
	ItemCode              *int64  `json:"item_code"`
	Mask                  string  `json:"mask"`
	ConfirmedQty          float64 `json:"confirmed_qty"`
	ConfirmedPrice        float64 `json:"confirmed_price"`
	ConfirmedDate         *string `json:"confirmed_date"`
	PoQty                 float64 `json:"po_qty"`
	PoPrice               float64 `json:"po_price"`
	PoDate                *string `json:"po_date"`
	Notes                 *string `json:"notes"`
}

// ---- Import landed cost (FREC0203 / FIMP) ----

type CreateImportProcessDTO struct {
	EnterpriseCode    int64              `json:"enterprise_code"`
	SupplierCode      *int64             `json:"supplier_code"`
	PurchaseOrderCode *int64             `json:"purchase_order_code"`
	Reference         *string            `json:"reference"`
	Incoterm          *string            `json:"incoterm"`
	Currency          string             `json:"currency"`
	ExchangeRate      float64            `json:"exchange_rate"`
	ApportionBasis    string             `json:"apportion_basis"`
	Notes             *string            `json:"notes"`
	Items             []ImportItemDTO    `json:"items"`
	Expenses          []ImportExpenseDTO `json:"expenses"`
}

type ImportItemDTO struct {
	ItemCode     int64   `json:"item_code"`
	Mask         string  `json:"mask"`
	Quantity     float64 `json:"quantity"`
	Weight       float64 `json:"weight"`
	FobUnitPrice float64 `json:"fob_unit_price"`
	Notes        *string `json:"notes"`
}

type ImportExpenseDTO struct {
	ExpenseType string  `json:"expense_type"`
	Amount      float64 `json:"amount"`
	InItemCost  *bool   `json:"in_item_cost"`
	Notes       *string `json:"notes"`
}

type UpdateImportProcessStatusDTO struct {
	Status string `json:"status"`
}

// ---- Procurement parameters (FUTL0125) ----

type UpsertProcurementParameterDTO struct {
	EnterpriseCode int64   `json:"enterprise_code"`
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
