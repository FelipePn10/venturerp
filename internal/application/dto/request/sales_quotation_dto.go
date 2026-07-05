package request

import "github.com/google/uuid"

type CreateSalesQuotationDTO struct {
	EnterpriseCode         int64     `json:"enterprise_code"`
	Status                 string    `json:"status"`
	QuotationType          string    `json:"quotation_type"`
	EmissionDate           string    `json:"emission_date"`
	DigitDate              string    `json:"digit_date"`
	ValidUntil             *string   `json:"valid_until,omitempty"`
	DeliveryDate           *string   `json:"delivery_date,omitempty"`
	DeliveryDateFirm       bool      `json:"delivery_date_firm"`
	PurchaseOrderNumber    *string   `json:"purchase_order_number,omitempty"`
	CustomerCode           *int64    `json:"customer_code,omitempty"`
	BillingAddressCode     *int64    `json:"billing_address_code,omitempty"`
	ShippingAddressCode    *int64    `json:"shipping_address_code,omitempty"`
	RepresentativeCode     *int64    `json:"representative_code,omitempty"`
	SalesDivisionCode      *int64    `json:"sales_division_code,omitempty"`
	PriceTableCode         *int64    `json:"price_table_code,omitempty"`
	PaymentTermCode        *int64    `json:"payment_term_code,omitempty"`
	CurrencyCode           string    `json:"currency_code"`
	ProbabilityPct         float64   `json:"probability_pct"`
	CommissionPct          float64   `json:"commission_pct"`
	IsNFCe                 bool      `json:"is_nfce"`
	Street                 *string   `json:"street,omitempty"`
	StreetNumber           *string   `json:"street_number,omitempty"`
	ForeignDocument        *string   `json:"foreign_document,omitempty"`
	ReleaseStatus          string    `json:"release_status"`
	CommercialBlocked      bool      `json:"commercial_blocked"`
	CommercialBlockReason  *string   `json:"commercial_block_reason,omitempty"`
	CarrierCode            *int64    `json:"carrier_code,omitempty"`
	FreightType            *string   `json:"freight_type,omitempty"`
	VerifyFreight          bool      `json:"verify_freight"`
	FreightValue           float64   `json:"freight_value"`
	RedeliveryFreightValue float64   `json:"redelivery_freight_value"`
	InsuranceValue         float64   `json:"insurance_value"`
	DiscountValue          float64   `json:"discount_value"`
	SurchargeValue         float64   `json:"surcharge_value"`
	RetainedTaxValue       float64   `json:"retained_tax_value"`
	DeliveryAuthorization  *string   `json:"delivery_authorization,omitempty"`
	Notes                  *string   `json:"notes,omitempty"`
	ObsCustomer            *string   `json:"obs_customer,omitempty"`
	CreatedBy              uuid.UUID `json:"created_by"`
}

type UpdateSalesQuotationDTO struct {
	Code                   int64   `json:"code"`
	Status                 string  `json:"status"`
	QuotationType          string  `json:"quotation_type"`
	ValidUntil             *string `json:"valid_until,omitempty"`
	DeliveryDate           *string `json:"delivery_date,omitempty"`
	DeliveryDateFirm       bool    `json:"delivery_date_firm"`
	PurchaseOrderNumber    *string `json:"purchase_order_number,omitempty"`
	CustomerCode           *int64  `json:"customer_code,omitempty"`
	BillingAddressCode     *int64  `json:"billing_address_code,omitempty"`
	ShippingAddressCode    *int64  `json:"shipping_address_code,omitempty"`
	RepresentativeCode     *int64  `json:"representative_code,omitempty"`
	SalesDivisionCode      *int64  `json:"sales_division_code,omitempty"`
	PriceTableCode         *int64  `json:"price_table_code,omitempty"`
	PaymentTermCode        *int64  `json:"payment_term_code,omitempty"`
	CurrencyCode           string  `json:"currency_code"`
	ProbabilityPct         float64 `json:"probability_pct"`
	CommissionPct          float64 `json:"commission_pct"`
	IsNFCe                 bool    `json:"is_nfce"`
	Street                 *string `json:"street,omitempty"`
	StreetNumber           *string `json:"street_number,omitempty"`
	ForeignDocument        *string `json:"foreign_document,omitempty"`
	ReleaseStatus          string  `json:"release_status"`
	CommercialBlocked      bool    `json:"commercial_blocked"`
	CommercialBlockReason  *string `json:"commercial_block_reason,omitempty"`
	CarrierCode            *int64  `json:"carrier_code,omitempty"`
	FreightType            *string `json:"freight_type,omitempty"`
	VerifyFreight          bool    `json:"verify_freight"`
	FreightValue           float64 `json:"freight_value"`
	RedeliveryFreightValue float64 `json:"redelivery_freight_value"`
	InsuranceValue         float64 `json:"insurance_value"`
	DiscountValue          float64 `json:"discount_value"`
	SurchargeValue         float64 `json:"surcharge_value"`
	RetainedTaxValue       float64 `json:"retained_tax_value"`
	DeliveryAuthorization  *string `json:"delivery_authorization,omitempty"`
	Notes                  *string `json:"notes,omitempty"`
	ObsCustomer            *string `json:"obs_customer,omitempty"`
}

type ChangeSalesQuotationStatusDTO struct {
	Code   int64  `json:"code"`
	Status string `json:"status"`
}

type CancelSalesQuotationDTO struct {
	Code       int64   `json:"code"`
	Reason     string  `json:"reason"`
	Complement *string `json:"complement,omitempty"`
}

type AttendSalesQuotationDTO struct {
	Code       int64     `json:"code"`
	Reason     string    `json:"reason"`
	Complement *string   `json:"complement,omitempty"`
	EventDate  string    `json:"event_date"`
	CreatedBy  uuid.UUID `json:"created_by"`
}

type UncancelSalesQuotationDTO struct {
	Code       int64     `json:"code"`
	Reason     string    `json:"reason"`
	Complement *string   `json:"complement,omitempty"`
	CreatedBy  uuid.UUID `json:"created_by"`
}

type CreateSalesQuotationItemDTO struct {
	SalesQuotationCode int64   `json:"sales_quotation_code"`
	Sequence           int     `json:"sequence"`
	ItemCode           int64   `json:"item_code"`
	Mask               string  `json:"mask"`
	SalesUOM           *string `json:"sales_uom,omitempty"`
	WarehouseCode      *int64  `json:"warehouse_code,omitempty"`
	PriceTableCode     *int64  `json:"price_table_code,omitempty"`
	RequestedQty       float64 `json:"requested_qty"`
	UnitPrice          float64 `json:"unit_price"`
	DeliveryDate       *string `json:"delivery_date,omitempty"`
	DeliveryDateFirm   bool    `json:"delivery_date_firm"`
	DiscountPct        float64 `json:"discount_pct"`
	IPIPct             float64 `json:"ipi_pct"`
	STPct              float64 `json:"st_pct"`
	Notes              *string `json:"notes,omitempty"`
}

type UpdateSalesQuotationItemDTO struct {
	Code             int64   `json:"code"`
	RequestedQty     float64 `json:"requested_qty"`
	UnitPrice        float64 `json:"unit_price"`
	AttendedQty      float64 `json:"attended_qty"`
	CancelledQty     float64 `json:"cancelled_qty"`
	DeliveryDate     *string `json:"delivery_date,omitempty"`
	DeliveryDateFirm bool    `json:"delivery_date_firm"`
	DiscountPct      float64 `json:"discount_pct"`
	IPIPct           float64 `json:"ipi_pct"`
	STPct            float64 `json:"st_pct"`
	Notes            *string `json:"notes,omitempty"`
}

type ConvertSalesQuotationDTO struct {
	Code      int64     `json:"code"`
	Status    string    `json:"status"`
	Origin    string    `json:"origin"`
	CreatedBy uuid.UUID `json:"created_by"`
}
