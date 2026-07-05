package response

import (
	"time"

	"github.com/google/uuid"
)

// SalesOrderResponse is the API representation of a sales order. It mirrors the
// domain entity but lives in the application boundary so the HTTP layer never
// leaks domain types directly.
type SalesOrderResponse struct {
	Code                        int64                    `json:"code"`
	OrderNumber                 int64                    `json:"order_number"`
	EnterpriseCode              int64                    `json:"enterprise_code"`
	Status                      string                   `json:"status"`
	Origin                      string                   `json:"origin"`
	EmissionDate                time.Time                `json:"emission_date"`
	DeliveryDate                *time.Time               `json:"delivery_date,omitempty"`
	DeliveryDateFirm            bool                     `json:"delivery_date_firm"`
	DigitDate                   time.Time                `json:"digit_date"`
	CustomerCode                *int64                   `json:"customer_code,omitempty"`
	BillingAddressCode          *int64                   `json:"billing_address_code,omitempty"`
	ShippingAddressCode         *int64                   `json:"shipping_address_code,omitempty"`
	RepresentativeCode          *int64                   `json:"representative_code,omitempty"`
	RepresentativeOrderNumber   *int64                   `json:"representative_order_number,omitempty"`
	PlanCode                    *int64                   `json:"plan_code,omitempty"`
	SalesDivisionCode           *int64                   `json:"sales_division_code,omitempty"`
	CommissionPct               float64                  `json:"commission_pct"`
	TaxTypeCode                 *int64                   `json:"tax_type_code,omitempty"`
	PresenceIndicator           *string                  `json:"presence_indicator,omitempty"`
	SalesChannel                *string                  `json:"sales_channel,omitempty"`
	DefaultNFType               *string                  `json:"default_nf_type,omitempty"`
	PriceTableCode              *int64                   `json:"price_table_code,omitempty"`
	CurrencyCode                string                   `json:"currency_code"`
	PaymentTermCode             *int64                   `json:"payment_term_code,omitempty"`
	AdditionalDays              int                      `json:"additional_days"`
	BearerCode                  *int64                   `json:"bearer_code,omitempty"`
	SaleDate                    *time.Time               `json:"sale_date,omitempty"`
	TotalWeightNet              float64                  `json:"total_weight_net"`
	TotalWeightGross            float64                  `json:"total_weight_gross"`
	TotalGross                  float64                  `json:"total_gross"`
	TotalNet                    float64                  `json:"total_net"`
	TotalNetNoST                float64                  `json:"total_net_no_st"`
	TotalWithIPIWithST          float64                  `json:"total_with_ipi_with_st"`
	Notes                       *string                  `json:"notes,omitempty"`
	ObsCustomer                 *string                  `json:"obs_customer,omitempty"`
	IsBlocked                   bool                     `json:"is_blocked"`
	BlockReason                 *string                  `json:"block_reason,omitempty"`
	IsFirm                      bool                     `json:"is_firm"`
	IsActive                    bool                     `json:"is_active"`
	IsNFCe                      bool                     `json:"is_nfce"`
	Street                      *string                  `json:"street,omitempty"`
	StreetNumber                *string                  `json:"street_number,omitempty"`
	ForeignDocument             *string                  `json:"foreign_document,omitempty"`
	CollectionEstablishmentCode *int64                   `json:"collection_establishment_code,omitempty"`
	NFTypeDescription           *string                  `json:"nf_type_description,omitempty"`
	CarrierCode                 *int64                   `json:"carrier_code,omitempty"`
	FreightType                 *string                  `json:"freight_type,omitempty"`
	FreightValue                float64                  `json:"freight_value"`
	InsuranceValue              float64                  `json:"insurance_value"`
	VolumeQuantity              float64                  `json:"volume_quantity"`
	VolumeType                  *string                  `json:"volume_type,omitempty"`
	NetWeight                   float64                  `json:"net_weight"`
	GrossWeight                 float64                  `json:"gross_weight"`
	DiscountValue               float64                  `json:"discount_value"`
	SurchargeValue              float64                  `json:"surcharge_value"`
	ProjectCode                 *string                  `json:"project_code,omitempty"`
	ProjectName                 *string                  `json:"project_name,omitempty"`
	CommercialAnalysisStatus    string                   `json:"commercial_analysis_status"`
	FinancialAnalysisStatus     string                   `json:"financial_analysis_status"`
	ReleaseStatus               string                   `json:"release_status"`
	ConferenceStatus            string                   `json:"conference_status"`
	CancelReason                *string                  `json:"cancel_reason,omitempty"`
	CancelComplement            *string                  `json:"cancel_complement,omitempty"`
	AttendedReason              *string                  `json:"attended_reason,omitempty"`
	AttendedAt                  *time.Time               `json:"attended_at,omitempty"`
	DelayReason                 *string                  `json:"delay_reason,omitempty"`
	DelayAction                 *string                  `json:"delay_action,omitempty"`
	Items                       []SalesOrderItemResponse `json:"items,omitempty"`
	CreatedAt                   time.Time                `json:"created_at"`
	UpdatedAt                   time.Time                `json:"updated_at"`
	CreatedBy                   uuid.UUID                `json:"created_by"`
}

type SalesOrderReportResponse struct {
	TotalOrders            int64   `json:"total_orders"`
	TotalGross             float64 `json:"total_gross"`
	TotalNet               float64 `json:"total_net"`
	OpenCount              int64   `json:"open_count"`
	ConfirmedCount         int64   `json:"confirmed_count"`
	InvoicedCount          int64   `json:"invoiced_count"`
	CancelledCount         int64   `json:"cancelled_count"`
	BlockedCount           int64   `json:"blocked_count"`
	CommercialPendingCount int64   `json:"commercial_pending_count"`
	FinancialPendingCount  int64   `json:"financial_pending_count"`
	ConferencePendingCount int64   `json:"conference_pending_count"`
	DelayedCount           int64   `json:"delayed_count"`
}

// SalesOrderItemResponse is the API representation of a sales order line.
type SalesOrderItemResponse struct {
	Code             int64      `json:"code"`
	SalesOrderCode   int64      `json:"sales_order_code"`
	Sequence         int        `json:"sequence"`
	ItemCode         int64      `json:"item_code"`
	Mask             string     `json:"mask"`
	DigitDate        time.Time  `json:"digit_date"`
	NFType           *string    `json:"nf_type,omitempty"`
	SalesUOM         *string    `json:"sales_uom,omitempty"`
	WarehouseCode    *int64     `json:"warehouse_code,omitempty"`
	PriceTableCode   *int64     `json:"price_table_code,omitempty"`
	RequestedQty     float64    `json:"requested_qty"`
	UnitPrice        float64    `json:"unit_price"`
	AttendedQty      float64    `json:"attended_qty"`
	CancelledQty     float64    `json:"cancelled_qty"`
	Balance          float64    `json:"balance"`
	DeliveryDate     *time.Time `json:"delivery_date,omitempty"`
	DeliveryDateFirm bool       `json:"delivery_date_firm"`
	CustomerDelivery *string    `json:"customer_delivery,omitempty"`
	Lot              *string    `json:"lot,omitempty"`
	CouponDelivery   *string    `json:"coupon_delivery,omitempty"`
	PaidAtCashier    bool       `json:"paid_at_cashier"`
	IPIPct           float64    `json:"ipi_pct"`
	ICMSPct          float64    `json:"icms_pct"`
	PISPct           float64    `json:"pis_pct"`
	COFINSPct        float64    `json:"cofins_pct"`
	STPct            float64    `json:"st_pct"`
	DiscountPct      float64    `json:"discount_pct"`
	TotalGross       float64    `json:"total_gross"`
	TotalNet         float64    `json:"total_net"`
	TotalNetWithIPI  float64    `json:"total_net_with_ipi"`
	TotalIPI         float64    `json:"total_ipi"`
	TotalST          float64    `json:"total_st"`
	UnitWeightNet    float64    `json:"unit_weight_net"`
	UnitWeightGross  float64    `json:"unit_weight_gross"`
	Status           string     `json:"status"`
	Notes            *string    `json:"notes,omitempty"`
	IsActive         bool       `json:"is_active"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
