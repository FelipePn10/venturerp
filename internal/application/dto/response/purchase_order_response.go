package response

import (
	"time"

	"github.com/google/uuid"
)

// PurchaseOrderResponse is the API representation of a purchase order header.
type PurchaseOrderResponse struct {
	Code                int64      `json:"code"`
	OrderNumber         int64      `json:"order_number"`
	EnterpriseCode      int64      `json:"enterprise_code"`
	Status              string     `json:"status"`
	Origin              string     `json:"origin"`
	EmissionDate        time.Time  `json:"emission_date"`
	DeliveryDate        *time.Time `json:"delivery_date,omitempty"`
	SupplierCode        *int64     `json:"supplier_code,omitempty"`
	PaymentTermCode     *int64     `json:"payment_term_code,omitempty"`
	CurrencyCode        string     `json:"currency_code"`
	ShippingAddressCode *int64     `json:"shipping_address_code,omitempty"`
	Notes               *string    `json:"notes,omitempty"`

	TotalGross    float64 `json:"total_gross"`
	TotalNet      float64 `json:"total_net"`
	TotalDiscount float64 `json:"total_discount"`

	PriceTableCode   *int64     `json:"price_table_code,omitempty"`
	InvoiceTypeCode  *int64     `json:"invoice_type_code,omitempty"`
	FinancialAccount *string    `json:"financial_account,omitempty"`
	RequestTypeCode  *int64     `json:"request_type_code,omitempty"`
	CurrencyDate     *time.Time `json:"currency_date,omitempty"`

	FreightType      string  `json:"freight_type"`
	FreightValueType *string `json:"freight_value_type,omitempty"`
	FreightValueMode *string `json:"freight_value_mode,omitempty"`
	FreightValue     float64 `json:"freight_value"`
	CarrierCode      *int64  `json:"carrier_code,omitempty"`

	RedispatchCarrierCode  *int64  `json:"redispatch_carrier_code,omitempty"`
	RedispatchFreightType  *string `json:"redispatch_freight_type,omitempty"`
	RedispatchFreightValue float64 `json:"redispatch_freight_value"`

	AdvanceDate  *time.Time `json:"advance_date,omitempty"`
	AdvanceValue float64    `json:"advance_value"`

	IncotermCode *string    `json:"incoterm_code,omitempty"`
	ShipmentDate *time.Time `json:"shipment_date,omitempty"`

	TalaoNumber  *string `json:"talao_number,omitempty"`
	AlcadaStatus string  `json:"alcada_status"`

	IsActive  bool      `json:"is_active"`
	IsFirm    bool      `json:"is_firm"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy uuid.UUID `json:"created_by"`

	Items []PurchaseOrderItemResponse `json:"items,omitempty"`
}

// PurchaseOrderItemResponse is the API representation of a purchase order line.
type PurchaseOrderItemResponse struct {
	Code              int64      `json:"code"`
	PurchaseOrderCode int64      `json:"purchase_order_code"`
	Sequence          int        `json:"sequence"`
	ItemCode          int64      `json:"item_code"`
	Mask              string     `json:"mask"`
	RequestedQty      float64    `json:"requested_qty"`
	ReceivedQty       float64    `json:"received_qty"`
	CancelledQty      float64    `json:"cancelled_qty"`
	UnitPrice         float64    `json:"unit_price"`
	TotalPrice        float64    `json:"total_price"`
	DiscountPct       float64    `json:"discount_pct"`
	IPIPct            float64    `json:"ipi_pct"`
	ICMSPct           float64    `json:"icms_pct"`
	ICMSSTPct         float64    `json:"icms_st_pct"`
	Status            string     `json:"status"`
	DeliveryDate      *time.Time `json:"delivery_date,omitempty"`
	PromisedDate      *time.Time `json:"promised_date,omitempty"`
	Notes             *string    `json:"notes,omitempty"`

	PurchaseUOM   *string `json:"purchase_uom,omitempty"`
	InternalUOM   *string `json:"internal_uom,omitempty"`
	InternalQty   float64 `json:"internal_qty"`
	InternalPrice float64 `json:"internal_price"`

	TolerancePct          float64 `json:"tolerance_pct"`
	CancelledToleranceQty float64 `json:"cancelled_tolerance_qty"`

	OperationTypeCode        *int64  `json:"operation_type_code,omitempty"`
	InvoiceTypeCode          *int64  `json:"invoice_type_code,omitempty"`
	AccountingAccount        *string `json:"accounting_account,omitempty"`
	CostCenterCode           *int64  `json:"cost_center_code,omitempty"`
	FiscalClassificationCode *int64  `json:"fiscal_classification_code,omitempty"`

	RequesterEmployeeCode *int64  `json:"requester_employee_code,omitempty"`
	ContractCode          *int64  `json:"contract_code,omitempty"`
	QuotationCode         *int64  `json:"quotation_code,omitempty"`
	UtilizationType       *string `json:"utilization_type,omitempty"`

	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PurchaseOrderReceiptResponse struct {
	PurchaseOrder    *PurchaseOrderResponse   `json:"purchase_order"`
	ReceivedLines    []PurchaseReceiptLine    `json:"received_lines"`
	Movements        []StockMovementResponse  `json:"movements"`
	InspectionOrders []ReceiptInspectionOrder `json:"inspection_orders,omitempty"`
	Warnings         []string                 `json:"warnings,omitempty"`
}

// ReceiptInspectionOrder references an inspection order automatically opened when a
// received line matched an active receiving inspection route (FINS0212 behaviour).
type ReceiptInspectionOrder struct {
	InspectionOrderID     int64   `json:"inspection_order_id"`
	OrderNumber           int64   `json:"order_number"`
	PurchaseOrderItemCode int64   `json:"purchase_order_item_code"`
	ItemCode              int64   `json:"item_code"`
	WarehouseID           int64   `json:"warehouse_id"`
	Quantity              float64 `json:"quantity"`
}

type PurchaseReceiptLine struct {
	PurchaseOrderItemCode int64   `json:"purchase_order_item_code"`
	ItemCode              int64   `json:"item_code"`
	Mask                  string  `json:"mask"`
	WarehouseID           int64   `json:"warehouse_id"`
	Quantity              float64 `json:"quantity"`
	StockQuantity         float64 `json:"stock_quantity"`
	RemainingQty          float64 `json:"remaining_qty"`
	// UnderInspection is true when the line was routed into the inspection
	// warehouse instead of the requested one because an active route matched.
	UnderInspection bool `json:"under_inspection,omitempty"`
}

// ApprovePurchaseOrderResponse is the outcome of an alçada (approval limit)
// evaluation on a purchase order.
type ApprovePurchaseOrderResponse struct {
	PurchaseOrder         *PurchaseOrderResponse `json:"purchase_order"`
	Approved              bool                   `json:"approved"`
	RequiresAuthorization bool                   `json:"requires_authorization"`
	Blocked               bool                   `json:"blocked"`
	AlcadaStatus          string                 `json:"alcada_status"`
	AppliedAmount         float64                `json:"applied_amount"`
	AppliedCeiling        float64                `json:"applied_ceiling"`
	Message               string                 `json:"message"`
}
