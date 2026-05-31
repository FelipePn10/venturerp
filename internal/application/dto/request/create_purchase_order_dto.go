package request

import "github.com/google/uuid"

type CreatePurchaseOrderDTO struct {
	EnterpriseCode      int64     `json:"enterprise_code"`
	Status              string    `json:"status"`
	Origin              string    `json:"origin"`
	EmissionDate        string    `json:"emission_date"`
	DeliveryDate        *string   `json:"delivery_date,omitempty"`
	SupplierCode        *int64    `json:"supplier_code,omitempty"`
	PaymentTermCode     *int64    `json:"payment_term_code,omitempty"`
	CurrencyCode        string    `json:"currency_code"`
	ShippingAddressCode *int64    `json:"shipping_address_code,omitempty"`
	Notes               *string   `json:"notes,omitempty"`
	TotalGross          float64   `json:"total_gross"`
	TotalNet            float64   `json:"total_net"`
	TotalDiscount       float64   `json:"total_discount"`
	IsFirm              bool      `json:"is_firm"`
	CreatedBy           uuid.UUID `json:"created_by"`
	// Comercial / fiscal (capa) — opcionais; quando vazios e houver fornecedor,
	// são preenchidos a partir dos defaults do fornecedor.
	PriceTableCode   *int64  `json:"price_table_code,omitempty"`
	InvoiceTypeCode  *int64  `json:"invoice_type_code,omitempty"`
	FinancialAccount *string `json:"financial_account,omitempty"`
	RequestTypeCode  *int64  `json:"request_type_code,omitempty"`
	CurrencyDate     *string `json:"currency_date,omitempty"`
	// Transporte
	FreightType            string  `json:"freight_type,omitempty"`
	FreightValueType       *string `json:"freight_value_type,omitempty"`
	FreightValueMode       *string `json:"freight_value_mode,omitempty"`
	FreightValue           float64 `json:"freight_value,omitempty"`
	CarrierCode            *int64  `json:"carrier_code,omitempty"`
	RedispatchCarrierCode  *int64  `json:"redispatch_carrier_code,omitempty"`
	RedispatchFreightType  *string `json:"redispatch_freight_type,omitempty"`
	RedispatchFreightValue float64 `json:"redispatch_freight_value,omitempty"`
	// Adiantamento / importação / outros
	AdvanceDate  *string `json:"advance_date,omitempty"`
	AdvanceValue float64 `json:"advance_value,omitempty"`
	IncotermCode *string `json:"incoterm_code,omitempty"`
	ShipmentDate *string `json:"shipment_date,omitempty"`
	TalaoNumber  *string `json:"talao_number,omitempty"`
}

// CreatePurchaseOrderItemDTO adds an item to an existing purchase order. Price,
// internal UM/qty/price and IPI% are resolved automatically (price table,
// conversões por item, classificação fiscal) when not provided.
type CreatePurchaseOrderItemDTO struct {
	PurchaseOrderCode        int64    `json:"-"`
	ItemCode                 int64    `json:"item_code"`
	Mask                     string   `json:"mask,omitempty"`
	RequestedQty             float64  `json:"requested_qty"`
	UnitPrice                float64  `json:"unit_price,omitempty"`
	PurchaseUOM              *string  `json:"purchase_uom,omitempty"`
	InternalUOM              *string  `json:"internal_uom,omitempty"`
	DiscountPct              float64  `json:"discount_pct,omitempty"`
	IPIPct                   *float64 `json:"ipi_pct,omitempty"`
	ICMSPct                  float64  `json:"icms_pct,omitempty"`
	ICMSSTPct                float64  `json:"icms_st_pct,omitempty"`
	TolerancePct             float64  `json:"tolerance_pct,omitempty"`
	DeliveryDate             *string  `json:"delivery_date,omitempty"`
	PromisedDate             *string  `json:"promised_date,omitempty"`
	OperationTypeCode        *int64   `json:"operation_type_code,omitempty"`
	InvoiceTypeCode          *int64   `json:"invoice_type_code,omitempty"`
	AccountingAccount        *string  `json:"accounting_account,omitempty"`
	CostCenterCode           *int64   `json:"cost_center_code,omitempty"`
	FiscalClassificationCode *int64   `json:"fiscal_classification_code,omitempty"`
	RequesterEmployeeCode    *int64   `json:"requester_employee_code,omitempty"`
	ContractCode             *int64   `json:"contract_code,omitempty"`
	QuotationCode            *int64   `json:"quotation_code,omitempty"`
	UtilizationType          *string  `json:"utilization_type,omitempty"`
	Notes                    *string  `json:"notes,omitempty"`
}

// ApprovePurchaseSuggestionDTO approves an MRP purchase suggestion (a PURCHASE
// planned order) and generates a purchase order. PlannedOrderCode comes from the
// URL; the body carries the buyer's choices.
type ApprovePurchaseSuggestionDTO struct {
	PlannedOrderCode int64     `json:"-"`
	EnterpriseCode   int64     `json:"enterprise_code"`
	SupplierCode     *int64    `json:"supplier_code,omitempty"`
	UnitPrice        float64   `json:"unit_price"`
	Notes            *string   `json:"notes,omitempty"`
	CreatedBy        uuid.UUID `json:"created_by"`
}
