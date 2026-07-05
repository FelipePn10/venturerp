package response

import (
	"time"

	"github.com/google/uuid"
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
	ProbabilityPct          float64                      `json:"probability_pct"`
	CommissionPct           float64                      `json:"commission_pct"`
	IsNFCe                  bool                         `json:"is_nfce"`
	Street                  *string                      `json:"street,omitempty"`
	StreetNumber            *string                      `json:"street_number,omitempty"`
	ForeignDocument         *string                      `json:"foreign_document,omitempty"`
	ReleaseStatus           string                       `json:"release_status"`
	CommercialBlocked       bool                         `json:"commercial_blocked"`
	CommercialBlockReason   *string                      `json:"commercial_block_reason,omitempty"`
	CarrierCode             *int64                       `json:"carrier_code,omitempty"`
	FreightType             *string                      `json:"freight_type,omitempty"`
	VerifyFreight           bool                         `json:"verify_freight"`
	FreightValue            float64                      `json:"freight_value"`
	RedeliveryFreightValue  float64                      `json:"redelivery_freight_value"`
	InsuranceValue          float64                      `json:"insurance_value"`
	DiscountValue           float64                      `json:"discount_value"`
	SurchargeValue          float64                      `json:"surcharge_value"`
	RetainedTaxValue        float64                      `json:"retained_tax_value"`
	TotalGross              float64                      `json:"total_gross"`
	TotalNet                float64                      `json:"total_net"`
	DeliveryAuthorization   *string                      `json:"delivery_authorization,omitempty"`
	Notes                   *string                      `json:"notes,omitempty"`
	ObsCustomer             *string                      `json:"obs_customer,omitempty"`
	CancelReason            *string                      `json:"cancel_reason,omitempty"`
	CancelComplement        *string                      `json:"cancel_complement,omitempty"`
	AttendedReason          *string                      `json:"attended_reason,omitempty"`
	AttendedAt              *time.Time                   `json:"attended_at,omitempty"`
	ConvertedSalesOrderCode *int64                       `json:"converted_sales_order_code,omitempty"`
	ConvertedAt             *time.Time                   `json:"converted_at,omitempty"`
	IsActive                bool                         `json:"is_active"`
	CreatedAt               time.Time                    `json:"created_at"`
	UpdatedAt               time.Time                    `json:"updated_at"`
	CreatedBy               uuid.UUID                    `json:"created_by"`
	Items                   []SalesQuotationItemResponse `json:"items,omitempty"`
}

type SalesQuotationItemResponse struct {
	Code               int64      `json:"code"`
	SalesQuotationCode int64      `json:"sales_quotation_code"`
	Sequence           int        `json:"sequence"`
	ItemCode           int64      `json:"item_code"`
	Mask               string     `json:"mask"`
	SalesUOM           *string    `json:"sales_uom,omitempty"`
	WarehouseCode      *int64     `json:"warehouse_code,omitempty"`
	PriceTableCode     *int64     `json:"price_table_code,omitempty"`
	RequestedQty       float64    `json:"requested_qty"`
	UnitPrice          float64    `json:"unit_price"`
	AttendedQty        float64    `json:"attended_qty"`
	CancelledQty       float64    `json:"cancelled_qty"`
	Balance            float64    `json:"balance"`
	DeliveryDate       *time.Time `json:"delivery_date,omitempty"`
	DeliveryDateFirm   bool       `json:"delivery_date_firm"`
	DiscountPct        float64    `json:"discount_pct"`
	IPIPct             float64    `json:"ipi_pct"`
	STPct              float64    `json:"st_pct"`
	TotalGross         float64    `json:"total_gross"`
	TotalNet           float64    `json:"total_net"`
	TotalNetWithIPI    float64    `json:"total_net_with_ipi"`
	Status             string     `json:"status"`
	Notes              *string    `json:"notes,omitempty"`
	IsActive           bool       `json:"is_active"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type SalesQuotationReportResponse struct {
	TotalQuotations int64   `json:"total_quotations"`
	TotalGross      float64 `json:"total_gross"`
	TotalNet        float64 `json:"total_net"`
	OpenCount       int64   `json:"open_count"`
	ApprovedCount   int64   `json:"approved_count"`
	ConvertedCount  int64   `json:"converted_count"`
	CancelledCount  int64   `json:"cancelled_count"`
	ExpiredCount    int64   `json:"expired_count"`
	WeightedNet     float64 `json:"weighted_net"`
	RetainedTax     float64 `json:"retained_tax"`
}
