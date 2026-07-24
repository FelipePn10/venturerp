package response

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SalesQuotationResponse struct {
	Code                    int64                        `json:"code"`
	QuotationNumber         int64                        `json:"quotation_number"`
	EnterpriseCode          int64                        `json:"enterprise_code"`
	Status                  string                       `json:"status"`
	QuotationType           string                       `json:"quotation_type"`
	EmissionDate            time.Time                    `json:"emission_date"`
	DigitDate               time.Time                    `json:"digit_date"`
	ValidUntil              *time.Time                   `json:"valid_until,omitempty"`
	DeliveryDate            *time.Time                   `json:"delivery_date,omitempty"`
	DeliveryDateFirm        bool                         `json:"delivery_date_firm"`
	PurchaseOrderNumber     *string                      `json:"purchase_order_number,omitempty"`
	CustomerCode            *int64                       `json:"customer_code,omitempty"`
	BillingAddressCode      *int64                       `json:"billing_address_code,omitempty"`
	ShippingAddressCode     *int64                       `json:"shipping_address_code,omitempty"`
	RepresentativeCode      *int64                       `json:"representative_code,omitempty"`
	SalesDivisionCode       *int64                       `json:"sales_division_code,omitempty"`
	PriceTableCode          *int64                       `json:"price_table_code,omitempty"`
	PaymentTermCode         *int64                       `json:"payment_term_code,omitempty"`
	CurrencyCode            string                       `json:"currency_code"`
	ProbabilityPct          decimal.Decimal              `json:"probability_pct"`
	CommissionPct           decimal.Decimal              `json:"commission_pct"`
	IsNFCe                  bool                         `json:"is_nfce"`
	DeliveryWithReceipt     bool                         `json:"delivery_with_receipt"`
	Street                  *string                      `json:"street,omitempty"`
	StreetNumber            *string                      `json:"street_number,omitempty"`
	ForeignDocument         *string                      `json:"foreign_document,omitempty"`
	ReleaseStatus           string                       `json:"release_status"`
	CommercialBlocked       bool                         `json:"commercial_blocked"`
	CommercialBlockReason   *string                      `json:"commercial_block_reason,omitempty"`
	CarrierCode             *int64                       `json:"carrier_code,omitempty"`
	FreightType             *string                      `json:"freight_type,omitempty"`
	VerifyFreight           bool                         `json:"verify_freight"`
	FreightValue            decimal.Decimal              `json:"freight_value"`
	RedeliveryFreightValue  decimal.Decimal              `json:"redelivery_freight_value"`
	InsuranceValue          decimal.Decimal              `json:"insurance_value"`
	DiscountValue           decimal.Decimal              `json:"discount_value"`
	SurchargeValue          decimal.Decimal              `json:"surcharge_value"`
	RetainedTaxValue        decimal.Decimal              `json:"retained_tax_value"`
	TotalGross              decimal.Decimal              `json:"total_gross"`
	TotalNet                decimal.Decimal              `json:"total_net"`
	DeliveryAuthorization   *string                      `json:"delivery_authorization,omitempty"`
	Notes                   *string                      `json:"notes,omitempty"`
	ObsCustomer             *string                      `json:"obs_customer,omitempty"`
	CancelReason            *string                      `json:"cancel_reason,omitempty"`
	CancelComplement        *string                      `json:"cancel_complement,omitempty"`
	AttendedReason          *string                      `json:"attended_reason,omitempty"`
	AttendedAt              *time.Time                   `json:"attended_at,omitempty"`
	ConvertedSalesOrderCode *int64                       `json:"converted_sales_order_code,omitempty"`
	ConvertedAt             *time.Time                   `json:"converted_at,omitempty"`
	DAVGeneratedAt          *time.Time                   `json:"dav_generated_at,omitempty"`
	DAVReportKey            *uuid.UUID                   `json:"dav_report_key,omitempty"`
	ConsumerAddress         *string                      `json:"consumer_address,omitempty"`
	IsActive                bool                         `json:"is_active"`
	CreatedAt               time.Time                    `json:"created_at"`
	UpdatedAt               time.Time                    `json:"updated_at"`
	CreatedBy               uuid.UUID                    `json:"created_by"`
	Items                   []SalesQuotationItemResponse `json:"items,omitempty"`
	CanPrintFiscalReceipt   bool                         `json:"can_print_fiscal_receipt"`
	CanPrintSalesOrder      bool                         `json:"can_print_sales_order"`
	CanSendEmail            bool                         `json:"can_send_email"`
	CanPrintDAVReport       bool                         `json:"can_print_dav_report"`
}

type SalesQuotationItemResponse struct {
	Code               int64           `json:"code"`
	SalesQuotationCode int64           `json:"sales_quotation_code"`
	Sequence           int             `json:"sequence"`
	ItemCode           int64           `json:"item_code"`
	Mask               string          `json:"mask"`
	SalesUOM           *string         `json:"sales_uom,omitempty"`
	WarehouseCode      *int64          `json:"warehouse_code,omitempty"`
	PriceTableCode     *int64          `json:"price_table_code,omitempty"`
	RequestedQty       decimal.Decimal `json:"requested_qty"`
	UnitPrice          decimal.Decimal `json:"unit_price"`
	AttendedQty        decimal.Decimal `json:"attended_qty"`
	CancelledQty       decimal.Decimal `json:"cancelled_qty"`
	Balance            decimal.Decimal `json:"balance"`
	DeliveryDate       *time.Time      `json:"delivery_date,omitempty"`
	DeliveryDateFirm   bool            `json:"delivery_date_firm"`
	DiscountPct        decimal.Decimal `json:"discount_pct"`
	IPIPct             decimal.Decimal `json:"ipi_pct"`
	STPct              decimal.Decimal `json:"st_pct"`
	TotalGross         decimal.Decimal `json:"total_gross"`
	TotalNet           decimal.Decimal `json:"total_net"`
	TotalNetWithIPI    decimal.Decimal `json:"total_net_with_ipi"`
	Status             string          `json:"status"`
	Notes              *string         `json:"notes,omitempty"`
	IsActive           bool            `json:"is_active"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type SalesQuotationReportResponse struct {
	TotalQuotations int64           `json:"total_quotations"`
	TotalGross      decimal.Decimal `json:"total_gross"`
	TotalNet        decimal.Decimal `json:"total_net"`
	OpenCount       int64           `json:"open_count"`
	ApprovedCount   int64           `json:"approved_count"`
	ConvertedCount  int64           `json:"converted_count"`
	CancelledCount  int64           `json:"cancelled_count"`
	ExpiredCount    int64           `json:"expired_count"`
	WeightedNet     decimal.Decimal `json:"weighted_net"`
	RetainedTax     decimal.Decimal `json:"retained_tax"`
}
