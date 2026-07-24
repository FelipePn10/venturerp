package request

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type CreateSalesQuotationDTO struct {
	QuotationNumber        int64           `json:"quotation_number,omitempty"`
	EnterpriseCode         int64           `json:"enterprise_code"`
	Status                 string          `json:"status"`
	QuotationType          string          `json:"quotation_type"`
	EmissionDate           string          `json:"emission_date"`
	DigitDate              string          `json:"digit_date"`
	ValidUntil             *string         `json:"valid_until,omitempty"`
	DeliveryDate           *string         `json:"delivery_date,omitempty"`
	DeliveryDateFirm       bool            `json:"delivery_date_firm"`
	PurchaseOrderNumber    *string         `json:"purchase_order_number,omitempty"`
	CustomerCode           *int64          `json:"customer_code,omitempty"`
	BillingAddressCode     *int64          `json:"billing_address_code,omitempty"`
	ShippingAddressCode    *int64          `json:"shipping_address_code,omitempty"`
	RepresentativeCode     *int64          `json:"representative_code,omitempty"`
	SalesDivisionCode      *int64          `json:"sales_division_code,omitempty"`
	PriceTableCode         *int64          `json:"price_table_code,omitempty"`
	PaymentTermCode        *int64          `json:"payment_term_code,omitempty"`
	CurrencyCode           string          `json:"currency_code"`
	ProbabilityPct         decimal.Decimal `json:"probability_pct"`
	CommissionPct          decimal.Decimal `json:"commission_pct"`
	IsNFCe                 bool            `json:"is_nfce"`
	DeliveryWithReceipt    bool            `json:"delivery_with_receipt"`
	Street                 *string         `json:"street,omitempty"`
	StreetNumber           *string         `json:"street_number,omitempty"`
	ForeignDocument        *string         `json:"foreign_document,omitempty"`
	ReleaseStatus          string          `json:"release_status"`
	CommercialBlocked      bool            `json:"commercial_blocked"`
	CommercialBlockReason  *string         `json:"commercial_block_reason,omitempty"`
	CarrierCode            *int64          `json:"carrier_code,omitempty"`
	FreightType            *string         `json:"freight_type,omitempty"`
	VerifyFreight          bool            `json:"verify_freight"`
	FreightValue           decimal.Decimal `json:"freight_value"`
	RedeliveryFreightValue decimal.Decimal `json:"redelivery_freight_value"`
	InsuranceValue         decimal.Decimal `json:"insurance_value"`
	DiscountValue          decimal.Decimal `json:"discount_value"`
	SurchargeValue         decimal.Decimal `json:"surcharge_value"`
	RetainedTaxValue       decimal.Decimal `json:"retained_tax_value"`
	DeliveryAuthorization  *string         `json:"delivery_authorization,omitempty"`
	Notes                  *string         `json:"notes,omitempty"`
	ObsCustomer            *string         `json:"obs_customer,omitempty"`
	CreatedBy              uuid.UUID       `json:"created_by"`
}

type UpdateSalesQuotationDTO struct {
	Code                   int64           `json:"code"`
	Status                 string          `json:"status"`
	QuotationType          string          `json:"quotation_type"`
	ValidUntil             *string         `json:"valid_until,omitempty"`
	DeliveryDate           *string         `json:"delivery_date,omitempty"`
	DeliveryDateFirm       bool            `json:"delivery_date_firm"`
	PurchaseOrderNumber    *string         `json:"purchase_order_number,omitempty"`
	CustomerCode           *int64          `json:"customer_code,omitempty"`
	BillingAddressCode     *int64          `json:"billing_address_code,omitempty"`
	ShippingAddressCode    *int64          `json:"shipping_address_code,omitempty"`
	RepresentativeCode     *int64          `json:"representative_code,omitempty"`
	SalesDivisionCode      *int64          `json:"sales_division_code,omitempty"`
	PriceTableCode         *int64          `json:"price_table_code,omitempty"`
	PaymentTermCode        *int64          `json:"payment_term_code,omitempty"`
	CurrencyCode           string          `json:"currency_code"`
	ProbabilityPct         decimal.Decimal `json:"probability_pct"`
	CommissionPct          decimal.Decimal `json:"commission_pct"`
	IsNFCe                 bool            `json:"is_nfce"`
	DeliveryWithReceipt    bool            `json:"delivery_with_receipt"`
	Street                 *string         `json:"street,omitempty"`
	StreetNumber           *string         `json:"street_number,omitempty"`
	ForeignDocument        *string         `json:"foreign_document,omitempty"`
	ReleaseStatus          string          `json:"release_status"`
	CommercialBlocked      bool            `json:"commercial_blocked"`
	CommercialBlockReason  *string         `json:"commercial_block_reason,omitempty"`
	CarrierCode            *int64          `json:"carrier_code,omitempty"`
	FreightType            *string         `json:"freight_type,omitempty"`
	VerifyFreight          bool            `json:"verify_freight"`
	FreightValue           decimal.Decimal `json:"freight_value"`
	RedeliveryFreightValue decimal.Decimal `json:"redelivery_freight_value"`
	InsuranceValue         decimal.Decimal `json:"insurance_value"`
	DiscountValue          decimal.Decimal `json:"discount_value"`
	SurchargeValue         decimal.Decimal `json:"surcharge_value"`
	RetainedTaxValue       decimal.Decimal `json:"retained_tax_value"`
	DeliveryAuthorization  *string         `json:"delivery_authorization,omitempty"`
	Notes                  *string         `json:"notes,omitempty"`
	ObsCustomer            *string         `json:"obs_customer,omitempty"`
}

type ChangeSalesQuotationStatusDTO struct {
	Code   int64  `json:"code"`
	Status string `json:"status"`
}

type ChangeSalesQuotationReleaseDTO struct {
	Code          int64  `json:"code"`
	ReleaseStatus string `json:"release_status"`
	Reason        string `json:"reason"`
}

type CancelSalesQuotationDTO struct {
	Code       int64   `json:"code"`
	ReasonCode int64   `json:"reason_code"`
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
	ReasonCode int64     `json:"reason_code"`
	Reason     string    `json:"reason"`
	Complement *string   `json:"complement,omitempty"`
	CreatedBy  uuid.UUID `json:"created_by"`
}

type CancelSalesQuotationItemDTO struct {
	Code       int64   `json:"code"`
	ReasonCode int64   `json:"reason_code"`
	Complement *string `json:"complement,omitempty"`
}

type CreateSalesQuotationItemDTO struct {
	SalesQuotationCode int64           `json:"sales_quotation_code"`
	Sequence           int             `json:"sequence"`
	ItemCode           int64           `json:"item_code"`
	Mask               string          `json:"mask"`
	SalesUOM           *string         `json:"sales_uom,omitempty"`
	WarehouseCode      *int64          `json:"warehouse_code,omitempty"`
	PriceTableCode     *int64          `json:"price_table_code,omitempty"`
	RequestedQty       decimal.Decimal `json:"requested_qty"`
	UnitPrice          decimal.Decimal `json:"unit_price"`
	DeliveryDate       *string         `json:"delivery_date,omitempty"`
	DeliveryDateFirm   bool            `json:"delivery_date_firm"`
	DiscountPct        decimal.Decimal `json:"discount_pct"`
	IPIPct             decimal.Decimal `json:"ipi_pct"`
	STPct              decimal.Decimal `json:"st_pct"`
	Notes              *string         `json:"notes,omitempty"`
}

type UpdateSalesQuotationItemDTO struct {
	Code             int64           `json:"code"`
	RequestedQty     decimal.Decimal `json:"requested_qty"`
	UnitPrice        decimal.Decimal `json:"unit_price"`
	AttendedQty      decimal.Decimal `json:"attended_qty"`
	CancelledQty     decimal.Decimal `json:"cancelled_qty"`
	DeliveryDate     *string         `json:"delivery_date,omitempty"`
	DeliveryDateFirm bool            `json:"delivery_date_firm"`
	DiscountPct      decimal.Decimal `json:"discount_pct"`
	IPIPct           decimal.Decimal `json:"ipi_pct"`
	STPct            decimal.Decimal `json:"st_pct"`
	Notes            *string         `json:"notes,omitempty"`
}

type ConvertSalesQuotationDTO struct {
	Code      int64     `json:"code"`
	Status    string    `json:"status"`
	Origin    string    `json:"origin"`
	CreatedBy uuid.UUID `json:"created_by"`
}

type SaveSalesQuotationParametersDTO struct {
	PurchaseOrderPrompt         string          `json:"purchase_order_prompt"`
	DeliveryAuthorizationPrompt string          `json:"delivery_authorization_prompt"`
	FinalConsumerCustomerCode   *int64          `json:"final_consumer_customer_code,omitempty"`
	AllowServiceItemsNFCe       bool            `json:"allow_service_items_nfce"`
	DefaultNFCe                 bool            `json:"default_nfce"`
	MinimumCIFFreight           decimal.Decimal `json:"minimum_cif_freight"`
	AddRedeliveryToFreight      bool            `json:"add_redelivery_to_freight"`
}

type SaveCommissionPatternDTO struct {
	Code          int64           `json:"code"`
	Description   string          `json:"description"`
	CommissionPct decimal.Decimal `json:"commission_pct"`
	InvoicePct    decimal.Decimal `json:"invoice_pct"`
	PaymentPct    decimal.Decimal `json:"payment_pct"`
}

type SaveCancellationReasonDTO struct {
	Code              int64  `json:"code"`
	Description       string `json:"description"`
	AllowUncancel     bool   `json:"allow_uncancel"`
	RequireComplement bool   `json:"require_complement"`
}

type CreateSalesQuotationAttachmentDTO struct {
	SalesQuotationCode int64     `json:"sales_quotation_code"`
	FileName           string    `json:"file_name"`
	ContentType        string    `json:"content_type"`
	FileSize           int64     `json:"file_size"`
	StorageKey         string    `json:"storage_key"`
	UploadedBy         uuid.UUID `json:"uploaded_by"`
	Content            []byte    `json:"-"`
}
