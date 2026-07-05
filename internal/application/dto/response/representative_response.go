package response

import "time"

type RepresentativeTypeResponse struct {
	Code                 int64     `json:"code"`
	Description          string    `json:"description"`
	IsFree               bool      `json:"is_free"`
	IgnoresDirectBilling bool      `json:"ignores_direct_billing"`
	IsActive             bool      `json:"is_active"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type RepresentativeResponse struct {
	Code                    int64                                         `json:"code"`
	IsCustomer              bool                                          `json:"is_customer"`
	CustomerCode            *int64                                        `json:"customer_code,omitempty"`
	IsSupplier              bool                                          `json:"is_supplier"`
	SupplierCode            *int64                                        `json:"supplier_code,omitempty"`
	Name                    string                                        `json:"name"`
	TradeName               *string                                       `json:"trade_name,omitempty"`
	TypeCode                *int64                                        `json:"type_code,omitempty"`
	CategoryCode            *int64                                        `json:"category_code,omitempty"`
	RegisterDate            time.Time                                     `json:"register_date"`
	CoreNumber              *string                                       `json:"core_number,omitempty"`
	DocumentNumber          string                                        `json:"document_number"`
	PostalCode              *string                                       `json:"postal_code,omitempty"`
	City                    *string                                       `json:"city,omitempty"`
	State                   *string                                       `json:"state,omitempty"`
	FullAddress             *string                                       `json:"full_address,omitempty"`
	Street                  *string                                       `json:"street,omitempty"`
	StreetNumber            *string                                       `json:"street_number,omitempty"`
	Complement              *string                                       `json:"complement,omitempty"`
	District                *string                                       `json:"district,omitempty"`
	MainPhone               *string                                       `json:"main_phone,omitempty"`
	MainEmail               *string                                       `json:"main_email,omitempty"`
	DeviceQuantity          int                                           `json:"device_quantity"`
	IsActive                bool                                          `json:"is_active"`
	Blocked                 bool                                          `json:"blocked"`
	BlockReason             *string                                       `json:"block_reason,omitempty"`
	CreatedAt               time.Time                                     `json:"created_at"`
	UpdatedAt               time.Time                                     `json:"updated_at"`
	Enterprises             []RepresentativeEnterpriseResponse            `json:"enterprises,omitempty"`
	Accounting              []RepresentativeAccountingResponse            `json:"accounting,omitempty"`
	Regions                 []RepresentativeRegionResponse                `json:"regions,omitempty"`
	Segments                []RepresentativeSegmentResponse               `json:"segments,omitempty"`
	SalesPlans              []RepresentativeSalesPlanResponse             `json:"sales_plans,omitempty"`
	Interests               []RepresentativeInterestResponse              `json:"interests,omitempty"`
	Phones                  []RepresentativePhoneResponse                 `json:"phones,omitempty"`
	Emails                  []RepresentativeEmailResponse                 `json:"emails,omitempty"`
	CorrespondenceAddresses []RepresentativeCorrespondenceAddressResponse `json:"correspondence_addresses,omitempty"`
	Contacts                []RepresentativeContactResponse               `json:"contacts,omitempty"`
}

type RepresentativeEnterpriseResponse struct {
	ID                    int64     `json:"id"`
	RepresentativeCode    int64     `json:"representative_code"`
	EnterpriseCode        int64     `json:"enterprise_code"`
	EnterpriseName        *string   `json:"enterprise_name,omitempty"`
	CommissionPatternCode *int64    `json:"commission_pattern_code,omitempty"`
	CommissionPct         float64   `json:"commission_pct"`
	IsDefault             bool      `json:"is_default"`
	IsActive              bool      `json:"is_active"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type RepresentativeAccountingResponse struct {
	ID                   int64     `json:"id"`
	RepresentativeCode   int64     `json:"representative_code"`
	EnterpriseCode       *int64    `json:"enterprise_code,omitempty"`
	EventType            string    `json:"event_type"`
	DebitAccountCode     *int64    `json:"debit_account_code,omitempty"`
	DebitCostCenterCode  *int64    `json:"debit_cost_center_code,omitempty"`
	CreditAccountCode    *int64    `json:"credit_account_code,omitempty"`
	CreditCostCenterCode *int64    `json:"credit_cost_center_code,omitempty"`
	HistoryCode          *int64    `json:"history_code,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type RepresentativeRegionResponse struct {
	ID                 int64     `json:"id"`
	RepresentativeCode int64     `json:"representative_code"`
	EnterpriseCode     *int64    `json:"enterprise_code,omitempty"`
	RegionCode         int64     `json:"region_code"`
	MicroregionCode    *int64    `json:"microregion_code,omitempty"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
}

type RepresentativeSegmentResponse struct {
	ID                 int64     `json:"id"`
	RepresentativeCode int64     `json:"representative_code"`
	EnterpriseCode     *int64    `json:"enterprise_code,omitempty"`
	MicroregionCode    *int64    `json:"microregion_code,omitempty"`
	MarketSegmentCode  int64     `json:"market_segment_code"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
}

type RepresentativeSalesPlanResponse struct {
	ID                 int64     `json:"id"`
	RepresentativeCode int64     `json:"representative_code"`
	EnterpriseCode     *int64    `json:"enterprise_code,omitempty"`
	MicroregionCode    *int64    `json:"microregion_code,omitempty"`
	SalesPlanCode      int64     `json:"sales_plan_code"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
}

type RepresentativeInterestResponse struct {
	ID                     int64     `json:"id"`
	RepresentativeCode     int64     `json:"representative_code"`
	ItemClassificationCode int64     `json:"item_classification_code"`
	IsActive               bool      `json:"is_active"`
	CreatedAt              time.Time `json:"created_at"`
}

type RepresentativePhoneResponse struct {
	ID                 int64     `json:"id"`
	RepresentativeCode int64     `json:"representative_code"`
	DDI                *string   `json:"ddi,omitempty"`
	DDD                *string   `json:"ddd,omitempty"`
	Phone              string    `json:"phone"`
	PhoneType          string    `json:"phone_type"`
	Ranking            int       `json:"ranking"`
	CreatedAt          time.Time `json:"created_at"`
}

type RepresentativeEmailResponse struct {
	ID                 int64     `json:"id"`
	RepresentativeCode int64     `json:"representative_code"`
	Email              string    `json:"email"`
	Ranking            int       `json:"ranking"`
	CreatedAt          time.Time `json:"created_at"`
}

type RepresentativeCorrespondenceAddressResponse struct {
	ID                 int64     `json:"id"`
	RepresentativeCode int64     `json:"representative_code"`
	PostalCode         *string   `json:"postal_code,omitempty"`
	City               *string   `json:"city,omitempty"`
	State              *string   `json:"state,omitempty"`
	FullAddress        *string   `json:"full_address,omitempty"`
	Street             *string   `json:"street,omitempty"`
	StreetNumber       *string   `json:"street_number,omitempty"`
	Complement         *string   `json:"complement,omitempty"`
	District           *string   `json:"district,omitempty"`
	IsDefault          bool      `json:"is_default"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type RepresentativeContactResponse struct {
	ID                 int64     `json:"id"`
	RepresentativeCode int64     `json:"representative_code"`
	ContactTypeCode    *int64    `json:"contact_type_code,omitempty"`
	Name               string    `json:"name"`
	Role               *string   `json:"role,omitempty"`
	Phone              *string   `json:"phone,omitempty"`
	Email              *string   `json:"email,omitempty"`
	Notes              *string   `json:"notes,omitempty"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type RepresentativeReportRowResponse struct {
	Code                 int64   `json:"code"`
	Name                 string  `json:"name"`
	TradeName            *string `json:"trade_name,omitempty"`
	TypeCode             *int64  `json:"type_code,omitempty"`
	TypeDescription      *string `json:"type_description,omitempty"`
	State                *string `json:"state,omitempty"`
	City                 *string `json:"city,omitempty"`
	MainPhone            *string `json:"main_phone,omitempty"`
	MainEmail            *string `json:"main_email,omitempty"`
	RegionCodes          []int64 `json:"region_codes"`
	IsActive             bool    `json:"is_active"`
	CommissionPct        float64 `json:"commission_pct"`
	DebitAccountCode     *int64  `json:"debit_account_code,omitempty"`
	CreditAccountCode    *int64  `json:"credit_account_code,omitempty"`
	GeneratedHistoryCode *int64  `json:"generated_history_code,omitempty"`
}

type RepresentativeFollowUpResponse struct {
	RepresentativeCode int64                                    `json:"representative_code"`
	RepresentativeName string                                   `json:"representative_name"`
	CustomerCount      int64                                    `json:"customer_count"`
	QuotationCount     int64                                    `json:"quotation_count"`
	OrderCount         int64                                    `json:"order_count"`
	TotalQuoted        float64                                  `json:"total_quoted"`
	TotalOrdered       float64                                  `json:"total_ordered"`
	AverageTicket      float64                                  `json:"average_ticket"`
	CommissionBase     float64                                  `json:"commission_base"`
	CommissionValue    float64                                  `json:"commission_value"`
	LastQuotationDate  *time.Time                               `json:"last_quotation_date,omitempty"`
	LastOrderDate      *time.Time                               `json:"last_order_date,omitempty"`
	Customers          []RepresentativeCustomerFollowUpResponse `json:"customers"`
}

type RepresentativeCustomerFollowUpResponse struct {
	CustomerCode      int64      `json:"customer_code"`
	QuotationCount    int64      `json:"quotation_count"`
	OrderCount        int64      `json:"order_count"`
	TotalQuoted       float64    `json:"total_quoted"`
	TotalOrdered      float64    `json:"total_ordered"`
	LastQuotationDate *time.Time `json:"last_quotation_date,omitempty"`
	LastOrderDate     *time.Time `json:"last_order_date,omitempty"`
}
