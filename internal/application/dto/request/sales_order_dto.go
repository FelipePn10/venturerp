package request

import "github.com/google/uuid"

type CreateSalesOrderDTO struct {
	EnterpriseCode              int64     `json:"enterprise_code"`
	Status                      string    `json:"status"`        // R, P, A, OA, OF
	Origin                      string    `json:"origin"`        // NORMAL, DEPENDENT, ASSISTANCE, RESERVE, COPY
	EmissionDate                string    `json:"emission_date"` // YYYY-MM-DD
	DeliveryDate                *string   `json:"delivery_date,omitempty"`
	DeliveryDateFirm            bool      `json:"delivery_date_firm"`
	CustomerCode                *int64    `json:"customer_code,omitempty"`
	BillingAddressCode          *int64    `json:"billing_address_code,omitempty"`
	ShippingAddressCode         *int64    `json:"shipping_address_code,omitempty"`
	RepresentativeCode          *int64    `json:"representative_code,omitempty"`
	RepresentativeOrderNumber   *int64    `json:"representative_order_number,omitempty"`
	PlanCode                    *int64    `json:"plan_code,omitempty"`
	SalesDivisionCode           *int64    `json:"sales_division_code,omitempty"`
	CommissionPct               float64   `json:"commission_pct"`
	TaxTypeCode                 *int64    `json:"tax_type_code,omitempty"`
	PresenceIndicator           *string   `json:"presence_indicator,omitempty"`
	SalesChannel                *string   `json:"sales_channel,omitempty"`
	DefaultNFType               *string   `json:"default_nf_type,omitempty"`
	PriceTableCode              *int64    `json:"price_table_code,omitempty"`
	CurrencyCode                string    `json:"currency_code"`
	PaymentTermCode             *int64    `json:"payment_term_code,omitempty"`
	AdditionalDays              int       `json:"additional_days"`
	BearerCode                  *int64    `json:"bearer_code,omitempty"`
	SaleDate                    *string   `json:"sale_date,omitempty"`
	Notes                       *string   `json:"notes,omitempty"`
	ObsCustomer                 *string   `json:"obs_customer,omitempty"`
	IsNFCe                      bool      `json:"is_nfce"`
	Street                      *string   `json:"street,omitempty"`
	StreetNumber                *string   `json:"street_number,omitempty"`
	ForeignDocument             *string   `json:"foreign_document,omitempty"`
	CollectionEstablishmentCode *int64    `json:"collection_establishment_code,omitempty"`
	NFTypeDescription           *string   `json:"nf_type_description,omitempty"`
	CarrierCode                 *int64    `json:"carrier_code,omitempty"`
	FreightType                 *string   `json:"freight_type,omitempty"`
	FreightValue                float64   `json:"freight_value"`
	InsuranceValue              float64   `json:"insurance_value"`
	VolumeQuantity              float64   `json:"volume_quantity"`
	VolumeType                  *string   `json:"volume_type,omitempty"`
	NetWeight                   float64   `json:"net_weight"`
	GrossWeight                 float64   `json:"gross_weight"`
	DiscountValue               float64   `json:"discount_value"`
	SurchargeValue              float64   `json:"surcharge_value"`
	ProjectCode                 *string   `json:"project_code,omitempty"`
	ProjectName                 *string   `json:"project_name,omitempty"`
	CreatedBy                   uuid.UUID `json:"created_by"`
}

type UpdateSalesOrderDTO struct {
	Code                        int64   `json:"code"`
	Status                      string  `json:"status"`
	Origin                      string  `json:"origin"`
	DeliveryDate                *string `json:"delivery_date,omitempty"`
	DeliveryDateFirm            bool    `json:"delivery_date_firm"`
	CustomerCode                *int64  `json:"customer_code,omitempty"`
	BillingAddressCode          *int64  `json:"billing_address_code,omitempty"`
	ShippingAddressCode         *int64  `json:"shipping_address_code,omitempty"`
	RepresentativeCode          *int64  `json:"representative_code,omitempty"`
	RepresentativeOrderNumber   *int64  `json:"representative_order_number,omitempty"`
	PlanCode                    *int64  `json:"plan_code,omitempty"`
	SalesDivisionCode           *int64  `json:"sales_division_code,omitempty"`
	CommissionPct               float64 `json:"commission_pct"`
	TaxTypeCode                 *int64  `json:"tax_type_code,omitempty"`
	PresenceIndicator           *string `json:"presence_indicator,omitempty"`
	SalesChannel                *string `json:"sales_channel,omitempty"`
	DefaultNFType               *string `json:"default_nf_type,omitempty"`
	PriceTableCode              *int64  `json:"price_table_code,omitempty"`
	CurrencyCode                string  `json:"currency_code"`
	PaymentTermCode             *int64  `json:"payment_term_code,omitempty"`
	AdditionalDays              int     `json:"additional_days"`
	BearerCode                  *int64  `json:"bearer_code,omitempty"`
	SaleDate                    *string `json:"sale_date,omitempty"`
	Notes                       *string `json:"notes,omitempty"`
	ObsCustomer                 *string `json:"obs_customer,omitempty"`
	IsFirm                      bool    `json:"is_firm"`
	IsNFCe                      bool    `json:"is_nfce"`
	Street                      *string `json:"street,omitempty"`
	StreetNumber                *string `json:"street_number,omitempty"`
	ForeignDocument             *string `json:"foreign_document,omitempty"`
	CollectionEstablishmentCode *int64  `json:"collection_establishment_code,omitempty"`
	NFTypeDescription           *string `json:"nf_type_description,omitempty"`
	CarrierCode                 *int64  `json:"carrier_code,omitempty"`
	FreightType                 *string `json:"freight_type,omitempty"`
	FreightValue                float64 `json:"freight_value"`
	InsuranceValue              float64 `json:"insurance_value"`
	VolumeQuantity              float64 `json:"volume_quantity"`
	VolumeType                  *string `json:"volume_type,omitempty"`
	NetWeight                   float64 `json:"net_weight"`
	GrossWeight                 float64 `json:"gross_weight"`
	DiscountValue               float64 `json:"discount_value"`
	SurchargeValue              float64 `json:"surcharge_value"`
	ProjectCode                 *string `json:"project_code,omitempty"`
	ProjectName                 *string `json:"project_name,omitempty"`
}

type BlockSalesOrderDTO struct {
	Code   int64  `json:"code"`
	Reason string `json:"reason"`
}

type ChangeStatusDTO struct {
	Code   int64  `json:"code"`
	Status string `json:"status"`
}

type CreateSalesOrderItemDTO struct {
	SalesOrderCode   int64   `json:"sales_order_code"`
	Sequence         int     `json:"sequence"`
	ItemCode         int64   `json:"item_code"`
	Mask             string  `json:"mask"`
	DigitDate        string  `json:"digit_date"` // YYYY-MM-DD
	NFType           *string `json:"nf_type,omitempty"`
	SalesUOM         *string `json:"sales_uom,omitempty"`
	WarehouseCode    *int64  `json:"warehouse_code,omitempty"`
	PriceTableCode   *int64  `json:"price_table_code,omitempty"`
	RequestedQty     float64 `json:"requested_qty"`
	UnitPrice        float64 `json:"unit_price"`
	DeliveryDate     *string `json:"delivery_date,omitempty"`
	DeliveryDateFirm bool    `json:"delivery_date_firm"`
	CustomerDelivery *string `json:"customer_delivery,omitempty"`
	Lot              *string `json:"lot,omitempty"`
	CouponDelivery   *string `json:"coupon_delivery,omitempty"`
	PaidAtCashier    bool    `json:"paid_at_cashier"`
	IPIPct           float64 `json:"ipi_pct"`
	ICMSPct          float64 `json:"icms_pct"`
	PISPct           float64 `json:"pis_pct"`
	COFINSPct        float64 `json:"cofins_pct"`
	STPct            float64 `json:"st_pct"`
	DiscountPct      float64 `json:"discount_pct"`
	UnitWeightNet    float64 `json:"unit_weight_net"`
	UnitWeightGross  float64 `json:"unit_weight_gross"`
	Notes            *string `json:"notes,omitempty"`
}

type UpdateSalesOrderItemDTO struct {
	Code             int64   `json:"code"`
	RequestedQty     float64 `json:"requested_qty"`
	UnitPrice        float64 `json:"unit_price"`
	AttendedQty      float64 `json:"attended_qty"`
	CancelledQty     float64 `json:"cancelled_qty"`
	DeliveryDate     *string `json:"delivery_date,omitempty"`
	DeliveryDateFirm bool    `json:"delivery_date_firm"`
	CustomerDelivery *string `json:"customer_delivery,omitempty"`
	Lot              *string `json:"lot,omitempty"`
	CouponDelivery   *string `json:"coupon_delivery,omitempty"`
	PaidAtCashier    bool    `json:"paid_at_cashier"`
	IPIPct           float64 `json:"ipi_pct"`
	ICMSPct          float64 `json:"icms_pct"`
	PISPct           float64 `json:"pis_pct"`
	COFINSPct        float64 `json:"cofins_pct"`
	STPct            float64 `json:"st_pct"`
	DiscountPct      float64 `json:"discount_pct"`
	UnitWeightNet    float64 `json:"unit_weight_net"`
	UnitWeightGross  float64 `json:"unit_weight_gross"`
	Notes            *string `json:"notes,omitempty"`
}
